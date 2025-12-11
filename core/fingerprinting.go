package core

import (
    "fmt"
    wav "shazoom/fileformat"
    "shazoom/models"
    "shazoom/utils"
)

const (
    maxFreqBits    = 9
    maxDeltaBits   = 14
    targetZoneSize = 5
)

func Fingerprint(peaks []Peak, songID uint32) map[int64]models.Couple {
    fingerprints := map[int64]models.Couple{}
    for i, anchor := range peaks {
        for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
            target := peaks[j]

            address64 := createAddress(anchor, target) 
            anchorTimeMs := uint32(anchor.Time * 1000)

            fingerprints[address64] = models.Couple{
                AnchorTime: anchorTimeMs,
                SongId:     songID,
            }
        }
    }

    return fingerprints
}

func createAddress(anchor, target Peak) int64 {
    anchorFreqBin := uint32(anchor.Freq / 10) 
    targetFreqBin := uint32(target.Freq / 10)

    deltaMsRaw := uint32((target.Time - anchor.Time) * 1000)

    anchorFreqBits := anchorFreqBin & ((1 << maxFreqBits) - 1) 
    targetFreqBits := targetFreqBin & ((1 << maxFreqBits) - 1) 
    deltaBits := deltaMsRaw & ((1 << maxDeltaBits) - 1)        

    address32 := (anchorFreqBits << 23) | (targetFreqBits << 14) | deltaBits

    return int64(address32)
}

func GenerateFingerprintsFromSamples(samples []float64, sampleRate int, songID uint32) (map[int64]models.Couple, error) {
    if len(samples) == 0 {
        return nil, fmt.Errorf("samples slice is empty")
    }

    duration := float64(len(samples)) / float64(sampleRate)

    fingerprints := make(map[int64]models.Couple)

    spectro, err := Spectrogram(samples, sampleRate)
    if err != nil {
        return nil, fmt.Errorf("error creating spectrogram: %w", err)
    }

    peaks := ExtractPeaks(spectro, duration, sampleRate)

    utils.ExtendMap(fingerprints, Fingerprint(peaks, songID))

    return fingerprints, nil
}

func GenerateFingerprints(songFilePath string, songID uint32) (map[int64]models.Couple, error) {
    wavFilePath, err := wav.ConvertToWAV(songFilePath, 2) 
    if err != nil {
        return nil, fmt.Errorf("error converting input file to WAV: %w", err)
    }

    wavInfo, err := wav.ReadWavInfo(wavFilePath)
    if err != nil {
        return nil, fmt.Errorf("error reading WAV info: %w", err)
    }

    fingerprints := make(map[int64]models.Couple)

    spectro, err := Spectrogram(wavInfo.LeftChannelSamples, wavInfo.SampleRate)
    if err != nil {
        return nil, fmt.Errorf("error creating spectrogram: %w", err)
    }

    peaks := ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
    utils.ExtendMap(fingerprints, Fingerprint(peaks, songID))

    if wavInfo.Channels == 2 {
        spectro, err = Spectrogram(wavInfo.RightChannelSamples, wavInfo.SampleRate)
        if err != nil {
            return nil, fmt.Errorf("error creating spectrogram for right channel: %w", err)
        }

        peaks = ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
        utils.ExtendMap(fingerprints, Fingerprint(peaks, songID))
    }

    return fingerprints, nil
}