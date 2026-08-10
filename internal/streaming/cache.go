package streaming

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func CachePath(cacheDir, trackID string) string {
	return filepath.Join(cacheDir, trackID+".opus")
}

func CacheExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var transcodeLocks sync.Map

func lockForPath(path string) *sync.Mutex {
	actual, _ := transcodeLocks.LoadOrStore(path, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func CleanupExpiredCache(cacheDir string, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	return filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			return os.Remove(path)
		}
		return nil
	})
}
