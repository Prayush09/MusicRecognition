package main

import (
	"fmt"
	"github.com/mjibson/go-dsp/fft"
)

func convertToFloat64(data []int16) []float64 {
	dataFloat64 := []float64{}
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

func FFT(allAudioData []int16) {
	totalChunk := 256

	for i := 0; i < totalChunk; i++ {
		chunk := getChunk(allAudioData, i)
		fmt.Printf("Processing chunk %d, Size: %d\n", i, len(chunk))

		//convert each of []int64 to []float64 for FFT conversion.
		chunkFloat := convertToFloat64(chunk)

		//apply fft function for each chunk	
		
		/*
			TODO:
					1. Apply FFT
						* Converting Time Domain data into frequency domain data

					2. Extract magnitude
					3. find peaks
		*/
	}
}
