package subsonic

import (
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

// Artist types
type artistsSubsonicResponse struct {
	baseResponse
	Artists artistElement `json:"artists" xml:"artists"`
}

type artistElement struct {
	Index []artistsIndex `json:"index" xml:"index"`
}

type artistsIndex struct {
	Name    string        `json:"name" xml:"name,attr"`
	Artists []artistEntry `json:"artist" xml:"artist"`
}

type artistEntry struct {
	ID         string `json:"id" xml:"id,attr"`
	Name       string `json:"name" xml:"name,attr"`
	AlbumCount int    `json:"albumCount" xml:"albumCount,attr"`
}

// Album types
type albumSubsonicResponse struct {
	baseResponse
	Album albumWithSongs `json:"album" xml:"album"`
}

type albumWithSongs struct {
	ID        string      `json:"id" xml:"id,attr"`
	Name      string      `json:"name" xml:"name,attr"`
	Artist    string      `json:"artist" xml:"artist,attr"`
	ArtistID  string      `json:"artistId" xml:"artistId,attr"`
	SongCount int         `json:"songCount" xml:"songCount,attr"`
	Song      []songEntry `json:"song" xml:"song"`
}

type songEntry struct {
	ID          string `json:"id" xml:"id,attr"`
	Title       string `json:"title" xml:"title,attr"`
	Album       string `json:"album" xml:"album,attr"`
	AlbumID     string `json:"albumId" xml:"albumId,attr"`
	Artist      string `json:"artist" xml:"artist,attr"`
	ArtistID    string `json:"artistId" xml:"artistId,attr"`
	CoverArt    string `json:"coverArt" xml:"coverArt,attr"`
	Track       int    `json:"track" xml:"track,attr"`
	Duration    int    `json:"duration" xml:"duration,attr"`
	Suffix      string `json:"suffix" xml:"suffix,attr"`
	ContentType string `json:"contentType" xml:"contentType,attr"`
	IsDir       bool   `json:"isDir" xml:"isDir,attr"`
	Type        string `json:"type" xml:"type,attr"`
}

// contentTypeForFormat maps a track's audio format (as stored by ffprobe,
// e.g. "flac") to its MIME type. Subsonic clients use this to decide how
// to handle the stream.
func contentTypeForFormat(format string) string {
	switch format {
	case "flac":
		return "audio/x-flac"
	case "mp3":
		return "audio/mpeg"
	case "aac", "m4a":
		return "audio/mp4"
	case "opus":
		return "audio/opus"
	case "ogg", "vorbis":
		return "audio/ogg"
	case "wav":
		return "audio/wav"
	default:
		return "audio/" + format
	}
}

type albumEntry struct {
	ID       string `json:"id" xml:"id,attr"`
	Name     string `json:"name" xml:"name,attr"`
	Artist   string `json:"artist" xml:"artist,attr"`
	ArtistID string `json:"artistId" xml:"artistId,attr"`
}

// ArtistWithAlbums types
type artistWithAlbumsSubsonicResponse struct {
	baseResponse
	Artist artistWithAlbums `json:"artist" xml:"artist"`
}

type artistWithAlbums struct {
	ID         string       `json:"id" xml:"id,attr"`
	Name       string       `json:"name" xml:"name,attr"`
	AlbumCount int          `json:"albumCount" xml:"albumCount,attr"`
	Album      []albumEntry `json:"album" xml:"album"`
}

// Search3 types
type search3SubsonicResponse struct {
	baseResponse
	SearchResult3 searchResult3 `json:"searchResult3" xml:"searchResult3"`
}

type searchResult3 struct {
	Artist []artistEntry `json:"artist" xml:"artist"`
	Album  []albumEntry  `json:"album" xml:"album"`
	Song   []songEntry   `json:"song" xml:"song"`
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

	encodeResponse(w, r, artistsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Artists:      artistElement{Index: index},
	})
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
			ID:          s.ID.String(),
			Title:       s.Title,
			Album:       album.Title,
			AlbumID:     album.ID.String(),
			Artist:      artist.Name,
			ArtistID:    artist.ID.String(),
			CoverArt:    album.ID.String(),
			Track:       int(s.TrackNumber),
			Duration:    int(s.DurationSeconds),
			Suffix:      s.Format,
			ContentType: contentTypeForFormat(s.Format),
			IsDir:       false,
			Type:        "music",
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

	encodeResponse(w, r, albumSubsonicResponse{
		baseResponse: newBaseResponse(),
		Album:        albumComplete,
	})
}

func (h *Handler) GetArtistHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var artistId pgtype.UUID
	if err := artistId.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	artist, err := h.Queries.GetArtist(r.Context(), artistId)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	albums, err := h.Queries.ListAlbumsByArtist(r.Context(), artistId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	items := make([]albumEntry, 0, len(albums))
	for _, a := range albums {
		items = append(items, albumEntry{
			ID:       a.ID.String(),
			Name:     a.Title,
			Artist:   artist.Name,
			ArtistID: artist.ID.String(),
		})
	}

	encodeResponse(w, r, artistWithAlbumsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Artist: artistWithAlbums{
			ID:         artist.ID.String(),
			Name:       artist.Name,
			AlbumCount: len(items),
			Album:      items,
		},
	})
}

func (h *Handler) GetSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	// Pagination (artistOffset/albumOffset/songOffset) isn't implemented
	// beyond a single page — a nonzero offset means the client already
	// has page one, so return empty results instead of repeating it
	// forever (clients that page in a loop until they see an empty
	// response would otherwise never stop).
	for _, param := range []string{"artistOffset", "albumOffset", "songOffset"} {
		if v := r.URL.Query().Get(param); v != "" && v != "0" {
			encodeResponse(w, r, search3SubsonicResponse{
				baseResponse: newBaseResponse(),
			})
			return
		}
	}

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
			ID:          s.ID.String(),
			Title:       s.Title,
			Track:       int(s.TrackNumber),
			Duration:    int(s.DurationSeconds),
			Suffix:      s.Format,
			ContentType: contentTypeForFormat(s.Format),
			IsDir:       false,
			Type:        "music",
		})
	}

	encodeResponse(w, r, search3SubsonicResponse{
		baseResponse: newBaseResponse(),
		SearchResult3: searchResult3{
			Artist: artists,
			Album:  albums,
			Song:   songs,
		},
	})
}
