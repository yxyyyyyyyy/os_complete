package codebasedag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"aort-r/internal/llm"
)

const (
	RequiredDeepSeekProvider = "deepseek"
	RequiredDeepSeekModel    = "deepseek-v4-flash"
	DefaultMaxModelCalls     = 10
)

type modelValidator interface {
	ValidateModel(context.Context) error
}

type StrictModelOptions struct {
	RequiredModel string
	MaxCalls      int
}

type ModelRequest struct {
	NodeID string
	Role   string
	Prompt string
}

type StrictModel struct {
	provider  llm.Provider
	validator modelValidator
	ledger    *CallLedger
}

func NewStrictModel(provider llm.Provider, opts StrictModelOptions) (*StrictModel, error) {
	if provider == nil {
		return nil, fmt.Errorf("strict model provider is nil")
	}
	requiredModel := opts.RequiredModel
	if requiredModel == "" {
		requiredModel = RequiredDeepSeekModel
	}
	maxCalls := opts.MaxCalls
	if maxCalls == 0 {
		maxCalls = DefaultMaxModelCalls
	}
	if maxCalls < 1 || maxCalls > DefaultMaxModelCalls {
		return nil, fmt.Errorf("max calls %d outside allowed range 1-%d", maxCalls, DefaultMaxModelCalls)
	}
	model := &StrictModel{
		provider: provider,
		ledger:   NewCallLedger(requiredModel, maxCalls),
	}
	if validator, ok := provider.(modelValidator); ok {
		model.validator = validator
	}
	return model, nil
}

func (m *StrictModel) Validate(ctx context.Context) error {
	if m.validator == nil {
		return fmt.Errorf("provider does not expose ValidateModel")
	}
	return m.validator.ValidateModel(ctx)
}

func (m *StrictModel) Complete(ctx context.Context, req ModelRequest) (string, CallRecord, error) {
	attemptID, err := m.ledger.Begin(req.NodeID, req.Role)
	if err != nil {
		return "", CallRecord{}, err
	}
	start := time.Now()
	resp, usage, err := m.provider.Complete(ctx, llm.Request{
		AgentID:  req.NodeID,
		Role:     req.Role,
		Provider: RequiredDeepSeekProvider,
		Prompt:   req.Prompt,
	})
	if err != nil {
		_ = m.ledger.Fail(attemptID, err)
		return "", CallRecord{}, err
	}
	sum := sha256.Sum256([]byte(resp.Text))
	record := CallRecord{
		SchemaVersion:    SchemaVersion,
		CallID:           resp.RequestID,
		NodeID:           req.NodeID,
		Role:             req.Role,
		Provider:         resp.Provider,
		RequestedModel:   m.ledger.RequiredModel,
		ActualModel:      resp.Model,
		EvidenceMode:     resp.EvidenceMode,
		Fallback:         resp.Fallback,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		DurationMS:       maxInt64(1, time.Since(start).Milliseconds()),
		OutputSHA256:     hex.EncodeToString(sum[:]),
		Status:           "succeeded",
		CreatedAt:        time.Now().UTC(),
	}
	if err := m.ledger.Finish(attemptID, record); err != nil {
		return "", CallRecord{}, err
	}
	return resp.Text, record, nil
}

func (m *StrictModel) Records() []CallRecord {
	return m.ledger.Snapshot()
}

type CallLedger struct {
	RequiredModel string
	MaxCalls      int
	mu            sync.Mutex
	Records       []CallRecord
	open          map[string]CallRecord
	seenCallIDs   map[string]struct{}
	nextAttempt   int
}

func NewCallLedger(requiredModel string, maxCalls int) *CallLedger {
	if requiredModel == "" {
		requiredModel = RequiredDeepSeekModel
	}
	if maxCalls == 0 {
		maxCalls = DefaultMaxModelCalls
	}
	return &CallLedger{
		RequiredModel: requiredModel,
		MaxCalls:      maxCalls,
		open:          make(map[string]CallRecord),
		seenCallIDs:   make(map[string]struct{}),
	}
}

func (l *CallLedger) Begin(nodeID, role string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.Records)+len(l.open) >= l.MaxCalls {
		return "", fmt.Errorf("call budget exhausted: max %d", l.MaxCalls)
	}
	l.nextAttempt++
	attemptID := fmt.Sprintf("attempt-%d", l.nextAttempt)
	l.open[attemptID] = CallRecord{
		SchemaVersion:  SchemaVersion,
		NodeID:         nodeID,
		Role:           role,
		RequestedModel: l.RequiredModel,
		Status:         "running",
		CreatedAt:      time.Now().UTC(),
	}
	return attemptID, nil
}

func (l *CallLedger) Finish(attemptID string, record CallRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.open[attemptID]; !ok {
		return fmt.Errorf("unknown or closed attempt %q", attemptID)
	}
	if err := l.validateRecord(record); err != nil {
		delete(l.open, attemptID)
		return err
	}
	if _, ok := l.seenCallIDs[record.CallID]; ok {
		delete(l.open, attemptID)
		return fmt.Errorf("duplicate API call ID %q", record.CallID)
	}
	l.seenCallIDs[record.CallID] = struct{}{}
	delete(l.open, attemptID)
	l.Records = append(l.Records, record)
	return nil
}

func (l *CallLedger) Fail(attemptID string, callErr error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	record, ok := l.open[attemptID]
	if !ok {
		return fmt.Errorf("unknown or closed attempt %q", attemptID)
	}
	record.Status = "failed"
	if callErr != nil {
		record.Error = sanitizeEvidenceError(callErr.Error())
	}
	delete(l.open, attemptID)
	l.Records = append(l.Records, record)
	return nil
}

func (l *CallLedger) Snapshot() []CallRecord {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]CallRecord(nil), l.Records...)
}

func (l *CallLedger) validateRecord(record CallRecord) error {
	if record.CallID == "" {
		return fmt.Errorf("missing API call ID")
	}
	if record.Provider != RequiredDeepSeekProvider {
		return fmt.Errorf("provider %q is not %q", record.Provider, RequiredDeepSeekProvider)
	}
	if record.ActualModel != l.RequiredModel {
		return fmt.Errorf("model %q is not required model %q", record.ActualModel, l.RequiredModel)
	}
	if record.EvidenceMode != "real-api" {
		return fmt.Errorf("evidence mode %q is not real-api", record.EvidenceMode)
	}
	if record.Fallback {
		return fmt.Errorf("fallback calls are forbidden")
	}
	if record.PromptTokens <= 0 || record.CompletionTokens <= 0 || record.TotalTokens <= 0 {
		return fmt.Errorf("usage tokens must be positive")
	}
	if record.OutputSHA256 == "" {
		return fmt.Errorf("output hash is required")
	}
	return nil
}

func sanitizeEvidenceError(text string) string {
	if len(text) > 512 {
		return text[:512]
	}
	return text
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
