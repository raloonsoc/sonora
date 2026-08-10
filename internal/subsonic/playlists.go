package subsonic

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type playlistsSubsonicResponse struct {
	baseResponse
	Playlists playlistsEntry `json:"playlists" xml:"playlists"`
}

type playlistsEntry struct {
	Playlist []playlistItem `json:"playlist" xml:"playlist"`
}

type playlistItem struct {
	ID        string    `json:"id" xml:"id,attr"`
	Name      string    `json:"name" xml:"name,attr"`
	Owner     string    `json:"owner" xml:"owner,attr"`
	Public    bool      `json:"public" xml:"public,attr"`
	Created   time.Time `json:"created" xml:"created,attr"`
	Changed   time.Time `json:"changed" xml:"changed,attr"`
	SongCount int       `json:"songCount" xml:"songCount,attr"`
	Duration  int       `json:"duration" xml:"duration,attr"`
}

type playlistWithSongsSubsonicResponse struct {
	baseResponse
	Playlist playlistWithSongs `json:"playlist" xml:"playlist"`
}

type playlistWithSongs struct {
	ID        string      `json:"id" xml:"id,attr"`
	Name      string      `json:"name" xml:"name,attr"`
	Owner     string      `json:"owner" xml:"owner,attr"`
	Public    bool        `json:"public" xml:"public,attr"`
	Created   time.Time   `json:"created" xml:"created,attr"`
	Changed   time.Time   `json:"changed" xml:"changed,attr"`
	SongCount int         `json:"songCount" xml:"songCount,attr"`
	Duration  int         `json:"duration" xml:"duration,attr"`
	Entry     []songEntry `json:"entry" xml:"entry"`
}

func (h *Handler) GetPlaylistsHandler(w http.ResponseWriter, r *http.Request) {

	username := r.URL.Query().Get("u")

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	playlists, err := h.Queries.ListPlaylistsByOwner(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	items := make([]playlistItem, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, playlistItem{
			ID:        p.ID.String(),
			Name:      p.Name,
			Owner:     username,
			Public:    p.Public,
			Created:   p.CreatedAt.Time,
			Changed:   p.UpdatedAt.Time,
			SongCount: int(p.SongCount),
			Duration:  int(p.DurationSeconds),
		})
	}

	encodeResponse(w, r, playlistsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Playlists:    playlistsEntry{Playlist: items},
	})
}

func (h *Handler) GetPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var playlistId pgtype.UUID
	if err := playlistId.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	playlist, err := h.Queries.GetPlaylist(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "invalid playlist", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("u")

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if playlist.OwnerID != user.ID && !playlist.Public {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tracks, err := h.Queries.ListPlaylistTracks(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "invalid tracks", http.StatusBadRequest)
		return
	}

	var songs []songEntry
	totalDuration := 0
	for _, pt := range tracks {
		track, err := h.Queries.GetTrack(r.Context(), pt.TrackID)
		if err != nil {
			continue
		}
		album, err := h.Queries.GetAlbum(r.Context(), track.AlbumID)
		if err != nil {
			continue
		}
		artist, err := h.Queries.GetArtist(r.Context(), track.ArtistID)
		if err != nil {
			continue
		}
		linkedArtists, err := h.Queries.ListArtistsByTrack(r.Context(), track.ID)
		if err != nil {
			continue
		}
		artistRefs, displayArtist := artistRefsAndDisplay(linkedArtists)

		year := 0
		if album.ReleaseYear.Valid {
			year = int(album.ReleaseYear.Int32)
		}

		songs = append(songs, songEntry{
			ID:            track.ID.String(),
			Title:         track.Title,
			Album:         album.Title,
			AlbumID:       album.ID.String(),
			Artist:        artist.Name,
			ArtistID:      artist.ID.String(),
			CoverArt:      album.ID.String(),
			Track:         int(track.TrackNumber),
			Duration:      int(track.DurationSeconds),
			Suffix:        track.Format,
			ContentType:   contentTypeForFormat(track.Format),
			IsDir:         false,
			Type:          "music",
			Genre:         track.Genre,
			DiscNumber:    int(track.DiscNumber),
			BitDepth:      int(track.BitDepth),
			SamplingRate:  int(track.SampleRate),
			ChannelCount:  int(track.Channels),
			Path:          track.Path,
			Year:          year,
			BitRate:       int(track.BitRate),
			Size:          int(track.SizeBytes),
			Artists:       artistRefs,
			DisplayArtist: displayArtist,
		})
		totalDuration += int(track.DurationSeconds)
	}
	encodeResponse(w, r, playlistWithSongsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Playlist: playlistWithSongs{
			ID:        playlist.ID.String(),
			Name:      playlist.Name,
			Owner:     username,
			Public:    playlist.Public,
			Created:   playlist.CreatedAt.Time,
			Changed:   playlist.UpdatedAt.Time,
			SongCount: len(songs),
			Duration:  totalDuration,
			Entry:     songs,
		},
	})

}

