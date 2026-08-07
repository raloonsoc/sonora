package ingest

import "testing"

func TestProbeFilePath(t *testing.T) {
	result, err := probeFile("testdata/test.flac")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format.Tags["title"] != "Test Song" {
		t.Errorf("title = %q, want %q", result.Format.Tags["title"], "Test Song")
	}
	if result.Format.Tags["artist"] != "Test Artist" {
		t.Errorf("artist = %q, want %q", result.Format.Tags["artist"], "Test Artist")
	}
	if result.Format.Duration != "2.000000" {
		t.Errorf("duration = %q, want %q", result.Format.Duration, "2.000000")
	}

	if len(result.Streams) != 1 {
		t.Fatalf("len(Streams) = %d, want 1", len(result.Streams))
	}
	if result.Streams[0].CodecType != "audio" {
		t.Errorf("Streams[0].CodecType = %q, want %q", result.Streams[0].CodecType, "audio")
	}
	if result.Streams[0].Channels != 1 {
		t.Errorf("Streams[0].Channels = %d, want 1", result.Streams[0].Channels)
	}
}

func TestProbeFilePath_MissingFile(t *testing.T) {
	_, err := probeFile("testdata/does-not-exist.flac")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
