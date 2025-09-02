package main

import (
	"fmt"
	"os"
)

func main(){

	//TODO: add better logic of switching instead of relying on the no. of arguments passed
	 if len(os.Args) < 2 {
        fmt.Println("Usage:")
        fmt.Println("  go run . record           - Record 5-second snippet")
        fmt.Println("  go run . upload song.mp3  - Process uploaded song")
        return
    }
    
    switch os.Args[1] {
		case "record":
			//recording 
		fmt.Println("Sound Recording Starts")
		audioData := Recording()
		fmt.Println("Sound Recording Ended")

		fmt.Println("FFT Processing Begins")
		//FFT Processing
		classifiedPeaks := FFT(audioData)
		fmt.Println("FFT Processing Ends")

		//constellation map 
		fmt.Println("Create Constellation Map")
		allPeaks := FlattenPeaks(classifiedPeaks)
		fmt.Println("Total Peaks found: %d", allPeaks)

		constellationMap := CreateConstellationMap(allPeaks)
		fmt.Println("Constellation map created")

		//hashing
		hashes := GenerateHashes(constellationMap)
		fmt.Println("Hashed Created: %d", hashes)
		fmt.Println("Just a fun commit")
	
    case "upload":
        // New upload processing logic
        filepath := os.Args[2]
        ProcessUploadedSong(filepath)
    
	}	
}