package subsonic

import (
	"net/http"

	"github.com/raloonsoc/sonora/internal/streaming"
)

// Subsonic clients traditionally suffix every endpoint with ".view"
// (a leftover from the original Java servlet API). Modern OpenSubsonic
// clients (like Feishin) still send it, so every route is registered
// twice: with and without the suffix.
func NewRouter(subsonicHandler *Handler, streamHandler *streaming.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	register := func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc("GET "+pattern, handler)
		mux.HandleFunc("GET "+pattern+".view", handler)
	}

	register("/rest/ping", subsonicHandler.PingHandler)
	register("/rest/getArtists", subsonicHandler.GetArtistsHandler)
	register("/rest/getArtist", subsonicHandler.GetArtistHandler)
	register("/rest/getAlbum", subsonicHandler.GetAlbumHandler)
	register("/rest/getSong", subsonicHandler.GetSongHandler)
	register("/rest/getCoverArt", subsonicHandler.GetCoverArtHandler)
	register("/rest/stream", streamHandler.StreamHandler)
	register("/rest/search3", subsonicHandler.GetSearchHandler)
	register("/rest/getLicense", subsonicHandler.GetLicenseHandler)
	register("/rest/getUser", subsonicHandler.GetUserHandler)
	register("/rest/getAlbumList2", subsonicHandler.GetAlbumListHandler)
	register("/rest/getGenres", subsonicHandler.GetGenresHandler)
	register("/rest/getPlaylists", subsonicHandler.GetPlaylistsHandler)
	register("/rest/getPlaylist", subsonicHandler.GetPlaylistHandler)
	register("/rest/createPlaylist", subsonicHandler.CreatePlaylistHandler)
	register("/rest/updatePlaylist", subsonicHandler.UpdatePlaylistHandler)
	register("/rest/deletePlaylist", subsonicHandler.DeletePlaylistHandler)
	register("/rest/getMusicFolders", subsonicHandler.GetMusicFoldersHandler)
	register("/rest/getOpenSubsonicExtensions", subsonicHandler.GetOpenSubsonicExtensionsHandler)
	register("/rest/scrobble", subsonicHandler.ScrobbleHandler)
	register("/rest/getLyricsBySongId", subsonicHandler.LyricsHandler)
	register("/rest/getLyrics", subsonicHandler.LegacyLyricsHandler)

	return mux
}
