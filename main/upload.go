package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"

	"shazam/main/db"
)

// Struct definitions that match the database models
type Song struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	FilePath    string    `json:"file_path"`
	Duration    float64   `json:"duration"`
	SampleRate  int       `json:"sample_rate"`
	FileFormat  string    `json:"file_format"`
}


func LoadWAVFile(fp string) ([]int16, int, error) {
	file, err := os.Open(fp)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open wav file: %w", err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid wav file")
	}

	format := decoder.Format()
	sampleRate := int(format.SampleRate)

	fmt.Printf("DEBUG: Format - SampleRate: %d, Channels: %d\n", format.SampleRate, format.NumChannels)

	var audioData []int16
	bufferSize := 8192
	buffer := &audio.IntBuffer{
		Data:   make([]int, bufferSize),
		Format: format,
	}

	totalSamples := 0

	for {
		n, err := decoder.PCMBuffer(buffer)
		fmt.Printf("DEBUG: PCMBuffer returned n=%d, err=%v\n", n, err)

		if err != nil {
			if err == io.EOF {
				fmt.Println("DEBUG: Reached EOF")
				break
			}
			return nil, 0, fmt.Errorf("error reading PCM data: %w", err)
		}

		totalSamples += n

		for i := range n {
			sample := int16(buffer.Data[i])
			audioData = append(audioData, sample)
		}

		if n < bufferSize {
			fmt.Printf("DEBUG: Read less than buffer size, breaking (n=%d)\n", n)
			break
		}
	}

	fmt.Printf("DEBUG: Total samples read: %d, Final audioData length: %d\n", totalSamples, len(audioData))

	if format.NumChannels == 2 {
		fmt.Println("DEBUG: Converting stereo to mono")
		audioData = StereoToMono(audioData)
		fmt.Printf("DEBUG: After mono conversion: %d samples\n", len(audioData))
	}

	return audioData, sampleRate, nil
}

func LoadMP3File(fp string) ([]int16, int, error) {
	file, err := os.Open(fp)
	if err != nil {
		return nil, 0, fmt.Errorf("error while accessing the mp3 file")
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, 0, fmt.Errorf("error while decoding the mp3 file")
	}

	sampleRate := decoder.SampleRate()
	bufferSize := 8192
	buffer := make([]byte, bufferSize)
	var audioData []int16

	for {
		n, err := decoder.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, fmt.Errorf("error reading MP3 data: %w", err)
		}

		for i := 0; i+1 < n; i += 2 {
			sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))
			audioData = append(audioData, sample)
		}
	}

	audioData = StereoToMono(audioData)

	return audioData, sampleRate, nil
}

func StereoToMono(stereoData []int16) []int16 {
	if len(stereoData)%2 != 0 {
		// Handle odd-length data
		stereoData = stereoData[:len(stereoData)-1]
	}

	monoData := make([]int16, len(stereoData)/2)

	for i := 0; i < len(monoData); i++ {
		left := int32(stereoData[i*2])    // Left channel
		right := int32(stereoData[i*2+1]) // Right channel

		// Average the two channels
		average := (left + right) / 2
		monoData[i] = int16(average)
	}

	return monoData
}

// Fixed StoreSongInDB function
func StoreSongInDB(song Song, sampleRate int) uint {
	// Convert your Song struct to the database Song struct
	dbSong := db.Song{
		Title:      song.Title,
		Artist:     song.Artist,
		FilePath:   song.FilePath,
		Duration:   song.Duration,
		SampleRate: sampleRate,
	}
	
	return db.StoreSongInDB(dbSong, sampleRate)
}

// Fixed StoreFingerprintsInDB function
func StoreFingerprintsInDB(songID uint, fingerprints []Fingerprint) {
	// Convert your Fingerprint structs to database Fingerprint structs
	var dbFingerprints []db.Fingerprint
	
	for _, fp := range fingerprints {
		dbFingerprints = append(dbFingerprints, db.Fingerprint{
			Hash:       int64(fp.Hash),
			SongID:     songID,
			TimeStamp: fp.TimeStamp, 
		})
	}
	
	db.StoreFingerprintsInDB(songID, dbFingerprints)
}

func ProcessUploadedSong(fp, title, artist string) error {
	fmt.Println("Processing song from this path: ", fp)

	var audioData []int16
	var sampleRate int
	var err error

	ext := strings.ToLower(filepath.Ext(fp))

	switch ext {
	case ".wav":
		audioData, sampleRate, err = LoadWAVFile(fp)
		if err != nil {
			return fmt.Errorf("error occurred while calling load wav function: %w", err)
		}

	case ".mp3":
		audioData, sampleRate, err = LoadMP3File(fp)
		if err != nil {
			return fmt.Errorf("error occurred while calling load mp3 function: %w", err)
		}
	
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	// Fingerprinting Process
	classifiedPeaks := FFT(audioData, sampleRate)
	allPeaks := FlattenPeaks(classifiedPeaks, 864, sampleRate)
	constellationMap := CreateConstellationMap(allPeaks)
	hashes := GenerateHashes(constellationMap)

	song := Song{
		Title:    title,
		Artist:   artist,
		FilePath: fp,
		Duration: float64(len(audioData)) / float64(sampleRate),
	}

	// Store in database - now returns uint instead of int
	songId := StoreSongInDB(song, sampleRate)
	StoreFingerprintsInDB(songId, hashes)

	ExportFingerprints(allPeaks, constellationMap, hashes, "techno-fingerprints.json")
	fmt.Printf("Successfully processed: %s by %s\n", title, artist)
	return nil
}

func ExportFingerprints(peaks []Peak, pairs []ConstellationPair, fingerprints []Fingerprint, filename string) {
	homeDir, _ := os.UserHomeDir()
	desktopPath := filepath.Join(homeDir, "Desktop", filename)

	data := map[string]interface{}{
		"peaks":        peaks,
		"pairs":        pairs,
		"fingerprints": fingerprints,
	}

	jsonData, _ := json.Marshal(data)
	err := os.WriteFile(desktopPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Printf("Exported fingerprints to Desktop/%s\n", filename)
}