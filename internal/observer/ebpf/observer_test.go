package ebpf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseTracepointEventLine(t *testing.T) {
	event, err := ParseTracepointEventLine("event=sched_process_exit pid=123 ppid=45 comm=aort-worker")
	if err != nil {
		t.Fatalf("ParseTracepointEventLine: %v", err)
	}
	if event.Event != "sched_process_exit" || event.PID != 123 || event.PPID != 45 || event.Comm != "aort-worker" {
		t.Fatalf("event = %#v", event)
	}
}

func TestParseTracepointEventLineRejectsMissingPID(t *testing.T) {
	if _, err := ParseTracepointEventLine("event=sched_process_exit comm=aort-worker"); err == nil {
		t.Fatalf("expected missing pid error")
	}
}

func TestRunSmokeWritesDegradedEvidenceWhenPlatformCannotAttach(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunSmoke(outDir)
	if err != nil {
		t.Fatalf("RunSmoke: %v", err)
	}
	if result.Observer != "ebpf" {
		t.Fatalf("observer = %q", result.Observer)
	}
	if runtime.GOOS != "linux" {
		if result.EvidenceMode != "degraded" {
			t.Fatalf("non-linux smoke must be degraded: %#v", result)
		}
		if result.ProgramLoaded {
			t.Fatalf("degraded smoke must not mark program_loaded: %#v", result)
		}
		if !strings.Contains(result.FallbackReason, "not linux") {
			t.Fatalf("fallback_reason = %q", result.FallbackReason)
		}
	}
	path := filepath.Join(outDir, "ebpf_smoke.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	var decoded SmokeResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode evidence: %v\n%s", err, raw)
	}
	if decoded.Observer != "ebpf" || decoded.EvidenceMode == "" {
		t.Fatalf("decoded evidence = %#v", decoded)
	}
}
