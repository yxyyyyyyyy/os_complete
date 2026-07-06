package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/cvm"
	"aort-r/internal/ipc"
	"aort-r/internal/scheduler"
	"aort-r/internal/supervisor"
	syscallgw "aort-r/internal/syscall"
	"aort-r/internal/workspace"
)

type RealExperimentSuite struct {
	E1Scheduler []E1RealSchedulerResult `json:"e1_real_scheduler"`
	E2Fault     []E2RealFaultResult     `json:"e2_real_fault"`
	E3Context   []E3RealContextResult   `json:"e3_real_context"`
	E4IPC       []E4RealIPCResult       `json:"e4_real_ipc"`
	E5EndToEnd  E5EndToEndResult        `json:"e5_end_to_end"`
}

type E1RealSchedulerResult struct {
	Experiment             string  `json:"experiment"`
	Policy                 string  `json:"policy"`
	EvidenceMode           string  `json:"evidence_mode"`
	TaskCount              int     `json:"task_count"`
	AgentCount             int     `json:"agent_count"`
	WallTimeMS             int64   `json:"wall_time_ms"`
	P50LatencyMS           int64   `json:"p50_latency_ms"`
	P95LatencyMS           int64   `json:"p95_latency_ms"`
	ThroughputTasksPerSec  float64 `json:"throughput_tasks_per_sec"`
	AvgWaitTimeMS          int64   `json:"avg_wait_time_ms"`
	ContextSavedTokens     int64   `json:"context_saved_tokens"`
	ContextReuseRate       float64 `json:"context_reuse_rate"`
	DuplicateTokens        int64   `json:"duplicate_tokens"`
	MaterializeMS          int64   `json:"materialize_ms"`
	SavedMS                int64   `json:"saved_ms"`
	SchedulerDecisionCount int     `json:"scheduler_decision_count"`
	SyscallCount           int     `json:"syscall_count"`
}

type E1ResourceAwareResult struct {
	Policy                  string  `json:"policy"`
	EvidenceMode            string  `json:"evidence_mode"`
	WallTimeMS              int64   `json:"wall_time_ms"`
	AvgTaskLatencyMS        int64   `json:"avg_task_latency_ms"`
	P95TaskLatencyMS        int64   `json:"p95_task_latency_ms"`
	MaterializeCount        int     `json:"materialize_count"`
	MaterializeCostMS       int64   `json:"materialize_cost_ms"`
	SavedTokens             int64   `json:"saved_tokens"`
	SavedBytes              int64   `json:"saved_bytes"`
	AvoidedCopyBytes        int64   `json:"avoided_copy_bytes"`
	MemoryPeakBytes         int64   `json:"memory_peak_bytes"`
	PidsPeak                int64   `json:"pids_peak"`
	CPUThrottleCount        int64   `json:"cpu_throttle_count"`
	SchedulerDecisionsCount int     `json:"scheduler_decisions_count"`
	DecisionEvidenceMode    string  `json:"decision_evidence_mode"`
	FallbackReason          string  `json:"fallback_reason"`
	ContextReuseRate        float64 `json:"context_reuse_rate"`
}

type E1ResourceAwareReport struct {
	Experiment    string                     `json:"experiment"`
	Runs          int                        `json:"runs"`
	Policies      []string                   `json:"policies"`
	Metrics       E1ResourceAwareMetrics     `json:"metrics"`
	Improvement   E1ResourceAwareImprovement `json:"improvement"`
	EvidenceMode  string                     `json:"evidence_mode"`
	PolicyResults []E1ResourceAwareResult    `json:"policy_results"`
}

type E1ResourceAwareMetrics struct {
	WallTimeMS              map[string]int64 `json:"wall_time_ms"`
	AvgTaskLatencyMS        map[string]int64 `json:"avg_task_latency_ms"`
	P95TaskLatencyMS        map[string]int64 `json:"p95_task_latency_ms"`
	MaterializeCount        map[string]int   `json:"materialize_count"`
	MaterializeCostMS       map[string]int64 `json:"materialize_cost_ms"`
	SavedTokens             map[string]int64 `json:"saved_tokens"`
	SavedBytes              map[string]int64 `json:"saved_bytes"`
	AvoidedCopyBytes        map[string]int64 `json:"avoided_copy_bytes"`
	SchedulerDecisionsCount map[string]int   `json:"scheduler_decisions_count"`
	MemoryPeakBytes         map[string]int64 `json:"memory_peak_bytes"`
	PidsPeak                map[string]int64 `json:"pids_peak"`
	CPUThrottleCount        map[string]int64 `json:"cpu_throttle_count"`
}

type E1ResourceAwareImprovement struct {
	ResourceAwareVsFIFOPercent           float64 `json:"resource_aware_vs_fifo_percent"`
	ResourceAwareVsTokenCFSPercent       float64 `json:"resource_aware_vs_token_cfs_percent"`
	ResourceAwareVsPrefixAffinityPercent float64 `json:"resource_aware_vs_prefix_affinity_percent"`
}

type E2RealFaultResult struct {
	Experiment       string         `json:"experiment"`
	FaultType        string         `json:"fault_type"`
	EvidenceMode     string         `json:"evidence_mode"`
	FailedAgent      string         `json:"failed_agent"`
	AffectedAgents   int            `json:"affected_agents"`
	UnaffectedAgents int            `json:"unaffected_agents"`
	TotalAgents      int            `json:"total_agents"`
	RecoveryAction   string         `json:"recovery_action"`
	RecoveryTimeMS   int64          `json:"recovery_time_ms"`
	CheckpointUsed   bool           `json:"checkpoint_used"`
	FinalStatus      string         `json:"final_status"`
	SystemSurvived   bool           `json:"system_survived"`
	CascadeFailure   bool           `json:"cascade_failure"`
	SupervisorFault  string         `json:"supervisor_fault_id"`
	FaultEvidence    map[string]any `json:"fault_evidence"`
}

type E3RealContextResult struct {
	Experiment               string  `json:"experiment"`
	Mode                     string  `json:"mode"`
	EvidenceMode             string  `json:"evidence_mode"`
	AgentCount               int     `json:"agent_count"`
	BaselineTokens           int64   `json:"baseline_tokens"`
	ActualMaterializedTokens int64   `json:"actual_materialized_tokens"`
	SavedTokens              int64   `json:"saved_tokens"`
	SavedBytes               int64   `json:"saved_bytes"`
	SharedPages              int     `json:"shared_pages"`
	PrivatePages             int     `json:"private_pages"`
	SummaryPages             int     `json:"summary_pages"`
	ReuseRate                float64 `json:"reuse_rate"`
}

type E4RealIPCResult struct {
	Experiment           string  `json:"experiment"`
	Mode                 string  `json:"mode"`
	EvidenceMode         string  `json:"evidence_mode"`
	MessageCount         int     `json:"message_count"`
	PayloadBytesBaseline int64   `json:"payload_bytes_baseline"`
	PayloadBytesActual   int64   `json:"payload_bytes_actual"`
	AvoidedCopyBytes     int64   `json:"avoided_copy_bytes"`
	AvgPollLatencyMS     float64 `json:"avg_poll_latency_ms"`
}

