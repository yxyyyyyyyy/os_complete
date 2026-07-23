package codebasedag

import (
	"context"
	"fmt"
	"time"
)

type RunnerConfig struct {
	RunID       string
	WorkloadDir string
	Provider    string
	Model       string
	MaxCalls    int
}

type PreflightResult struct {
	Passed   bool            `json:"passed"`
	Gates    map[string]bool `json:"gates,omitempty"`
	Manifest SourceManifest  `json:"manifest,omitempty"`
}

type Preflight interface {
	Check(context.Context, RunnerConfig) (PreflightResult, error)
}

type PreflightFunc func(context.Context, RunnerConfig) (PreflightResult, error)

func (fn PreflightFunc) Check(ctx context.Context, cfg RunnerConfig) (PreflightResult, error) {
	return fn(ctx, cfg)
}

type NodeExecutionRequest struct {
	RunID        string
	NodeID       string
	Dependencies []string
}

type NodeExecutionResult struct {
	OutputSHA256 string
	LLMCallID    string
}

type NodeExecutor interface {
	ExecuteNode(context.Context, NodeExecutionRequest) (NodeExecutionResult, error)
}

type NodeExecutorFunc func(context.Context, NodeExecutionRequest) (NodeExecutionResult, error)

func (fn NodeExecutorFunc) ExecuteNode(ctx context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
	return fn(ctx, req)
}

type RunnerDeps struct {
	Preflight    Preflight
	NodeExecutor NodeExecutor
	Clock        func() time.Time
	Journal      *EvidenceJournal
}

type Summary struct {
	SchemaVersion     string          `json:"schema_version"`
	RunID             string          `json:"run_id"`
	Preflight         PreflightResult `json:"preflight"`
	Nodes             []NodeState     `json:"nodes"`
	AllRequiredPassed bool            `json:"all_required_passed"`
}

func Run(ctx context.Context, cfg RunnerConfig, deps RunnerDeps) (Summary, error) {
	if err := validateRunnerConfig(cfg); err != nil {
		return Summary{}, err
	}
	if deps.Preflight == nil {
		return Summary{}, fmt.Errorf("preflight dependency is required")
	}
	if deps.NodeExecutor == nil {
		return Summary{}, fmt.Errorf("node executor dependency is required")
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}

	graph := NewCodebaseGraph()
	if err := graph.Validate(); err != nil {
		return Summary{}, err
	}
	state := NewExecutionState(graph.Nodes())
	preflight, err := deps.Preflight.Check(ctx, cfg)
	_ = deps.Journal.Append(EvidenceEvent{RunID: cfg.RunID, Type: EventPreflight, At: deps.Clock(), Fields: preflightGateFields(preflight)})
	if err != nil {
		_ = state.Transition("preflight", NodeFailed, TransitionEvidence{Reason: err.Error(), At: deps.Clock()})
		return summaryFromState(cfg.RunID, preflight, state, false), err
	}
	if !preflight.Passed {
		_ = state.Transition("preflight", NodeFailed, TransitionEvidence{Reason: "preflight gates failed", At: deps.Clock()})
		return summaryFromState(cfg.RunID, preflight, state, false), fmt.Errorf("preflight gates failed")
	}
	if err := state.Transition("preflight", NodeReady, TransitionEvidence{Reason: "preflight complete", At: deps.Clock()}); err != nil {
		return Summary{}, err
	}
	if err := state.Transition("preflight", NodeRunning, TransitionEvidence{Reason: "preflight complete", At: deps.Clock()}); err != nil {
		return Summary{}, err
	}
	if err := state.Transition("preflight", NodeSucceeded, TransitionEvidence{Reason: "preflight complete", At: deps.Clock()}); err != nil {
		return Summary{}, err
	}

	for {
		ready := graph.Ready(state.Completed())
		if len(ready) == 0 {
			break
		}
		for _, nodeID := range ready {
			if nodeID == "preflight" {
				continue
			}
			_ = deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeStart, "dependencies complete", deps.Clock())
			if err := state.Transition(nodeID, NodeReady, TransitionEvidence{Reason: "dependencies complete", At: deps.Clock()}); err != nil {
				return summaryFromState(cfg.RunID, preflight, state, false), err
			}
			if err := state.Transition(nodeID, NodeRunning, TransitionEvidence{Reason: "node dispatched", At: deps.Clock()}); err != nil {
				return summaryFromState(cfg.RunID, preflight, state, false), err
			}
			result, err := deps.NodeExecutor.ExecuteNode(ctx, NodeExecutionRequest{
				RunID:        cfg.RunID,
				NodeID:       nodeID,
				Dependencies: graph.Dependencies(nodeID),
			})
			if err != nil {
				_ = state.Transition(nodeID, NodeFailed, TransitionEvidence{Reason: err.Error(), At: deps.Clock()})
				_ = deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeEnd, err.Error(), deps.Clock())
				return summaryFromState(cfg.RunID, preflight, state, false), err
			}
			if err := state.Transition(nodeID, NodeSucceeded, TransitionEvidence{
				Reason:       "node completed",
				OutputSHA256: result.OutputSHA256,
				LLMCallID:    result.LLMCallID,
				At:           deps.Clock(),
			}); err != nil {
				return summaryFromState(cfg.RunID, preflight, state, false), err
			}
			_ = deps.Journal.AppendNode(cfg.RunID, nodeID, EventNodeEnd, "node completed", deps.Clock())
			if result.OutputSHA256 != "" {
				_ = deps.Journal.AppendArtifact(cfg.RunID, nodeID, "outputs/"+nodeID+".json", result.OutputSHA256, 0, deps.Clock())
			}
		}
	}
	summary := summaryFromState(cfg.RunID, preflight, state, true)
	return summary, nil
}

