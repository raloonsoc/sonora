package streaming

import (
	"fmt"
	"os"
	"os/exec"
)

// TranscodeFormat describes one target format Sonora can transcode to:
// the ffmpeg args needed to produce it, the file extension for the cache
// path, and the MIME type to report to the client.
type TranscodeFormat struct {
	Extension   string
	ContentType string
	ffmpegArgs  []string
}

// Opus is the default: at equal bitrate it's the most transparent of the
// three to the ear, so a client that doesn't ask for a specific format
// (via the OpenSubsonic "format" query param) gets the best fidelity
// Sonora can offer without falling back to lossless.
var formatOpus = TranscodeFormat{
	Extension:   "opus",
	ContentType: "audio/opus",
	ffmpegArgs:  []string{"-vn", "-c:a", "libopus", "-b:a", "128k", "-f", "ogg"},
}

// AAC needs a fragmented MP4 container (-f ipod -movflags
// frag_keyframe+empty_moov), not raw ADTS. Plain "-c:a aac" streams ADTS,
// which Safari, macOS/iOS apps, and Amperfy fail to play at all — this is
// a documented, reproducible bug in that combination (see
// navidrome/navidrome#2194), not a Sonora-specific workaround. 192k
// because AAC needs a higher bitrate than Opus to reach comparable
// transparency; matching Opus's 128k here would be a real fidelity loss,
// not just a compatibility trade-off.
var formatAAC = TranscodeFormat{
	Extension:   "m4a",
	ContentType: "audio/mp4",
	ffmpegArgs:  []string{"-vn", "-c:a", "aac", "-b:a", "192k", "-f", "ipod", "-movflags", "frag_keyframe+empty_moov"},
}

// MP3 is the lowest-fidelity of the three at a comparable bitrate, but is
// the closest thing to a universally-supported fallback across Subsonic
// clients — offered only when a client explicitly asks for it.
var formatMP3 = TranscodeFormat{
	Extension:   "mp3",
	ContentType: "audio/mpeg",
	// -f mp3 is required for the same reason Opus needs -f ogg: the
	// destination is written to "<dest>.tmp" for atomic rename (see
	// Transcode), so ffmpeg can't infer the container from a ".mp3"
	// extension that isn't actually there at write time.
	ffmpegArgs: []string{"-vn", "-c:a", "libmp3lame", "-b:a", "192k", "-f", "mp3"},
}

var transcodeFormats = map[string]TranscodeFormat{
	"opus": formatOpus,
	"aac":  formatAAC,
	"mp3":  formatMP3,
}

// ResolveFormat maps an OpenSubsonic "format" query param to a supported
// TranscodeFormat, defaulting to Opus when requested is empty or not one
// Sonora knows how to produce.
func ResolveFormat(requested string) TranscodeFormat {
	if f, ok := transcodeFormats[requested]; ok {
		return f
	}
	return formatOpus
}

// Transcode converts sourcePath to the given format at destPath. The
// output is written to a temp file and renamed into place only on
// success, so a crash mid-transcode or a source ffmpeg can't decode never
// leaves a corrupt or partial file at destPath for CacheExists to
// mistake for a valid cache entry.
func Transcode(sourcePath, destPath string, format TranscodeFormat) error {
	tmpPath := destPath + ".tmp"
	args := append([]string{"-y", "-i", sourcePath}, format.ffmpegArgs...)
	args = append(args, tmpPath)

	cmd := exec.Command("ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("streaming: transcoding to %s: %w", format.Extension, err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("streaming: moving transcoded file into place: %w", err)
	}
	return nil
}
