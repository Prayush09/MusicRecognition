package core_test

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"shazoom/fileformat"
	"shazoom/models"
	"testing"
)

func GetTestPath(rel string) string {
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	absPath := filepath.Join(testDir, rel)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		panic("Test file not found: " + absPath)
	}

	return absPath
}

func MakeRecordTestData(t *testing.T, filepath string) (models.RecordData, []byte) {
	wavBytes, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("error while reading WAV file: %v", err)
	}

	channels := int(binary.LittleEndian.Uint16(wavBytes[22:24]))
	sampleRate := int(binary.LittleEndian.Uint32(wavBytes[24:28]))
	sampleSize := int(binary.LittleEndian.Uint16(wavBytes[34:36]))

	byteRate := int(binary.LittleEndian.Uint32(wavBytes[28:32]))
	duration := float64(len(wavBytes)-44) / float64(byteRate)

	base64Audio := base64.StdEncoding.EncodeToString(wavBytes)

	rec := models.RecordData{
		Audio:      base64Audio,
		Duration:   duration,
		Channels:   channels,
		SampleRate: sampleRate,
		SampleSize: sampleSize,
	}

	return rec, wavBytes
}

func TestProcessRecording(t *testing.T, recData models.RecordData, wavBytes []byte) (sample []float64, SampleRate int, Duration float64) {
	samples, err := fileformat.ProcessRecording(&recData, false)
	if err != nil {
		t.Fatalf("ProcessRecording returned error: %v", err)
	}

	if len(samples) == 0 {
		t.Fatal("ProcessRecording returned zero samples")
	}

	expectedSamplesApprox := int(recData.Duration * 44100)
	tolerance := 0.1 //  10% tolerance

	minSamples := int(float64(expectedSamplesApprox) * (1 - tolerance))
	maxSamples := int(float64(expectedSamplesApprox) * (1 + tolerance))

	if len(samples) < minSamples || len(samples) > maxSamples {
		t.Logf("Expected approximately %d samples (±10%%), got %d", expectedSamplesApprox, len(samples))
		t.Logf("Duration: %.2f seconds, Sample rate: %d Hz", recData.Duration, recData.SampleRate)
	}

	t.Logf("Successfully processed %d samples from %.2f second recording", len(samples), recData.Duration)
	return samples, recData.SampleRate, float64(recData.Duration)
}

func LoadRealAudio(t *testing.T) ([]float64, int, float64) {
	if err := os.MkdirAll("tmp", 0755); err != nil {
		t.Fatalf("Failed to create tmp directory: %v", err)
	}
	defer os.RemoveAll("tmp")

	path := GetTestPath("testdata/sample3.mp3")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Test file does not exist: %s", path)
	}

	wavPath, err := fileformat.ConvertToWAV(path, 1)
	if err != nil {
		t.Fatalf("Failed to convert to WAV: %v", err)
	}

	recData, wavBytes := MakeRecordTestData(t, wavPath)

	samples, SampleRate, Duration := TestProcessRecording(t, recData, wavBytes)

	return samples, SampleRate, Duration
}
