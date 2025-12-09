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

	return matches, time.Since(startTime), nil
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

	
	timestamps := map[uint32]uint32{} //timestamps for the songID
	targetZones := map[uint32]map[uint32]int{} // Count of targetzones within a timestamp for a specific songID
	matches := map[uint32][][2]uint32{} // Matches containing the sampleTime and dbTime

	for address, couples := range m {
		for _, couple := range couples {
			matches[couple.SongId] = append(
				matches[couple.SongId], 
				[2]uint32{sample[address], couple.AnchorTime},
			)

			//add|update the couple time if the current one is a smaller difference 
			if existingTime, ok := timestamps[couple.SongId]; !ok || couple.AnchorTime < existingTime {
				timestamps[couple.SongId] = couple.AnchorTime
			}

			//add targetzone map for the current couple if not present.
			if _, ok := targetZones[couple.SongId]; !ok {
				targetZones[couple.SongId] = make(map[uint32]int)
			}

			targetZones[couple.SongId][couple.AnchorTime]++
		}
	}

	//matches = filterMatches(10, matches, targetZones)
	
	//scoring logic
	scores := analyzeRelativeTiming(matches)

	var selectedCandidates []Match
	
	for songId, points := range scores {
		song, songExists, err := db.GetSongByID(songId)
		if !songExists {
			logger.Info(fmt.Sprintf("song provided (%v) doesn't exist in our DB :(", songId))
			continue
		}

		if err != nil {
			logger.Info(fmt.Sprintf("failed to fetch the song by ID (%v): %v", songId, err))
		}


		match := Match{songId, song.Title, song.Artist, song.YouTubeID, timestamps[songId], points}
		selectedCandidates = append(selectedCandidates, match)
	}

	sort.Slice(selectedCandidates, func(i, j int) bool{
		return selectedCandidates[i].Score > selectedCandidates[j].Score
	})

	return selectedCandidates, time.Since(startTime), nil
}	

/*
	for each song in the database, we increase the count of the score 
	if the delta between the sampleTime and the songTime is consistent for all/most of the anchor peaks.
	And then the song with the most consistent time delta will gain the highest score.
*/
func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]float64 {
		scores := make(map[uint32]float64)

		for songId, times := range matches {
			differenceCounts := make(map[int32]int)

			for _, timePair := range times {
				sampleTime := int32(timePair[0]) // sample provided time 
				dbTime := int32(timePair[1]) //matched pair in db time
				difference := dbTime - sampleTime 

				//a little variation allowed (100ms)
				differenceVariance := difference / 100
				differenceCounts[differenceVariance]++;
			}

			//find max count from all the different 'differenceVariance'.
			maxCount := 0
			for _, count := range differenceCounts {
				if count > maxCount {
					maxCount = count
				}
			}
		
			scores[songId] = float64(maxCount)
		}

		return scores
}

//only matches that passes the minimum threshold target zones are considered for scores
func filterMatches(threshold int, matches map[uint32][][2]uint32, targetZones map[uint32]map[uint32]int) map[uint32][][2]uint32 {
	for songId, anchorTimes := range targetZones {
		for anchorTime, count := range anchorTimes {
			if count < targetZoneSize {
				delete(targetZones[songId], anchorTime)
			}
		}
	}

	filteredMatches := map[uint32][][2]uint32{}
	for songId, zones := range targetZones {
		if len(zones) >= threshold {
			filteredMatches[songId] = matches[songId]
		}
	}

	return filteredMatches
} 