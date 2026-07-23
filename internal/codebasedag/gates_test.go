package codebasedag

import (
	"strings"
	"testing"
)

func TestOpenWorldGateCatalogComplete(t *testing.T) {
	catalog := OpenWorldGateCatalog()
	if len(catalog) < 15 {
		t.Fatalf("catalog size=%d", len(catalog))
	}
	seen := map[GateID]bool{}
	for _, gate := range catalog {
		if gate.ID == "" || gate.Title == "" || gate.Description == "" {
			t.Fatalf("bad gate %#v", gate)
		}
		if seen[gate.ID] {
			t.Fatalf("duplicate %s", gate.ID)
		}
		seen[gate.ID] = true
	}
	md := GateCatalogMarkdown()
	if !strings.Contains(md, string(GateNoMOOC)) || !strings.Contains(md, "30000") {
		t.Fatalf("markdown=%s", md)
	}
}

func TestEvaluateOpenWorldGatesPassAndFail(t *testing.T) {
	summary := EvidenceSummary{
		SchemaVersion: SchemaVersion,
		RunID:         "g1",
		SourceManifest: SourceManifest{
			PhysicalLines: DefaultMinPhysicalLines,
			NonblankLines: DefaultMinNonblankLines,
		},
		Calls: []CallRecord{{
			Provider: RequiredDeepSeekProvider, RequestedModel: RequiredDeepSeekModel, ActualModel: RequiredDeepSeekModel,
			EvidenceMode: "real-api",
		}},
		Patches: []PatchRecord{
			{NodeID: "resource-coder"}, {NodeID: "context-coder"}, {NodeID: "evidence-coder"},
		},
		Artifacts:            map[string]string{"summary.json": strings.Repeat("a", 64)},
		AllRequiredPassed:    true,
		HumanFunctionalEdits: 0,
	}
	probe := OSProbeResult{
		OS:      OSRelease{ID: "openEuler", VersionID: "24.03"},
		Cgroup:  CgroupProbe{FilesystemType: "cgroup2fs", EvidenceMode: "real-cgroup-v2"},
		Overlay: OverlayProbe{MountSucceeded: true, EvidenceMode: "real-overlayfs"},
		Memfd:   MemfdProbe{FDPassingSucceeded: true, EvidenceMode: "real-memfd"},
	}
	budget := CallBudgetSnapshot{Successes: 7, Attempts: 7, RequiredMin: 7, MaxCalls: 10, Satisfied: true}
	report := EvaluateOpenWorldGates(summary, probe, budget, 0)
	if !report.Passed {
		t.Fatalf("expected pass: %#v", report)
	}
	summary.HumanFunctionalEdits = 1
	fail := EvaluateOpenWorldGates(summary, probe, budget, 0)
	if fail.Passed {
		t.Fatal("expected fail on human edits")
	}
}

func TestGateHelpers(t *testing.T) {
	if allCallsProvider(nil, "deepseek") {
		t.Fatal("empty calls")
	}
	if allCallsModel(nil, "m") {
		t.Fatal("empty model calls")
	}
	if !noFallbackCalls([]CallRecord{{EvidenceMode: "real-api"}}) {
		t.Fatal("real-api should pass")
	}
	if noFallbackCalls([]CallRecord{{EvidenceMode: "mock", Fallback: true}}) {
		t.Fatal("mock should fail")
	}
	if countCoderPatches(nil) != 0 {
		t.Fatal("empty patches")
	}
}
