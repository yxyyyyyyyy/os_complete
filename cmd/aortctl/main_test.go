package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"aort-r/internal/codebasedag"
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

func TestAortctlResourceIsolationScenarioWritesReviewArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"scenario", "resource-isolation", "--mode", "all", "--runs", "1", "--warmup", "0", "--timeout", "2s", "--out", outDir}); err != nil {
		t.Fatalf("resource-isolation: %v", err)
	}
	var summary struct {
		ScenarioID string                               `json:"scenario_id"`
		PerRun     []map[string]any                     `json:"per_run"`
		Summary    map[string]map[string]map[string]any `json:"summary"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "summary.json"), &summary)
	if summary.ScenarioID != "resource-isolation" || len(summary.PerRun) != 3 {
		t.Fatalf("unexpected scenario summary: %#v", summary)
	}
	for _, mode := range []string{"baseline", "isolation-only", "aort-r"} {
		if _, ok := summary.Summary[mode]; !ok {
			t.Fatalf("missing mode %q: %#v", mode, summary.Summary)
		}
	}
}

func TestAortctlContextSharingScenarioWritesAllRatioArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"scenario", "context-sharing", "--mode", "all", "--runs", "1", "--warmup", "0", "--agents", "3", "--context-size", "256", "--timeout", "2s", "--out", outDir}); err != nil {
		t.Fatalf("context-sharing: %v", err)
	}
	var summary struct {
		ScenarioID string           `json:"scenario_id"`
		PerRun     []map[string]any `json:"per_run"`
		Summary    map[string]any   `json:"summary"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "summary.json"), &summary)
	if summary.ScenarioID != "context-sharing" || len(summary.PerRun) != 15 || len(summary.Summary) != 15 {
		t.Fatalf("unexpected context summary: scenario=%s runs=%d modes=%d", summary.ScenarioID, len(summary.PerRun), len(summary.Summary))
	}
	for _, name := range []string{"comparison.csv", "report.md"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestAortctlContextSharingMatrixSmokeWritesArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"scenario", "context-sharing", "--matrix-smoke", "--timeout", "2s", "--out", outDir}); err != nil {
		t.Fatalf("context-sharing matrix-smoke: %v", err)
	}
	var summary struct {
		ScenarioID string           `json:"scenario_id"`
		PerRun     []map[string]any `json:"per_run"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "summary.json"), &summary)
	if summary.ScenarioID != "context-sharing-matrix" || len(summary.PerRun) != 16 {
		t.Fatalf("unexpected matrix summary: scenario=%s runs=%d", summary.ScenarioID, len(summary.PerRun))
	}
}

func TestAortctlRealAgentDemoMockWritesArtifacts(t *testing.T) {
	outDir := t.TempDir()
	if err := run([]string{"scenario", "real-agent-demo", "--provider", "mock", "--seed", "21", "--timeout", "3s", "--out", outDir}); err != nil {
		t.Fatalf("real-agent-demo: %v", err)
	}
	var summary struct {
		ScenarioID string `json:"scenario_id"`
		Status     string `json:"status"`
		Agents     []any  `json:"agents"`
		LLMCalls   []any  `json:"llm_calls"`
		ToolCalls  []any  `json:"tool_calls"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "summary.json"), &summary)
	if summary.ScenarioID != "real-agent-demo" || summary.Status != "passed" || len(summary.Agents) != 6 || len(summary.LLMCalls) < 1 || len(summary.ToolCalls) < 3 {
		t.Fatalf("unexpected demo summary: %+v", summary)
	}
}