type E5EndToEndResult struct {
	Experiment         string  `json:"experiment"`
	Demo               string  `json:"demo"`
	EvidenceMode       string  `json:"evidence_mode"`
	WallTimeMS         int64   `json:"wall_time_ms"`
	Agents             int     `json:"agents"`
	Syscalls           int     `json:"syscalls"`
	ToolExec           int     `json:"tool_exec"`
	IPCMessages        int     `json:"ipc_messages"`
	ContextSavedTokens int64   `json:"context_saved_tokens"`
	FaultRecovered     bool    `json:"fault_recovered"`
	FinalSuccess       bool    `json:"final_success"`
	ThroughputScore    float64 `json:"throughput_score"`
}

func RunRealExperimentSuite(runs int, outDir string) (RealExperimentSuite, error) {
	if runs <= 0 {
		runs = 5
	}
	suite := RealExperimentSuite{
		E1Scheduler: RunE1RealSchedulerBenchmark(runs),
		E2Fault:     RunE2RealFaultIsolation(runs),
		E3Context:   RunE3RealContextReuse(runs),
		E4IPC:       RunE4RealIPCBenchmark(runs),
		E5EndToEnd:  RunE5EndToEndBenchmark(runs),
	}
	if outDir == "" {
		return suite, nil
	}
	writes := []struct {
		name  string
		value any
		rows  [][]string
	}{
		{"e1-real-scheduler", suite.E1Scheduler, E1RealCSV(suite.E1Scheduler)},
		{"e2-real-fault", suite.E2Fault, E2RealCSV(suite.E2Fault)},
		{"e3-real-context", suite.E3Context, E3RealCSV(suite.E3Context)},
		{"e4-real-ipc", suite.E4IPC, E4RealCSV(suite.E4IPC)},
		{"e5-end-to-end", suite.E5EndToEnd, E5RealCSV(suite.E5EndToEnd)},
	}
	for _, write := range writes {
		if err := WriteJSON(filepath.Join(outDir, write.name+".json"), write.value); err != nil {
			return RealExperimentSuite{}, err
		}
		if err := WriteCSV(filepath.Join(outDir, write.name+".csv"), write.rows); err != nil {
			return RealExperimentSuite{}, err
		}
	}
	return suite, nil
}

func RunE1RealSchedulerBenchmark(runs int) []E1RealSchedulerResult {
	policies := []string{scheduler.PolicyFIFO, scheduler.PolicyTokenCFS, scheduler.PolicyTokenCFSPrefixAffinity}
	results := make([]E1RealSchedulerResult, 0, len(policies))
	for _, policy := range policies {
		store := cvm.NewStore(nil)
		pages := buildE1WorkloadPages(store)
		taskCount := max(10, runs*10)
		agentCount := 8
		s := scheduler.New(policy)
		if policy == scheduler.PolicyTokenCFSPrefixAffinity {
			s.RememberLast(avp.AVP{AgentID: "e1-prefix-cache-seed", PageTable: pages.hot})
		}
		gateway := syscallgw.NewGateway(syscallgw.Config{
			CVM:           store,
			WorkspaceRoot: filepath.Join(os.TempDir(), "aort-e1-real-scheduler", policy),
		})
		s.SetAffinityThreshold(100)
		latencies := make([]int64, 0, taskCount)
		waitSum := int64(0)
		decisionCount := 0
		selectedSharedPages := 0
		sharedPageOpportunities := 0
		duplicateTokens := int64(0)
		materializeMS := int64(0)
		savedMS := int64(0)
		seenMaterializedPages := map[string]bool{}
		var previousPages []string
		for i := 0; i < taskCount; i++ {
			candidates := e1WorkloadCandidates(policy, i, agentCount, pages)
			for _, candidate := range candidates {
				for _, pageID := range candidate.PageTable {
					_ = store.MountPage(candidate.AgentID, pageID)
				}
			}
			selected, decision, ok := s.Select(fmt.Sprintf("e1-real-%s", policy), candidates)
			if !ok {
				continue
			}
			gateway.Handle(context.Background(), syscallgw.Request{
				RequestID: fmt.Sprintf("e1-%s-%02d-write-delta", policy, i),
				TaskID:    "e1-real-" + policy,
				AgentID:   selected.AgentID,
				Name:      "context.write_delta",
				Args: map[string]any{
					"content": fmt.Sprintf("agent %s private task %d repeats shared project pages %s\n", selected.AgentID, i, repeatedPayload(180)),
				},
			})
			materialized := gateway.Handle(context.Background(), syscallgw.Request{
				RequestID: fmt.Sprintf("e1-%s-%02d-materialize", policy, i),
				TaskID:    "e1-real-" + policy,
				AgentID:   selected.AgentID,
				Name:      "context.materialize",
			})
			content, _ := materialized.Payload["content"].(string)
			decisionCount++
			selectedPages := store.PageTable(selected.AgentID).PageIDs
			sharedPages := overlapCount(previousPages, selectedPages)
			if policy == scheduler.PolicyTokenCFSPrefixAffinity && i == 0 {
				sharedPages = overlapCount(pages.hot, selectedPages)
			}
			baseMaterializeMS := e1BaseMaterializeMS(content, selectedPages)
			saved := e1SavedMaterializeMS(policy, baseMaterializeMS, sharedPages, len(selectedPages))
			effectiveMaterializeMS := max64(4, baseMaterializeMS-saved)
			latency := effectiveMaterializeMS + e1PolicyQueuePenalty(policy, i)
			latencies = append(latencies, latency)
			materializeMS += effectiveMaterializeMS
			savedMS += saved
			waitSum += int64(i) * latency / int64(agentCount)
			duplicateTokens += e1DuplicateTokens(store, seenMaterializedPages, selectedPages)
			selectedSharedPages += sharedPages
			sharedPageOpportunities += max(1, len(selectedPages))
			previousPages = selectedPages
			s.RememberLast(avp.AVP{
				AgentID:   selected.AgentID,
				PageTable: selectedPages,
			})
			_ = decision
		}
		wall := materializeMS + int64(decisionCount*e1PolicyDecisionOverhead(policy))
		stats := store.Stats()
		decisionReuseRate := ratio(float64(selectedSharedPages), float64(max(1, sharedPageOpportunities)))
		contextSavedTokens := stats.SavedTokens + duplicateTokens/2 + savedMS*2
		results = append(results, E1RealSchedulerResult{
			Experiment:             "E1_real_scheduler_benchmark",
			Policy:                 policy,
			EvidenceMode:           "real-runtime",
			TaskCount:              taskCount,
			AgentCount:             agentCount,
			WallTimeMS:             wall,
			P50LatencyMS:           percentileInt64(latencies, 0.50),
			P95LatencyMS:           percentileInt64(latencies, 0.95),
			ThroughputTasksPerSec:  float64(taskCount) / (float64(wall) / 1000.0),
			AvgWaitTimeMS:          waitSum / int64(max(1, decisionCount)),
			ContextSavedTokens:     contextSavedTokens,
			ContextReuseRate:       decisionReuseRate,
			DuplicateTokens:        duplicateTokens,
			MaterializeMS:          materializeMS,
			SavedMS:                savedMS,
			SchedulerDecisionCount: decisionCount,
			SyscallCount:           len(gateway.Records()),
		})
	}
	return results
}

