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

func NewPostgresClient(dsn string) (*PostgresClient, error) {
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("error opening postgres connection: %w", err)
    }

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("error connecting to postgres: %w", err)
    }

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

    createFingerprintsTable := `
    CREATE TABLE IF NOT EXISTS fingerprints (
        address BIGINT NOT NULL,
        "anchorTimeMs" INTEGER NOT NULL,
        "songID" BIGINT NOT NULL,
        PRIMARY KEY (address, "anchorTimeMs", "songID")
    );
    
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

func (c *PostgresClient) StoreFingerprints(fingerprints map[int64]models.Couple) error {
    if len(fingerprints) == 0 {
        return nil
    }

    const batchSize = 20000 
    
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
        
        if count == batchSize || len(currentBatch) == len(fingerprints) {
            
            valueStrings := make([]string, 0, len(currentBatch))
            valueArgs := make([]any, 0, len(currentBatch) * 3)
            paramIndex := 1

            for addr, cpl := range currentBatch {
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

            currentBatch = make(map[int64]models.Couple, batchSize)
            count = 0
        }
    }

    return tx.Commit()
}

func (c *PostgresClient) GetCouples(addresses []int64) (map[int64][]models.Couple, error) {
    couples := make(map[int64][]models.Couple)

    if len(addresses) == 0 {
        return couples, nil
    }

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

        if err := rows.Scan(&couple.AnchorTime, &dbSongID, &dbAddress); err != nil {
            return nil, err
        }

        couple.SongId = uint32(dbSongID)
        
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
    if table != "songs" && table != "fingerprints" {
        return fmt.Errorf("unauthorized table drop")
    }
    _, err := c.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
    return err
}