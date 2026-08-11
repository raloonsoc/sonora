package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsPlaceholderArtistImage(t *testing.T) {
	cases := map[string]bool{
		"":                                                                  true,
		"https://cdn-images.dzcdn.net/images/artist/d41d8cd98f00b204e9800998ecf8427e/1000x1000-000000-80-0-0.jpg": true,
		"https://cdn-images.dzcdn.net/images/artist/96b688020014a21cb80a0268b90287f5/1000x1000-000000-80-0-0.jpg": false,
	}
	for url, want := range cases {
		if got := isPlaceholderArtistImage(url); got != want {
			t.Errorf("isPlaceholderArtistImage(%q) = %v, want %v", url, got, want)
		}
	}
}

func testDeezerServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchArtistImageURL_RealPicture(t *testing.T) {
	srv := testDeezerServer(t, `{"data":[{"picture_xl":"https://cdn-images.dzcdn.net/images/artist/96b688020014a21cb80a0268b90287f5/1000x1000-000000-80-0-0.jpg"}]}`, http.StatusOK)

	got, err := fetchArtistImageURL(t.Context(), srv.Client(), srv.URL, "Radiohead")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://cdn-images.dzcdn.net/images/artist/96b688020014a21cb80a0268b90287f5/1000x1000-000000-80-0-0.jpg"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Deezer never returns "no results" for artist search — it fuzzy-matches
// and always returns something, with the placeholder hash embedded in the
// URL as the only signal there's no real photo. FetchArtistImageURL must
// treat that the same as no match: empty string, no error.
func TestFetchArtistImageURL_PlaceholderTreatedAsNoMatch(t *testing.T) {
	srv := testDeezerServer(t, `{"data":[{"picture_xl":"https://cdn-images.dzcdn.net/images/artist/d41d8cd98f00b204e9800998ecf8427e/1000x1000-000000-80-0-0.jpg"}]}`, http.StatusOK)

	got, err := fetchArtistImageURL(t.Context(), srv.Client(), srv.URL, "qwertyuiopasdfghjkl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string for a placeholder image", got)
	}
}

func TestFetchArtistImageURL_EmptyResults(t *testing.T) {
	srv := testDeezerServer(t, `{"data":[]}`, http.StatusOK)

	got, err := fetchArtistImageURL(t.Context(), srv.Client(), srv.URL, "Nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string for empty results", got)
	}
}

func TestFetchArtistImageURL_NonOKStatus(t *testing.T) {
	srv := testDeezerServer(t, `{}`, http.StatusInternalServerError)

	_, err := fetchArtistImageURL(t.Context(), srv.Client(), srv.URL, "Radiohead")
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}
