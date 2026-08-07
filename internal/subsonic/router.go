package subsonic

import (
	"net/http"

	"github.com/raloonsoc/sonora/internal/streaming"
)

func NewRouter(subsonicHandler *Handler, streamHandler *streaming.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /rest/ping", subsonicHandler.PingHandler)
	mux.HandleFunc("GET /rest/getArtists", subsonicHandler.GetArtistsHandler)
	mux.HandleFunc("GET /rest/getAlbum", subsonicHandler.GetAlbumHandler)
	mux.HandleFunc("GET /rest/getCoverArt", subsonicHandler.GetCoverArtHandler)
	mux.HandleFunc("GET /rest/stream", streamHandler.StreamHandler)
	mux.HandleFunc("GET /rest/search3", subsonicHandler.GetSearchHandler)
	return mux
}
