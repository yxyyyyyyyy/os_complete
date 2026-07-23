package dag

import (
	"reflect"
	"strings"
	"testing"
)

func TestGraphReadyReturnsNodesWhoseDepsCompleted(t *testing.T) {
	g := NewGraph()
	g.AddNode("planner", nil)
	g.AddNode("coder", []string{"planner"})
	g.AddNode("tester", []string{"coder"})
	ready := g.Ready(map[string]bool{"planner": true})
	if len(ready) != 1 || ready[0] != "coder" {
		t.Fatalf("ready = %#v", ready)
	}
}

func TestGraphReadyReturnsRootWhenNoDeps(t *testing.T) {
	g := NewGraph()
	g.AddNode("planner", nil)
	ready := g.Ready(nil)
	if len(ready) != 1 || ready[0] != "planner" {
		t.Fatalf("ready = %#v", ready)
	}
}

func TestGraphReadySkipsCompletedNodes(t *testing.T) {
	g := NewGraph()
	g.AddNode("planner", nil)
	ready := g.Ready(map[string]bool{"planner": true})
	if len(ready) != 0 {
		t.Fatalf("ready = %#v", ready)
	}
}

func TestGraphNodesReturnsSortedCopy(t *testing.T) {
	g := NewGraph()
	g.AddNode("tester", []string{"coder"})
	g.AddNode("planner", nil)
	g.AddNode("coder", []string{"planner"})

	nodes := g.Nodes()
	if want := []string{"coder", "planner", "tester"}; !reflect.DeepEqual(nodes, want) {
		t.Fatalf("nodes = %#v, want %#v", nodes, want)
	}
	nodes[0] = "mutated"
	if again := g.Nodes(); again[0] != "coder" {
		t.Fatalf("Nodes exposed internal state: %#v", again)
	}
}

func TestGraphValidateAcceptsFanOutFanInGraph(t *testing.T) {
	g := NewGraph()
	g.AddNode("planner", nil)
	g.AddNode("a", []string{"planner"})
	g.AddNode("b", []string{"planner"})
	g.AddNode("integrate", []string{"a", "b"})
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestGraphValidateRejectsMissingDependencySelfDependencyAndCycle(t *testing.T) {
	missing := NewGraph()
	missing.AddNode("tester", []string{"coder"})
	if err := missing.Validate(); err == nil || !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("missing dependency error = %v", err)
	}

	self := NewGraph()
	self.AddNode("planner", []string{"planner"})
	if err := self.Validate(); err == nil || !strings.Contains(err.Error(), "self dependency") {
		t.Fatalf("self dependency error = %v", err)
	}

	cycle := NewGraph()
	cycle.AddNode("a", []string{"b"})
	cycle.AddNode("b", []string{"a"})
	err := cycle.Validate()
	if err == nil {
		t.Fatal("cycle should fail")
	}
	if !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
		t.Fatalf("cycle error should name nodes, got %v", err)
	}
}

func TestGraphAddNodeReplacesDuplicateNode(t *testing.T) {
	g := NewGraph()
	g.AddNode("coder", []string{"planner"})
	g.AddNode("coder", nil)
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	if ready := g.Ready(nil); len(ready) != 1 || ready[0] != "coder" {
		t.Fatalf("ready = %#v", ready)
	}
}
