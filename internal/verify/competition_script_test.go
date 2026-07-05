package verify

import (
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
		"go run ./cmd/aortctl experiment e2",
		"go run ./cmd/aortctl demo software-real",
		"go run ./cmd/aortctl demo fault workspace-rmrf",
		"FINAL_EVIDENCE_INDEX.json",
		"FINAL_SUMMARY.md",
		"evidence_mode_summary",
		"AORT-R competition verification completed.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("script missing %q", want)
		}
	}
	cmd := exec.Command("bash", "-n", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, output)
	}
}
