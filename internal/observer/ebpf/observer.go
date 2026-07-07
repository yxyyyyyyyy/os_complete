package ebpf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessEvent struct {
	Event string `json:"event"`
	PID   int    `json:"pid"`
	PPID  int    `json:"ppid"`
	Comm  string `json:"comm"`
}

type SmokeResult struct {
	Observer             string   `json:"observer"`
	EvidenceMode         string   `json:"evidence_mode"`
	Linux                bool     `json:"linux"`
	RootOrCapable        bool     `json:"root_or_capable"`
	ProgramLoaded        bool     `json:"program_loaded"`
	AttachedTracepoints  []string `json:"attached_tracepoints"`
	EventsCollected      int      `json:"events_collected"`
	WorkerPIDObserved    bool     `json:"worker_pid_observed"`
	WorkerPID            int      `json:"worker_pid,omitempty"`
	CleanupSuccess       bool     `json:"cleanup_success"`
	FallbackReason       string   `json:"fallback_reason"`
	TraceFSAvailable     bool     `json:"tracefs_available"`
	VerifierLog          string   `json:"verifier_log,omitempty"`
	SupportedDescription string   `json:"supported_description"`
}

func RunSmoke(outDir string) (SmokeResult, error) {
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "ebpf_smoke")
	}
	result := platformSmoke()
	result.Observer = "ebpf"
	result.SupportedDescription = "optional real eBPF observer for process-level runtime events"
	if result.AttachedTracepoints == nil {
		result.AttachedTracepoints = []string{}
	}
	if err := writeJSON(filepath.Join(outDir, "ebpf_smoke.json"), result); err != nil {
		return result, err
	}
	return result, nil
}

func ParseTracepointEventLine(line string) (ProcessEvent, error) {
	fields := strings.Fields(line)
	event := ProcessEvent{}
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "event":
			event.Event = value
		case "pid":
			pid, err := strconv.Atoi(value)
			if err != nil {
				return ProcessEvent{}, fmt.Errorf("invalid pid %q", value)
			}
			event.PID = pid
		case "ppid":
			ppid, err := strconv.Atoi(value)
			if err != nil {
				return ProcessEvent{}, fmt.Errorf("invalid ppid %q", value)
			}
			event.PPID = ppid
		case "comm":
			event.Comm = value
		}
	}
	if event.PID == 0 {
		return ProcessEvent{}, fmt.Errorf("tracepoint event missing pid")
	}
	if event.Event == "" {
		event.Event = "unknown"
	}
	return event, nil
}

func degraded(linux, capable, tracefs bool, reason string) SmokeResult {
	return SmokeResult{
		Observer:            "ebpf",
		EvidenceMode:        "degraded",
		Linux:               linux,
		RootOrCapable:       capable,
		ProgramLoaded:       false,
		AttachedTracepoints: []string{},
		CleanupSuccess:      true,
		FallbackReason:      reason,
		TraceFSAvailable:    tracefs,
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
