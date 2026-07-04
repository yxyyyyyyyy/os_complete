package experiment

import "testing"

func TestRunE1SchedulerProducesAllPolicies(t *testing.T) {
	results := RunE1Scheduler(5)
	if len(results) != 3 {
		t.Fatalf("results = %#v", results)
	}
	seen := map[string]bool{}
	for _, result := range results {
		seen[result.Policy] = true
		if result.DecisionCount == 0 || result.JainFairness <= 0 {
			t.Fatalf("result = %#v", result)
		}
	}
	for _, policy := range []string{"fifo", "token-cfs", "token-cfs-prefix-affinity"} {
		if !seen[policy] {
			t.Fatalf("missing policy %s in %#v", policy, results)
		}
	}
}

func TestRunE2FaultIsolationProducesCapsuleComparison(t *testing.T) {
	results := RunE2FaultIsolation(5)
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Mode == results[1].Mode {
		t.Fatalf("expected mode comparison: %#v", results)
	}
	for _, result := range results {
		if result.FaultCount == 0 {
			t.Fatalf("fault_count missing: %#v", result)
		}
		if result.Mode == "per-agent-capsule" && !result.TaskSuccess {
			t.Fatalf("capsule mode should recover task: %#v", result)
		}
	}
}

func TestRunE3ContextSharingShowsSavedTokensAndBytes(t *testing.T) {
	result := RunE3ContextSharing(5)
	if result.TotalPromptTokens <= result.UniquePageTokens {
		t.Fatalf("expected full-copy tokens to exceed unique tokens: %#v", result)
	}
	if result.SavedTokens <= 0 || result.SavedBytes <= 0 {
		t.Fatalf("expected positive savings: %#v", result)
	}
	if result.MaterializeTimeMS <= 0 {
		t.Fatalf("expected materialize timing: %#v", result)
	}
}
