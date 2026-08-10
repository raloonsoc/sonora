package subsonic

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type starred2SubsonicResponse struct {
	baseResponse
	Starred2 starred2Element `json:"starred2" xml:"starred2"`
}

type starred2Element struct {
	Artist []artistEntry `json:"artist" xml:"artist"`
	Album  []albumEntry  `json:"album" xml:"album"`
	Song   []songEntry   `json:"song" xml:"song"`
}

func (h *Handler) StarHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("u")
	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := starOrUnstar(r, h, user.ID, true); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	encodeResponse(w, r, newBaseResponse())
}

func (h *Handler) UnstarHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("u")
	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := starOrUnstar(r, h, user.ID, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	encodeResponse(w, r, newBaseResponse())
}

func starOrUnstar(r *http.Request, h *Handler, userID pgtype.UUID, star bool) error {
	groups := map[string]string{
		"id":       "track",
		"albumId":  "album",
		"artistId": "artist",
	}

	for param, itemType := range groups {
		for _, idStr := range r.URL.Query()[param] {
			var itemID pgtype.UUID
			if err := itemID.Scan(idStr); err != nil {
				return fmt.Errorf("invalid %s", param)
			}

			if star {
				if err := h.Queries.StarItem(r.Context(), sqlc.StarItemParams{
					UserID:   userID,
					ItemType: itemType,
					ItemID:   itemID,
				}); err != nil {
					return err
				}
			} else {
				if err := h.Queries.UnstarItem(r.Context(), sqlc.UnstarItemParams{
					UserID:   userID,
					ItemType: itemType,
					ItemID:   itemID,
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (h *Handler) GetStarred2Handler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("u")
	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	starredTracks, err := h.Queries.ListStarredTracks(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	starredAlbums, err := h.Queries.ListStarredAlbums(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	starredArtists, err := h.Queries.ListStarredArtists(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var songs []songEntry

	for _, t := range starredTracks {
		album, err := h.Queries.GetAlbum(r.Context(), t.AlbumID)
		if err != nil {
			continue
		}
		artist, err := h.Queries.GetArtist(r.Context(), t.ArtistID)
		if err != nil {
			continue
		}
		linkedArtists, err := h.Queries.ListArtistsByTrack(r.Context(), t.ID)
		if err != nil {
			continue
		}
		artistRefs, displayArtist := artistRefsAndDisplay(linkedArtists)

		year := 0
		if album.ReleaseYear.Valid {
			year = int(album.ReleaseYear.Int32)
		}
		songs = append(songs, songEntry{
			ID:            t.ID.String(),
			Title:         t.Title,
			Album:         album.Title,
			AlbumID:       album.ID.String(),
			Artist:        artist.Name,
			ArtistID:      artist.ID.String(),
			CoverArt:      album.ID.String(),
			Track:         int(t.TrackNumber),
			Duration:      int(t.DurationSeconds),
			Suffix:        t.Format,
			Type:          "music",
			Genre:         t.Genre,
			DiscNumber:    int(t.DiscNumber),
			BitDepth:      int(t.BitDepth),
			SamplingRate:  int(t.SampleRate),
			ChannelCount:  int(t.Channels),
			Path:          t.Path,
			Year:          year,
			BitRate:       int(t.BitRate),
			Size:          int(t.SizeBytes),
			Artists:       artistRefs,
			DisplayArtist: displayArtist,
			Starred:       &t.StarredAt.Time,
		})
	}
	var albums []albumEntry
	for _, a := range starredAlbums {
		artist, err := h.Queries.GetArtist(r.Context(), a.ArtistID)
		if err != nil {
			continue
		}
		albums = append(albums, albumEntry{
			ID:       a.ID.String(),
			Name:     a.Title,
			Artist:   artist.Name,
			ArtistID: artist.ID.String(),
			CoverArt: a.ID.String(),
			Duration: int(a.DurationSeconds),
			Starred:  &a.StarredAt.Time,
		})
	}

	var artists []artistEntry
	for _, a := range starredArtists {
		artists = append(artists, artistEntry{
			ID:      a.ID.String(),
			Name:    a.Name,
			Starred: &a.StarredAt.Time,
		})
	}
	encodeResponse(w, r, starred2SubsonicResponse{
		baseResponse: newBaseResponse(),
		Starred2: starred2Element{
			Artist: artists,
			Album:  albums,
			Song:   songs,
		},
	})
}
