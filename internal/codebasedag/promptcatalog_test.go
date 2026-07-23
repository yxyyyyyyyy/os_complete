package codebasedag

import (
	"strings"
	"testing"
)

func TestDefaultPromptCatalogValidatesAndRenders(t *testing.T) {
	c := DefaultPromptCatalog()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	system, user, err := c.RenderRole("planner", map[string]string{
		"ticket":   "review-remediation",
		"manifest": "physical=30000",
		"contract": "no mooc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(system, "planner") || !strings.Contains(user, "review-remediation") {
		t.Fatalf("system=%q user=%q", system, user)
	}
	if _, _, err := c.RenderRole("planner", map[string]string{"ticket": "x"}); err == nil {
		t.Fatal("unresolved placeholders should fail")
	}
	if _, _, err := c.RenderRole("nope", nil); err == nil {
		t.Fatal("unknown role")
	}
}

func TestPromptTemplateForbiddenPhraseDetection(t *testing.T) {
	tpl := DefaultPromptCatalog().Roles["context-coder"]
	_, _, err := tpl.Render(map[string]string{
		"scope":    "x",
		"files":    "y",
		"findings": "Please enable KV cache sharing now",
	})
	if err == nil {
		t.Fatal("forbidden phrase should fail")
	}
}

func TestPromptCatalogCoversAllRoles(t *testing.T) {
	c := DefaultPromptCatalog()
	for _, role := range []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "fixer", "finalizer"} {
		tpl := c.Roles[role]
		if err := tpl.Validate(); err != nil {
			t.Fatalf("%s: %v", role, err)
		}
		if tpl.MaxTokens < 1024 {
			t.Fatalf("%s max tokens too small", role)
		}
	}
}
