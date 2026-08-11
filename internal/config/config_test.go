package config

import "testing"

func TestLoad_MissingRequiredVar(t *testing.T) {
	t.Setenv("SONORA_DATABASE_URL", "postgres://sonora:sonora@postgres:5432/sonora?sslmode=disable")
	t.Setenv("SONORA_JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when SONORA_JWT_SECRET is missing, got nil")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("SONORA_DATABASE_URL", "postgres://sonora:sonora@postgres:5432/sonora?sslmode=disable")
	t.Setenv("SONORA_JWT_SECRET", "some-secret")

	t.Setenv("SONORA_HTTP_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":4533" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":4533")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

// LogLevel is parsed with slog.Level.UnmarshalText in main.go, not here —
// Load() only has to hand back whatever string was configured, valid or
// not, so an invalid value can be caught and defaulted at startup instead
// of failing config loading outright.
func TestLoad_LogLevelPassesThroughRawValue(t *testing.T) {
	t.Setenv("SONORA_DATABASE_URL", "postgres://sonora:sonora@postgres:5432/sonora?sslmode=disable")
	t.Setenv("SONORA_JWT_SECRET", "some-secret")
	t.Setenv("SONORA_LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}
