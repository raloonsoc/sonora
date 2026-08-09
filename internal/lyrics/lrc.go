package lyrics

import (
	"strconv"
	"strings"
)

type Line struct {
	TimestampMs int
	Text        string
}

func parseTimestamp(ts string) (int, error) {
	parts := strings.Split(ts, ":")
	secParts := strings.Split(parts[1], ".")
	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, err
	}

	centis, err := strconv.Atoi(secParts[1])
	if err != nil {
		return 0, err
	}

	totalMs := minutes*60000 + seconds*1000 + centis*10

	return totalMs, nil
}

func Parse(content string) ([]Line, error) {
	lines := strings.Split(content, "\n")
	var result []Line
	for _, line := range lines {
		if !strings.HasPrefix(line, "[") {
			continue
		}
		idx := strings.Index(line, "]")
		if idx == -1 {
			continue
		}
		timestampRaw := line[1:idx]
		if len(timestampRaw) == 0 || timestampRaw[0] < '0' || timestampRaw[0] > '9' {
			continue
		}
		timestamp, err := parseTimestamp(timestampRaw)
		if err != nil {
			return nil, err
		}

		text := line[idx+1:]
		result = append(result, Line{
			TimestampMs: timestamp,
			Text:        text,
		})
	}
	return result, nil
}
