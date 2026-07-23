package codebasedag

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandEvidence records one real subprocess execution.
type CommandEvidence struct {
	SchemaVersion string    `json:"schema_version"`
	Name          string    `json:"name"`
	Command       []string  `json:"command"`
	WorkDir       string    `json:"work_dir"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DurationMS    int64     `json:"duration_ms"`
	ExitCode      int       `json:"exit_code"`
	StdoutSHA256  string    `json:"stdout_sha256"`
	StderrSHA256  string    `json:"stderr_sha256"`
	StdoutExcerpt string    `json:"stdout_excerpt,omitempty"`
	StderrExcerpt string    `json:"stderr_excerpt,omitempty"`
	TimedOut      bool      `json:"timed_out"`
	SignalKilled  bool      `json:"signal_killed"`
}

type TesterConfig struct {
	WorkDir        string
	ChangedGoFiles []string
	Timeout        time.Duration
	SkipRace       bool
	ExtraEnv       []string
}

type TesterReport struct {
	SchemaVersion string            `json:"schema_version"`
	NodeID        string            `json:"node_id"`
	Status        string            `json:"status"`
	Commands      []CommandEvidence `json:"commands"`
	Tests         []TestRecord      `json:"tests"`
}

// RunMachineTester executes gofmt, git diff --check, go test, optional race, and go vet.
func RunMachineTester(ctx context.Context, cfg TesterConfig) (TesterReport, error) {
	if cfg.WorkDir == "" {
		return TesterReport{}, fmt.Errorf("tester work dir is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	report := TesterReport{
		SchemaVersion: SchemaVersion,
		NodeID:        "tester",
		Status:        "failed",
	}
	runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var cmds [][]string
	if len(cfg.ChangedGoFiles) > 0 {
		args := append([]string{"gofmt", "-w"}, cfg.ChangedGoFiles...)
		cmds = append(cmds, args)
	}
	cmds = append(cmds,
		[]string{"git", "diff", "--check"},
		[]string{"go", "test", "./..."},
	)
	if !cfg.SkipRace {
		cmds = append(cmds, []string{"go", "test", "-race", "./internal/codebasedag/..."})
	}
	cmds = append(cmds, []string{"go", "vet", "./..."})

	for i, argv := range cmds {
		ev, err := runCapturedCommand(runCtx, fmt.Sprintf("tester-%d-%s", i+1, argv[0]), cfg.WorkDir, argv, cfg.ExtraEnv)
		report.Commands = append(report.Commands, ev)
		report.Tests = append(report.Tests, TestRecord{
			SchemaVersion: SchemaVersion,
			Name:          ev.Name,
			Command:       ev.Command,
			ExitCode:      ev.ExitCode,
			StdoutSHA256:  ev.StdoutSHA256,
			StderrSHA256:  ev.StderrSHA256,
			StartedAt:     ev.StartedAt,
			FinishedAt:    ev.FinishedAt,
		})
		if err != nil || ev.ExitCode != 0 {
			return report, fmt.Errorf("machine tester command %v failed: exit=%d timed_out=%t stderr=%s", argv, ev.ExitCode, ev.TimedOut, ev.StderrExcerpt)
		}
	}
	report.Status = "passed"
	return report, nil
}

func runCapturedCommand(ctx context.Context, name, dir string, argv, extraEnv []string) (CommandEvidence, error) {
	ev := CommandEvidence{
		SchemaVersion: SchemaVersion,
		Name:          name,
		Command:       append([]string(nil), argv...),
		WorkDir:       dir,
		StartedAt:     time.Now().UTC(),
		ExitCode:      -1,
	}
	if len(argv) == 0 {
		return ev, fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(scrubTesterEnv(os.Environ()), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	ev.FinishedAt = time.Now().UTC()
	ev.DurationMS = ev.FinishedAt.Sub(ev.StartedAt).Milliseconds()
	ev.StdoutSHA256 = sha256Hex(stdout.Bytes())
	ev.StderrSHA256 = sha256Hex(stderr.Bytes())
	ev.StdoutExcerpt = truncateEvidence(stdout.String(), 4096)
	ev.StderrExcerpt = truncateEvidence(stderr.String(), 4096)
	if ctx.Err() == context.DeadlineExceeded {
		ev.TimedOut = true
	}
	if cmd.ProcessState != nil {
		ev.ExitCode = cmd.ProcessState.ExitCode()
		if !cmd.ProcessState.Exited() {
			ev.SignalKilled = true
		}
	}
	if err != nil {
		return ev, err
	}
	return ev, nil
}

func goFilesOnly(paths []string) []string {
	var out []string
	for _, p := range paths {
		if strings.HasSuffix(p, ".go") {
			out = append(out, filepath.ToSlash(p))
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(in []string) {
	for i := 0; i < len(in); i++ {
		for j := i + 1; j < len(in); j++ {
			if in[j] < in[i] {
				in[i], in[j] = in[j], in[i]
			}
		}
	}
}

func scrubTesterEnv(env []string) []string {
	blocked := map[string]struct{}{
		"DEEPSEEK_API_KEY": {},
		"OPENAI_API_KEY": {},
		"AORT_LLM_PROVIDER": {},
		"AORT_LLM_FALLBACK_PROVIDER": {},
		"DEEPSEEK_BASE_URL": {},
		"DEEPSEEK_MODEL": {},
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if _, ok := blocked[key]; ok {
			continue
		}
		out = append(out, kv)
	}
	return out
}
