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

func TestAortctlPressureExperimentCommandsWriteEvidence(t *testing.T) {
	outDir := t.TempDir()
	e1Dir := filepath.Join(outDir, "e1_pressure")
	if err := run([]string{"experiment", "e1-pressure", "--runs", "2", "--out", e1Dir}); err != nil {
		t.Fatalf("e1-pressure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e1Dir, "e1_pressure.json")); err != nil {
		t.Fatalf("e1-pressure evidence missing: %v", err)
	}

	e2Dir := filepath.Join(outDir, "e2_pressure_fault")
	if err := run([]string{"experiment", "e2-pressure-fault", "--runs", "2", "--out", e2Dir}); err != nil {
		t.Fatalf("e2-pressure-fault: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e2Dir, "e2_pressure_fault.json")); err != nil {
		t.Fatalf("e2-pressure-fault evidence missing: %v", err)
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

func TestAortctlRealOnlyCommandsWriteFailureEvidenceForNonCgroupRoot(t *testing.T) {
	outDir := t.TempDir()
	cgroupRoot := filepath.Join(t.TempDir(), "not-cgroup2fs")
	if err := run([]string{"experiment", "real-cgroup-smoke", "--out", outDir, "--cgroup-root", cgroupRoot}); err == nil {
		t.Fatalf("real-cgroup-smoke should reject non-cgroup2fs roots")
	}
	if _, err := os.Stat(filepath.Join(outDir, "real_cgroup_smoke.json")); err != nil {
		t.Fatalf("real-cgroup-smoke failure evidence missing: %v", err)
	}

	pressureDir := filepath.Join(t.TempDir(), "pressure")
	if err := run([]string{"experiment", "real-pressure-smoke", "--runs", "3", "--out", pressureDir, "--require-real", "--cgroup-root", cgroupRoot}); err == nil {
		t.Fatalf("real-pressure-smoke should reject non-cgroup2fs roots")
	}
	if _, err := os.Stat(filepath.Join(pressureDir, "real_pressure_smoke.json")); err != nil {
		t.Fatalf("real-pressure-smoke failure evidence missing: %v", err)
	}
}
