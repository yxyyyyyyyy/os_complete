package pressure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePSILine(t *testing.T) {
	line, err := ParseLine("some avg10=12.50 avg60=3.25 avg300=0.75 total=123456")
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if line.Kind != "some" || line.Avg10 != 12.50 || line.Avg60 != 3.25 || line.Avg300 != 0.75 || line.Total != 123456 {
		t.Fatalf("line = %#v", line)
	}
}

func TestMonitorSamplesPSIAndComputesThrottle(t *testing.T) {
	root := t.TempDir()
	writePSI(t, root, "cpu.pressure", "some avg10=72.00 avg60=20.00 avg300=4.00 total=90\n")
	writePSI(t, root, "memory.pressure", "some avg10=4.00 avg60=1.00 avg300=0.50 total=80\nfull avg10=1.00 avg60=0.10 avg300=0.01 total=3\n")
	writePSI(t, root, "io.pressure", "some avg10=2.00 avg60=1.00 avg300=0.50 total=70\n")
	monitor := NewMonitor(Config{
		Root:                 root,
		CPUAvg10Threshold:    50,
		MemoryAvg10Threshold: 20,
		IOAvg10Threshold:     30,
	})

	status := monitor.Sample()
	if status.Mode != ModePSI || status.Degraded {
		t.Fatalf("status = %#v", status)
	}
	if !status.Throttle || !strings.Contains(status.ThrottleReason, "cpu") {
		t.Fatalf("status = %#v", status)
	}
	if status.CPU.Some.Avg10 != 72 {
		t.Fatalf("cpu = %#v", status.CPU)
	}
}

func TestMonitorReportsDegradedWhenPSIUnavailable(t *testing.T) {
	monitor := NewMonitor(Config{Root: filepath.Join(t.TempDir(), "missing")})
	status := monitor.Sample()
	if status.Mode != ModeDegraded || !status.Degraded {
		t.Fatalf("status = %#v", status)
	}
	if status.Reason == "" {
		t.Fatalf("missing degraded reason: %#v", status)
	}
}

func writePSI(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}
