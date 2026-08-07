package ingest

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type loudnormResult struct {
	InputI       string `json:"input_i"`
	InputTP      string `json:"input_tp"`
	InputLRA     string `json:"input_lra"`
	InputThresh  string `json:"input_thresh"`
	TargetOffset string `json:"target_offset"`
}

func loudnormAnalyze(path string) (loudnormResult, error) {
	cmd := exec.Command("ffmpeg", "-i", path, "-af", "loudnorm=print_format=json", "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return loudnormResult{}, fmt.Errorf("ingest: running ffmpeg loudnorm: %w", err)
	}

	text := string(output)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 {
		return loudnormResult{}, fmt.Errorf("ingest: loudnorm JSON not found in output")
	}

	var result loudnormResult
	if err := json.Unmarshal([]byte(text[start:end+1]), &result); err != nil {
		return loudnormResult{}, fmt.Errorf("ingest: parsing loudnorm output: %w", err)
	}

	return result, nil
}

func replayGainFromLoudnorm(result loudnormResult) (*float64, error) {
	inputI, err := strconv.ParseFloat(result.InputI, 64)
	if err != nil {
		return nil, fmt.Errorf("ingest: parsing input_i: %w", err)
	}

	const referenceLUFS = -18.0
	gain := referenceLUFS - inputI

	return &gain, nil
}
