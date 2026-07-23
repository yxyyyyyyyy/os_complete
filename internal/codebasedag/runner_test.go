package codebasedag

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunnerExecutesGraphWithFakeDependencies(t *testing.T) {
	var order []string
	journalPath := filepath.Join(t.TempDir(), "events.jsonl")
	journal, err := NewEvidenceJournal(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	summary, err := Run(context.Background(), RunnerConfig{
		RunID:       "run-1",
		WorkloadDir: t.TempDir(),
		Provider:    "deepseek",
		Model:       RequiredDeepSeekModel,
		MaxCalls:    10,
	}, RunnerDeps{
		Preflight: PreflightFunc(func(context.Context, RunnerConfig) (PreflightResult, error) {
			order = append(order, "preflight")
			return PreflightResult{Passed: true, Gates: map[string]bool{"unit": true}}, nil
		}),
		NodeExecutor: NodeExecutorFunc(func(_ context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
			order = append(order, req.NodeID)
			return NodeExecutionResult{OutputSHA256: req.NodeID + "-hash", LLMCallID: req.NodeID + "-call"}, nil
		}),
		Journal: journal,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"preflight", "planner", "context-coder", "evidence-coder", "resource-coder", "integrate", "tester", "reviewer", "finalizer"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order = %#v, want %#v", order, wantOrder)
	}
	if !summary.AllRequiredPassed || summary.RunID != "run-1" {
		t.Fatalf("summary = %#v", summary)
	}
	for _, node := range summary.Nodes {
		if node.Status != NodeSucceeded {
			t.Fatalf("node did not succeed: %#v", node)
		}
	}
	events := readJournalEvents(t, journalPath)
	if len(events) < 18 {
		t.Fatalf("expected runner journal events, got %d", len(events))
	}
	if events[0].Type != EventPreflight {
		t.Fatalf("first event = %#v", events[0])
	}
}

func TestRunnerStopsOnPreflightFailure(t *testing.T) {
	_, err := Run(context.Background(), RunnerConfig{
		RunID:       "run-2",
		WorkloadDir: t.TempDir(),
		Provider:    "deepseek",
		Model:       RequiredDeepSeekModel,
		MaxCalls:    10,
	}, RunnerDeps{
		Preflight: PreflightFunc(func(context.Context, RunnerConfig) (PreflightResult, error) {
			return PreflightResult{Passed: false, Gates: map[string]bool{"manifest": false}}, nil
		}),
		NodeExecutor: NodeExecutorFunc(func(context.Context, NodeExecutionRequest) (NodeExecutionResult, error) {
			t.Fatal("node executor should not run after failed preflight")
			return NodeExecutionResult{}, nil
		}),
	})
	if err == nil {
		t.Fatal("preflight failure should stop runner")
	}
}

func TestExecutionStateRejectsIllegalTransitionsAndCopiesSnapshots(t *testing.T) {
	state := NewExecutionState([]string{"planner"})
	if err := state.Transition("planner", NodeReady, TransitionEvidence{Reason: "deps complete"}); err != nil {
		t.Fatal(err)
	}
	if err := state.Transition("planner", NodeRunning, TransitionEvidence{Reason: "started"}); err != nil {
		t.Fatal(err)
	}
	if err := state.Transition("planner", NodeSucceeded, TransitionEvidence{OutputSHA256: "hash", LLMCallID: "call"}); err != nil {
		t.Fatal(err)
	}
	if err := state.Transition("planner", NodeRunning, TransitionEvidence{Reason: "restart"}); err == nil {
		t.Fatal("terminal transition should fail")
	}
	snapshot := state.Snapshot()
	snapshot[0].Status = NodeFailed
	if state.Snapshot()[0].Status != NodeSucceeded {
		t.Fatal("Snapshot must return a copy")
	}
	completed := state.Completed()
	completed["planner"] = false
	if !state.Completed()["planner"] {
		t.Fatal("Completed must return a copy")
	}
}

func TestRunnerPropagatesNodeFailure(t *testing.T) {
	errBoom := errors.New("boom")
	_, err := Run(context.Background(), RunnerConfig{
		RunID:       "run-3",
		WorkloadDir: t.TempDir(),
		Provider:    "deepseek",
		Model:       RequiredDeepSeekModel,
		MaxCalls:    10,
	}, RunnerDeps{
		Preflight: PreflightFunc(func(context.Context, RunnerConfig) (PreflightResult, error) {
			return PreflightResult{Passed: true}, nil
		}),
		NodeExecutor: NodeExecutorFunc(func(_ context.Context, req NodeExecutionRequest) (NodeExecutionResult, error) {
			if req.NodeID == "planner" {
				return NodeExecutionResult{}, errBoom
			}
			return NodeExecutionResult{}, nil
		}),
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v", err)
	}
}
