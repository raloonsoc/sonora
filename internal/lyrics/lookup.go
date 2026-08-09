package lyrics

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func Lookup(ctx context.Context, audioPath, artist, title, album string, durationSeconds int, lrclibFallback bool) ([]Line, error) {
	ext := filepath.Ext(audioPath)
	lrcPath := strings.TrimSuffix(audioPath, ext) + ".lrc"

	var lrcText string

	content, err := os.ReadFile(lrcPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if !lrclibFallback {
			return nil, nil
		}

		lrcText, err = FetchFromLRCLIB(ctx, artist, title, album, durationSeconds)
		if err != nil {
			return nil, err
		}

		if lrcText == "" {
			lrcSearch, err := SearchLRCLIB(ctx, artist, title)
			if err != nil {
				return nil, err
			}
			if lrcSearch == "" {
				return nil, nil
			}
			lrcText = lrcSearch
		}
	} else {
		lrcText = string(content)
	}
	return Parse(lrcText)
}