func (h *Handler) DeletePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var playlistId pgtype.UUID
	if err := playlistId.Scan(idParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	playlist, err := h.Queries.GetPlaylist(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "invalid playlist", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("u")

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if playlist.OwnerID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.Queries.DeletePlaylist(r.Context(), playlistId); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	encodeResponse(w, r, newBaseResponse())
}

func (h *Handler) CreatePlaylistHandler(w http.ResponseWriter, r *http.Request) {

	playlistIdParam := r.URL.Query().Get("playlistId")
	nameParam := r.URL.Query().Get("name")
	songIds := r.URL.Query()["songId"]

	username := r.URL.Query().Get("u")

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var playlist sqlc.Playlist

	if playlistIdParam == "" {
		if nameParam == "" {
			http.Error(w, "playlistId or name required", http.StatusBadRequest)
			return
		}
		playlist, err = h.Queries.CreatePlaylist(r.Context(), sqlc.CreatePlaylistParams{
			Name:    nameParam,
			OwnerID: user.ID,
			Public:  false,
		})
		if err != nil {
			http.Error(w, "invalid playlistId", http.StatusBadRequest)
			return
		}
	} else {
		var playlistId pgtype.UUID
		if err := playlistId.Scan(playlistIdParam); err != nil {
			http.Error(w, "invalid playlistId", http.StatusBadRequest)
			return
		}
		playlist, err = h.Queries.GetPlaylist(r.Context(), playlistId)
		if err != nil {
			http.Error(w, "invalid playlist", http.StatusBadRequest)
			return
		}
		if playlist.OwnerID != user.ID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := h.Queries.ClearPlaylistTracks(r.Context(), playlist.ID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	for i, sid := range songIds {
		var trackId pgtype.UUID
		if err := trackId.Scan(sid); err != nil {
			http.Error(w, "invalid songId", http.StatusBadRequest)
			return
		}
		if _, err := h.Queries.AddPlaylistTrack(r.Context(), sqlc.AddPlaylistTrackParams{
			PlaylistID: playlist.ID,
			TrackID:    trackId,
			Position:   int32(i),
		}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	encodeResponse(w, r, playlistWithSongsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Playlist: playlistWithSongs{
			ID:        playlist.ID.String(),
			Name:      playlist.Name,
			Owner:     username,
			Public:    playlist.Public,
			Created:   playlist.CreatedAt.Time,
			Changed:   playlist.UpdatedAt.Time,
			SongCount: len(songIds),
		},
	})

}

func (h *Handler) UpdatePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	playlistIdParam := r.URL.Query().Get("playlistId")
	if playlistIdParam == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var playlistId pgtype.UUID
	if err := playlistId.Scan(playlistIdParam); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	playlist, err := h.Queries.GetPlaylist(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "invalid playlist", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("u")

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if playlist.OwnerID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	nameParam := r.URL.Query().Get("name")
	publicParam := r.URL.Query().Get("public")

	effectiveName := playlist.Name
	if nameParam != "" {
		effectiveName = nameParam
	}
	effectivePublic := playlist.Public
	if publicParam != "" {
		effectivePublic = publicParam == "true"
	}

	playlist, err = h.Queries.UpdatePlaylist(r.Context(), sqlc.UpdatePlaylistParams{
		ID:     playlistId,
		Name:   effectiveName,
		Public: effectivePublic,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	indicesToRemove := r.URL.Query()["songIndexToRemove"]
	positions := make([]int, 0, len(indicesToRemove))
	for _, idxStr := range indicesToRemove {
		pos, err := strconv.Atoi(idxStr)
		if err != nil {
			http.Error(w, "invalid songIndexToRemove", http.StatusBadRequest)
			return
		}
		positions = append(positions, pos)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(positions)))

	for _, pos := range positions {
		if err := h.Queries.RemovePlaylistTrackAtPosition(r.Context(), sqlc.RemovePlaylistTrackAtPositionParams{
			PlaylistID: playlistId,
			Position:   int32(pos),
		}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := h.Queries.ShiftPlaylistTracksAfterPosition(r.Context(), sqlc.ShiftPlaylistTracksAfterPositionParams{
			PlaylistID: playlistId,
			Position:   int32(pos),
		}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	songIdsToAdd := r.URL.Query()["songIdToAdd"]
	count, err := h.Queries.CountPlaylistTracks(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for i, sid := range songIdsToAdd {
		var trackId pgtype.UUID
		if err := trackId.Scan(sid); err != nil {
			http.Error(w, "invalid songIdToAdd", http.StatusBadRequest)
			return
		}
		if _, err := h.Queries.AddPlaylistTrack(r.Context(), sqlc.AddPlaylistTrackParams{
			PlaylistID: playlistId,
			TrackID:    trackId,
			Position:   int32(count) + int32(i),
		}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	encodeResponse(w, r, newBaseResponse())
}
