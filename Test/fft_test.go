package test

import (
	"math"
	"math/cmplx"
	"shazoom/core"
	"testing"
)

func TestFFT_BasicSignal(t *testing.T) {
	// Test with a simple sine wave at known frequency
	sampleRate := 1000.0
	frequency := 10.0 // 10 Hz
	numSamples := 64

	signal := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		signal[i] = math.Sin(2 * math.Pi * frequency * float64(i) / sampleRate)
	}

	result := core.FFT(signal)

	// Check length
	if len(result) != numSamples {
		t.Errorf("Expected FFT output length %d, got %d", numSamples, len(result))
	}

	// The peak should be near bin corresponding to 10 Hz
	expectedBin := int(frequency * float64(numSamples) / sampleRate)
	peakBin := 0
	maxMag := 0.0

	for i := 0; i < numSamples/2; i++ {
		mag := cmplx.Abs(result[i])
		if mag > maxMag {
			maxMag = mag
			peakBin = i
		}
	}

	// Allow some tolerance
	if math.Abs(float64(peakBin-expectedBin)) > 2 {
		t.Errorf("Expected peak near bin %d, got bin %d", expectedBin, peakBin)
	}
}

func TestFFT_DCSignal(t *testing.T) {
	// Test with DC signal (constant value)
	signal := make([]float64, 8)
	for i := range signal {
		signal[i] = 5.0
	}

	result := core.FFT(signal)

	// DC component should be in bin 0
	dcValue := cmplx.Abs(result[0])
	expectedDC := 5.0 * float64(len(signal))

	if math.Abs(dcValue-expectedDC) > 0.01 {
		t.Errorf("Expected DC component %.2f, got %.2f", expectedDC, dcValue)
	}

	// Other bins should be near zero
	for i := 1; i < len(result); i++ {
		mag := cmplx.Abs(result[i])
		if mag > 0.01 {
			t.Errorf("Expected near-zero magnitude at bin %d, got %.4f", i, mag)
		}
	}
}

func TestFFT_PowerOfTwo(t *testing.T) {
	// Test various power-of-two sizes
	sizes := []int{2, 4, 8, 16, 32, 64, 128, 256}

	for _, size := range sizes {
		signal := make([]float64, size)
		for i := range signal {
			signal[i] = float64(i)
		}

		result := core.FFT(signal)

		if len(result) != size {
			t.Errorf("Size %d: expected output length %d, got %d", size, size, len(result))
		}
	}
}

func TestFFT_Symmetry(t *testing.T) {
	// For real input, FFT should have conjugate symmetry
	signal := []float64{1, 2, 3, 4, 4, 3, 2, 1}
	result := core.FFT(signal)

	n := len(result)
	// Check if result[k] is conjugate of result[n-k]
	for k := 1; k < n/2; k++ {
		expected := cmplx.Conj(result[n-k])
		if cmplx.Abs(result[k]-expected) > 1e-10 {
			t.Errorf("Conjugate symmetry violated at bin %d", k)
		}
	}
}

func BenchmarkFFT_128(b *testing.B) {
	signal := make([]float64, 128)
	for i := range signal {
		signal[i] = math.Sin(2 * math.Pi * float64(i) / 128)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.FFT(signal)
	}
}

func BenchmarkFFT_1024(b *testing.B) {
	signal := make([]float64, 1024)
	for i := range signal {
		signal[i] = math.Sin(2 * math.Pi * float64(i) / 1024)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.FFT(signal)
	}
}
