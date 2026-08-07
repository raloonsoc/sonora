package subsonic

import (
	"encoding/json"
	"net/http"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type Handler struct {
	Queries *sqlc.Queries
}

type pingResponse struct {
	SubsonicResponse subsonicResponse `json:"subsonic-response"`
}

type subsonicResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	resp := pingResponse{
		SubsonicResponse: subsonicResponse{
			Status:  "ok",
			Version: "1.0.0",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
