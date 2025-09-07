package main

import (
	"fmt"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
)
//TODO: Fix the clustering of peaks. Also figure out why the peaks are stored at 96kHz (out of human hearing range;-;)
var RANGE = []int{40, 80, 120, 180, 300}

func convertToFloat64(data []int16) []float64 {
	dataFloat64 := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		dataFloat64[i] = float64(data[i])
	}
	return dataFloat64
}

func getChunk(data []int16, chunkNumber int) []int16 {
	chunkSize := len(data) / 256

	start := chunkSize * chunkNumber
	end := start + chunkSize

	return data[start:end]
}

func applyFFT(chunk []float64) []complex128 {
	return fft.FFTReal(chunk)
}

func extractMagnitudes(fftData []complex128) []float64 {
	magnitudes := make([]float64, len(fftData))
	for i := 0; i < len(fftData); i++ {
		magnitudes[i] = cmplx.Abs(fftData[i])
	}
	return magnitudes
}


// detect all local maxima
func findPeaks(magnitudes []float64, threshold float64, binWidth float64, chunkIndex int) []Peak {
	peaks := []Peak{}

	for i := 1; i < len(magnitudes)-1; i++ {
		if magnitudes[i] > magnitudes[i-1] &&
			magnitudes[i] > magnitudes[i+1] &&
			magnitudes[i] > threshold {

			peak := Peak{
				Frequency: float64(i) * binWidth,
				Magnitude: magnitudes[i],
				TimeChunk: chunkIndex,
			}
			peaks = append(peaks, peak)
		}
	}

	return peaks
}

// find which band a frequency belongs to
func findRange(freq int) int {
	i := 0
	for i < len(RANGE) && RANGE[i] < freq {
		i++
	}
	if i >= len(RANGE) {
		return RANGE[len(RANGE)-1] // clamp to last band if above range
	}
	return RANGE[i]
}

// keep only the strongest peak per band per chunk
func classifyPeaks(allPeaks [][]Peak) [][]Peak {
	classified := make([][]Peak, len(allPeaks))

	for i := 0; i < len(allPeaks); i++ {
		bestPerBand := make(map[int]Peak)

		for _, peak := range allPeaks[i] {
			band := findRange(int(peak.Frequency))

			best, exists := bestPerBand[band]
			if !exists || peak.Magnitude > best.Magnitude {
				bestPerBand[band] = peak
			}
		}

		for _, p := range bestPerBand {
			classified[i] = append(classified[i], p)
		}
	}

	return classified
}

// main FFT driver
func FFT(allAudioData []int16) [][]Peak {
	totalChunk := 256
	allPeaks := make([][]Peak, totalChunk)

	// Bin width = sampleRate / FFT_size
	sampleRate := 44100.0
	fftSize := 1024.0
	binWidth := sampleRate / fftSize

	for i := range totalChunk {
		chunk := getChunk(allAudioData, i)
		fmt.Printf("Processing chunk %d, Size: %d\n", i, len(chunk))

		chunkFloat := convertToFloat64(chunk)
		processedData := applyFFT(chunkFloat)
		magnitudes := extractMagnitudes(processedData)

		peaks := findPeaks(magnitudes, 50000.0, binWidth, i)
		allPeaks[i] = peaks
	}

	// classify peaks into bands
	classified := classifyPeaks(allPeaks)

	return classified
}
