package scheduler

import (
	"testing"

	"aort-r/internal/avp"
)

func TestFIFOSelectsEarliestReadyAgent(t *testing.T) {
	s := New(PolicyFIFO)
	selected, decision, ok := s.Select("task-1", []avp.AVP{
		{AgentID: "agent-late", TaskID: "task-1", State: avp.StateReady, CreatedAt: 30},
		{AgentID: "agent-early", TaskID: "task-1", State: avp.StateReady, CreatedAt: 10},
		{AgentID: "agent-running", TaskID: "task-1", State: avp.StateRunning, CreatedAt: 1},
	})

	if !ok {
		t.Fatalf("no agent selected")
	}
	if selected.AgentID != "agent-early" {
		t.Fatalf("selected = %s", selected.AgentID)
	}
	if decision.Policy != PolicyFIFO || decision.SelectedAgent != "agent-early" {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestTokenCFSSelectsLowestVRuntimeAndChargesTokensByWeight(t *testing.T) {
	s := New(PolicyTokenCFS)
	selected, _, ok := s.Select("task-1", []avp.AVP{
		{AgentID: "agent-heavy", TaskID: "task-1", State: avp.StateReady, Weight: 100, VRuntime: 60},
		{AgentID: "agent-light", TaskID: "task-1", State: avp.StateReady, Weight: 50, VRuntime: 10},
	})
	if !ok || selected.AgentID != "agent-light" {
		t.Fatalf("selected = %#v ok=%v", selected, ok)
	}

	updated := ChargeTokens(selected, 200)
	if updated.ConsumedTokens != 200 {
		t.Fatalf("consumed = %d", updated.ConsumedTokens)
	}
	if updated.VRuntime != 14 {
		t.Fatalf("vruntime = %d", updated.VRuntime)
	}
}

func TestPrefixAffinityPrefersSharedPagesWithinVRuntimeThreshold(t *testing.T) {
	s := New(PolicyTokenCFSPrefixAffinity)
	s.SetAffinityThreshold(20)
	s.RememberLast(avp.AVP{AgentID: "agent-planner", PageTable: []string{"system", "project", "task"}})

	selected, decision, ok := s.Select("task-1", []avp.AVP{
		{
			AgentID:   "agent-low-vruntime",
			TaskID:    "task-1",
			State:     avp.StateReady,
			Weight:    100,
			VRuntime:  100,
			PageTable: []string{"system"},
		},
		{
			AgentID:   "agent-shared-prefix",
			TaskID:    "task-1",
			State:     avp.StateReady,
			Weight:    100,
			VRuntime:  112,
			PageTable: []string{"system", "project", "task"},
		},
	})

	if !ok {
		t.Fatalf("no agent selected")
	}
	if selected.AgentID != "agent-shared-prefix" {
		t.Fatalf("selected = %s decision=%#v", selected.AgentID, decision)
	}
	if decision.SharedPages["agent-shared-prefix"] != 3 || decision.Reason == "" {
		t.Fatalf("decision = %#v", decision)
	}
}
