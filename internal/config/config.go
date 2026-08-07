package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr string
	LogLevel string

	DatabaseURL string
	JWTSecret   string

	LibraryPath  string
	IncomingPath string

	IngestDebounceSeconds int

	TranscodeCachePath string
	TranscodeWorkers   int

	CoverArtDir string

	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string

	LyricsLRCLIBFallback bool
}

func getEnvOrDefault(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvOrDefaultInt(key string, fallback int) (int, error) {
	if v := os.Getenv(key); v != "" {
		valueInt, err := strconv.Atoi(v)
		if err != nil {
			return fallback, err
		}
		return valueInt, nil
	}
	return fallback, nil
}

func getEnvOrDefaultBool(key string, fallback bool) (bool, error) {
	if v := os.Getenv(key); v != "" {
		valueBool, err := strconv.ParseBool(v)
		if err != nil {
			return fallback, err
		}
		return valueBool, nil
	}
	return fallback, nil
}

func requireEnv(key string) (string, error) {
	if v := os.Getenv(key); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("config: required environment variable %s is not set", key)
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           getEnvOrDefault("SONORA_HTTP_ADDR", ":4533"),
		LogLevel:           getEnvOrDefault("SONORA_LOG_LEVEL", "info"),
		LibraryPath:        getEnvOrDefault("SONORA_LIBRARY_PATH", "/music"),
		IncomingPath:       getEnvOrDefault("SONORA_INCOMING_PATH", "/music/incoming"),
		TranscodeCachePath: getEnvOrDefault("SONORA_TRANSCODE_CACHE_PATH", "/cache"),
		CoverArtDir:        getEnvOrDefault("SONORA_COVER_ART_DIR", "/cache/covers"),
		S3Endpoint:         getEnvOrDefault("SONORA_S3_ENDPOINT", ""),
		S3Bucket:           getEnvOrDefault("SONORA_S3_BUCKET", ""),
		S3AccessKey:        getEnvOrDefault("SONORA_S3_ACCESS_KEY", ""),
		S3SecretKey:        getEnvOrDefault("SONORA_S3_SECRET_KEY", ""),
	}

	var err error

	cfg.DatabaseURL, err = requireEnv("SONORA_DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.JWTSecret, err = requireEnv("SONORA_JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	cfg.IngestDebounceSeconds, err = getEnvOrDefaultInt("SONORA_INGEST_DEBOUNCE_SECONDS", 5)
	if err != nil {
		return Config{}, err
	}

	cfg.TranscodeWorkers, err = getEnvOrDefaultInt("SONORA_TRANSCODE_WORKERS", 2)
	if err != nil {
		return Config{}, err
	}

	cfg.LyricsLRCLIBFallback, err = getEnvOrDefaultBool("SONORA_LYRICS_LRCLIB_FALLBACK", true)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
