package ingest

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

var audioExtensions = map[string]bool{
	".flac": true,
	".mp3":  true,
	".m4a":  true,
	".aac":  true,
	".opus": true,
	".ogg":  true,
	".wav":  true,
}

func WatchLibrary(path string, interval time.Duration, queries *sqlc.Queries, processed chan<- string) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		paths, err := queries.ListTrackPaths(context.Background())
		if err != nil {
			return fmt.Errorf("ingest: listing track paths %s: %w", path, err)
		}
		pathRoute := make(map[string]bool, len(paths))
		for _, p := range paths {
			pathRoute[p] = true
		}
		filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil //ignore folders
			}
			if pathRoute[p] {
				return nil //exists in db
			}
			if !audioExtensions[filepath.Ext(p)] {
				return nil //not valid filetype
			}
			processed <- p
			return nil
		})
	}
	return nil
}
