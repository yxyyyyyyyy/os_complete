package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var ErrProviderUnavailable = errors.New("llm provider unavailable")

type ProviderError struct {
	Reason string
	Err    error
}

func (e ProviderError) Error() string {
	if e.Err == nil {
		return e.Reason
	}
	if e.Reason == "" {
		return e.Err.Error()
	}
	return e.Reason + ": " + e.Err.Error()
}

func (e ProviderError) Unwrap() error {
	return e.Err
}

func AsProviderError(err error, target *ProviderError) bool {
	return errors.As(err, target)
}

type Request struct {
	AgentID  string `json:"agent_id"`
	Role     string `json:"role"`
	Provider string `json:"provider,omitempty"`
	Prompt   string `json:"prompt"`
}

type Response struct {
	RequestID         string `json:"request_id,omitempty"`
	Text              string `json:"text"`
	Provider          string `json:"provider"`
	Model             string `json:"model,omitempty"`
	RequestedProvider string `json:"requested_provider,omitempty"`
	Fallback          bool   `json:"fallback"`
	FallbackFrom      string `json:"fallback_from,omitempty"`
	FallbackReason    string `json:"fallback_reason,omitempty"`
	EvidenceMode      string `json:"evidence_mode"`
}

type Usage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	PromptMS         int64  `json:"prompt_ms"`
	TTFTMS           int64  `json:"ttft_ms"`
	TotalMS          int64  `json:"total_ms"`
	Mode             string `json:"mode"`
}

type Provider interface {
	Complete(context.Context, Request) (Response, Usage, error)
}

type ProviderFunc func(context.Context, Request) (Response, Usage, error)

func (fn ProviderFunc) Complete(ctx context.Context, req Request) (Response, Usage, error) {
	return fn(ctx, req)
}

type Router struct {
	mu               sync.RWMutex
	providers        map[string]Provider
	roleRoute        map[string]string
	defaultProvider  string
	fallbackProvider string
}

func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
		roleRoute: make(map[string]string),
	}
}

func (r *Router) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

func (r *Router) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultProvider = name
}

func (r *Router) SetFallback(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbackProvider = name
}

func (r *Router) SetRoleProvider(role, provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roleRoute[strings.ToLower(role)] = provider
}

func (r *Router) Complete(ctx context.Context, req Request) (Response, Usage, error) {
	name, provider, fallbackName, fallback := r.selectProvider(req)
	if provider == nil {
		return Response{}, Usage{}, fmt.Errorf("%w: %s", ErrProviderUnavailable, name)
	}
	resp, usage, err := provider.Complete(ctx, req)
	if err == nil {
		resp.Provider = valueOr(resp.Provider, name)
		resp.EvidenceMode = valueOr(resp.EvidenceMode, usage.Mode)
		if resp.EvidenceMode == "" {
			resp.EvidenceMode = "real-api"
		}
		return resp, usage, nil
	}
	if fallback == nil || fallbackName == "" || fallbackName == name {
		return Response{}, Usage{}, err
	}
	resp, usage, fallbackErr := fallback.Complete(ctx, req)
	if fallbackErr != nil {
		return Response{}, Usage{}, fallbackErr
	}
	resp.Provider = valueOr(resp.Provider, fallbackName)
	resp.RequestedProvider = name
	resp.FallbackFrom = name
	resp.Fallback = true
	resp.FallbackReason = providerErrorReason(err)
	resp.EvidenceMode = "mock"
	return resp, usage, nil
}

func (r *Router) selectProvider(req Request) (string, Provider, string, Provider) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name := req.Provider
	if name == "" && req.Role != "" {
		name = r.roleRoute[strings.ToLower(req.Role)]
	}
	if name == "" {
		name = r.defaultProvider
	}
	return name, r.providers[name], r.fallbackProvider, r.providers[r.fallbackProvider]
}

func valueOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func providerErrorReason(err error) string {
	var providerErr ProviderError
	if errors.As(err, &providerErr) && providerErr.Reason != "" {
		return providerErr.Reason
	}
	return "api_error"
}

type MockProvider struct {
	name string
}

func NewMockProvider(name string) MockProvider {
	if name == "" {
		name = "mock"
	}
	return MockProvider{name: name}
}

func (p MockProvider) Complete(ctx context.Context, req Request) (Response, Usage, error) {
	select {
	case <-ctx.Done():
		return Response{}, Usage{}, ctx.Err()
	default:
	}
	start := time.Now()
	promptTokens := estimateTokens(req.Prompt)
	completion := fmt.Sprintf("mock completion for %s: runtime syscall accepted", req.AgentID)
	usage := Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: estimateTokens(completion),
		CachedTokens:     promptTokens / 2,
		PromptMS:         1,
		TTFTMS:           1,
		TotalMS:          maxInt64(1, time.Since(start).Milliseconds()),
		Mode:             "mock",
	}
	return Response{Text: completion, Provider: p.name, Model: p.name, EvidenceMode: "mock"}, usage, nil
}

func ParseLlamaTimingUsage(raw map[string]any) Usage {
	return Usage{
		PromptTokens:     number(raw["prompt_n"]),
		CompletionTokens: number(raw["predicted_n"]),
		CachedTokens:     number(raw["cached_tokens"]),
		PromptMS:         int64(number(raw["prompt_ms"])),
		TTFTMS:           int64(number(raw["time_to_first_ms"])),
		TotalMS:          int64(number(raw["total_ms"])),
		Mode:             "llamacpp-local",
	}
}

func estimateTokens(content string) int {
	tokens := len([]rune(content)) / 4
	if tokens == 0 && content != "" {
		return 1
	}
	return tokens
}

func number(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	default:
		return 0
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
