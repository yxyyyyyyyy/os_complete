package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
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
	if resp.Provider != "mock" || resp.Model != "mock" {
		t.Fatalf("provider/model = %q/%q", resp.Provider, resp.Model)
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
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: ErrProviderUnavailable}
	}))
	router.Register("mock", NewMockProvider("mock"))
	router.SetDefault("broken")
	router.SetFallback("mock")

	resp, _, err := router.Complete(context.Background(), Request{AgentID: "tester-1", Prompt: "run tests"})
	if err != nil {
		t.Fatalf("Complete with fallback: %v", err)
	}
	if resp.Provider != "mock" || resp.Model != "mock" || resp.FallbackFrom != "broken" {
		t.Fatalf("response = %#v", resp)
	}
	if !resp.Fallback || resp.FallbackReason != "api_error" || resp.EvidenceMode != "mock" {
		t.Fatalf("fallback metadata = %#v", resp)
	}
}

func TestDeepSeekProviderUsesOpenAICompatibleChatCompletion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"content":"deepseek completion"}}],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`)),
		}, nil
	})}

	provider := NewDeepSeekProvider(DeepSeekConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.deepseek.test",
		Model:   "deepseek-v4-flash",
		Client:  client,
	})
	resp, usage, err := provider.Complete(context.Background(), Request{AgentID: "planner", Prompt: "plan"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "deepseek" || resp.Model != "deepseek-v4-flash" || resp.Fallback || resp.EvidenceMode != "real-api" {
		t.Fatalf("response = %#v", resp)
	}
	if resp.Text != "deepseek completion" || usage.PromptTokens != 7 || usage.CompletionTokens != 3 || usage.Mode != "real-api" {
		t.Fatalf("usage/text resp=%#v usage=%#v", resp, usage)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestDeepSeekProviderReportsNoAPIKeyWithoutLeakingSecret(t *testing.T) {
	provider := NewDeepSeekProvider(DeepSeekConfig{APIKey: "", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash"})
	_, _, err := provider.Complete(context.Background(), Request{AgentID: "planner", Prompt: "plan"})
	if err == nil {
		t.Fatal("expected no_api_key error")
	}
	var providerErr ProviderError
	if !AsProviderError(err, &providerErr) || providerErr.Reason != "no_api_key" {
		t.Fatalf("err=%#v", err)
	}
}

func TestRouterFallsBackToMockAfterDeepSeekAPIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":"temporary"}`)),
		}, nil
	})}
	router := NewRouter()
	router.Register("deepseek", NewDeepSeekProvider(DeepSeekConfig{
		APIKey: "test-key",
		Model:  "deepseek-v4-flash",
		Client: client,
	}))
	router.Register("mock", NewMockProvider("mock"))
	router.SetDefault("deepseek")
	router.SetFallback("mock")

	resp, _, err := router.Complete(context.Background(), Request{AgentID: "planner", Prompt: "plan"})
	if err != nil {
		t.Fatalf("Complete with fallback: %v", err)
	}
	if resp.Provider != "mock" || resp.Model != "mock" || resp.RequestedProvider != "deepseek" {
		t.Fatalf("response = %#v", resp)
	}
	if !resp.Fallback || resp.FallbackReason != "api_error" || resp.EvidenceMode != "mock" {
		t.Fatalf("fallback metadata = %#v", resp)
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
