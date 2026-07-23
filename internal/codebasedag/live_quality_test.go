package codebasedag

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreflightRejectsDirtyWorktreeInStrictMode(t *testing.T) {
	repo := initTinyRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "dirty.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	preflight := LocalPreflight{
		ManifestOptions: ManifestOptions{MinPhysical: 1, MinNonblank: 1},
		RequireClean:    true,
	}
	_, err := preflight.Check(context.Background(), RunnerConfig{
		RunID: "dirty-run", WorkloadDir: repo, Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
	})
	if err == nil {
		t.Fatal("expected dirty worktree rejection")
	}
}

func TestValidateRunFailsWhenArtifactHashTampered(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRunStore(dir, "tamper-run")
	if err != nil {
		t.Fatal(err)
	}
	summary := validEvidenceSummaryFixture()
	if err := store.WriteJSON("summary.json", summary); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalizeHashes(); err != nil {
		t.Fatal(err)
	}
	// Tamper an artifact file content without updating index expectations in summary.
	path := filepath.Join(store.Dir, "summary.json")
	data, _ := os.ReadFile(path)
	var decoded EvidenceSummary
	_ = json.Unmarshal(data, &decoded)
	decoded.Artifacts["summary.json"] = strings.Repeat("0", 64)
	tampered, _ := json.MarshalIndent(decoded, "", "  ")
	_ = os.WriteFile(path, append(tampered, '\n'), 0o600)
	// ValidateEvidenceSummary only checks hash shape today; assert tamper detectable via index mismatch helper.
	idx, err := readArtifactIndex(store)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256File(path)
	if idx["summary.json"] == sum {
		t.Fatal("expected on-disk hash to diverge from pre-tamper index after rewrite")
	}
	if err := VerifyArtifactIntegrity(store.Dir); err == nil {
		t.Fatal("expected integrity verification failure")
	}
}

func TestMachineTesterFailsOnNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/bad\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package bad\n\nfunc Broken() { missing }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := RunMachineTester(context.Background(), TesterConfig{
		WorkDir:  dir,
		SkipRace: true,
		Timeout:  1 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected tester failure")
	}
}

func validEvidenceSummaryFixture() EvidenceSummary {
	calls := make([]CallRecord, 0, 7)
	nodes := []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"}
	for i, node := range nodes {
		calls = append(calls, CallRecord{
			SchemaVersion: SchemaVersion, CallID: "c" + string(rune('1'+i)), NodeID: node, Role: node,
			Provider: RequiredDeepSeekProvider, RequestedModel: RequiredDeepSeekModel, ActualModel: RequiredDeepSeekModel,
			EvidenceMode: "real-api", PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, DurationMS: 1,
			OutputSHA256: strings.Repeat("a", 64), Status: "succeeded",
		})
	}
	return EvidenceSummary{
		SchemaVersion: SchemaVersion,
		RunID:         "tamper-run",
		SourceManifest: SourceManifest{
			SchemaVersion: SchemaVersion, GitCommit: "abc", TreeHash: strings.Repeat("b", 64),
			PhysicalLines: DefaultMinPhysicalLines, NonblankLines: DefaultMinNonblankLines, TrackedGoFiles: 10,
		},
		Nodes: []NodeRecord{
			{SchemaVersion: SchemaVersion, NodeID: "preflight", Kind: KindPreflight, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "planner", Kind: KindPlanner, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "resource-coder", Kind: KindCoder, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "context-coder", Kind: KindCoder, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "evidence-coder", Kind: KindCoder, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "integrate", Kind: KindIntegrate, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "tester", Kind: KindTester, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "reviewer", Kind: KindReviewer, Status: NodeSucceeded},
			{SchemaVersion: SchemaVersion, NodeID: "finalizer", Kind: KindFinalizer, Status: NodeSucceeded},
		},
		Calls: calls,
		Patches: []PatchRecord{
			{NodeID: "resource-coder", SHA256: strings.Repeat("c", 64), ChangedFiles: []string{"a.go"}, SourceCallID: "c1"},
			{NodeID: "context-coder", SHA256: strings.Repeat("d", 64), ChangedFiles: []string{"b.go"}, SourceCallID: "c2"},
			{NodeID: "evidence-coder", SHA256: strings.Repeat("e", 64), ChangedFiles: []string{"c.go"}, SourceCallID: "c3"},
		},
		Tests:             []TestRecord{{SchemaVersion: SchemaVersion, Name: "go-test", Command: []string{"go", "test", "./..."}, ExitCode: 0}},
		Artifacts:         map[string]string{"summary.json": strings.Repeat("f", 64)},
		AllRequiredPassed: true,
	}
}

func sha256File(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return sha256Hex(data)
}
