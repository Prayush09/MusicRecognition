package core_test

import (
	"fmt"
	"shazoom/core"
	"sort"
	"sync"
	"testing"
	"time"
	"encoding/binary"
	"bytes"
	"github.com/gordonklaus/portaudio"
	"shazoom/fileformat"
	"path/filepath"
	"os"
)

//-----------------------------------------------------------------------
// 6. Match Test
//-----------------------------------------------------------------------

/*
	No. of songs in the DB = 3
		Requirements:
			1. TODO: Create a microphone recorder (use a package) => this will go through the pipeline (FindMatches function in shazoom-core)
			2. TODO: Verify that the connection to db is established
			3. TODO: Add a new song to the
*/

const (
	sampleRate = 44100
	Channels   = 1 //mono
)

// return type: map[uint32]uint32
func recordSample(t *testing.T) []byte {
	var mutex sync.Mutex
	var audioBytes []byte

	err := portaudio.Initialize()
	if err != nil {
		t.Fatalf("Could not initialize portaudio: %v", err)
	}

	defer portaudio.Terminate()

	callback := func(in []int16) {
        buffer := new(bytes.Buffer)
        err := binary.Write(buffer, binary.LittleEndian, in)
        if err != nil {
            t.Logf("Error writing int16 to buffer: %v", err)
            return 
        }

        mutex.Lock()
        audioBytes = append(audioBytes, buffer.Bytes()...)
        mutex.Unlock()
    }

	//callback argument helps portaudio understand which format the output is required in
	stream, err := portaudio.OpenDefaultStream(Channels, 0, sampleRate, 0, callback)
	if err != nil {
		t.Fatalf("Failed to initalize stream: %v", err)
	}
	defer stream.Close()

	//start recording
	t.Log("Recording for the next 10 seconds")
	err = stream.Start()
	if err != nil {
		t.Fatalf("Stream failed to start recording: %v", err)
	}
	
	//Keep recording in this new channel `sig`, until interrupted using CTRL+C
	time.Sleep(10 * time.Second)

	err = stream.Stop()
	if err != nil {
		t.Fatalf("error occured while closing stream: %v", err)
	}

	fmt.Printf("Total size of recorded sample data, %d/n", len(audioBytes))

	return audioBytes
}

func TestMatching(t *testing.T) {
	audioBytes := recordSample(t) // returns []byte
    
    // Define constants and paths
    const BITS_PER_SAMPLE = 16
    const CHANNELS = 1
    tempDir := t.TempDir()
    rawWavPath := filepath.Join(tempDir, "raw_recording.wav")
    
    // 2. Write the raw bytes to a file using the existing utility
    err := fileformat.WriteWavFile(rawWavPath, audioBytes, sampleRate, CHANNELS, BITS_PER_SAMPLE)
    if err != nil {
        t.Fatalf("Failed to write raw WAV file: %v", err)
    }

    // 3. Reformat the WAV file (ensure full standard compliance, Mono/44100Hz)
    // NOTE: This step using ffmpeg is still valuable for robustness!
    reformatedWavFile, err := fileformat.ReformatWav(rawWavPath, CHANNELS)
    if err != nil {
        t.Fatalf("Failed to reformat WAV: %v", err)
    }
	defer os.Remove(reformatedWavFile)

    // 4. Read the reformatted WAV data back into []float64 (where normalization happens)
    wavInfo, err := fileformat.ReadWavInfo(reformatedWavFile)
    if err != nil {
        t.Fatalf("Failed to read reformatted WAV info: %v", err)
    }
    
    finalSamples := wavInfo.LeftChannelSamples
    
    audioDuration := float64(len(finalSamples)) / float64(sampleRate)
    matches, matchTime, err := core.FindMatches(finalSamples, audioDuration, sampleRate)
	if err != nil {
		t.Fatalf("An error occurred while finding matches: %v", err)
	}

	t.Logf("Successfully found %v matches, in %v time", len(matches), matchTime)

	//sort the matches in decending order using score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) == 0{
		t.Fatal("No matches found in the database")
	}

	const expectedTitle = "Le Aaunga"
	match := matches[0]

	if expectedTitle != match.SongTitle {
		t.Fatalf("Failed to match with the expected title: %v", match.SongTitle)
	} else {
		t.Logf("The match: Title Name: %v\n, Artist Name: %v\n which matched as expected.", match.SongTitle, match.SongArtist)
	}
	
}

/*
	type Match struct{
	SongId uint32
	SongTitle string
	SongArtist string
	YoutubeID string
	Timestamp uint32
	Score float64
}
*/
