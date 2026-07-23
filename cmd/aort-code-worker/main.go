package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

type WorkerReport struct {
	SchemaVersion string `json:"schema_version"`
	RunID         string `json:"run_id"`
	AgentID       string `json:"agent_id"`
	NodeID        string `json:"node_id"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	OutputSHA256  string `json:"output_sha256"`
	LLMCallID     string `json:"llm_call_id,omitempty"`
	OutputBytes   int    `json:"output_bytes"`
	EvidenceMode  string `json:"evidence_mode"`
}

type workerOptions struct {
	runID        string
	agentID      string
	nodeID       string
	role         string
	replayOutput string
	expectedHash string
	llmCallID    string
	selfStop     bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	if opts.replayOutput == "" {
		return fmt.Errorf("live gateway execution is not wired in this worker skeleton; use --replay-output for deterministic replay")
	}
	data, err := os.ReadFile(opts.replayOutput)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	gotHash := hex.EncodeToString(sum[:])
	if opts.expectedHash != "" && gotHash != opts.expectedHash {
		return fmt.Errorf("replay output hash mismatch: got %s want %s", gotHash, opts.expectedHash)
	}
	report := WorkerReport{
		SchemaVersion: "codebase-dag/v1",
		RunID:         opts.runID,
		AgentID:       opts.agentID,
		NodeID:        opts.nodeID,
		Role:          opts.role,
		Status:        "replayed",
		OutputSHA256:  gotHash,
		LLMCallID:     opts.llmCallID,
		OutputBytes:   len(data),
		EvidenceMode:  "replay-only",
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func parseOptions(args []string, output io.Writer) (workerOptions, error) {
	fs := flag.NewFlagSet("aort-code-worker", flag.ContinueOnError)
	fs.SetOutput(output)
	var opts workerOptions
	fs.StringVar(&opts.runID, "run-id", "", "codebase DAG run ID")
	fs.StringVar(&opts.agentID, "agent-id", "", "worker agent ID")
	fs.StringVar(&opts.nodeID, "node-id", "", "DAG node ID")
	fs.StringVar(&opts.role, "role", "", "node role")
	fs.StringVar(&opts.replayOutput, "replay-output", "", "existing output artifact to replay without an LLM call")
	fs.StringVar(&opts.expectedHash, "expected-hash", "", "expected SHA-256 for replay output")
	fs.StringVar(&opts.llmCallID, "llm-call-id", "", "original LLM call ID for replay attribution")
	fs.BoolVar(&opts.selfStop, "self-stop", false, "reserved for Linux stopped-worker startup")
	if err := fs.Parse(args); err != nil {
		return workerOptions{}, err
	}
	if opts.runID == "" {
		return workerOptions{}, fmt.Errorf("--run-id is required")
	}
	if opts.agentID == "" {
		return workerOptions{}, fmt.Errorf("--agent-id is required")
	}
	if opts.nodeID == "" {
		return workerOptions{}, fmt.Errorf("--node-id is required")
	}
	if opts.role == "" {
		return workerOptions{}, fmt.Errorf("--role is required")
	}
	if opts.expectedHash != "" && len(opts.expectedHash) != 64 {
		return workerOptions{}, fmt.Errorf("--expected-hash must be a SHA-256 hex string")
	}
	return opts, nil
}
