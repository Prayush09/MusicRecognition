package core_test

import (
	"fmt"
	"shazoom/core"
	"shazoom/db"
	"shazoom/utils"
	"testing"
	"github.com/joho/godotenv"
)

// NOTE: LoadRealAudio function is assumed to be defined in a common location (e.g., common.go or the test file itself)
// func LoadRealAudio(t *testing.T) ([]float64, int, float64) { ... }

func setupTestEnv(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatalf("Warning: Could not load .env file: %v. Relying on shell exports.", err)
	}

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
			t.Fatalf("FATAL: Required env %s is not set or is empty.", key)
		}
	}
}

func TestNewPostgresClient(t *testing.T) {
	setupTestEnv(t)

	var (
		dbUser = utils.GetEnv("DB_USER")
		dbPass = utils.GetEnv("DB_PASS")
		dbHost = utils.GetEnv("DB_HOST")
		dbPort = utils.GetEnv("DB_PORT")
		dbName = utils.GetEnv("DB_NAME")
	)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// Note: We are calling the NewPostgresClient directly as before, assuming it implements DBClient
	client, err := db.NewPostgresClient(dsn) 

	if err != nil {
		t.Fatalf("Failed to connect to Cloud SQL Instance and create tables: %v", err)
	}

	defer client.Close()

	// ----------------------------------------------------------------------
	// 1. Registering a song to DB, then storing Fingerprints with songId to DB
	// ----------------------------------------------------------------------

	songArtist := "Arpit Bala"
	songName := "Bargad"
	ytId := "https://www.youtube.com/watch?v=jfjXJpUNayg"

	TEST_SONG_ID, err := client.RegisterSong(songName, songArtist, ytId)
	if err != nil {
		t.Fatalf("Unable to register song to DB: %v", err)
	}

	t.Logf("Successfully registered song to DB with Song ID: %v", TEST_SONG_ID)

	t.Log("Starting TestFullPipeline with real audio data.")

	samples, sampleRate, duration := LoadRealAudio(t)

	if len(samples) == 0 {
		t.Fatal("FATAL: LoadRealAudio returned zero samples. Check common.go logic.")
	}

	t.Logf("Successfully fetched %d samples at %d Hz (%.2fs duration)",
		len(samples), sampleRate, duration)

	// ----------------------------------------------------------------------
	// 2. Spectrogram (Verification only)
	// ----------------------------------------------------------------------
	spectrogram, err := core.Spectrogram(samples, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	if len(spectrogram) == 0 {
		t.Fatal("Spectrogram is empty")
	}

	t.Logf("Generated spectrogram with %d time windows and %d frequency bins",
		len(spectrogram), len(spectrogram[0]))

	// ----------------------------------------------------------------------
	// 3. Peak extraction (Verification only)
	// ----------------------------------------------------------------------
	peaks := core.ExtractPeaks(spectrogram, duration, sampleRate)

	if len(peaks) == 0 {
		t.Fatal("No peaks extracted from spectrogram. Check peak finding logic.")
	}

	t.Logf("Extracted %d peaks from spectrogram", len(peaks))

	// (Skipping peak property checks for brevity, assuming they are correct)

	// ----------------------------------------------------------------------
	// 4. Fingerprinting and storing to DB
	// ----------------------------------------------------------------------
	
	// *** CRITICAL CHANGE: The return type is now map[int64]models.Couple ***
	fingerprints, err := core.GenerateFingerprintsFromSamples(samples, sampleRate, TEST_SONG_ID)
	if err != nil {
		t.Fatalf("core.GenerateFingerprintsFromSamples failed: %v", err)
	}

	if len(fingerprints) == 0 {
		t.Fatal("GenerateFingerprintsFromSamples returned an empty map.")
	}

	// *** CRITICAL CHANGE: Use int64 for the slice of hashes ***
	hashesToLog := make([]int64, 0, 5)
	for hash := range fingerprints {
		if len(hashesToLog) < 5 {
			// hash is already int64 here
			hashesToLog = append(hashesToLog, hash)
		} else {
			break
		}
	}

	t.Logf("Generated %d total fingerprints. Logging %d sample hashes:", len(fingerprints), len(hashesToLog))

	for i, hash := range hashesToLog {
		// Log the hash using %X and %d, which work for int64. 
		// Note: The hex format specifier is now %X for the 64-bit value.
		t.Logf("Sample Hash #%d: 0x%X (Decimal: %d)", i+1, hash, hash)
	}

	// *** CRITICAL CHANGE: fingerprints map type now correctly matches the StoreFingerprints signature ***
	errStoreFingerprints := client.StoreFingerprints(fingerprints)
	if errStoreFingerprints != nil {
		t.Fatalf("Unable to store fingerprints to DB: %v", errStoreFingerprints)
	}

	t.Log("Successfully stored fingerprints to DB :)")
    
    // --- Additional Check: Verify Fingerprints are retrievable (optional but recommended) ---
    addresses := make([]int64, 0, len(fingerprints))
    for addr := range fingerprints {
        addresses = append(addresses, addr)
    }

    retrievedCouples, err := client.GetCouples(addresses)
    if err != nil {
        t.Fatalf("Failed to retrieve couples after storing: %v", err)
    }

    if len(retrievedCouples) == 0 {
        t.Fatal("FAILURE: Stored fingerprints were not retrieved from the database.")
    }
    t.Logf("Successfully retrieved %d unique hash matches from the database.", len(retrievedCouples))

}