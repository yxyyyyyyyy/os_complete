package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDeepSeekProviderLiveOrFallbackFromEnv(t *testing.T) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("DEEPSEEK_API_KEY is not set")
	}
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-flash"
	}

	router := NewRouter()
	router.Register("deepseek", NewDeepSeekProvider(DeepSeekConfig{
		APIKey:  key,
		BaseURL: baseURL,
		Model:   model,
	}))
	router.Register("mock", NewMockProvider("mock"))
	router.SetDefault("deepseek")
	router.SetFallback("mock")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	resp, usage, err := router.Complete(ctx, Request{
		AgentID: "deepseek-live-smoke",
		Role:    "tester",
		Prompt:  "Reply with the single word ok.",
	})
	if err != nil {
		t.Fatalf("DeepSeek router call should return real response or mock fallback: %v", err)
	}

	switch resp.Provider {
	case "deepseek":
		if resp.Model != model || resp.Fallback || resp.EvidenceMode != "real-api" || usage.Mode != "real-api" {
			t.Fatalf("bad DeepSeek real-api response resp=%#v usage=%#v", resp, usage)
		}
		if usage.PromptTokens == 0 || usage.CompletionTokens == 0 || usage.TotalMS == 0 {
			t.Fatalf("missing DeepSeek usage evidence resp=%#v usage=%#v", resp, usage)
		}
	case "mock":
		if resp.RequestedProvider != "deepseek" || !resp.Fallback || resp.FallbackReason == "" || resp.EvidenceMode != "mock" {
			t.Fatalf("bad DeepSeek fallback response resp=%#v usage=%#v", resp, usage)
		}
	default:
		t.Fatalf("unexpected provider response resp=%#v usage=%#v", resp, usage)
	}
}
