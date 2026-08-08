package subsonic

import (
	"net/http"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

type Handler struct {
	Queries *sqlc.Queries
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	encodeResponse(w, r, newBaseResponse())
}
