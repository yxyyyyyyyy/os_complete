package codebasedag

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSeedRestorePatchApplies(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "codebasedag")
	pkg := filepath.Join(dir, "resourceagent", "chunk001")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(dir, "judge_resource.go"): `package codebasedag

// ResourceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ResourceJudgeMarker = "judge-resource-complete"
`,
		filepath.Join(dir, "judge_context.go"): `package codebasedag

// ContextJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ContextJudgeMarker = "judge-context-complete"
`,
		filepath.Join(dir, "judge_evidence.go"): `package codebasedag

// EvidenceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const EvidenceJudgeMarker = "judge-evidence-complete"
`,
		filepath.Join(pkg, "gen_demo_001.go"): "package chunk001\n\nfunc Demo() int { return 1 }\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SeedIncompleteJudgeTasks(root); err != nil {
		t.Fatal(err)
	}
	if _, err := SeedBrokenResourceAgent(root); err != nil {
		t.Fatal(err)
	}
	patch, changed, err := BuildSeedRestorePatch(root)
	if err != nil || patch == "" || len(changed) < 2 {
		t.Fatalf("restore patch err=%v files=%v patch_len=%d", err, changed, len(patch))
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	run("git", "init")
	run("git", "add", "-A")
	run("git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "seed")
	if err := CheckPatchApplies(context.Background(), "git", root, patch); err != nil {
		t.Fatal(err)
	}
	coder := CoderOutput{SchemaVersion: SchemaVersion, NodeID: "resource-coder", SeedRestore: true, Summary: "restore"}
	if err := MaterializeCoderPatch(root, []string{"internal/codebasedag/judge_resource.go", "internal/codebasedag/resourceagent"}, &coder); err != nil {
		t.Fatal(err)
	}
	if err := CheckPatchApplies(context.Background(), "git", root, coder.Patch); err != nil {
		t.Fatal(err)
	}
}

func TestSeedBrokenResourceAgentInjectsCompileBreaks(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "internal", "codebasedag", "resourceagent", "chunk001")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(pkg, "gen_demo_001.go")
	if err := os.WriteFile(src, []byte("package resourceagent\n\nfunc Demo001() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := SeedBrokenResourceAgent(root)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 || n > MaxSeedBrokenResourceAgentFiles {
		t.Fatalf("broken=%d want 1..%d", n, MaxSeedBrokenResourceAgentFiles)
	}
	data, _ := os.ReadFile(src)
	if !strings.Contains(string(data), "AORT_SEED_BROKEN") {
		t.Fatalf("missing break marker: %s", data)
	}
	phys, files, err := CountResourceAgentPhysicalLines(root)
	if err != nil || files < 1 || phys < 1 {
		t.Fatalf("count phys=%d files=%d err=%v", phys, files, err)
	}
}

func TestPathAllowedSupportsDirectoryRules(t *testing.T) {
	allowed := map[string]struct{}{
		"internal/codebasedag/resourceagent":     {},
		"internal/codebasedag/judge_resource.go": {},
	}
	if !pathAllowed("internal/codebasedag/resourceagent/chunk001/gen_x.go", allowed) {
		t.Fatal("expected directory allow")
	}
	if pathAllowed("internal/other/x.go", allowed) {
		t.Fatal("expected deny")
	}
}
