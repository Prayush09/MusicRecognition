package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shazam/main/db"
)

const (
	SONGS_DIR = "songs"
)

// find processes audio file and searches for matches
func find(filePath string) {
	fmt.Printf("🔍 Searching for matches in: %s\n", filePath)
	
	audioData, sampleRate, peaks, hashes, err := ProcessAudioFile(filePath)
	if err != nil {
		fmt.Printf("❌ Error processing audio file: %v\n", err)
		return
	}
	
	// Convert hashes to database format for querying
	var dbHashes []db.QueryFingerprint
	for _, hash := range hashes {
		dbHashes = append(dbHashes, db.QueryFingerprint{
			Hash:      hash.Hash,
			TimeStamp: hash.TimeStamp,
		})
	}
	
	// Query for matches
	match, stats, err := ProcessQuery(audioData, sampleRate)
	if err != nil {
		fmt.Printf("❌ Error finding matches: %v\n", err)
		return
	}
	
	if match == nil {
		fmt.Println("\n❌ No match found.")
		fmt.Printf("📊 Processed %d peaks, %d hashes in %v\n", 
			stats.TotalPeaks, stats.TotalHashes, stats.ProcessingTime)
		return
	}
	
	fmt.Printf("🎉 Match found: '%s' by %s\n", match.Song.Title, match.Song.Artist)
	fmt.Printf("📈 Confidence: %.1f%%, Matches: %d\n", match.Confidence, match.MatchingHashes)
	fmt.Printf("⏱️  Search took: %v\n", stats.ProcessingTime)
}

// upload processes and saves a song to the database
func upload(filePath, title, artist string) error {
	fmt.Printf("📤 Uploading song: %s by %s\n", title, artist)
	fmt.Printf("📁 File path: %s\n", filePath)
	
	// Process the audio file
	audioData, sampleRate, peaks, hashes, err := ProcessAudioFile(filePath)
	if err != nil {
		return fmt.Errorf("error processing audio file: %w", err)
	}
	
	duration := float64(len(audioData)) / float64(sampleRate)
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Create song record
	song := Song{
		Title:      title,
		Artist:     artist,
		FilePath:   filePath,
		Duration:   duration,
		SampleRate: sampleRate,
		FileFormat: ext[1:], // Remove the dot
	}
	
	// Store in database
	fmt.Println("💾 Storing song in database...")
	songID := StoreSongInDB(song, sampleRate)
	if songID == 0 {
		return fmt.Errorf("failed to store song in database")
	}
	
	fmt.Printf("✅ Song stored with ID: %d\n", songID)
	
	// Store fingerprints
	fmt.Println("🔐 Storing fingerprints in database...")
	StoreFingerprintsInDB(songID, hashes)
	
	// Export for debugging (optional)
	ExportFingerprints(peaks, hashes, fmt.Sprintf("%s-%s-fingerprints.json", artist, title))
	
	fmt.Printf("🎉 Successfully uploaded: '%s' by %s\n", title, artist)
	fmt.Printf("📊 Stats: %.2f seconds, %d peaks, %d fingerprints\n", duration, len(peaks), len(hashes))
	
	return nil
}

// save processes files/directories and saves them to database
func save(path string, force bool) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("❌ Error accessing path %v: %v\n", path, err)
		return
	}

	if fileInfo.IsDir() {
		fmt.Printf("📁 Processing directory: %s\n", path)
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("❌ Error walking path %v: %v\n", filePath, err)
				return err
			}
			
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(filePath))
				if ext == ".wav" || ext == ".mp3" {
					err := saveSong(filePath, force)
					if err != nil {
						fmt.Printf("❌ Error saving song (%v): %v\n", filePath, err)
					}
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("❌ Error processing directory %v: %v\n", path, err)
		}
	} else {
		err := saveSong(path, force)
		if err != nil {
			fmt.Printf("❌ Error saving song (%v): %v\n", path, err)
		}
	}
}

// saveSong processes a single song file
func saveSong(filePath string, force bool) error {
	fmt.Printf("💿 Processing song: %s\n", filePath)
	
	// Extract filename as default title/artist
	fileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	parts := strings.Split(fileName, "-")
	
	var title, artist string
	if len(parts) >= 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(strings.Join(parts[1:], "-"))
	} else {
		title = fileName
		artist = "Unknown Artist"
	}
	
	// Process and upload
	return upload(filePath, title, artist)
}

// clean removes all songs and fingerprints from database
func clean() {
	fmt.Println("🧹 Cleaning database...")
	
	// Get all songs first
	songs, err := db.GetAllSongs()
	if err != nil {
		fmt.Printf("❌ Error fetching songs: %v\n", err)
		return
	}
	
	fmt.Printf("🗑️  Removing %d songs from database...\n", len(songs))
	
	// Delete each song (this will cascade delete fingerprints)
	for i, song := range songs {
		err := db.DeleteSong(song.ID)
		if err != nil {
			fmt.Printf("❌ Error deleting song %d: %v\n", song.ID, err)
		} else {
			fmt.Printf("✅ Deleted [%d/%d]: %s by %s\n", i+1, len(songs), song.Title, song.Artist)
		}
	}
	
	fmt.Println("🎉 Database cleaned successfully!")
}

// stats shows database statistics
func stats() {
	fmt.Println("📊 Database Statistics:")
	
	dbStats, err := db.GetDatabaseStats()
	if err != nil {
		fmt.Printf("❌ Error fetching stats: %v\n", err)
		return
	}
	
	fmt.Println("═══════════════════════════")
	fmt.Printf("🎵 Total songs: %v\n", dbStats["total_songs"])
	fmt.Printf("✅ Processed songs: %v\n", dbStats["processed_songs"])
	fmt.Printf("🔢 Total fingerprints: %v\n", dbStats["total_fingerprints"])
	fmt.Printf("🔍 Queries (24h): %v\n", dbStats["queries_24h"])
	fmt.Printf("📈 Avg fingerprints per song: %.1f\n", dbStats["avg_fingerprints_per_song"])
	fmt.Printf("⚡ Processing rate: %.1f%%\n", dbStats["processing_rate"])
}

// list shows all songs in database
func list() {
	songs, err := db.GetAllSongs()
	if err != nil {
		fmt.Printf("❌ Error fetching songs: %v\n", err)
		return
	}
	
	if len(songs) == 0 {
		fmt.Println("📭 No songs in database")
		return
	}
	
	fmt.Printf("\n🎵 Songs in Database (%d total):\n", len(songs))
	fmt.Println("═══════════════════════════════════════════════════════════════")
	
	for i, song := range songs {
		status := "⏳ Processing"
		if song.IsProcessed {
			status = "✅ Ready"
		}
		
		fmt.Printf("[%d] %s - %s by %s\n", i+1, status, song.Title, song.Artist)
		fmt.Printf("    📁 File: %s (%.1f MB)\n", song.FileName, float64(song.FileSize)/(1024*1024))
		fmt.Printf("    ⏱️  Duration: %.1fs | Sample Rate: %d Hz | Format: %s\n", 
			song.Duration, song.SampleRate, song.FileFormat)
		fmt.Printf("    📅 Added: %s\n", song.UploadedAt.Format("2006-01-02 15:04:05"))
		
		if i < len(songs)-1 {
			fmt.Println("    ───────────────────────────────────────────────────────────")
		}
	}
}
