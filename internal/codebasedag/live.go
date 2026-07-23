package codebasedag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"aort-r/internal/llm"
)

// LiveNodeExecutor runs LLM nodes through StrictModel and tool-only nodes
// (currently integrate) without consuming DeepSeek budget.
type LiveNodeExecutor struct {
	Model  *StrictModel
	Ticket Ticket
	Store  *RunStore
	Clock  func() time.Time
}

func NewLiveNodeExecutor(model *StrictModel, ticket Ticket, store *RunStore) LiveNodeExecutor {
	return LiveNodeExecutor{Model: model, Ticket: ticket, Store: store, Clock: time.Now}
}

func (e LiveNodeExecutor) ExecuteNode(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	switch req.NodeID {
	case "integrate":
		return e.executeIntegrate(req)
	default:
		return e.executeLLM(ctx, req)
	}
}

func (e LiveNodeExecutor) executeIntegrate(req NodeExecutionRequest) (NodeExecutionResult, error) {
	payload := []byte(fmt.Sprintf(`{"schema_version":%q,"node_id":"integrate","status":"merged","dependencies":%q}`, SchemaVersion, strings.Join(req.Dependencies, ",")))
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	if e.Store != nil {
		if err := e.Store.WriteBytes("outputs/integrate.json", payload, 0o600); err != nil {
			return NodeExecutionResult{}, err
		}
	}
	return NodeExecutionResult{OutputSHA256: hash}, nil
}

func (e LiveNodeExecutor) executeLLM(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	if e.Model == nil {
		return NodeExecutionResult{}, fmt.Errorf("strict model is required")
	}
	policy, ok := e.Ticket.NodePolicies[req.NodeID]
	if !ok {
		return NodeExecutionResult{}, fmt.Errorf("node policy for %q is missing", req.NodeID)
	}
	prompt, err := BuildPrompt(PromptRequest{
		NodeID:          req.NodeID,
		Role:            policy.Role,
		Ticket:          e.Ticket.ID,
		AllowedFiles:    policy.AllowedFiles,
		ImmutableFiles:  policy.ImmutableFiles,
		SharedContextID: e.Ticket.SharedContext,
		PrivateContext:  policy.PrivateContext,
	})
	if err != nil {
		return NodeExecutionResult{}, err
	}
	text, record, err := e.Model.Complete(ctx, ModelRequest{NodeID: req.NodeID, Role: string(policy.Role), Prompt: prompt})
	if err != nil {
		return NodeExecutionResult{}, err
	}
	if e.Store != nil {
		if err := e.Store.WriteBytes(fmt.Sprintf("outputs/%s.json", req.NodeID), []byte(text), 0o600); err != nil {
			return NodeExecutionResult{}, err
		}
		if err := e.Store.AppendJSONL("llm_calls.jsonl", record); err != nil {
			return NodeExecutionResult{}, err
		}
	}
	return NodeExecutionResult{OutputSHA256: record.OutputSHA256, LLMCallID: record.CallID}, nil
}

// LiveRunConfig configures an Open World live codebase-dag execution.
type LiveRunConfig struct {
	RunnerConfig
	OutDir       string
	Ticket       string
	MinPhysical  int
	MinNonblank  int
	GitPath      string
	APIKey       string
	BaseURL      string
	JournalPath  string
	RequireKey   bool
}

type LiveRunResult struct {
	Summary Summary       `json:"summary"`
	Calls   []CallRecord  `json:"calls"`
	Dir     string        `json:"dir"`
}

func RunLive(ctx context.Context, cfg LiveRunConfig) (LiveRunResult, error) {
	if cfg.OutDir == "" {
		return LiveRunResult{}, fmt.Errorf("out dir is required")
	}
	if cfg.Ticket == "" {
		cfg.Ticket = "review-remediation"
	}
	if cfg.Ticket != "review-remediation" {
		return LiveRunResult{}, fmt.Errorf("unsupported ticket %q", cfg.Ticket)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return LiveRunResult{}, fmt.Errorf("DEEPSEEK_API_KEY is required for live codebase-dag")
	}
	prevKey := os.Getenv("DEEPSEEK_API_KEY")
	if err := os.Setenv("DEEPSEEK_API_KEY", cfg.APIKey); err != nil {
		return LiveRunResult{}, err
	}
	defer func() { _ = os.Setenv("DEEPSEEK_API_KEY", prevKey) }()
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("DEEPSEEK_BASE_URL")
	}
	if cfg.Model == "" {
		cfg.Model = RequiredDeepSeekModel
	}
	if cfg.Provider == "" {
		cfg.Provider = RequiredDeepSeekProvider
	}
	if cfg.MaxCalls == 0 {
		cfg.MaxCalls = DefaultMaxModelCalls
	}

	store, err := NewRunStore(cfg.OutDir, cfg.RunID)
	if err != nil {
		return LiveRunResult{}, err
	}
	journalPath := cfg.JournalPath
	if journalPath == "" {
		journalPath = store.Dir + "/process_journal.jsonl"
	}
	journal, err := NewEvidenceJournal(journalPath)
	if err != nil {
		return LiveRunResult{}, err
	}
	defer journal.Close()

	provider := llm.NewDeepSeekProvider(llm.DeepSeekConfig{
		APIKey:    cfg.APIKey,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		MaxTokens: 8192,
		Timeout:   180 * time.Second,
	})
	model, err := NewStrictModel(provider, StrictModelOptions{RequiredModel: cfg.Model, MaxCalls: cfg.MaxCalls})
	if err != nil {
		return LiveRunResult{}, err
	}
	if err := model.Validate(ctx); err != nil {
		return LiveRunResult{}, fmt.Errorf("deepseek model gate: %w", err)
	}

	preflight := LocalPreflight{
		ManifestOptions: ManifestOptions{MinPhysical: cfg.MinPhysical, MinNonblank: cfg.MinNonblank, GitPath: cfg.GitPath},
		RequireAPIKey:   true,
	}
	ticket := ReviewRemediationTicket()
	executor := NewLiveNodeExecutor(model, ticket, store)
	summary, runErr := Run(ctx, cfg.RunnerConfig, RunnerDeps{
		Preflight:    preflight,
		NodeExecutor: executor,
		Journal:      journal,
		Clock:        time.Now,
	})

	calls := model.Records()
	_ = store.WriteJSON("llm_calls.json", calls)
	_ = store.WriteJSON("summary.json", summary)
	_ = store.WriteJSON("preflight.json", summary.Preflight)
	if _, hashErr := store.FinalizeHashes(); hashErr != nil && runErr == nil {
		runErr = hashErr
	}
	result := LiveRunResult{Summary: summary, Calls: calls, Dir: store.Dir}
	if runErr != nil {
		return result, runErr
	}
	if len(calls) < DefaultRequiredMinCalls {
		return result, fmt.Errorf("insufficient successful real calls: got %d want >= %d", len(calls), DefaultRequiredMinCalls)
	}
	for _, call := range calls {
		if call.Provider != RequiredDeepSeekProvider || call.ActualModel != RequiredDeepSeekModel || call.Fallback || call.EvidenceMode != "real-api" {
			return result, fmt.Errorf("non-real DeepSeek call recorded: %#v", call)
		}
	}
	return result, nil
}
