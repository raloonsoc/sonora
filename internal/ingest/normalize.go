package ingest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/raloonsoc/sonora/internal/domain"
)

func trackFromProbe(path string, output ffprobeOutput) (domain.Track, error) {
	durationFloat, err := strconv.ParseFloat(output.Format.Duration, 64)
	if err != nil {
		return domain.Track{}, fmt.Errorf("ingest: parsing duration: %w", err)
	}

	durationSeconds := int(durationFloat)

	var audioStream ffprobeStream
	found := false
	for _, s := range output.Streams {
		if s.CodecType == "audio" {
			audioStream = s
			found = true
			break
		}
	}

	if !found {
		return domain.Track{}, fmt.Errorf("ingest: no audio stream found in %s", path)
	}

	title := output.Format.Tags["title"]
	genre := output.Format.Tags["genre"]
	trackNumberStr, _, _ := strings.Cut(output.Format.Tags["track"], "/")
	trackNumber, _ := strconv.Atoi(trackNumberStr)
	discNumberStr, _, _ := strings.Cut(output.Format.Tags["disc"], "/")
	discNumber, _ := strconv.Atoi(discNumberStr)
	bitRate, _ := strconv.Atoi(output.Format.BitRate)
	sizeBytes, _ := strconv.ParseInt(output.Format.Size, 10, 64)
	track := domain.Track{
		DurationSeconds: durationSeconds,
		Title:           title,
		Genre:           genre,
		Path:            path,
		TrackNumber:     trackNumber,
		DiscNumber:      discNumber,
		Format:          audioStream.CodecName,
		BitRate:         bitRate,
		SizeBytes:       sizeBytes,
	}
	return track, nil
}
