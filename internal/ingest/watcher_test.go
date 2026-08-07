package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestWatchLibrary_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("creating watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		t.Fatalf("watching dir: %v", err)
	}

	filePath := filepath.Join(dir, "new-track.flac")
	if err := os.WriteFile(filePath, []byte("fake audio data"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	select {
	case event := <-watcher.Events:
		if event.Name != filePath {
			t.Errorf("event.Name = %q, want %q", event.Name, filePath)
		}
	case err := <-watcher.Errors:
		t.Fatalf("watcher error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for file event")
	}
}

func TestWatchLibrary_Debounce(t *testing.T) {
	dir := t.TempDir()
	processed := make(chan string)

	go func() {
		if err := WatchLibrary(dir, 200*time.Millisecond, processed); err != nil {
			t.Errorf("WatchLibrary returned error: %v", err)
		}
	}()

	filePath := filepath.Join(dir, "new-track.flac")

	// Write the file multiple times quickly, simulating a slow copy.
	// Only one "processed" notification should arrive, after the debounce
	// window closes without any new writes.
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filePath, []byte("chunk"), 0o644); err != nil {
			t.Fatalf("writing test file: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case gotPath := <-processed:
		if gotPath != filePath {
			t.Errorf("processed path = %q, want %q", gotPath, filePath)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for debounced notification")
	}

	select {
	case gotPath := <-processed:
		t.Errorf("received unexpected second notification: %q", gotPath)
	case <-time.After(500 * time.Millisecond):
		// expected: no second notification
	}
}