func preflightGateFields(preflight PreflightResult) map[string]string {
	fields := make(map[string]string, len(preflight.Gates)+1)
	fields["passed"] = fmt.Sprintf("%t", preflight.Passed)
	for gate, passed := range preflight.Gates {
		fields["gate."+gate] = fmt.Sprintf("%t", passed)
	}
	return fields
}

func validateRunnerConfig(cfg RunnerConfig) error {
	if cfg.RunID == "" {
		return fmt.Errorf("run ID is required")
	}
	if cfg.WorkloadDir == "" {
		return fmt.Errorf("workload dir is required")
	}
	if cfg.Provider != RequiredDeepSeekProvider {
		return fmt.Errorf("provider %q is not allowed", cfg.Provider)
	}
	if cfg.Model != RequiredDeepSeekModel {
		return fmt.Errorf("model %q is not allowed", cfg.Model)
	}
	maxCalls := cfg.MaxCalls
	if maxCalls == 0 {
		maxCalls = DefaultMaxModelCalls
	}
	if maxCalls < 7 || maxCalls > DefaultMaxModelCalls {
		return fmt.Errorf("max calls %d outside allowed range 7-%d", maxCalls, DefaultMaxModelCalls)
	}
	return nil
}

func summaryFromState(runID string, preflight PreflightResult, state *ExecutionState, passed bool) Summary {
	return Summary{
		SchemaVersion:     SchemaVersion,
		RunID:             runID,
		Preflight:         preflight,
		Nodes:             state.Snapshot(),
		AllRequiredPassed: passed,
	}
}

type TransitionEvidence struct {
	Reason       string
	OutputSHA256 string
	LLMCallID    string
	At           time.Time
}

type NodeState struct {
	NodeID       string       `json:"node_id"`
	Status       NodeStatus   `json:"status"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Reason       string       `json:"reason,omitempty"`
	OutputSHA256 string       `json:"output_sha256,omitempty"`
	LLMCallID    string       `json:"llm_call_id,omitempty"`
	History      []Transition `json:"history,omitempty"`
}

type Transition struct {
	From         NodeStatus `json:"from"`
	To           NodeStatus `json:"to"`
	At           time.Time  `json:"at"`
	Reason       string     `json:"reason,omitempty"`
	OutputSHA256 string     `json:"output_sha256,omitempty"`
	LLMCallID    string     `json:"llm_call_id,omitempty"`
}

type ExecutionState struct {
	nodes map[string]NodeState
	order []string
}

func NewExecutionState(nodeIDs []string) *ExecutionState {
	nodes := make(map[string]NodeState, len(nodeIDs))
	order := append([]string(nil), nodeIDs...)
	for _, nodeID := range order {
		nodes[nodeID] = NodeState{NodeID: nodeID, Status: NodePending}
	}
	return &ExecutionState{nodes: nodes, order: order}
}

func (s *ExecutionState) Transition(nodeID string, to NodeStatus, evidence TransitionEvidence) error {
	state, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("unknown node %q", nodeID)
	}
	if !legalTransition(state.Status, to) {
		return fmt.Errorf("illegal node transition %s -> %s for %s", state.Status, to, nodeID)
	}
	at := evidence.At
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	transition := Transition{
		From:         state.Status,
		To:           to,
		At:           at,
		Reason:       evidence.Reason,
		OutputSHA256: evidence.OutputSHA256,
		LLMCallID:    evidence.LLMCallID,
	}
	state.Status = to
	state.UpdatedAt = at
	state.Reason = evidence.Reason
	if evidence.OutputSHA256 != "" {
		state.OutputSHA256 = evidence.OutputSHA256
	}
	if evidence.LLMCallID != "" {
		state.LLMCallID = evidence.LLMCallID
	}
	state.History = append(append([]Transition(nil), state.History...), transition)
	s.nodes[nodeID] = state
	return nil
}

func (s *ExecutionState) EnsureNode(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	if _, ok := s.nodes[nodeID]; ok {
		return nil
	}
	s.nodes[nodeID] = NodeState{NodeID: nodeID, Status: NodePending}
	s.order = append(s.order, nodeID)
	return nil
}

func (s *ExecutionState) Completed() map[string]bool {
	out := make(map[string]bool, len(s.nodes))
	for nodeID, state := range s.nodes {
		if state.Status == NodeSucceeded {
			out[nodeID] = true
		}
	}
	return out
}

func (s *ExecutionState) Snapshot() []NodeState {
	out := make([]NodeState, 0, len(s.order))
	for _, nodeID := range s.order {
		state := s.nodes[nodeID]
		state.History = append([]Transition(nil), state.History...)
		out = append(out, state)
	}
	return out
}

func legalTransition(from, to NodeStatus) bool {
	switch from {
	case NodePending:
		return to == NodeReady
	case NodeReady:
		return to == NodeRunning
	case NodeRunning:
		return to == NodeSucceeded || to == NodeFailed
	case NodeFailed:
		return to == NodeReplaying
	case NodeReplaying:
		return to == NodeRunning
	case NodeSucceeded:
		return false
	default:
		return false
	}
}
