package subsonic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/streaming"
)

// Every Subsonic client sends endpoints with a legacy ".view" suffix
// (getArtists.view), and some also without it. NewRouter registers both
// forms for every route via its register() closure; this test locks in
// that both forms actually reach the handler, since a route added without
// going through register() would silently only work for one of the two.
func TestNewRouter_RegistersBothViewSuffixedAndBareRoutes(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")

	mux := NewRouter(&Handler{Queries: queries}, &streaming.Handler{Queries: queries})

	for _, path := range []string{"/rest/ping", "/rest/ping.view"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestNewRouter_UnknownRouteIs404(t *testing.T) {
	queries := testQueries(t)
	mux := NewRouter(&Handler{Queries: queries}, &streaming.Handler{Queries: queries})

	req := httptest.NewRequest(http.MethodGet, "/rest/doesNotExist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
