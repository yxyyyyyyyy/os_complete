package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDeepSeekProviderLiveFromEnv(t *testing.T) {
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

	provider := NewDeepSeekProvider(DeepSeekConfig{
		APIKey:  key,
		BaseURL: baseURL,
		Model:   model,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := provider.ValidateModel(ctx); err != nil {
		t.Fatalf("ValidateModel: %v", err)
	}

	router := NewRouter()
	router.Register("deepseek", provider)
	router.SetDefault("deepseek")

	resp, usage, err := router.Complete(ctx, Request{
		AgentID: "deepseek-live-smoke",
		Role:    "tester",
		Prompt:  "Reply with the single word ok.",
	})
	if err != nil {
		t.Fatalf("DeepSeek live call failed: %v", err)
	}
	if resp.Provider != "deepseek" || resp.Model != model || resp.Fallback || resp.EvidenceMode != "real-api" || usage.Mode != "real-api" {
		t.Fatalf("bad DeepSeek real-api response resp=%#v usage=%#v", resp, usage)
	}
	if resp.RequestID == "" || usage.PromptTokens == 0 || usage.CompletionTokens == 0 || usage.TotalTokens == 0 || usage.TotalMS == 0 {
		t.Fatalf("missing DeepSeek usage evidence resp=%#v usage=%#v", resp, usage)
	}
}
