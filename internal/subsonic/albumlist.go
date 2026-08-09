package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type albumListSubsonicResponse struct {
	baseResponse
	AlbumList2 albumListEntry `json:"albumList2" xml:"albumList2"`
}

type albumListEntry struct {
	Album []albumListItem `json:"album" xml:"album"`
}

type albumListItem struct {
	ID        string    `json:"id" xml:"id,attr"`
	Album     string    `json:"album" xml:"album,attr"`
	Title     string    `json:"title" xml:"title,attr"`
	Name      string    `json:"name" xml:"name,attr"`
	CoverArt  string    `json:"coverArt" xml:"coverArt,attr"`
	SongCount int       `json:"songCount" xml:"songCount,attr"`
	Created   time.Time `json:"created" xml:"created,attr"`
	Duration  int       `json:"duration" xml:"duration,attr"`
	PlayCount int       `json:"playCount" xml:"playCount,attr"`
	ArtistID  string    `json:"artistId" xml:"artistId,attr"`
	Artist    string    `json:"artist" xml:"artist,attr"`
	Year      int       `json:"year" xml:"year,attr"`
	Genre     string    `json:"genre" xml:"genre"`
}

type albumRow struct {
	ID              pgtype.UUID
	Title           string
	ArtistID        pgtype.UUID
	ReleaseYear     pgtype.Int4
	CreatedAt       pgtype.Timestamptz
	DurationSeconds int32
	PlayCount       int32
	SongCount       int32
	Genre           string
}

type genresSubsonicResponse struct {
	baseResponse
	Genres genresEntry `json:"genres" xml:"genres"`
}

type genresEntry struct {
	Genre []genreItem `json:"genre" xml:"genre"`
}

type genreItem struct {
	Value      string `json:"value" xml:"value,attr"`
	SongCount  int    `json:"songCount" xml:"songCount,attr"`
	AlbumCount int    `json:"albumCount" xml:"albumCount,attr"`
}

func (h *Handler) GetAlbumListHandler(w http.ResponseWriter, r *http.Request) {
	size := int32(500)
	if v := r.URL.Query().Get("size"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			size = int32(parsed)
		}
	}
	offset := int32(0)
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			offset = int32(parsed)
		}
	}

	listType := r.URL.Query().Get("type")

	var albums []albumRow
	var err error
	if listType == "newest" {
		rows, e := h.Queries.ListAlbumsNewest(r.Context(), sqlc.ListAlbumsNewestParams{Limit: size, Offset: offset})
		err = e
		for _, row := range rows {
			albums = append(albums, albumRow{
				ID:              row.ID,
				Title:           row.Title,
				ArtistID:        row.ArtistID,
				ReleaseYear:     row.ReleaseYear,
				CreatedAt:       row.CreatedAt,
				DurationSeconds: row.DurationSeconds,
				PlayCount:       row.PlayCount,
				SongCount:       row.SongCount,
				Genre:           row.Genre,
			})
		}
	} else {
		rows, e := h.Queries.ListAlbumsAlphabetical(r.Context(), sqlc.ListAlbumsAlphabeticalParams{Limit: size, Offset: offset})
		err = e
		for _, row := range rows {
			albums = append(albums, albumRow{
				ID:              row.ID,
				Title:           row.Title,
				ArtistID:        row.ArtistID,
				ReleaseYear:     row.ReleaseYear,
				CreatedAt:       row.CreatedAt,
				DurationSeconds: row.DurationSeconds,
				PlayCount:       row.PlayCount,
				SongCount:       row.SongCount,
				Genre:           row.Genre,
			})
		}
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	artistNames := make(map[string]string)
	items := make([]albumListItem, 0, len(albums))
	for _, a := range albums {
		artistID := a.ArtistID.String()
		artistName, ok := artistNames[artistID]
		if !ok {
			artist, err := h.Queries.GetArtist(r.Context(), a.ArtistID)
			if err == nil {
				artistName = artist.Name
			}
			artistNames[artistID] = artistName
		}

		year := 0
		if a.ReleaseYear.Valid {
			year = int(a.ReleaseYear.Int32)
		}
		items = append(items, albumListItem{
			ID:        a.ID.String(),
			Album:     a.Title,
			Title:     a.Title,
			Name:      a.Title,
			CoverArt:  a.ID.String(),
			ArtistID:  artistID,
			Artist:    artistName,
			Created:   a.CreatedAt.Time,
			SongCount: int(a.SongCount),
			Genre:     a.Genre,
			Year:      year,
			PlayCount: int(a.PlayCount),
			Duration:  int(a.DurationSeconds),
		})
	}

	encodeResponse(w, r, albumListSubsonicResponse{
		baseResponse: newBaseResponse(),
		AlbumList2:   albumListEntry{Album: items},
	})
}

func (h *Handler) GetGenresHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Queries.ListGenres(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	items := make([]genreItem, 0, len(rows))
	for _, g := range rows {
		items = append(items, genreItem{
			Value:      g.Genre,
			SongCount:  int(g.SongCount),
			AlbumCount: int(g.AlbumCount),
		})
	}

	encodeResponse(w, r, genresSubsonicResponse{
		baseResponse: newBaseResponse(),
		Genres:       genresEntry{Genre: items},
	})
}
