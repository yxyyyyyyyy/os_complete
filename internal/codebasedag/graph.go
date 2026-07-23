package codebasedag

import (
	"fmt"
	"sort"

	"aort-r/internal/dag"
)

type FixerLoop struct {
	ReviewNode     string `json:"review_node"`
	FixerPrefix    string `json:"fixer_prefix"`
	RecheckNode    string `json:"recheck_node"`
	FinalizerNode  string `json:"finalizer_node"`
	MaxIterations  int    `json:"max_iterations"`
	TriggerVerdict string `json:"trigger_verdict"`
}

type CodebaseGraph struct {
	graph *dag.Graph
	deps  map[string][]string
	loops []FixerLoop
}

func NewCodebaseGraph() CodebaseGraph {
	spec := map[string][]string{
		"preflight":      nil,
		"planner":        {"preflight"},
		"resource-coder": {"planner"},
		"context-coder":  {"planner"},
		"evidence-coder": {"planner"},
		"integrate":      {"resource-coder", "context-coder", "evidence-coder"},
		"tester":         {"integrate"},
		"reviewer":       {"tester"},
		"finalizer":      {"reviewer"},
	}
	g := dag.NewGraph()
	deps := make(map[string][]string, len(spec))
	for node, nodeDeps := range spec {
		copied := append([]string(nil), nodeDeps...)
		sort.Strings(copied)
		g.AddNode(node, copied)
		deps[node] = copied
	}
	return CodebaseGraph{
		graph: g,
		deps:  deps,
		loops: []FixerLoop{{
			ReviewNode:     "reviewer",
			FixerPrefix:    "fixer",
			RecheckNode:    "tester",
			FinalizerNode:  "finalizer",
			MaxIterations:  2,
			TriggerVerdict: "fix",
		}},
	}
}

func (g CodebaseGraph) Validate() error {
	if g.graph == nil {
		return fmt.Errorf("codebase graph is nil")
	}
	if err := g.graph.Validate(); err != nil {
		return err
	}
	for _, loop := range g.loops {
		if loop.MaxIterations <= 0 {
			return fmt.Errorf("fixer loop for %q has invalid max iterations %d", loop.ReviewNode, loop.MaxIterations)
		}
		for _, node := range []string{loop.ReviewNode, loop.RecheckNode, loop.FinalizerNode} {
			if _, ok := g.deps[node]; !ok {
				return fmt.Errorf("fixer loop references missing node %q", node)
			}
		}
		if loop.FixerPrefix == "" || loop.TriggerVerdict == "" {
			return fmt.Errorf("fixer loop for %q is incomplete", loop.ReviewNode)
		}
	}
	return nil
}

func (g CodebaseGraph) Nodes() []string {
	if g.graph == nil {
		return nil
	}
	return g.graph.Nodes()
}

func (g CodebaseGraph) Ready(completed map[string]bool) []string {
	if g.graph == nil {
		return nil
	}
	return g.graph.Ready(completed)
}

func (g CodebaseGraph) Dependencies(node string) []string {
	return append([]string(nil), g.deps[node]...)
}

func (g CodebaseGraph) FixerLoops() []FixerLoop {
	return append([]FixerLoop(nil), g.loops...)
}
