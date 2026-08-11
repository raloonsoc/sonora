package subsonic

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

// testQueries connects to the real Postgres used by CI/local dev (see
// SONORA_TEST_DATABASE_URL) and truncates the tables this package's tests
// touch. subsonic handlers are read/write over real SQL (starring,
// browsing), so a mock would only prove the handler calls some interface —
// it would not catch a query returning the wrong rows, which is the class
// of bug this package actually had (see seedStarredTrack's doc comment).
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

	if _, err := pool.Exec(ctx, "TRUNCATE tracks, albums, artists, users, starred_items CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	return sqlc.New(pool)
}
