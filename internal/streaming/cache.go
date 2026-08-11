package streaming

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

func CachePath(cacheDir, trackID, extension string) string {
	return filepath.Join(cacheDir, trackID+"."+extension)
}

func CacheExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var transcodeLocks sync.Map

type trackLock struct {
	mu       sync.Mutex
	refCount atomic.Int64
}

func lockForPath(path string) *trackLock {
	actual, _ := transcodeLocks.LoadOrStore(path, &trackLock{})
	lock := actual.(*trackLock)
	lock.refCount.Add(1)
	lock.mu.Lock()
	return lock
}

func unlockForPath(path string, lock *trackLock) {
	lock.mu.Unlock()
	if lock.refCount.Add(-1) == 0 {
		transcodeLocks.Delete(path)
	}
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
