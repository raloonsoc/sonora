package ingest

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

func WatchLibrary(path string, debounce time.Duration, processed chan<- string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("ingest: creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		return fmt.Errorf("ingest: watching %s: %w", path, err)
	}

	timer := time.NewTimer(debounce)
	timer.Stop()
	var pendingPath string

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				pendingPath = event.Name
				timer.Stop()
				timer.Reset(debounce)
			}
		case <-timer.C:
			slog.Info("info: file ready to process", "path", pendingPath)
			processed <- pendingPath
		case err := <-watcher.Errors:
			slog.Error("ingest: watcher error", "error", err)
		}
	}
}
