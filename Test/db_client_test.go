package core_test

import (
	"fmt"
	"shazoom/db"
	"shazoom/utils"
	"testing"

	"github.com/joho/godotenv"
)

// NOTE: This setupTestEnv function remains correct for validating environment variables.
func setupTestEnv(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatalf("Warning: Could not load .env file: %v. Relying on shell exports.", err)
	}
	// FIX: Change all keys to be all uppercase to match the shell export
	DB_HOST := utils.GetEnv("DB_HOST")
	DB_PORT := utils.GetEnv("DB_PORT")
	DB_PASS := utils.GetEnv("DB_PASS")
	DB_NAME := utils.GetEnv("DB_NAME")
	DB_USER := utils.GetEnv("DB_USER")
	vars := map[string]string{
		"DB_HOST": DB_HOST,
		"DB_PORT": DB_PORT,
		"DB_PASS": DB_PASS,
		"DB_NAME": DB_NAME,
		"DB_USER": DB_USER,
	}
	for key, val := range vars {
		if val == "" {
			// Note: We are using val here, which is the return from utils.GetEnv
			t.Fatalf("FATAL: Required env %s is not set or is empty.", key)
		}
	}
}

func TestNewPostgresClient(t *testing.T) {
	// 1. Validate Environment Variables
	setupTestEnv(t)

	// 2. Retrieve variables using the correct utils.GetEnv() with no fallback,
	// since setupTestEnv already verified they exist.
	var (
		dbUser = utils.GetEnv("DB_USER")
		dbPass = utils.GetEnv("DB_PASS")
		dbHost = utils.GetEnv("DB_HOST")
		dbPort = utils.GetEnv("DB_PORT")
		dbName = utils.GetEnv("DB_NAME")
	)

	// 3. Construct the DSN
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// 4. Connect and Execute Table Creation
	// We assume NewPostgresClient handles: connection, ping, and createPostgresTables(db)
	client, err := db.NewPostgresClient(dsn)

	// 5. ASSERTION 1: Check for Connection/Table Creation Errors
	if err != nil {
		t.Fatalf("Failed to connect to Cloud SQL Instance and create tables: %v", err)
	}

	// 6. RESOURCE CLEANUP: Defer must be set immediately after successful connection
	defer client.Close()

	// 7. ASSERTION 2 (Optional but Recommended): Verify a basic query runs
	// Since we can't fully check table existence without knowing your interface,
	// a basic query is a good proxy. Assuming DBClient has a TotalSongs method:
	totalSongs, err := client.TotalSongs()
	if err != nil {
		t.Fatalf("Failed to run a basic query (e.g., TotalSongs). Tables might not be created: %v", err)
	}

	t.Logf("Successfully connected, tables created, and database is queryable. Total songs: %d", totalSongs)
}

/* 
	=== RUN   TestNewPostgresClient
successfully created postgreSQL client and created tables    /Users/prayushgiri/Projects/Shazm Music Algorithm Project/test/db_client_test.go:77: Successfully connected, tables created, and database is queryable. Total songs: 0
--- PASS: TestNewPostgresClient (0.54s)
PASS
ok      shazoom/test    1.021s
*/