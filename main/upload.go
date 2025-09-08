package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"github.com/go-audio/wav"
	"io"
	"github.com/go-audio/audio"
	// "github.com/hajimehoshi/go-mp3"
	"encoding/json"
)

type Song struct {
	ID int
	Title string
	Artist string
	FilePath string
	Duration float64
}

//TODO: ADD PROCESSALLSONGSFROMDIRECTORY function, and implement LoadFromWAV and LoadFromMP3 functions.
func LoadWAVFile1(fp string) ([]int16, int, error){
	file, err := os.Open(fp)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to open WAV File: %w", fp)
	}

	defer file.Close()

	
	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("Invalid WAV File!")
	}

	//get format, create buffer and start decoding wav file
	format := decoder.Format()
	sampleRate := int(format.SampleRate)
	bitDepth := 16

	//read samples: 
	var audioData []int16
	bufferSize := 8192
	buffer := &audio.IntBuffer{
		Data: make([]int, bufferSize),
		Format: format,
	}

	for {
		n, err := decoder.PCMBuffer(buffer)
		if err != nil {
            if err == io.EOF {
                break
            }
            return nil, 0, fmt.Errorf("error reading PCM data: %w", err)
        }

		for i := 0; i < n; i++ {
            curr := int16(buffer.Data[i])  
			var sample int16
			switch bitDepth {
			case 8:
				// 8-bit samples are unsigned (0-255), convert to signed 16-bit
				sample = int16((curr - 128) * 256)
			case 16:
				// 16-bit samples, direct conversion
				sample = int16(curr)
			case 24, 32:
				// Higher bit depth, scale down to 16-bit
				sample = int16(curr / (1 << (bitDepth - 16)))
			default:
				sample = int16(curr)
			}
			
			audioData = append(audioData, sample)
        }

		if n < bufferSize {
			break
		}
	}
	
	if format.NumChannels == 2 {
		audioData = StereoToMono(audioData)
	}
	
	return audioData, sampleRate, nil
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
    
    fmt.Printf("DEBUG: Format - SampleRate: %d, Channels: %d\n", format.SampleRate, format.NumChannels)  // ADD THIS
    
    var audioData []int16
    bufferSize := 8192
    buffer := &audio.IntBuffer{
        Data:   make([]int, bufferSize),
        Format: format,
    }
    
    totalSamples := 0  // ADD THIS
    
    for {
        n, err := decoder.PCMBuffer(buffer)
        fmt.Printf("DEBUG: PCMBuffer returned n=%d, err=%v\n", n, err)  // ADD THIS
        
        if err != nil {
            if err == io.EOF {
                fmt.Println("DEBUG: Reached EOF")  
                break
            }
            return nil, 0, fmt.Errorf("error reading PCM data: %w", err)
        }
        
        totalSamples += n  // ADD THIS
        
        for i := range n {
            sample := int16(buffer.Data[i])
            audioData = append(audioData, sample)
        }
        
        if n < bufferSize {
            fmt.Printf("DEBUG: Read less than buffer size, breaking (n=%d)\n", n)  // ADD THIS
            break
        }
    }
    
    fmt.Printf("DEBUG: Total samples read: %d, Final audioData length: %d\n", totalSamples, len(audioData))  // ADD THIS
    
    if format.NumChannels == 2 {
        fmt.Println("DEBUG: Converting stereo to mono")  // ADD THIS
        audioData = StereoToMono(audioData)
        fmt.Printf("DEBUG: After mono conversion: %d samples\n", len(audioData))  // ADD THIS
    }
    
    return audioData, sampleRate, nil
}

func LoadMP3File(fp string) ([]int16, int, error){
	hello := []int16{1}
	return hello, 0, nil
}

func StereoToMono(stereoData []int16) []int16 {
	if len(stereoData)%2 != 0 {
        // Handle odd-length data
        stereoData = stereoData[:len(stereoData)-1]
    }
    
    monoData := make([]int16, len(stereoData)/2)
    
    for i := 0; i < len(monoData); i++ {
        left := int32(stereoData[i*2])      // Left channel
        right := int32(stereoData[i*2+1])   // Right channel
        
        // Average the two channels
        average := (left + right) / 2
        monoData[i] = int16(average)
    }
    
    return monoData
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
			return fmt.Errorf("error occured while calling load wav function")
			
		}

	case ".mp3":
		audioData, sampleRate, err = LoadMP3File(fp)
		if err != nil {
			return fmt.Errorf("error occured while calling load mp3 function")
		}
	}

	//Fingerprinting Process
	classifiedPeaks := FFT(audioData, sampleRate)
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

	ExportFingerprints(allPeaks, constellationMap, hashes, "techno-fingerprints.json")
	fmt.Printf("Successfully processed: %s by %s\n", title, artist)
	return nil
}



func ExportFingerprints(peaks []Peak, pairs []ConstellationPair, fingerprints []Fingerprint, filename string) {
    homeDir, _ := os.UserHomeDir()
	desktopPath := filepath.Join(homeDir, "Desktop", filename)

	data := map[string]interface{}{
        "peaks":       peaks,
        "pairs":       pairs,
        "fingerprints": fingerprints,
    }
    
    jsonData, _ := json.Marshal(data)
    err := os.WriteFile(desktopPath, jsonData, 0644)
    if err != nil {
        fmt.Printf("Error writing file: %v\n", err)
        return
    }
    
    fmt.Printf("🎵 Exported fingerprints to Desktop/%s\n", filename)
}