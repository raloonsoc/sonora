package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func TestCreatePlaylistHandler_ByName(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track One")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/createPlaylist?u=alice&name=Road+Trip&songId="+track.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.CreatePlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Playlist struct {
				Name      string `json:"name"`
				Owner     string `json:"owner"`
				SongCount int    `json:"songCount"`
			} `json:"playlist"`
		} `json:"subsonic-response"`
	}
	req2 := httptest.NewRequest(http.MethodGet, "/rest/createPlaylist?u=alice&name=Road+Trip&songId="+track.ID.String()+"&f=json", nil)
	rec2 := httptest.NewRecorder()
	h.CreatePlaylistHandler(rec2, req2)
	if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec2.Body.String())
	}

	if payload.SubsonicResponse.Playlist.Name != "Road Trip" {
		t.Errorf("name = %q, want %q", payload.SubsonicResponse.Playlist.Name, "Road Trip")
	}
	if payload.SubsonicResponse.Playlist.Owner != "alice" {
		t.Errorf("owner = %q, want %q", payload.SubsonicResponse.Playlist.Owner, "alice")
	}
	if payload.SubsonicResponse.Playlist.SongCount != 1 {
		t.Errorf("songCount = %d, want 1", payload.SubsonicResponse.Playlist.SongCount)
	}

	playlists, err := queries.ListPlaylistsByOwner(t.Context(), user.ID)
	if err != nil || len(playlists) != 2 { // one from each request above
		t.Fatalf("ListPlaylistsByOwner: %v, %d playlists", err, len(playlists))
	}
}

func TestCreatePlaylistHandler_MissingNameAndID(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/createPlaylist?u=alice", nil)
	rec := httptest.NewRecorder()
	h.CreatePlaylistHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreatePlaylistHandler_ReplacesTracksWhenGivenExistingID(t *testing.T) {
	queries := testQueries(t)
	user := seedUser(t, queries, "alice", "pw")
	trackA := seedArtistAlbumTrack(t, queries, "Track A")
	trackB := seedArtistAlbumTrack(t, queries, "Track B")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Mix", OwnerID: user.ID})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}
	if _, err := queries.AddPlaylistTrack(t.Context(), sqlc.AddPlaylistTrackParams{
		PlaylistID: pl.ID, TrackID: trackA.ID, Position: 0,
	}); err != nil {
		t.Fatalf("seeding playlist track: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/createPlaylist?u=alice&playlistId="+pl.ID.String()+"&songId="+trackB.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.CreatePlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	tracks, err := queries.ListPlaylistTracks(t.Context(), pl.ID)
	if err != nil {
		t.Fatalf("ListPlaylistTracks: %v", err)
	}
	if len(tracks) != 1 || tracks[0].TrackID != trackB.ID {
		t.Errorf("playlist tracks = %+v, want exactly [trackB] (old tracks must be cleared)", tracks)
	}
}

func TestCreatePlaylistHandler_CannotReplaceAnotherUsersPlaylist(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "mallory", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Alice's Mix", OwnerID: owner.ID})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/createPlaylist?u=mallory&playlistId="+pl.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.CreatePlaylistHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestGetPlaylistsHandler_OnlyReturnsCallersPlaylists(t *testing.T) {
	queries := testQueries(t)
	alice := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "bob", "pw")

	if _, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Alice's", OwnerID: alice.ID}); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getPlaylists?u=bob&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetPlaylistsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Playlists struct {
				Playlist []map[string]any `json:"playlist"`
			} `json:"playlists"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.Playlists.Playlist) != 0 {
		t.Errorf("bob sees %d playlists, want 0 (alice's playlist must not leak)", len(payload.SubsonicResponse.Playlists.Playlist))
	}
}

func TestGetPlaylistHandler_OwnerCanReadPrivate(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	track := seedArtistAlbumTrack(t, queries, "Track A")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Private Mix", OwnerID: owner.ID, Public: false})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}
	if _, err := queries.AddPlaylistTrack(t.Context(), sqlc.AddPlaylistTrackParams{PlaylistID: pl.ID, TrackID: track.ID, Position: 0}); err != nil {
		t.Fatalf("seeding playlist track: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getPlaylist?u=alice&id="+pl.ID.String()+"&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetPlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetPlaylistHandler_StrangerCannotReadPrivate(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "mallory", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Private Mix", OwnerID: owner.ID, Public: false})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getPlaylist?u=mallory&id="+pl.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetPlaylistHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestGetPlaylistHandler_StrangerCanReadPublic(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "bob", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Public Mix", OwnerID: owner.ID, Public: true})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getPlaylist?u=bob&id="+pl.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetPlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetPlaylistHandler_ComputesTotalDurationFromTracks(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	trackA := seedArtistAlbumTrack(t, queries, "Track A") // 180s, from the shared helper
	trackB := seedArtistAlbumTrack(t, queries, "Track B") // 180s

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Mix", OwnerID: owner.ID, Public: true})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}
	for i, tr := range []sqlc.Track{trackA, trackB} {
		if _, err := queries.AddPlaylistTrack(t.Context(), sqlc.AddPlaylistTrackParams{
			PlaylistID: pl.ID, TrackID: tr.ID, Position: int32(i),
		}); err != nil {
			t.Fatalf("seeding playlist track: %v", err)
		}
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getPlaylist?u=alice&id="+pl.ID.String()+"&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetPlaylistHandler(rec, req)

	var payload struct {
		SubsonicResponse struct {
			Playlist struct {
				SongCount int `json:"songCount"`
				Duration  int `json:"duration"`
			} `json:"playlist"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if payload.SubsonicResponse.Playlist.SongCount != 2 {
		t.Errorf("songCount = %d, want 2", payload.SubsonicResponse.Playlist.SongCount)
	}
	if payload.SubsonicResponse.Playlist.Duration != 360 {
		t.Errorf("duration = %d, want 360 (180+180)", payload.SubsonicResponse.Playlist.Duration)
	}
}

func TestDeletePlaylistHandler_OnlyOwnerCanDelete(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "mallory", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Mix", OwnerID: owner.ID})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/deletePlaylist?u=mallory&id="+pl.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.DeletePlaylistHandler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("stranger delete: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if _, err := queries.GetPlaylist(t.Context(), pl.ID); err != nil {
		t.Fatal("playlist was deleted despite a forbidden request")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/rest/deletePlaylist?u=alice&id="+pl.ID.String(), nil)
	rec2 := httptest.NewRecorder()
	h.DeletePlaylistHandler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("owner delete: status = %d, body = %s", rec2.Code, rec2.Body.String())
	}
	if _, err := queries.GetPlaylist(t.Context(), pl.ID); err == nil {
		t.Error("playlist still exists after owner deleted it")
	}
}

func TestUpdatePlaylistHandler_RenameAndVisibility(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Old Name", OwnerID: owner.ID, Public: false})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/updatePlaylist?u=alice&playlistId="+pl.ID.String()+"&name=New+Name&public=true", nil)
	rec := httptest.NewRecorder()
	h.UpdatePlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	updated, err := queries.GetPlaylist(t.Context(), pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("name = %q, want %q", updated.Name, "New Name")
	}
	if !updated.Public {
		t.Error("public = false, want true")
	}
}