func TestAortctlReviewFinalIndexesScenarioOutputs(t *testing.T) {
	root := t.TempDir()
	dirs := map[string]string{
		"resource-isolation": filepath.Join(root, "resource"),
		"context-sharing":    filepath.Join(root, "context"),
		"real-agent-demo":    filepath.Join(root, "demo"),
	}
	for scenario, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := `{"scenario_id":"` + scenario + `","status":"passed","evidence_mode":"degraded","per_run":[{"success":true}]}`
		if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	legacy := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "FINAL_EVIDENCE_INDEX.json"), []byte(`{"legacy":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "review-final")
	if err := run([]string{"evidence", "review-final", "--resource-dir", dirs["resource-isolation"], "--context-dir", dirs["context-sharing"], "--demo-dir", dirs["real-agent-demo"], "--legacy-final-dir", legacy, "--out", out}); err != nil {
		t.Fatalf("review-final: %v", err)
	}
	var index struct {
		AllRequiredPassed bool `json:"all_required_passed"`
	}
	decodeJSONFile(t, filepath.Join(out, "REVIEW_EVIDENCE_INDEX.json"), &index)
	if !index.AllRequiredPassed {
		t.Fatal("review-final should pass with all scenario summaries")
	}
}

func TestAortctlCodebaseDAGScenarioAndEvidenceStubs(t *testing.T) {
	outDir := t.TempDir()
	workload := t.TempDir()
	goFile := filepath.Join(workload, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"scenario", "codebase-dag",
		"--workload", workload,
		"--out", outDir,
		"--run-id", "cli-test",
		"--preflight-only",
		"--min-physical", "1",
		"--min-nonblank", "1",
		"--git-path", fakeCodebaseDAGGit(t),
	}); err != nil {
		t.Fatalf("codebase-dag scenario: %v", err)
	}
	var summary struct {
		SchemaVersion string `json:"schema_version"`
		RunID         string `json:"run_id"`
		Preflight     struct {
			Passed bool `json:"passed"`
		} `json:"preflight"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "cli-test", "summary.json"), &summary)
	if summary.SchemaVersion != codebasedag.SchemaVersion || summary.RunID != "cli-test" || !summary.Preflight.Passed {
		t.Fatalf("unexpected codebase-dag summary: %#v", summary)
	}

	strictDir := filepath.Join(outDir, "strict")
	if err := os.MkdirAll(strictDir, 0o755); err != nil {
		t.Fatal(err)
	}
	strict := codebasedag.EvidenceSummary{
		SchemaVersion: codebasedag.SchemaVersion,
		RunID:         "strict",
		SourceManifest: codebasedag.SourceManifest{
			SchemaVersion:  codebasedag.SchemaVersion,
			PhysicalLines:  codebasedag.DefaultMinPhysicalLines,
			NonblankLines:  codebasedag.DefaultMinNonblankLines,
			TrackedGoFiles: 1,
			TreeHash:       strings.Repeat("b", 64),
		},
		AllRequiredPassed: true,
		Artifacts:         map[string]string{"summary.json": strings.Repeat("c", 64)},
		Tests:             []codebasedag.TestRecord{{SchemaVersion: codebasedag.SchemaVersion, Name: "go test", ExitCode: 0}},
	}
	for _, node := range []string{"preflight", "planner", "resource-coder", "context-coder", "evidence-coder", "integrate", "tester", "reviewer", "finalizer"} {
		strict.Nodes = append(strict.Nodes, codebasedag.NodeRecord{SchemaVersion: codebasedag.SchemaVersion, NodeID: node, Status: codebasedag.NodeSucceeded})
	}
	strict.Patches = []codebasedag.PatchRecord{
		{NodeID: "resource-coder", SHA256: strings.Repeat("d", 64), ChangedFiles: []string{"internal/review/resource_isolation.go"}, SourceCallID: "call-a"},
		{NodeID: "context-coder", SHA256: strings.Repeat("e", 64), ChangedFiles: []string{"internal/review/context_sharing.go"}, SourceCallID: "call-b"},
		{NodeID: "evidence-coder", SHA256: strings.Repeat("f", 64), ChangedFiles: []string{"internal/review/review_final.go"}, SourceCallID: "call-c"},
	}
	for i := 0; i < 7; i++ {
		strict.Calls = append(strict.Calls, codebasedag.CallRecord{
			SchemaVersion:    codebasedag.SchemaVersion,
			CallID:           "call-" + string(rune('a'+i)),
			Provider:         codebasedag.RequiredDeepSeekProvider,
			RequestedModel:   codebasedag.RequiredDeepSeekModel,
			ActualModel:      codebasedag.RequiredDeepSeekModel,
			EvidenceMode:     "real-api",
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
			OutputSHA256:     strings.Repeat("a", 64),
			Status:           "succeeded",
		})
	}
	data, err := json.Marshal(strict)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(strictDir, "summary.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"evidence", "codebase-dag", "--run", strictDir}); err != nil {
		t.Fatalf("codebase-dag evidence: %v", err)
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
		Skipped   []string `json:"skipped"`
		Missing   []string `json:"missing"`
		Generated []string `json:"generated_files"`
	}
	decodeJSONFile(t, filepath.Join(outDir, "all_experiments_summary.json"), &summary)
	if summary.Experiment != "all" || summary.Runs != 1 {
		t.Fatalf("unexpected all summary header: %#v", summary)
	}
	wantNames := []string{"e1", "e1-pressure", "e2", "e2-pressure-fault", "software-real", "workspace probe", "workspace-rmrf", "tool-workspace", "real-cgroup-smoke", "real-pressure-smoke", "deepseek-real-smoke", "ebpf-smoke", "ipc shm-smoke", "cvm memory-smoke", "real-all"}
	if got := stepNames(summary.Steps); !slices.Equal(got, wantNames) {
		t.Fatalf("step order mismatch\ngot  %v\nwant %v", got, wantNames)
	}
	for _, step := range summary.Steps {
		if step.Command == "" {
			t.Fatalf("step %q missing command", step.Name)
		}
		switch step.Status {
		case "passed", "failed", "degraded", "skipped", "missing":
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
	for _, key := range []string{"timestamp", "git", "system", "generic_competition_verify", "real_only_openEuler", "evidence_mode_summary", "real_only_summary", "generated_files", "missing_files", "known_limits", "ebpf_observer", "ipc_shm", "cvm_memory", "replay", "deepseek_real_smoke", "allowed_degraded"} {
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
	for _, key := range []string{"real_env", "real_cgroup_smoke", "real_pressure_smoke", "deepseek_real_smoke", "workspace_probe", "workspace_rmrf", "tool_workspace", "real_all"} {
		if _, ok := realOnly[key]; !ok {
			t.Fatalf("real-only evidence missing %q: %#v", key, realOnly)
		}
	}
	modeSummary := index["evidence_mode_summary"].(map[string]any)
	for _, key := range []string{"ebpf", "ipc", "cvm", "replay", "llm"} {
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

func fakeCodebaseDAGGit(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "git")
	script := `#!/bin/sh
set -eu
if [ "$1" != "-C" ]; then
  echo "missing -C" >&2
  exit 2
fi
cmd="$3"
case "$cmd" in
  rev-parse)
    printf '%s\n' 0123456789abcdef0123456789abcdef01234567
    ;;
  status)
    ;;
  ls-files)
    printf 'main.go\0'
    ;;
  *)
    echo "unexpected fake git command: $cmd" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
