package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Song struct {
	ID int
	Title string
	Artist string
	FilePath string
	Duration float64
}

//TODO: ADD PROCESSALLSONGSFROMDIRECTORY function, and implement LoadFromWAV and LoadFromMP3 functions.
func LoadWAVFile(fp string) ([]int16, int, error){

}

func LoadMP3File(fp string) ([]int16, int, error){
	
}

func StoreSongInDB(song Song) int {
	// TODO: Implement with Prisma
	// INSERT INTO songs (title, artist, file_path, duration) VALUES (...)
	// RETURN song_id
	fmt.Printf("Storing song: %s by %s\n", song.Title, song.Artist)
	return 1 // Placeholder
}

func StoreFingerprintsInDB(songID int, fingerprints []Fingerprint) {
	// TODO: Implement with Prisma
	// Batch INSERT INTO fingerprints (hash, song_id, time_offset) VALUES (...)
	fmt.Printf("Storing %d fingerprints for song ID %d\n", len(fingerprints), songID)
}

func ProcessAudioFile(fp, title, artist string) error {
	fmt.Println("Processing song from this path: ", fp)

	var audioData []int16
	var sampleRate int
	var err error

	ext := strings.ToLower(filepath.Ext(fp))
	
	switch ext {

	case ".wav":
		audioData, sampleRate, err = LoadWAVFile(fp)

	case ".mp3":
		audioData, sampleRate, err = LoadMP3File(fp)
	}

	//Fingerprinting Process
	classifiedPeaks := FFT(audioData)
	allPeaks := FlattenPeaks(classifiedPeaks, 864, sampleRate) //Chunk Size : 864
	constellationMap := CreateConstellationMap(allPeaks)
	hashes := GenerateHashes(constellationMap)


	song := Song{
		Title: title,
		Artist: artist,
		FilePath: fp,
		Duration: float64(len(audioData))/float64(sampleRate),
	}

	songId := StoreSongInDB(song)
	StoreFingerprintsInDB(songId, hashes)

	fmt.Println("Successfully processed: %s by %s\n", title, artist)
}

