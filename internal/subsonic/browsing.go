package subsonic

import (
	"net/http"
	"sort"
	"strings"
	"time"

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
	ID         string    `json:"id" xml:"id,attr"`
	Name       string    `json:"name" xml:"name,attr"`
	AlbumCount int       `json:"albumCount" xml:"albumCount,attr"`
	Starred    time.Time `json:"starred,omitempty" xml:"starred,attr,omitempty"`
}

// Song types
type songSubsonicResponse struct {
	baseResponse
	Song songEntry `json:"song" xml:"song"`
}

// Album types
type albumSubsonicResponse struct {
	baseResponse
	Album albumWithSongs `json:"album" xml:"album"`
}

type albumWithSongs struct {
	ID        string      `json:"id" xml:"id,attr"`
	Parent    string      `json:"parent" xml:"parent,attr"`
	Album     string      `json:"album" xml:"album,attr"`
	Title     string      `json:"title" xml:"title,attr"`
	Name      string      `json:"name" xml:"name,attr"`
	IsDir     bool        `json:"isDir" xml:"isDir,attr"`
	CoverArt  string      `json:"coverArt" xml:"coverArt,attr"`
	SongCount int         `json:"songCount" xml:"songCount,attr"`
	Created   time.Time   `json:"created" xml:"created,attr"`
	Duration  int         `json:"duration" xml:"duration,attr"`
	PlayCount int         `json:"playCount" xml:"playCount,attr"`
	ArtistID  string      `json:"artistId" xml:"artistId,attr"`
	Artist    string      `json:"artist" xml:"artist,attr"`
	Year      int         `json:"year" xml:"year,attr"`
	Genre     string      `json:"genre" xml:"genre,attr"`
	Song      []songEntry `json:"song" xml:"song"`
}

type songEntry struct {
	ID            string         `json:"id" xml:"id,attr"`
	Parent        string         `json:"parent" xml:"parent,attr"`
	Title         string         `json:"title" xml:"title,attr"`
	IsDir         bool           `json:"isDir" xml:"isDir,attr"`
	IsVideo       bool           `json:"isVideo" xml:"isVideo,attr"`
	Type          string         `json:"type" xml:"type,attr"`
	AlbumID       string         `json:"albumId" xml:"albumId,attr"`
	Album         string         `json:"album" xml:"album,attr"`
	ArtistID      string         `json:"artistId" xml:"artistId,attr"`
	Artist        string         `json:"artist" xml:"artist,attr"`
	CoverArt      string         `json:"coverArt" xml:"coverArt,attr"`
	Duration      int            `json:"duration" xml:"duration,attr"`
	BitRate       int            `json:"bitRate" xml:"bitRate,attr"`
	BitDepth      int            `json:"bitDepth" xml:"bitDepth,attr"`
	SamplingRate  int            `json:"samplingRate" xml:"samplingRate,attr"`
	ChannelCount  int            `json:"channelCount" xml:"channelCount,attr"`
	Track         int            `json:"track" xml:"track,attr"`
	Year          int            `json:"year" xml:"year,attr"`
	Genre         string         `json:"genre" xml:"genre,attr"`
	Size          int            `json:"size" xml:"size,attr"`
	DiscNumber    int            `json:"discNumber" xml:"discNumber,attr"`
	Suffix        string         `json:"suffix" xml:"suffix,attr"`
	ContentType   string         `json:"contentType" xml:"contentType,attr"`
	Path          string         `json:"path" xml:"path,attr"`
	Artists       []artistID3Ref `json:"artists" xml:"artists"`
	DisplayArtist string         `json:"displayArtist" xml:"displayArtist,attr"`
	Starred       time.Time      `json:"starred,omitempty" xml:"starred,attr,omitempty"`
}

