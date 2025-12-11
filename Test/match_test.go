package core_test

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "path/filepath"
    "shazoom/core"
    "shazoom/fileformat"
    "sort"
    "sync"
    "testing"
    "time"

    "github.com/gordonklaus/portaudio"
)

const (
    sampleRate = 44100
    Channels   = 1
)

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

    stream, err := portaudio.OpenDefaultStream(Channels, 0, sampleRate, 0, callback)
    if err != nil {
        t.Fatalf("Failed to initalize stream: %v", err)
    }
    defer stream.Close()

    t.Log("Recording for the next 10 seconds (Fresh Sample)...")
    err = stream.Start()
    if err != nil {
        t.Fatalf("Stream failed to start recording: %v", err)
    }

    time.Sleep(10 * time.Second)

    err = stream.Stop()
    if err != nil {
        t.Fatalf("error occured while closing stream: %v", err)
    }

    fmt.Printf("Total size of recorded sample  %d\n", len(audioBytes))

    return audioBytes
}

func TestMatching(t *testing.T) {
    audioBytes := recordSample(t)

    const BITS_PER_SAMPLE = 16
    const CHANNELS = 1
    tempDir := t.TempDir()
    rawWavPath := filepath.Join(tempDir, "raw_recording.wav")

    err := fileformat.WriteWavFile(rawWavPath, audioBytes, sampleRate, CHANNELS, BITS_PER_SAMPLE)
    if err != nil {
        t.Fatalf("Failed to write raw WAV file: %v", err)
    }

    audioBytes = nil

    defer func() {
        os.Remove(rawWavPath)
        t.Log("Cleanup: Deleted raw_recording.wav")
    }()

    reformatedWavFile, err := fileformat.ReformatWav(rawWavPath, CHANNELS)
    if err != nil {
        t.Fatalf("Failed to reformat WAV: %v", err)
    }

    defer func() {
        os.Remove(reformatedWavFile)
        t.Log("Cleanup: Deleted reformatted .wav file")
    }()

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

    sort.Slice(matches, func(i, j int) bool {
        return matches[i].Score > matches[j].Score
    })

    if len(matches) == 0 {
        t.Fatal("No matches found in the database")
    }

    const expectedTitle = "Le Aaunga"
    match := matches[0]

    printMatchDebug := func(idx int) string {
        if idx < len(matches) {
            return fmt.Sprintf("Title: %s, Score: %.2f", matches[idx].SongTitle, matches[idx].Score)
        }
        return "N/A"
    }

    if expectedTitle != match.SongTitle {
        t.Fatalf("\nFailed to match with the expected title: '%s' != '%s'\n|| Top 3 Candidates:\n1. %s\n2. %s\n3. %s\n",
            expectedTitle, match.SongTitle,
            printMatchDebug(0),
            printMatchDebug(1),
            printMatchDebug(2))
    } else {
        t.Logf("SUCCESS: Matched '%s' by %s (Score: %.2f)", match.SongTitle, match.SongArtist, match.Score)
    }
}