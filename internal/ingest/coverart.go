package ingest

import (
	"fmt"
	"os/exec"
)

func extractCoverArt(sourcePath, destPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", sourcePath, "-an", "-vcodec", "copy", destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ingest: extracting cover art: %w", err)
	}
	return nil
}
