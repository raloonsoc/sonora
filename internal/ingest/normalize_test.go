package ingest

import "testing"

func TestTrackFromProbe(t *testing.T) {
	output, err := probeFile("testdata/test.flac")
	if err != nil {
		t.Fatalf("probeFile: unexpected error: %v", err)
	}

	track, err := trackFromProbe("testdata/test.flac", output)
	if err != nil {
		t.Fatalf("trackFromProbe: unexpected error: %v", err)
	}

	if track.Title != "Test Song" {
		t.Errorf("Title = %q, want %q", track.Title, "Test Song")
	}
	if track.Genre != "Electronic" {
		t.Errorf("Genre = %q, want %q", track.Genre, "Electronic")
	}
	if track.DurationSeconds != 2 {
		t.Errorf("DurationSeconds = %d, want 2", track.DurationSeconds)
	}
	if track.Path != "testdata/test.flac" {
		t.Errorf("Path = %q, want %q", track.Path, "testdata/test.flac")
	}
	if track.TrackNumber != 3 {
		t.Errorf("TrackNumber = %d, want 3", track.TrackNumber)
	}
}

func TestTrackFromProbe_TrackNumberWithTotal(t *testing.T) {
	output := ffprobeOutput{
		Format: ffprobeFormat{
			Duration: "1.0",
			Tags:     map[string]string{"track": "3/12", "disc": "2/2"},
		},
		Streams: []ffprobeStream{{CodecType: "audio"}},
	}

	track, err := trackFromProbe("fake.flac", output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if track.TrackNumber != 3 {
		t.Errorf("TrackNumber = %d, want 3", track.TrackNumber)
	}
	if track.DiscNumber != 2 {
		t.Errorf("DiscNumber = %d, want 2", track.DiscNumber)
	}
}

func TestTrackFromProbe_NoAudioStream(t *testing.T) {
	output := ffprobeOutput{
		Format:  ffprobeFormat{Duration: "1.0"},
		Streams: []ffprobeStream{{CodecType: "video"}},
	}

	_, err := trackFromProbe("fake.mp4", output)
	if err == nil {
		t.Fatal("expected error for missing audio stream, got nil")
	}
}
