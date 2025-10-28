package test

import (
	"encoding/base64"
	"shazoom/fileformat"
	"shazoom/models"
	"testing"
)

// // Helper to create a small test WAV byte slice (1-second mono 16-bit silence at 44.1kHz)
// func generateTestWavData(durationSeconds int, sampleRate int, channels int) []byte {
// 	numSamples := durationSeconds * sampleRate * channels
// 	data := make([]byte, numSamples*2) // 2 bytes per sample for 16-bit
// 	return data
// }

// func TestWriteAndReadWavFile(t *testing.T) {
// 	// Arrange
// 	tmpFile := filepath.Join(os.TempDir(), "test.wav")
// 	defer os.Remove(tmpFile)

// 	sampleRate := 44100
// 	channels := 1
// 	bitsPerSample := 16
// 	data := generateTestWavData(1, sampleRate, channels)

// 	// Act: Write WAV file
// 	err := fileformat.WriteWavFile(tmpFile, data, sampleRate, channels, bitsPerSample)
// 	if err != nil {
// 		t.Fatalf("Failed to write WAV file: %v", err)
// 	}

// 	// Act: Read WAV info
// 	info, err := fileformat.ReadWavInfo(tmpFile)
// 	if err != nil {
// 		t.Fatalf("Failed to read WAV file: %v", err)
// 	}

// 	// Assert
// 	if info.SampleRate != sampleRate {
// 		t.Errorf("Expected sample rate %d, got %d", sampleRate, info.SampleRate)
// 	}
// 	if info.Channels != channels {
// 		t.Errorf("Expected channels %d, got %d", channels, info.Channels)
// 	}
// 	if len(info.Data) != len(data) {
// 		t.Errorf("Expected data length %d, got %d", len(data), len(info.Data))
// 	}
// }

// func TestWavBytesToSample(t *testing.T) {
// 	data := []byte{0x00, 0x00, 0xFF, 0x7F, 0x00, 0x80} // 3 samples: 0, 32767, -32768

// 	samples, err := fileformat.WavBytesToSample(data)
// 	if err != nil {
// 		t.Fatalf("Failed to convert bytes to samples: %v", err)
// 	}

// 	expected := []float64{0.0, 32767.0 / 32768.0, -1.0}
// 	for i, s := range samples {
// 		if s != expected[i] {
// 			t.Errorf("Sample %d: expected %f, got %f", i, expected[i], s)
// 		}
// 	}
// }

func TestGetMetadata(t *testing.T) {
	filepath := "/Users/prayushgiri/Downloads/Tum Se Teri Baaton Mein Aisa Uljha Jiya 320 Kbps.mp3"

	metadata, err := fileformat.GetMetadata(filepath)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	// Print metadata for inspection (optional)
	t.Logf("Metadata: %+v", metadata)

	// Basic checks
	if len(metadata.Streams) == 0 {
		t.Errorf("Expected at least 1 stream in metadata")
	}
	if metadata.Format.FormFilename == "" {
		t.Errorf("Expected format filename to be set")
	}
}

/*
   Audio Data :
       type RecordData struct {
       Audio      string  `json:"audio"`
       Duration   float64 `json:"duration"`
       Channels   int     `json:"channels"`
       SampleRate int  	`json:"sample_rate"`
       SampleSize int   	`json:"sample_size"`
   }

*/

func TestProcessRecording(t *testing.T) {
	// A minimal 16-bit PCM mono WAV sample (1 kHz sine wave for 0.001s) in bytes, base64-encoded
	// For real testing, replace with an actual small audio snippet
	audioBytes := []byte{0x00, 0x00, 0x10, 0x00} // minimal dummy PCM data
	audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)

	//dummy
	recData := &models.RecordData{
		Audio:      audioBase64,
		SampleRate: 8000,
		Channels:   1,
		SampleSize: 16,
	}

	// Disable saving to avoid filesystem side effects
	samples, err := fileformat.ProcessRecording(recData, false)
	if err != nil {
		t.Fatalf("ProcessRecording returned error: %v", err)
	}

	// Expect 2 samples (len(audioBytes)/2)
	expectedNumSamples := len(audioBytes) / 2
	if len(samples) != expectedNumSamples {
		t.Errorf("expected %d samples, got %d", expectedNumSamples, len(samples))
	}
}
