package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// deezerNoPictureHash is Deezer's artist placeholder image: the MD5 of an
// empty string. Deezer's artist search never returns "not found" — it
// fuzzy-matches and always returns a result — so this hash embedded in the
// picture URL is the only signal that no real photo exists for the query.
const deezerNoPictureHash = "d41d8cd98f00b204e9800998ecf8427e"

func isPlaceholderArtistImage(pictureURL string) bool {
	return pictureURL == "" || strings.Contains(pictureURL, deezerNoPictureHash)
}

const deezerSearchArtistURL = "https://api.deezer.com/search/artist"

// FetchArtistImageURL looks up an artist photo via the Deezer public
// search API (no API key required). iTunes Search — already used for
// album cover art fallback — has no equivalent for artists: its
// musicArtist entity returns no artwork field at all.
func FetchArtistImageURL(ctx context.Context, artistName string) (string, error) {
	return fetchArtistImageURL(ctx, http.DefaultClient, deezerSearchArtistURL, artistName)
}

func fetchArtistImageURL(ctx context.Context, client *http.Client, searchURL, artistName string) (string, error) {
	u, err := url.Parse(searchURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", artistName)
	q.Set("limit", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ingest: artist image lookup returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			PictureXL string `json:"picture_xl"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Data) == 0 {
		return "", nil
	}

	pictureURL := result.Data[0].PictureXL
	if isPlaceholderArtistImage(pictureURL) {
		return "", nil
	}

	return pictureURL, nil
}