type artistID3Ref struct {
	ID   string `json:"id" xml:"id,attr"`
	Name string `json:"name" xml:"name,attr"`
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
	Album    string    `json:"album" xml:"album,attr"`
	Artist   string    `json:"artist" xml:"artist,attr"`
	ArtistID string    `json:"artistId" xml:"artistId,attr"`
	CoverArt string    `json:"coverArt" xml:"coverArt,attr"`
	Duration int       `json:"duration" xml:"duration,attr"`
	ID       string    `json:"id" xml:"id,attr"`
	Name     string    `json:"name" xml:"name,attr"`
	Starred  time.Time `json:"starred,omitempty" xml:"starred,attr,omitempty"`
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

	album, err := h.Queries.GetAlbumWithStats(r.Context(), albumId)
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

	year := 0
	if album.ReleaseYear.Valid {
		year = int(album.ReleaseYear.Int32)
	}

	var songs []songEntry

	for _, s := range tracks {
		linkedArtists, err := h.Queries.ListArtistsByTrack(r.Context(), s.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		artistRefs, displayArtist := artistRefsAndDisplay(linkedArtists)
		songs = append(songs, songEntry{
			ID:            s.ID.String(),
			Title:         s.Title,
			Album:         album.Title,
			AlbumID:       album.ID.String(),
			Artist:        artist.Name,
			ArtistID:      artist.ID.String(),
			CoverArt:      album.ID.String(),
			Track:         int(s.TrackNumber),
			Duration:      int(s.DurationSeconds),
			Suffix:        s.Format,
			ContentType:   contentTypeForFormat(s.Format),
			IsDir:         false,
			Type:          "music",
			Genre:         s.Genre,
			DiscNumber:    int(s.DiscNumber),
			BitDepth:      int(s.BitDepth),
			SamplingRate:  int(s.SampleRate),
			ChannelCount:  int(s.Channels),
			Path:          s.Path,
			Year:          year,
			BitRate:       int(s.BitRate),
			Size:          int(s.SizeBytes),
			Artists:       artistRefs,
			DisplayArtist: displayArtist,
		})
	}

	albumComplete := albumWithSongs{
		ID:        album.ID.String(),
		Parent:    artist.ID.String(),
		Album:     album.Title,
		Title:     album.Title,
		Name:      album.Title,
		IsDir:     true,
		CoverArt:  album.ID.String(),
		SongCount: len(songs),
		Created:   album.CreatedAt.Time,
		Duration:  int(album.DurationSeconds),
		PlayCount: int(album.PlayCount),
		Artist:    artist.Name,
		ArtistID:  artist.ID.String(),
		Year:      year,
		Genre:     album.Genre,
		Song:      songs,
	}

	encodeResponse(w, r, albumSubsonicResponse{
		baseResponse: newBaseResponse(),
		Album:        albumComplete,
	})
}

func (h *Handler) GetSongHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var trackId pgtype.UUID
	if err := trackId.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	track, err := h.Queries.GetTrack(r.Context(), trackId)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	album, err := h.Queries.GetAlbum(r.Context(), track.AlbumID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	artist, err := h.Queries.GetArtist(r.Context(), track.ArtistID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	linkedArtists, err := h.Queries.ListArtistsByTrack(r.Context(), track.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	artistRefs, displayArtist := artistRefsAndDisplay(linkedArtists)

	encodeResponse(w, r, songSubsonicResponse{
		baseResponse: newBaseResponse(),
		Song: songEntry{
			ID:            track.ID.String(),
			Title:         track.Title,
			Album:         album.Title,
			AlbumID:       album.ID.String(),
			Artist:        artist.Name,
			ArtistID:      artist.ID.String(),
			CoverArt:      album.ID.String(),
			Track:         int(track.TrackNumber),
			Duration:      int(track.DurationSeconds),
			Suffix:        track.Format,
			ContentType:   contentTypeForFormat(track.Format),
			IsDir:         false,
			Type:          "music",
			Artists:       artistRefs,
			DisplayArtist: displayArtist,
		},
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

func artistRefsAndDisplay(artists []sqlc.Artist) ([]artistID3Ref, string) {
	refs := make([]artistID3Ref, 0, len(artists))
	names := make([]string, 0, len(artists))
	for _, a := range artists {
		refs = append(refs, artistID3Ref{ID: a.ID.String(), Name: a.Name})
		names = append(names, a.Name)
	}
	return refs, strings.Join(names, " & ")
}
