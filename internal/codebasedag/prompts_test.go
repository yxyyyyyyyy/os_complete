package codebasedag

import (
	"os"
	"strings"
	"testing"
)

func TestBuildPromptIncludesRoleContractAndOmitsSecrets(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-secret-value")
	req := PromptRequest{
		NodeID:          "resource-coder",
		Role:            KindCoder,
		Ticket:          "review-remediation",
		AllowedFiles:    []string{"internal/review/resource.go", "cmd/aortctl/main.go"},
		ImmutableFiles:  []string{"internal/codebasedag/acceptance/resource_real.sh"},
		SharedContextID: "sha256:abc123",
		PrivateContext:  "resource isolation needs real cgroup evidence",
	}
	prompt, err := BuildPrompt(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"node_id: resource-coder",
		"role: coder",
		"review-remediation",
		"internal/review/resource.go",
		"internal/codebasedag/acceptance/resource_real.sh",
		"sha256:abc123",
		`"patch"`,
		"Return exactly one JSON object",
		"unified diff",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, os.Getenv("DEEPSEEK_API_KEY")) {
		t.Fatal("prompt leaked API key")
	}
}

func TestBuildPromptRejectsUnknownRoleAndUnsafeAllowedFiles(t *testing.T) {
	if _, err := BuildPrompt(PromptRequest{NodeID: "n", Role: NodeKind("unknown"), Ticket: "t"}); err == nil {
		t.Fatal("unknown role should fail")
	}
	if _, err := BuildPrompt(PromptRequest{
		NodeID:       "n",
		Role:         KindPlanner,
		Ticket:       "t",
		AllowedFiles: []string{"../escape.go"},
	}); err == nil {
		t.Fatal("escaping allowlist path should fail")
	}
}

func TestBuildPromptProvidesSchemasForEveryRole(t *testing.T) {
	for _, role := range []NodeKind{KindPlanner, KindCoder, KindTester, KindReviewer, KindFixer, KindFinalizer} {
		prompt, err := BuildPrompt(PromptRequest{NodeID: string(role) + "-node", Role: role, Ticket: "ticket", SharedContextID: "ctx"})
		if err != nil {
			t.Fatalf("%s: %v", role, err)
		}
		if !strings.Contains(prompt, `"schema_version"`) || !strings.Contains(prompt, SchemaVersion) {
			t.Fatalf("%s prompt lacks schema version:\n%s", role, prompt)
		}
	}
}
