package subsonic

import (
	"net/http"

	"github.com/raloonsoc/sonora/internal/auth"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func SubsonicAuthMiddleware(queries *sqlc.Queries, jwtSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("u")
		token := r.URL.Query().Get("t")
		salt := r.URL.Query().Get("s")

		if username == "" || token == "" || salt == "" {
			http.Error(w, "missing auth params", http.StatusUnauthorized)
			return
		}

		user, err := queries.GetUserByUsername(r.Context(), username)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		plainPassword, err := auth.DecryptReversible(jwtSecret, user.PasswordSubsonicEncrypted)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if !auth.VerifySubsonicToken(plainPassword, salt, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
