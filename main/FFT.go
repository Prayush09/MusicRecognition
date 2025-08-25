package main

import (
	"fmt"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
)

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

type Peak struct {
	Frequency float64
	Magnitude float64
	TimeChunk int
}

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


// what should be the return type of this function
func FFT(allAudioData []int16) {
	totalChunk := 256

	for i := 0; i < totalChunk; i++ {
		chunk := getChunk(allAudioData, i)
		fmt.Printf("Processing chunk %d, Size: %d\n", i, len(chunk))

		//convert each of []int64 to []float64 for FFT conversion.
		chunkFloat := convertToFloat64(chunk)

		//apply fft function for each chunk
		processed_Data := applyFFT(chunkFloat)

		//extract magnitutde:
		magnitudes := extractMagnitudes(processed_Data)
		fmt.Println("first five magnitues: ", magnitudes[:5])

		//find peaks
		peaks := findPeaks(magnitudes, 50000.0, 44100/1024, i)


		if i%50 == 0 && len(peaks) > 0 {
			fmt.Printf("Chunk %d peaks at frequencies: ", i)
			for _, peak := range peaks[:5] {
				fmt.Printf("Frequency: %f, Magnitude: %f\n", peak.Frequency, peak.Magnitude)
			}
		}
		/*
			TODO:
					1. Apply FFT
						* Converting Time Domain data into frequency domain data
					2. Extract magnitude
					3. find peaks
		*/
	}
}
