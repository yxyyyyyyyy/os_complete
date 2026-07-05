package scheduler

import (
	"strings"
	"testing"

	"aort-r/internal/avp"
	"aort-r/internal/evidence"
)

func TestResourceAwarePolicyPenalizesResourcePressureAndLogsDecisionSchema(t *testing.T) {
	s := New(PolicyTokenCFSPrefixAffinityResourceAware)
	s.SetAffinityThreshold(100)
	s.SetResourcePressure(ResourcePressure{
		EvidenceMode:   evidence.ModeDegraded,
		FallbackReason: "fixture cgroup files unavailable",
		PSIPressure:    0.25,
	})
	s.RememberLast(avp.AVP{AgentID: "previous", PageTable: []string{"system", "project", "task"}})

	selected, decision, ok := s.Select("task-resource-aware", []avp.AVP{
		{
			AgentID:       "agent-low-vruntime-high-pressure",
			TaskID:        "task-resource-aware",
			State:         avp.StateReady,
			VRuntime:      1,
			Weight:        100,
			MemoryCurrent: 950 * 1024 * 1024,
			PidsCurrent:   63,
			CPUStat:       map[string]uint64{"nr_throttled": 300},
			PageTable:     []string{"system"},
		},
		{
			AgentID:       "agent-balanced-shared-context",
			TaskID:        "task-resource-aware",
			State:         avp.StateReady,
			VRuntime:      12,
			Weight:        100,
			MemoryCurrent: 80 * 1024 * 1024,
			PidsCurrent:   4,
			CPUStat:       map[string]uint64{"nr_throttled": 0},
			PageTable:     []string{"system", "project", "task"},
		},
	})

	if !ok {
		t.Fatalf("no agent selected")
	}
	if selected.AgentID != "agent-balanced-shared-context" {
		t.Fatalf("selected %s with decision %#v", selected.AgentID, decision)
	}
	if decision.Policy != PolicyTokenCFSPrefixAffinityResourceAware {
		t.Fatalf("policy = %q", decision.Policy)
	}
	if decision.DecisionID == "" || decision.Timestamp == "" || decision.SelectedTask != "task-resource-aware" {
		t.Fatalf("decision identifiers missing: %#v", decision)
	}
	if decision.EvidenceMode != evidence.ModeDegraded || decision.FallbackReason == "" {
		t.Fatalf("evidence fallback missing: %#v", decision)
	}
	if decision.MemoryPressure <= 0 || decision.PidsPressure <= 0 || decision.CPUThrottlePressure <= 0 || decision.PSIPressure <= 0 {
		t.Fatalf("resource pressure not recorded: %#v", decision)
	}
	if !decision.DependencyReady || decision.FinalScore <= 0 {
		t.Fatalf("dependency/final score missing: %#v", decision)
	}
	if !strings.Contains(decision.Reason, "resource pressure") {
		t.Fatalf("reason should explain resource-aware choice: %q", decision.Reason)
	}
}

func TestResourceAwarePolicySkipsDependencyBlockedCandidates(t *testing.T) {
	s := New(PolicyTokenCFSPrefixAffinityResourceAware)

	selected, decision, ok := s.Select("task-dependency", []avp.AVP{
		{AgentID: "blocked", TaskID: "task-dependency", State: avp.StateWaitingIPC, VRuntime: 0},
		{AgentID: "ready", TaskID: "task-dependency", State: avp.StateReady, VRuntime: 10},
	})

	if !ok || selected.AgentID != "ready" {
		t.Fatalf("selected=%#v ok=%v decision=%#v", selected, ok, decision)
	}
	if !decision.DependencyReady {
		t.Fatalf("selected candidate should be dependency-ready: %#v", decision)
	}
}
