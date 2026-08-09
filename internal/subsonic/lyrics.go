package subsonic

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
	"github.com/raloonsoc/sonora/internal/lyrics"
)

type lyricsListSubsonicResponse struct {
	baseResponse
	LyricsList lyricsListElement `json:"lyricsList" xml:"lyricsList"`
}

type lyricsListElement struct {
	StructuredLyrics []structuredLyrics `json:"structuredLyrics" xml:"structuredLyrics"`
}

type structuredLyrics struct {
	Lang   string      `json:"lang" xml:"lang,attr"`
	Synced bool        `json:"synced" xml:"synced,attr"`
	Line   []lyricLine `json:"line" xml:"line"`
}

type lyricLine struct {
	Start int    `json:"start" xml:"start,attr"`
	Value string `json:"value" xml:",chardata"`
}

// Legacy (pre-OpenSubsonic) lyrics response: plain text only, no timestamps.
type legacyLyricsSubsonicResponse struct {
	baseResponse
	Lyrics legacyLyrics `json:"lyrics" xml:"lyrics"`
}

type legacyLyrics struct {
	Artist string `json:"artist,omitempty" xml:"artist,attr,omitempty"`
	Title  string `json:"title,omitempty" xml:"title,attr,omitempty"`
	Value  string `json:"value" xml:",chardata"`
}

func (h *Handler) LyricsHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var id pgtype.UUID

	if err := id.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	track, err := h.Queries.GetTrack(r.Context(), id)
	if err != nil {
		http.Error(w, "invalid track", http.StatusNotFound)
		return
	}

	artist, err := h.Queries.GetArtist(r.Context(), track.ArtistID)
	if err != nil {
		http.Error(w, "invalid artist", http.StatusNotFound)
		return
	}

	lines, err := lyrics.Lookup(r.Context(), track.Path, artist.Name, track.Title, "", int(track.DurationSeconds), h.LyricsLRCLIBFallback)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var respLines []lyricLine

	for _, line := range lines {
		respLines = append(respLines, lyricLine{
			Start: line.TimestampMs,
			Value: line.Text,
		})
	}

	encodeResponse(w, r, lyricsListSubsonicResponse{
		baseResponse: newBaseResponse(),
		LyricsList: lyricsListElement{
			StructuredLyrics: []structuredLyrics{
				{
					Lang:   "und",
					Synced: true,
					Line:   respLines,
				},
			},
		},
	})
}

// LegacyLyricsHandler implements the pre-OpenSubsonic getLyrics endpoint:
// lookup by free-text artist/title (no id) instead of a track id, and a
// flat plain-text response instead of structuredLyrics.
func (h *Handler) LegacyLyricsHandler(w http.ResponseWriter, r *http.Request) {
	artistParam := r.URL.Query().Get("artist")
	titleParam := r.URL.Query().Get("title")
	if titleParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	candidates, err := h.Queries.SearchTracks(r.Context(), sqlc.SearchTracksParams{
		Column1: pgtype.Text{String: titleParam, Valid: true},
		Limit:   50,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var track *sqlc.Track
	var artist sqlc.Artist

	for _, candidate := range candidates {
		candidateArtist, err := h.Queries.GetArtist(r.Context(), candidate.ArtistID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if artistParam == "" || strings.EqualFold(candidateArtist.Name, artistParam) {
			c := candidate
			track = &c
			artist = candidateArtist
			break
		}
	}

	if track == nil {
		encodeResponse(w, r, legacyLyricsSubsonicResponse{
			baseResponse: newBaseResponse(),
			Lyrics:       legacyLyrics{},
		})
		return
	}

	lines, err := lyrics.Lookup(r.Context(), track.Path, artist.Name, track.Title, "", int(track.DurationSeconds), h.LyricsLRCLIBFallback)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	textLines := make([]string, 0, len(lines))
	for _, line := range lines {
		textLines = append(textLines, line.Text)
	}

	encodeResponse(w, r, legacyLyricsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Lyrics: legacyLyrics{
			Artist: artist.Name,
			Title:  track.Title,
			Value:  strings.Join(textLines, "\n"),
		},
	})
}
