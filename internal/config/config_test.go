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

func TestLoadOpenEulerConfig(t *testing.T) {
	cfg, err := Load("../../configs/openeuler-dev.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "openeuler" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if cfg.DataDir != "/var/lib/aort" {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
	if cfg.SocketPath != "/run/aort/aortd.sock" {
		t.Fatalf("SocketPath = %q", cfg.SocketPath)
	}
	if cfg.WorkerCommand != "/usr/local/bin/aort-worker" {
		t.Fatalf("WorkerCommand = %q", cfg.WorkerCommand)
	}
	if cfg.CgroupRoot != "/sys/fs/cgroup/aort.slice" {
		t.Fatalf("CgroupRoot = %q", cfg.CgroupRoot)
	}
}
