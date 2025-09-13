package main

import (
	"fmt"
	"os"
	"log"
	"github.com/joho/godotenv"
	"shazam/main/db"
)

var sampleRate int

func main(){

	if err := godotenv.Load(); err != nil {
		log.Fatal(("Unable to load env"))
	}

	if err := db.InitDB(); err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer db.CloseDB()

	//TODO: add better logic of switching instead of relying on the no. of arguments passed
	 if len(os.Args) < 2 {
        fmt.Println("Usage:")
        fmt.Println("  go run . record           - Record 5-second snippet")
        fmt.Println("  go run . upload song.mp3 title artist - Process uploaded song")
        return
    }
    
    switch os.Args[1] {
		case "record":
			
			fmt.Println("Sound Recording Starts")
			audioData := Recording()
			fmt.Println("Sound Recording Ended")

			fmt.Println("FFT Processing Begins")
			
			classifiedPeaks := FFT(audioData, 0)
			fmt.Println("FFT Processing Ends")

			
			fmt.Println("Create Constellation Map")
			allPeaks := FlattenPeaks(classifiedPeaks, 864, sampleRate)
			fmt.Printf("Total Peaks found: %d\n", len(allPeaks))

			constellationMap := CreateConstellationMap(allPeaks)
			fmt.Println("Constellation map created")

			
			hashes := GenerateHashes(constellationMap)
			fmt.Printf("Hashed Created: %d", len(hashes))
			fmt.Println("Just a fun commit")
	
    case "upload":
        // New upload processing logic
        filepath := os.Args[2]
		titleOfSong := os.Args[3]
		artistName := os.Args[4]

        ProcessUploadedSong(filepath, titleOfSong, artistName)
		
	}	
}