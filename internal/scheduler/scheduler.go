package scheduler

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/evidence"
)

const (
	PolicyFIFO                                = "fifo"
	PolicyTokenCFS                            = "token-cfs"
	PolicyTokenCFSPrefixAffinity              = "token-cfs-prefix-affinity"
	PolicyTokenCFSPrefixAffinityResourceAware = "token-cfs-prefix-affinity-resource-aware"
)

type DecisionLog struct {
	ID                   string                       `json:"id"`
	DecisionID           string                       `json:"decision_id"`
	Timestamp            string                       `json:"timestamp"`
	TaskID               string                       `json:"task_id"`
	SelectedTask         string                       `json:"selected_task"`
	Candidates           []string                     `json:"candidates"`
	CandidateDetails     []CandidateDecision          `json:"candidate_details,omitempty"`
	SelectedAgent        string                       `json:"selected_agent"`
	Policy               string                       `json:"policy"`
	Reason               string                       `json:"reason"`
	VRuntimeBefore       map[string]uint64            `json:"vruntime_before"`
	VRuntimeAfter        map[string]uint64            `json:"vruntime_after"`
	SharedPages          map[string]int               `json:"shared_pages"`
	VRuntimeScore        float64                      `json:"vruntime_score"`
	ContextAffinityScore float64                      `json:"context_affinity_score"`
	MemoryPressure       float64                      `json:"memory_pressure"`
	PidsPressure         float64                      `json:"pids_pressure"`
	CPUThrottlePressure  float64                      `json:"cpu_throttle_pressure"`
	PSIPressure          float64                      `json:"psi_pressure"`
	DependencyReady      bool                         `json:"dependency_ready"`
	SpawnPriority        float64                      `json:"dynamic_spawn_priority"`
	FinalScore           float64                      `json:"final_score"`
	EvidenceMode         evidence.Mode                `json:"evidence_mode"`
	FallbackReason       string                       `json:"fallback_reason"`
	ResourcePressure     ResourcePressure             `json:"resource_pressure"`
	ScoreByAgent         map[string]CandidateDecision `json:"score_by_agent,omitempty"`
	CreatedAt            int64                        `json:"created_at"`
}

type CandidateDecision struct {
	AgentID              string  `json:"agent_id"`
	TaskID               string  `json:"task_id"`
	VRuntimeScore        float64 `json:"vruntime_score"`
	ContextAffinityScore float64 `json:"context_affinity_score"`
	MemoryPressure       float64 `json:"memory_pressure"`
	PidsPressure         float64 `json:"pids_pressure"`
	CPUThrottlePressure  float64 `json:"cpu_throttle_pressure"`
	PSIPressure          float64 `json:"psi_pressure"`
	DependencyReady      bool    `json:"dependency_ready"`
	SpawnPriority        float64 `json:"dynamic_spawn_priority"`
	FinalScore           float64 `json:"final_score"`
}

type ResourcePressure struct {
	MemoryPressure      float64       `json:"memory_pressure"`
	PidsPressure        float64       `json:"pids_pressure"`
	CPUThrottlePressure float64       `json:"cpu_throttle_pressure"`
	PSIPressure         float64       `json:"psi_pressure"`
	EvidenceMode        evidence.Mode `json:"evidence_mode"`
	FallbackReason      string        `json:"fallback_reason"`
}

type Scheduler struct {
	mu                sync.RWMutex
	policy            string
	affinityThreshold uint64
	last              *avp.AVP
	decisions         []DecisionLog
	resourcePressure  ResourcePressure
}

func New(policy string) *Scheduler {
	if policy == "" {
		policy = PolicyFIFO
	}
	return &Scheduler{policy: policy, affinityThreshold: 50}
}

