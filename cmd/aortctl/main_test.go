package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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
	var workspaceEvidence struct {
		RuntimeRoot string `json:"runtime_root"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "workspace_isolation_evidence.json"), &workspaceEvidence)
	if !strings.HasPrefix(workspaceEvidence.RuntimeRoot, filepath.Join(outDir, "workspace_rmrf_runtime")) {
		t.Fatalf("workspace fault should use outDir-scoped runtime root, got %q", workspaceEvidence.RuntimeRoot)
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

func TestAortctlUpgradeSmokeCommandsWriteEvidence(t *testing.T) {
	outDir := t.TempDir()
	ebpfDir := filepath.Join(outDir, "ebpf")
	if err := run([]string{"observer", "ebpf-smoke", "--out", ebpfDir}); err != nil {
		t.Fatalf("ebpf-smoke: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ebpfDir, "ebpf_smoke.json")); err != nil {
		t.Fatalf("ebpf evidence missing: %v", err)
	}

	shmDir := filepath.Join(outDir, "ipc_shm")
	if err := run([]string{"ipc", "shm-smoke", "--out", shmDir}); err != nil {
		t.Fatalf("shm-smoke: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shmDir, "ipc_shm_smoke.json")); err != nil {
		t.Fatalf("shm evidence missing: %v", err)
	}

	cvmDir := filepath.Join(outDir, "cvm_memory")
	if err := run([]string{"cvm", "memory-smoke", "--out", cvmDir}); err != nil {
		t.Fatalf("cvm memory-smoke: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cvmDir, "cvm_memory_smoke.json")); err != nil {
		t.Fatalf("cvm evidence missing: %v", err)
	}

	tracePath := filepath.Join(outDir, "trace.json")
	rawTrace := `[{"event_id":"e1","timestamp":"2026-07-07T00:00:00Z","type":"scheduler_decision","agent_id":"agent-1","task_id":"task-1","payload":{"status":"running"}},{"event_id":"e2","timestamp":"2026-07-07T00:00:01Z","type":"task_completed","agent_id":"agent-1","task_id":"task-1","payload":{"final_status":"completed"}}]`
	if err := os.WriteFile(tracePath, []byte(rawTrace), 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}
	replayDir := filepath.Join(outDir, "replay")
	if err := run([]string{"replay", "--trace", tracePath, "--out", replayDir}); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if _, err := os.Stat(filepath.Join(replayDir, "replay_result.json")); err != nil {
		t.Fatalf("replay evidence missing: %v", err)
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

func TestAortctlExperimentAllWritesStepSummary(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"experiment", "all", "--runs", "1", "--out", outDir}); err != nil {
		t.Fatalf("experiment all: %v", err)
	}
	summaryPath := filepath.Join(outDir, "all_experiments_summary.json")
	rawSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read all summary: %v", err)
	}
	for _, want := range []string{`"failed": []`, `"missing": []`} {
		if !strings.Contains(string(rawSummary), want) {
			t.Fatalf("all summary should encode empty lists as [] and include %s:\n%s", want, rawSummary)
		}
	}

	var summary struct {
		Experiment        string `json:"experiment"`
		Runs              int    `json:"runs"`
		EvidenceMode      string `json:"evidence_mode"`
		AllRequiredPassed bool   `json:"all_required_passed"`
		Steps             []struct {
			Name           string   `json:"name"`
			Command        string   `json:"command"`
			Status         string   `json:"status"`
			GeneratedFiles []string `json:"generated_files"`
			Error          string   `json:"error"`
			FallbackReason string   `json:"fallback_reason,omitempty"`
		} `json:"steps"`
		Passed    []string `json:"passed"`
		Failed    []string `json:"failed"`
		Degraded  []string `json:"degraded"`
		Missing   []string `json:"missing"`
		Generated []string `json:"generated_files"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "all_experiments_summary.json"), &summary)
	if summary.Experiment != "all" || summary.Runs != 1 {
		t.Fatalf("unexpected all summary header: %#v", summary)
	}
	wantNames := []string{"e1", "e1-pressure", "e2", "e2-pressure-fault", "software-real", "workspace probe", "workspace-rmrf", "tool-workspace", "real-cgroup-smoke", "real-pressure-smoke", "ebpf-smoke", "ipc shm-smoke", "cvm memory-smoke", "real-all"}
	if got := stepNames(summary.Steps); !slices.Equal(got, wantNames) {
		t.Fatalf("step order mismatch\ngot  %v\nwant %v", got, wantNames)
	}
	for _, step := range summary.Steps {
		if step.Command == "" {
			t.Fatalf("step %q missing command", step.Name)
		}
		switch step.Status {
		case "passed", "failed", "degraded", "missing":
		default:
			t.Fatalf("step %q has invalid status %q", step.Name, step.Status)
		}
		if step.Status == "degraded" && step.FallbackReason == "" {
			t.Fatalf("degraded step %q missing fallback_reason: %#v", step.Name, step)
		}
		if step.Status == "degraded" && step.FallbackReason == "degraded" {
			t.Fatalf("degraded step %q should have a clear fallback_reason: %#v", step.Name, step)
		}
	}
	if runtime.GOOS != "linux" && containsName(summary.Passed, "real-cgroup-smoke") {
		t.Fatalf("real-cgroup-smoke must not be passed on non-linux hosts: %#v", summary)
	}
	if len(summary.Generated) == 0 {
		t.Fatalf("summary should list generated files: %#v", summary)
	}
	if !summary.AllRequiredPassed && len(summary.Failed) == 0 && len(summary.Missing) == 0 {
		t.Fatalf("all_required_passed should allow degraded-only runs: %#v", summary)
	}
}

