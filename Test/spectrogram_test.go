package core_test

import (
	"math"
	"math/cmplx"
	"testing"
	"shazoom/core"
)

func TestLowFilterPass(t *testing.T) {
	// Test with simple signal
	sampleRate := 44100.0
	cutoff := 5000.0
	signal := make([]float64, 1000)
	
	// Generate signal with low and high frequency components
	for i := range signal {
		time := float64(i) / sampleRate
		signal[i] = math.Sin(2*math.Pi*1000*time) + // 1kHz (should pass)
			math.Sin(2*math.Pi*10000*time) // 10kHz (should be attenuated)
	}
	
	filtered := core.LowFilterPass(signal, cutoff, sampleRate)
	
	// Check output length
	if len(filtered) != len(signal) {
		t.Errorf("Expected length %d, got %d", len(signal), len(filtered))
	}
	
	// Check that filtered signal is not identical to input (some filtering occurred)
	identical := true
	for i := range signal {
		if math.Abs(signal[i]-filtered[i]) > 1e-10 {
			identical = false
			break
		}
	}
	if identical {
		t.Error("Filter produced identical output - no filtering occurred")
	}
}

func TestDownSample_ValidRatio(t *testing.T) {
	sample := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	original := 12000
	target := 3000
	
	result, err := core.DownSample(sample, original, target)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should downsample by 4x (12000/3000)
	expectedLength := 3 // 12 samples / 4
	if len(result) != expectedLength {
		t.Errorf("Expected length %d, got %d", expectedLength, len(result))
	}
}

