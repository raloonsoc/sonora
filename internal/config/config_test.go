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
}
