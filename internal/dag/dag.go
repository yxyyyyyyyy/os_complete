package dag

import "sort"

type Graph struct {
	deps map[string][]string
}

func NewGraph() *Graph {
	return &Graph{deps: make(map[string][]string)}
}

func (g *Graph) AddNode(id string, deps []string) {
	g.deps[id] = append([]string(nil), deps...)
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
