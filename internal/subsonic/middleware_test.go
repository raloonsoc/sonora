package subsonic

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raloonsoc/sonora/internal/auth"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

const testJWTSecret = "test-secret-not-for-prod"

// seedUser creates a user the way sonora create-user does: a bcrypt hash for
// the future native API, and a separately AES-256-GCM-encrypted copy of the
// same plaintext password used only to answer the Subsonic MD5(password+salt)
// handshake, which requires the server to recover the plaintext.
func seedUser(t *testing.T, queries *sqlc.Queries, username, password string) sqlc.User {
	t.Helper()

	hashed, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}
	encrypted, err := auth.EncryptReversible(testJWTSecret, password)
	if err != nil {
		t.Fatalf("encrypting password: %v", err)
	}

	user, err := queries.CreateUser(context.Background(), sqlc.CreateUserParams{
		Username:                  username,
		PasswordEncrypted:         hashed,
		PasswordSubsonicEncrypted: encrypted,
	})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	return user
}

func subsonicToken(password, salt string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
}

func TestSubsonicAuthMiddleware_ValidCredentials(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "correct-horse")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := SubsonicAuthMiddleware(queries, testJWTSecret, next)

	salt := "s4lt"
	req := httptest.NewRequest(http.MethodGet, "/rest/ping?u=alice&s="+salt+"&t="+subsonicToken("correct-horse", salt), nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called for valid credentials")
	}
}

func TestSubsonicAuthMiddleware_WrongPassword(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "correct-horse")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	mw := SubsonicAuthMiddleware(queries, testJWTSecret, next)

	salt := "s4lt"
	// Token computed from the wrong password: should fail even though the
	// username exists and the request is otherwise well-formed.
	req := httptest.NewRequest(http.MethodGet, "/rest/ping?u=alice&s="+salt+"&t="+subsonicToken("wrong-password", salt), nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Error("next handler was called despite wrong password")
	}
}

func TestSubsonicAuthMiddleware_UnknownUser(t *testing.T) {
	queries := testQueries(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for an unknown user")
	})
	mw := SubsonicAuthMiddleware(queries, testJWTSecret, next)

	salt := "s4lt"
	req := httptest.NewRequest(http.MethodGet, "/rest/ping?u=ghost&s="+salt+"&t="+subsonicToken("anything", salt), nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestSubsonicAuthMiddleware_MissingParams(t *testing.T) {
	queries := testQueries(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called when auth params are missing")
	})
	mw := SubsonicAuthMiddleware(queries, testJWTSecret, next)

	for name, query := range map[string]string{
		"missing username": "s=salt&t=abc",
		"missing token":     "u=alice&s=salt",
		"missing salt":      "u=alice&t=abc",
		"all missing":       "",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/rest/ping?"+query, nil)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

// A stale/wrong JWTSecret means DecryptReversible fails (GCM authentication
// fails on ciphertext encrypted under a different key), which must surface
// as 500, not as a silent auth bypass or a panic.
func TestSubsonicAuthMiddleware_WrongServerSecretFailsClosed(t *testing.T) {
	queries := testQueries(t)
	seedUser(t, queries, "alice", "correct-horse")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called when the server can't decrypt the stored password")
	})
	mw := SubsonicAuthMiddleware(queries, "a-different-secret-entirely", next)

	salt := "s4lt"
	req := httptest.NewRequest(http.MethodGet, "/rest/ping?u=alice&s="+salt+"&t="+subsonicToken("correct-horse", salt), nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for an OPTIONS preflight")
	})
	mw := CORSMiddleware(next)

	req := httptest.NewRequest(http.MethodOptions, "/rest/ping", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestCORSMiddleware_PassesThroughNonOptions(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := CORSMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/rest/ping", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Error("next handler was not called for a GET request")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}
