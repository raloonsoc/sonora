package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// seedTrackWithLRC is like seedArtistAlbumTrack, but points the track at a
// real temp file with a sibling .lrc, so lyrics.Lookup can read a synced
// lyrics file without any network access. LyricsLRCLIBFallback stays false
// in every test in this file for the same reason: no test here should
// depend on reaching the real LRCLIB API.
func seedTrackWithLRC(t *testing.T, h *Handler, title, artistName, lrcContent string) (trackID string) {
	t.Helper()
	dir := t.TempDir()
	audioPath := dir + "/" + title + ".flac"
	if err := os.WriteFile(audioPath, []byte("not real audio, only the path matters here"), 0o644); err != nil {
		t.Fatalf("writing fake audio file: %v", err)
	}
	if lrcContent != "" {
		if err := os.WriteFile(dir+"/"+title+".lrc", []byte(lrcContent), 0o644); err != nil {
			t.Fatalf("writing .lrc file: %v", err)
		}
	}

	artist, err := h.Queries.CreateArtist(t.Context(), artistName)
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	album, err := h.Queries.CreateAlbum(t.Context(), createAlbumParams(artist.ID))
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}
	track, err := h.Queries.CreateTrack(t.Context(), createTrackParams(title, album.ID, artist.ID, audioPath))
	if err != nil {
		t.Fatalf("creating track: %v", err)
	}
	return track.ID.String()
}

func TestLyricsHandler_ReturnsSyncedLinesFromLocalLRC(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries, LyricsLRCLIBFallback: false}

	lrc := "[00:01.00]First line\n[00:05.50]Second line\n"
	trackID := seedTrackWithLRC(t, h, "Synced Song", "Artist", lrc)

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyricsBySongId?u=alice&id="+trackID+"&f=json", nil)
	rec := httptest.NewRecorder()
	h.LyricsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			LyricsList struct {
				StructuredLyrics []struct {
					Synced bool `json:"synced"`
					Line   []struct {
						Start int    `json:"start"`
						Value string `json:"value"`
					} `json:"line"`
				} `json:"structuredLyrics"`
			} `json:"lyricsList"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}

	lines := payload.SubsonicResponse.LyricsList.StructuredLyrics[0].Line
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %+v", len(lines), lines)
	}
	if lines[0].Value != "First line" || lines[0].Start != 1000 {
		t.Errorf("line[0] = %+v, want {Start:1000, Value:\"First line\"}", lines[0])
	}
	if lines[1].Start != 5500 {
		t.Errorf("line[1].Start = %d, want 5500", lines[1].Start)
	}
}

func TestLyricsHandler_NoLocalFileAndFallbackDisabled(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries, LyricsLRCLIBFallback: false}

	trackID := seedTrackWithLRC(t, h, "No Lyrics Song", "Artist", "")

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyricsBySongId?u=alice&id="+trackID+"&f=json", nil)
	rec := httptest.NewRecorder()
	h.LyricsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			LyricsList struct {
				StructuredLyrics []struct {
					Line []any `json:"line"`
				} `json:"structuredLyrics"`
			} `json:"lyricsList"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if len(payload.SubsonicResponse.LyricsList.StructuredLyrics[0].Line) != 0 {
		t.Errorf("got %d lines for a track with no .lrc and no fallback, want 0", len(payload.SubsonicResponse.LyricsList.StructuredLyrics[0].Line))
	}
}

func TestLyricsHandler_TrackNotFound(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyricsBySongId?u=alice&id=00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	h.LyricsHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLegacyLyricsHandler_MatchesByTitleAndArtist(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries, LyricsLRCLIBFallback: false}

	seedTrackWithLRC(t, h, "Shared Title", "Right Artist", "[00:00.00]Correct song\n")
	seedTrackWithLRC(t, h, "Shared Title", "Wrong Artist", "[00:00.00]Wrong song\n")

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyrics?artist=Right+Artist&title=Shared+Title&f=json", nil)
	rec := httptest.NewRecorder()
	h.LegacyLyricsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Lyrics struct {
				Artist string `json:"artist"`
				Value  string `json:"value"`
			} `json:"lyrics"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if payload.SubsonicResponse.Lyrics.Artist != "Right Artist" {
		t.Errorf("artist = %q, want %q", payload.SubsonicResponse.Lyrics.Artist, "Right Artist")
	}
	if !strings.Contains(payload.SubsonicResponse.Lyrics.Value, "Correct song") {
		t.Errorf("value = %q, want it to contain %q", payload.SubsonicResponse.Lyrics.Value, "Correct song")
	}
}

func TestLegacyLyricsHandler_NoMatchReturnsEmptyLyrics(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyrics?artist=Nobody&title=Nonexistent+Song&f=json", nil)
	rec := httptest.NewRecorder()
	h.LegacyLyricsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Lyrics struct {
				Value string `json:"value"`
			} `json:"lyrics"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if payload.SubsonicResponse.Lyrics.Value != "" {
		t.Errorf("value = %q, want empty for no match", payload.SubsonicResponse.Lyrics.Value)
	}
}

func TestLegacyLyricsHandler_MissingTitle(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getLyrics?artist=Someone", nil)
	rec := httptest.NewRecorder()
	h.LegacyLyricsHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
