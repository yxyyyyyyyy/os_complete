package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultDeepSeekModel   = "deepseek-v4-flash"
)

type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
}

type DeepSeekProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewDeepSeekProvider(cfg DeepSeekConfig) DeepSeekProvider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultDeepSeekBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultDeepSeekModel
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return DeepSeekProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   model,
		client:  client,
	}
}

func (p DeepSeekProvider) Complete(ctx context.Context, req Request) (Response, Usage, error) {
	if p.apiKey == "" {
		return Response{}, Usage{}, ProviderError{Reason: "no_api_key", Err: ErrProviderUnavailable}
	}
	start := time.Now()
	payload := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"stream": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: err}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: err}
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: err}
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek status %d", httpResp.StatusCode)}
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&decoded); err != nil {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: err}
	}
	if len(decoded.Choices) == 0 {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned no choices")}
	}
	duration := time.Since(start).Milliseconds()
	if duration == 0 {
		duration = 1
	}
	usage := Usage{
		PromptTokens:     decoded.Usage.PromptTokens,
		CompletionTokens: decoded.Usage.CompletionTokens,
		TotalMS:          duration,
		Mode:             "real-api",
	}
	return Response{
		Text:         decoded.Choices[0].Message.Content,
		Provider:     "deepseek",
		Model:        p.model,
		EvidenceMode: "real-api",
	}, usage, nil
}
