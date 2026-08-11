package streaming

import (
	"fmt"
	"os"
	"os/exec"
)

func TranscodeToOpus(sourcePath, destPath string) error {
	tmpPath := destPath + ".tmp"
	cmd := exec.Command("ffmpeg", "-y", "-i", sourcePath, "-vn", "-c:a", "libopus", "-b:a", "128k", "-f", "ogg", tmpPath)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("streaming: transcoding to opus: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("streaming: moving transcoded file into place: %w", err)
	}
	return nil
}
