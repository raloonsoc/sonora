package ingest

import "testing"

func TestLoudnormAnalyze(t *testing.T) {
	result, err := loudnormAnalyze("testdata/test.flac")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.InputI == "" {
		t.Error("InputI is empty, want a parsed loudness value")
	}
}

func TestLoudnormAnalyze_MissingFile(t *testing.T) {
	_, err := loudnormAnalyze("testdata/does-not-exist.flac")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReplayGainFromLoudnorm(t *testing.T) {
	gain, err := replayGainFromLoudnorm(loudnormResult{InputI: "-21.75"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gain == nil {
		t.Fatal("gain is nil, want a value")
	}

	want := 3.75
	if *gain != want {
		t.Errorf("gain = %v, want %v", *gain, want)
	}
}

func TestReplayGainFromLoudnorm_InvalidInput(t *testing.T) {
	_, err := replayGainFromLoudnorm(loudnormResult{InputI: "not-a-number"})
	if err == nil {
		t.Fatal("expected error for invalid input_i, got nil")
	}
}
