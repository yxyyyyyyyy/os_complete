package experiment

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/evidence"
	"aort-r/internal/trace"
	"aort-r/internal/workspace"
)

func TestRealCgroupSmokeRejectsNonCgroupRootWithoutDegradedPass(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunRealCgroupSmoke(RealCgroupSmokeConfig{
		CgroupRoot: filepath.Join(t.TempDir(), "not-cgroup2fs"),
		OutDir:     outDir,
	})
	if err == nil {
		t.Fatalf("expected non-cgroup root to fail real-cgroup-smoke")
	}
	if result.EvidenceMode == string(evidence.ModeRealCgroupV2) {
		t.Fatalf("non-cgroup root must not be reported as real-cgroup-v2: %#v", result)
	}
	if result.CgroupKillSuccess || result.DestroySuccess {
		t.Fatalf("failed smoke must not report destructive real operations as successful: %#v", result)
	}
	if !strings.Contains(result.FailureReason, "cgroup fs is not cgroup2fs") {
		t.Fatalf("failure reason = %q", result.FailureReason)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "real_cgroup_smoke.json")); statErr != nil {
		t.Fatalf("failure evidence should still be written: %v", statErr)
	}
}

func TestRealPressureSmokeRequireRealRejectsNonCgroupRoot(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunRealPressureSmoke(RealPressureSmokeConfig{
		Runs:        3,
		OutDir:      outDir,
		RequireReal: true,
		CgroupRoot:  filepath.Join(t.TempDir(), "not-cgroup2fs"),
	})
	if err == nil {
		t.Fatalf("expected non-cgroup root to fail real-pressure-smoke")
	}
	if result.EvidenceMode == string(evidence.ModeRealCgroupV2) || result.ResourceSamplerMode == string(evidence.ModeRealCgroupV2) {
		t.Fatalf("non-cgroup root must not be reported as real pressure evidence: %#v", result)
	}
	if result.CleanupSuccess != true {
		t.Fatalf("cleanup should still be attempted and reported: %#v", result)
	}
	if !strings.Contains(result.FailureReason, "cgroup fs is not cgroup2fs") {
		t.Fatalf("failure reason = %q", result.FailureReason)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "real_pressure_smoke.json")); statErr != nil {
		t.Fatalf("failure evidence should still be written: %v", statErr)
	}
}

func TestToolWorkspaceDemoRejectsDegradedWhenRealOverlayRequired(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunToolWorkspaceDemo(workspace.Config{
		Root:          filepath.Join(t.TempDir(), "workspaces"),
		ForceDegraded: true,
	}, outDir, true)
	if err == nil {
		t.Fatalf("expected require-real-overlayfs to reject degraded workspace")
	}
	if result.EvidenceMode == string(evidence.ModeRealOverlayFS) || result.WorkspaceMode == workspace.ModeOverlayFS {
		t.Fatalf("forced degraded workspace must not be reported as real-overlayfs: %#v", result)
	}
	if result.ToolExecUsesWorkspace {
		t.Fatalf("tool execution should not run after real-overlayfs preflight fails: %#v", result)
	}
	if result.FallbackReason == "" {
		t.Fatalf("fallback reason missing: %#v", result)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "tool_workspace_evidence.json")); statErr != nil {
		t.Fatalf("failure evidence should still be written: %v", statErr)
	}
}

func TestBuildRealAllIndexFailsClosedForMissingRealEvidence(t *testing.T) {
	index := BuildRealAllIndex(RealAllInputs{
		EnvOpenEuler:                    true,
		EnvCgroup2FS:                    true,
		EnvOverlayFSMount:               true,
		CgroupSmoke:                     RealCgroupSmokeResult{EvidenceMode: string(evidence.ModeDegraded), CgroupKillSuccess: false},
		PressureSmoke:                   RealPressureSmokeResult{EvidenceMode: string(evidence.ModeRealCgroupV2), ResourceSamplerMode: string(evidence.ModeRealCgroupV2), ResourceAwareAvoidedHighPressureAgent: true, SelectedHighPressureAgentCount: 0, CascadeFailure: false, CleanupSuccess: true},
		WorkspaceProbeRealOverlay:       true,
		WorkspaceRMFault:                workspace.RMFaultEvidence{EvidenceMode: evidence.ModeRealOverlayFS, CascadeFailure: false},
		ToolWorkspace:                   ToolWorkspaceEvidence{EvidenceMode: string(evidence.ModeRealOverlayFS), ToolExecUsesWorkspace: true},
		E2PressureFaultCascadeFailure:   false,
		SoftwareRealRuntimeSuccess:      true,
		SoftwareRealRuntimeEvidenceMode: string(evidence.ModeRealRuntime),
	})
	if index.AllPassed {
		t.Fatalf("real-all must fail when cgroup smoke is not real: %#v", index)
	}
	if index.RealCgroupV2 || index.CgroupKillSuccess {
		t.Fatalf("cgroup booleans should fail closed: %#v", index)
	}
	if len(index.FailureReasons) == 0 {
		t.Fatalf("failure reasons missing: %#v", index)
	}
}

