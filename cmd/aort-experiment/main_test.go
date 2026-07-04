package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunAllWritesRequiredRealExperimentArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run("all", 3, outDir); err != nil {
		t.Fatalf("run all: %v", err)
	}
	for _, name := range []string{
		"e1-real-scheduler.json",
		"e1-real-scheduler.csv",
		"e2-real-fault.json",
		"e2-real-fault.csv",
		"e3-real-context.json",
		"e3-real-context.csv",
		"e4-real-ipc.json",
		"e4-real-ipc.csv",
		"e5-end-to-end.json",
		"e5-end-to-end.csv",
	} {
		if info, err := os.Stat(filepath.Join(outDir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing or empty artifact %s info=%#v err=%v", name, info, err)
		}
	}
}
