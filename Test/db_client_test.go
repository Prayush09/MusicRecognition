package core_test

import (
	"fmt"
	"shazoom/core"
	"shazoom/db"
	"shazoom/utils"
	"testing"

	"github.com/joho/godotenv"
)

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

	client, err := db.NewPostgresClient(dsn)

	if err != nil {
		t.Fatalf("Failed to connect to Cloud SQL Instance and create tables: %v", err)
	}

	defer client.Close()

	// ----------------------------------------------------------------------
	// 1. Registering a song to DB, then storing Fingerprints with songId to DB
	// ----------------------------------------------------------------------

	//Adding manually for now => TODO:  handle frontend to extract these details when user uploads a song.
	songArtist := "Sachin-Jigar"
	songName := "Tum se"
	ytId := "https://www.youtube.com/watch?v=Nnop2walGmM"

	TEST_SONG_ID, err := client.RegisterSong(songName, songArtist, ytId)
	if err != nil {
		t.Fatalf("Unable to register song to DB: %v", err)
	}

	t.Logf("Successfully registered song to DB with Song ID: %v", TEST_SONG_ID)

	// Testing of song processing and storing fingerprints
	t.Log("Starting TestFullPipeline with real audio data.")

	samples, sampleRate, duration := LoadRealAudio(t)

	if len(samples) == 0 {
		t.Fatal("FATAL: LoadRealAudio returned zero samples. Check common.go logic.")
	}

	t.Logf("Successfully fetched %d samples at %d Hz (%.2fs duration)",
		len(samples), sampleRate, duration)

	// ----------------------------------------------------------------------
	// 2. Spectrogram
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
	// 3. Peak extraction
	// ----------------------------------------------------------------------
	peaks := core.ExtractPeaks(spectrogram, duration, sampleRate)

	if len(peaks) == 0 {
		t.Fatal("No peaks extracted from spectrogram. Check peak finding logic.")
	}

	t.Logf("Extracted %d peaks from spectrogram", len(peaks))

	// ----------------------------------------------------------------------
	// 4. Verify Peak Properties
	// ----------------------------------------------------------------------
	for i, peak := range peaks {
		if peak.Time < 0 || peak.Time > duration {
			t.Errorf("Peak %d has invalid time %.2f (should be between 0 and %.2f)",
				i, peak.Time, duration)
		}

		mag := peak.Freq
		if mag < 0 { //0 can be ignored as they are often just boundary conditions, padding or noise
			t.Errorf("Peak %d has non-positive magnitude %.2f", i, mag)
		}
	}

	for i := 1; i < len(peaks); i++ {
		if peaks[i].Time < peaks[i-1].Time {
			t.Errorf("Peaks not chronologically ordered at index %d", i)
		}
	}

	// ----------------------------------------------------------------------
	// 5.Fingerprinting and storing to DB
	// ----------------------------------------------------------------------
	fingerprints, err := core.GenerateFingerprintsFromSamples(samples, sampleRate, TEST_SONG_ID)
	if err != nil {
		t.Fatalf("core.GenerateFingerprints failed: %v", err)
	}

	if len(fingerprints) == 0 {
		t.Fatal("GenerateFingerprints returned an empty map.")
	}

	hashesToLog := make([]uint32, 0, 5)
	for hash := range fingerprints {
		if len(hashesToLog) < 5 {
			hashesToLog = append(hashesToLog, hash)
		} else {
			break
		}
	}

	t.Logf("Generated %d total fingerprints. Logging %d sample hashes:", len(fingerprints), len(hashesToLog))

	for i, hash := range hashesToLog {
		t.Logf("Sample Hash #%d: 0x%08X (Decimal: %d)", i+1, hash, hash)
	}

	errStoreFingerprints := client.StoreFingerprints(fingerprints)
	if errStoreFingerprints != nil {
		t.Fatalf("Unable to store fingerprints to DB: %v", errStoreFingerprints)
	}

	t.Log("Successfully stored fingerprints to DB :)")
}
