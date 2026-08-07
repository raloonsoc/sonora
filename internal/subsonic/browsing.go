package subsonic

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

// Artist types
type artistResponse struct {
	SubsonicResponse artistsSubsonicResponse `json:"subsonic-response"`
}

type artistsSubsonicResponse struct {
	Status  string        `json:"status"`
	Version string        `json:"version"`
	Artists artistElement `json:"artists"`
}

type artistElement struct {
	Index []artistsIndex `json:"index"`
}

type artistsIndex struct {
	Name    string        `json:"name"`
	Artists []artistEntry `json:"artist"`
}

type artistEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
}

// Album types
type albumResponse struct {
	SubsonicResponse albumSubsonicResponse `json:"subsonic-response"`
}

type albumSubsonicResponse struct {
	Status  string         `json:"status"`
	Version string         `json:"version"`
	Album   albumWithSongs `json:"album"`
}

type albumWithSongs struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Artist    string      `json:"artist"`
	ArtistID  string      `json:"artistId"`
	SongCount int         `json:"songCount"`
	Song      []songEntry `json:"song"`
}

type songEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Album    string `json:"album"`
	Artist   string `json:"artist"`
	Track    int    `json:"track"`
	Duration int    `json:"duration"`
	Suffix   string `json:"suffix"`
}

type albumEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	ArtistID string `json:"artistId"`
}

// Search3 types

type search3Response struct {
	SubsonicResponse search3SubsonicResponse `json:"subsonic-response"`
}

type search3SubsonicResponse struct {
	Status        string        `json:"status"`
	Version       string        `json:"version"`
	SearchResult3 searchResult3 `json:"searchResult3"`
}

type searchResult3 struct {
	Artist []artistEntry `json:"artist"`
	Album  []albumEntry  `json:"album"`
	Song   []songEntry   `json:"song"`
}

func (h *Handler) GetArtistsHandler(w http.ResponseWriter, r *http.Request) {
	artists, err := h.Queries.ListArtists(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	grouped := make(map[string][]artistEntry)
	for _, a := range artists {
		letter := strings.ToUpper(string(a.Name[0]))
		grouped[letter] = append(grouped[letter], artistEntry{
			ID:   a.ID.String(),
			Name: a.Name,
		})
	}
	letters := make([]string, 0, len(grouped))
	for letter := range grouped {
		letters = append(letters, letter)
	}
	sort.Strings(letters)

	var index []artistsIndex
	for _, letter := range letters {
		index = append(index, artistsIndex{
			Name:    letter,
			Artists: grouped[letter],
		})
	}
	resp := artistResponse{
		SubsonicResponse: artistsSubsonicResponse{
			Status:  "ok",
			Version: "1.0.0",
			Artists: artistElement{Index: index},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetAlbumHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	artist, err := h.Queries.GetArtist(r.Context(), album.ArtistID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	tracks, err := h.Queries.ListTracksByAlbum(r.Context(), album.ID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var songs []songEntry

	for _, s := range tracks {
		songs = append(songs, songEntry{
			ID:       s.ID.String(),
			Title:    s.Title,
			Album:    album.Title,
			Artist:   artist.Name,
			Track:    int(s.TrackNumber),
			Duration: int(s.DurationSeconds),
			Suffix:   s.Format,
		})
	}

	albumComplete := albumWithSongs{
		ID:        album.ID.String(),
		Name:      album.Title,
		Artist:    artist.Name,
		ArtistID:  artist.ID.String(),
		SongCount: len(songs),
		Song:      songs,
	}

	resp := albumResponse{
		SubsonicResponse: albumSubsonicResponse{
			Status:  "ok",
			Version: "1.0.0",
			Album:   albumComplete,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}

func (h *Handler) GetSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	searchTerm := pgtype.Text{String: query, Valid: true}

	artistsResult, err := h.Queries.SearchArtists(r.Context(), sqlc.SearchArtistsParams{Column1: searchTerm, Limit: 20})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	albumsResult, err := h.Queries.SearchAlbums(r.Context(), sqlc.SearchAlbumsParams{Column1: searchTerm, Limit: 20})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tracksResult, err := h.Queries.SearchTracks(r.Context(), sqlc.SearchTracksParams{Column1: searchTerm, Limit: 20})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var artists []artistEntry
	for _, a := range artistsResult {
		artists = append(artists, artistEntry{
			ID:   a.ID.String(),
			Name: a.Name,
		})
	}

	var albums []albumEntry
	for _, a := range albumsResult {
		albums = append(albums, albumEntry{
			ID:   a.ID.String(),
			Name: a.Title,
		})
	}

	var songs []songEntry
	for _, s := range tracksResult {
		songs = append(songs, songEntry{
			ID:       s.ID.String(),
			Title:    s.Title,
			Track:    int(s.TrackNumber),
			Duration: int(s.DurationSeconds),
			Suffix:   s.Format,
		})
	}
	resp := search3Response{
		SubsonicResponse: search3SubsonicResponse{
			Status:  "ok",
			Version: "1.0.0",
			SearchResult3: searchResult3{
				Artist: artists,
				Album:  albums,
				Song:   songs,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
