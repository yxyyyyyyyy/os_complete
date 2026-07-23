package codebasedag

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildReportMarkdownContainsAuthoritativeGates(t *testing.T) {
	summary := EvidenceSummary{
		SchemaVersion: SchemaVersion,
		RunID:         "run-report-1",
		SourceManifest: SourceManifest{
			SchemaVersion:  SchemaVersion,
			PhysicalLines:  DefaultMinPhysicalLines,
			NonblankLines:  DefaultMinNonblankLines,
			TrackedGoFiles: 120,
			TreeHash:       strings.Repeat("a", 64),
		},
		Nodes: []NodeRecord{
			{SchemaVersion: SchemaVersion, NodeID: "planner", Kind: KindPlanner, Status: NodeSucceeded, LLMCallID: "c1", OutputSHA256: strings.Repeat("b", 64)},
			{SchemaVersion: SchemaVersion, NodeID: "resource-coder", Kind: KindCoder, Status: NodeSucceeded, LLMCallID: "c2", OutputSHA256: strings.Repeat("c", 64)},
		},
		Calls: []CallRecord{{
			SchemaVersion: SchemaVersion, CallID: "c1", NodeID: "planner", Role: "planner",
			Provider: RequiredDeepSeekProvider, RequestedModel: RequiredDeepSeekModel, ActualModel: RequiredDeepSeekModel,
			EvidenceMode: "real-api", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, DurationMS: 100, Status: "succeeded",
			OutputSHA256: strings.Repeat("d", 64),
		}},
		Patches: []PatchRecord{
			{NodeID: "resource-coder", SourceCallID: "c2", SHA256: strings.Repeat("e", 64), Bytes: 120, ChangedFiles: []string{"internal/review/resource_isolation.go"}},
			{NodeID: "context-coder", SourceCallID: "c3", SHA256: strings.Repeat("f", 64), Bytes: 130, ChangedFiles: []string{"internal/review/context_sharing.go"}},
			{NodeID: "evidence-coder", SourceCallID: "c4", SHA256: strings.Repeat("1", 64), Bytes: 140, ChangedFiles: []string{"internal/review/review_final.go"}},
		},
		Tests: []TestRecord{{
			SchemaVersion: SchemaVersion, Name: "go test", Command: []string{"go", "test", "./..."}, ExitCode: 0,
			StdoutSHA256: strings.Repeat("2", 64), StderrSHA256: strings.Repeat("3", 64),
		}},
		Artifacts:         map[string]string{"summary.json": strings.Repeat("4", 64)},
		AllRequiredPassed: true,
	}
	probe := OSProbeResult{
		OS:      OSRelease{ID: "openEuler", VersionID: "24.03", PrettyName: "openEuler 24.03 (LTS)"},
		Cgroup:  CgroupProbe{FilesystemType: "cgroup2fs", Writable: true, NestedCreate: true, EvidenceMode: "real-cgroup-v2"},
		Overlay: OverlayProbe{Available: true, MountSucceeded: true, EvidenceMode: "real-overlayfs"},
		Memfd:   MemfdProbe{Available: true, MmapSucceeded: true, FDPassingSucceeded: true, EvidenceMode: "real-memfd"},
	}
	events := []EvidenceEvent{{
		SchemaVersion: SchemaVersion, RunID: "run-report-1", Type: EventPreflight, At: time.Unix(1, 0).UTC(), Message: "gates ok",
	}}
	doc := BuildReport(summary, probe, events)
	md := doc.Markdown()
	for _, want := range []string{
		"run-report-1",
		"status: **passed**",
		"physical_go_lines: 30000",
		"Open World Environment",
		"real-cgroup-v2",
		"Process Journal",
		"Authority",
		"not model KV-cache sharing",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q\n%s", want, md)
		}
	}
	if doc.Status != "passed" {
		t.Fatalf("status=%q", doc.Status)
	}
}

func TestBuildReportFailedStatusWhenGatesFail(t *testing.T) {
	summary := EvidenceSummary{SchemaVersion: SchemaVersion, RunID: "run-fail", AllRequiredPassed: false}
	doc := BuildReport(summary, OSProbeResult{}, nil)
	if doc.Status != "failed" {
		t.Fatalf("status=%q", doc.Status)
	}
	md := doc.Markdown()
	if !strings.Contains(md, "status: **failed**") {
		t.Fatalf("markdown=%s", md)
	}
}

