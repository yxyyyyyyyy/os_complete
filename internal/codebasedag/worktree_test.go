package codebasedag

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorktreeApplyPatchAndRejectEscape(t *testing.T) {
	repo := initTinyRepo(t)
	parent := t.TempDir()
	wt, err := CreateWorktree(context.Background(), repo, parent, "wt1", "git")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = wt.Cleanup(context.Background()) })
	if wt.BaseCommit == "" || wt.BaseTreeHash == "" {
		t.Fatalf("missing base refs: %#v", wt)
	}

	patch := "diff --git a/hello.go b/hello.go\n--- a/hello.go\n+++ b/hello.go\n@@ -1,3 +1,3 @@\n package main\n \n-func Hello() string { return \"hi\" }\n+func Hello() string { return \"hello\" }\n"
	res, err := wt.ApplyValidatedPatch(context.Background(), "resource-coder", patch, []string{"hello.go"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.TreeHashAfter == "" || res.PatchSHA256 == "" {
		t.Fatalf("apply result = %#v", res)
	}
	data, err := os.ReadFile(filepath.Join(wt.WorkDir, "hello.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `return "hello"`) {
		t.Fatalf("patch not applied: %s", data)
	}
}

func TestDetectHunkConflictsSameFile(t *testing.T) {
	a := "diff --git a/hello.go b/hello.go\n--- a/hello.go\n+++ b/hello.go\n@@ -1,3 +1,3 @@\n package main\n \n-func Hello() string { return \"hi\" }\n+func Hello() string { return \"a\" }\n"
	b := "diff --git a/hello.go b/hello.go\n--- a/hello.go\n+++ b/hello.go\n@@ -1,3 +1,3 @@\n package main\n \n-func Hello() string { return \"hi\" }\n+func Hello() string { return \"b\" }\n"
	conflicts := DetectHunkConflicts(map[string]string{"resource-coder": a, "context-coder": b})
	if len(conflicts) == 0 {
		t.Fatal("expected hunk conflict")
	}
}

func TestApplyRejectsUnapplicablePatch(t *testing.T) {
	repo := initTinyRepo(t)
	parent := t.TempDir()
	wt, err := CreateWorktree(context.Background(), repo, parent, "wt-bad", "git")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = wt.Cleanup(context.Background()) })
	bad := "diff --git a/hello.go b/hello.go\n--- a/hello.go\n+++ b/hello.go\n@@ -1,3 +1,3 @@\n package other\n \n-func Missing() {}\n+func Missing() { return }\n"
	_, err = wt.ApplyValidatedPatch(context.Background(), "resource-coder", bad, []string{"hello.go"})
	if err == nil {
		t.Fatal("expected apply failure")
	}
}

func TestDecodeReviewOutputAcceptsFixAndReject(t *testing.T) {
	for _, verdict := range []string{"pass", "fix", "reject"} {
		body := `{"schema_version":"codebase-dag/v1","node_id":"reviewer","verdict":"` + verdict + `","blocking_findings":[],"non_blocking_findings":[]}`
		out, err := DecodeReviewOutput("reviewer", []byte(body))
		if err != nil {
			t.Fatalf("verdict %s: %v", verdict, err)
		}
		if out.Verdict != verdict {
			t.Fatalf("got %q", out.Verdict)
		}
	}
	_, err := DecodeReviewOutput("reviewer", []byte(`{"schema_version":"codebase-dag/v1","node_id":"reviewer","verdict":"fail","blocking_findings":[],"non_blocking_findings":[]}`))
	if err == nil {
		t.Fatal("legacy fail verdict must be rejected")
	}
}

func initTinyRepo(t *testing.T) string {
	t.Helper()
	dir := newTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc Hello() string { return \"hi\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "init", "--template="},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "test"},
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return dir
}
