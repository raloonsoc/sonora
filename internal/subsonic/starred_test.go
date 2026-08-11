package subsonic

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func starItemParams(userID pgtype.UUID, itemType string, itemID pgtype.UUID) sqlc.StarItemParams {
	return sqlc.StarItemParams{UserID: userID, ItemType: itemType, ItemID: itemID}
}

// createAlbumParams/createTrackParams centralize the boilerplate for
// building a minimal, valid album/track row in tests that need to control
// the track's path (e.g. lyrics tests need a real temp path to place a
// sibling .lrc file next to).
func createAlbumParams(artistID pgtype.UUID) sqlc.CreateAlbumParams {
	return sqlc.CreateAlbumParams{Title: "Test Album", ArtistID: artistID}
}

func createTrackParams(title string, albumID, artistID pgtype.UUID, path string) sqlc.CreateTrackParams {
	return sqlc.CreateTrackParams{
		Title:           title,
		AlbumID:         albumID,
		ArtistID:        artistID,
		DurationSeconds: 180,
		Path:            path,
		Format:          "flac",
	}
}

// seedArtistAlbumTrack inserts the minimal artist/album/track graph a track
// row needs (both are NOT NULL foreign keys), returning the track. It also
// links the artist in track_artists, mirroring what the real ingest
// pipeline does: ListArtists/SearchArtists JOIN track_artists to filter out
// album_artist-only rows (see CLAUDE.md), so an artist with no
// track_artists row is invisible to those queries and any test seeding
// data without it would silently diverge from production.
func seedArtistAlbumTrack(t *testing.T, queries *sqlc.Queries, title string) sqlc.Track {
	t.Helper()
	ctx := context.Background()

	artist, err := queries.CreateArtist(ctx, "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	album, err := queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{
		Title:    "Test Album",
		ArtistID: artist.ID,
	})
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}
	track, err := queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title:           title,
		AlbumID:         album.ID,
		ArtistID:        artist.ID,
		DurationSeconds: 180,
		Path:            "/music/" + title + ".flac",
		Format:          "flac",
	})
	if err != nil {
		t.Fatalf("creating track: %v", err)
	}
	if err := queries.CreateTrackArtist(ctx, sqlc.CreateTrackArtistParams{
		TrackID: track.ID, ArtistID: artist.ID, Position: 0,
	}); err != nil {
		t.Fatalf("linking track_artists: %v", err)
	}
	return track
}

func TestStarHandler_StarsTrack(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Song A")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/star?u=alice&id="+track.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.StarHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	starredAt, err := queries.GetStarredAt(context.Background(), sqlc.GetStarredAtParams{
		UserID: user.ID, ItemType: "track", ItemID: track.ID,
	})
	if err != nil {
		t.Fatalf("track was not recorded as starred: %v", err)
	}
	if !starredAt.Valid {
		t.Error("starred_at is not valid")
	}
}

func TestUnstarHandler_RemovesStar(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Song A")
	if err := queries.StarItem(context.Background(), sqlc.StarItemParams{
		UserID: user.ID, ItemType: "track", ItemID: track.ID,
	}); err != nil {
		t.Fatalf("seeding star: %v", err)
	}
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/unstar?u=alice&id="+track.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.UnstarHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if _, err := queries.GetStarredAt(context.Background(), sqlc.GetStarredAtParams{
		UserID: user.ID, ItemType: "track", ItemID: track.ID,
	}); err == nil {
		t.Error("track is still recorded as starred after unstar")
	}
}

// This is the regression test for the bug documented in CLAUDE.md: Starred
// fields must be *time.Time, not time.Time, because encoding/json's
// omitempty does not treat a zero-value struct as empty — a plain
// time.Time field always serializes, which made every item look starred
// in Feishin regardless of whether the user had actually starred it.
//
// It seeds two tracks, stars only one, and asserts the unstarred track's
// "starred" field is genuinely absent from both JSON and XML — not present
// with a zero-value timestamp, which would look "false-y" to a human
// skimming the JSON but is not what Subsonic clients check for.
func TestGetStarred2Handler_OnlyReturnsStarredItems(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	starred := seedArtistAlbumTrack(t, queries, "Starred Song")
	notStarred := seedArtistAlbumTrack(t, queries, "Unstarred Song")

	if err := queries.StarItem(context.Background(), sqlc.StarItemParams{
		UserID: user.ID, ItemType: "track", ItemID: starred.ID,
	}); err != nil {
		t.Fatalf("seeding star: %v", err)
	}

	h := &Handler{Queries: queries}

	t.Run("JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/getStarred2?u=alice&f=json", nil)
		rec := httptest.NewRecorder()
		h.GetStarred2Handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var payload struct {
			SubsonicResponse struct {
				Starred2 struct {
					Song []map[string]any `json:"song"`
				} `json:"starred2"`
			} `json:"subsonic-response"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
		}

		songs := payload.SubsonicResponse.Starred2.Song
		if len(songs) != 1 {
			t.Fatalf("got %d starred songs, want 1 (unstarred track must not appear at all): %+v", len(songs), songs)
		}
		if songs[0]["id"] != starred.ID.String() {
			t.Errorf("starred song id = %v, want %s", songs[0]["id"], starred.ID.String())
		}
		if _, present := songs[0]["starred"]; !present {
			t.Error(`starred song is missing its own "starred" timestamp field`)
		}

		body := rec.Body.String()
		if strings.Contains(body, notStarred.ID.String()) {
			t.Errorf("unstarred track %s must not appear anywhere in the response body", notStarred.ID.String())
		}
	})

	t.Run("XML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rest/getStarred2?u=alice", nil)
		rec := httptest.NewRecorder()
		h.GetStarred2Handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var payload struct {
			XMLName  xml.Name `xml:"subsonic-response"`
			Starred2 struct {
				Song []struct {
					ID      string `xml:"id,attr"`
					Starred string `xml:"starred,attr"`
				} `xml:"song"`
			} `xml:"starred2"`
		}
		if err := xml.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decoding XML response: %v\nbody: %s", err, rec.Body.String())
		}

		songs := payload.Starred2.Song
		if len(songs) != 1 {
			t.Fatalf("got %d starred songs, want 1: %+v", len(songs), songs)
		}
		if songs[0].ID != starred.ID.String() {
			t.Errorf("starred song id = %q, want %q", songs[0].ID, starred.ID.String())
		}
		if songs[0].Starred == "" {
			t.Error(`starred song is missing its "starred" XML attribute`)
		}
	})
}

func TestGetStarred2Handler_NoStarredItems(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	seedArtistAlbumTrack(t, queries, "Never Starred")

	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getStarred2?u=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetStarred2Handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Starred2 struct {
				Song []map[string]any `json:"song"`
			} `json:"starred2"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.Starred2.Song) != 0 {
		t.Errorf("got %d starred songs, want 0", len(payload.SubsonicResponse.Starred2.Song))
	}
}

func TestStarHandler_InvalidID(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/star?u=alice&id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.StarHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestStarHandler_UnknownUser(t *testing.T) {
	queries := testQueries(t)
	track := seedArtistAlbumTrack(t, queries, "Song A")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/star?u=ghost&id="+track.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.StarHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
