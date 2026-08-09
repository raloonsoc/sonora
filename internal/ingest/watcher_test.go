package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchLibrary_DetectsNewFile(t *testing.T) {
	queries := testQueries(t)
	dir := t.TempDir()
	processed := make(chan string)

	go WatchLibrary(dir, 50*time.Millisecond, queries, processed)

	filePath := filepath.Join(dir, "new-track.flac")
	if err := os.WriteFile(filePath, []byte("fake audio data"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	select {
	case gotPath := <-processed:
		if gotPath != filePath {
			t.Errorf("processed path = %q, want %q", gotPath, filePath)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for file to be picked up")
	}
}

func TestWatchLibrary_IgnoresNonAudioFiles(t *testing.T) {
	queries := testQueries(t)
	dir := t.TempDir()
	processed := make(chan string)

	go WatchLibrary(dir, 50*time.Millisecond, queries, processed)

	if err := os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("not audio"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	select {
	case gotPath := <-processed:
		t.Errorf("received unexpected notification for non-audio file: %q", gotPath)
	case <-time.After(300 * time.Millisecond):
		// expected: no notification
	}
}
