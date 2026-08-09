package streaming

import (
	"fmt"
	"os/exec"
)

func TranscodeToOpus(sourcePath, destPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", sourcePath, "-vn", "-c:a", "libopus", "-b:a", "128k", destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("streaming: transcoding to opus: %w", err)
	}
	return nil
}
