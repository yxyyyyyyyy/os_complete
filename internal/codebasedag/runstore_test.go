package codebasedag

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRunStoreRejectsInvalidRunIDs(t *testing.T) {
	root := t.TempDir()
	for _, runID := range []string{"", "../escape", "a/b", `a\b`, " white", "white ", "has space", "."} {
		if _, err := NewRunStore(root, runID); err == nil {
			t.Fatalf("run ID %q should fail", runID)
		}
	}
}

func TestRunStoreCreatesExclusiveRunDirectoryAndArtifacts(t *testing.T) {
	root := t.TempDir()
	store, err := NewRunStore(root, "run-001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRunStore(root, "run-001"); err == nil {
		t.Fatal("second store with same run ID should fail")
	}

	if err := store.WriteJSON("manifest/source.json", map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteJSON("manifest/source.json", map[string]string{"status": "overwrite"}); err == nil {
		t.Fatal("WriteJSON must create artifacts exclusively")
	}
	if err := store.WriteBytes("patches/accepted.diff", []byte("diff --git a/a.go b/a.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendJSONL("events/events.jsonl", map[string]string{"node": "planner"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendJSONL("events/events.jsonl", map[string]string{"node": "tester"}); err != nil {
		t.Fatal(err)
	}

	lines := readLines(t, filepath.Join(store.Dir, "events/events.jsonl"))
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %d", len(lines))
	}
	for _, line := range lines {
		var decoded map[string]string
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("invalid jsonl line %q: %v", line, err)
		}
	}

	hashes, err := store.FinalizeHashes()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["ARTIFACT_SHA256.json"]; ok {
		t.Fatal("hash index must exclude itself")
	}
	want := sha256.Sum256([]byte("diff --git a/a.go b/a.go\n"))
	if hashes["patches/accepted.diff"] != hex.EncodeToString(want[:]) {
		t.Fatalf("patch hash = %q", hashes["patches/accepted.diff"])
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "ARTIFACT_SHA256.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunStoreRejectsEscapingAndSymlinkPaths(t *testing.T) {
	store, err := NewRunStore(t.TempDir(), "run-002")
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../escape.json", "/absolute.json", `bad\slash.json`, ".", "dir/.."} {
		if err := store.WriteBytes(rel, []byte("x"), 0o600); err == nil {
			t.Fatalf("path %q should fail", rel)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(store.Dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteBytes("link/file.txt", []byte("x"), 0o600); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink path error = %v", err)
	}
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}
