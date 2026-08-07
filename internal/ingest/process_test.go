package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func testQueries(t *testing.T) *sqlc.Queries {
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

	return sqlc.New(pool)
}

func TestProcessFile_CreatesArtistAlbumTrack(t *testing.T) {
	queries := testQueries(t)
	ctx := context.Background()
	coverDir := t.TempDir()

	err := ProcessFile(ctx, "testdata/test.flac", queries, coverDir)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	artist, err := queries.GetArtistByName(ctx, "Test Artist")
	if err != nil {
		t.Fatalf("artist was not created: %v", err)
	}

	album, err := queries.GetAlbumByTitleAndArtist(ctx, sqlc.GetAlbumByTitleAndArtistParams{
		Title:    "Test Album",
		ArtistID: artist.ID,
	})
	if err != nil {
		t.Fatalf("album was not created: %v", err)
	}

	tracks, err := queries.ListTracksByAlbum(ctx, album.ID)
	if err != nil {
		t.Fatalf("ListTracksByAlbum: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Title != "Test Song" {
		t.Errorf("track title = %q, want %q", tracks[0].Title, "Test Song")
	}
	if tracks[0].TrackNumber != 3 {
		t.Errorf("track number = %d, want 3", tracks[0].TrackNumber)
	}
	if tracks[0].Channels == 0 {
		t.Error("channels should not be zero")
	}
	if tracks[0].SampleRate == 0 {
		t.Error("sample rate should not be zero")
	}
}

func TestProcessFile_ReusesExistingArtistAndAlbum(t *testing.T) {
	queries := testQueries(t)
	ctx := context.Background()
	coverDir := t.TempDir()

	if err := ProcessFile(ctx, "testdata/test.flac", queries, coverDir); err != nil {
		t.Fatalf("first ProcessFile: %v", err)
	}
	if err := ProcessFile(ctx, "testdata/test_with_cover.flac", queries, coverDir); err != nil {
		t.Fatalf("second ProcessFile: %v", err)
	}

	artists, err := queries.ListArtists(ctx)
	if err != nil {
		t.Fatalf("ListArtists: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists, want 1 (should be deduplicated)", len(artists))
	}

	albums, err := queries.ListAlbumsByArtist(ctx, artists[0].ID)
	if err != nil {
		t.Fatalf("ListAlbumsByArtist: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums, want 1 (should be deduplicated)", len(albums))
	}

	tracks, err := queries.ListTracksByAlbum(ctx, albums[0].ID)
	if err != nil {
		t.Fatalf("ListTracksByAlbum: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
}

func TestProcessFile_ExtractsCoverArt(t *testing.T) {
	queries := testQueries(t)
	ctx := context.Background()
	coverDir := t.TempDir()

	if err := ProcessFile(ctx, "testdata/test_with_cover.flac", queries, coverDir); err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}

	artist, err := queries.GetArtistByName(ctx, "Test Artist")
	if err != nil {
		t.Fatalf("artist was not created: %v", err)
	}
	album, err := queries.GetAlbumByTitleAndArtist(ctx, sqlc.GetAlbumByTitleAndArtistParams{
		Title:    "Test Album",
		ArtistID: artist.ID,
	})
	if err != nil {
		t.Fatalf("album was not created: %v", err)
	}

	if album.CoverArtPath == "" {
		t.Fatal("album.CoverArtPath was not set")
	}

	info, err := os.Stat(filepath.Join(coverDir, album.ID.String()+".jpg"))
	if err != nil {
		t.Fatalf("cover art file was not written: %v", err)
	}
	if info.Size() == 0 {
		t.Error("cover art file is empty")
	}
}

func TestProcessFile_MissingArtistTag(t *testing.T) {
	queries := testQueries(t)
	ctx := context.Background()
	coverDir := t.TempDir()

	err := ProcessFile(ctx, "testdata/no_artist.flac", queries, coverDir)
	if err == nil {
		t.Fatal("expected an error for a file missing the artist tag, got nil")
	}
}
