package codebasedag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeAcceptanceWritesExecutableImmutableScripts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "acceptance")
	records, err := MaterializeAcceptance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("records = %d", len(records))
	}
	for _, record := range records {
		if record.Path == "" || record.SHA256 == "" {
			t.Fatalf("bad record: %#v", record)
		}
		info, err := os.Stat(filepath.Join(dir, filepath.Base(record.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o500 {
			t.Fatalf("%s mode = %o", record.Path, info.Mode().Perm())
		}
	}
	if _, err := MaterializeAcceptance(dir); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second materialize error = %v", err)
	}
}

func TestVerifyAcceptanceDetectsMutation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "acceptance")
	records, err := MaterializeAcceptance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyAcceptance(dir, records); err != nil {
		t.Fatal(err)
	}
	mutated := filepath.Join(dir, filepath.Base(records[0].Path))
	if err := os.Chmod(mutated, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mutated, []byte("#!/bin/sh\nexit 99\n"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := VerifyAcceptance(dir, records); err == nil {
		t.Fatal("mutated script should fail verification")
	}
}
