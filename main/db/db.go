package db

import (
	"fmt"
	"shazam/main/models"
	"shazam/main/utils"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type PostgresClient struct {
	db *gorm.DB
}

//todo: replace all fingerprints, song, and couples with models.fingerprint, models.song and models.couples

// NewPostgresClient creates a new GORM-based PostgreSQL client for Neon DB
func NewPostgresClient(dsn string) (*PostgresClient, error) {
	// Configure GORM for PostgreSQL with optimizations
	config := &gorm.Config{
		PrepareStmt: true,                                // Enable prepared statements
		Logger:      logger.Default.LogMode(logger.Warn), // Reduce logging noise
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to PostgreSQL: %s", err)
	}

	// Configure connection pool for Neon DB free tier
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	// Optimize for Neon DB free tier limits
	sqlDB.SetMaxOpenConns(10)                 // Limit concurrent connections
	sqlDB.SetMaxIdleConns(5)                  // Keep some connections alive
	sqlDB.SetConnMaxLifetime(time.Hour)       // Recycle connections
	sqlDB.SetConnMaxIdleTime(time.Minute * 5) // Close idle connections

	// Auto-migrate tables
	err = db.AutoMigrate(&models.Song{}, &models.Fingerprint{})
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %s", err)
	}

	// Create additional indexes for better performance
	if err := createOptimizedIndexes(db); err != nil {
		// Log warning but don't fail - indexes might already exist
		fmt.Printf("Warning: Could not create some indexes: %v\n", err)
	}

	return &PostgresClient{db: db}, nil
}

// createOptimizedIndexes creates additional indexes for better query performance
func createOptimizedIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fingerprints_address_song ON fingerprints (address, song_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_songs_title_artist ON songs (title, artist)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fingerprints_song_time ON fingerprints (song_id, anchor_time_ms)",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			// Continue with other indexes even if one fails
			continue
		}
	}
	return nil
}

// Close closes the database connection
func (c *PostgresClient) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// StoreFingerprints stores fingerprints using efficient batch insert with conflict resolution
func (c *PostgresClient) StoreFingerprints(fingerprints map[uint32]Couple) error {
	if len(fingerprints) == 0 {
		return nil
	}

	// Convert map to slice for batch insert
	fpSlice := make([]Fingerprint, 0, len(fingerprints))
	for address, couple := range fingerprints {
		fpSlice = append(fpSlice, Fingerprint{
			Address:      address,
			AnchorTimeMs: couple.AnchorTimeMs,
			SongID:       couple.SongID,
		})
	}

	// Use transaction with batch insert and conflict resolution
	return c.db.Transaction(func(tx *gorm.DB) error {
		// Use ON CONFLICT DO NOTHING for PostgreSQL
		return tx.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(fpSlice, 1000).Error
	})
}

// GetCouples retrieves couples by addresses efficiently
func (c *PostgresClient) GetCouples(addresses []uint32) (map[uint32][]Couple, error) {
	if len(addresses) == 0 {
		return make(map[uint32][]Couple), nil
	}

	couples := make(map[uint32][]Couple)

	// Process in chunks to avoid parameter limits
	const chunkSize = 100
	for i := 0; i < len(addresses); i += chunkSize {
		end := i + chunkSize
		if end > len(addresses) {
			end = len(addresses)
		}

		chunk := addresses[i:end]
		var fingerprints []Fingerprint

		err := c.db.Where("address IN ?", chunk).Find(&fingerprints).Error
		if err != nil {
			return nil, fmt.Errorf("error querying database: %s", err)
		}

		// Group by address
		for _, fp := range fingerprints {
			couples[fp.Address] = append(couples[fp.Address], Couple{
				AnchorTimeMs: fp.AnchorTimeMs,
				SongID:       fp.SongID,
			})
		}
	}

	// Ensure all addresses have entries (even if empty)
	for _, addr := range addresses {
		if _, exists := couples[addr]; !exists {
			couples[addr] = []Couple{}
		}
	}

	return couples, nil
}

// TotalSongs returns the total number of songs
func (c *PostgresClient) TotalSongs() (int, error) {
	var count int64
	err := c.db.Model(&Song{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("error counting songs: %s", err)
	}
	return int(count), nil
}

// RegisterSong registers a new song in the database
func (c *PostgresClient) RegisterSong(songTitle, songArtist, ytID string) (uint32, error) {
	songID := utils.GenerateUniqueID()
	songKey := utils.GenerateSongKey(songTitle, songArtist)

	song := Song{
		ID:     uint(songID),
		Title:  songTitle,
		Artist: songArtist,
		YtID:   ytID,
		Key:    songKey,
	}

	err := c.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&song).Error
	})

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
			strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
			return 0, fmt.Errorf("song with ytID or key already exists: %v", err)
		}
		return 0, fmt.Errorf("failed to register song: %v", err)
	}

	return songID, nil
}

// GetSong retrieves a song by filter key (generic method)
func (c *PostgresClient) GetSong(filterKey string, value interface{}) (Song, bool, error) {
	validKeys := map[string]bool{
		"id":    true,
		"yt_id": true,
		"key":   true,
	}

	if !validKeys[filterKey] {
		return Song{}, false, fmt.Errorf("invalid filter key: %s", filterKey)
	}

	var song Song
	err := c.db.Where(fmt.Sprintf("%s = ?", filterKey), value).First(&song).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return Song{}, false, nil
		}
		return Song{}, false, fmt.Errorf("failed to retrieve song: %s", err)
	}

	return song, true, nil
}

// GetSongByID retrieves a song by ID
func (c *PostgresClient) GetSongByID(songID uint32) (Song, bool, error) {
	return c.GetSong("id", songID)
}

// GetSongByYTID retrieves a song by YouTube ID
func (c *PostgresClient) GetSongByYTID(ytID string) (Song, bool, error) {
	return c.GetSong("yt_id", ytID)
}

// GetSongByKey retrieves a song by key
func (c *PostgresClient) GetSongByKey(key string) (Song, bool, error) {
	return c.GetSong("key", key)
}

// DeleteSongByID deletes a song by ID
func (c *PostgresClient) DeleteSongByID(songID uint32) error {
	err := c.db.Delete(&Song{}, songID).Error
	if err != nil {
		return fmt.Errorf("failed to delete song: %v", err)
	}
	return nil
}

// DeleteCollection deletes a collection (table) from the database
func (c *PostgresClient) DeleteCollection(collectionName string) error {
	err := c.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", collectionName)).Error
	if err != nil {
		return fmt.Errorf("error deleting collection: %v", err)
	}
	return nil
}
