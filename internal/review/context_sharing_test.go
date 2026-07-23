package review

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestContextSharingMatrixSmokeSubset(t *testing.T) {
	result, err := RunContextSharingMatrix(context.Background(), ContextMatrixConfig{
		Modes:        []string{"full-copy", "aort-r"},
		ContextSizes: []int{4096, 65536},
		AgentCounts:  []int{2, 4},
		SharedRatios: []float64{0, 1.0},
		Runs:         1,
		Timeout:      2 * time.Second,
		OutDir:       t.TempDir(),
		Seed:         23,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 2 modes × 2 sizes × 2 agents × 2 ratios × 1 run = 16
	if len(result.PerRun) != 16 {
		t.Fatalf("matrix cells = %d, want 16", len(result.PerRun))
	}
	for _, run := range result.PerRun {
		if run.Labels["context_size"] == "" || run.Labels["agents"] == "" {
			t.Fatalf("missing matrix labels: %#v", run.Labels)
		}
		for _, name := range []string{"logical_context_bytes", "materialized_bytes", "saved_bytes"} {
			if _, ok := run.Metrics[name]; !ok {
				t.Fatalf("missing metric %s in %s", name, run.RunID)
			}
		}
	}
}

func TestContextSharingRejectsInvalidRatioAndMode(t *testing.T) {
	if _, err := RunContextSharing(context.Background(), ContextSharingConfig{Mode: "invalid", OutDir: t.TempDir()}); err == nil {
		t.Fatal("invalid context mode should fail")
	}
	if _, err := RunContextSharing(context.Background(), ContextSharingConfig{Mode: "aort-r", SharedRatio: 1.2, OutDir: t.TempDir()}); err == nil {
		t.Fatal("shared ratio outside [0,1] should fail")
	}
}

func TestContextSharingCoversThreeModesAndFiveRatios(t *testing.T) {
	result, err := RunContextSharing(context.Background(), ContextSharingConfig{
		Mode:        "all",
		Runs:        1,
		Warmup:      0,
		Seed:        11,
		Timeout:     2 * time.Second,
		Agents:      3,
		ContextSize: 256,
		OutDir:      t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PerRun) != 15 {
		t.Fatalf("per-run count = %d, want 15 (3 modes x 5 ratios)", len(result.PerRun))
	}
	seen := map[string]bool{}
	for _, run := range result.PerRun {
		seen[run.Mode+"@"+run.Labels["shared_ratio"]] = true
		for _, name := range []string{"logical_context_bytes", "physical_bytes_written", "bytes_transferred", "materialized_bytes", "saved_bytes", "shared_pages", "private_pages", "page_hit_ratio", "ipc_p50_ms", "ipc_p95_ms", "throughput_agents_per_sec", "fairness", "prefix_affinity_hits"} {
			if _, ok := run.Metrics[name]; !ok {
				t.Fatalf("run %s missing metric %s", run.RunID, name)
			}
		}
		if run.Metrics["saved_bytes"].Kind != MeasurementDerived || run.Metrics["page_hit_ratio"].Kind != MeasurementDerived {
			t.Fatalf("derived labels missing: %+v", run.Metrics)
		}
		if run.Metrics["logical_context_bytes"].Value < run.Metrics["materialized_bytes"].Value {
			t.Fatalf("materialized bytes cannot exceed logical bytes: %+v", run.Metrics)
		}
	}
	if len(seen) != 15 {
		t.Fatalf("variants = %v", seen)
	}
}

func TestContextSharingComputesSavedBytesFromCounters(t *testing.T) {
	result, err := RunContextSharing(context.Background(), ContextSharingConfig{
		Mode:        "aort-r",
		Runs:        1,
		Warmup:      0,
		Seed:        3,
		Agents:      2,
		ContextSize: 512,
		SharedRatio: 0.5,
		Timeout:     2 * time.Second,
		OutDir:      t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	run := result.PerRun[0]
	logical := run.Metrics["logical_context_bytes"].Value
	transferred := run.Metrics["bytes_transferred"].Value
	saved := run.Metrics["saved_bytes"].Value
	if math.Abs(saved-(logical-transferred)) > 0.0001 {
		t.Fatalf("saved bytes must be derived from counters: logical=%v transferred=%v saved=%v", logical, transferred, saved)
	}
}

func TestAORTRatioZeroDoesNotInventSavings(t *testing.T) {
	result, err := RunContextSharing(context.Background(), ContextSharingConfig{
		Mode:           "aort-r",
		Runs:           1,
		Warmup:         0,
		Seed:           5,
		Agents:         3,
		ContextSize:    256,
		SharedRatio:    0,
		SharedRatioSet: true,
		Timeout:        2 * time.Second,
		OutDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	run := result.PerRun[0]
	logical := run.Metrics["logical_context_bytes"].Value
	if got := run.Metrics["bytes_transferred"].Value; got != logical {
		t.Fatalf("0%% shared transferred=%v, want logical=%v", got, logical)
	}
	if got := run.Metrics["materialized_bytes"].Value; got != logical {
		t.Fatalf("0%% shared materialized=%v, want logical=%v", got, logical)
	}
	if got := run.Metrics["saved_bytes"].Value; got != 0 {
		t.Fatalf("0%% shared saved=%v, want 0", got)
	}
}

func TestAORTCountsPrefixAffinityAfterFirstSharedAgent(t *testing.T) {
	result, err := RunContextSharing(context.Background(), ContextSharingConfig{
		Mode:           "aort-r",
		Runs:           1,
		Warmup:         0,
		Seed:           9,
		Agents:         3,
		ContextSize:    512,
		SharedRatio:    0.5,
		SharedRatioSet: true,
		Timeout:        2 * time.Second,
		OutDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PerRun[0].Metrics["prefix_affinity_hits"].Value; got != 2 {
		t.Fatalf("prefix affinity hits=%v, want 2", got)
	}
	if got := result.PerRun[0].Metrics["shared_pages"].Value; got != 1 {
		t.Fatalf("shared pages=%v, want one public page", got)
	}
	if got := result.PerRun[0].Metrics["private_pages"].Value; got != 3 {
		t.Fatalf("private pages=%v, want one per agent", got)
	}
}

func TestContextComparisonUsesFullCopyBaselineAtSameRatio(t *testing.T) {
	result, err := RunContextSharing(context.Background(), ContextSharingConfig{
		Mode:           "all",
		Runs:           1,
		Warmup:         0,
		Seed:           13,
		Agents:         3,
		ContextSize:    512,
		SharedRatio:    0.5,
		SharedRatioSet: true,
		Timeout:        2 * time.Second,
		OutDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range result.Comparison {
		if row.Mode == "aort-r@50" && row.Metric == "bytes_transferred" {
			found = row.ImprovementValid && row.BaselineMean > 0
		}
	}
	if !found {
		t.Fatalf("aort-r@50 comparison did not use full-copy@50 baseline: %+v", result.Comparison)
	}
}
