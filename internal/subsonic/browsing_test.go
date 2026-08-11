package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

// seedNamedArtist creates an artist with its own track, linked via
// track_artists — see seedArtistAlbumTrack's doc comment for why the link
// is required for the artist to be visible to ListArtists at all.
func seedNamedArtist(t *testing.T, queries *sqlc.Queries, name string) sqlc.Artist {
	t.Helper()
	ctx := t.Context()

	artist, err := queries.CreateArtist(ctx, name)
	if err != nil {
		t.Fatalf("creating artist %q: %v", name, err)
	}
	album, err := queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{Title: name + " Album", ArtistID: artist.ID})
	if err != nil {
		t.Fatalf("creating album for %q: %v", name, err)
	}
	track, err := queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title: name + " Track", AlbumID: album.ID, ArtistID: artist.ID,
		DurationSeconds: 180, Path: "/music/" + name + ".flac", Format: "flac",
	})
	if err != nil {
		t.Fatalf("creating track for %q: %v", name, err)
	}
	if err := queries.CreateTrackArtist(ctx, sqlc.CreateTrackArtistParams{
		TrackID: track.ID, ArtistID: artist.ID, Position: 0,
	}); err != nil {
		t.Fatalf("linking track_artists for %q: %v", name, err)
	}
	return artist
}

func TestGetArtistsHandler_GroupsByFirstLetterAndSorts(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	for _, name := range []string{"Zebra", "Apple", "Zeppelin", "apple sauce"} {
		seedNamedArtist(t, queries, name)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getArtists?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetArtistsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Artists struct {
				Index []struct {
					Name   string `json:"name"`
					Artist []struct {
						Name string `json:"name"`
					} `json:"artist"`
				} `json:"index"`
			} `json:"artists"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}

	index := payload.SubsonicResponse.Artists.Index
	if len(index) != 2 {
		t.Fatalf("got %d letter groups, want 2 (A, Z): %+v", len(index), index)
	}
	// Letters must come out sorted: A before Z.
	if index[0].Name != "A" || index[1].Name != "Z" {
		t.Errorf("letter groups = [%s, %s], want [A, Z]", index[0].Name, index[1].Name)
	}
	if len(index[0].Artist) != 2 {
		t.Errorf("group %q has %d artists, want 2 (Apple, apple sauce)", index[0].Name, len(index[0].Artist))
	}
	if len(index[1].Artist) != 2 {
		t.Errorf("group %q has %d artists, want 2 (Zebra, Zeppelin)", index[1].Name, len(index[1].Artist))
	}
}

func TestGetArtistsHandler_Empty(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getArtists?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetArtistsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Artists struct {
				Index []any `json:"index"`
			} `json:"artists"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.Artists.Index) != 0 {
		t.Errorf("got %d letter groups for an empty library, want 0", len(payload.SubsonicResponse.Artists.Index))
	}
}

func TestGetAlbumHandler_ReturnsAlbumWithSongsAndStarredState(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")

	if err := queries.StarItem(t.Context(), starItemParams(user.ID, "album", track.AlbumID)); err != nil {
		t.Fatalf("starring album: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbum?u=alice&id="+track.AlbumID.String()+"&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Album struct {
				ID        string           `json:"id"`
				SongCount int              `json:"songCount"`
				Starred   string           `json:"starred"`
				Song      []map[string]any `json:"song"`
			} `json:"album"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}

	album := payload.SubsonicResponse.Album
	if album.ID != track.AlbumID.String() {
		t.Errorf("album id = %q, want %q", album.ID, track.AlbumID.String())
	}
	if album.SongCount != 1 {
		t.Errorf("songCount = %d, want 1", album.SongCount)
	}
	if album.Starred == "" {
		t.Error(`starred album is missing its "starred" timestamp`)
	}
	if len(album.Song) != 1 || album.Song[0]["id"] != track.ID.String() {
		t.Errorf("song list = %+v, want exactly track %s", album.Song, track.ID.String())
	}
	// The track itself was never starred, only its album — the per-song
	// "starred" field must not leak the album's starred state onto it.
	if _, present := album.Song[0]["starred"]; present {
		t.Error(`unstarred track inside a starred album must not carry a "starred" field`)
	}
}

func TestGetAlbumHandler_NotFound(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbum?u=alice&id=00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetAlbumHandler_MissingID(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbum?u=alice", nil)
	rec := httptest.NewRecorder()
	h.GetAlbumHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
