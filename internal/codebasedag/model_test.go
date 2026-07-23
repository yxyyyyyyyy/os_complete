package codebasedag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aort-r/internal/llm"
)

func TestStrictModelValidatesModelAndRecordsSuccessfulCalls(t *testing.T) {
	provider := &fakeStrictProvider{
		response: llm.Response{
			RequestID:    "api-call-1",
			Text:         `{"schema_version":"codebase-dag/v1","node_id":"planner"}`,
			Provider:     "deepseek",
			Model:        RequiredDeepSeekModel,
			EvidenceMode: "real-api",
		},
		usage: llm.Usage{PromptTokens: 11, CompletionTokens: 7, TotalTokens: 18, Mode: "real-api"},
	}
	model, err := NewStrictModel(provider, StrictModelOptions{MaxCalls: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if provider.validateCalls != 1 {
		t.Fatalf("ValidateModel calls = %d", provider.validateCalls)
	}

	text, record, err := model.Complete(context.Background(), ModelRequest{
		NodeID: "planner",
		Role:   "planner",
		Prompt: "return json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != provider.response.Text {
		t.Fatalf("text = %q", text)
	}
	if record.CallID != "api-call-1" || record.ActualModel != RequiredDeepSeekModel || record.TotalTokens != 18 {
		t.Fatalf("record = %#v", record)
	}
	if record.OutputSHA256 == "" {
		t.Fatal("output hash must be recorded")
	}
}

func TestStrictModelRejectsMockFallbackWrongModelAndBadUsage(t *testing.T) {
	cases := []struct {
		name     string
		response llm.Response
		usage    llm.Usage
		want     string
	}{
		{
			name: "wrong provider",
			response: llm.Response{
				RequestID: "call-1", Text: "x", Provider: "mock", Model: RequiredDeepSeekModel, EvidenceMode: "real-api",
			},
			usage: llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, Mode: "real-api"},
			want:  "provider",
		},
		{
			name: "wrong model",
			response: llm.Response{
				RequestID: "call-1", Text: "x", Provider: "deepseek", Model: "deepseek-chat", EvidenceMode: "real-api",
			},
			usage: llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, Mode: "real-api"},
			want:  "model",
		},
		{
			name: "fallback",
			response: llm.Response{
				RequestID: "call-1", Text: "x", Provider: "deepseek", Model: RequiredDeepSeekModel, EvidenceMode: "real-api", Fallback: true,
			},
			usage: llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, Mode: "real-api"},
			want:  "fallback",
		},
		{
			name: "bad usage",
			response: llm.Response{
				RequestID: "call-1", Text: "x", Provider: "deepseek", Model: RequiredDeepSeekModel, EvidenceMode: "real-api",
			},
			usage: llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 0, Mode: "real-api"},
			want:  "usage",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model, err := NewStrictModel(&fakeStrictProvider{response: tc.response, usage: tc.usage}, StrictModelOptions{MaxCalls: 10})
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = model.Complete(context.Background(), ModelRequest{NodeID: "node", Role: "role", Prompt: "prompt"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want contains %q", err, tc.want)
			}
		})
	}
}

func TestStrictModelConsumesBudgetBeforeProviderCallAndStopsAtMax(t *testing.T) {
	provider := &fakeStrictProvider{
		response: llm.Response{
			RequestID:    "call",
			Text:         "ok",
			Provider:     "deepseek",
			Model:        RequiredDeepSeekModel,
			EvidenceMode: "real-api",
		},
		usage: llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2, Mode: "real-api"},
	}
	model, err := NewStrictModel(provider, StrictModelOptions{MaxCalls: 2})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, _, err := model.Complete(context.Background(), ModelRequest{NodeID: "n", Role: "r", Prompt: "p"}); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err = model.Complete(context.Background(), ModelRequest{NodeID: "n", Role: "r", Prompt: "p"})
	if err == nil || !strings.Contains(err.Error(), "call budget") {
		t.Fatalf("budget error = %v", err)
	}
	if provider.completeCalls != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.completeCalls)
	}
}

func TestCallLedgerRejectsDuplicateCallID(t *testing.T) {
	ledger := NewCallLedger(RequiredDeepSeekModel, 10)
	first, err := ledger.Begin("planner", "planner")
	if err != nil {
		t.Fatal(err)
	}
	record := CallRecord{
		CallID:           "dup",
		NodeID:           "planner",
		Role:             "planner",
		Provider:         "deepseek",
		RequestedModel:   RequiredDeepSeekModel,
		ActualModel:      RequiredDeepSeekModel,
		EvidenceMode:     "real-api",
		PromptTokens:     1,
		CompletionTokens: 1,
		TotalTokens:      2,
		OutputSHA256:     strings.Repeat("a", 64),
		Status:           "succeeded",
	}
	if err := ledger.Finish(first, record); err != nil {
		t.Fatal(err)
	}
	second, err := ledger.Begin("tester", "tester")
	if err != nil {
		t.Fatal(err)
	}
	record.NodeID = "tester"
	record.Role = "tester"
	if err := ledger.Finish(second, record); err == nil {
		t.Fatal("duplicate API call ID should fail")
	}
}

type fakeStrictProvider struct {
	response      llm.Response
	usage         llm.Usage
	err           error
	validateErr   error
	validateCalls int
	completeCalls int
}

func (p *fakeStrictProvider) ValidateModel(context.Context) error {
	p.validateCalls++
	return p.validateErr
}

func (p *fakeStrictProvider) Complete(context.Context, llm.Request) (llm.Response, llm.Usage, error) {
	p.completeCalls++
	if p.err != nil {
		return llm.Response{}, llm.Usage{}, p.err
	}
	if strings.HasPrefix(p.response.RequestID, "call") {
		p.response.RequestID = "call-" + string(rune('0'+p.completeCalls))
	}
	return p.response, p.usage, nil
}

var errFakeProvider = errors.New("fake provider error")
