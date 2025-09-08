package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"flag"
	"github.com/gordonklaus/portaudio"
)

func Recording() []int16 {
	err := portaudio.Initialize()
	if err != nil {
		log.Println("Port Audio failed to initialize!")
		os.Exit(1)
	}
	log.Println("Port Audio has initalized!")

	if len(os.Args) < 2 {
		fmt.Println("Required missing argument: Filename")
		fmt.Println("Usage: go run recording.go <Filename.wav>")
		return []int16{}
	}

	rate := flag.Float64("rate", 0, "Sample rate (leave 0 to use device default)")

	inputDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		log.Println("Failed to get default input device:", err)
		return []int16{}
	}

	sampleRate := inputDevice.DefaultSampleRate
	if *rate > 0 {
		sampleRate = *rate
	}
	fmt.Printf("Using device: %s (sample rate: %.0f Hz)\n", inputDevice.Name, sampleRate)

	// Configure stream parameters
	parameters := portaudio.HighLatencyParameters(inputDevice, nil)
	parameters.Input.Channels = 1
	parameters.SampleRate = sampleRate
	parameters.FramesPerBuffer = 1024


	buffer := make([]int16, 1024)
	stream, err := portaudio.OpenStream(parameters, buffer)
	if err != nil {
		log.Println("Stream Parameters have not been set")
		return []int16{}
	}

	err = stream.Start()
	if err != nil {
		log.Println("Stream has failed to start")
	}

	fmt.Println("Recording for 5 seconds.")
	var allAudioData []int16
	startTime := time.Now()
	for time.Since(startTime) < 5*time.Second {
		err := stream.Read()
		if err != nil {
			log.Println("Recording of audio has failed")
			break
		}
		allAudioData = append(allAudioData, buffer...)
	}

	stream.Stop()
	fmt.Println("Total samples:", len(allAudioData))
	fmt.Println("Starting FFT Processing")
	
	

	err = portaudio.Terminate()
	if err != nil {
		log.Println("Port Audio Termination has failed.")
	}
	fmt.Println("Terminated Port Audio Successfully!")

	return allAudioData
}
