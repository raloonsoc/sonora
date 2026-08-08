package subsonic

import (
	"net/http"
)

type playlistsSubsonicResponse struct {
	baseResponse
	Playlists playlistsEntry `json:"playlists" xml:"playlists"`
}

type playlistsEntry struct {
	Playlist []playlistItem `json:"playlist" xml:"playlist"`
}

type playlistItem struct {
	ID     string `json:"id" xml:"id,attr"`
	Name   string `json:"name" xml:"name,attr"`
	Owner  string `json:"owner" xml:"owner,attr"`
	Public bool   `json:"public" xml:"public,attr"`
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
			ID:     p.ID.String(),
			Name:   p.Name,
			Owner:  username,
			Public: p.Public,
		})
	}

	encodeResponse(w, r, playlistsSubsonicResponse{
		baseResponse: newBaseResponse(),
		Playlists:    playlistsEntry{Playlist: items},
	})
}
