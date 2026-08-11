package subsonic

import (
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Handler) GetCoverArtHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var albumId pgtype.UUID
	if err := albumId.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	album, err := h.Queries.GetAlbum(r.Context(), albumId)
	if err != nil {
		// Not an album id — try artist next. Artist photos are stored as
		// a remote URL (Deezer), not a local file, so they're served via
		// redirect rather than os.Open below.
		artist, artistErr := h.Queries.GetArtist(r.Context(), albumId)
		if artistErr != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if artist.ImageUrl == "" {
			http.Error(w, "artist image not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, artist.ImageUrl, http.StatusFound)
		return
	}

	if album.CoverArtPath == "" {
		http.Error(w, "album cover art not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(album.CoverArtPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
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
