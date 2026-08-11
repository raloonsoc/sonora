package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func TestGetUserHandler_ReflectsAdminFlag(t *testing.T) {
	queries := testQueries(t)
	if _, err := queries.CreateUser(t.Context(), sqlc.CreateUserParams{
		Username: "admin", PasswordEncrypted: []byte("x"), PasswordSubsonicEncrypted: []byte("y"), IsAdmin: true,
	}); err != nil {
		t.Fatalf("seeding admin user: %v", err)
	}
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getUser?username=admin&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetUserHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			User struct {
				Username  string `json:"username"`
				AdminRole bool   `json:"adminRole"`
			} `json:"user"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if !payload.SubsonicResponse.User.AdminRole {
		t.Error("adminRole = false, want true for an admin user")
	}
}

func TestGetUserHandler_NonAdmin(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "pw")
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getUser?username=alice&f=json", nil)
	rec := httptest.NewRecorder()
	h.GetUserHandler(rec, req)

	var payload struct {
		SubsonicResponse struct {
			User struct {
				AdminRole bool `json:"adminRole"`
			} `json:"user"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if payload.SubsonicResponse.User.AdminRole {
		t.Error("adminRole = true, want false for a non-admin user")
	}
}

func TestGetUserHandler_UnknownUser(t *testing.T) {
	queries := testQueries(t)
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getUser?username=ghost", nil)
	rec := httptest.NewRecorder()
	h.GetUserHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetUserHandler_MissingUsername(t *testing.T) {
	queries := testQueries(t)
	h := &Handler{Queries: queries}

	req := httptest.NewRequest(http.MethodGet, "/rest/getUser", nil)
	rec := httptest.NewRecorder()
	h.GetUserHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetLicenseHandler_AlwaysValid(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/rest/getLicense?f=json", nil)
	rec := httptest.NewRecorder()
	h.GetLicenseHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			License struct {
				Valid bool `json:"valid"`
			} `json:"license"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if !payload.SubsonicResponse.License.Valid {
		t.Error("license.valid = false, want true")
	}
}

func TestGetOpenSubsonicExtensionsHandler(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/rest/getOpenSubsonicExtensions?f=json", nil)
	rec := httptest.NewRecorder()
	h.GetOpenSubsonicExtensionsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetMusicFoldersHandler(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/rest/getMusicFolders?f=json", nil)
	rec := httptest.NewRecorder()
	h.GetMusicFoldersHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPingHandler(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/rest/ping?f=json", nil)
	rec := httptest.NewRecorder()
	h.PingHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		SubsonicResponse struct {
			Status string `json:"status"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	if payload.SubsonicResponse.Status != "ok" {
		t.Errorf("status = %q, want %q", payload.SubsonicResponse.Status, "ok")
	}
}
