package core

import (
	"fmt"
	"shazoom/fileformat"
	"shazoom/models"
	"shazoom/utils"
)

// 32 bits availabe => 9 bits for anchor, 9 bits for target and 14 bits for time Delta. 5 such addresses of target peaks per anchor peak.
const (
	maxFreqBits = 9
	maxDeltaBits = 14
	targetZoneSize = 5
)

//functions to implement - Fingerprint (Takes peaks and songId, generate fingerprints(address[Hash] and couple[anchorTime and Id]) and returns a map that links the couple with the address)
//creating Address - Implementing a function to create address (unique hash)
// Core Logic Function GenerateFingerprints (Song => Wav => spectrogram => Fingerprints)
func Fingerprint(peaks []Peak, songId uint32) map[uint32]models.Couple {	
	fingerprints := map[uint32]models.Couple{}

	//go through the peaks
	for i, anchor := range peaks{  
		for j := i + 1; j < len(peaks) && j <= i + targetZoneSize; j++{ //making sure the target peak is not too far away from the anchor peak
			target := peaks[j]

			address := createAddress(anchor, target)
			anchorTimeMs := uint32(anchor.Time * 1000) 

			fingerprints[address] = models.Couple{
				AnchorTime: anchorTimeMs,
				SongId: songId,
			}
		}
	}

	return fingerprints
}

//Our address will contain three core things (anchor frequency bits, target frequency bits and the time delta in between these bits which will allow for a quick matching)
func createAddress(anchor, target Peak) uint32 {
	//Make the frequency digestable
	anchorFreqBin := uint32(anchor.Freq / 10) 
	targetFreqBin := uint32(target.Freq / 10)
	deltaMsRaw := uint32((target.Time - anchor.Time) * 1000)

	//masking bins to range
	anchorFreqBits := anchorFreqBin & ((1 << maxFreqBits) - 1)
	targetFreqBits := targetFreqBin & ((1 << maxFreqBits) - 1)
	deltaTimeBits := deltaMsRaw & ((1 << maxDeltaBits) - 1)

	//creating a unique address from anchor, target and delta bits [MSB - AnchorBits (23-31) | TargetBits (22-14) | Delta (13-0) (LSB)]
	address := (anchorFreqBits << 23) | (targetFreqBits << 14) | deltaTimeBits

	return address
}

//core function to implement the whole pipeline
func GenerateFingerprints(songPath string, songId uint32) (map[uint32]models.Couple, error){
	//TODO: Need to a way to figure out if the incoming audio is stereo (channel == 2) or mono (channel == 1)
	wavFilePath, err := fileformat.ConvertToWAV(songPath, 1)
	if err != nil {
		return nil, fmt.Errorf("error converting input file to WAV, %v", err)
	}

	wavInfo, err := fileformat.ReadWavInfo(wavFilePath)
	if err != nil {
		return nil, fmt.Errorf("error fetching WAV infor, %v", err)
	}

	//make fingerprint map
	fingerprint := make(map[uint32]models.Couple)

	spectro, err := Spectrogram(wavInfo.LeftChannelSamples, wavInfo.SampleRate)
	if err != nil {
		return nil, fmt.Errorf("error creating spectrogram: %v", err)
	}

	peaks := ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
	utils.ExtendMap(fingerprint, Fingerprint(peaks, songId)) //transfer the function returned map data into our local map 

	if wavInfo.Channels == 2 {
		spectro, err = Spectrogram(wavInfo.RightChannelSamples, wavInfo.SampleRate)
		if err != nil {
			return nil, fmt.Errorf("error creating spectrogram for right channel: %v", err)
		}

		peaks = ExtractPeaks(spectro, wavInfo.Duration, wavInfo.SampleRate)
		utils.ExtendMap(fingerprint, Fingerprint(peaks, songId))
	}

	return fingerprint, nil
}

