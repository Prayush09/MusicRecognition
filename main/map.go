package main

import "sort"

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

}


func GenerateHashes(){
	
}