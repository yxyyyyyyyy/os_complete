package experiment

import (
	"encoding/json"
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
