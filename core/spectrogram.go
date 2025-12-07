package core

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

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
)

const (
    downSampleRatio = 4
    freqBinSize     = 1024           
    maxFreq         = 5000.0 
    hopSize         = freqBinSize / 2 
    windowType      = "hanning"      // another window type can be hamming 
)


func Spectrogram(sample []float64, sampleRate int) ([][]float64, error) {
    filteredSample := LowPassFilter(maxFreq, float64(sampleRate), sample)

    downsampledSample, err := Downsample(filteredSample, sampleRate, sampleRate/downSampleRatio)
    if err != nil {
        return nil, fmt.Errorf("couldn't downsample audio sample: %v", err)
    }

    window := make([]float64, freqBinSize)
    for i := range window {
        theta := 2 * math.Pi * float64(i) / float64(freqBinSize-1)
        switch windowType {
        case "hamming":
            window[i] = 0.54 - 0.46*math.Cos(theta)
        default: // Hanning window
            window[i] = 0.5 - 0.5*math.Cos(theta)
        }
    }

    spectrogram := make([][]float64, 0)

    for start := 0; start+freqBinSize <= len(downsampledSample); start += hopSize {
        end := start + freqBinSize
        frame := make([]float64, freqBinSize)
        copy(frame, downsampledSample[start:end])
        for j := range window {
            frame[j] *= window[j]
        }        
        fftResult := FFT(frame) 
        magnitude := make([]float64, len(fftResult)/2) // Only take the first half (Nyquist limit - non-redundant frequency information is contained only in the first half of the FFT output,)
        for j := range magnitude {
            magnitude[j] = cmplx.Abs(fftResult[j])
        }

        spectrogram = append(spectrogram, magnitude)
    }

    return spectrogram, nil
}

func LowPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
    rc := 1.0 / (2.0 * math.Pi * cutoffFrequency)
    dt := 1.0 / sampleRate                     
    alpha := dt / (rc + dt)                    

    filteredSignal := make([]float64, len(input))

    var prevOutput float64 = 0
    for i, x := range input {
        if i == 0 {
            filteredSignal[i] = x * alpha
        } else {
            // y[i] = alpha * x[i] + (1 - alpha) * y[i-1]
            filteredSignal[i] = alpha*x + (1-alpha)*prevOutput
        }
        prevOutput = filteredSignal[i]
    }

    return filteredSignal
}

func Downsample(input []float64, originalSampleRate, targetSampleRate int) ([]float64, error) {
    if targetSampleRate <= 0 || originalSampleRate <= 0 {
        return nil, errors.New("sample rates must be positive")
    }

    if targetSampleRate > originalSampleRate {
        return nil, errors.New("target sample rate must be less than or equal to original sample rate")
    }

    ratio := originalSampleRate / targetSampleRate
    if ratio <= 0 {
        return nil, errors.New("invalid ratio calculated from sample rates")
    }

    var resampled []float64
    for i := 0; i < len(input); i += ratio {
        end := i + ratio
        if end > len(input) {
            end = len(input)
        }

        sum := 0.0
        for j := i; j < end; j++ {
            sum += input[j]
        }

        avg := sum / float64(end-i)
        resampled = append(resampled, avg)
    }

    return resampled, nil
}

// Peak - a point of interest in spectrogram
type Peak struct {
    Time float64 // Time in seconds
    Freq float64 // Frequency in Hz
}

//create an array of collections of peaks
func ExtractPeaks(spectrogram [][]float64, audioDuration float64, sampleRate int) []Peak {
    if len(spectrogram) < 1 {
        return []Peak{}
    }

    type maxies struct {
        maxMag  float64
        freqIdx int
    }

    bands := []struct{ min, max int }{
        {0, 10}, {10, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 512},
    }

    var peaks []Peak
    frameDuration := audioDuration / float64(len(spectrogram))
    effectiveSampleRate := float64(sampleRate) / float64(downSampleRatio)
    freqResolution := effectiveSampleRate / float64(freqBinSize)

    for frameIdx, frame := range spectrogram {
        var maxMags []float64
        var freqIndices []int

        binBandMaxies := []maxies{}
        for _, band := range bands {
            var maxx maxies
            var maxMag float64 = -1.0 
            for idx, mag := range frame[band.min:band.max] {
                if mag > maxMag {
                    maxMag = mag
                    freqIdx := band.min + idx
                    maxx = maxies{mag, freqIdx}
                }
            }
            if maxx.maxMag > 0 {
                binBandMaxies = append(binBandMaxies, maxx)
            }
        }

        for _, value := range binBandMaxies {
            maxMags = append(maxMags, value.maxMag)
            freqIndices = append(freqIndices, value.freqIdx)
        }
        var maxMagsSum float64
        for _, max := range maxMags {
            maxMagsSum += max
        }
        if len(maxMags) == 0 {
            continue
        }
        avg := maxMagsSum / float64(len(maxMags))
        for i, value := range maxMags {
            //selected peak > avg(peaks) (like it is for getting a job these day lol)
            if value > avg {
                peakTime := float64(frameIdx) * frameDuration
                peakFreq := float64(freqIndices[i]) * freqResolution

                peaks = append(peaks, Peak{Time: peakTime, Freq: peakFreq})
            }
        }
    }

    return peaks
}