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
