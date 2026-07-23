package codebasedag

import (
	"strings"
	"testing"
)

func TestBuildIntegrationPlanOrdersCodersAndDetectsConflicts(t *testing.T) {
	patches := []PatchRecord{
		{NodeID: "evidence-coder", SourceCallID: "c3", SHA256: strings.Repeat("c", 64), ChangedFiles: []string{"internal/review/review_final.go"}, Bytes: 3},
		{NodeID: "resource-coder", SourceCallID: "c1", SHA256: strings.Repeat("a", 64), ChangedFiles: []string{"internal/review/resource_isolation.go"}, Bytes: 1},
		{NodeID: "context-coder", SourceCallID: "c2", SHA256: strings.Repeat("b", 64), ChangedFiles: []string{"internal/review/context_sharing.go"}, Bytes: 2},
		{NodeID: "fixer-1", SourceCallID: "c4", SHA256: strings.Repeat("d", 64), ChangedFiles: []string{"internal/review/resource_isolation.go"}, Bytes: 4},
	}
	plan, err := BuildIntegrationPlan("run-1", patches)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.OrderedPatches) != 4 {
		t.Fatalf("ordered=%d", len(plan.OrderedPatches))
	}
	if plan.OrderedPatches[0].NodeID != "resource-coder" || plan.OrderedPatches[1].NodeID != "context-coder" || plan.OrderedPatches[2].NodeID != "evidence-coder" {
		t.Fatalf("order=%v", plan.OrderedPatches)
	}
	contract := DefaultReviewRemediationContract()
	if err := plan.ValidateAgainstContract(contract); err != nil {
		t.Fatal(err)
	}

	conflictPatches := []PatchRecord{
		{NodeID: "resource-coder", SourceCallID: "c1", SHA256: strings.Repeat("a", 64), ChangedFiles: []string{"internal/review/shared.go"}},
		{NodeID: "context-coder", SourceCallID: "c2", SHA256: strings.Repeat("b", 64), ChangedFiles: []string{"internal/review/shared.go"}},
	}
	if _, err := BuildIntegrationPlan("run-2", conflictPatches); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestBuildIntegrationPlanRejectsBadInput(t *testing.T) {
	if _, err := BuildIntegrationPlan("", nil); err == nil {
		t.Fatal("empty run id")
	}
	if _, err := BuildIntegrationPlan("r", nil); err == nil {
		t.Fatal("empty patches")
	}
	if _, err := BuildIntegrationPlan("r", []PatchRecord{{NodeID: ""}}); err == nil {
		t.Fatal("empty node")
	}
	if _, err := BuildIntegrationPlan("r", []PatchRecord{
		{NodeID: "resource-coder", ChangedFiles: []string{"a.go"}},
		{NodeID: "resource-coder", ChangedFiles: []string{"b.go"}},
	}); err == nil {
		t.Fatal("duplicate node")
	}
}

func TestArtifactIndexExclusiveAndMerkle(t *testing.T) {
	idx := NewArtifactIndex()
	if err := idx.Put("summary.json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := idx.Put("summary.json", []byte(`{"ok":true}`)); err == nil {
		t.Fatal("duplicate put")
	}
	if err := idx.PutHash("report.md", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := idx.PutHash("bad", "zz"); err == nil {
		t.Fatal("short hash")
	}
	paths := idx.SortedPaths()
	if strings.Join(paths, ",") != "report.md,summary.json" {
		t.Fatalf("paths=%v", paths)
	}
	root1 := idx.MerkleRoot()
	root2 := idx.MerkleRoot()
	if root1 == "" || root1 != root2 {
		t.Fatalf("merkle unstable %q %q", root1, root2)
	}
	m := idx.Map()
	if len(m) != 2 {
		t.Fatalf("map=%v", m)
	}
}

func TestTraceBuilderSequencesAndFilters(t *testing.T) {
	if _, err := NewTraceBuilder(""); err == nil {
		t.Fatal("empty run id")
	}
	tr, err := NewTraceBuilder("run-trace")
	if err != nil {
		t.Fatal(err)
	}
	tr.Add("preflight", "preflight", "ok", map[string]string{"gate": "cgroup"})
	tr.Add("planner", "llm.call", "done", map[string]string{"model": "deepseek-v4-flash"})
	tr.Add("tester", "tool.exec", "go test", nil)
	events := tr.Events()
	if len(events) != 3 || events[0].Seq != 1 || events[2].Seq != 3 {
		t.Fatalf("events=%#v", events)
	}
	llm := tr.Filter("llm.call")
	if len(llm) != 1 || llm[0].NodeID != "planner" {
		t.Fatalf("llm=%#v", llm)
	}
}
