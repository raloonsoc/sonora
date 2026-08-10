package lyrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func FetchFromLRCLIB(ctx context.Context, artist, track, album string, durationSeconds int) (string, error) {
	u, err := url.Parse("https://lrclib.net/api/get")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("artist_name", artist)
	q.Set("track_name", track)
	q.Set("album_name", album)
	q.Set("duration", strconv.Itoa(durationSeconds))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics: lrclib returned status %d", resp.StatusCode)
	}

	var result struct {
		SyncedLyrics string `json:"syncedLyrics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.SyncedLyrics, nil
}

func SearchLRCLIB(ctx context.Context, artist, track string) (string, error) {
	u, err := url.Parse("https://lrclib.net/api/search")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("q", artist+" "+track)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics: lrclib returned status %d", resp.StatusCode)
	}

	var results []struct {
		SyncedLyrics string `json:"syncedLyrics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return "", err
	}

	for _, r := range results {
		if r.SyncedLyrics != "" {
			return r.SyncedLyrics, nil
		}
	}

	return "", nil
}
