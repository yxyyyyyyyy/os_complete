package verify

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompetitionVerifyScriptContract(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "competition_verify.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read competition verify script: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"set -euo pipefail",
		"scripts/check_openeuler_env.sh",
		"go test ./...",
		"scripts/smoke_openeuler.sh",
		"go run ./cmd/aortctl experiment e1",
		"go run ./cmd/aortctl experiment e1-pressure",
		"go run ./cmd/aortctl experiment e2",
		"go run ./cmd/aortctl experiment e2-pressure-fault",
		"go run ./cmd/aortctl demo software-real",
		"go run ./cmd/aortctl workspace probe",
		"go run ./cmd/aortctl demo fault workspace-rmrf",
		"e1_pressure.json",
		"e2_pressure_fault.json",
		"workspace_probe.json",
		"workspace_probe",
		"FINAL_EVIDENCE_INDEX.json",
		"FINAL_SUMMARY.md",
		"evidence_mode_summary",
		"missing_files",
		"live_openeuler_cgroup",
		"[ \"$env_check\" != \"passed\" ]",
		"[ \"$smoke\" != \"passed\" ]",
		"command=",
		"log_file=",
		"status: passed",
		"status: failed",
		"status: degraded",
		"status: missing",
		"AORT-R competition verification completed.",
		"See experiments/results/final/FINAL_EVIDENCE_INDEX.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("script missing %q", want)
		}
	}
	if strings.Contains(text, "AORT-R competition verification completed. See experiments/results/final/FINAL_EVIDENCE_INDEX.json") {
		t.Fatalf("completion message should be two lines, not one compressed line")
	}
	cmd := exec.Command("bash", "-n", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, output)
	}
}

func TestOpenEulerEnvCheckRequiresOpenEulerForRealCgroupEvidence(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "check_openeuler_env.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read env check script: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"is_openeuler",
		`fallback_reasons.append("host is not openEuler")`,
		`flag("is_openeuler") and flag("cgroup_v2")`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("env check script missing %q", want)
		}
	}
}

func TestSmokeOpenEulerSelectsCapsuleWithLiveCounters(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is required to execute smoke validation snippet")
	}

	path := filepath.Join("..", "..", "scripts", "smoke_openeuler.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read smoke script: %v", err)
	}
	script := extractValidateRealCapsulePython(t, string(data))

	tmp := t.TempDir()
	agentsPath := filepath.Join(tmp, "agents.json")
	capsulesPath := filepath.Join(tmp, "capsules.json")
	envPath := filepath.Join(tmp, "agent.env")
	summaryPath := filepath.Join(tmp, "agent_summary.json")
	capsuleRealPath := filepath.Join(tmp, "capsule_real.json")

	writeJSON(t, agentsPath, []map[string]any{
		{
			"agent_id":       "agent-zero",
			"pid":            101,
			"capsule_mode":   "real",
			"cgroup_path":    "/sys/fs/cgroup/aort.slice/agent-zero",
			"memory_current": 0,
			"pids_current":   5,
		},
		{
			"agent_id":       "agent-live",
			"pid":            102,
			"capsule_mode":   "real",
			"cgroup_path":    "/sys/fs/cgroup/aort.slice/agent-live",
			"memory_current": 8192,
			"pids_current":   5,
		},
	})
	writeJSON(t, capsulesPath, []map[string]any{
		{
			"agent_id":       "agent-zero",
			"evidence_mode":  "real-cgroup-v2",
			"real_cgroup_v2": true,
			"capsule_mode":   "real",
			"cgroup_path":    "/sys/fs/cgroup/aort.slice/agent-zero",
			"memory_current": 0,
			"pids_current":   5,
			"cpu_stat":       map[string]any{"usage_usec": 11},
			"events":         map[string]any{"populated": 1},
			"frozen":         false,
		},
		{
			"agent_id":       "agent-live",
			"evidence_mode":  "real-cgroup-v2",
			"real_cgroup_v2": true,
			"capsule_mode":   "real",
			"cgroup_path":    "/sys/fs/cgroup/aort.slice/agent-live",
			"memory_current": 8192,
			"pids_current":   5,
			"cpu_stat":       map[string]any{"usage_usec": 22},
			"events":         map[string]any{"populated": 1},
			"frozen":         false,
		},
	})

	cmd := exec.Command(python, "-c", script, agentsPath, capsulesPath, envPath, summaryPath, capsuleRealPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate_real_capsule failed: %v\n%s", err, output)
	}

	var summary struct {
		AgentID       string `json:"agent_id"`
		MemoryCurrent int64  `json:"memory_current"`
		PidsCurrent   int64  `json:"pids_current"`
	}
	decodeJSON(t, summaryPath, &summary)
	if summary.AgentID != "agent-live" {
		t.Fatalf("selected agent_id = %q, want agent-live", summary.AgentID)
	}
	if summary.MemoryCurrent <= 0 || summary.PidsCurrent <= 0 {
		t.Fatalf("selected capsule counters must be live, got memory=%d pids=%d", summary.MemoryCurrent, summary.PidsCurrent)
	}

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read agent.env: %v", err)
	}
	if !strings.Contains(string(envData), "AGENT_ID=agent-live") {
		t.Fatalf("agent.env did not select live agent:\n%s", envData)
	}
}

func TestSmokeOpenEulerPrettyPrintsCapturedJSON(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "smoke_openeuler.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read smoke script: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"pretty_json_file()",
		"json.dumps(data, indent=2, ensure_ascii=False)",
		"pretty_json_file \"$output\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke script missing pretty-print hook %q", want)
		}
	}
}

func extractValidateRealCapsulePython(t *testing.T, text string) string {
	t.Helper()
	start := strings.Index(text, "validate_real_capsule() {")
	if start < 0 {
		t.Fatal("validate_real_capsule function not found")
	}
	body := text[start:]
	marker := "<<'PY'\n"
	pyStart := strings.Index(body, marker)
	if pyStart < 0 {
		t.Fatal("validate_real_capsule python heredoc not found")
	}
	python := body[pyStart+len(marker):]
	pyEnd := strings.Index(python, "\nPY\n}")
	if pyEnd < 0 {
		t.Fatal("validate_real_capsule python heredoc terminator not found")
	}
	return python[:pyEnd]
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func decodeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
