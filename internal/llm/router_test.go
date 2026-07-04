package llm

import (
	"context"
	"strings"
	"testing"
)

func TestRouterCallsMockProviderAndReturnsUsage(t *testing.T) {
	router := NewRouter()
	router.Register("mock", NewMockProvider("mock"))
	router.SetDefault("mock")

	resp, usage, err := router.Complete(context.Background(), Request{
		AgentID: "planner-1",
		Role:    "planner",
		Prompt:  "Design an AORT task plan.",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "mock" {
		t.Fatalf("provider = %q", resp.Provider)
	}
	if !strings.Contains(resp.Text, "planner-1") {
		t.Fatalf("text = %q", resp.Text)
	}
	if usage.PromptTokens == 0 || usage.TotalMS == 0 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestRouterFallsBackToMockAfterProviderFailure(t *testing.T) {
	router := NewRouter()
	router.Register("broken", ProviderFunc(func(context.Context, Request) (Response, Usage, error) {
		return Response{}, Usage{}, ErrProviderUnavailable
	}))
	router.Register("mock", NewMockProvider("mock"))
	router.SetDefault("broken")
	router.SetFallback("mock")

	resp, _, err := router.Complete(context.Background(), Request{AgentID: "tester-1", Prompt: "run tests"})
	if err != nil {
		t.Fatalf("Complete with fallback: %v", err)
	}
	if resp.Provider != "mock" || resp.FallbackFrom != "broken" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestParseLlamaTimingUsageReadsCacheFields(t *testing.T) {
	usage := ParseLlamaTimingUsage(map[string]any{
		"prompt_n":         float64(120),
		"predicted_n":      float64(24),
		"cached_tokens":    float64(80),
		"prompt_ms":        float64(42),
		"time_to_first_ms": float64(15),
	})

	if usage.PromptTokens != 120 || usage.CompletionTokens != 24 || usage.CachedTokens != 80 {
		t.Fatalf("usage = %#v", usage)
	}
	if usage.TTFTMS != 15 || usage.PromptMS != 42 {
		t.Fatalf("timing = %#v", usage)
	}
}
