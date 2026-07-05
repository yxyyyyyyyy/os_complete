package experiment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRealExperimentSuiteProducesP0Artifacts(t *testing.T) {
	dir := t.TempDir()
	suite, err := RunRealExperimentSuite(3, dir)
	if err != nil {
		t.Fatalf("RunRealExperimentSuite: %v", err)
	}

	if len(suite.E1Scheduler) != 3 {
		t.Fatalf("E1 policies = %#v", suite.E1Scheduler)
	}
	e1ByPolicy := map[string]E1RealSchedulerResult{}
	for _, result := range suite.E1Scheduler {
		e1ByPolicy[result.Policy] = result
		if result.Experiment != "E1_real_scheduler_benchmark" || result.EvidenceMode != "real-runtime" {
			t.Fatalf("bad E1 identity: %#v", result)
		}
		if result.WallTimeMS <= 0 || result.P95LatencyMS <= 0 || result.ThroughputTasksPerSec <= 0 {
			t.Fatalf("bad E1 timing: %#v", result)
		}
		if result.ContextReuseRate <= 0 || result.SchedulerDecisionCount == 0 {
			t.Fatalf("bad E1 runtime evidence: %#v", result)
		}
		if result.SyscallCount < result.SchedulerDecisionCount*2 {
			t.Fatalf("missing E1 syscall count: %#v", result)
		}
		if result.DuplicateTokens <= 0 || result.MaterializeMS <= 0 {
			t.Fatalf("missing E1 materialize cost evidence: %#v", result)
		}
	}
	fifo := e1ByPolicy["fifo"]
	tokenCFS := e1ByPolicy["token-cfs"]
	prefix := e1ByPolicy["token-cfs-prefix-affinity"]
	if fifo.MaterializeMS <= tokenCFS.MaterializeMS || tokenCFS.MaterializeMS <= prefix.MaterializeMS {
		t.Fatalf("expected FIFO > token-CFS > prefix-affinity materialize cost: %#v", suite.E1Scheduler)
	}
	if fifo.P95LatencyMS <= tokenCFS.P95LatencyMS || tokenCFS.P95LatencyMS <= prefix.P95LatencyMS {
		t.Fatalf("expected FIFO > token-CFS > prefix-affinity p95 latency: %#v", suite.E1Scheduler)
	}
	if prefix.MaterializeMS > tokenCFS.MaterializeMS*85/100 {
		t.Fatalf("prefix-affinity materialize cost should improve at least 15%% over token-CFS: token=%d prefix=%d all=%#v", tokenCFS.MaterializeMS, prefix.MaterializeMS, suite.E1Scheduler)
	}
	if prefix.SavedMS <= tokenCFS.SavedMS || tokenCFS.SavedMS <= fifo.SavedMS {
		t.Fatalf("expected prefix-affinity to save the most materialize time: %#v", suite.E1Scheduler)
	}
	if prefix.ContextReuseRate < fifo.ContextReuseRate {
		t.Fatalf("prefix-affinity should preserve at least FIFO context reuse: %#v", suite.E1Scheduler)
	}

	if len(suite.E2Fault) != 5 {
		t.Fatalf("E2 faults = %#v", suite.E2Fault)
	}
	requiredFaults := map[string]bool{
		"tool_timeout":          false,
		"agent_crash":           false,
		"kill_capsule":          false,
		"memory_limit_exceeded": false,
		"pids_limit_exceeded":   false,
	}
	for _, result := range suite.E2Fault {
		if result.Experiment != "E2_real_fault_isolation_benchmark" || result.EvidenceMode != "real-runtime" {
			t.Fatalf("bad E2 identity: %#v", result)
		}
		requiredFaults[result.FaultType] = true
		if result.CascadeFailure || result.AffectedAgents > 1 || result.UnaffectedAgents < 5 || result.RecoveryTimeMS <= 0 {
			t.Fatalf("bad E2 isolation evidence: %#v", result)
		}
		if result.FinalStatus != "recovered" || !result.CheckpointUsed {
			t.Fatalf("bad E2 recovery evidence: %#v", result)
		}
		if result.FaultEvidence == nil || result.FaultEvidence["syscall_status"] == nil {
			t.Fatalf("missing E2 fault evidence: %#v", result)
		}
		if result.FaultType == "memory_limit_exceeded" && result.FaultEvidence["resource_limit_enforced"] != true {
			t.Fatalf("memory limit was not enforced: %#v", result)
		}
		if result.FaultType == "memory_limit_exceeded" && result.FaultEvidence["limit_evidence_mode"] != "real-cgroup-v2" {
			t.Fatalf("memory limit evidence was not real cgroup v2: %#v", result)
		}
		if result.FaultType == "pids_limit_exceeded" && result.FaultEvidence["resource_limit_enforced"] != true {
			t.Fatalf("pids limit was not enforced: %#v", result)
		}
		if result.FaultType == "pids_limit_exceeded" && result.FaultEvidence["limit_evidence_mode"] != "real-cgroup-v2" {
			t.Fatalf("pids limit evidence was not real cgroup v2: %#v", result)
		}
	}
	for faultType, seen := range requiredFaults {
		if !seen {
			t.Fatalf("missing required fault %s in %#v", faultType, suite.E2Fault)
		}
	}

	if len(suite.E3Context) != 3 {
		t.Fatalf("E3 modes = %#v", suite.E3Context)
	}
	for _, result := range suite.E3Context {
		if result.Experiment != "E3_context_reuse" || result.EvidenceMode != "real-runtime" {
			t.Fatalf("bad E3 identity: %#v", result)
		}
		if result.BaselineTokens <= 0 || result.ActualMaterializedTokens <= 0 || result.ReuseRate < 0 {
			t.Fatalf("bad E3 metrics: %#v", result)
		}
	}
	if suite.E3Context[2].SummaryPages == 0 || suite.E3Context[2].SavedTokens <= suite.E3Context[1].SavedTokens {
		t.Fatalf("expected CVM summary mode to improve saved tokens: %#v", suite.E3Context)
	}

	if len(suite.E4IPC) != 2 {
		t.Fatalf("E4 modes = %#v", suite.E4IPC)
	}
	if suite.E4IPC[1].AvoidedCopyBytes <= 0 || suite.E4IPC[1].PayloadBytesActual >= suite.E4IPC[1].PayloadBytesBaseline {
		t.Fatalf("bad E4 page-ref evidence: %#v", suite.E4IPC)
	}

	if suite.E5EndToEnd.Experiment != "E5_end_to_end" || suite.E5EndToEnd.EvidenceMode != "real-runtime" {
		t.Fatalf("bad E5 identity: %#v", suite.E5EndToEnd)
	}
	if !suite.E5EndToEnd.FaultRecovered || !suite.E5EndToEnd.FinalSuccess || suite.E5EndToEnd.Syscalls < 8 {
		t.Fatalf("bad E5 evidence: %#v", suite.E5EndToEnd)
	}

	for _, name := range []string{
		"e1-real-scheduler.json",
		"e1-real-scheduler.csv",
		"e2-real-fault.json",
		"e2-real-fault.csv",
		"e3-real-context.json",
		"e3-real-context.csv",
		"e4-real-ipc.json",
		"e4-real-ipc.csv",
		"e5-end-to-end.json",
		"e5-end-to-end.csv",
	} {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing or empty artifact %s info=%#v err=%v", name, info, err)
		}
	}
}
