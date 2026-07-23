package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultDeepSeekModel   = "deepseek-v4-flash"
	defaultDeepSeekMaxTok  = 4096
	defaultDeepSeekTimeout = 180 * time.Second
)

type DeepSeekConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	Client      *http.Client
}

type DeepSeekProvider struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
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
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultDeepSeekMaxTok
	}
	client := cfg.Client
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultDeepSeekTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	return DeepSeekProvider{
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		client:      client,
	}
}

func (p DeepSeekProvider) ValidateModel(ctx context.Context) error {
	if p.apiKey == "" {
		return ProviderError{Reason: "no_api_key", Err: ErrProviderUnavailable}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return ProviderError{Reason: "api_error", Err: err}
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return ProviderError{Reason: "api_error", Err: err}
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek models status %d", httpResp.StatusCode)}
	}
	limited := io.LimitReader(httpResp.Body, 8<<20)
	var decoded struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(limited).Decode(&decoded); err != nil {
		return ProviderError{Reason: "api_error", Err: err}
	}
	for _, item := range decoded.Data {
		if item.ID == p.model {
			return nil
		}
	}
	return fmt.Errorf("required model %q is unavailable", p.model)
}

func (p DeepSeekProvider) Complete(ctx context.Context, req Request) (Response, Usage, error) {
	if p.apiKey == "" {
		return Response{}, Usage{}, ProviderError{Reason: "no_api_key", Err: ErrProviderUnavailable}
	}
	var lastErr error
	start := time.Now()
	for attempt := 1; attempt <= 3; attempt++ {
		resp, usage, err := p.completeOnce(ctx, req)
		if err == nil {
			if attempt > 1 {
				usage.TotalMS = time.Since(start).Milliseconds()
				if usage.TotalMS == 0 {
					usage.TotalMS = 1
				}
			}
			return resp, usage, nil
		}
		lastErr = err
		if !isRetryableDeepSeekEmpty(err) || attempt == 3 {
			break
		}
		select {
		case <-ctx.Done():
			return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: ctx.Err()}
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	return Response{}, Usage{}, lastErr
}

func isRetryableDeepSeekEmpty(err error) bool {
	pe, ok := err.(ProviderError)
	if !ok {
		return false
	}
	if pe.Err == nil {
		return false
	}
	msg := pe.Err.Error()
	return strings.Contains(msg, "no choices") || strings.Contains(msg, "empty content")
}

func (p DeepSeekProvider) completeOnce(ctx context.Context, req Request) (Response, Usage, error) {
	start := time.Now()
	payload := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"stream":      false,
		"max_tokens":  p.maxTokens,
		"temperature": p.temperature,
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
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
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
	if decoded.ID == "" {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned no request id")}
	}
	if decoded.Model == "" {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned no model")}
	}
	if len(decoded.Choices) == 0 {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned no choices")}
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned empty content (finish_reason=%s reasoning_len=%d)", decoded.Choices[0].FinishReason, len(decoded.Choices[0].Message.ReasoningContent))}
	}
	if decoded.Usage.PromptTokens <= 0 || decoded.Usage.CompletionTokens <= 0 || decoded.Usage.TotalTokens <= 0 {
		return Response{}, Usage{}, ProviderError{Reason: "api_error", Err: fmt.Errorf("deepseek returned incomplete usage")}
	}
	duration := time.Since(start).Milliseconds()
	if duration == 0 {
		duration = 1
	}
	usage := Usage{
		PromptTokens:     decoded.Usage.PromptTokens,
		CompletionTokens: decoded.Usage.CompletionTokens,
		TotalTokens:      decoded.Usage.TotalTokens,
		TotalMS:          duration,
		Mode:             "real-api",
	}
	return Response{
		RequestID:    decoded.ID,
		Text:         content,
		Provider:     "deepseek",
		Model:        decoded.Model,
		EvidenceMode: "real-api",
	}, usage, nil
}
