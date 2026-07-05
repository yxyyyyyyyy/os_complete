package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestRunSpecificRealExperimentNamesWriteRealRuntimeArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run("e1-real-scheduler", 2, outDir); err != nil {
		t.Fatalf("run e1-real-scheduler: %v", err)
	}
	if err := run("e2-real-fault", 2, outDir); err != nil {
		t.Fatalf("run e2-real-fault: %v", err)
	}

	for _, check := range []struct {
		path       string
		experiment string
	}{
		{filepath.Join(outDir, "e1-real-scheduler.json"), "E1_real_scheduler_benchmark"},
		{filepath.Join(outDir, "e2-real-fault.json"), "E2_real_fault_isolation_benchmark"},
	} {
		data, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", check.path, err)
		}
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("decode %s: %v", check.path, err)
		}
		if len(rows) == 0 {
			t.Fatalf("empty rows in %s", check.path)
		}
		if rows[0]["experiment"] != check.experiment || rows[0]["evidence_mode"] != "real-runtime" {
			t.Fatalf("bad real experiment row in %s: %#v", check.path, rows[0])
		}
	}

	for _, name := range []string{"e1-real-scheduler.csv", "e2-real-fault.csv"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(data), "real-runtime") {
			t.Fatalf("%s does not contain real-runtime evidence: %s", name, string(data))
		}
	}
}
