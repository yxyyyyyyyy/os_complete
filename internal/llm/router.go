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

type Request struct {
	AgentID  string `json:"agent_id"`
	Role     string `json:"role"`
	Provider string `json:"provider,omitempty"`
	Prompt   string `json:"prompt"`
}

type Response struct {
	Text         string `json:"text"`
	Provider     string `json:"provider"`
	FallbackFrom string `json:"fallback_from,omitempty"`
}

type Usage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
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
	resp.FallbackFrom = name
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
	return Response{Text: completion, Provider: p.name}, usage, nil
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
