package subsonic

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func TestGetCoverArtHandler_ServesFile(t *testing.T) {
	queries := testQueries(t)
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}

	coverPath := t.TempDir() + "/cover.jpg"
	if err := os.WriteFile(coverPath, []byte("fake jpeg bytes"), 0o644); err != nil {
		t.Fatalf("writing cover file: %v", err)
	}
	album, err := queries.CreateAlbum(t.Context(), sqlc.CreateAlbumParams{
		Title: "Test Album", ArtistID: artist.ID, CoverArtPath: coverPath,
	})
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id="+album.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "fake jpeg bytes" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "fake jpeg bytes")
	}
}

func TestGetCoverArtHandler_NoCoverArtPath(t *testing.T) {
	queries := testQueries(t)
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	album, err := queries.CreateAlbum(t.Context(), sqlc.CreateAlbumParams{Title: "Test Album", ArtistID: artist.ID})
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id="+album.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetCoverArtHandler_FileMissingOnDisk(t *testing.T) {
	queries := testQueries(t)
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	album, err := queries.CreateAlbum(t.Context(), sqlc.CreateAlbumParams{
		Title: "Test Album", ArtistID: artist.ID, CoverArtPath: "/nonexistent/path/cover.jpg",
	})
	if err != nil {
		t.Fatalf("creating album: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id="+album.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetCoverArtHandler_MissingID(t *testing.T) {
	queries := testQueries(t)
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt", nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
