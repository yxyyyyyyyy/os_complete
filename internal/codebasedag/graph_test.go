package codebasedag

import (
	"reflect"
	"testing"
)

func TestNewCodebaseGraphHasExactBaseNodesAndDependencies(t *testing.T) {
	graph := NewCodebaseGraph()
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	wantNodes := []string{
		"context-coder",
		"evidence-coder",
		"fault-agent",
		"finalizer",
		"integrate",
		"planner",
		"preflight",
		"resource-coder",
		"reviewer",
		"tester",
	}
	if got := graph.Nodes(); !reflect.DeepEqual(got, wantNodes) {
		t.Fatalf("nodes = %#v, want %#v", got, wantNodes)
	}
	assertDeps(t, graph, "preflight")
	assertDeps(t, graph, "planner", "preflight")
	assertDeps(t, graph, "resource-coder", "planner")
	assertDeps(t, graph, "context-coder", "planner")
	assertDeps(t, graph, "evidence-coder", "planner")
	assertDeps(t, graph, "fault-agent", "planner")
	assertDeps(t, graph, "integrate", "context-coder", "evidence-coder", "fault-agent", "resource-coder")
	assertDeps(t, graph, "tester", "integrate")
	assertDeps(t, graph, "reviewer", "tester")
	assertDeps(t, graph, "finalizer", "reviewer")
}

func TestCodebaseGraphReadyOrderAndFixerLoopPolicy(t *testing.T) {
	graph := NewCodebaseGraph()
	ready := graph.Ready(map[string]bool{"preflight": true, "planner": true})
	wantReady := []string{"resource-coder", "context-coder", "evidence-coder", "fault-agent"}
	if !reflect.DeepEqual(ready, wantReady) {
		t.Fatalf("ready = %#v, want %#v", ready, wantReady)
	}
	loops := graph.FixerLoops()
	wantLoops := []FixerLoop{{
		ReviewNode:     "reviewer",
		FixerPrefix:    "fixer",
		RecheckNode:    "tester",
		FinalizerNode:  "finalizer",
		MaxIterations:  2,
		TriggerVerdict: "fix",
	}}
	if !reflect.DeepEqual(loops, wantLoops) {
		t.Fatalf("fixer loops = %#v, want %#v", loops, wantLoops)
	}

	loops[0].MaxIterations = 99
	if graph.FixerLoops()[0].MaxIterations != 2 {
		t.Fatal("FixerLoops must return a defensive copy")
	}
}

func assertDeps(t *testing.T, graph CodebaseGraph, node string, deps ...string) {
	t.Helper()
	if got := graph.Dependencies(node); !reflect.DeepEqual(got, deps) {
		t.Fatalf("%s deps = %#v, want %#v", node, got, deps)
	}
}