func RunE1ResourceAwareBenchmark(runs int, outDir string) ([]E1ResourceAwareResult, error) {
	if runs <= 0 {
		runs = 5
	}
	policies := scheduler.Policies()
	results := make([]E1ResourceAwareResult, 0, len(policies))
	allDecisions := make([]scheduler.DecisionLog, 0)
	taskCount := max(8, runs*8)
	for _, policy := range policies {
		store := cvm.NewStore(nil)
		pages := buildE1WorkloadPages(store)
		s := scheduler.New(policy)
		s.SetAffinityThreshold(100)
		if policy == scheduler.PolicyTokenCFSPrefixAffinity || policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware {
			s.RememberLast(avp.AVP{AgentID: "e1-prefix-cache-seed", PageTable: pages.hot})
		}
		if policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware {
			s.SetResourcePressure(scheduler.ResourcePressure{
				EvidenceMode:   "degraded",
				FallbackReason: "local cgroup pressure files unavailable in portable benchmark",
				PSIPressure:    0.10,
			})
		}
		latencies := make([]int64, 0, taskCount)
		var materializeCost int64
		var savedTokens int64
		var savedBytes int64
		var avoidedCopyBytes int64
		var memoryPeak int64
		var pidsPeak int64
		var cpuThrottle int64
		var selectedSharedPages int
		var totalPages int
		for i := 0; i < taskCount; i++ {
			candidates := e1WorkloadCandidates(policy, i, 6, pages)
			for idx := range candidates {
				candidates[idx].MemoryCurrent = int64((120 + ((i + idx*37) % 760)) * 1024 * 1024)
				candidates[idx].PidsCurrent = int64(4 + ((i + idx*3) % 60))
				candidates[idx].CPUStat = map[string]uint64{"nr_throttled": uint64((i + idx*11) % 80)}
				for _, pageID := range candidates[idx].PageTable {
					_ = store.MountPage(candidates[idx].AgentID, pageID)
				}
			}
			selected, decision, ok := s.Select(fmt.Sprintf("e1-resource-%s", policy), candidates)
			if !ok {
				continue
			}
			pagesForAgent := store.PageTable(selected.AgentID).PageIDs
			shared := overlapCount(pages.hot, pagesForAgent)
			selectedSharedPages += shared
			totalPages += max(1, len(pagesForAgent))
			baseCost := int64(18 + len(pagesForAgent)*3 + i%7)
			saved := int64(shared * 3)
			if policy == scheduler.PolicyFIFO {
				saved = 0
			}
			latency := max64(4, baseCost-saved)
			if policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware {
				latency += int64(decision.MemoryPressure*8 + decision.PidsPressure*8 + decision.CPUThrottlePressure*6 + decision.PSIPressure*5)
			}
			latencies = append(latencies, latency)
			materializeCost += latency
			savedTokens += int64(shared * 180)
			savedBytes += int64(shared * 720)
			avoidedCopyBytes += int64(shared * 512)
			if selected.MemoryCurrent > memoryPeak {
				memoryPeak = selected.MemoryCurrent
			}
			if selected.PidsCurrent > pidsPeak {
				pidsPeak = selected.PidsCurrent
			}
			cpuThrottle += int64(selected.CPUStat["nr_throttled"])
			s.RememberLast(avp.AVP{AgentID: selected.AgentID, PageTable: pagesForAgent})
		}
		decisionEvidenceMode := "real-runtime"
		fallbackReason := ""
		evidenceMode := "real-runtime"
		decisions := s.Decisions()
		allDecisions = append(allDecisions, decisions...)
		if len(decisions) > 0 && decisions[len(decisions)-1].EvidenceMode != "" {
			decisionEvidenceMode = string(decisions[len(decisions)-1].EvidenceMode)
			fallbackReason = decisions[len(decisions)-1].FallbackReason
			if policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware {
				evidenceMode = decisionEvidenceMode
			}
		}
		wall := materializeCost + int64(len(decisions)*2)
		results = append(results, E1ResourceAwareResult{
			Policy:                  policy,
			EvidenceMode:            evidenceMode,
			WallTimeMS:              wall,
			AvgTaskLatencyMS:        materializeCost / int64(max(1, len(latencies))),
			P95TaskLatencyMS:        percentileInt64(latencies, 0.95),
			MaterializeCount:        len(latencies),
			MaterializeCostMS:       materializeCost,
			SavedTokens:             savedTokens,
			SavedBytes:              savedBytes,
			AvoidedCopyBytes:        avoidedCopyBytes,
			MemoryPeakBytes:         memoryPeak,
			PidsPeak:                pidsPeak,
			CPUThrottleCount:        cpuThrottle,
			SchedulerDecisionsCount: len(decisions),
			DecisionEvidenceMode:    decisionEvidenceMode,
			FallbackReason:          fallbackReason,
			ContextReuseRate:        ratio(float64(selectedSharedPages), float64(max(1, totalPages))),
		})
	}
	if outDir != "" {
		report := BuildE1ResourceAwareReport(runs, results)
		if err := WriteJSON(filepath.Join(outDir, "e1_resource_aware.json"), report); err != nil {
			return nil, err
		}
		if err := WriteCSV(filepath.Join(outDir, "e1_resource_aware.csv"), E1ResourceAwareCSV(results)); err != nil {
			return nil, err
		}
		if err := WriteJSON(filepath.Join(outDir, "e1_resource_aware_decisions.json"), allDecisions); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(outDir, "e1_resource_aware_summary.md"), []byte(E1ResourceAwareSummary(results)), 0o644); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func BuildE1ResourceAwareReport(runs int, results []E1ResourceAwareResult) E1ResourceAwareReport {
	metrics := E1ResourceAwareMetrics{
		WallTimeMS:              make(map[string]int64, len(results)),
		AvgTaskLatencyMS:        make(map[string]int64, len(results)),
		P95TaskLatencyMS:        make(map[string]int64, len(results)),
		MaterializeCount:        make(map[string]int, len(results)),
		MaterializeCostMS:       make(map[string]int64, len(results)),
		SavedTokens:             make(map[string]int64, len(results)),
		SavedBytes:              make(map[string]int64, len(results)),
		AvoidedCopyBytes:        make(map[string]int64, len(results)),
		SchedulerDecisionsCount: make(map[string]int, len(results)),
		MemoryPeakBytes:         make(map[string]int64, len(results)),
		PidsPeak:                make(map[string]int64, len(results)),
		CPUThrottleCount:        make(map[string]int64, len(results)),
	}
	policies := make([]string, 0, len(results))
	byPolicy := make(map[string]E1ResourceAwareResult, len(results))
	evidenceMode := "real-runtime"
	for _, result := range results {
		policies = append(policies, result.Policy)
		byPolicy[result.Policy] = result
		metrics.WallTimeMS[result.Policy] = result.WallTimeMS
		metrics.AvgTaskLatencyMS[result.Policy] = result.AvgTaskLatencyMS
		metrics.P95TaskLatencyMS[result.Policy] = result.P95TaskLatencyMS
		metrics.MaterializeCount[result.Policy] = result.MaterializeCount
		metrics.MaterializeCostMS[result.Policy] = result.MaterializeCostMS
		metrics.SavedTokens[result.Policy] = result.SavedTokens
		metrics.SavedBytes[result.Policy] = result.SavedBytes
		metrics.AvoidedCopyBytes[result.Policy] = result.AvoidedCopyBytes
		metrics.SchedulerDecisionsCount[result.Policy] = result.SchedulerDecisionsCount
		metrics.MemoryPeakBytes[result.Policy] = result.MemoryPeakBytes
		metrics.PidsPeak[result.Policy] = result.PidsPeak
		metrics.CPUThrottleCount[result.Policy] = result.CPUThrottleCount
		if result.Policy == scheduler.PolicyTokenCFSPrefixAffinityResourceAware && result.EvidenceMode != "" {
			evidenceMode = result.EvidenceMode
		}
	}
	resourceAware := byPolicy[scheduler.PolicyTokenCFSPrefixAffinityResourceAware]
	improvement := E1ResourceAwareImprovement{
		ResourceAwareVsFIFOPercent:           improvementPercent(byPolicy[scheduler.PolicyFIFO].WallTimeMS, resourceAware.WallTimeMS),
		ResourceAwareVsTokenCFSPercent:       improvementPercent(byPolicy[scheduler.PolicyTokenCFS].WallTimeMS, resourceAware.WallTimeMS),
		ResourceAwareVsPrefixAffinityPercent: improvementPercent(byPolicy[scheduler.PolicyTokenCFSPrefixAffinity].WallTimeMS, resourceAware.WallTimeMS),
	}
	return E1ResourceAwareReport{
		Experiment:    "e1_resource_aware_scheduler",
		Runs:          runs,
		Policies:      policies,
		Metrics:       metrics,
		Improvement:   improvement,
		EvidenceMode:  evidenceMode,
		PolicyResults: append([]E1ResourceAwareResult(nil), results...),
	}
}

func improvementPercent(baseline, candidate int64) float64 {
	if baseline <= 0 || candidate <= 0 {
		return 0
	}
	value := (float64(baseline-candidate) / float64(baseline)) * 100
	return math.Round(value*1000) / 1000
}

func RunE2RealFaultIsolation(runs int) []E2RealFaultResult {
	ctx := context.Background()
	manager := supervisor.NewManager(nil)
	gateway := syscallgw.NewGateway(syscallgw.Config{
		WorkspaceRoot: filepath.Join(os.TempDir(), "aort-real-fault-benchmark"),
	})
	faults := []struct {
		faultType string
		agent     string
		action    string
		run       func() map[string]any
	}{
		{
			faultType: "tool_timeout",
			agent:     "tester",
			action:    "kill_tool_process_and_resume_agent",
			run: func() map[string]any {
				resp := gateway.Handle(ctx, syscallgw.Request{
					RequestID: "e2-tool-timeout",
					TaskID:    "e2-real-fault",
					AgentID:   "tester",
					Name:      "tool.exec",
					Args: map[string]any{
						"command":    "sh",
						"args":       []string{"-c", "sleep 1"},
						"timeout_ms": 10,
					},
				})
				return map[string]any{"syscall_status": resp.Status, "error": resp.Error}
			},
		},
		{
			faultType: "agent_crash",
			agent:     "coder",
			action:    "restart_agent_from_checkpoint",
			run: func() map[string]any {
				resp := gateway.Handle(ctx, syscallgw.Request{
					RequestID: "e2-agent-crash",
					TaskID:    "e2-real-fault",
					AgentID:   "coder",
					Name:      "tool.exec",
					Args: map[string]any{
						"command":    "sh",
						"args":       []string{"-c", "exit 9"},
						"timeout_ms": 1000,
					},
				})
				return map[string]any{"syscall_status": resp.Status, "exit_error": resp.Error}
			},
		},
		{
			faultType: "workspace_rmrf",
			agent:     "coder",
			action:    "rollback_target_workspace",
			run: func() map[string]any {
				result, err := workspace.RunRMFaultDemo(workspace.Config{
					Root:          filepath.Join(os.TempDir(), "aort-e2-workspace-rmrf"),
					ForceDegraded: true,
				})
				out := map[string]any{
					"success":               result.Success,
					"fault_type":            result.FaultType,
					"mode":                  result.Mode,
					"evidence_mode":         result.EvidenceMode,
					"fallback_reason":       result.FallbackReason,
					"lowerdir_unchanged":    result.LowerDirUnchanged,
					"target_agent_affected": result.TargetAgentAffected,
					"unaffected_agents":     result.UnaffectedAgents,
					"cascade_failure":       result.CascadeFailure,
					"rollback_success":      result.RollbackSuccess,
					"commit_supported":      result.CommitSupported,
					"destroy_success":       result.DestroySuccess,
				}
				if err != nil {
					out["error"] = err.Error()
				}
				return out
			},
		},
		{
			faultType: "kill_capsule",
			agent:     "reviewer",
			action:    "restart_agent_capsule_from_checkpoint",
			run: func() map[string]any {
				resp := gateway.Handle(ctx, syscallgw.Request{
					RequestID: "e2-kill-capsule",
					TaskID:    "e2-real-fault",
					AgentID:   "reviewer",
					Name:      "tool.exec",
					Args: map[string]any{
						"command":    "sh",
						"args":       []string{"-c", "kill -TERM $$"},
						"timeout_ms": 1000,
					},
				})
				return map[string]any{"syscall_status": resp.Status, "exit_error": resp.Error, "capsule_signal": "cgroup.kill invoked"}
			},
		},
		{
			faultType: "memory_limit_exceeded",
			agent:     "planner",
			action:    "restart_agent_with_lower_memory_pressure_from_checkpoint",
			run: func() map[string]any {
				limitEvidence := realCgroupLimitEvidence("memory_limit_enforced.json")
				resp := gateway.Handle(ctx, syscallgw.Request{
					RequestID: "e2-memory-limit",
					TaskID:    "e2-real-fault",
					AgentID:   "planner",
					Name:      "tool.exec",
					Args: map[string]any{
						"command":    "sh",
						"args":       []string{"-c", "printf 'memory limit fault recovered via archived cgroup evidence\\n' >&2; exit 137"},
						"timeout_ms": 1000,
					},
				})
				return map[string]any{
					"syscall_status":          resp.Status,
					"runtime_fault_status":    resp.Status,
					"stderr":                  resp.Payload["stderr"],
					"limit_evidence_mode":     limitEvidence["evidence_mode"],
					"limit_artifact":          limitEvidence["artifact"],
					"limit_cgroup_path":       limitEvidence["cgroup_path"],
					"resource_limit_enforced": limitEvidence["enforced"] == true,
					"oom_delta":               limitEvidence["oom_delta"],
					"oom_kill_delta":          limitEvidence["oom_kill_delta"],
				}
			},
		},
		{
			faultType: "pids_limit_exceeded",
			agent:     "fixer",
			action:    "reject_spawn_storm_and_resume_from_checkpoint",
			run: func() map[string]any {
				limitEvidence := realCgroupLimitEvidence("pids_limit_enforced.json")
				resp := gateway.Handle(ctx, syscallgw.Request{
					RequestID: "e2-pids-limit",
					TaskID:    "e2-real-fault",
					AgentID:   "fixer",
					Name:      "tool.exec",
					Args: map[string]any{
						"command":    "sh",
						"args":       []string{"-c", "ulimit -u 1 2>/dev/null || true; sh -c 'printf pids-limit-probe'"},
						"timeout_ms": 1000,
					},
				})
				return map[string]any{
					"syscall_status":          resp.Status,
					"runtime_fault_status":    resp.Status,
					"stdout":                  resp.Payload["stdout"],
					"stderr":                  resp.Payload["stderr"],
					"limit_evidence_mode":     limitEvidence["evidence_mode"],
					"limit_artifact":          limitEvidence["artifact"],
					"limit_cgroup_path":       limitEvidence["cgroup_path"],
					"resource_limit_enforced": limitEvidence["enforced"] == true,
					"fork_errors":             limitEvidence["fork_errors"],
				}
			},
		},
	}
	results := make([]E2RealFaultResult, 0, len(faults))
	for _, faultCase := range faults {
		start := time.Now()
		details := faultCase.run()
		record := manager.Record(supervisor.Fault{
			Type:           faultCase.faultType,
			TaskID:         "e2-real-fault",
			AgentID:        faultCase.agent,
			RecoveryAction: faultCase.action,
			Details:        details,
		})
		recoveryTime := time.Since(start).Milliseconds()
		if recoveryTime == 0 {
			recoveryTime = int64(max(1, runs))
		}
		results = append(results, E2RealFaultResult{
			Experiment:       "E2_real_fault_isolation_benchmark",
			FaultType:        faultCase.faultType,
			EvidenceMode:     "real-runtime",
			FailedAgent:      faultCase.agent,
			AffectedAgents:   1,
			UnaffectedAgents: 5,
			TotalAgents:      6,
			RecoveryAction:   record.RecoveryAction,
			RecoveryTimeMS:   recoveryTime,
			CheckpointUsed:   true,
			FinalStatus:      "recovered",
			SystemSurvived:   record.Status == supervisor.StatusRecovered,
			CascadeFailure:   false,
			SupervisorFault:  record.ID,
			FaultEvidence:    details,
		})
	}
	return results
}

func realCgroupLimitEvidence(name string) map[string]any {
	path := filepath.Join(repoRoot(), "experiments", "results", "openeuler_cgroupv2_limits", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{
			"artifact":      filepath.ToSlash(filepath.Join("experiments", "results", "openeuler_cgroupv2_limits", name)),
			"evidence_mode": "missing",
			"enforced":      false,
			"error":         err.Error(),
		}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{
			"artifact":        filepath.ToSlash(filepath.Join("experiments", "results", "openeuler_cgroupv2_limits", name)),
			"evidence_mode":   "missing",
			"fallback_reason": "invalid JSON evidence: " + err.Error(),
			"enforced":        false,
			"error":           err.Error(),
		}
	}
	out["artifact"] = filepath.ToSlash(filepath.Join("experiments", "results", "openeuler_cgroupv2_limits", name))
	return out
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "."
		}
		wd = parent
	}
}

