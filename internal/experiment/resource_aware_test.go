package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	var rows []E1ResourceAwareResult
	data, err := os.ReadFile(filepath.Join(outDir, "e1_resource_aware.json"))
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if rows[0].MemoryPeakBytes == 0 || rows[0].PidsPeak == 0 {
		t.Fatalf("resource metrics missing: %#v", rows[0])
	}
}

func TestRunE2RealFaultIsolationIncludesWorkspaceRMFault(t *testing.T) {
	results := RunE2RealFaultIsolation(2)
	for _, result := range results {
		if result.FaultType == "workspace_rmrf" {
			if result.EvidenceMode != string(evidence.ModeRealRuntime) && result.EvidenceMode != string(evidence.ModeDegradedCopy) {
				t.Fatalf("unexpected workspace evidence mode: %#v", result)
			}
			if result.CascadeFailure || !result.SystemSurvived {
				t.Fatalf("workspace fault should be isolated: %#v", result)
			}
			if result.FaultEvidence["evidence_mode"] == "" || result.FaultEvidence["fallback_reason"] == nil {
				t.Fatalf("workspace evidence should include mode/fallback: %#v", result.FaultEvidence)
			}
			return
		}
	}
	t.Fatalf("workspace_rmrf fault missing from E2: %#v", results)
}
