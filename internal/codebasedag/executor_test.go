package codebasedag

import (
	"context"
	"strings"
	"testing"

	"aort-r/internal/llm"
)

func TestModelNodeExecutorBuildsPromptsAndConsumesLedger(t *testing.T) {
	provider := &fakeStrictProvider{
		response: llm.Response{
			RequestID:    "call",
			Text:         validCoderJSON,
			Provider:     RequiredDeepSeekProvider,
			Model:        RequiredDeepSeekModel,
			EvidenceMode: "real-api",
		},
		usage: llm.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8, Mode: "real-api"},
	}
	model, err := NewStrictModel(provider, StrictModelOptions{MaxCalls: 10})
	if err != nil {
		t.Fatal(err)
	}
	executor := NewModelNodeExecutor(model, Ticket{
		ID:            "review-remediation",
		SharedContext: "sha256:ctx",
		NodePolicies: map[string]NodePolicy{
			"resource-coder": {
				Role:           KindCoder,
				AllowedFiles:   []string{"internal/review/resource.go"},
				ImmutableFiles: []string{"internal/codebasedag/acceptance/resource_real.sh"},
				PrivateContext: "resource evidence",
			},
		},
	})
	result, err := executor.ExecuteNode(context.Background(), NodeExecutionRequest{RunID: "run", NodeID: "resource-coder"})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputSHA256 == "" || result.LLMCallID == "" {
		t.Fatalf("result = %#v", result)
	}
	if provider.completeCalls != 1 {
		t.Fatalf("provider calls = %d", provider.completeCalls)
	}
	if got := model.Records(); len(got) != 1 || got[0].TotalTokens != 8 {
		t.Fatalf("records = %#v", got)
	}
}

func TestModelNodeExecutorRejectsUnknownPolicyAndBudgetExhaustion(t *testing.T) {
	model, err := NewStrictModel(&fakeStrictProvider{
		response: llm.Response{RequestID: "call", Text: "{}", Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, EvidenceMode: "real-api"},
		usage:    llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, Mode: "real-api"},
	}, StrictModelOptions{MaxCalls: 1})
	if err != nil {
		t.Fatal(err)
	}
	executor := NewModelNodeExecutor(model, Ticket{ID: "ticket", SharedContext: "ctx", NodePolicies: map[string]NodePolicy{
		"planner": {Role: KindPlanner},
	}})
	if _, err := executor.ExecuteNode(context.Background(), NodeExecutionRequest{RunID: "run", NodeID: "missing"}); err == nil || !strings.Contains(err.Error(), "policy") {
		t.Fatalf("missing policy error = %v", err)
	}
	if _, err := executor.ExecuteNode(context.Background(), NodeExecutionRequest{RunID: "run", NodeID: "planner"}); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.ExecuteNode(context.Background(), NodeExecutionRequest{RunID: "run", NodeID: "planner"}); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("budget error = %v", err)
	}
}
