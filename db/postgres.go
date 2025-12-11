package db

import (
	"database/sql"
	"fmt"
	"shazoom/models"
	"shazoom/utils"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresClient struct {
	db *sql.DB
}

// NewPostgresClient connects to Cloud SQL via the DSN
func NewPostgresClient(dsn string) (*PostgresClient, error) {
	// Open connection
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening postgres connection: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to postgres: %w", err)
	}

	// Initialize tables
	if err := createPostgresTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %w", err)
	}

	fmt.Printf("successfully created postgreSQL client and created tables\n")
	return &PostgresClient{db: db}, nil
}

func (c *PostgresClient) Close() error {
	return c.db.Close()
}

func createPostgresTables(db *sql.DB) error {
	createSongsTable := `
    CREATE TABLE IF NOT EXISTS songs (
        id BIGINT PRIMARY KEY,
        title TEXT NOT NULL,
        artist TEXT NOT NULL,
        "ytID" TEXT, 
        key TEXT NOT NULL UNIQUE
    );`

	// Fingerprints table
	// Note: address is BIGINT (int64) to handle full range of hash values safely
	createFingerprintsTable := `
    CREATE TABLE IF NOT EXISTS fingerprints (
        address BIGINT NOT NULL,
        "anchorTimeMs" INTEGER NOT NULL,
        "songID" BIGINT NOT NULL,
        PRIMARY KEY (address, "anchorTimeMs", "songID")
    );
    
    -- Index on address for faster matching speed
    CREATE INDEX IF NOT EXISTS idx_fingerprints_address ON fingerprints (address);
    `

	if _, err := db.Exec(createSongsTable); err != nil {
		return fmt.Errorf("creating songs table: %w", err)
	}
	if _, err := db.Exec(createFingerprintsTable); err != nil {
		return fmt.Errorf("creating fingerprints table: %w", err)
	}

	return nil
}

// StoreFingerprints now accepts map[int64]... to align with BIGINT in DB
func (c *PostgresClient) StoreFingerprints(fingerprints map[int64]models.Couple) error {
	if len(fingerprints) == 0 {
		return nil
	}

	const batchSize = 20000 // Max fingerprints per batch
	
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentBatch := make(map[int64]models.Couple, batchSize)
	count := 0
	
	for address, couple := range fingerprints {
		currentBatch[address] = couple
		count++
		
		// If batch is full or we are at the end of the map
		if count == batchSize || len(currentBatch) == len(fingerprints) {
			
			// Execute batch insertion
			valueStrings := make([]string, 0, len(currentBatch))
			valueArgs := make([]any, 0, len(currentBatch) * 3)
			paramIndex := 1

			for addr, cpl := range currentBatch {
				// addr is already int64, cpl.SongId is uint32 (cast to int64)
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", paramIndex, paramIndex+1, paramIndex+2)) 
				valueArgs = append(valueArgs, addr, cpl.AnchorTime, int64(cpl.SongId))
				paramIndex += 3
			}

			insertQuery := fmt.Sprintf(`
                INSERT INTO fingerprints (address, "anchorTimeMs", "songID") 
                VALUES %s 
                ON CONFLICT (address, "anchorTimeMs", "songID") DO NOTHING
            `, strings.Join(valueStrings, ","))
			
			if _, err = tx.Exec(insertQuery, valueArgs...); err != nil {
				return err
			}

			// Reset batch
			// Note: Re-slicing fingerprints map logic needs care in Go loop, 
			// but since we iterate 'range fingerprints', we just clear 'currentBatch'.
			currentBatch = make(map[int64]models.Couple, batchSize)
			count = 0
		}
	}

	return tx.Commit()
}

// GetCouples now accepts []int64 and returns map[int64]... to match DB types
func (c *PostgresClient) GetCouples(addresses []int64) (map[int64][]models.Couple, error) {
	couples := make(map[int64][]models.Couple)

	if len(addresses) == 0 {
		return couples, nil
	}

	// The query uses ANY($1) where $1 is an array of BIGINTs
	query := `SELECT "anchorTimeMs", "songID", address FROM fingerprints WHERE address = ANY($1)`
	
	rows, err := c.db.Query(query, addresses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var couple models.Couple
		var dbSongID int64
		var dbAddress int64

		// Scan into matching Postgres types (INTEGER -> int, BIGINT -> int64)
		if err := rows.Scan(&couple.AnchorTime, &dbSongID, &dbAddress); err != nil {
			return nil, err
		}

		// Safe to cast SongID back to uint32 (it's an ID)
		couple.SongId = uint32(dbSongID)
		
		// Keep address as int64 for the map key
		couples[dbAddress] = append(couples[dbAddress], couple)
	}

	return couples, nil
}

func (c *PostgresClient) TotalSongs() (int, error) {
	var count int
	err := c.db.QueryRow(`SELECT COUNT(*) FROM songs`).Scan(&count)
	return count, err
}

func (c *PostgresClient) RegisterSong(songTitle, songArtist, ytID string) (uint32, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	songID := utils.GenerateUniqueID()
	songKey := utils.GenerateSongKey(songTitle, songArtist)

	query := `INSERT INTO songs (id, title, artist, "ytID", key) VALUES ($1, $2, $3, $4, $5)`
	
	// Explicitly cast songID (uint32) to int64 for BIGINT column
	_, err = tx.Exec(query, int64(songID), songTitle, songArtist, ytID, songKey)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return 0, fmt.Errorf("song already exists: %w", err)
		}
		return 0, fmt.Errorf("failed to insert song: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return songID, nil
}

func (c *PostgresClient) GetSong(filterKey string, value interface{}) (Song, bool, error) {
	validKeys := map[string]bool{"id": true, "ytID": true, "key": true}
	if !validKeys[filterKey] {
		return Song{}, false, fmt.Errorf("invalid filter key")
	}

	// Handle case sensitivity for ytID column
	if filterKey == "ytID" {
		filterKey = `"ytID"`
	}

	query := fmt.Sprintf(`SELECT title, artist, "ytID" FROM songs WHERE %s = $1`, filterKey)
	
	var song Song
	err := c.db.QueryRow(query, value).Scan(&song.Title, &song.Artist, &song.YouTubeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return Song{}, false, nil
		}
		return Song{}, false, err
	}

	return song, true, nil
}

// Helpers: Cast IDs to int64 where necessary
func (c *PostgresClient) GetSongByID(id uint32) (Song, bool, error) { 
	return c.GetSong("id", int64(id)) 
}

func (c *PostgresClient) GetSongByYTID(id string) (Song, bool, error) { 
	return c.GetSong("ytID", id) 
}

func (c *PostgresClient) GetSongByKey(k string) (Song, bool, error) { 
	return c.GetSong("key", k) 
}

func (c *PostgresClient) DeleteSongByID(id uint32) error {
	_, err := c.db.Exec(`DELETE FROM songs WHERE id = $1`, int64(id))
	return err
}

func (c *PostgresClient) DeleteCollection(table string) error {
	// Only allow specific tables to be dropped for safety
	if table != "songs" && table != "fingerprints" {
		return fmt.Errorf("unauthorized table drop")
	}
	_, err := c.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	return err
}