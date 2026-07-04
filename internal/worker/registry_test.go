package worker

import (
	"testing"
	"time"

	"aort-r/internal/avp"
)

func TestRegistryRegisterUpdatesPIDAndState(t *testing.T) {
	registry := NewRegistry(nil)
	registry.CreateAgent("agent-planner", "Planner", "task-1")

	registry.HandleMessage(Message{
		Type:    MessageRegister,
		AgentID: "agent-planner",
		Role:    "Planner",
		TaskID:  "task-1",
		PID:     12345,
	})

	agent, ok := registry.Get("agent-planner")
	if !ok {
		t.Fatalf("agent not found")
	}
	if agent.PID != 12345 {
		t.Fatalf("PID = %d", agent.PID)
	}
	if agent.State != avp.StateRunning {
		t.Fatalf("State = %s", agent.State)
	}
	if agent.LastSeen == 0 {
		t.Fatalf("LastSeen was not recorded")
	}
}

func TestRegistryHeartbeatTimeoutMarksAgentFailed(t *testing.T) {
	registry := NewRegistry(nil)
	registry.CreateAgent("agent-planner", "Planner", "task-1")
	registry.HandleMessage(Message{
		Type:    MessageRegister,
		AgentID: "agent-planner",
		Role:    "Planner",
		TaskID:  "task-1",
		PID:     12345,
	})

	agent, _ := registry.Get("agent-planner")
	registry.MarkLastSeenForTest(agent.AgentID, time.Now().Add(-7*time.Second))
	failed := registry.MarkHeartbeatLost(time.Now(), 6*time.Second)
	if len(failed) != 1 {
		t.Fatalf("failed = %#v", failed)
	}

	agent, _ = registry.Get("agent-planner")
	if agent.State != avp.StateFailed {
		t.Fatalf("State = %s", agent.State)
	}
}

func TestRegistryReportCompletedUpdatesState(t *testing.T) {
	registry := NewRegistry(nil)
	registry.CreateAgent("agent-planner", "Planner", "task-1")
	registry.HandleMessage(Message{
		Type:    MessageRegister,
		AgentID: "agent-planner",
		Role:    "Planner",
		TaskID:  "task-1",
		PID:     12345,
	})

	registry.HandleMessage(Message{
		Type:    MessageReport,
		AgentID: "agent-planner",
		TaskID:  "task-1",
		Status:  string(avp.StateCompleted),
	})

	agent, _ := registry.Get("agent-planner")
	if agent.State != avp.StateCompleted {
		t.Fatalf("State = %s", agent.State)
	}
}

func TestRegistryRestoreAgentPreservesCheckpointState(t *testing.T) {
	registry := NewRegistry(nil)
	registry.RestoreAgent(avp.AVP{
		AgentID:      "agent-coder",
		TaskID:       "task-1",
		Role:         "Coder",
		State:        avp.StateReady,
		VRuntime:     42,
		Dependencies: []string{"agent-planner"},
		PageTable:    []string{"page-a", "page-b"},
		CreatedAt:    100,
		UpdatedAt:    120,
	})

	agent, ok := registry.Get("agent-coder")
	if !ok {
		t.Fatalf("agent not found")
	}
	if agent.State != avp.StateReady || agent.VRuntime != 42 {
		t.Fatalf("agent = %#v", agent)
	}
	if len(agent.Dependencies) != 1 || agent.Dependencies[0] != "agent-planner" {
		t.Fatalf("dependencies = %#v", agent.Dependencies)
	}
	if len(agent.PageTable) != 2 || agent.PageTable[1] != "page-b" {
		t.Fatalf("page table = %#v", agent.PageTable)
	}
}
