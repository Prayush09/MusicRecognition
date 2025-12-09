package core_test

import (
	"shazoom/core"
	"testing"
)

func TestFullPipeline(t *testing.T) {
	t.Log("Starting TestFullPipeline with real audio data.")

	samples, sampleRate, duration := LoadRealAudio(t)

	if len(samples) == 0 {
		t.Fatal("FATAL: LoadRealAudio returned zero samples. Check common.go logic.")
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
		// Check time bounds
		if peak.Time < 0 || peak.Time > duration {
			t.Errorf("Peak %d has invalid time %.2f (should be between 0 and %.2f)",
				i, peak.Time, duration)
		}

		// Check frequency is within expected range (0-5000 Hz based on your maxFreq)
		if peak.Freq > 5000 {
			t.Errorf("Peak %d has frequency %.2f Hz exceeding max of 5000 Hz", i, peak.Freq)
		}
	}

	// ----------------------------------------------------------------------
	// 5.FINGERPRINTING
	// ----------------------------------------------------------------------
	const TEST_SONG_ID = 9876

	fingerprints, err := core.GenerateFingerprintsFromSamples(samples, sampleRate, TEST_SONG_ID)
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
