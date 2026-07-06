package resource

import (
	"os"
	"path/filepath"
	"testing"

	"aort-r/internal/avp"
	"aort-r/internal/evidence"
)

func TestCgroupSamplerReadsCgroupAndPSIPressure(t *testing.T) {
	root := t.TempDir()
	cgroup := filepath.Join(root, "agent")
	writeFile(t, filepath.Join(cgroup, "memory.current"), "536870912\n")
	writeFile(t, filepath.Join(cgroup, "memory.max"), "1073741824\n")
	writeFile(t, filepath.Join(cgroup, "pids.current"), "16\n")
	writeFile(t, filepath.Join(cgroup, "pids.max"), "64\n")
	writeFile(t, filepath.Join(cgroup, "cpu.stat"), "usage_usec 100\nnr_throttled 25\nthrottled_usec 2500000\n")
	pressureRoot := filepath.Join(root, "pressure")
	writeFile(t, filepath.Join(pressureRoot, "cpu"), "some avg10=12.50 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "memory"), "some avg10=7.00 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "io"), "some avg10=3.00 avg60=0.00 avg300=0.00 total=1\n")

	sampler := NewCgroupSampler(pressureRoot)
	agent := avp.AVP{AgentID: "agent", CgroupPath: cgroup}
	pressure, err := sampler.Sample(agent)
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if pressure.EvidenceMode != evidence.ModeRealCgroupV2 {
		t.Fatalf("evidence mode = %q", pressure.EvidenceMode)
	}
	if pressure.MemoryPressure != 0.5 || pressure.PidsPressure != 0.25 {
		t.Fatalf("memory/pids pressure = %#v", pressure)
	}
	if pressure.CPUThrottlePressure != 0.25 {
		t.Fatalf("cpu throttle pressure = %#v", pressure)
	}
	if pressure.PSIPressure != 0.125 {
		t.Fatalf("psi pressure = %#v", pressure)
	}
}

func TestCgroupSamplerEnrichesAVPCandidate(t *testing.T) {
	root := t.TempDir()
	cgroup := filepath.Join(root, "agent")
	writeFile(t, filepath.Join(cgroup, "memory.current"), "1024\n")
	writeFile(t, filepath.Join(cgroup, "memory.max"), "2048\n")
	writeFile(t, filepath.Join(cgroup, "pids.current"), "3\n")
	writeFile(t, filepath.Join(cgroup, "pids.max"), "9\n")
	writeFile(t, filepath.Join(cgroup, "cpu.stat"), "nr_throttled 5\n")
	pressureRoot := filepath.Join(root, "pressure")
	writeFile(t, filepath.Join(pressureRoot, "cpu"), "some avg10=0.00 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "memory"), "some avg10=0.00 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "io"), "some avg10=0.00 avg60=0.00 avg300=0.00 total=1\n")

	enriched, pressure, err := NewCgroupSampler(pressureRoot).Enrich(avp.AVP{
		AgentID:    "agent",
		CgroupPath: cgroup,
	})
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if enriched.MemoryCurrent != 1024 || enriched.PidsCurrent != 3 {
		t.Fatalf("candidate was not enriched: %#v", enriched)
	}
	if enriched.CPUStat["nr_throttled"] != 5 {
		t.Fatalf("cpu stat not enriched: %#v", enriched.CPUStat)
	}
	if pressure.MemoryPressure != 0.5 || pressure.PidsPressure != 0.333 {
		t.Fatalf("pressure = %#v", pressure)
	}
}

func TestCgroupSamplerReportsDegradedWhenCgroupFilesMissing(t *testing.T) {
	pressure, err := NewCgroupSampler(t.TempDir()).Sample(avp.AVP{AgentID: "agent"})
	if err == nil {
		t.Fatal("expected missing cgroup path error")
	}
	if pressure.EvidenceMode != evidence.ModeDegraded {
		t.Fatalf("expected degraded pressure, got %#v", pressure)
	}
	if pressure.FallbackReason == "" {
		t.Fatalf("fallback reason missing: %#v", pressure)
	}
}

func writeFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
