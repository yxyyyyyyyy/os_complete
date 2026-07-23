package codebasedag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEvidenceSummaryAcceptsStrictPassingRun(t *testing.T) {
	summary := strictTestSummary()
	if err := ValidateEvidenceSummary(summary); err != nil {
		t.Fatal(err)
	}
}

func TestValidateEvidenceSummaryRejectsStrictFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*EvidenceSummary)
		want   string
	}{
		{name: "wrong schema", mutate: func(s *EvidenceSummary) { s.SchemaVersion = "old" }, want: "schema"},
		{name: "below physical gate", mutate: func(s *EvidenceSummary) { s.SourceManifest.PhysicalLines = DefaultMinPhysicalLines - 1 }, want: "physical"},
		{name: "wrong provider", mutate: func(s *EvidenceSummary) { s.Calls[0].Provider = "mock" }, want: "provider"},
		{name: "wrong model", mutate: func(s *EvidenceSummary) { s.Calls[0].ActualModel = "deepseek-chat" }, want: "model"},
		{name: "fallback", mutate: func(s *EvidenceSummary) { s.Calls[0].Fallback = true }, want: "fallback"},
		{name: "failed node", mutate: func(s *EvidenceSummary) { s.Nodes[0].Status = NodeFailed }, want: "node"},
		{name: "too few calls", mutate: func(s *EvidenceSummary) { s.Calls = s.Calls[:6] }, want: "at least 7"},
		{name: "too many calls", mutate: func(s *EvidenceSummary) {
			for len(s.Calls) < 11 {
				next := s.Calls[0]
				next.CallID = next.CallID + "x"
				s.Calls = append(s.Calls, next)
			}
		}, want: "at most 10"},
		{name: "human edit", mutate: func(s *EvidenceSummary) { s.HumanFunctionalEdits = 1 }, want: "human"},
		{name: "failed test", mutate: func(s *EvidenceSummary) { s.Tests[0].ExitCode = 1 }, want: "test"},
		{name: "missing patch", mutate: func(s *EvidenceSummary) { s.Patches = nil }, want: "patch"},
		{name: "missing artifact hash", mutate: func(s *EvidenceSummary) { s.Artifacts = nil }, want: "artifact"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := strictTestSummary()
			tc.mutate(&summary)
			err := ValidateEvidenceSummary(summary)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRunReadsSummaryJSON(t *testing.T) {
	dir := t.TempDir()
	writeJSONForTest(t, filepath.Join(dir, "summary.json"), strictTestSummary())
	if err := ValidateRun(dir); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRun(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing run directory should fail")
	}
}

func strictTestSummary() EvidenceSummary {
	nodes := []NodeRecord{
		{SchemaVersion: SchemaVersion, NodeID: "preflight", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "planner", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "resource-coder", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "context-coder", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "evidence-coder", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "integrate", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "tester", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "reviewer", Status: NodeSucceeded},
		{SchemaVersion: SchemaVersion, NodeID: "finalizer", Status: NodeSucceeded},
	}
	calls := make([]CallRecord, 0, 7)
	for i := 0; i < 7; i++ {
		calls = append(calls, CallRecord{
			SchemaVersion:    SchemaVersion,
			CallID:           "call-" + string(rune('a'+i)),
			NodeID:           "node",
			Provider:         RequiredDeepSeekProvider,
			RequestedModel:   RequiredDeepSeekModel,
			ActualModel:      RequiredDeepSeekModel,
			EvidenceMode:     "real-api",
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
			OutputSHA256:     strings.Repeat("a", 64),
			Status:           "succeeded",
		})
	}
	return EvidenceSummary{
		SchemaVersion: SchemaVersion,
		RunID:         "run-test",
		SourceManifest: SourceManifest{
			SchemaVersion:  SchemaVersion,
			PhysicalLines:  DefaultMinPhysicalLines,
			NonblankLines:  DefaultMinNonblankLines,
			TrackedGoFiles: 1,
			TreeHash:       strings.Repeat("b", 64),
		},
		Nodes: nodes,
		Calls: calls,
		Patches: []PatchRecord{
			{NodeID: "resource-coder", SHA256: strings.Repeat("d", 64), ChangedFiles: []string{"internal/review/live_resource_hook.go"}, SourceCallID: "call-a"},
			{NodeID: "context-coder", SHA256: strings.Repeat("e", 64), ChangedFiles: []string{"internal/review/live_context_hook.go"}, SourceCallID: "call-b"},
			{NodeID: "evidence-coder", SHA256: strings.Repeat("f", 64), ChangedFiles: []string{"internal/review/live_evidence_hook.go"}, SourceCallID: "call-c"},
		},
		Tests:                []TestRecord{{SchemaVersion: SchemaVersion, Name: "go test", ExitCode: 0}},
		Artifacts:            map[string]string{"summary.json": strings.Repeat("c", 64)},
		AllRequiredPassed:    true,
		HumanFunctionalEdits: 0,
	}
}

func writeJSONForTest(t *testing.T, path string, value any) {
	t.Helper()
	data := mustJSON(t, value)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
