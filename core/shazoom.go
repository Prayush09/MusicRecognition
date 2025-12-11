package core

import (
	"fmt"
	"shazoom/db"
	"shazoom/utils"
	"sort"
	"time"
)

type Match struct {
	SongId     uint32
	SongTitle  string
	SongArtist string
	YoutubeID  string
	Timestamp  uint32
	Score      float64
}

func FindMatches(audioSample []float64, audioDuration float64, sampleRate int) ([]Match, time.Duration, error) {
	startTime := time.Now()

	// 1. Generate Spectrogram
	spectrogram, err := Spectrogram(audioSample, sampleRate)
	if err != nil {
		return nil, time.Since(startTime), fmt.Errorf("failed to generate spectrogram for samples: %v", err)
	}

	// 2. Extract Peaks
	peaks := ExtractPeaks(spectrogram, audioDuration, sampleRate)

	// 3. Generate Fingerprints (returns map[uint32]models.Couple)
	// 'Fingerprint' produces uint32 hashes.
	sampleFingerprint := Fingerprint(peaks, utils.GenerateUniqueID())

	// --- CRITICAL CHANGE: Cast to int64 for Database Compatibility ---
	// We convert the map keys from uint32 to int64 to match the BIGINT column in Postgres.
	sampleFingerprintMap := make(map[int64]uint32)
	
	for address32, couple := range sampleFingerprint {
		address64 := int64(address32) // Safe cast for DB lookup
		sampleFingerprintMap[address64] = couple.AnchorTime
	}

	fmt.Printf("Generated %d fingerprints from the recorded sample.\n", len(sampleFingerprint))
	
	// Pass the map with int64 keys to the matching function
	matches, _, err := FindMatchesUsingFingerPrints(sampleFingerprintMap)
	if err != nil {
		return nil, time.Since(startTime), err
	}

	return matches, time.Since(startTime), nil
}

// FindMatchesUsingFingerPrints now accepts map[int64]uint32
func FindMatchesUsingFingerPrints(sample map[int64]uint32) ([]Match, time.Duration, error) {
	startTime := time.Now()
	logger := utils.GetLogger()

	// --- CRITICAL CHANGE: Slice is now []int64 ---
	addresses := make([]int64, 0, len(sample))
	for address := range sample {
		addresses = append(addresses, address)
	}

	// Renamed variable to 'dbClient' to avoid shadowing package name 'db'
	dbClient, err := db.NewDBClient()
	if err != nil {
		return nil, time.Since(startTime), err
	}
	defer dbClient.Close()

	// Call DB with []int64, receive map[int64][]models.Couple
	m, err := dbClient.GetCouples(addresses)
	if err != nil {
		return nil, time.Since(startTime), err
	}

	timestamps := map[uint32]uint32{}          // timestamps for the songID
	targetZones := map[uint32]map[uint32]int{} // Count of targetzones within a timestamp for a specific songID
	matches := map[uint32][][2]uint32{}        // Matches containing the sampleTime and dbTime

	// Iterate over the results. 'address' is now int64.
	for address, couples := range m {
		for _, couple := range couples {
			// Logic remains the same, but types align correctly now.
			// sample[address] uses the int64 key to lookup the sample time.
			matches[couple.SongId] = append(
				matches[couple.SongId],
				[2]uint32{sample[address], couple.AnchorTime},
			)

			// add|update the couple time if the current one is a smaller difference
			if existingTime, ok := timestamps[couple.SongId]; !ok || couple.AnchorTime < existingTime {
				timestamps[couple.SongId] = couple.AnchorTime
			}

			// add targetzone map for the current couple if not present.
			if _, ok := targetZones[couple.SongId]; !ok {
				targetZones[couple.SongId] = make(map[uint32]int)
			}

			targetZones[couple.SongId][couple.AnchorTime]++
		}
	}

	// matches = filterMatches(10, matches, targetZones)

	// scoring logic
	scores := analyzeRelativeTiming(matches)

	var selectedCandidates []Match

	for songId, points := range scores {
		song, songExists, err := dbClient.GetSongByID(songId)
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

	sort.Slice(selectedCandidates, func(i, j int) bool {
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

    // Tune this based on your time units (anchorTime units)
    const tolerance int32 = 3

    for songId, times := range matches {
        n := len(times)
        if n == 0 {
            continue
        }
        if n == 1 {
            scores[songId] = 1
            continue
        }

        // collect deltas: dbTime - sampleTime
        deltas := make([]int32, n)
        for i, timePair := range times {
            sampleTime := int32(timePair[0])
            dbTime := int32(timePair[1])
            deltas[i] = dbTime - sampleTime
        }

        sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })

        // find longest streak where neighbouring deltas differ <= tolerance
        maxStreak := 1
        streak := 1
        for i := 1; i < n; i++ {
            if deltas[i]-deltas[i-1] <= tolerance {
                streak++
            } else {
                if streak > maxStreak {
                    maxStreak = streak
                }
                streak = 1
            }
        }
        if streak > maxStreak {
            maxStreak = streak
        }

        scores[songId] = float64(maxStreak)
    }

    return scores
}
