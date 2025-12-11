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

// Fingerprint generates fingerprints from a list of peaks and stores them in a map.
// The map key is now int64 to match the BIGINT column in the database.
// The value remains models.Couple.
func Fingerprint(peaks []Peak, songID uint32) map[int64]models.Couple {
	// --- CHANGE: Map key is now int64 ---
	fingerprints := map[int64]models.Couple{}

	for i, anchor := range peaks {
		// Search the "target zone" (next targetZoneSize peaks)
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			// Hash is generated as uint32 inside createAddress, then cast to int64
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

// createAddress generates a unique address (hash) for a pair of anchor and target points.
// It generates the 32-bit hash and casts it to int64 for database safety.
func createAddress(anchor, target Peak) int64 {
	// Note: Assuming Peak has float64 Freq and Time members (as inferred from your usage)
	anchorFreqBin := uint32(anchor.Freq / 10) // Scale down to fit in 9 bits
	targetFreqBin := uint32(target.Freq / 10)

	deltaMsRaw := uint32((target.Time - anchor.Time) * 1000)

	// Mask to fit within bit constraints
	anchorFreqBits := anchorFreqBin & ((1 << maxFreqBits) - 1) // 9 bits (0 to 511)
	targetFreqBits := targetFreqBin & ((1 << maxFreqBits) - 1) // 9 bits
	deltaBits := deltaMsRaw & ((1 << maxDeltaBits) - 1)        // 14 bits (max ~16 seconds)

	// Combine into 32-bit address (uint32)
	// Layout: [9 bits Anchor Freq] [9 bits Target Freq] [14 bits Delta Time]
	address32 := (anchorFreqBits << 23) | (targetFreqBits << 14) | deltaBits

	// --- CRITICAL CHANGE: Cast uint32 hash to int64 before use in DB layer ---
	return int64(address32)
}

// GenerateFingerprintsFromSamples processes pre-loaded audio samples to generate fingerprints.
// The return type is updated to map[int64]...
func GenerateFingerprintsFromSamples(samples []float64, sampleRate int, songID uint32) (map[int64]models.Couple, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("samples slice is empty")
	}

	// Calculate duration based on samples and sample rate
	duration := float64(len(samples)) / float64(sampleRate)

	// --- CHANGE: Initialize map with int64 key ---
	fingerprints := make(map[int64]models.Couple)

	// 1. Spectrogram and Peak Extraction
	spectro, err := Spectrogram(samples, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("error creating spectrogram: %w", err)
	}

	peaks := ExtractPeaks(spectro, duration, sampleRate)

	// 2. Fingerprinting
	// Fingerprint now returns map[int64]..., so ExtendMap works seamlessly
	utils.ExtendMap(fingerprints, Fingerprint(peaks, songID))

	return fingerprints, nil
}

// GenerateFingerprints is the original function, modified to handle file reading
// and channel separation. The return type is updated to map[int64]...
func GenerateFingerprints(songFilePath string, songID uint32) (map[int64]models.Couple, error) {
	// The previous implementation used ConvertToWAV with channels=1, effectively mixing or taking the left channel.
	wavFilePath, err := wav.ConvertToWAV(songFilePath, 2) // Convert to stereo to be safe
	if err != nil {
		return nil, fmt.Errorf("error converting input file to WAV: %w", err)
	}

	wavInfo, err := wav.ReadWavInfo(wavFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading WAV info: %w", err)
	}

	// --- CHANGE: Initialize map with int64 key ---
	fingerprints := make(map[int64]models.Couple)

	// Left Channel Processing
	spectro, err := Spectrogram(wavInfo.LeftChannelSamples, wavInfo.SampleRate)
	if err != nil {
		return nil, fmt.Errorf("error creating spectrogram: %w", err)
	}

	peaks := ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
	// Fingerprint now returns map[int64]..., so ExtendMap works seamlessly
	utils.ExtendMap(fingerprints, Fingerprint(peaks, songID))

	// Right Channel Processing (if stereo)
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