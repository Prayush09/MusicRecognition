package main

import (
	"sort"
	"math"
	"crypto/sha256"
    "encoding/binary"
)


type Peak struct {
	Frequency float64
	Magnitude float64
	TimeChunk int
	Time float64
}

type ConstellationPair struct {
	Anchor Peak
	Target Peak
}

type Fingerprint struct {
	Hash uint64
	TimeStamp float64
}


func FlattenPeaks(data [][]Peak, chunkSize , sampleRate int) []Peak {
	var allPeaks []Peak
	chunkDuration := float64(chunkSize) / float64(sampleRate)

	for chunkIndex, chunkPeaks := range data {
		timestamp := chunkDuration * float64(chunkIndex)

		//re-organizing data from [{Chunk, Frequency, Magnitude}] -> [{Time, Frequency, Magnitude}]
		for _, peak := range chunkPeaks {
			allPeaks = append(allPeaks, Peak{
				Time: timestamp,
				Frequency: peak.Frequency,
				Magnitude: peak.Magnitude,
			})
		}
	}
	
	//sorting in chronological order
	sort.Slice(allPeaks, func(i, j int) bool {
		return allPeaks[i].Time < allPeaks[j].Time
	})

	return allPeaks
}


func CreateConstellationMap(allPeaks []Peak) []ConstellationPair {
	var ConstellationMap []ConstellationPair

	//Constellation Parameters
	const (
		minSeperation = 0.05
		maxSeperation = 2.0
		maxTargetPerAnchor = 5
	)

	for i, anchorPeak := range allPeaks {
		targetsFound := 0

		//Looking for target peaks that comes after this anchor
		for j := i+1; j < len(allPeaks) && targetsFound < maxTargetPerAnchor; j++ {
			targetPeak := allPeaks[j]
			timeDelta := targetPeak.Time - anchorPeak.Time

			//skip as current target is too close to anchor to form a meaningful pair
			if timeDelta < minSeperation {
				continue
			}

			//break as the current target is too far away from anchor and so no meaning in checking further
			if timeDelta > maxSeperation {
				break
			}

			//Similar frequency pairing should be avoided as we need distinct frequency for identification
			freqDelta := math.Abs(targetPeak.Frequency - anchorPeak.Frequency)
			if freqDelta < 10 {
				continue
			}

			//create pair
			pair := ConstellationPair{
				Anchor: anchorPeak,
				Target: targetPeak,
			}

			ConstellationMap = append(ConstellationMap, pair)
			targetsFound++
		}
	}

	return ConstellationMap
}	


func GenerateHashes(ConstellationMap []ConstellationPair) []Fingerprint {
	var fingerprints []Fingerprint

	for _, pair := range ConstellationMap {
		anchorFreq := uint64(pair.Anchor.Frequency)
		targetFreq := uint64(pair.Target.Frequency)
		timeDelta := uint64((pair.Target.Time - pair.Anchor.Time) * 1000) //convert to milliseconds
	

		//convert uint64 to 8*3 bytes (3 components) for SHA256 input 
		data := make([]byte, 24)
		binary.BigEndian.PutUint64(data[0:8], anchorFreq)
		binary.BigEndian.PutUint64(data[9:16], targetFreq)
		binary.BigEndian.PutUint64(data[17:24], timeDelta)

		// generate SHA256 hash : 32 bytes of hashData
		hashBytes := sha256.Sum256(data)

		//converting first 8 bytes to uint64 (uint64 can only hold till 8 bytes of data)
		hash := binary.BigEndian.Uint64(hashBytes[:8])

		//create fingerprint using this has and anchor time
		fingerprint := Fingerprint{
			Hash: hash,
			TimeStamp: pair.Anchor.Time,
		}

		fingerprints = append(fingerprints, fingerprint)
	}

	return fingerprints
}