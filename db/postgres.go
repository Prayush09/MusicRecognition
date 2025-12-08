package db

import (
	//"context"
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

	fmt.Printf("successfully created postgreSQL client and created tables")
	return &PostgresClient{db: db}, nil
}

func (c *PostgresClient) Close() error {
	return c.db.Close()
}

func createPostgresTables(db *sql.DB) error {
	// Postgres uses SERIAL for auto-increment and specific types
	createSongsTable := `
	CREATE TABLE IF NOT EXISTS songs (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		artist TEXT NOT NULL,
		"ytID" TEXT, 
		key TEXT NOT NULL UNIQUE
	);`

	// Fingerprints table
	// Note: We use a composite primary key to avoid exact duplicates
	createFingerprintsTable := `
	CREATE TABLE IF NOT EXISTS fingerprints (
		address INTEGER NOT NULL,
		"anchorTimeMs" INTEGER NOT NULL,
		"songID" INTEGER NOT NULL,
		PRIMARY KEY (address, "anchorTimeMs", "songID")
	);
	
	-- Optional: Create an index on address for faster matching speed
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

func (c *PostgresClient) StoreFingerprints(fingerprints map[uint32]models.Couple) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Postgres uses $1, $2, $3 syntax
	// ON CONFLICT DO NOTHING handles duplicates gracefully
	stmt, err := tx.Prepare(`
		INSERT INTO fingerprints (address, "anchorTimeMs", "songID") 
		VALUES ($1, $2, $3) 
		ON CONFLICT (address, "anchorTimeMs", "songID") DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for address, couple := range fingerprints {
		if _, err := stmt.Exec(address, couple.AnchorTime, couple.SongId); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *PostgresClient) GetCouples(addresses []uint32) (map[uint32][]models.Couple, error) {
	couples := make(map[uint32][]models.Couple)

	// Since we can't easily pass a slice to SQL, we loop or use specific handling.
	// For simplicity in this implementation, we simply query. 
	// PERFORMANCE NOTE: For high volume, use "github.com/lib/pq".Array() or build a dynamic IN clause.
	// Below is a dynamic IN clause builder.

	if len(addresses) == 0 {
		return couples, nil
	}

	// Build query: SELECT ... WHERE address IN ($1, $2, $3...)
	query := `SELECT "anchorTimeMs", "songID", address FROM fingerprints WHERE address = ANY($1)`
	
	// Postgres driver can handle slices with ANY() if passed correctly, 
	// but strictly implementation-dependent. The safest standard way for pgx/stdlib:
	
	rows, err := c.db.Query(query, addresses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var couple models.Couple
		var address uint32
		if err := rows.Scan(&couple.AnchorTime, &couple.SongId, &address); err != nil {
			return nil, err
		}
		couples[address] = append(couples[address], couple)
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

	songID := utils.GenerateUniqueID() // Assuming this returns uint32
	songKey := utils.GenerateSongKey(songTitle, songArtist)

	// Note: We insert our own Generated ID, ensuring the DB respects it.
	// Postgres syntax uses $ placeholders.
	query := `INSERT INTO songs (id, title, artist, "ytID", key) VALUES ($1, $2, $3, $4, $5)`
	
	_, err = tx.Exec(query, songID, songTitle, songArtist, ytID, songKey)
	if err != nil {
		// Basic duplicate check based on error string, or use pgx specific error codes
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
	// Whitelist keys to prevent SQL injection
	validKeys := map[string]bool{"id": true, "ytID": true, "key": true}
	if !validKeys[filterKey] {
		return Song{}, false, fmt.Errorf("invalid filter key")
	}

	// Note the quotes around "ytID" because mixed case columns in Postgres need quotes
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

// Wrapper methods to satisfy interface
func (c *PostgresClient) GetSongByID(id uint32) (Song, bool, error) { return c.GetSong("id", id) }
func (c *PostgresClient) GetSongByYTID(id string) (Song, bool, error) { return c.GetSong("ytID", id) }
func (c *PostgresClient) GetSongByKey(k string) (Song, bool, error) { return c.GetSong("key", k) }

func (c *PostgresClient) DeleteSongByID(id uint32) error {
	_, err := c.db.Exec(`DELETE FROM songs WHERE id = $1`, id)
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