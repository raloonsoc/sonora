package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCoverArt(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "cover.jpg")

	err := extractCoverArt("testdata/test_with_cover.flac", destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("cover art file was not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("cover art file is empty")
	}
}

func TestExtractCoverArt_NoEmbeddedCover(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "cover.jpg")

	err := extractCoverArt("testdata/test.flac", destPath)
	if err == nil {
		t.Fatal("expected error when file has no embedded cover, got nil")
	}
}
