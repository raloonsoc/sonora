package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/raloonsoc/sonora/internal/auth"
	"github.com/raloonsoc/sonora/internal/config"
	"github.com/raloonsoc/sonora/internal/db"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
	"github.com/raloonsoc/sonora/internal/ingest"
	"github.com/raloonsoc/sonora/internal/streaming"
	"github.com/raloonsoc/sonora/internal/subsonic"
)

func main() {

	if len(os.Args) > 1 && os.Args[1] == "create-user" {
		createUserCmd(os.Args[2:])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.RunMigration(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	queries := sqlc.New(pool)
	handler := &streaming.Handler{Queries: queries, TranscodeCacheDir: filepath.Join(cfg.TranscodeCachePath, "transcodes"), TranscodeSem: make(chan struct{}, cfg.TranscodeWorkers)}

	mux := subsonic.NewRouter(&subsonic.Handler{Queries: queries, LyricsLRCLIBFallback: cfg.LyricsLRCLIBFallback}, handler)

	processed := make(chan string)
	go func() {
		for path := range processed {
			if err := ingest.ProcessFile(context.Background(), path, queries, cfg.CoverArtDir); err != nil {
				slog.Error("ingest: processing file failed", "path", path, "error", err)
			}
		}
	}()

	interval := time.Duration(cfg.IngestPollIntervalSeconds) * time.Second
	go func() {
		if err := ingest.WatchLibrary(context.Background(), cfg.LibraryPath, interval, queries, processed); err != nil {
			slog.Error("ingest: watcher failed", "error", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := streaming.CleanupExpiredCache(handler.TranscodeCacheDir, 30*24*time.Hour); err != nil {
				slog.Error("streaming: cache cleanup failed", "error", err)
			}
		}
	}()

	slog.Info("starting sonora", "http_addr", cfg.HTTPAddr)
	protectedMux := subsonic.SubsonicAuthMiddleware(queries, cfg.JWTSecret, mux)
	corsProtectedMux := subsonic.CORSMiddleware(protectedMux)
	loggedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		corsProtectedMux.ServeHTTP(w, r)
	})
	if err := http.ListenAndServe(cfg.HTTPAddr, loggedMux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func createUserCmd(args []string) {
	fs := flag.NewFlagSet("create-user", flag.ExitOnError)
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "password")
	admin := fs.Bool("admin", false, "make this user admin")
	_ = fs.Parse(args) // ExitOnError already terminates the process on failure

	if *username == "" || *password == "" {
		slog.Error("create-user: --username and --password are required")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	hashed, err := auth.HashPassword(*password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	encrypted, err := auth.EncryptReversible(cfg.JWTSecret, *password)
	if err != nil {
		slog.Error("failed to encrypt password", "error", err)
		os.Exit(1)
	}

	user, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:                  *username,
		PasswordEncrypted:         hashed,
		PasswordSubsonicEncrypted: encrypted,
		IsAdmin:                   *admin,
	})
	if err != nil {
		slog.Error("failed to create user", "error", err)
		os.Exit(1)
	}
	slog.Info("user created", "username", user.Username, "id", user.ID.String())
}
