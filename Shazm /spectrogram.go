package core

import (
	"fmt"
	"math"
)

const (
	downSampleRatio = 4
	freqBinSize = 1024
	maxFreq     = 5000.0 //5 kHz
	hopSize     = 1024 / 32
)

/*
This code implements audio fingerprinting through spectral analysis.
 It processes audio signals by filtering, downsampling, and extracting frequency peaks
 to create unique identifiers for audio matching.

 The pipeline performs the following steps:
 1. Low-pass filtering to remove frequencies above 5kHz (anti-aliasing)
 2. Downsampling by 4× to reduce computational load while preserving relevant frequencies
 3. Short-Time Fourier Transform (STFT) with Hamming windowing to convert audio into a spectrogram
 4. Peak extraction across frequency bands to identify the most significant spectral features
*/
func Spectrogram(sample []float64, sampleRate int) ([][]complex128, error) {
	filteredSample := LowFilterPass(sample, maxFreq, float64(sampleRate))
	
	downsampledSample := DownSampling(filteredSample, sampleRate, sampleRate/downSampleRatio)

}

func LowFilterPass(sample []float64, cutoffFrequency, sampleRate float64) []float64 {
	rc := 1.0 / (2.0 * math.Pi * cutoffFrequency) //Resistor Capacitor Time constant - determines how fast the capacitor charges/discharges, which determines the cutoff frequency.
	dt := 1.0 / sampleRate                        //time between sample[0] and sample[1]
	alpha := dt / (rc + dt)                       //smoothning factor
	/*
		alpha will be a number between 0 and 1 that determines how much filtering happens:
			If alpha ≈ 1: almost no filtering (signal passes through)
			If alpha ≈ 0: heavy filtering (lots of smoothing)
	*/

	filteredSignal := make([]float64, len(sample))

	var prevOutput float64 = 0
	for i, x := range sample {
		if i == 0 {
			//first sample - just apply the smoothning factor
			filteredSignal[i] = alpha * x
		} else {
			//Update change based on current and previous sample frequencies.
			filteredSignal[i] = alpha*x + (1-alpha)*prevOutput
		}
		prevOutput = filteredSignal[i]
	}

	return filteredSignal
}

func DownSample(sample []float64, originalSampleRate, targetSampleRate float64) {
	//TODO: complete it 
}