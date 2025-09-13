package main

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"sort"
)

type Peak struct {
	Time      float64 `json:"Time"`
	Frequency float64 `json:"Frequency"`
	Magnitude float64 `json:"Magnitude"`
	TimeChunk int     `json:"TimeChunk"`
}

type ConstellationPair struct {
	Anchor Peak   `json:"Anchor"`
	Target Peak   `json:"Target"`
	Hash   uint64 `json:"Hash"`
}

type Fingerprint struct {
	Hash      uint64  `json:"hash"`
	TimeStamp float64 `json:"time_stamp"`
}

func FlattenPeaks(data [][]Peak, chunkSize, sampleRate int) []Peak {
	var allPeaks []Peak
	chunkDuration := float64(chunkSize) / float64(sampleRate)
	
	for chunkIndex, chunkPeaks := range data {
		timestamp := chunkDuration * float64(chunkIndex)
		
		// Re-organizing data from [{Chunk, Frequency, Magnitude}] -> [{Time, Frequency, Magnitude}]
		for _, peak := range chunkPeaks {
			allPeaks = append(allPeaks, Peak{
				Time:      timestamp,
				Frequency: peak.Frequency,
				Magnitude: peak.Magnitude,
				TimeChunk: chunkIndex, // Preserve original chunk info
			})
		}
	}
	
	// Sorting in chronological order
	sort.Slice(allPeaks, func(i, j int) bool {
		return allPeaks[i].Time < allPeaks[j].Time
	})
	
	return allPeaks
}

func CreateConstellationMap(allPeaks []Peak) []ConstellationPair {
	var ConstellationMap []ConstellationPair
	
	// Constellation Parameters
	const (
		minSeparation      = 0.05  // Fixed typo: "Seperation" -> "Separation"
		maxSeparation      = 2.0
		maxTargetPerAnchor = 5
		minFreqDelta       = 10.0  // Made this configurable
	)
	
	for i, anchorPeak := range allPeaks {
		targetsFound := 0
		
		// Looking for target peaks that come after this anchor
		for j := i + 1; j < len(allPeaks) && targetsFound < maxTargetPerAnchor; j++ {
			targetPeak := allPeaks[j]
			timeDelta := targetPeak.Time - anchorPeak.Time
			
			// Skip as current target is too close to anchor to form a meaningful pair
			if timeDelta < minSeparation {
				continue
			}
			
			// Break as the current target is too far away from anchor
			if timeDelta > maxSeparation {
				break
			}
			
			// Similar frequency pairing should be avoided as we need distinct frequency for identification
			freqDelta := math.Abs(targetPeak.Frequency - anchorPeak.Frequency)
			if freqDelta < minFreqDelta {
				continue
			}
			
			// Generate hash for this pair
			hash := generatePairHash(anchorPeak, targetPeak)
			
			// Create pair
			pair := ConstellationPair{
				Anchor: anchorPeak,
				Target: targetPeak,
				Hash:   hash, // Store hash in the pair as well
			}
			
			ConstellationMap = append(ConstellationMap, pair)
			targetsFound++
		}
	}
	
	return ConstellationMap
}

// Helper function to generate hash for a pair (used in CreateConstellationMap)
func generatePairHash(anchor, target Peak) uint64 {
	anchorFreq := uint64(anchor.Frequency)
	targetFreq := uint64(target.Frequency)
	timeDelta := uint64((target.Time - anchor.Time) * 1000) // convert to milliseconds
	
	// Convert uint64 to 8*3 bytes (3 components) for SHA256 input
	data := make([]byte, 24)
	binary.BigEndian.PutUint64(data[0:8], anchorFreq)
	binary.BigEndian.PutUint64(data[8:16], targetFreq)
	binary.BigEndian.PutUint64(data[16:24], timeDelta)
	
	// Generate SHA256 hash: 32 bytes of hashData
	hashBytes := sha256.Sum256(data)
	
	// Converting first 8 bytes to uint64 (uint64 can only hold up to 8 bytes of data)
	return binary.BigEndian.Uint64(hashBytes[:8])
}

func GenerateHashes(ConstellationMap []ConstellationPair) []Fingerprint {
	var fingerprints []Fingerprint
	
	for _, pair := range ConstellationMap {
		// Generate hash for this pair (or use pre-computed hash from pair)
		var hash uint64
		if pair.Hash != 0 {
			// Use pre-computed hash from CreateConstellationMap
			hash = pair.Hash
		} else {
			// Generate hash if not already computed
			hash = generatePairHash(pair.Anchor, pair.Target)
		}
		
		// Create fingerprint using this hash and anchor time
		fingerprint := Fingerprint{
			Hash:      hash,
			TimeStamp: pair.Anchor.Time,
		}
		
		fingerprints = append(fingerprints, fingerprint)
	}
	
	return fingerprints
}

// Additional helper functions for debugging and analysis

// GetPeakStats returns statistics about the peaks
func GetPeakStats(peaks []Peak) (minFreq, maxFreq, avgMagnitude float64, totalPeaks int) {
	if len(peaks) == 0 {
		return 0, 0, 0, 0
	}
	
	minFreq = peaks[0].Frequency
	maxFreq = peaks[0].Frequency
	totalMagnitude := 0.0
	
	for _, peak := range peaks {
		if peak.Frequency < minFreq {
			minFreq = peak.Frequency
		}
		if peak.Frequency > maxFreq {
			maxFreq = peak.Frequency
		}
		totalMagnitude += peak.Magnitude
	}
	
	avgMagnitude = totalMagnitude / float64(len(peaks))
	totalPeaks = len(peaks)
	
	return minFreq, maxFreq, avgMagnitude, totalPeaks
}

// FilterPeaksByFrequency filters peaks within a frequency range
func FilterPeaksByFrequency(peaks []Peak, minFreq, maxFreq float64) []Peak {
	var filtered []Peak
	
	for _, peak := range peaks {
		if peak.Frequency >= minFreq && peak.Frequency <= maxFreq {
			filtered = append(filtered, peak)
		}
	}
	
	return filtered
}

// GetConstellationStats returns statistics about constellation pairs
func GetConstellationStats(pairs []ConstellationPair) (avgTimeDelta, avgFreqSpread float64, totalPairs int) {
	if len(pairs) == 0 {
		return 0, 0, 0
	}
	
	totalTimeDelta := 0.0
	totalFreqSpread := 0.0
	
	for _, pair := range pairs {
		timeDelta := pair.Target.Time - pair.Anchor.Time
		freqSpread := math.Abs(pair.Target.Frequency - pair.Anchor.Frequency)
		
		totalTimeDelta += timeDelta
		totalFreqSpread += freqSpread
	}
	
	avgTimeDelta = totalTimeDelta / float64(len(pairs))
	avgFreqSpread = totalFreqSpread / float64(len(pairs))
	totalPairs = len(pairs)
	
	return avgTimeDelta, avgFreqSpread, totalPairs
}