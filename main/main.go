package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"shazam/main/db"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Unable to load env")
	}

	

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . record - Record 5-second snippet and search")
		fmt.Println("  go run . upload song.mp3 title artist - Process uploaded song")
		fmt.Println("  go run . stats - Show database statistics")
		fmt.Println("  go run . clean - Clean database (remove all songs)")
		fmt.Println("  go run . list - List all songs")
		return
	}

	// Initialize the optimized database with context
	ctx := context.Background()
	if err := db.InitDBGlobal(ctx); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDBGlobal()

	switch os.Args[1] {
	case "record":
		fmt.Println("Sound Recording Starts")
		audioData, sampleRate := Recording()
		fmt.Printf("Sound Recording Ended (Sample Rate: %d Hz, Duration: %.2f seconds)\n",
			sampleRate, float64(len(audioData))/float64(sampleRate))

		// Process the recorded snippet and query database
		match, stats, err := ProcessQuery(audioData, sampleRate)
		if err != nil {
			fmt.Printf("Query error: %v\n", err)
			return
		}

		// Display processing statistics
		fmt.Printf("\n Processing Stats:\n")
		fmt.Printf("   Peaks extracted: %d\n", stats.TotalPeaks)
		fmt.Printf("   Hashes generated: %d\n", stats.TotalHashes)
		fmt.Printf("   Database matches: %d\n", stats.DatabaseMatches)
		fmt.Printf("   Processing time: %v\n", stats.ProcessingTime)
		fmt.Printf("   Candidates evaluated: %d\n", stats.CandidateCount)

		if match == nil {
			fmt.Println("\n No match found!")
		} else {
			fmt.Println("\n === MATCH FOUND ===")
			fmt.Printf(" Song: %s\n", match.Song.Title)
			fmt.Printf(" Artist: %s\n", match.Song.Artist)
			fmt.Printf(" Confidence: %.1f%%\n", match.Confidence)
			fmt.Printf(" Time in song: %.1f seconds\n", match.TimeOffset)
			fmt.Printf(" Matching hashes: %d\n", match.MatchingHashes)
			fmt.Printf(" Match density: %.3f\n", match.MatchDensity)

			if match.ScoreRatio > 1.2 {
				fmt.Printf(" High confidence match (score ratio: %.2f)\n", match.ScoreRatio)
			}
		}

	case "upload":
		if len(os.Args) < 5 {
			fmt.Println("Usage: go run . upload song.mp3 'title' 'artist'")
			return
		}

		filepath := os.Args[2]
		titleOfSong := os.Args[3]
		artistName := os.Args[4]

		fmt.Printf(" Processing: %s by %s from %s\n", titleOfSong, artistName, filepath)

		err := ProcessUploadedSong(filepath, titleOfSong, artistName)
		if err != nil {
			fmt.Printf(" Error processing song: %v\n", err)
		} else {
			fmt.Printf(" Successfully processed and stored song!\n")
		}

	case "stats":
		stats, err := db.GetDatabaseStats()
		if err != nil {
			fmt.Printf(" Error fetching stats: %v\n", err)
			return
		}

		fmt.Printf("\n Database Statistics:\n")
		fmt.Printf("═══════════════════════════\n")
		fmt.Printf(" Total songs: %v\n", stats["total_songs"])
		fmt.Printf(" Processed songs: %v\n", stats["processed_songs"])
		fmt.Printf(" Total fingerprints: %v\n", stats["total_fingerprints"])
		fmt.Printf(" Queries (24h): %v\n", stats["queries_24h"])
		fmt.Printf(" Avg fingerprints per song: %.1f\n", stats["avg_fingerprints_per_song"])
		fmt.Printf(" Processing rate: %.1f%%\n", stats["processing_rate"])

	case "clean":
		fmt.Println("  This will delete ALL songs and fingerprints from the database!")
		fmt.Print("Are you sure? (yes/no): ")

		var response string
		fmt.Scanln(&response)

		if response != "yes" {
			fmt.Println(" Operation cancelled")
			return
		}

		songs, err := db.GetAllSongs()
		if err != nil {
			fmt.Printf(" Error fetching songs: %v\n", err)
			return
		}

		fmt.Printf(" Cleaning %d songs...\n", len(songs))

		for i, song := range songs {
			err := db.DeleteSong(song.ID)
			if err != nil {
				fmt.Printf(" Error deleting song %d: %v\n", song.ID, err)
			} else {
				fmt.Printf("  Deleted [%d/%d]: %s by %s\n", i+1, len(songs), song.Title, song.Artist)
			}
		}
		fmt.Println(" Database cleaned successfully!")

	case "list":
		songs, err := db.GetAllSongs()
		if err != nil {
			fmt.Printf("Error fetching songs: %v\n", err)
			return
		}

		if len(songs) == 0 {
			fmt.Println(" No songs in database")
			return
		}

		fmt.Printf("\n Songs in Database (%d total):\n", len(songs))
		fmt.Println("═══════════════════════════════════════════════════════════════")

		for i, song := range songs {
			status := " Processing"
			if song.IsProcessed {
				status = " Ready"
			}

			fmt.Printf("[%d] %s - %s by %s\n", i+1, status, song.Title, song.Artist)
			fmt.Printf("     File: %s (%.1f MB)\n", song.FileName, float64(song.FileSize)/(1024*1024))
			fmt.Printf("    ⏱  Duration: %.1fs | Sample Rate: %d Hz | Format: %s\n",
				song.Duration, song.SampleRate, song.FileFormat)
			fmt.Printf("     Added: %s\n", song.UploadedAt.Format("2006-01-02 15:04:05"))

			if i < len(songs)-1 {
				fmt.Println("    ───────────────────────────────────────────────────────────")
			}
		}

	case "test":
		// Hidden command for testing database connection
		fmt.Println(" Testing database connection...")

		stats, err := db.GetDatabaseStats()
		if err != nil {
			fmt.Printf(" Database connection test failed: %v\n", err)
			return
		}

		fmt.Printf(" Database connection successful!\n")
		fmt.Printf(" Songs: %v, Fingerprints: %v\n", stats["total_songs"], stats["total_fingerprints"])

	default:
		fmt.Printf(" Unknown command: %s\n", os.Args[1])
		fmt.Println("Available commands: 'record', 'upload', 'stats', 'clean', 'list'")
	}
}