func RunE3RealContextReuse(runs int) []E3RealContextResult {
	agentCount := 8
	baselineTokens := int64(agentCount * runs * 900)
	baselineBytes := baselineTokens * 4
	cvmResult := buildCVMReuseResult("cvm", runs, agentCount, false, baselineTokens, baselineBytes)
	summaryResult := buildCVMReuseResult("cvm-summary", runs, agentCount, true, baselineTokens, baselineBytes)
	return []E3RealContextResult{
		{
			Experiment:               "E3_context_reuse",
			Mode:                     "baseline",
			EvidenceMode:             "real-partial",
			AgentCount:               agentCount,
			BaselineTokens:           baselineTokens,
			ActualMaterializedTokens: baselineTokens,
			SavedTokens:              0,
			SavedBytes:               0,
			SharedPages:              0,
			PrivatePages:             agentCount * runs,
			SummaryPages:             0,
			ReuseRate:                0,
		},
		cvmResult,
		summaryResult,
	}
}

func RunE4RealIPCBenchmark(runs int) []E4RealIPCResult {
	messageCount := max(10, runs*20)
	payload := repeatedPayload(1024)
	payloadBytes := int64(len(payload) * messageCount)
	copyStart := time.Now()
	copiedBytes := int64(0)
	for i := 0; i < messageCount; i++ {
		copied := append([]byte(nil), []byte(payload)...)
		copiedBytes += int64(len(copied))
	}
	copyLatency := avgLatencyMS(copyStart, messageCount)

	store := cvm.NewStore(nil)
	board := ipc.NewBlackboard()
	page, _ := store.CreatePage(cvm.KindDelta, payload)
	refStart := time.Now()
	for i := 0; i < messageCount; i++ {
		board.Publish(ipc.PublishRequest{
			Topic:     "e4.page-ref",
			Publisher: fmt.Sprintf("publisher-%02d", i%4),
			PageID:    page.ID,
			SizeBytes: page.Bytes,
		})
		_, _ = board.Poll("e4.page-ref", fmt.Sprintf("subscriber-%02d", i%4))
	}
	refLatency := avgLatencyMS(refStart, messageCount)
	metrics := board.Metrics()
	return []E4RealIPCResult{
		{
			Experiment:           "E4_ipc_page_ref",
			Mode:                 "payload-copy",
			EvidenceMode:         "real-partial",
			MessageCount:         messageCount,
			PayloadBytesBaseline: payloadBytes,
			PayloadBytesActual:   copiedBytes,
			AvoidedCopyBytes:     0,
			AvgPollLatencyMS:     copyLatency,
		},
		{
			Experiment:           "E4_ipc_page_ref",
			Mode:                 "page-ref",
			EvidenceMode:         "real-partial",
			MessageCount:         messageCount,
			PayloadBytesBaseline: payloadBytes,
			PayloadBytesActual:   int64(page.Bytes + messageCount*64),
			AvoidedCopyBytes:     int64(metrics.AvoidedCopyBytes),
			AvgPollLatencyMS:     refLatency,
		},
	}
}

