package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResourceIsolationRejectsUnsafeConfiguration(t *testing.T) {
	_, err := RunResourceIsolation(context.Background(), ResourceIsolationConfig{Mode: "invalid", OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("invalid mode should fail")
	}
	_, err = RunResourceIsolation(context.Background(), ResourceIsolationConfig{Mode: "aort-r", Timeout: time.Nanosecond, OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("expired timeout should preserve a failed run and return an error")
	}
}

func TestResourceIsolationWritesMeasuredThreeModeEvidence(t *testing.T) {
	out := t.TempDir()
	result, err := RunResourceIsolation(context.Background(), ResourceIsolationConfig{
		Mode:    "all",
		Runs:    1,
		Warmup:  0,
		Seed:    7,
		Timeout: 2 * time.Second,
		OutDir:  out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PerRun) != 3 || len(result.Summary) != 3 {
		t.Fatalf("modes = %d/%d", len(result.PerRun), len(result.Summary))
	}
	for _, mode := range []string{"baseline", "isolation-only", "aort-r"} {
		stats, ok := result.Summary[mode]["task_success"]
		if !ok || stats.Count != 1 {
			t.Fatalf("missing task_success for %s: %+v", mode, result.Summary[mode])
		}
	}
	for _, run := range result.PerRun {
		if filepath.IsAbs(run.Artifact) {
			t.Fatalf("raw artifact must be relative: %q", run.Artifact)
		}
		if run.Metrics["cross_agent_contamination"].Kind != MeasurementMeasured {
			t.Fatalf("contamination measurement = %+v", run.Metrics["cross_agent_contamination"])
		}
	}
	if _, err := os.Stat(filepath.Join(out, "raw")); err != nil {
		t.Fatal(err)
	}
}

func TestSafeRemoveWithinRejectsRootAndOutsidePaths(t *testing.T) {
	root := t.TempDir()
	if err := safeRemoveWithin(root, root); err == nil {
		t.Fatal("must not remove generated root itself")
	}
	if err := safeRemoveWithin(root, filepath.Dir(root)); err == nil {
		t.Fatal("must not remove a parent outside generated root")
	}
}

func TestFaultAgentSupportsBoundedFaultTypes(t *testing.T) {
	root := t.TempDir()
	for _, faultType := range []string{"memory_hog", "pids_hog", "cpu_hog", "workspace_rmrf"} {
		dir := filepath.Join(root, faultType)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err, _, pids := executeFaultAgent(context.Background(), "baseline", faultType, dir, root, false); err != nil {
			t.Fatalf("%s: %v", faultType, err)
		} else if pids <= 0 {
			t.Fatalf("%s should report a bounded process count", faultType)
		}
	}
	for _, faultType := range []string{"exit_nonzero", "hang_timeout"} {
		dir := filepath.Join(root, faultType)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		err, _, pids := executeFaultAgent(ctx, "isolation-only", faultType, dir, root, false)
		cancel()
		if err == nil {
			t.Fatalf("%s should return a measured fault error", faultType)
		}
		if pids <= 0 {
			t.Fatalf("%s should report a bounded process count", faultType)
		}
	}
}

func TestResourceIsolationOpenEulerSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping openEuler smoke in short mode")
	}
	if os.Getenv("AORT_REQUIRE_OPENEULER") != "1" {
		t.Skip("set AORT_REQUIRE_OPENEULER=1 on a real openEuler/cgroup2 host to enable")
	}
	out := t.TempDir()
	result, err := RunResourceIsolation(context.Background(), ResourceIsolationConfig{
		Mode: "all", Runs: 2, Warmup: 0, Seed: 42, Timeout: 5 * time.Second, OutDir: out,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range result.PerRun {
		if _, ok := run.Metrics["normal_agent_success_rate"]; !ok {
			t.Fatalf("missing success rate in %s", run.RunID)
		}
		if _, ok := run.Metrics["fault_terminate_ms"]; !ok {
			t.Fatalf("missing fault terminate metric in %s", run.RunID)
		}
	}
}
