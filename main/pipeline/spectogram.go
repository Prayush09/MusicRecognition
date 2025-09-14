package main

import (
	"errors"
	"fmt"
	"math"
)

const (
	dspRatio    = 4
	freqBinSize = 2048 // Increased from 1024 to match your current chunk size
	maxFreq     = 8000.0 // Increased to match your current range
	hopSize     = freqBinSize / 2 // 50% overlap like your current implementation
)

//spectrogram => 

func Spectrogram(sample []float64, sampleRate int) ([][]complex128, error) {
	// Apply low-pass filter
	filteredSample := LowPassFilter(maxFreq, float64(sampleRate), sample)

	// Downsample to reduce computation
	downsampledSample, err := Downsample(filteredSample, sampleRate, sampleRate/dspRatio)
	if err != nil {
		return nil, fmt.Errorf("couldn't downsample audio sample: %v", err)
	}

	numOfWindows := (len(downsampledSample) - freqBinSize) / hopSize + 1
	spectrogram := make([][]complex128, numOfWindows)

	// Create Hann window (your current approach)
	window := make([]float64, freqBinSize)
	for i := range window {
		window[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/(float64(freqBinSize)-1)))
	}

	fmt.Printf("Spectrogram Parameters: Windows=%d, BinSize=%d, HopSize=%d\n", 
		numOfWindows, freqBinSize, hopSize)

	// Perform STFT with your optimizations
	for i := 0; i < numOfWindows; i++ {
		start := i * hopSize
		end := start + freqBinSize
		if end > len(downsampledSample) {
			end = len(downsampledSample)
		}

		bin := make([]float64, freqBinSize)
		copy(bin, downsampledSample[start:end])

		// Apply Hann window
		for j := range bin {
			if j < len(window) {
				bin[j] *= window[j]
			}
		}

		spectrogram[i] = FFT(bin)

		if i%50 == 0 {
			fmt.Printf("Processed %d spectrogram windows\n", i)
		}
	}

	return spectrogram, nil
}

// LowPassFilter with improved implementation
func LowPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
	rc := 1.0 / (2 * math.Pi * cutoffFrequency)
	dt := 1.0 / sampleRate
	alpha := dt / (rc + dt)

	filteredSignal := make([]float64, len(input))
	var prevOutput float64 = 0

	for i, x := range input {
		if i == 0 {
			filteredSignal[i] = x * alpha
		} else {
			filteredSignal[i] = alpha*x + (1-alpha)*prevOutput
		}
		prevOutput = filteredSignal[i]
	}
	return filteredSignal
}

// Enhanced downsampling with anti-aliasing
func Downsample(input []float64, originalSampleRate, targetSampleRate int) ([]float64, error) {
	if targetSampleRate <= 0 || originalSampleRate <= 0 {
		return nil, errors.New("sample rates must be positive")
	}
	if targetSampleRate > originalSampleRate {
		return nil, errors.New("target sample rate must be less than or equal to original sample rate")
	}

	ratio := float64(originalSampleRate) / float64(targetSampleRate)
	if ratio <= 0 {
		return nil, errors.New("invalid ratio calculated from sample rates")
	}

	outputLength := int(float64(len(input)) / ratio)
	resampled := make([]float64, outputLength)

	for i := 0; i < outputLength; i++ {
		sourceIndex := float64(i) * ratio
		lowerIndex := int(sourceIndex)
		upperIndex := lowerIndex + 1

		if upperIndex >= len(input) {
			resampled[i] = input[lowerIndex]
		} else {
			// Linear interpolation for better quality
			fraction := sourceIndex - float64(lowerIndex)
			resampled[i] = input[lowerIndex]*(1-fraction) + input[upperIndex]*fraction
		}
	}

	return resampled, nil
}
