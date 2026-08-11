package subsonic

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScrobbleHandler_IncrementsPlayCount(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/scrobble?u=alice&id="+track.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.ScrobbleHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	updated, err := queries.GetTrack(t.Context(), track.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if updated.PlayCount != 1 {
		t.Errorf("play_count = %d, want 1", updated.PlayCount)
	}
	if !updated.LastPlayedAt.Valid {
		t.Error("last_played_at was not set")
	}
}

// submission=false is how clients report "now playing" without counting it
// as a completed play — Sonora must accept it as a no-op, not an error, and
// must not touch play_count.
func TestScrobbleHandler_SubmissionFalseIsNoOp(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/scrobble?u=alice&id="+track.ID.String()+"&submission=false", nil)
	rec := httptest.NewRecorder()
	h.ScrobbleHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	updated, err := queries.GetTrack(t.Context(), track.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if updated.PlayCount != 0 {
		t.Errorf("play_count = %d, want 0 (submission=false must not count as a play)", updated.PlayCount)
	}
}

func TestScrobbleHandler_MissingID(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/scrobble?u=alice", nil)
	rec := httptest.NewRecorder()
	h.ScrobbleHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestScrobbleHandler_InvalidTime(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/scrobble?u=alice&id="+track.ID.String()+"&time=not-a-number", nil)
	rec := httptest.NewRecorder()
	h.ScrobbleHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
