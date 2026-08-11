package streaming

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func testHandler(t *testing.T) (*Handler, string) {
	t.Helper()

	dbURL := os.Getenv("SONORA_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SONORA_TEST_DATABASE_URL not set, skipping test that needs postgres")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE tracks, albums, artists CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	queries := sqlc.New(pool)

	artist, err := queries.CreateArtist(ctx, "Test Artist")
	if err != nil {
		t.Fatalf("creating test artist: %v", err)
	}

	album, err := queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{
		Title:        "Test Album",
		ArtistID:     artist.ID,
		CoverArtPath: "",
	})
	if err != nil {
		t.Fatalf("creating test album: %v", err)
	}

	track, err := queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title:           "Test Song",
		AlbumID:         album.ID,
		ArtistID:        artist.ID,
		DurationSeconds: 2,
		Path:            "../ingest/testdata/test.flac",
		Format:          "flac",
	})
	if err != nil {
		t.Fatalf("creating test track: %v", err)
	}

	return &Handler{Queries: queries}, track.ID.String()
}

// testHiResHandler is like testHandler but points at a 96kHz fixture, so
// StreamHandler takes the transcode path instead of passthrough. Each
// call gets its own TranscodeCacheDir (t.TempDir()) so tests don't share
// cached output across formats.
func testHiResHandler(t *testing.T) (*Handler, string) {
	t.Helper()

	dbURL := os.Getenv("SONORA_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SONORA_TEST_DATABASE_URL not set, skipping test that needs postgres")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE tracks, albums, artists CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	queries := sqlc.New(pool)

	artist, err := queries.CreateArtist(ctx, "Test Artist")
	if err != nil {
		t.Fatalf("creating test artist: %v", err)
	}
	album, err := queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{Title: "Test Album", ArtistID: artist.ID})
	if err != nil {
		t.Fatalf("creating test album: %v", err)
	}
	track, err := queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title: "Hi-Res Song", AlbumID: album.ID, ArtistID: artist.ID,
		DurationSeconds: 1, Path: "testdata/hires.flac", Format: "flac", SampleRate: 96000,
	})
	if err != nil {
		t.Fatalf("creating test track: %v", err)
	}

	h := &Handler{
		Queries:           queries,
		TranscodeCacheDir: t.TempDir(),
		TranscodeSem:      make(chan struct{}, 2),
	}
	return h, track.ID.String()
}

func TestStreamHandler_MissingID(t *testing.T) {
	h, _ := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()

	h.StreamHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestStreamHandler_InvalidID(t *testing.T) {
	h, _ := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id=not-a-uuid", nil)
	rec := httptest.NewRecorder()

	h.StreamHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestStreamHandler_UnknownID(t *testing.T) {
	h, _ := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id=00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()

	h.StreamHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestStreamHandler_FullFile(t *testing.T) {
	h, id := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id, nil)
	rec := httptest.NewRecorder()

	h.StreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if len(body) == 0 {
		t.Error("body is empty")
	}
}

func TestStreamHandler_RangeRequest(t *testing.T) {
	h, id := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id, nil)
	req.Header.Set("Range", "bytes=0-9")
	rec := httptest.NewRecorder()

	h.StreamHandler(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusPartialContent)
	}

	if got := rec.Header().Get("Content-Range"); got == "" {
		t.Error("Content-Range header is missing")
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if len(body) != 10 {
		t.Errorf("body length = %d, want 10", len(body))
	}
}

// TestStreamHandler_TranscodeDefaultsToOpus locks in that a client which
// doesn't send the OpenSubsonic "format" param (the common case before
// any client negotiates format) still gets Opus — the most fidelity per
// bit of the three formats Sonora offers, which is what a hi-res track
// was getting before format negotiation existed at all.
func TestStreamHandler_TranscodeDefaultsToOpus(t *testing.T) {
	h, id := testHiResHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id, nil)
	rec := httptest.NewRecorder()
	h.StreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/opus" {
		t.Errorf("Content-Type = %q, want %q", got, "audio/opus")
	}
}

// TestStreamHandler_TranscodeRespectsRequestedFormat is the actual fix for
// clients like Amperfy that can't play Opus: format=aac must produce a
// playable AAC stream, not silently keep serving Opus.
func TestStreamHandler_TranscodeRespectsRequestedFormat(t *testing.T) {
	for format, wantContentType := range map[string]string{
		"aac": "audio/mp4",
		"mp3": "audio/mpeg",
	} {
		t.Run(format, func(t *testing.T) {
			h, id := testHiResHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/stream?id="+id+"&format="+format, nil)
			rec := httptest.NewRecorder()
			h.StreamHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != wantContentType {
				t.Errorf("Content-Type = %q, want %q", got, wantContentType)
			}
			if rec.Body.Len() == 0 {
				t.Error("body is empty")
			}
		})
	}
}

// format=raw (OpenSubsonic 1.9.0+) must disable transcoding entirely, even
// for a track that would otherwise be transcoded for exceeding 48kHz.
func TestStreamHandler_FormatRawDisablesTranscode(t *testing.T) {
	h, id := testHiResHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id+"&format=raw", nil)
	rec := httptest.NewRecorder()
	h.StreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	original, err := os.ReadFile("testdata/hires.flac")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if rec.Body.Len() != len(original) {
		t.Errorf("body length = %d, want %d (the original file, byte for byte)", rec.Body.Len(), len(original))
	}
}

// An unrecognized format value must not error out the request — it falls
// back to the Opus default rather than rejecting playback outright.
func TestStreamHandler_UnknownFormatFallsBackToOpus(t *testing.T) {
	h, id := testHiResHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id+"&format=nonsense", nil)
	rec := httptest.NewRecorder()
	h.StreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/opus" {
		t.Errorf("Content-Type = %q, want %q", got, "audio/opus")
	}
}

// A track at or below 48kHz must passthrough regardless of the requested
// format — Sonora only transcodes when it has to.
func TestStreamHandler_PassthroughIgnoresFormatParam(t *testing.T) {
	h, id := testHandler(t) // 44.1kHz fixture

	req := httptest.NewRequest(http.MethodGet, "/stream?id="+id+"&format=aac", nil)
	rec := httptest.NewRecorder()
	h.StreamHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	// Passthrough never sets Content-Type explicitly — it's left to
	// http.ServeContent's own extension-based detection, same as before
	// format negotiation existed.
	if got := rec.Header().Get("Content-Type"); got == "audio/mp4" {
		t.Errorf("Content-Type = %q, a 44.1kHz track must not be transcoded to AAC just because format=aac was requested", got)
	}
}
