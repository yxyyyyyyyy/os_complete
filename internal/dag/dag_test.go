package dag

import "testing"

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
