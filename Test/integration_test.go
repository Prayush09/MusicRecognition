package core_test

import (
	"shazoom/core"
	"testing"
)

const TEST_DURATION_SECONDS = 10.0


func TestFullPipeline(t *testing.T) {
	t.Log("Starting TestFullPipeline with real audio data.")
	
	samples, sampleRate, duration := LoadRealAudio(t)
	
	if len(samples) == 0 {
		t.Fatal("FATAL: LoadRealAudio returned zero samples. Check common.go logic.")
	}

	maxSamples := int(TEST_DURATION_SECONDS * float64(sampleRate))
    
    if len(samples) > maxSamples {
        
        samples = samples[:maxSamples] 
        
        duration = TEST_DURATION_SECONDS 
        t.Logf("Truncating audio to %d samples (%.1fs) for testing.", maxSamples, TEST_DURATION_SECONDS)
    }

	t.Logf("Successfully fetched %d samples at %d Hz (%.2fs duration)",
		len(samples), sampleRate, duration)

	// ----------------------------------------------------------------------
	// 2. SPECTROGRAM GENERATION
	// ----------------------------------------------------------------------
	spectrogram, err := core.Spectrogram(samples, sampleRate)
	if err != nil {
		t.Fatalf("Spectrogram generation failed: %v", err)
	}

	if len(spectrogram) == 0 {
		t.Fatal("Spectrogram is empty")
	}

	t.Logf("Generated spectrogram with %d time windows and %d frequency bins",
		len(spectrogram), len(spectrogram[0]))

	// ----------------------------------------------------------------------
	// 3. PEAK EXTRACTION
	// ----------------------------------------------------------------------
	peaks := core.ExtractPeaks(spectrogram, duration, sampleRate)

	if len(peaks) == 0 {
		t.Fatal("No peaks extracted from spectrogram. Check peak finding logic.")
	}

	t.Logf("Extracted %d peaks from spectrogram", len(peaks))

	// ----------------------------------------------------------------------
	// 4. VERIFY PEAK PROPERTIES 
	// ----------------------------------------------------------------------
	for i, peak := range peaks {
		if peak.Time < 0 || peak.Time > duration {
			t.Errorf("Peak %d has invalid time %.2f (should be between 0 and %.2f)",
				i, peak.Time, duration)
		}

		mag := peak.Freq
		if mag <= 0 {
			t.Errorf("Peak %d has non-positive magnitude %.2f", i, mag)
		}
	}

	for i := 1; i < len(peaks); i++ {
		if peaks[i].Time < peaks[i-1].Time {
			t.Errorf("Peaks not chronologically ordered at index %d", i)
		}
	}

	// ----------------------------------------------------------------------
	// 5.FINGERPRINTING 
	// ----------------------------------------------------------------------
	const TEST_SONG_ID = 9876
	path := GetTestPath("testdata/sample1.mp3")

	fingerprints, err := core.GenerateFingerprints(path, TEST_SONG_ID)
	if err != nil {
		t.Fatalf("core.GenerateFingerprints failed: %v", err)
	}

	if len(fingerprints) == 0 {
		t.Fatal("GenerateFingerprints returned an empty map.")
	}

	hashesToLog := make([]uint32, 0, 5)
	for hash := range fingerprints {
		if len(hashesToLog) < 5 {
			hashesToLog = append(hashesToLog, hash)
		} else {
			break
		}
	}

	t.Logf("Generated %d total fingerprints. Logging %d sample hashes:", len(fingerprints), len(hashesToLog))

	for i, hash := range hashesToLog {
		t.Logf("Sample Hash #%d: 0x%08X (Decimal: %d)", i+1, hash, hash)
	}

	// ----------------------------------------------------------------------
	// 6.TODO: MATCHING REMAINING
	// ----------------------------------------------------------------------
}
