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

// An id that isn't an album falls through to artist lookup — artist
// photos are a remote Deezer URL, not a local file, so they're served via
// redirect rather than os.Open.
func TestGetCoverArtHandler_RedirectsForArtistImage(t *testing.T) {
	queries := testQueries(t)
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}
	if err := queries.UpdateArtistImageURL(t.Context(), sqlc.UpdateArtistImageURLParams{
		ID: artist.ID, ImageUrl: "https://example.com/artist.jpg",
	}); err != nil {
		t.Fatalf("seeding artist image: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id="+artist.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "https://example.com/artist.jpg" {
		t.Errorf("Location = %q, want %q", got, "https://example.com/artist.jpg")
	}
}

func TestGetCoverArtHandler_ArtistWithNoImage(t *testing.T) {
	queries := testQueries(t)
	artist, err := queries.CreateArtist(t.Context(), "Test Artist")
	if err != nil {
		t.Fatalf("creating artist: %v", err)
	}

	h := &Handler{Queries: queries}
	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id="+artist.ID.String(), nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetCoverArtHandler_NeitherAlbumNorArtist(t *testing.T) {
	queries := testQueries(t)
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id=00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	h.GetCoverArtHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