func (s *Scheduler) SetPolicy(policy string) error {
	switch policy {
	case PolicyFIFO, PolicyTokenCFS, PolicyTokenCFSPrefixAffinity, PolicyTokenCFSPrefixAffinityResourceAware:
	default:
		return fmt.Errorf("unsupported scheduler policy %q", policy)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
	return nil
}

func Policies() []string {
	return []string{
		PolicyFIFO,
		PolicyTokenCFS,
		PolicyTokenCFSPrefixAffinity,
		PolicyTokenCFSPrefixAffinityResourceAware,
	}
}

func (s *Scheduler) Policy() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

func (s *Scheduler) SetResourcePressure(pressure ResourcePressure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resourcePressure = normalizeResourcePressure(pressure)
}

func (s *Scheduler) ResourcePressure() ResourcePressure {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return normalizeResourcePressure(s.resourcePressure)
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

	selected, reason, sharedPages, scoreByAgent := s.selectLocked(ready)
	decision := s.makeDecisionLocked(taskID, ready, selected, reason, sharedPages, scoreByAgent)
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

func (s *Scheduler) selectLocked(candidates []avp.AVP) (avp.AVP, string, map[string]int, map[string]CandidateDecision) {
	switch s.policy {
	case PolicyTokenCFS:
		return minVRuntime(candidates), "lowest vruntime", nil, nil
	case PolicyTokenCFSPrefixAffinity:
		selected, reason, shared := s.selectPrefixAffinity(candidates)
		return selected, reason, shared, nil
	case PolicyTokenCFSPrefixAffinityResourceAware:
		return s.selectResourceAware(candidates)
	default:
		return fifo(candidates), "earliest ready agent", nil, nil
	}
}

func (s *Scheduler) selectPrefixAffinity(candidates []avp.AVP) (avp.AVP, string, map[string]int) {
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
}

func (s *Scheduler) selectResourceAware(candidates []avp.AVP) (avp.AVP, string, map[string]int, map[string]CandidateDecision) {
	shared := s.sharedPages(candidates)
	scoreByAgent := make(map[string]CandidateDecision, len(candidates))
	best := candidates[0]
	bestScore := math.Inf(-1)
	for _, candidate := range candidates {
		score := s.resourceAwareScore(candidate, shared[candidate.AgentID])
		scoreByAgent[candidate.AgentID] = score
		if score.FinalScore > bestScore || (score.FinalScore == bestScore && candidate.CreatedAt < best.CreatedAt) {
			best = candidate
			bestScore = score.FinalScore
		}
	}
	return best, "selected due to low vruntime + high context affinity + low resource pressure", shared, scoreByAgent
}

func (s *Scheduler) resourceAwareScore(candidate avp.AVP, sharedPages int) CandidateDecision {
	global := normalizeResourcePressure(s.resourcePressure)
	memoryPressure := maxFloat(global.MemoryPressure, clamp(float64(candidate.MemoryCurrent)/(1024*1024*1024), 0, 1))
	pidsPressure := maxFloat(global.PidsPressure, clamp(float64(candidate.PidsCurrent)/64, 0, 1))
	cpuThrottle := global.CPUThrottlePressure
	if candidate.CPUStat != nil {
		cpuThrottle = maxFloat(cpuThrottle, clamp(float64(candidate.CPUStat["nr_throttled"])/100, 0, 1))
		cpuThrottle = maxFloat(cpuThrottle, clamp(float64(candidate.CPUStat["throttled_usec"])/10_000_000, 0, 1))
	}
	vruntimeScore := 1000 / (1 + float64(candidate.VRuntime))
	contextScore := float64(sharedPages)
	spawnPriority := float64(candidate.Priority)
	finalScore := vruntimeScore + contextScore*30 + spawnPriority*5 -
		memoryPressure*180 - pidsPressure*160 - cpuThrottle*120 - global.PSIPressure*90
	return CandidateDecision{
		AgentID:              candidate.AgentID,
		TaskID:               candidate.TaskID,
		VRuntimeScore:        round3(vruntimeScore),
		ContextAffinityScore: round3(contextScore),
		MemoryPressure:       round3(memoryPressure),
		PidsPressure:         round3(pidsPressure),
		CPUThrottlePressure:  round3(cpuThrottle),
		PSIPressure:          round3(global.PSIPressure),
		DependencyReady:      candidate.State == avp.StateReady,
		SpawnPriority:        round3(spawnPriority),
		FinalScore:           round3(finalScore),
	}
}

func (s *Scheduler) makeDecisionLocked(taskID string, candidates []avp.AVP, selected avp.AVP, reason string, shared map[string]int, scoreByAgent map[string]CandidateDecision) DecisionLog {
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
	now := time.Now()
	decisionID := now.Format("20060102150405.000000000")
	resourcePressure := normalizeResourcePressure(s.resourcePressure)
	selectedScore := CandidateDecision{AgentID: selected.AgentID, TaskID: selected.TaskID, DependencyReady: selected.State == avp.StateReady}
	details := make([]CandidateDecision, 0, len(scoreByAgent))
	if scoreByAgent != nil {
		for _, candidate := range candidates {
			score := scoreByAgent[candidate.AgentID]
			details = append(details, score)
			if candidate.AgentID == selected.AgentID {
				selectedScore = score
			}
			resourcePressure.MemoryPressure = maxFloat(resourcePressure.MemoryPressure, score.MemoryPressure)
			resourcePressure.PidsPressure = maxFloat(resourcePressure.PidsPressure, score.PidsPressure)
			resourcePressure.CPUThrottlePressure = maxFloat(resourcePressure.CPUThrottlePressure, score.CPUThrottlePressure)
			resourcePressure.PSIPressure = maxFloat(resourcePressure.PSIPressure, score.PSIPressure)
		}
	}
	return DecisionLog{
		ID:                   decisionID,
		DecisionID:           decisionID,
		Timestamp:            now.UTC().Format(time.RFC3339Nano),
		TaskID:               taskID,
		SelectedTask:         taskID,
		Candidates:           ids,
		CandidateDetails:     details,
		SelectedAgent:        selected.AgentID,
		Policy:               s.policy,
		Reason:               reason,
		VRuntimeBefore:       before,
		VRuntimeAfter:        after,
		SharedPages:          shared,
		VRuntimeScore:        selectedScore.VRuntimeScore,
		ContextAffinityScore: selectedScore.ContextAffinityScore,
		MemoryPressure:       round3(resourcePressure.MemoryPressure),
		PidsPressure:         round3(resourcePressure.PidsPressure),
		CPUThrottlePressure:  round3(resourcePressure.CPUThrottlePressure),
		PSIPressure:          round3(resourcePressure.PSIPressure),
		DependencyReady:      selectedScore.DependencyReady,
		SpawnPriority:        selectedScore.SpawnPriority,
		FinalScore:           selectedScore.FinalScore,
		EvidenceMode:         resourcePressure.EvidenceMode,
		FallbackReason:       resourcePressure.FallbackReason,
		ResourcePressure:     resourcePressure,
		ScoreByAgent:         scoreByAgent,
		CreatedAt:            now.UnixMilli(),
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

func normalizeResourcePressure(pressure ResourcePressure) ResourcePressure {
	pressure.MemoryPressure = round3(clamp(pressure.MemoryPressure, 0, 1))
	pressure.PidsPressure = round3(clamp(pressure.PidsPressure, 0, 1))
	pressure.CPUThrottlePressure = round3(clamp(pressure.CPUThrottlePressure, 0, 1))
	pressure.PSIPressure = round3(clamp(pressure.PSIPressure, 0, 1))
	if pressure.EvidenceMode == "" {
		pressure.EvidenceMode = evidence.ModeDegraded
	}
	if !evidence.IsValid(pressure.EvidenceMode) {
		pressure.EvidenceMode = evidence.ModeDegraded
	}
	if pressure.EvidenceMode == evidence.ModeDegraded && pressure.FallbackReason == "" {
		pressure.FallbackReason = "resource pressure sampler not configured or local cgroup pressure files unavailable"
	}
	return pressure
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
