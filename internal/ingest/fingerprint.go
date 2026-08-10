package ingest

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type FingerprintOut struct {
	Duration    float64 `json:"duration"`
	Fingerprint string  `json:"fingerprint"`
}

func FingerprintFile(path string) (string, error) {
	cmd := exec.Command("fpcalc", "-json", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ingest: calculating fingerprint file %s: %w", path, err)
	}
	var fingerprintOutput FingerprintOut
	if err := json.Unmarshal(output, &fingerprintOutput); err != nil {
		return "", fmt.Errorf("ingest: parsing fpcalc output: %w", err)
	}

	return fingerprintOutput.Fingerprint, nil
}