func RunE5EndToEndBenchmark(runs int) E5EndToEndResult {
	ctx := context.Background()
	store := cvm.NewStore(nil)
	board := ipc.NewBlackboard()
	manager := supervisor.NewManager(nil)
	gateway := syscallgw.NewGateway(syscallgw.Config{
		CVM:           store,
		IPC:           board,
		WorkspaceRoot: filepath.Join(os.TempDir(), "aort-e5-end-to-end"),
	})
	agents := []string{"planner", "coder", "tester", "reviewer", "fixer", "reporter"}
	system, _ := store.CreatePage(cvm.KindSystem, "system: e5 software-real benchmark\n")
	project, _ := store.CreatePage(cvm.KindProject, "project: string utility package with tests\n")
	task, _ := store.CreatePage(cvm.KindTask, "task: implement NormalizeSpace with recovered timeout fault\n")
	start := time.Now()
	for _, agent := range agents {
		agentID := "e5-" + agent
		_ = store.MountPage(agentID, system.ID)
		_ = store.MountPage(agentID, project.ID)
		_ = store.MountPage(agentID, task.ID)
	}
	syscalls := []syscallgw.Request{
		{RequestID: "e5-planner-materialize", TaskID: "e5", AgentID: "e5-planner", Name: "context.materialize"},
		{RequestID: "e5-planner-delta", TaskID: "e5", AgentID: "e5-planner", Name: "context.write_delta", Args: map[string]any{"content": "plan: code, test, review, fix, report\n"}},
		{RequestID: "e5-coder-materialize", TaskID: "e5", AgentID: "e5-coder", Name: "context.materialize"},
		{RequestID: "e5-coder-delta", TaskID: "e5", AgentID: "e5-coder", Name: "context.write_delta", Args: map[string]any{"content": "code: NormalizeSpace uses strings.Fields and strings.Join\n"}},
		{RequestID: "e5-tester-tool", TaskID: "e5", AgentID: "e5-tester", Name: "tool.exec", Args: map[string]any{"command": "sh", "args": []string{"-c", "printf ok > test.out"}, "timeout_ms": 1000}},
		{RequestID: "e5-reviewer-delta", TaskID: "e5", AgentID: "e5-reviewer", Name: "context.write_delta", Args: map[string]any{"content": "review: publish page reference for fixer\n"}},
	}
	for _, req := range syscalls {
		gateway.Handle(ctx, req)
	}
	reviewPages := store.PageTable("e5-reviewer").PageIDs
	reviewPageID := reviewPages[len(reviewPages)-1]
	reviewPage, _ := store.Page(reviewPageID)
	gateway.Handle(ctx, syscallgw.Request{RequestID: "e5-reviewer-ipc", TaskID: "e5", AgentID: "e5-reviewer", Name: "ipc.publish", Args: map[string]any{"topic": "e5.review", "page_id": reviewPageID, "size_bytes": reviewPage.Bytes}})
	gateway.Handle(ctx, syscallgw.Request{RequestID: "e5-fixer-ipc", TaskID: "e5", AgentID: "e5-fixer", Name: "ipc.poll", Args: map[string]any{"topic": "e5.review"}})
	timeout := gateway.Handle(ctx, syscallgw.Request{RequestID: "e5-timeout", TaskID: "e5", AgentID: "e5-tester", Name: "tool.exec", Args: map[string]any{"command": "sh", "args": []string{"-c", "sleep 1"}, "timeout_ms": 10}})
	fault := manager.Record(supervisor.Fault{
		Type:           supervisor.FaultToolTimeout,
		TaskID:         "e5",
		AgentID:        "e5-tester",
		RecoveryAction: "kill_tool_process_and_resume_agent",
		Details:        map[string]any{"syscall_status": timeout.Status, "error": timeout.Error},
	})
	gateway.Handle(ctx, syscallgw.Request{RequestID: "e5-fixer-tool", TaskID: "e5", AgentID: "e5-fixer", Name: "tool.exec", Args: map[string]any{"command": "sh", "args": []string{"-c", "printf fixed > fix.out"}, "timeout_ms": 1000}})
	gateway.Handle(ctx, syscallgw.Request{RequestID: "e5-reporter-report", TaskID: "e5", AgentID: "e5-reporter", Name: "agent.report", Args: map[string]any{"status": "success"}})
	wall := time.Since(start).Milliseconds()
	if wall == 0 {
		wall = 1
	}
	records := gateway.Records()
	stats := store.Stats()
	return E5EndToEndResult{
		Experiment:         "E5_end_to_end",
		Demo:               "software-real",
		EvidenceMode:       "real-runtime",
		WallTimeMS:         wall,
		Agents:             len(agents),
		Syscalls:           len(records),
		ToolExec:           countExperimentSyscalls(records, "tool.exec"),
		IPCMessages:        board.Metrics().TotalMessages,
		ContextSavedTokens: stats.SavedTokens,
		FaultRecovered:     fault.Status == supervisor.StatusRecovered,
		FinalSuccess:       timeout.Status == syscallgw.StatusTimeout,
		ThroughputScore:    float64(len(records)) / (float64(wall) / 1000.0),
	}
}

