package main

import (
	"encoding/binary"
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

// Song struct that matches the database models
type Song struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	FilePath    string    `json:"file_path"`
	Duration    float64   `json:"duration"`
	SampleRate  int       `json:"sample_rate"`
	FileFormat  string    `json:"file_format"`
}

// LoadWAVFile loads and decodes WAV audio files
func LoadWAVFile(fp string) ([]int16, int, error) {
	fmt.Printf("📂 Loading WAV file: %s\n", fp)
	
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

	fmt.Printf("🎵 WAV Format - SampleRate: %d Hz, Channels: %d\n", 
		format.SampleRate, format.NumChannels)

	var audioData []int16
	bufferSize := 8192
	buffer := &audio.IntBuffer{
		Data:   make([]int, bufferSize),
		Format: format,
	}

	for {
		n, err := decoder.PCMBuffer(buffer)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("✅ WAV file loaded successfully\n")
				break
			}
			return nil, 0, fmt.Errorf("error reading PCM  %w", err)
		}

		for i := range n {
			sample := int16(buffer.Data[i])
			audioData = append(audioData, sample)
		}

		if n < bufferSize {
			break
		}
	}

	fmt.Printf("📊 Total samples read: %d, Duration: %.2f seconds\n", 
		len(audioData), float64(len(audioData))/float64(sampleRate))

	// Convert stereo to mono if necessary
	if format.NumChannels == 2 {
		fmt.Println("🔄 Converting stereo to mono")
		audioData = StereoToMono(audioData)
		fmt.Printf("📊 After mono conversion: %d samples\n", len(audioData))
	}

	return audioData, sampleRate, nil
}

// LoadMP3File loads and decodes MP3 audio files
func LoadMP3File(fp string) ([]int16, int, error) {
	fmt.Printf("📂 Loading MP3 file: %s\n", fp)
	
	file, err := os.Open(fp)
	if err != nil {
		return nil, 0, fmt.Errorf("error while accessing the mp3 file: %w", err)
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, 0, fmt.Errorf("error while decoding the mp3 file: %w", err)
	}

	sampleRate := decoder.SampleRate()
	fmt.Printf("🎵 MP3 Sample Rate: %d Hz\n", sampleRate)
	
	bufferSize := 8192
	buffer := make([]byte, bufferSize)
	var audioData []int16

	for {
		n, err := decoder.Read(buffer)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("✅ MP3 file loaded successfully\n")
				break
			}
			return nil, 0, fmt.Errorf("error reading MP3  %w", err)
		}

		// Convert bytes to int16 samples
		for i := 0; i+1 < n; i += 2 {
			sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))
			audioData = append(audioData, sample)
		}
	}

	// MP3 is typically stereo, convert to mono
	fmt.Println("🔄 Converting MP3 stereo to mono")
	audioData = StereoToMono(audioData)

	fmt.Printf("📊 Final audio: %d samples, Duration: %.2f seconds\n", 
		len(audioData), float64(len(audioData))/float64(sampleRate))

	return audioData, sampleRate, nil
}

// StereoToMono converts stereo audio data to mono
func StereoToMono(stereoData []int16) []int16 {
	if len(stereoData) == 0 {
		return stereoData
	}
	
	if len(stereoData)%2 != 0 {
		// Handle odd-length data
		stereoData = stereoData[:len(stereoData)-1]
	}

	monoData := make([]int16, len(stereoData)/2)

	for i := 0; i < len(monoData); i++ {
		left := int32(stereoData[i*2])    // Left channel
		right := int32(stereoData[i*2+1]) // Right channel

		// Average the two channels with overflow protection
		average := (left + right) / 2
		
		// Clamp to int16 range
		if average > 32767 {
			average = 32767
		} else if average < -32768 {
			average = -32768
		}
		
		monoData[i] = int16(average)
	}

	return monoData
}

// convertToFloat64 converts int16 audio samples to float64
func convertToFloat64(audioData []int16) []float64 {
	if len(audioData) == 0 {
		return []float64{}
	}
	
	dataFloat64 := make([]float64, len(audioData))
	for i, sample := range audioData {
		// Normalize int16 (-32768 to 32767) to float64 (-1.0 to 1.0)
		dataFloat64[i] = float64(sample) / 32768.0
	}
	return dataFloat64
}

// StoreSongInDB stores song metadata in database
func StoreSongInDB(song Song, sampleRate int) uint {
	dbSong := db.Song{
		Title:      song.Title,
		Artist:     song.Artist,
		FilePath:   song.FilePath,
		Duration:   song.Duration,
		SampleRate: sampleRate,
	}
	
	return db.StoreSongInDB(dbSong, sampleRate)
}
func ProcessAudioFile(filePath string) ([]int16, int, []Peak, []Hash, error) {
	var audioData []int16
	var sampleRate int
	var err error

	ext := strings.ToLower(filepath.Ext(filePath))

	// Load audio file based on extension
	switch ext {
	case ".wav":
		audioData, sampleRate, err = LoadWAVFile(filePath)
	case ".mp3":
		audioData, sampleRate, err = LoadMP3File(filePath)
	default:
		return nil, 0, nil, nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return nil, 0, nil, nil, fmt.Errorf("error loading audio file: %w", err)
	}

	if len(audioData) == 0 {
		return nil, 0, nil, nil, fmt.Errorf("no audio data loaded from file")
	}

	duration := float64(len(audioData)) / float64(sampleRate)
	if duration < 1.0 {
		return nil, 0, nil, nil, fmt.Errorf("audio file too short: %.2f seconds", duration)
	}

	//TODO: use the new pipeline and start testing

	return audioData, sampleRate, peaks, hashes, nil
}