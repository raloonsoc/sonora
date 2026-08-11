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
	TranscodeSem      chan struct{}
}

func (h *Handler) StreamHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	var trackID pgtype.UUID
	if err := trackID.Scan(idParam); err != nil {
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	track, err := h.Queries.GetTrack(r.Context(), trackID)
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	// format=raw (OpenSubsonic 1.9.0+) explicitly disables transcoding,
	// same as when the source is already within the passthrough range.
	requestedFormat := r.URL.Query().Get("format")
	needsTranscode := track.SampleRate > 48000 && requestedFormat != "raw"

	sourcePath := track.Path
	contentType := ""

	if needsTranscode {
		format := ResolveFormat(requestedFormat)
		cachePath := CachePath(h.TranscodeCacheDir, track.ID.String(), format.Extension)
		mu := lockForPath(cachePath)
		if !CacheExists(cachePath) {
			err := func() error {
				h.TranscodeSem <- struct{}{}
				defer func() { <-h.TranscodeSem }()
				return Transcode(track.Path, cachePath, format)
			}()

			if err != nil {
				unlockForPath(cachePath, mu)
				http.Error(w, "transcode failed", http.StatusInternalServerError)
				return
			}
		}
		unlockForPath(cachePath, mu)
		sourcePath = cachePath
		contentType = format.ContentType
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

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
