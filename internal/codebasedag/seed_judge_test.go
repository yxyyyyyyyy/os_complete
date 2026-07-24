package codebasedag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedIncompleteJudgeTasksIgnoresCommentProse(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "codebasedag")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"judge_resource.go": `package codebasedag

// ResourceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ResourceJudgeMarker = "judge-resource-complete"
`,
		"judge_context.go": `package codebasedag

// ContextJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ContextJudgeMarker = "judge-context-complete"
`,
		"judge_evidence.go": `package codebasedag

// EvidenceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const EvidenceJudgeMarker = "judge-evidence-complete"
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SeedIncompleteJudgeTasks(root); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"judge_resource.go": `const ResourceJudgeMarker = "seed-incomplete"`,
		"judge_context.go":  `const ContextJudgeMarker = "seed-incomplete"`,
		"judge_evidence.go": `const EvidenceJudgeMarker = "seed-incomplete"`,
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		if !strings.Contains(body, want) {
			t.Fatalf("%s missing seeded const: %s", name, body)
		}
		if strings.Contains(body, `= "judge-`) {
			t.Fatalf("%s still has complete marker: %s", name, body)
		}
	}
}
