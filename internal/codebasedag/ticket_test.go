package codebasedag

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReviewRemediationTicketDefinesNodePolicies(t *testing.T) {
	ticket := ReviewRemediationTicket()
	if ticket.ID != "review-remediation" || ticket.SharedContext == "" {
		t.Fatalf("ticket = %#v", ticket)
	}
	for _, node := range []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "finalizer"} {
		if _, ok := ticket.NodePolicies[node]; !ok {
			t.Fatalf("missing policy for %s", node)
		}
	}
	if got := ticket.NodePolicies["resource-coder"].AllowedFiles; !containsString(got, "internal/review/live_resource_hook.go") {
		t.Fatalf("resource allowlist = %#v", got)
	}
}

func TestLoadTicketFileValidatesAndSortsPolicies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ticket.json")
	raw := `{
	  "id": "custom",
	  "shared_context": "sha256:ticket",
	  "node_policies": {
	    "coder": {
	      "role": "coder",
	      "allowed_files": ["b.go", "a.go"],
	      "immutable_files": ["acceptance/check.sh"],
	      "private_context": "do work"
	    }
	  }
	}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	ticket, err := LoadTicketFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.go", "b.go"}
	if got := ticket.NodePolicies["coder"].AllowedFiles; !reflect.DeepEqual(got, want) {
		t.Fatalf("allowed files = %#v, want %#v", got, want)
	}
}

func TestLoadTicketFileRejectsUnsafePolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ticket.json")
	raw := `{"id":"bad","shared_context":"ctx","node_policies":{"coder":{"role":"coder","allowed_files":["../escape.go"]}}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTicketFile(path); err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("error = %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
