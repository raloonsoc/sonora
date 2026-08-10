package streaming

import (
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type Handler struct {
	Queries           *sqlc.Queries
	TranscodeCacheDir string
}

func (h *Handler) StreamHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	var trackID pgtype.UUID
	if err := trackID.Scan(idParam); err != nil {
		http.Error(w, "invalid id paramater", http.StatusBadRequest)
		return
	}

	track, err := h.Queries.GetTrack(r.Context(), trackID)
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	sourcePath := track.Path
	if track.SampleRate > 48000 {
		cachePath := CachePath(h.TranscodeCacheDir, track.ID.String())
		mu := lockForPath(cachePath)
		mu.Lock()
		if !CacheExists(cachePath) {
			if err := TranscodeToOpus(track.Path, cachePath); err != nil {
				mu.Unlock()
				http.Error(w, "transcode failed", http.StatusInternalServerError)
				return
			}
		}
		mu.Unlock()
		sourcePath = cachePath
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
