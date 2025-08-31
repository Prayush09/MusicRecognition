package main

import (
	"fmt"
)

func main(){
	//recording 
	fmt.Println("Sound Recording Starts")
	audioData := Recording()
	fmt.Println("Sound Recording Ended")

	fmt.Println("FFT Processing Begins")
	//FFT Processing
	classifiedPeaks := FFT(audioData)
	fmt.Println("FFT Processing Ends")

	//constellation map 
	fmt.Println("Create Constellation Map")
	allPeaks := FlattenPeaks(classifiedPeaks)
	fmt.Println("Total Peaks found: %d", allPeaks)

	constellationMap := CreateConstellationMap(allPeaks)
	fmt.Println("Constellation map created")

	//hashing
	hashes := GenerateHashes(constellationMap)
	fmt.Println("Hashed Created: %d", hashes)

	//TODO: Implement FlattenPeaks, CreateConstellationMap, GenerateHashes
}