func TestUpdatePlaylistHandler_AddAndRemoveTracks(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	trackA := seedArtistAlbumTrack(t, queries, "Track A")
	trackB := seedArtistAlbumTrack(t, queries, "Track B")
	trackC := seedArtistAlbumTrack(t, queries, "Track C")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Mix", OwnerID: owner.ID})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}
	for i, tr := range []sqlc.Track{trackA, trackB} {
		if _, err := queries.AddPlaylistTrack(t.Context(), sqlc.AddPlaylistTrackParams{
			PlaylistID: pl.ID, TrackID: tr.ID, Position: int32(i),
		}); err != nil {
			t.Fatalf("seeding playlist track: %v", err)
		}
	}

	h := &Handler{Queries: queries}
	// Remove trackA (position 0), add trackC.
	req := httptest.NewRequest(http.MethodGet, "/rest/updatePlaylist?u=alice&playlistId="+pl.ID.String()+"&songIndexToRemove=0&songIdToAdd="+trackC.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.UpdatePlaylistHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	tracks, err := queries.ListPlaylistTracks(t.Context(), pl.ID)
	if err != nil {
		t.Fatalf("ListPlaylistTracks: %v", err)
	}
	got := make(map[string]bool, len(tracks))
	for _, tr := range tracks {
		got[tr.TrackID.String()] = true
	}
	if got[trackA.ID.String()] {
		t.Error("trackA should have been removed")
	}
	if !got[trackB.ID.String()] {
		t.Error("trackB should still be present")
	}
	if !got[trackC.ID.String()] {
		t.Error("trackC should have been added")
	}
}

func TestUpdatePlaylistHandler_ForbiddenForNonOwner(t *testing.T) {
	queries := testQueries(t)
	owner := seedUser(t, queries, "alice", "pw")
	seedUser(t, queries, "mallory", "pw")

	pl, err := queries.CreatePlaylist(t.Context(), sqlc.CreatePlaylistParams{Name: "Mix", OwnerID: owner.ID})
	if err != nil {
		t.Fatalf("seeding playlist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/updatePlaylist?u=mallory&playlistId="+pl.ID.String()+"&name=Hijacked", nil)
	rec := httptest.NewRecorder()
	h.UpdatePlaylistHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	unchanged, err := queries.GetPlaylist(t.Context(), pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if unchanged.Name != "Mix" {
		t.Errorf("name = %q, want unchanged %q", unchanged.Name, "Mix")
	}
}
