package core

import (
	"fmt"
	"shazoom/db"	
	"shazoom/utils"
	"sort"
	"time"
)

type Match struct{
	SongId uint32
	SongTitle string
	SongArtist string
	YoutubeID string
	Timestamp uint32
	Score float64
}

func FindMatches(audioSample []float64, audioDuration float64, sampleRate int) ([]Match, time.Duration, error) {
	startTime := time.Now();

	//going through the whole pipeline to generate fingerprints for sample
	spectrogram, err := Spectrogram(audioSample, sampleRate)
	if err != nil {
		return nil, time.Since(startTime), fmt.Errorf("failed to generate spectrogram for samples: %v", err)
	}

	peaks := ExtractPeaks(spectrogram, audioDuration, sampleRate)
	sampleFingerprint := Fingerprint(peaks, utils.GenerateUniqueID())

	sampleFingerprintMap := make(map[uint32]uint32)
	for address, couple := range sampleFingerprint {
		sampleFingerprintMap[address] = couple.AnchorTime
	}

	matches, _, _ := FindMatchesUsingFingerPrints(sampleFingerprintMap)

	return matches
}

//function used to search Database
func FindMatchesUsingFingerPrints(sample map[uint32]uint32) ([]Match, time.Duration, error){
	startTime := time.Now()
	logger := utils.GetLogger()

	addresses := make([]uint32, 0, len(sample))
	for address := range sample {
		addresses = append(addresses, address)
	}

	db, err := db.NewDBClient()
	if err != nil {
		return nil, time.Since(startTime), err
	}

	defer db.Close()

	m, err := db.GetCouples(addresses)
	if err != nil {
		return nil, time.Since(startTime), err
	}

	
	//TODO: Figure out which type of data structures are we going to use for optimal matching while staying in targetzones and within reasonable timestamps
	//TODO: timestamps :=
	//TODO: targetZones := 
	matches := map[uint32][][2]uint32{}
}	