func TestRunSoftwareRealDemoWritesCliArtifactPath(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunSoftwareRealDemo(2, outDir)
	if err != nil {
		t.Fatalf("RunSoftwareRealDemo: %v", err)
	}
	if result.EvidenceMode != string(evidence.ModeRealRuntime) || !result.FinalSuccess {
		t.Fatalf("software-real result is not real-runtime success: %#v", result)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "software_real_demo", "result.json")); statErr != nil {
		t.Fatalf("software-real result artifact missing: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "software_real_demo", "trace.json")); statErr != nil {
		t.Fatalf("software-real trace artifact missing: %v", statErr)
	}
	rawTrace, err := os.ReadFile(filepath.Join(outDir, "software_real_demo", "trace.json"))
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var events []trace.TraceEvent
	if err := json.Unmarshal(rawTrace, &events); err != nil {
		t.Fatalf("decode trace: %v", err)
	}
	foundRuntimeSyscall := false
	for _, event := range events {
		if event.Type == "syscall.finished" && strings.HasPrefix(event.EventID, "e5-") {
			foundRuntimeSyscall = true
		}
		if strings.HasPrefix(event.EventID, "software-real-") {
			t.Fatalf("trace should come from runtime events, not manual software-real sequence: %#v", event)
		}
	}
	if !foundRuntimeSyscall {
		t.Fatalf("trace missing runtime syscall events: %#v", events)
	}
}

func TestRunDeepSeekRealSmokeSkipsWhenNotEnabled(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunDeepSeekRealSmoke(DeepSeekRealSmokeConfig{
		OutDir: outDir,
		Enable: false,
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("RunDeepSeekRealSmoke disabled: %v", err)
	}
	if result.Status != "skipped" || result.EvidenceMode != string(evidence.ModeMissing) || result.LLMMock {
		t.Fatalf("disabled smoke should be skipped without mock pass: %#v", result)
	}
	if result.APIKeyPresent || !result.APIKeyRedacted {
		t.Fatalf("disabled smoke should not report key use: %#v", result)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "deepseek_real_smoke.json")); statErr != nil {
		t.Fatalf("deepseek skipped evidence missing: %v", statErr)
	}
}

func TestRunDeepSeekRealSmokeRequiresAPIKeyWhenEnabled(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunDeepSeekRealSmoke(DeepSeekRealSmokeConfig{
		OutDir:  outDir,
		Enable:  true,
		BaseURL: "https://api.deepseek.test",
	})
	if err == nil {
		t.Fatalf("expected enabled DeepSeek smoke without API key to fail")
	}
	if result.Status != "failed" || result.RequestSuccess || result.EvidenceMode == string(evidence.ModeRealAPI) {
		t.Fatalf("missing-key smoke must not pass or become mock: %#v", result)
	}
	if result.APIKeyPresent || !result.APIKeyRedacted || result.APIKeySource != "env" {
		t.Fatalf("key metadata should be redacted env-only: %#v", result)
	}
}

