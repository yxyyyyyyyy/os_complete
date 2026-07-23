package codebasedag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessEvidenceBundleCreateExclusiveAndVerify(t *testing.T) {
	root := t.TempDir()
	payloadDir := filepath.Join(root, "payload")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payloadDir, "summary.json"), []byte(`{"status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle := NewProcessEvidenceBundle("phase4-test")
	bundle.Host = "116.204.94.247"
	bundle.RemoteDir = "/root/aort-r-huawei-run"
	bundle.Notes = []string{"create-exclusive", "no-overwrite"}
	if err := bundle.AddPath(payloadDir, "summary.json"); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "bundle")
	if err := WriteProcessEvidenceBundle(out, bundle); err != nil {
		t.Fatal(err)
	}
	// copy payload beside manifest for verify helper expecting files under dir
	if err := os.WriteFile(filepath.Join(out, "summary.json"), []byte(`{"status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProcessEvidenceBundle(out)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != "phase4-test" || loaded.SchemaVersion != ProcessEvidenceSchema {
		t.Fatalf("loaded=%#v", loaded)
	}
	if err := VerifyProcessEvidenceBundleFiles(out, loaded); err != nil {
		t.Fatal(err)
	}
	if err := WriteProcessEvidenceBundle(out, bundle); err == nil {
		t.Fatal("overwrite should fail")
	}
}

func TestProcessEvidenceBundleValidationErrors(t *testing.T) {
	b := NewProcessEvidenceBundle("")
	if err := b.Validate(); err == nil {
		t.Fatal("empty phase")
	}
	b.Phase = "x"
	b.CapturedAt = time.Time{}
	if err := b.Validate(); err == nil {
		t.Fatal("zero time")
	}
	b = NewProcessEvidenceBundle("x")
	if err := b.Validate(); err == nil {
		t.Fatal("no files")
	}
	if err := b.AddFile("../evil", []byte("x")); err == nil {
		t.Fatal("evil path")
	}
	if err := b.AddFile("ok.json", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := b.AddFile("ok.json", []byte("y")); err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err=%v", err)
	}
}

func TestIndexProcessEvidenceRootsSorted(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b-run", "a-run", "c-run"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "ignore.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := IndexProcessEvidenceRoots(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "a-run,b-run,c-run" {
		t.Fatalf("names=%v", names)
	}
}