func TestAortctlEvidenceFinalWritesIndexAndSummary(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"evidence", "final", "--out", outDir}); err != nil {
		t.Fatalf("evidence final: %v", err)
	}

	var index map[string]any
	decodeJSONFile(t, filepath.Join(outDir, "FINAL_EVIDENCE_INDEX.json"), &index)
	for _, key := range []string{"timestamp", "git", "system", "generic_competition_verify", "real_only_openEuler", "evidence_mode_summary", "real_only_summary", "generated_files", "missing_files", "known_limits", "ebpf_observer", "ipc_shm", "cvm_memory", "replay"} {
		if _, ok := index[key]; !ok {
			t.Fatalf("final evidence index missing %q: %#v", key, index)
		}
	}
	generic := index["generic_competition_verify"].(map[string]any)
	for _, key := range []string{"go_test", "smoke", "e1_scheduler", "e1_pressure", "e2_fault_isolation", "e2_pressure_fault", "software_real_demo", "workspace_probe", "workspace_isolation", "ebpf_observer", "ipc_shm", "cvm_memory", "replay"} {
		if _, ok := generic[key]; !ok {
			t.Fatalf("generic evidence missing %q: %#v", key, generic)
		}
	}
	realOnly := index["real_only_openEuler"].(map[string]any)
	for _, key := range []string{"real_env", "real_cgroup_smoke", "real_pressure_smoke", "workspace_probe", "workspace_rmrf", "tool_workspace", "real_all"} {
		if _, ok := realOnly[key]; !ok {
			t.Fatalf("real-only evidence missing %q: %#v", key, realOnly)
		}
	}
	modeSummary := index["evidence_mode_summary"].(map[string]any)
	for _, key := range []string{"ebpf", "ipc", "cvm", "replay"} {
		if _, ok := modeSummary[key]; !ok {
			t.Fatalf("mode summary missing %q: %#v", key, modeSummary)
		}
	}
	knownLimits, ok := index["known_limits"].([]any)
	if !ok {
		t.Fatalf("known_limits has unexpected type: %#v", index["known_limits"])
	}
	foundPortablePressureLimit := false
	for _, item := range knownLimits {
		if strings.Contains(item.(string), "Portable E1 benchmark may use degraded pressure fallback") &&
			strings.Contains(item.(string), "real-pressure-smoke proves real-cgroup-v2 ResourceSampler on openEuler") {
			foundPortablePressureLimit = true
		}
	}
	if !foundPortablePressureLimit {
		t.Fatalf("known_limits does not distinguish portable E1 degraded pressure from real-pressure-smoke: %#v", knownLimits)
	}
	summary, err := os.ReadFile(filepath.Join(outDir, "FINAL_SUMMARY.md"))
	if err != nil {
		t.Fatalf("read FINAL_SUMMARY.md: %v", err)
	}
	for _, want := range []string{"generic evidence", "real-only openEuler evidence", "evidence_mode_summary", "known_limits", "Git commit", "fresh clone"} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("FINAL_SUMMARY.md missing %q:\n%s", want, summary)
		}
	}
}

func decodeJSONFile(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode %s: %v\n%s", path, err, data)
	}
}

func stepNames(steps []struct {
	Name           string   `json:"name"`
	Command        string   `json:"command"`
	Status         string   `json:"status"`
	GeneratedFiles []string `json:"generated_files"`
	Error          string   `json:"error"`
	FallbackReason string   `json:"fallback_reason,omitempty"`
}) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Name)
	}
	return names
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
