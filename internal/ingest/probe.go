package ingest

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type ffprobeFormat struct {
	Duration string            `json:"duration"`
	Size     string            `json:"size"`
	BitRate  string            `json:"bit_rate"`
	Tags     map[string]string `json:"tags"`
}

type ffprobeStream struct {
	CodecType        string `json:"codec_type"`
	CodecName        string `json:"codec_name"`
	SampleRate       string `json:"sample_rate"`
	Channels         int    `json:"channels"`
	BitsPerRawSample string `json:"bits_per_raw_sample"`
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

func normalizeTagKeys(tags map[string]string) map[string]string {
	normalized := make(map[string]string, len(tags))
	for k, v := range tags {
		normalized[strings.ToLower(k)] = v
	}
	return normalized
}
func probeFile(path string) (ffprobeOutput, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	output, err := cmd.Output()

	if err != nil {
		return ffprobeOutput{}, fmt.Errorf("ingest: running probe: %w", err)
	}

	var result ffprobeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return ffprobeOutput{}, fmt.Errorf("ingest: parsing ffprobe output: %w", err)
	}

	result.Format.Tags = normalizeTagKeys(result.Format.Tags)

	return result, nil
}
