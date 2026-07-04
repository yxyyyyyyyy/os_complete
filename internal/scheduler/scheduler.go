package scheduler

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"aort-r/internal/avp"
)

const (
	PolicyFIFO                   = "fifo"
	PolicyTokenCFS               = "token-cfs"
	PolicyTokenCFSPrefixAffinity = "token-cfs-prefix-affinity"
)

type DecisionLog struct {
	ID             string            `json:"id"`
	TaskID         string            `json:"task_id"`
	Candidates     []string          `json:"candidates"`
	SelectedAgent  string            `json:"selected_agent"`
	Policy         string            `json:"policy"`
	Reason         string            `json:"reason"`
	VRuntimeBefore map[string]uint64 `json:"vruntime_before"`
	VRuntimeAfter  map[string]uint64 `json:"vruntime_after"`
	SharedPages    map[string]int    `json:"shared_pages"`
	CreatedAt      int64             `json:"created_at"`
}

type Scheduler struct {
	mu                sync.RWMutex
	policy            string
	affinityThreshold uint64
	last              *avp.AVP
	decisions         []DecisionLog
}

func New(policy string) *Scheduler {
	if policy == "" {
		policy = PolicyFIFO
	}
	return &Scheduler{policy: policy, affinityThreshold: 50}
}

func (s *Scheduler) SetPolicy(policy string) error {
	switch policy {
	case PolicyFIFO, PolicyTokenCFS, PolicyTokenCFSPrefixAffinity:
	default:
		return fmt.Errorf("unsupported scheduler policy %q", policy)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
	return nil
}

func (s *Scheduler) Policy() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

func (s *Scheduler) SetAffinityThreshold(threshold uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.affinityThreshold = threshold
}

func (s *Scheduler) RememberLast(agent avp.AVP) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyAgent := agent
	s.last = &copyAgent
}

func (s *Scheduler) Select(taskID string, candidates []avp.AVP) (avp.AVP, DecisionLog, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ready := readyAgents(candidates)
	if len(ready) == 0 {
		return avp.AVP{}, DecisionLog{}, false
	}

	selected, reason, sharedPages := s.selectLocked(ready)
	decision := s.makeDecisionLocked(taskID, ready, selected, reason, sharedPages)
	copySelected := selected
	s.last = &copySelected
	s.decisions = append(s.decisions, decision)
	return selected, decision, true
}

func (s *Scheduler) Decisions() []DecisionLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DecisionLog, len(s.decisions))
	copy(out, s.decisions)
	return out
}

func (s *Scheduler) selectLocked(candidates []avp.AVP) (avp.AVP, string, map[string]int) {
	switch s.policy {
	case PolicyTokenCFS:
		return minVRuntime(candidates), "lowest vruntime", nil
	case PolicyTokenCFSPrefixAffinity:
		base := minVRuntime(candidates)
		shared := s.sharedPages(candidates)
		if s.last == nil {
			return base, "lowest vruntime; no previous prefix group", shared
		}
		best := base
		bestShared := shared[base.AgentID]
		for _, candidate := range candidates {
			if shared[candidate.AgentID] <= bestShared {
				continue
			}
			if vruntimeDistance(candidate.VRuntime, base.VRuntime) <= s.affinityThreshold {
				best = candidate
				bestShared = shared[candidate.AgentID]
			}
		}
		if best.AgentID != base.AgentID {
			return best, fmt.Sprintf("prefix affinity shared_pages=%d within threshold=%d", bestShared, s.affinityThreshold), shared
		}
		return base, "lowest vruntime; no affinity candidate within threshold", shared
	default:
		return fifo(candidates), "earliest ready agent", nil
	}
}

func (s *Scheduler) makeDecisionLocked(taskID string, candidates []avp.AVP, selected avp.AVP, reason string, shared map[string]int) DecisionLog {
	ids := make([]string, 0, len(candidates))
	before := make(map[string]uint64, len(candidates))
	after := make(map[string]uint64, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.AgentID)
		before[candidate.AgentID] = candidate.VRuntime
		after[candidate.AgentID] = candidate.VRuntime
	}
	if shared == nil {
		shared = make(map[string]int)
	}
	return DecisionLog{
		ID:             time.Now().Format("20060102150405.000000000"),
		TaskID:         taskID,
		Candidates:     ids,
		SelectedAgent:  selected.AgentID,
		Policy:         s.policy,
		Reason:         reason,
		VRuntimeBefore: before,
		VRuntimeAfter:  after,
		SharedPages:    shared,
		CreatedAt:      time.Now().UnixMilli(),
	}
}

func (s *Scheduler) sharedPages(candidates []avp.AVP) map[string]int {
	out := make(map[string]int, len(candidates))
	if s.last == nil {
		return out
	}
	lastPages := make(map[string]struct{}, len(s.last.PageTable)+len(s.last.ContextPages))
	for _, page := range append(append([]string(nil), s.last.PageTable...), s.last.ContextPages...) {
		lastPages[page] = struct{}{}
	}
	for _, candidate := range candidates {
		for _, page := range append(append([]string(nil), candidate.PageTable...), candidate.ContextPages...) {
			if _, ok := lastPages[page]; ok {
				out[candidate.AgentID]++
			}
		}
	}
	return out
}

func ChargeTokens(agent avp.AVP, consumedTokens int) avp.AVP {
	weight := agent.Weight
	if weight <= 0 {
		weight = 100
	}
	agent.ConsumedTokens += consumedTokens
	agent.VRuntime += uint64(consumedTokens / weight)
	if consumedTokens > 0 && consumedTokens/weight == 0 {
		agent.VRuntime++
	}
	return agent
}

func readyAgents(candidates []avp.AVP) []avp.AVP {
	ready := make([]avp.AVP, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.State == avp.StateReady {
			ready = append(ready, candidate)
		}
	}
	return ready
}

func fifo(candidates []avp.AVP) avp.AVP {
	out := append([]avp.AVP(nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out[0]
}

func minVRuntime(candidates []avp.AVP) avp.AVP {
	out := append([]avp.AVP(nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].VRuntime == out[j].VRuntime {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].VRuntime < out[j].VRuntime
	})
	return out[0]
}

func vruntimeDistance(left, right uint64) uint64 {
	return uint64(math.Abs(float64(left) - float64(right)))
}
