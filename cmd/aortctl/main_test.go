package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAortctlResourceAwareExperimentAndWorkspaceFaultCommands(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"experiment", "e1", "--policy", "resource-aware", "--runs", "2", "--out", outDir}); err != nil {
		t.Fatalf("resource-aware e1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "e1_resource_aware.json")); err != nil {
		t.Fatalf("e1 resource-aware artifact missing: %v", err)
	}

	if err := run([]string{"demo", "fault", "workspace-rmrf", "--out", outDir}); err != nil {
		t.Fatalf("workspace fault: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "workspace_isolation_evidence.json")); err != nil {
		t.Fatalf("workspace evidence missing: %v", err)
	}
}

func TestAortctlWorkspaceProbeCommandWritesEvidence(t *testing.T) {
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "workspace_probe.json")
	if err := run([]string{"workspace", "probe", "--out", outFile, "--root", filepath.Join(outDir, "probe-root")}); err != nil {
		t.Fatalf("workspace probe: %v", err)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("workspace probe evidence missing: %v", err)
	}
}

func TestAortctlSoftwareRealDemoCommandWritesResult(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"demo", "software-real", "--out", outDir}); err != nil {
		t.Fatalf("software-real: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "software_real_demo", "result.json")); err != nil {
		t.Fatalf("software-real result missing: %v", err)
	}
}
