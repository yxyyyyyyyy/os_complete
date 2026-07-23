package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeWorkerReplayOutputVerifiesHashAndReports(t *testing.T) {
	dir := t.TempDir()
	output := []byte(`{"schema_version":"codebase-dag/v1","node_id":"planner"}`)
	path := filepath.Join(dir, "planner.json")
	if err := os.WriteFile(path, output, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(output)
	var stdout bytes.Buffer
	err := run([]string{
		"--run-id", "run",
		"--agent-id", "agent-1",
		"--node-id", "planner",
		"--role", "planner",
		"--replay-output", path,
		"--expected-hash", hex.EncodeToString(sum[:]),
		"--llm-call-id", "call-1",
	}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	var report WorkerReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Status != "replayed" || report.OutputSHA256 != hex.EncodeToString(sum[:]) || report.LLMCallID != "call-1" {
		t.Fatalf("report = %#v", report)
	}
}

func TestCodeWorkerRejectsMissingRequiredFlagsAndHashMismatch(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "run-id") {
		t.Fatalf("missing flags error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{
		"--run-id", "run",
		"--agent-id", "agent",
		"--node-id", "planner",
		"--role", "planner",
		"--replay-output", path,
		"--expected-hash", strings.Repeat("0", 64),
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("hash mismatch error = %v", err)
	}
}