func TestRunDeepSeekRealSmokePassesWithRealAPIShape(t *testing.T) {
	outDir := t.TempDir()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header was not set correctly")
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}],"usage":{"total_tokens":3}}`)),
			Request:    r,
		}, nil
	})}

	result, err := RunDeepSeekRealSmoke(DeepSeekRealSmokeConfig{
		OutDir:  outDir,
		Enable:  true,
		APIKey:  "test-key",
		BaseURL: "https://api.deepseek.test",
		Model:   "deepseek-v4-flash",
		Client:  client,
	})
	if err != nil {
		t.Fatalf("RunDeepSeekRealSmoke enabled: %v", err)
	}
	if result.Status != "passed" || result.EvidenceMode != string(evidence.ModeRealAPI) || result.Provider != "deepseek" {
		t.Fatalf("real smoke should pass as DeepSeek real-api: %#v", result)
	}
	if result.LLMMock || !result.RequestSuccess || !result.ResponseNonEmpty || result.StatusCode != 200 || result.LatencyMS <= 0 {
		t.Fatalf("real smoke response fields missing: %#v", result)
	}
	if !result.APIKeyPresent || !result.APIKeyRedacted || result.APIKeySource != "env" || !result.CleanupSuccess {
		t.Fatalf("secret/cleanup evidence wrong: %#v", result)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "deepseek_real_smoke.json"))
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	if strings.Contains(string(raw), "test-key") {
		t.Fatalf("evidence leaked API key: %s", raw)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestStatusFromDeepSeekRealSmokeHonorsRequiredMode(t *testing.T) {
	outDir := t.TempDir()
	path := filepath.Join(outDir, "deepseek_real_smoke.json")
	skipped := DeepSeekRealSmokeResult{
		Experiment:     "deepseek_real_smoke",
		Status:         "skipped",
		EvidenceMode:   string(evidence.ModeMissing),
		Provider:       "deepseek",
		LLMMock:        false,
		APIKeySource:   "env",
		APIKeyRedacted: true,
		CleanupSuccess: true,
		FailureReason:  "AORT_ENABLE_REAL_LLM is not set to 1",
	}
	if err := WriteJSON(path, skipped); err != nil {
		t.Fatalf("write skipped deepseek smoke: %v", err)
	}
	if got := statusFromDeepSeekRealSmoke(path, false); got != "skipped" {
		t.Fatalf("optional skipped deepseek smoke status = %q", got)
	}
	if got := statusFromDeepSeekRealSmoke(path, true); got != "failed" {
		t.Fatalf("required skipped deepseek smoke status = %q", got)
	}

	passed := DeepSeekRealSmokeResult{
		Experiment:       "deepseek_real_smoke",
		Status:           "passed",
		EvidenceMode:     string(evidence.ModeRealAPI),
		Provider:         "deepseek",
		Model:            "deepseek-v4-flash",
		LLMMock:          false,
		RequestSuccess:   true,
		ResponseNonEmpty: true,
		StatusCode:       200,
		LatencyMS:        5,
		APIKeySource:     "env",
		APIKeyPresent:    true,
		APIKeyRedacted:   true,
		CleanupSuccess:   true,
	}
	if err := WriteJSON(path, passed); err != nil {
		t.Fatalf("write passed deepseek smoke: %v", err)
	}
	if got := statusFromDeepSeekRealSmoke(path, true); got != "passed" {
		t.Fatalf("required passed deepseek smoke status = %q", got)
	}
}

func TestAllRealOnlyRequiredPassedReconcilesIndividualStatuses(t *testing.T) {
	realOnly := map[string]string{
		"real_env":            "passed",
		"real_cgroup_smoke":   "passed",
		"real_pressure_smoke": "failed",
		"deepseek_real_smoke": "skipped",
		"workspace_probe":     "passed",
		"workspace_rmrf":      "passed",
		"tool_workspace":      "passed",
		"real_all":            "passed",
	}
	if allRealOnlyRequiredPassed(realOnly, true, false) {
		t.Fatalf("required real-only summary must fail when real_pressure_smoke failed")
	}
	realOnly["real_pressure_smoke"] = "passed"
	if !allRealOnlyRequiredPassed(realOnly, true, false) {
		t.Fatalf("optional skipped deepseek smoke should not fail default real-only summary")
	}
	if allRealOnlyRequiredPassed(realOnly, true, true) {
		t.Fatalf("required deepseek smoke must fail when it is skipped")
	}
	realOnly["deepseek_real_smoke"] = "passed"
	if !allRealOnlyRequiredPassed(realOnly, true, true) {
		t.Fatalf("required deepseek smoke should pass when all required statuses passed")
	}
}

func TestAllowedDegradedEvidenceReportsEBPFReason(t *testing.T) {
	root := t.TempDir()
	ebpfDir := filepath.Join(root, "experiments", "results", "ebpf_smoke")
	if err := WriteJSON(filepath.Join(ebpfDir, "ebpf_smoke.json"), map[string]any{
		"observer":        "ebpf",
		"evidence_mode":   string(evidence.ModeDegraded),
		"program_loaded":  true,
		"cleanup_success": true,
		"fallback_reason": "attach failed: invalid argument",
	}); err != nil {
		t.Fatalf("write ebpf evidence: %v", err)
	}
	allowed := allowedDegradedEvidence(root, map[string]string{"ebpf_observer": "degraded"})
	if allowed["ebpf_observer"] != "attach failed: invalid argument" {
		t.Fatalf("allowed degraded eBPF reason = %#v", allowed)
	}
}

func TestGitDirtyFromPorcelainIgnoresUntrackedFiles(t *testing.T) {
	if gitDirtyFromPorcelain("?? scratch.txt\n?? experiments/results/audit_all/summary.json\n") {
		t.Fatalf("untracked files should not mark final evidence dirty")
	}
	if !gitDirtyFromPorcelain(" M README.md\n") {
		t.Fatalf("tracked modifications should mark final evidence dirty")
	}
	if !gitDirtyFromPorcelain("M  internal/cvm/store.go\n") {
		t.Fatalf("staged tracked modifications should mark final evidence dirty")
	}
}

func TestGitDirtyFromPorcelainIgnoresGeneratedEvidence(t *testing.T) {
	status := " M experiments/results/final/FINAL_EVIDENCE_INDEX.json\n M experiments/results/replay/replay_result.json\n"
	if gitDirtyFromPorcelain(status) {
		t.Fatalf("generated evidence files should not mark final evidence dirty")
	}
	if !gitDirtyFromPorcelain(status + " M internal/syscall/gateway.go\n") {
		t.Fatalf("source changes should still mark final evidence dirty")
	}
}