func TestFormatHelpersHandleEmptyCollections(t *testing.T) {
	if got := formatNodes(nil); got != "no nodes recorded" {
		t.Fatalf("nodes=%q", got)
	}
	if got := formatCalls(nil); got != "no llm calls recorded" {
		t.Fatalf("calls=%q", got)
	}
	if got := formatPatches(nil); got != "no patches recorded" {
		t.Fatalf("patches=%q", got)
	}
	if got := formatTests(nil); got != "no tests recorded" {
		t.Fatalf("tests=%q", got)
	}
	if got := formatJournal(nil); got != "no journal events" {
		t.Fatalf("journal=%q", got)
	}
	if shortHash("abcd") != "abcd" {
		t.Fatal("short hash of short value changed")
	}
	if got := shortHash(strings.Repeat("a", 64)); got != strings.Repeat("a", 12) {
		t.Fatalf("shortHash=%q", got)
	}
}

func TestFormatAcceptanceIncludesValidatorError(t *testing.T) {
	summary := EvidenceSummary{SchemaVersion: SchemaVersion, RunID: "x", AllRequiredPassed: false}
	body := formatAcceptance(summary)
	if !strings.Contains(body, "all_required_passed=false") {
		t.Fatalf("body=%s", body)
	}
	if !strings.Contains(body, "validator_error=") {
		t.Fatalf("missing validator error field: %s", body)
	}
}

func TestDefaultLimitationsAreExplicit(t *testing.T) {
	items := defaultLimitations()
	if len(items) < 4 {
		t.Fatalf("limitations=%d", len(items))
	}
	joined := strings.Join(items, "\n")
	for _, want := range []string{"KV-cache", "zero-copy", "sandbox", "MOOC"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %s", want, joined)
		}
	}
}

func TestFormatProbeAndCollections(t *testing.T) {
	probe := OSProbeResult{
		OS:      OSRelease{ID: "openEuler", VersionID: "24.03", PrettyName: "pretty"},
		Cgroup:  CgroupProbe{FilesystemType: "cgroup2fs", Writable: true, NestedCreate: true, EvidenceMode: "real-cgroup-v2"},
		Overlay: OverlayProbe{Available: true, MountSucceeded: true, EvidenceMode: "real-overlayfs"},
		Memfd:   MemfdProbe{Available: true, MmapSucceeded: true, FDPassingSucceeded: true, EvidenceMode: "real-memfd"},
	}
	body := formatProbe(probe)
	if !strings.Contains(body, "openEuler 24.03") || !strings.Contains(body, "real-overlayfs") {
		t.Fatalf("probe=%s", body)
	}
	nodes := formatNodes([]NodeRecord{{NodeID: "b"}, {NodeID: "a", Status: NodeSucceeded}})
	if !strings.HasPrefix(nodes, "- a ") {
		t.Fatalf("nodes should be sorted: %s", nodes)
	}
	calls := formatCalls([]CallRecord{{CallID: "1", NodeID: "planner", Role: "planner", Provider: "deepseek", RequestedModel: "m", ActualModel: "m", EvidenceMode: "real-api", Status: "succeeded"}})
	if !strings.Contains(calls, "planner") {
		t.Fatalf("calls=%s", calls)
	}
	patches := formatPatches([]PatchRecord{{NodeID: "resource-coder", SourceCallID: "c", SHA256: strings.Repeat("a", 64), ChangedFiles: []string{"a.go"}, Bytes: 9}})
	if !strings.Contains(patches, "resource-coder") {
		t.Fatalf("patches=%s", patches)
	}
	tests := formatTests([]TestRecord{{Name: "unit", Command: []string{"go", "test"}, ExitCode: 0}})
	if !strings.Contains(tests, `cmd="go test"`) {
		t.Fatalf("tests=%s", tests)
	}
	journal := formatJournal([]EvidenceEvent{{Type: EventNodeStart, NodeID: "planner", Message: "start", At: time.Unix(2, 0).UTC()}})
	if !strings.Contains(journal, "planner") || !strings.Contains(journal, "start") {
		t.Fatalf("journal=%s", journal)
	}
	_ = fmt.Sprintf("ok")
}
