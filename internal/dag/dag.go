package dag

import (
	"fmt"
	"sort"
)

type Graph struct {
	deps map[string][]string
}

func NewGraph() *Graph {
	return &Graph{deps: make(map[string][]string)}
}

func (g *Graph) AddNode(id string, deps []string) {
	g.deps[id] = append([]string(nil), deps...)
}

func (g *Graph) Nodes() []string {
	nodes := make([]string, 0, len(g.deps))
	for node := range g.deps {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

func (g *Graph) Validate() error {
	for node, deps := range g.deps {
		for _, dep := range deps {
			if dep == node {
				return fmt.Errorf("self dependency for node %q", node)
			}
			if _, ok := g.deps[dep]; !ok {
				return fmt.Errorf("node %q has missing dependency %q", node, dep)
			}
		}
	}

	color := make(map[string]int, len(g.deps))
	var visit func(string) error
	visit = func(node string) error {
		switch color[node] {
		case 1:
			return fmt.Errorf("cycle detected involving node %q", node)
		case 2:
			return nil
		}
		color[node] = 1
		deps := append([]string(nil), g.deps[node]...)
		sort.Strings(deps)
		for _, dep := range deps {
			if color[dep] == 1 {
				return fmt.Errorf("cycle detected involving nodes %q and %q", node, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		color[node] = 2
		return nil
	}
	for _, node := range g.Nodes() {
		if err := visit(node); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) Ready(completed map[string]bool) []string {
	if completed == nil {
		completed = map[string]bool{}
	}
	ready := make([]string, 0)
	for node, deps := range g.deps {
		if completed[node] {
			continue
		}
		allDone := true
		for _, dep := range deps {
			if !completed[dep] {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, node)
		}
	}
	sort.Strings(ready)
	return ready
}
