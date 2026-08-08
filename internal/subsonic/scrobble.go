package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func (h *Handler) ScrobbleHandler(w http.ResponseWriter, r *http.Request) {
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

	var playedAt pgtype.Timestamptz

	timeParam := r.URL.Query().Get("time")
	if timeParam != "" {
		ms, err := strconv.ParseInt(timeParam, 10, 64)
		if err != nil {
			http.Error(w, "invalid time", http.StatusBadRequest)
			return
		}
		playedAt = pgtype.Timestamptz{Time: time.UnixMilli(ms), Valid: true}
	}

	var submission bool = true

	submissionParam := r.URL.Query().Get("submission")
	if submissionParam == "false" {
		submission = false
	}

	if !submission {
		encodeResponse(w, r, newBaseResponse())
		return
	}

	if err := h.Queries.ScrobbleTrack(r.Context(), sqlc.ScrobbleTrackParams{
		PlayedAt: playedAt,
		ID:       id,
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	encodeResponse(w, r, newBaseResponse())
}