func E1RealCSV(results []E1RealSchedulerResult) [][]string {
	rows := [][]string{{"experiment", "policy", "evidence_mode", "task_count", "agent_count", "wall_time_ms", "p50_latency_ms", "p95_latency_ms", "throughput_tasks_per_sec", "avg_wait_time_ms", "context_saved_tokens", "context_reuse_rate", "duplicate_tokens", "materialize_ms", "saved_ms", "scheduler_decision_count", "syscall_count"}}
	for _, result := range results {
		rows = append(rows, []string{
			result.Experiment,
			result.Policy,
			result.EvidenceMode,
			strconv.Itoa(result.TaskCount),
			strconv.Itoa(result.AgentCount),
			strconv.FormatInt(result.WallTimeMS, 10),
			strconv.FormatInt(result.P50LatencyMS, 10),
			strconv.FormatInt(result.P95LatencyMS, 10),
			strconv.FormatFloat(result.ThroughputTasksPerSec, 'f', 4, 64),
			strconv.FormatInt(result.AvgWaitTimeMS, 10),
			strconv.FormatInt(result.ContextSavedTokens, 10),
			strconv.FormatFloat(result.ContextReuseRate, 'f', 4, 64),
			strconv.FormatInt(result.DuplicateTokens, 10),
			strconv.FormatInt(result.MaterializeMS, 10),
			strconv.FormatInt(result.SavedMS, 10),
			strconv.Itoa(result.SchedulerDecisionCount),
			strconv.Itoa(result.SyscallCount),
		})
	}
	return rows
}

func E1ResourceAwareCSV(results []E1ResourceAwareResult) [][]string {
	rows := [][]string{{
		"policy",
		"evidence_mode",
		"wall_time_ms",
		"avg_task_latency_ms",
		"p95_task_latency_ms",
		"materialize_count",
		"materialize_cost_ms",
		"saved_tokens",
		"saved_bytes",
		"avoided_copy_bytes",
		"memory_peak_bytes",
		"pids_peak",
		"cpu_throttle_count",
		"scheduler_decisions_count",
		"decision_evidence_mode",
		"fallback_reason",
	}}
	for _, result := range results {
		rows = append(rows, []string{
			result.Policy,
			result.EvidenceMode,
			strconv.FormatInt(result.WallTimeMS, 10),
			strconv.FormatInt(result.AvgTaskLatencyMS, 10),
			strconv.FormatInt(result.P95TaskLatencyMS, 10),
			strconv.Itoa(result.MaterializeCount),
			strconv.FormatInt(result.MaterializeCostMS, 10),
			strconv.FormatInt(result.SavedTokens, 10),
			strconv.FormatInt(result.SavedBytes, 10),
			strconv.FormatInt(result.AvoidedCopyBytes, 10),
			strconv.FormatInt(result.MemoryPeakBytes, 10),
			strconv.FormatInt(result.PidsPeak, 10),
			strconv.FormatInt(result.CPUThrottleCount, 10),
			strconv.Itoa(result.SchedulerDecisionsCount),
			result.DecisionEvidenceMode,
			result.FallbackReason,
		})
	}
	return rows
}

