package test

import (
	"math"
	"math/cmplx"
	"shazoom/core"
	"testing"
)

// TestFullPipeline tests the entire audio fingerprinting pipeline
func TestFullPipeline(t *testing.T) {
	// Generate a 2-second audio signal with multiple frequency components
	sampleRate := 44100
	duration := 2.0
	numSamples := int(float64(sampleRate) * duration)

	sample := make([]float64, numSamples)

	// Create a complex signal with multiple frequencies
	// 440 Hz (A4), 880 Hz (A5), and 1760 Hz (A6)
	for i := range sample {
		t := float64(i) / float64(sampleRate)
		sample[i] = 0.5*math.Sin(2*math.Pi*440*t) +
			0.3*math.Sin(2*math.Pi*880*t) +
			0.2*math.Sin(2*math.Pi*1760*t)
	}

	// Step 1: Generate spectrogram
	spectrogram, err := core.Spectrogram(sample, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	if len(spectrogram) == 0 {
		t.Fatal("Spectrogram is empty")
	}

	t.Logf("Generated spectrogram with %d time windows and %d frequency bins",
		len(spectrogram), len(spectrogram[0]))

	// Step 2: Extract peaks
	peaks := core.ExtractPeaks(spectrogram, duration)

	if len(peaks) == 0 {
		t.Fatal("No peaks extracted from spectrogram")
	}

	t.Logf("Extracted %d peaks from spectrogram", len(peaks))

	// Step 3: Verify peak properties
	for i, peak := range peaks {
		// Check time is within valid range
		if peak.Time < 0 || peak.Time > duration {
			t.Errorf("Peak %d has invalid time %.2f (should be between 0 and %.2f)",
				i, peak.Time, duration)
		}

		// Check frequency magnitude is positive
		mag := cmplx.Abs(peak.Freq)
		if mag <= 0 {
			t.Errorf("Peak %d has non-positive magnitude %.2f", i, mag)
		}
	}

	// Step 4: Verify peaks are in chronological order
	for i := 1; i < len(peaks); i++ {
		if peaks[i].Time < peaks[i-1].Time {
			t.Errorf("Peaks not chronologically ordered at index %d", i)
		}
	}
}

// TestPipelineWithSilence tests the pipeline with silent audio
func TestPipelineWithSilence(t *testing.T) {
	sampleRate := 44100
	duration := 1.0
	numSamples := int(float64(sampleRate) * duration)

	// Create silent sample
	sample := make([]float64, numSamples)

	spectrogram, err := core.Spectrogram(sample, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	peaks := core.ExtractPeaks(spectrogram, duration)

	// Silent audio should produce few or no significant peaks
	if len(peaks) > 10 {
		t.Logf("Warning: Silent audio produced %d peaks (expected few)", len(peaks))
	}
}

// TestPipelineWithWhiteNoise tests the pipeline with random noise
func TestPipelineWithWhiteNoise(t *testing.T) {
	sampleRate := 44100
	duration := 1.0
	numSamples := int(float64(sampleRate) * duration)

	sample := make([]float64, numSamples)
	for i := range sample {
		// Generate white noise between -1 and 1
		sample[i] = 2*math.Sin(float64(i*i)) - 1
	}

	spectrogram, err := core.Spectrogram(sample, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	peaks := core.ExtractPeaks(spectrogram, duration)

	t.Logf("White noise produced %d peaks", len(peaks))

	// Should still produce some peaks
	if len(peaks) == 0 {
		t.Error("Expected some peaks from white noise")
	}
}

// TestPipelineWithChirp tests with a frequency sweep
func TestPipelineWithChirp(t *testing.T) {
	sampleRate := 44100
	duration := 2.0
	numSamples := int(float64(sampleRate) * duration)

	sample := make([]float64, numSamples)

	// Create a chirp from 100 Hz to 2000 Hz
	startFreq := 100.0
	endFreq := 2000.0

	for i := range sample {
		t := float64(i) / float64(sampleRate)
		// Linear frequency sweep
		freq := startFreq + (endFreq-startFreq)*t/duration
		sample[i] = math.Sin(2 * math.Pi * freq * t)
	}

	spectrogram, err := core.Spectrogram(sample, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	peaks := core.ExtractPeaks(spectrogram, duration)

	if len(peaks) == 0 {
		t.Error("Expected peaks from chirp signal")
	}

	t.Logf("Chirp signal produced %d peaks", len(peaks))
}

// TestComponentIntegration tests individual components work together
func TestComponentIntegration(t *testing.T) {
	// Create test signal
	sampleRate := 44100
	sample := generateTestSignal(sampleRate, 1.0, 440.0)

	t.Run("LowPassFilter", func(t *testing.T) {
		filtered := core.LowFilterPass(sample, 5000.0, float64(sampleRate))
		if len(filtered) != len(sample) {
			t.Errorf("Filter changed sample length: %d -> %d", len(sample), len(filtered))
		}
	})

	t.Run("Downsample", func(t *testing.T) {
		targetRate := sampleRate / 4
		downsampled, err := core.DownSample(sample, sampleRate, targetRate)
		if err != nil {
			t.Fatalf("Downsample failed: %v", err)
		}

		expectedLength := len(sample) / 4
		if math.Abs(float64(len(downsampled)-expectedLength)) > 1 {
			t.Errorf("Expected ~%d samples after downsampling, got %d",
				expectedLength, len(downsampled))
		}
	})

	t.Run("FFT", func(t *testing.T) {
		// Take a window of samples
		windowSize := 1024
		if len(sample) < windowSize {
			t.Skip("Sample too small for FFT test")
		}

		window := sample[:windowSize]
		fftResult := core.FFT(window)

		if len(fftResult) != windowSize {
			t.Errorf("FFT changed sample length: %d -> %d", windowSize, len(fftResult))
		}
	})

	t.Run("FilterThenDownsample", func(t *testing.T) {
		filtered := core.LowFilterPass(sample, 5000.0, float64(sampleRate))
		targetRate := sampleRate / 4
		downsampled, err := core.DownSample(filtered, sampleRate, targetRate)

		if err != nil {
			t.Fatalf("Combined filter+downsample failed: %v", err)
		}

		if len(downsampled) == 0 {
			t.Error("Combined operation produced empty result")
		}
	})
}

// TestPipelineConsistency verifies that running the pipeline twice gives consistent results
func TestPipelineConsistency(t *testing.T) {
	sampleRate := 44100
	sample := generateTestSignal(sampleRate, 1.0, 440.0)

	// Run pipeline twice
	spec1, err1 := core.Spectrogram(sample, sampleRate)
	spec2, err2 := core.Spectrogram(sample, sampleRate)

	if err1 != nil || err2 != nil {
		t.Fatalf("Pipeline errors: %v, %v", err1, err2)
	}

	// Check spectrograms are identical
	if len(spec1) != len(spec2) {
		t.Errorf("Inconsistent spectrogram lengths: %d vs %d", len(spec1), len(spec2))
	}

	for i := range spec1 {
		if len(spec1[i]) != len(spec2[i]) {
			t.Errorf("Window %d has inconsistent length: %d vs %d",
				i, len(spec1[i]), len(spec2[i]))
		}

		for j := range spec1[i] {
			if spec1[i][j] != spec2[i][j] {
				t.Errorf("Mismatch at [%d][%d]: %v vs %v",
					i, j, spec1[i][j], spec2[i][j])
			}
		}
	}
}

// Helper function to generate test signals
func generateTestSignal(sampleRate int, duration, frequency float64) []float64 {
	numSamples := int(float64(sampleRate) * duration)
	signal := make([]float64, numSamples)

	for i := range signal {
		t := float64(i) / float64(sampleRate)
		signal[i] = math.Sin(2 * math.Pi * frequency * t)
	}

	return signal
}

// BenchmarkFullPipeline benchmarks the entire pipeline
func BenchmarkFullPipeline(b *testing.B) {
	sampleRate := 44100
	sample := generateTestSignal(sampleRate, 2.0, 440.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spectrogram, _ := core.Spectrogram(sample, sampleRate)
		core.ExtractPeaks(spectrogram, 2.0)
	}
}
