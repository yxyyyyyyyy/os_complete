package codebasedag

import (
	"strings"
	"testing"
)

func TestDefaultReviewRemediationContractValidAndScoped(t *testing.T) {
	c := DefaultReviewRemediationContract()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := c.AllowPath("resource-coder", "internal/review/live_resource_hook.go"); err != nil {
		t.Fatal(err)
	}
	if err := c.AllowPath("context-coder", "internal/review/live_context_hook.go"); err != nil {
		t.Fatal(err)
	}
	if err := c.AllowPath("evidence-coder", "internal/review/live_evidence_hook.go"); err != nil {
		t.Fatal(err)
	}
	if err := c.AllowPath("resource-coder", "internal/codebasedag/acceptance/resource_real.sh"); err == nil {
		t.Fatal("forbidden acceptance path should fail")
	}
	if err := c.AllowPath("resource-coder", "internal/review/context_sharing.go"); err == nil {
		t.Fatal("cross-scope path should fail")
	}
	block := c.PromptBlock()
	for _, want := range []string{"review-remediation", "deepseek-v4-flash", "FORBIDDEN", "OBJECTIVES", "MOOC"} {
		if !strings.Contains(block, want) {
			t.Fatalf("prompt missing %q\n%s", want, block)
		}
	}
}

func TestReviewRemediationContractValidationErrors(t *testing.T) {
	c := DefaultReviewRemediationContract()
	c.TicketID = ""
	if err := c.Validate(); err == nil {
		t.Fatal("empty ticket")
	}
	c = DefaultReviewRemediationContract()
	c.Objectives = c.Objectives[:1]
	if err := c.Validate(); err == nil {
		t.Fatal("few objectives")
	}
	c = DefaultReviewRemediationContract()
	delete(c.CoderScopes, "evidence-coder")
	if err := c.Validate(); err == nil {
		t.Fatal("missing coder scope")
	}
}
