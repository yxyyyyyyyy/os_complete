package shm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunSmokeWritesEvidence(t *testing.T) {
	outDir := t.TempDir()
	result, err := RunSmoke(outDir)
	if err != nil {
		t.Fatalf("RunSmoke: %v", err)
	}
	if result.IPCMode == "" || result.EvidenceMode == "" {
		t.Fatalf("result missing mode: %#v", result)
	}
	if runtime.GOOS != "linux" {
		if result.EvidenceMode != "degraded" || result.FallbackReason == "" {
			t.Fatalf("non-linux must degrade with reason: %#v", result)
		}
		if !strings.Contains(result.FallbackReason, "not linux") {
			t.Fatalf("fallback_reason = %q", result.FallbackReason)
		}
	}
	path := filepath.Join(outDir, "ipc_shm_smoke.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	var decoded SmokeResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode evidence: %v\n%s", err, raw)
	}
	if decoded.CleanupSuccess != result.CleanupSuccess {
		t.Fatalf("decoded = %#v result = %#v", decoded, result)
	}
}

func TestMemoryTransportDataIntegrity(t *testing.T) {
	result, err := exerciseMemoryTransport([]byte("shared context page"), 2)
	if runtime.GOOS != "linux" {
		if err == nil {
			t.Fatalf("expected non-linux transport error")
		}
		return
	}
	if err != nil {
		t.Fatalf("exerciseMemoryTransport: %v", err)
	}
	if !result.MemfdCreateSuccess || !result.MmapSuccess || !result.FDPassingSuccess || !result.WorkerMmapSuccess {
		t.Fatalf("transport did not complete: %#v", result)
	}
	if !result.DataIntegrityOK || result.SharedPages != 1 || result.AvoidedCopyBytes == 0 {
		t.Fatalf("unexpected transport result: %#v", result)
	}
}
