package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database connection
var db *gorm.DB

// GORM Models
type Song struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	Artist      string    `gorm:"size:255;not null" json:"artist"`
	FileName    string    `gorm:"size:500" json:"file_name"`
	FilePath    string    `gorm:"size:500" json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Duration    float64   `json:"duration"`
	SampleRate  int       `json:"sample_rate"`
	FileFormat  string    `gorm:"size:10" json:"file_format"`
	UploadedAt  time.Time `gorm:"autoCreateTime" json:"uploaded_at"`
	ProcessedAt *time.Time `json:"processed_at"`
	IsProcessed bool      `gorm:"default:false" json:"is_processed"`
	
	// Relationships
	Fingerprints []Fingerprint `gorm:"foreignKey:SongID;constraint:OnDelete:CASCADE" json:"-"`
	QueryResults []QueryResult `gorm:"foreignKey:SongID;constraint:OnDelete:CASCADE" json:"-"`
}

type Fingerprint struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	Hash       int64   `gorm:"index:idx_hash;not null" json:"hash"`
	SongID     uint    `gorm:"index:idx_song_id;not null" json:"song_id"`
	TimeStamp  float64 `gorm:"column:time_offset;not null" json:"time_stamp"`
	
	// Optional debugging fields
	AnchorFreq float64 `json:"anchor_frequency,omitempty"`
	TargetFreq float64 `json:"target_frequency,omitempty"`
	TimeDelta  float64 `json:"time_delta,omitempty"`
	
	// Relationships
	Song Song `gorm:"foreignKey:SongID" json:"-"`
}

type QuerySession struct {
	ID            string     `gorm:"primaryKey;type:varchar(50)" json:"id"`
	QueryDuration float64    `json:"query_duration"`
	SampleRate    int        `json:"sample_rate"`
	TotalPeaks    int        `json:"total_peaks"`
	TotalPairs    int        `json:"total_pairs"`
	TotalHashes   int        `json:"total_hashes"`
	MatchFound    bool       `gorm:"default:false" json:"match_found"`
	BestMatchID   *uint      `json:"best_match_song_id,omitempty"`
	MatchScore    int        `json:"match_score"`
	TimeInSong    float64    `json:"time_in_song"`
	Confidence    float64    `json:"confidence_score"`
	QueryTime     time.Time  `gorm:"autoCreateTime" json:"query_time"`
	ProcessTime   float64    `json:"process_time_ms"`
	
	// Relationships
	BestMatch    *Song         `gorm:"foreignKey:BestMatchID" json:"best_match,omitempty"`
	QueryResults []QueryResult `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"query_results,omitempty"`
}

type QueryResult struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	SessionID      string  `gorm:"index:idx_session_id;not null" json:"session_id"`
	SongID         uint    `gorm:"index:idx_song_id;not null" json:"song_id"`
	MatchingHashes int     `gorm:"not null" json:"matching_hashes"`
	TimeOffset     float64 `gorm:"not null" json:"time_offset"`
	Confidence     float64 `json:"confidence_score"`
	
	// Relationships
	Session QuerySession `gorm:"foreignKey:SessionID" json:"-"`
	Song    Song         `gorm:"foreignKey:SongID" json:"song,omitempty"`
}


func InitDB() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(&Song{}, &Fingerprint{}, &QuerySession{}, &QueryResult{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	fmt.Println("✅ Successfully connected to database and migrated schema")
	return nil
}


func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// StoreSongInDB stores a song using GORM and returns the generated ID
func StoreSongInDB(songData Song, sampleRate int) uint {
	// Extract file info
	fileName := filepath.Base(songData.FilePath)
	fileExt := filepath.Ext(songData.FilePath)
	if fileExt != "" {
		fileExt = fileExt[1:] // Remove the dot
	}
	
	// Get file size
	var fileSize int64
	if stat, err := os.Stat(songData.FilePath); err == nil {
		fileSize = stat.Size()
	}

	song := Song{
		Title:       songData.Title,
		Artist:      songData.Artist,
		FileName:    fileName,
		FilePath:    songData.FilePath,
		FileSize:    fileSize,
		Duration:    songData.Duration,
		SampleRate:  sampleRate,
		FileFormat:  fileExt,
		UploadedAt:  time.Now(),
		IsProcessed: false,
	}

	result := db.Create(&song)
	if result.Error != nil {
		log.Printf("Error storing song in database: %v", result.Error)
		return 0
	}

	fmt.Printf("Stored song: %s by %s (ID: %d)\n", song.Title, song.Artist, song.ID)
	return song.ID
}

// StoreFingerprintsInDB stores fingerprints in batch using GORM
func StoreFingerprintsInDB(songID uint, fingerprints []Fingerprint) {
	if len(fingerprints) == 0 {
		fmt.Printf("No fingerprints to store for song ID %d\n", songID)
		return
	}

	// Convert your fingerprint format to GORM model
	var dbFingerprints []Fingerprint
	for _, fp := range fingerprints {
		dbFingerprints = append(dbFingerprints, Fingerprint{
			Hash:       int64(fp.Hash),
			SongID:     songID,
			TimeStamp: fp.TimeStamp,
			// Add optional fields if available
			AnchorFreq: 0, // You can extract from constellation pairs if needed
			TargetFreq: 0,
			TimeDelta:  0,
		})
	}

	// Batch insert fingerprints
	const batchSize = 1000
	for i := 0; i < len(dbFingerprints); i += batchSize {
		end := i + batchSize
		if end > len(dbFingerprints) {
			end = len(dbFingerprints)
		}

		batch := dbFingerprints[i:end]
		result := db.CreateInBatches(&batch, batchSize)
		if result.Error != nil {
			log.Printf("Error inserting fingerprint batch: %v", result.Error)
			continue
		}

		fmt.Printf("Inserted batch %d-%d fingerprints for song ID %d\n", i+1, end, songID)
	}

	// Mark song as processed
	now := time.Now()
	db.Model(&Song{}).Where("id = ?", songID).Updates(Song{
		IsProcessed: true,
		ProcessedAt: &now,
	})

	fmt.Printf("Successfully stored %d fingerprints for song ID %d\n", len(dbFingerprints), songID)
}

// QueryFingerprints finds matching songs for a snippet's fingerprints
func QueryFingerprints(snippetHashes []Fingerprint) (map[uint]map[float64]int, error) {
	matches := make(map[uint]map[float64]int) // songID -> timeOffset -> count

	for _, snippetHash := range snippetHashes {
		var dbMatches []Fingerprint
		
		result := db.Where("hash = ?", int64(snippetHash.Hash)).Find(&dbMatches)
		if result.Error != nil {
			log.Printf("Error querying fingerprint hash %d: %v", snippetHash.Hash, result.Error)
			continue
		}

		// Process matches
		for _, match := range dbMatches {
			songID := match.SongID
			// Calculate time offset between snippet and song
			timeOffset := match.TimeStamp - snippetHash.TimeStamp

			if matches[songID] == nil {
				matches[songID] = make(map[float64]int)
			}
			matches[songID][timeOffset]++
		}
	}

	return matches, nil
}

// CreateQuerySession creates a new query session and returns the ID
func CreateQuerySession(duration float64, sampleRate int, peaks, pairs, hashes int) (string, error) {
	// Generate a simple ID (you could use UUID library for better IDs)
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	session := QuerySession{
		ID:            sessionID,
		QueryDuration: duration,
		SampleRate:    sampleRate,
		TotalPeaks:    peaks,
		TotalPairs:    pairs,
		TotalHashes:   hashes,
		MatchFound:    false,
		QueryTime:     time.Now(),
	}

	result := db.Create(&session)
	if result.Error != nil {
		return "", fmt.Errorf("error creating query session: %w", result.Error)
	}

	return session.ID, nil
}

// UpdateQuerySessionResult updates the query session with results
func UpdateQuerySessionResult(sessionID string, matchFound bool, bestSongID uint, matchScore int, timeInSong float64, processTime float64) error {
	updates := QuerySession{
		MatchFound:  matchFound,
		MatchScore:  matchScore,
		TimeInSong:  timeInSong,
		ProcessTime: processTime,
	}

	if matchFound {
		updates.BestMatchID = &bestSongID
	}

	result := db.Model(&QuerySession{}).Where("id = ?", sessionID).Updates(updates)
	return result.Error
}

// StoreQueryResults stores individual query results
func StoreQueryResults(sessionID string, results map[uint]map[float64]int) error {
	for songID, offsets := range results {
		// Find the most common offset and total matches
		var bestOffset float64
		var maxCount int
		totalMatches := 0

		for offset, count := range offsets {
			totalMatches += count
			if count > maxCount {
				maxCount = count
				bestOffset = offset
			}
		}

		// Calculate confidence (percentage of total matches for this song)
		confidence := float64(maxCount) / float64(totalMatches) * 100

		// Store result
		queryResult := QueryResult{
			SessionID:      sessionID,
			SongID:         songID,
			MatchingHashes: totalMatches,
			TimeOffset:     bestOffset,
			Confidence:     confidence,
		}

		result := db.Create(&queryResult)
		if result.Error != nil {
			log.Printf("Error storing query result for song %d: %v", songID, result.Error)
		}
	}

	return nil
}

// GetSongByID retrieves a song by ID
func GetSongByID(songID uint) (*Song, error) {
	var song Song
	result := db.First(&song, songID)
	if result.Error != nil {
		return nil, fmt.Errorf("error fetching song: %w", result.Error)
	}
	return &song, nil
}

// GetAllSongs retrieves all songs with basic info
func GetAllSongs() ([]Song, error) {
	var songs []Song
	result := db.Find(&songs)
	if result.Error != nil {
		return nil, fmt.Errorf("error fetching songs: %w", result.Error)
	}
	return songs, nil
}

// DeleteSong deletes a song and all its fingerprints (CASCADE)
func DeleteSong(songID uint) error {
	result := db.Delete(&Song{}, songID)
	if result.Error != nil {
		return fmt.Errorf("error deleting song: %w", result.Error)
	}
	
	fmt.Printf("Deleted song ID %d and all its fingerprints\n", songID)
	return nil
}