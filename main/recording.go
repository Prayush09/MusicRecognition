package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"flag"
	"github.com/gordonklaus/portaudio"
)

// RecordingResult holds both audio data and metadata
type RecordingResult struct {
	AudioData    []int16
	SampleRate   int
	Duration     float64
	TotalSamples int
}

// Recording returns both audio data and the actual sample rate used
func Recording() ([]int16, int) {
	result := RecordingWithInfo()
	return result.AudioData, result.SampleRate
}

// RecordingWithInfo returns complete recording information
func RecordingWithInfo() RecordingResult {
	err := portaudio.Initialize()
	if err != nil {
		log.Println("❌ Port Audio failed to initialize!")
		os.Exit(1)
	}
	log.Println("✅ Port Audio has initialized!")

	// Parse command line flags
	flag.Parse()
	rate := flag.Float64("rate", 0, "Sample rate (leave 0 to use device default)")

	inputDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		log.Println("❌ Failed to get default input device:", err)
		return RecordingResult{}
	}

	// Determine actual sample rate - prefer higher rates for better fingerprinting
	sampleRate := inputDevice.DefaultSampleRate
	if *rate > 0 {
		sampleRate = *rate
	} else if sampleRate < 44100 {
		sampleRate = 44100 // Ensure minimum quality for fingerprinting
	}

	fmt.Printf("🎤 Using device: %s (sample rate: %.0f Hz)\n", inputDevice.Name, sampleRate)

	// Configure stream parameters optimized for fingerprinting
	parameters := portaudio.HighLatencyParameters(inputDevice, nil)
	parameters.Input.Channels = 1 // Mono for fingerprinting
	parameters.SampleRate = sampleRate
	parameters.FramesPerBuffer = 2048 // Match spectrogram frame size

	buffer := make([]int16, 2048)
	stream, err := portaudio.OpenStream(parameters, buffer)
	if err != nil {
		log.Println("❌ Stream Parameters have not been set:", err)
		return RecordingResult{}
	}

	err = stream.Start()
	if err != nil {
		log.Println("❌ Stream has failed to start:", err)
		return RecordingResult{}
	}

	fmt.Println("🔴 Recording for 5 seconds...")
	var allAudioData []int16
	startTime := time.Now()

	// Record with progress indication
	for time.Since(startTime) < 5*time.Second {
		err := stream.Read()
		if err != nil {
			log.Println("❌ Recording of audio has failed:", err)
			break
		}
		allAudioData = append(allAudioData, buffer...)
		
		// Progress indicator
		elapsed := time.Since(startTime).Seconds()
		if int(elapsed)%1 == 0 && elapsed < 5 {
			fmt.Printf("⏱️  Recording... %.0f/5 seconds\n", elapsed)
		}
	}

	stream.Stop()
	
	actualSampleRate := int(stream.Info().SampleRate)
	duration := float64(len(allAudioData)) / float64(actualSampleRate)

	fmt.Printf("📊 Recording complete!\n")
	fmt.Printf("   Total samples: %d\n", len(allAudioData))
	fmt.Printf("   Actual sample rate: %d Hz\n", actualSampleRate)
	fmt.Printf("   Recording duration: %.2f seconds\n", duration)
	fmt.Println("🔍 Starting spectrogram processing...")

	err = portaudio.Terminate()
	if err != nil {
		log.Println("⚠️  Port Audio Termination has failed:", err)
	} else {
		fmt.Println("✅ Terminated Port Audio Successfully!")
	}

	return RecordingResult{
		AudioData:    allAudioData,
		SampleRate:   actualSampleRate,
		Duration:     duration,
		TotalSamples: len(allAudioData),
	}
}

// RecordingWithQualityCheck performs additional quality checks
func RecordingWithQualityCheck() RecordingResult {
	result := RecordingWithInfo()
	
	// Quality validation
	if len(result.AudioData) < result.SampleRate { // Less than 1 second
		fmt.Printf("⚠️  Warning: Recording too short (%.2fs)\n", result.Duration)
	}
	
	if result.SampleRate < 22050 {
		fmt.Printf("⚠️  Warning: Low sample rate (%d Hz) may affect fingerprinting accuracy\n", result.SampleRate)
	}
	
	// Check for silence or very low signal
	var totalEnergy int64
	for _, sample := range result.AudioData {
		totalEnergy += int64(sample * sample)
	}
	avgEnergy := float64(totalEnergy) / float64(len(result.AudioData))
	
	if avgEnergy < 1000 {
		fmt.Printf("⚠️  Warning: Very low signal level detected (avg energy: %.0f)\n", avgEnergy)
	} else {
		fmt.Printf("✅ Audio quality check passed (avg energy: %.0f)\n", avgEnergy)
	}
	
	return result
}
