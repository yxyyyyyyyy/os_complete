package demo

import (
	"context"
	"fmt"
	"sync/atomic"

	"aort-r/internal/events"
)

type Runner struct {
	nextTask uint64
}

type Agent struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	State      string `json:"state"`
	PID        int    `json:"pid"`
	LastSeen   int64  `json:"last_seen"`
	CgroupPath string `json:"cgroup_path"`
}

type DAGNode struct {
	ID           string   `json:"id"`
	Role         string   `json:"role"`
	Dependencies []string `json:"dependencies"`
}

type Result struct {
	TaskID string         `json:"task_id"`
	Status string         `json:"status"`
	Agents []Agent        `json:"agents"`
	DAG    []DAGNode      `json:"dag"`
	Events []events.Event `json:"events"`
}

func NewSoftwareDemoRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(ctx context.Context) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	n := atomic.AddUint64(&r.nextTask, 1)
	taskID := fmt.Sprintf("task-%d", n)
	result := Result{
		TaskID: taskID,
		Status: "completed",
		Agents: []Agent{
			{ID: "planner-1", Role: "planner", State: "COMPLETED"},
			{ID: "coder-a-1", Role: "coder-a", State: "COMPLETED"},
			{ID: "coder-b-1", Role: "coder-b", State: "COMPLETED"},
			{ID: "tester-1", Role: "tester", State: "COMPLETED"},
			{ID: "reviewer-1", Role: "reviewer", State: "COMPLETED"},
			{ID: "fixer-1", Role: "fixer", State: "COMPLETED"},
		},
		DAG: []DAGNode{
			{ID: "planner", Role: "planner"},
			{ID: "coder-a", Role: "coder-a", Dependencies: []string{"planner"}},
			{ID: "coder-b", Role: "coder-b", Dependencies: []string{"planner"}},
			{ID: "tester", Role: "tester", Dependencies: []string{"coder-a", "coder-b"}},
			{ID: "reviewer", Role: "reviewer", Dependencies: []string{"tester"}},
			{ID: "fixer", Role: "fixer", Dependencies: []string{"reviewer"}},
		},
	}
	result.Events = demoEvents(taskID)
	return result, nil
}

func (r Result) Roles() []string {
	roles := make([]string, 0, len(r.Agents))
	for _, agent := range r.Agents {
		roles = append(roles, agent.Role)
	}
	return roles
}

func demoEvents(taskID string) []events.Event {
	rows := []struct {
		eventType string
		agentID   string
		payload   map[string]any
	}{
		{"task.created", "", map[string]any{"requirement": "Todo Web API"}},
		{"agent.created", "planner-1", map[string]any{"role": "planner"}},
		{"agent.state_changed", "planner-1", map[string]any{"state": "READY"}},
		{"scheduler.selected", "planner-1", map[string]any{"strategy": "mock", "decision_reason": "root DAG node"}},
		{"syscall.finished", "planner-1", map[string]any{"name": "context.materialize", "exit_code": 0}},
		{"agent.state_changed", "planner-1", map[string]any{"state": "COMPLETED"}},
		{"agent.created", "coder-a-1", map[string]any{"role": "coder-a"}},
		{"agent.created", "coder-b-1", map[string]any{"role": "coder-b"}},
		{"agent.state_changed", "coder-a-1", map[string]any{"state": "COMPLETED"}},
		{"agent.state_changed", "coder-b-1", map[string]any{"state": "COMPLETED"}},
		{"agent.created", "tester-1", map[string]any{"role": "tester"}},
		{"syscall.finished", "tester-1", map[string]any{"name": "tool.go_test", "exit_code": 1}},
		{"agent.created", "reviewer-1", map[string]any{"role": "reviewer"}},
		{"agent.created", "fixer-1", map[string]any{"role": "fixer"}},
		{"agent.state_changed", "fixer-1", map[string]any{"state": "COMPLETED"}},
		{"syscall.finished", "tester-1", map[string]any{"name": "tool.go_test", "exit_code": 0}},
		{"task.completed", "", map[string]any{"status": "completed"}},
	}
	out := make([]events.Event, 0, len(rows))
	for i, row := range rows {
		out = append(out, events.Event{
			ID:        fmt.Sprintf("%s-e-%03d", taskID, i+1),
			TaskID:    taskID,
			AgentID:   row.agentID,
			Type:      row.eventType,
			Source:    "runtime",
			Timestamp: int64(i + 1),
			Payload:   row.payload,
		})
	}
	return out
}
