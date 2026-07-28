package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DB_PATH", "test.duckdb")
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("ENVIRONMENT")
	}()

	cfg := LoadConfig()
	if cfg.Port != "9090" {
		t.Errorf("expected Port 9090, got %s", cfg.Port)
	}
	if cfg.DBPath != "test.duckdb" {
		t.Errorf("expected DBPath test.duckdb, got %s", cfg.DBPath)
	}
	if cfg.Environment != "test" {
		t.Errorf("expected Environment test, got %s", cfg.Environment)
	}
}