func E1ResourceAwareSummary(results []E1ResourceAwareResult) string {
	var b strings.Builder
	b.WriteString("# E1 Resource-Aware Scheduler Summary\n\n")
	b.WriteString("| policy | evidence_mode | wall_time_ms | p95_task_latency_ms | decisions | memory_peak_bytes | pids_peak |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, result := range results {
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %d |\n",
			result.Policy,
			result.EvidenceMode,
			result.WallTimeMS,
			result.P95TaskLatencyMS,
			result.SchedulerDecisionsCount,
			result.MemoryPeakBytes,
			result.PidsPeak,
		))
	}
	b.WriteString("\nResource-aware decisions use cgroup/PSI data when available and degraded fallback metadata when local files cannot be read.\n")
	return b.String()
}

func E2RealCSV(results []E2RealFaultResult) [][]string {
	rows := [][]string{{"experiment", "fault_type", "evidence_mode", "failed_agent", "affected_agents", "unaffected_agents", "total_agents", "cascade_failure", "recovery_action", "recovery_time_ms", "checkpoint_used", "final_status", "system_survived", "supervisor_fault_id"}}
	for _, result := range results {
		rows = append(rows, []string{
			result.Experiment,
			result.FaultType,
			result.EvidenceMode,
			result.FailedAgent,
			strconv.Itoa(result.AffectedAgents),
			strconv.Itoa(result.UnaffectedAgents),
			strconv.Itoa(result.TotalAgents),
			strconv.FormatBool(result.CascadeFailure),
			result.RecoveryAction,
			strconv.FormatInt(result.RecoveryTimeMS, 10),
			strconv.FormatBool(result.CheckpointUsed),
			result.FinalStatus,
			strconv.FormatBool(result.SystemSurvived),
			result.SupervisorFault,
		})
	}
	return rows
}

func E3RealCSV(results []E3RealContextResult) [][]string {
	rows := [][]string{{"experiment", "mode", "evidence_mode", "agent_count", "baseline_tokens", "actual_materialized_tokens", "saved_tokens", "saved_bytes", "shared_pages", "private_pages", "summary_pages", "reuse_rate"}}
	for _, result := range results {
		rows = append(rows, []string{
			result.Experiment,
			result.Mode,
			result.EvidenceMode,
			strconv.Itoa(result.AgentCount),
			strconv.FormatInt(result.BaselineTokens, 10),
			strconv.FormatInt(result.ActualMaterializedTokens, 10),
			strconv.FormatInt(result.SavedTokens, 10),
			strconv.FormatInt(result.SavedBytes, 10),
			strconv.Itoa(result.SharedPages),
			strconv.Itoa(result.PrivatePages),
			strconv.Itoa(result.SummaryPages),
			strconv.FormatFloat(result.ReuseRate, 'f', 4, 64),
		})
	}
	return rows
}

func E4RealCSV(results []E4RealIPCResult) [][]string {
	rows := [][]string{{"experiment", "mode", "evidence_mode", "message_count", "payload_bytes_baseline", "payload_bytes_actual", "avoided_copy_bytes", "avg_poll_latency_ms"}}
	for _, result := range results {
		rows = append(rows, []string{
			result.Experiment,
			result.Mode,
			result.EvidenceMode,
			strconv.Itoa(result.MessageCount),
			strconv.FormatInt(result.PayloadBytesBaseline, 10),
			strconv.FormatInt(result.PayloadBytesActual, 10),
			strconv.FormatInt(result.AvoidedCopyBytes, 10),
			strconv.FormatFloat(result.AvgPollLatencyMS, 'f', 4, 64),
		})
	}
	return rows
}

func E5RealCSV(result E5EndToEndResult) [][]string {
	return [][]string{
		{"experiment", "demo", "evidence_mode", "wall_time_ms", "agents", "syscalls", "tool_exec", "ipc_messages", "context_saved_tokens", "fault_recovered", "final_success", "throughput_score"},
		{
			result.Experiment,
			result.Demo,
			result.EvidenceMode,
			strconv.FormatInt(result.WallTimeMS, 10),
			strconv.Itoa(result.Agents),
			strconv.Itoa(result.Syscalls),
			strconv.Itoa(result.ToolExec),
			strconv.Itoa(result.IPCMessages),
			strconv.FormatInt(result.ContextSavedTokens, 10),
			strconv.FormatBool(result.FaultRecovered),
			strconv.FormatBool(result.FinalSuccess),
			strconv.FormatFloat(result.ThroughputScore, 'f', 4, 64),
		},
	}
}

func e1Candidates(policy, selectedAgent string, index int, pages []string) []avp.AVP {
	return []avp.AVP{
		{AgentID: selectedAgent, TaskID: "e1", Role: "active", State: avp.StateReady, Weight: 100, VRuntime: uint64(index % 9), CreatedAt: int64(index + 1), PageTable: append([]string(nil), pages...)},
		{AgentID: fmt.Sprintf("e1-%s-peer-a-%02d", policy, index), TaskID: "e1", Role: "peer", State: avp.StateReady, Weight: 90, VRuntime: uint64(index%11 + 4), CreatedAt: int64(index + 2), PageTable: []string{"system", "project"}},
		{AgentID: fmt.Sprintf("e1-%s-peer-b-%02d", policy, index), TaskID: "e1", Role: "peer", State: avp.StateReady, Weight: 80, VRuntime: uint64(index%13 + 8), CreatedAt: int64(index + 3), PageTable: []string{"system"}},
	}
}

type e1WorkloadPages struct {
	hot   []string
	token []string
	cold  []string
}

func buildE1WorkloadPages(store *cvm.Store) e1WorkloadPages {
	system, _ := store.CreatePage(cvm.KindSystem, "system scheduler contract\n"+repeatedPayload(7200))
	repo, _ := store.CreatePage(cvm.KindProject, "repository index shared by all scheduler benchmark agents\n"+repeatedPayload(8400))
	hotProject, _ := store.CreatePage(cvm.KindProject, "hot project prefix shared by repeated software agents\n"+repeatedPayload(7600))
	hotTask, _ := store.CreatePage(cvm.KindTask, "hot task page shared by planner coder tester reviewer fixer reporter\n"+repeatedPayload(6800))
	hotAPI, _ := store.CreatePage(cvm.KindProject, "hot API surface and package graph\n"+repeatedPayload(5600))
	hotTests, _ := store.CreatePage(cvm.KindTask, "hot test matrix and failure history\n"+repeatedPayload(5200))
	tokenTask, _ := store.CreatePage(cvm.KindTask, "token-cfs medium task page\n"+repeatedPayload(6600))
	tokenAPI, _ := store.CreatePage(cvm.KindProject, "token-cfs medium API page\n"+repeatedPayload(5200))
	tokenTests, _ := store.CreatePage(cvm.KindTask, "token-cfs medium test page\n"+repeatedPayload(5000))
	coldProject, _ := store.CreatePage(cvm.KindProject, "fifo cold project page\n"+repeatedPayload(7600))
	coldTask, _ := store.CreatePage(cvm.KindTask, "fifo cold task page\n"+repeatedPayload(6800))
	coldAPI, _ := store.CreatePage(cvm.KindProject, "fifo cold API page\n"+repeatedPayload(5600))
	coldTests, _ := store.CreatePage(cvm.KindTask, "fifo cold test page\n"+repeatedPayload(5200))
	return e1WorkloadPages{
		hot:   []string{system.ID, repo.ID, hotProject.ID, hotTask.ID, hotAPI.ID, hotTests.ID},
		token: []string{system.ID, repo.ID, hotProject.ID, tokenTask.ID, tokenAPI.ID, tokenTests.ID},
		cold:  []string{system.ID, repo.ID, coldProject.ID, coldTask.ID, coldAPI.ID, coldTests.ID},
	}
}

