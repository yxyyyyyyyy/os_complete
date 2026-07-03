package config

import "testing"

func TestLoadDevConfig(t *testing.T) {
	cfg, err := Load("../../configs/dev.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.Mode != "mock" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if cfg.DataDir != ".aort-dev" {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
}