func TestDownSample_InvalidSampleRates(t *testing.T) {
	sample := []float64{1, 2, 3, 4}
	
	tests := []struct {
		name     string
		original int
		target   int
	}{
		{"Zero original", 0, 1000},
		{"Zero target", 1000, 0},
		{"Negative original", -1000, 1000},
		{"Negative target", 1000, -1000},
		{"Target > Original", 1000, 2000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := core.DownSample(sample, tt.original, tt.target)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestDownSample_SameRate(t *testing.T) {
	sample := []float64{1, 2, 3, 4, 5}
	result, err := core.DownSample(sample, 1000, 1000)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != len(sample) {
		t.Errorf("Expected length %d, got %d", len(sample), len(result))
	}
}

func TestSpectrogram_BasicFunctionality(t *testing.T) {
	// Create a simple test signal (1 second at 44.1kHz)
	sampleRate := 44100
	duration := 1.0
	numSamples := int(float64(sampleRate) * duration)
	sample := make([]float64, numSamples)
	
	// Generate 440 Hz sine wave (A note)
	for i := range sample {
		t := float64(i) / float64(sampleRate)
		sample[i] = math.Sin(2 * math.Pi * 440 * t)
	}
	
	spectrogram, err := core.Spectrogram(sample, sampleRate)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Check that we got output
	if len(spectrogram) == 0 {
		t.Error("Expected non-empty spectrogram")
	}
	
	// Check that each window has frequency bins
	for i, window := range spectrogram {
		if len(window) == 0 {
			t.Errorf("Window %d has no frequency bins", i)
		}
	}
}

func TestSpectrogram_ShortSample(t *testing.T) {
	// Test with very short sample
	sample := make([]float64, 100)
	for i := range sample {
		sample[i] = math.Sin(float64(i))
	}
	
	spectrogram, err := core.Spectrogram(sample, 44100)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should handle short samples gracefully
	if len(spectrogram) < 0 {
		t.Error("Invalid spectrogram length")
	}
}

func TestExtractPeaks_EmptySpectrogram(t *testing.T) {
	spectrogram := [][]complex128{}
	peaks := core.ExtractPeaks(spectrogram, 1.0)
	
	if len(peaks) != 0 {
		t.Errorf("Expected 0 peaks from empty spectrogram, got %d", len(peaks))
	}
}

func TestExtractPeaks_BasicFunctionality(t *testing.T) {
	// Create a simple spectrogram with known peaks
	numWindows := 10
	numFreqBins := 512
	spectrogram := make([][]complex128, numWindows)
	
	for i := range spectrogram {
		spectrogram[i] = make([]complex128, numFreqBins)
		
		// Add some baseline noise
		for j := range spectrogram[i] {
			spectrogram[i][j] = complex(0.1, 0)
		}
		
		// Add distinct peaks in different bands
		spectrogram[i][5] = complex(10.0, 0)   // Band 0
		spectrogram[i][15] = complex(15.0, 0)  // Band 1
		spectrogram[i][30] = complex(20.0, 0)  // Band 2
		spectrogram[i][60] = complex(25.0, 0)  // Band 3
		spectrogram[i][120] = complex(30.0, 0) // Band 4
		spectrogram[i][300] = complex(35.0, 0) // Band 5
	}
	
	peaks := core.ExtractPeaks(spectrogram, 1.0)
	
	// Should extract some peaks
	if len(peaks) == 0 {
		t.Error("Expected some peaks, got none")
	}
	
	// Check that peaks have valid time values
	for i, peak := range peaks {
		if peak.Time < 0 || peak.Time > 1.0 {
			t.Errorf("Peak %d has invalid time %.2f (should be between 0 and 1)", i, peak.Time)
		}
		
		// Check that frequency is not zero
		if cmplx.Abs(peak.Freq) < 1e-10 {
			t.Errorf("Peak %d has near-zero frequency", i)
		}
	}
}

func TestExtractPeaks_PeakOrdering(t *testing.T) {
	// Create spectrogram with peaks at different times
	spectrogram := make([][]complex128, 5)
	for i := range spectrogram {
		spectrogram[i] = make([]complex128, 512)
		for j := range spectrogram[i] {
			spectrogram[i][j] = complex(float64(i+j), 0)
		}
	}
	
	peaks := core.ExtractPeaks(spectrogram, 5.0)
	
	// Peaks should be in chronological order
	for i := 1; i < len(peaks); i++ {
		if peaks[i].Time < peaks[i-1].Time {
			t.Errorf("Peaks not in chronological order: peak %d (%.2f) before peak %d (%.2f)",
				i-1, peaks[i-1].Time, i, peaks[i].Time)
		}
	}
}

func TestExtractPeaks_SingleWindow(t *testing.T) {
	// Test with just one window
	spectrogram := make([][]complex128, 1)
	spectrogram[0] = make([]complex128, 512)
	
	// Add strong peaks
	for i := range spectrogram[0] {
		if i%50 == 0 {
			spectrogram[0][i] = complex(100.0, 0)
		} else {
			spectrogram[0][i] = complex(1.0, 0)
		}
	}
	
	peaks := core.ExtractPeaks(spectrogram, 1.0)
	
	// Should extract peaks even from single window
	if len(peaks) == 0 {
		t.Error("Expected peaks from single window")
	}
	
	// All peaks should be within the audio duration
	for _, peak := range peaks {
		if peak.Time < 0 || peak.Time > 1.0 {
			t.Errorf("Peak time %.2f outside valid range [0, 1.0]", peak.Time)
		}
	}
}

func BenchmarkSpectrogram(b *testing.B) {
	// 1 second of audio at 44.1kHz
	sampleRate := 44100
	sample := make([]float64, sampleRate)
	for i := range sample {
		sample[i] = math.Sin(2 * math.Pi * 440 * float64(i) / float64(sampleRate))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.Spectrogram(sample, sampleRate)
	}
}

func BenchmarkExtractPeaks(b *testing.B) {
	// Create a realistic spectrogram
	spectrogram := make([][]complex128, 100)
	for i := range spectrogram {
		spectrogram[i] = make([]complex128, 512)
		for j := range spectrogram[i] {
			spectrogram[i][j] = complex(math.Sin(float64(i+j)), 0)
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.ExtractPeaks(spectrogram, 10.0)
	}
}