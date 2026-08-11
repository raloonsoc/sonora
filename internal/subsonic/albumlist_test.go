package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func TestGetAlbumListHandler_AlphabeticalIsDefault(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	seedArtistAlbumTrack(t, queries, "Track One")

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbumList2?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			AlbumList2 struct {
				Album []map[string]any `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.AlbumList2.Album) != 1 {
		t.Fatalf("got %d albums, want 1", len(payload.SubsonicResponse.AlbumList2.Album))
	}
	if payload.SubsonicResponse.AlbumList2.Album[0]["artist"] != "Test Artist" {
		t.Errorf("artist = %v, want %q", payload.SubsonicResponse.AlbumList2.Album[0]["artist"], "Test Artist")
	}
}

func TestGetAlbumListHandler_MarksStarredAlbums(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")

	if err := queries.StarItem(t.Context(), starItemParams(user.ID, "album", track.AlbumID)); err != nil {
		t.Fatalf("starring album: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbumList2?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumListHandler(rec, req)

	var payload struct {
		SubsonicResponse struct {
			AlbumList2 struct {
				Album []map[string]any `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if _, present := payload.SubsonicResponse.AlbumList2.Album[0]["starred"]; !present {
		t.Error(`starred album is missing its "starred" field in getAlbumList2`)
	}
}

func TestGetAlbumListHandler_UnknownUser(t *testing.T) {
	queries := testQueries(t)
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbumList2?u=ghost", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumListHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetGenresHandler_AggregatesCounts(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	// ListGenres filters out tracks with genre = '' (see tracks.sql), so
	// the shared seedArtistAlbumTrack helper (which leaves genre unset)
	// would not show up here — this needs a track with a real genre.
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	album, err := queries.CreateAlbum(t.Context(), sqlc.CreateAlbumParams{Title: "Test Album", ArtistID: artist.ID})
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}
	if _, err := queries.CreateTrack(t.Context(), sqlc.CreateTrackParams{
		Title: "Track One", AlbumID: album.ID, ArtistID: artist.ID,
		DurationSeconds: 180, Path: "/music/track-one.flac", Format: "flac", Genre: "Rock",
	}); err != nil {
		t.Fatalf("creating track: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getGenres?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetGenresHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Genres struct {
				Genre []map[string]any `json:"genre"`
			} `json:"genres"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.Genres.Genre) != 1 {
		t.Fatalf("got %d genres, want 1: %+v", len(payload.SubsonicResponse.Genres.Genre), payload.SubsonicResponse.Genres.Genre)
	}
}
