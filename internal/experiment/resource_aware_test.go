package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/evidence"
	"aort-r/internal/scheduler"
)

func TestRunE1ResourceAwareWritesRequiredArtifactsAndSchema(t *testing.T) {
	outDir := t.TempDir()
	results, err := RunE1ResourceAwareBenchmark(3, outDir)
	if err != nil {
		t.Fatalf("RunE1ResourceAwareBenchmark: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected four scheduler policies, got %#v", results)
	}
	seen := map[string]bool{}
	for _, result := range results {
		seen[result.Policy] = true
		if result.EvidenceMode == "" || !evidence.IsValid(evidence.Mode(result.EvidenceMode)) {
			t.Fatalf("invalid evidence mode: %#v", result)
		}
		if result.SchedulerDecisionsCount == 0 {
			t.Fatalf("missing decision count: %#v", result)
		}
	}
	if !seen[scheduler.PolicyTokenCFSPrefixAffinityResourceAware] {
		t.Fatalf("resource-aware policy missing: %#v", results)
	}

	for _, name := range []string{"e1_resource_aware.json", "e1_resource_aware.csv", "e1_resource_aware_summary.md"} {
		if info, err := os.Stat(filepath.Join(outDir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing artifact %s info=%#v err=%v", name, info, err)
		}
	}
	var report E1ResourceAwareReport
	data, err := os.ReadFile(filepath.Join(outDir, "e1_resource_aware.json"))
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if report.Experiment != "e1_resource_aware_scheduler" || report.Runs != 3 {
		t.Fatalf("bad report identity: %#v", report)
	}
	if len(report.Policies) != 4 || !containsPolicy(report.Policies, scheduler.PolicyTokenCFSPrefixAffinityResourceAware) {
		t.Fatalf("report policies incomplete: %#v", report.Policies)
	}
	if report.Metrics.MemoryPeakBytes[scheduler.PolicyFIFO] == 0 || report.Metrics.PidsPeak[scheduler.PolicyFIFO] == 0 {
		t.Fatalf("resource metrics missing: %#v", report.Metrics)
	}
	if report.Metrics.SchedulerDecisionsCount[scheduler.PolicyTokenCFSPrefixAffinityResourceAware] == 0 {
		t.Fatalf("decision metrics missing: %#v", report.Metrics)
	}
	if report.Improvement.ResourceAwareVsFIFOPercent < -1000 || report.Improvement.ResourceAwareVsFIFOPercent > 1000 {
		t.Fatalf("resource-aware improvement should be numeric: %#v", report.Improvement)
	}
	if report.EvidenceMode == "" || !evidence.IsValid(evidence.Mode(report.EvidenceMode)) {
		t.Fatalf("invalid report evidence mode: %#v", report)
	}
	const pressureFallback = "resource pressure sampler not configured or local cgroup pressure files unavailable"
	for _, result := range report.PolicyResults {
		if result.Policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware && result.FallbackReason != pressureFallback {
			t.Fatalf("resource-aware fallback_reason = %q, want %q", result.FallbackReason, pressureFallback)
		}
	}
}

func TestRunE1PressureBenchmarkProvesRiskReduction(t *testing.T) {
	outDir := t.TempDir()
	report, err := RunE1PressureBenchmark(5, outDir)
	if err != nil {
		t.Fatalf("RunE1PressureBenchmark: %v", err)
	}
	if report.Experiment != "e1_pressure_resource_risk" || report.Runs != 5 {
		t.Fatalf("bad report identity: %#v", report)
	}
	if report.SelectedHighPressureAgentCount != 0 {
		t.Fatalf("resource-aware should avoid high-pressure selections: %#v", report)
	}
	if report.AvoidedHighPressureAgentCount != 10 {
		t.Fatalf("avoided high-pressure count = %d, want 10: %#v", report.AvoidedHighPressureAgentCount, report)
	}
	if !report.ResourceAwareReducedRisk {
		t.Fatalf("resource-aware reduced risk flag missing: %#v", report)
	}
	if report.EvidenceMode == "" || !evidence.IsValid(evidence.Mode(report.EvidenceMode)) {
		t.Fatalf("invalid evidence mode: %#v", report)
	}
	if info, err := os.Stat(filepath.Join(outDir, "e1_pressure.json")); err != nil || info.Size() == 0 {
		t.Fatalf("missing e1 pressure artifact info=%#v err=%v", info, err)
	}
	var decoded E1PressureReport
	data, err := os.ReadFile(filepath.Join(outDir, "e1_pressure.json"))
	if err != nil {
		t.Fatalf("read e1 pressure artifact: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode e1 pressure artifact: %v", err)
	}
	if decoded.AvoidedHighPressureAgentCount != 10 || !decoded.ResourceAwareReducedRisk {
		t.Fatalf("bad persisted report: %#v", decoded)
	}
}

func TestRunE2PressureFaultCombinesPressureAndWorkspaceFault(t *testing.T) {
	outDir := t.TempDir()
	report, err := RunE2PressureFault(3, outDir)
	if err != nil {
		t.Fatalf("RunE2PressureFault: %v", err)
	}
	if report.Experiment != "e2_pressure_fault_isolation" || report.Runs != 3 {
		t.Fatalf("bad report identity: %#v", report)
	}
	if report.CascadeFailure || !report.RecoverySuccess || !report.UnaffectedAgentsContinued {
		t.Fatalf("pressure fault should recover without cascade: %#v", report)
	}
	if !report.MemoryPressureInjected || !report.PidsPressureInjected || !report.WorkspaceRMFaultInjected {
		t.Fatalf("combined pressure/workspace injections missing: %#v", report)
	}
	if report.EvidenceMode == "" || !evidence.IsValid(evidence.Mode(report.EvidenceMode)) {
		t.Fatalf("invalid evidence mode: %#v", report)
	}
	if !strings.HasPrefix(report.WorkspaceFaultEvidence.RuntimeRoot, filepath.Join(outDir, "workspace_rmrf_runtime")) {
		t.Fatalf("workspace fault should use outDir-scoped runtime root, got %q", report.WorkspaceFaultEvidence.RuntimeRoot)
	}
	if info, err := os.Stat(filepath.Join(outDir, "e2_pressure_fault.json")); err != nil || info.Size() == 0 {
		t.Fatalf("missing e2 pressure fault artifact info=%#v err=%v", info, err)
	}
}

func containsPolicy(policies []string, want string) bool {
	for _, policy := range policies {
		if policy == want {
			return true
		}
	}
	return false
}

func TestRunE2RealFaultIsolationIncludesWorkspaceRMFault(t *testing.T) {
	results := RunE2RealFaultIsolation(2)
	seenKillCapsule := false
	seenWorkspace := false
	for _, result := range results {
		if result.FaultType == "kill_capsule" {
			seenKillCapsule = true
			if result.FaultEvidence["kill_method"] != "cgroup.kill" && result.FaultEvidence["kill_method"] != "pid-signal-fallback" {
				t.Fatalf("kill_capsule evidence missing kill_method: %#v", result)
			}
		}
		if result.FaultType == "workspace_rmrf" {
			seenWorkspace = true
			if result.EvidenceMode != string(evidence.ModeRealRuntime) && result.EvidenceMode != string(evidence.ModeDegradedCopy) {
				t.Fatalf("unexpected workspace evidence mode: %#v", result)
			}
			if result.CascadeFailure || !result.SystemSurvived {
				t.Fatalf("workspace fault should be isolated: %#v", result)
			}
			if result.FaultEvidence["evidence_mode"] == "" || result.FaultEvidence["fallback_reason"] == nil {
				t.Fatalf("workspace evidence should include mode/fallback: %#v", result.FaultEvidence)
			}
		}
	}
	if !seenKillCapsule {
		t.Fatalf("kill_capsule fault missing from E2: %#v", results)
	}
	if !seenWorkspace {
		t.Fatalf("workspace_rmrf fault missing from E2: %#v", results)
	}
}
