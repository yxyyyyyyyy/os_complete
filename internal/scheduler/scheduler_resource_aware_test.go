package scheduler

import (
	"strings"
	"testing"

	"aort-r/internal/avp"
	"aort-r/internal/evidence"
)

func TestResourceAwarePressureDimensionsIndependentlyPenalizeCandidates(t *testing.T) {
	tests := []struct {
		name   string
		high   avp.AVP
		reason string
	}{
		{
			name: "memory",
			high: avp.AVP{
				AgentID:       "agent-a-low-vruntime-high-memory",
				TaskID:        "task-memory-pressure",
				State:         avp.StateReady,
				VRuntime:      1,
				Weight:        100,
				MemoryCurrent: 1024 * 1024 * 1024,
				PageTable:     []string{"cold"},
			},
			reason: "memory_pressure",
		},
		{
			name: "pids",
			high: avp.AVP{
				AgentID:     "agent-a-low-vruntime-high-pids",
				TaskID:      "task-pids-pressure",
				State:       avp.StateReady,
				VRuntime:    1,
				Weight:      100,
				PidsCurrent: 64,
				PageTable:   []string{"cold"},
			},
			reason: "pids_pressure",
		},
		{
			name: "cpu-throttle",
			high: avp.AVP{
				AgentID:   "agent-a-low-vruntime-high-cpu",
				TaskID:    "task-cpu-pressure",
				State:     avp.StateReady,
				VRuntime:  1,
				Weight:    100,
				CPUStat:   map[string]uint64{"nr_throttled": 100},
				PageTable: []string{"cold"},
			},
			reason: "cpu_throttle_pressure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPages := []string{
				"shared-01", "shared-02", "shared-03", "shared-04", "shared-05",
				"shared-06", "shared-07", "shared-08", "shared-09", "shared-10",
				"shared-11", "shared-12", "shared-13", "shared-14", "shared-15",
			}
			low := avp.AVP{
				AgentID:       "agent-b-higher-vruntime-low-pressure-high-context",
				TaskID:        tt.high.TaskID,
				State:         avp.StateReady,
				VRuntime:      12,
				Weight:        100,
				MemoryCurrent: 32 * 1024 * 1024,
				PidsCurrent:   2,
				CPUStat:       map[string]uint64{"nr_throttled": 0},
				PageTable:     sharedPages,
			}

			tokenCFS := New(PolicyTokenCFS)
			tokenSelected, _, ok := tokenCFS.Select(tt.high.TaskID, []avp.AVP{tt.high, low})
			if !ok || tokenSelected.AgentID != tt.high.AgentID {
				t.Fatalf("token-cfs should select lower vruntime candidate, selected=%#v ok=%v", tokenSelected, ok)
			}

			resourceAware := New(PolicyTokenCFSPrefixAffinityResourceAware)
			resourceAware.SetResourcePressure(ResourcePressure{
				EvidenceMode:   evidence.ModeDegraded,
				FallbackReason: "test fixture pressure sampler unavailable",
			})
			resourceAware.RememberLast(avp.AVP{AgentID: "previous", PageTable: sharedPages})
			selected, decision, ok := resourceAware.Select(tt.high.TaskID, []avp.AVP{tt.high, low})
			if !ok || selected.AgentID != low.AgentID {
				t.Fatalf("resource-aware should avoid %s pressure, selected=%#v decision=%#v ok=%v", tt.name, selected, decision, ok)
			}
			highScore := decision.ScoreByAgent[tt.high.AgentID]
			lowScore := decision.ScoreByAgent[low.AgentID]
			if highScore.FinalScore >= lowScore.FinalScore {
				t.Fatalf("high pressure candidate should score lower: high=%#v low=%#v", highScore, lowScore)
			}
			if !strings.Contains(decision.Reason, "resource pressure") {
				t.Fatalf("decision reason should explain resource-aware pressure scoring: %q", decision.Reason)
			}
			if decision.FallbackReason == "" || decision.EvidenceMode != evidence.ModeDegraded {
				t.Fatalf("degraded pressure evidence should be explicit: %#v", decision)
			}
			if highScore.MemoryPressure == 0 && highScore.PidsPressure == 0 && highScore.CPUThrottlePressure == 0 {
				t.Fatalf("expected pressure field to be populated for %s: %#v", tt.reason, highScore)
			}
		})
	}
}

func TestResourceAwareDefaultDegradedPressureExplainsFallback(t *testing.T) {
	s := New(PolicyTokenCFSPrefixAffinityResourceAware)
	s.RememberLast(avp.AVP{AgentID: "previous", PageTable: []string{"shared"}})

	_, decision, ok := s.Select("task-default-degraded-pressure", []avp.AVP{
		{
			AgentID:   "agent-a",
			TaskID:    "task-default-degraded-pressure",
			State:     avp.StateReady,
			VRuntime:  1,
			Weight:    100,
			PageTable: []string{"shared"},
		},
		{
			AgentID:   "agent-b",
			TaskID:    "task-default-degraded-pressure",
			State:     avp.StateReady,
			VRuntime:  2,
			Weight:    100,
			PageTable: []string{"other"},
		},
	})
	if !ok {
		t.Fatal("expected resource-aware decision")
	}
	if decision.EvidenceMode != evidence.ModeDegraded {
		t.Fatalf("default resource pressure evidence should be degraded: %#v", decision)
	}
	const want = "resource pressure sampler not configured or local cgroup pressure files unavailable"
	if decision.FallbackReason != want {
		t.Fatalf("fallback_reason = %q, want %q", decision.FallbackReason, want)
	}
}

func TestResourceAwareRecordsPSIPressure(t *testing.T) {
	s := New(PolicyTokenCFSPrefixAffinityResourceAware)
	s.SetResourcePressure(ResourcePressure{
		PSIPressure:  0.42,
		EvidenceMode: evidence.ModeDegraded,
	})

	_, decision, ok := s.Select("task-psi-pressure", []avp.AVP{
		{
			AgentID:   "agent-a",
			TaskID:    "task-psi-pressure",
			State:     avp.StateReady,
			VRuntime:  1,
			Weight:    100,
			PageTable: []string{"shared"},
		},
		{
			AgentID:   "agent-b",
			TaskID:    "task-psi-pressure",
			State:     avp.StateReady,
			VRuntime:  2,
			Weight:    100,
			PageTable: []string{"other"},
		},
	})
	if !ok {
		t.Fatal("expected resource-aware decision")
	}
	if decision.PSIPressure != 0.42 {
		t.Fatalf("psi pressure should be recorded: %#v", decision)
	}
	const want = "resource pressure sampler not configured or local cgroup pressure files unavailable"
	if decision.FallbackReason != want {
		t.Fatalf("fallback_reason = %q, want %q", decision.FallbackReason, want)
	}
}
