package codebasedag

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFixerAckPatchAppliesWithGit(t *testing.T) {
	root := t.TempDir()
	rel := "internal/codebasedag/judge_resource.go"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package codebasedag\n\nconst ResourceJudgeMarker = \"judge-resource-complete\"\n"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = root
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "test")
	git("add", "-A")
	git("commit", "-qm", "init")

	patch, files, err := BuildFixerAckPatch(root, []string{rel})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != rel {
		t.Fatalf("files=%v", files)
	}
	patchPath := filepath.Join(root, "ack.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		t.Fatal(err)
	}
	git("apply", "--check", patchPath)
	if !strings.Contains(patch, "aort-fixer-ack") {
		t.Fatalf("missing marker in patch")
	}
}