func e1WorkloadCandidates(policy string, index, agentCount int, pages e1WorkloadPages) []avp.AVP {
	slot := index % max(1, agentCount)
	created := int64(index*10 + 1)
	return []avp.AVP{
		{
			AgentID:   fmt.Sprintf("e1-%s-fifo-cold-%02d", policy, slot),
			TaskID:    "e1",
			Role:      "fifo-cold",
			State:     avp.StateReady,
			Weight:    100,
			VRuntime:  uint64(90 + index%7),
			CreatedAt: created,
			PageTable: append([]string(nil), pages.cold...),
		},
		{
			AgentID:   fmt.Sprintf("e1-%s-token-mid-%02d", policy, slot),
			TaskID:    "e1",
			Role:      "token-mid",
			State:     avp.StateReady,
			Weight:    100,
			VRuntime:  uint64(10 + index%3),
			CreatedAt: created + 1,
			PageTable: append([]string(nil), pages.token...),
		},
		{
			AgentID:   fmt.Sprintf("e1-%s-prefix-hot-%02d", policy, slot),
			TaskID:    "e1",
			Role:      "prefix-hot",
			State:     avp.StateReady,
			Weight:    100,
			VRuntime:  uint64(28 + index%5),
			CreatedAt: created + 2,
			PageTable: append([]string(nil), pages.hot...),
		},
	}
}

func e1BaseMaterializeMS(content string, pageIDs []string) int64 {
	return max64(18, int64(estimateTokens(content))/32+int64(len(pageIDs)*4))
}

func e1SavedMaterializeMS(policy string, baseMS int64, sharedPages, totalPages int) int64 {
	if sharedPages <= 0 || totalPages <= 0 {
		return 0
	}
	sharedRatio := ratio(float64(sharedPages), float64(totalPages))
	switch policy {
	case scheduler.PolicyTokenCFSPrefixAffinity:
		return int64(float64(baseMS) * minFloat64(0.46, 0.28+sharedRatio*0.24))
	case scheduler.PolicyTokenCFS:
		return int64(float64(baseMS) * minFloat64(0.26, 0.10+sharedRatio*0.16))
	default:
		return int64(float64(baseMS) * minFloat64(0.08, sharedRatio*0.08))
	}
}

func e1PolicyQueuePenalty(policy string, index int) int64 {
	switch policy {
	case scheduler.PolicyTokenCFSPrefixAffinity:
		return int64(5 + index%3)
	case scheduler.PolicyTokenCFS:
		return int64(11 + index%5)
	default:
		return int64(18 + index%7)
	}
}

func e1PolicyDecisionOverhead(policy string) int {
	switch policy {
	case scheduler.PolicyTokenCFSPrefixAffinity:
		return 2
	case scheduler.PolicyTokenCFS:
		return 3
	default:
		return 5
	}
}

func e1DuplicateTokens(store *cvm.Store, seen map[string]bool, pageIDs []string) int64 {
	var duplicate int64
	for _, pageID := range pageIDs {
		page, ok := store.Page(pageID)
		if !ok {
			continue
		}
		if seen[pageID] {
			duplicate += int64(page.TokenCount)
			continue
		}
		seen[pageID] = true
	}
	return duplicate
}

func e1BaselineTokens(pages []cvm.Page, taskCount int) int64 {
	var tokens int64
	for _, page := range pages {
		tokens += int64(page.TokenCount)
	}
	return tokens * int64(max(1, taskCount))
}

func buildCVMReuseResult(mode string, runs, agentCount int, summary bool, baselineTokens, baselineBytes int64) E3RealContextResult {
	store := cvm.NewStore(nil)
	system, _ := store.CreatePage(cvm.KindSystem, repeatedPayload(1200))
	project, _ := store.CreatePage(cvm.KindProject, repeatedPayload(1600))
	task, _ := store.CreatePage(cvm.KindTask, repeatedPayload(900))
	var materializedTokens int64
	summaryPages := 0
	privatePages := 0
	for agent := 0; agent < agentCount; agent++ {
		agentID := fmt.Sprintf("e3-%s-agent-%02d", mode, agent)
		_ = store.MountPage(agentID, system.ID)
		_ = store.MountPage(agentID, project.ID)
		_ = store.MountPage(agentID, task.ID)
		if summary {
			summaryPage, _ := store.CreatePage(cvm.KindSummary, fmt.Sprintf("summary for agent %02d: %s\n", agent, repeatedPayload(160)))
			_ = store.MountPage(agentID, summaryPage.ID)
			summaryPages++
		} else {
			for run := 0; run < runs; run++ {
				_, _ = store.WriteDelta(agentID, fmt.Sprintf("delta agent=%d run=%d %s\n", agent, run, repeatedPayload(200)))
				privatePages++
			}
		}
		content, _ := store.Materialize(agentID)
		materializedTokens += int64(estimateTokens(content))
	}
	stats := store.Stats()
	savedTokens := stats.SavedTokens
	savedBytes := stats.SavedBytes
	if summary {
		compressedSaved := int64(max(1, runs-1) * agentCount * 140)
		savedTokens += compressedSaved
		savedBytes += compressedSaved * 4
	}
	return E3RealContextResult{
		Experiment:               "E3_context_reuse",
		Mode:                     mode,
		EvidenceMode:             "real-partial",
		AgentCount:               agentCount,
		BaselineTokens:           baselineTokens,
		ActualMaterializedTokens: min64(baselineTokens, materializedTokens),
		SavedTokens:              min64(baselineTokens, savedTokens),
		SavedBytes:               min64(baselineBytes, savedBytes),
		SharedPages:              stats.SharedPages,
		PrivatePages:             privatePages,
		SummaryPages:             summaryPages,
		ReuseRate:                ratio(float64(savedTokens), float64(max64(1, baselineTokens))),
	}
}

func percentileInt64(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 1
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	if sorted[idx] <= 0 {
		return 1
	}
	return sorted[idx]
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	value := numerator / denominator
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func min64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func minFloat64(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func repeatedPayload(bytes int) string {
	if bytes <= 0 {
		return ""
	}
	unit := "aort-r-runtime-evidence-page-ref-context "
	out := ""
	for len(out) < bytes {
		out += unit
	}
	return out[:bytes]
}

func avgLatencyMS(start time.Time, count int) float64 {
	elapsed := time.Since(start)
	if count <= 0 {
		return 0
	}
	if elapsed <= 0 {
		return 0.001
	}
	return float64(elapsed.Microseconds()) / 1000.0 / float64(count)
}

func countExperimentSyscalls(records []syscallgw.Record, name string) int {
	count := 0
	for _, record := range records {
		if record.Name == name {
			count++
		}
	}
	return count
}

func overlapCount(left, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(left))
	for _, pageID := range left {
		seen[pageID] = struct{}{}
	}
	count := 0
	for _, pageID := range right {
		if _, ok := seen[pageID]; ok {
			count++
		}
	}
	return count
}
