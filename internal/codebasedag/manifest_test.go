package codebasedag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildSourceManifestCountsTrackedGoFilesAndHashes(t *testing.T) {
	repo := newTestGitRepo(t)
	writeFile(t, repo, "alpha/a.go", []byte("package alpha\n\nfunc A() {}\n"), 0o644)
	writeFile(t, repo, "beta/b.go", []byte("package beta\nfunc B() {}"), 0o644)
	writeFile(t, repo, "README.md", []byte("# ignored by Go manifest\n"), 0o644)
	writeFile(t, repo, ".gitignore", []byte("ignored.go\n"), 0o644)
	writeFile(t, repo, "ignored.go", []byte("package ignored\n"), 0o644)
	gitPath := fakeGitPath(t)

	manifest, seeds, err := BuildSourceManifestWithOptions(context.Background(), repo, ManifestOptions{
		MinPhysical: 5,
		MinNonblank: 4,
		GitPath:     gitPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if manifest.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q", manifest.SchemaVersion)
	}
	if manifest.GitDirty {
		t.Fatal("fresh committed repo should not be dirty")
	}
	if manifest.PhysicalLines != 5 || manifest.NonblankLines != 4 || manifest.TrackedGoFiles != 2 {
		t.Fatalf("line counts = physical %d nonblank %d files %d", manifest.PhysicalLines, manifest.NonblankLines, manifest.TrackedGoFiles)
	}
	gotPaths := []string{manifest.Files[0].Path, manifest.Files[1].Path}
	if want := []string{"alpha/a.go", "beta/b.go"}; !reflect.DeepEqual(gotPaths, want) {
		t.Fatalf("paths = %#v, want %#v", gotPaths, want)
	}
	for _, file := range manifest.Files {
		data := readFile(t, repo, file.Path)
		sum := sha256.Sum256(data)
		if file.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("%s hash = %s", file.Path, file.SHA256)
		}
	}
	if manifest.TreeHash == "" {
		t.Fatal("tree hash must be populated")
	}
	if len(seeds) != 4 {
		t.Fatalf("seed files = %d, want tracked regular files including markdown and .gitignore", len(seeds))
	}

	writeFile(t, repo, "untracked.go", []byte("package dirty\n"), 0o644)
	dirtyManifest, _, err := BuildSourceManifestWithOptions(context.Background(), repo, ManifestOptions{
		MinPhysical: 5,
		MinNonblank: 4,
		GitPath:     gitPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !dirtyManifest.GitDirty {
		t.Fatal("visible untracked file should mark repo dirty")
	}
}

func TestSourceManifestLargeCodeGateUsesDefaultAndOverrideThresholds(t *testing.T) {
	manifest := SourceManifest{PhysicalLines: DefaultMinPhysicalLines, NonblankLines: DefaultMinNonblankLines - 1}
	if err := manifest.ValidateLargeCodebase(); err == nil {
		t.Fatal("default nonblank threshold must fail")
	}
	if err := manifest.ValidateLargeCodebaseWithOptions(ManifestOptions{MinPhysical: 1, MinNonblank: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildSourceManifestRejectsSymlinkTrackedFile(t *testing.T) {
	repo := newTestGitRepo(t)
	writeFile(t, repo, "target.go", []byte("package demo\n"), 0o644)
	if err := os.Symlink("target.go", filepath.Join(repo, "link.go")); err != nil {
		t.Fatal(err)
	}

	_, _, err := BuildSourceManifestWithOptions(context.Background(), repo, ManifestOptions{
		MinPhysical: 1,
		MinNonblank: 1,
		GitPath:     fakeGitPath(t),
	})
	if err == nil {
		t.Fatal("tracked symlink must be rejected")
	}
}

func newTestGitRepo(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpRoot := filepath.Join(wd, "..", "..", ".cache", "codebasedag-tests")
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	repo, err := os.MkdirTemp(tmpRoot, "repo-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(repo)
	})
	return repo
}

func fakeGitPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(wd, "..", "..", ".cache", "fake-git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "git")
	script := `#!/bin/sh
set -eu
if [ "$1" != "-C" ]; then
  echo "missing -C" >&2
  exit 2
fi
repo="$2"
cmd="$3"
case "$cmd" in
  rev-parse)
    printf '%s\n' 0123456789abcdef0123456789abcdef01234567
    ;;
  status)
    if [ -f "$repo/untracked.go" ]; then
      printf '%s\n' '?? untracked.go'
    fi
    ;;
  ls-files)
    if [ -L "$repo/link.go" ]; then
      printf 'link.go\0target.go\0'
    else
      printf '.gitignore\0README.md\0alpha/a.go\0beta/b.go\0'
    fi
    ;;
  *)
    echo "unexpected fake git command: $cmd" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFile(t *testing.T, root, rel string, data []byte, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(data)
}
