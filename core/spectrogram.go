package core

import (
	"fmt"
	"math"
	"math/cmplx"
)

const (
	downSampleRatio = 4
	freqBinSize = 1024
	maxFreq     = 5000.0 //5 kHz
	hopSize     = 32
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
	//cleaning and down sizing the sample to make it ready for fingerprinting.
	filteredSample := LowFilterPass(sample, maxFreq, float64(sampleRate))
	downsampledSample, err := DownSample(filteredSample, sampleRate, sampleRate/downSampleRatio)
	if err != nil {
		return nil, fmt.Errorf("error occurred while down sampling: %v", err)
	}

	numOfWindows := len(downsampledSample) / (freqBinSize - hopSize)
	spectrogram := make([][]complex128, numOfWindows)

	//hamming window formulae = 0.54 - 0.46 * cos(2π * i / (N - 1)) || Used to taper the signal at both ends of the window.
	window := make([]float64, freqBinSize)
	for i := range window {
		window[i] = 0.54 - 0.46 * math.Cos(2*math.Pi * float64(i) / (float64(freqBinSize) - 1))
	}

	//performing STFT
	for i := 0; i < numOfWindows; i++ {
		start := i * hopSize
		end := start + freqBinSize
		if end > len(downsampledSample) {
			end = len(downsampledSample)
		}

		bin := make([]float64, freqBinSize)
		copy(bin, downsampledSample[start : end])

		//apply hamming window
		for j := range window {
			bin[j] *= window[j] 
		}

		//apply FFT(time -> frequency) to this bin then add to spectrogram.
		spectrogram[i] = FFT(bin)
	}

	return spectrogram, nil
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

//normalizing (taking avg) the values in sample input provided in ratio's of say 4 if org sample is 44k and target is 11k. Sending back this smaller data
func DownSample(sample []float64, originalSampleRate, targetSampleRate int) ([]float64, error) {
	if targetSampleRate <= 0 || originalSampleRate <= 0 {
		return nil,fmt.Errorf("provided sample rates must be positive");
	}

	if targetSampleRate > originalSampleRate {
		return nil, fmt.Errorf("target sample rate must be <= original sample rate");
	}

	ratio := originalSampleRate / targetSampleRate
	
	var downSampled []float64
	for i := 0; i < len(sample); i += ratio {
		end := i + ratio;
		if end > len(sample){
			end = len(sample)
		}

		sum := 0.0
		for j := 0; j < end; j++{
			sum += sample[j];
		}

		avg := sum / float64(end-i) 
		downSampled = append(downSampled, avg)
	}

	return downSampled, nil
}



type Peak struct{
	Time float64 //At what point in time did the peak happened
	Freq complex128 //frequency at that time
}

func ExtractPeaks(spectrogram [][]complex128, audioDuration float64) []Peak {
	if len(spectrogram) < 1 {
		return []Peak{}
	}

	type maxFreq struct {
		mag float64
		freq complex128
		freqIndex int
	}

	bands := []struct{min, max int}{{0,10}, {10,20}, {20,40}, {40,80}, {80,160}, {160,512}}

	var peaks []Peak
	binDuration := audioDuration / float64(len(spectrogram))

	for binIdx, bin := range spectrogram {
		var maxMags []float64
		var maxFreqs []complex128
		var freqIndices []float64

		binBandMax := []maxFreq{}
		for _, band := range bands {
			var max maxFreq
			var maxMag float64
			for idx, freq := range bin[band.min : band.max] {
				magnitude := cmplx.Abs(freq)
				if magnitude > maxMag {
					maxMag = magnitude
					freqIdx := band.min + idx
					max = maxFreq{magnitude, freq, freqIdx}
				}
			}
			binBandMax = append(binBandMax, max)
		}

		for _, value := range binBandMax {
			maxMags = append(maxMags, value.mag)
			maxFreqs = append(maxFreqs, value.freq)
			freqIndices = append(freqIndices, float64(value.freqIndex))
		}

		//calculate average magnitude
		var magSum float64
		for _, value := range maxMags{
			magSum += value
		}

		avg := magSum/float64(len(maxFreqs))

		//Add peaks that exceed the average of max magnitudes
		for i, value := range maxMags {
			if value > avg {
				peakTimeInBin := freqIndices[i] * binDuration / float64(len(bin))
				
				//absolute time of peak
				peakTime := float64(binIdx) * binDuration + peakTimeInBin

				peaks = append(peaks, Peak{Time: peakTime, Freq: maxFreqs[i]})
			}
		}
	}

	return peaks
}