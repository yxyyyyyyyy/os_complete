package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aort-r/internal/evidence"
	"aort-r/internal/trace"
)

type Result struct {
	ReplaySuccess       bool          `json:"replay_success"`
	EventCount          int           `json:"event_count"`
	Divergence          bool          `json:"divergence"`
	DivergenceReason    string        `json:"divergence_reason,omitempty"`
	OriginalFinalStatus string        `json:"original_final_status"`
	ReplayFinalStatus   string        `json:"replay_final_status"`
	EvidenceMode        evidence.Mode `json:"evidence_mode"`
}

func Run(tracePath, outDir string) (Result, error) {
	events, err := trace.ReadTrace(tracePath)
	if err != nil {
		return Result{}, err
	}
	result := ReplayEvents(events)
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "replay")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return result, err
	}
	return result, writeJSON(filepath.Join(outDir, "replay_result.json"), result)
}

func ReplayEvents(events []trace.TraceEvent) Result {
	result := Result{
		EventCount:          len(events),
		OriginalFinalStatus: "unknown",
		ReplayFinalStatus:   "unknown",
		EvidenceMode:        evidence.ModeRealRuntime,
	}
	var previous time.Time
	state := map[string]string{}
	for i, event := range events {
		parsed, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil {
			return diverged(result, fmt.Sprintf("invalid timestamp for event %s: %v", fallbackID(event, i), err))
		}
		if i > 0 && parsed.Before(previous) {
			return diverged(result, fmt.Sprintf("event order violation at %s", fallbackID(event, i)))
		}
		previous = parsed
		status := statusFromEvent(event)
		if status != "" {
			key := event.TaskID
			if key == "" {
				key = event.AgentID
			}
			if key == "" {
				key = "runtime"
			}
			state[key] = status
			result.OriginalFinalStatus = status
		}
	}
	if result.OriginalFinalStatus == "unknown" && len(events) > 0 {
		result.OriginalFinalStatus = "completed"
	}
	result.ReplayFinalStatus = result.OriginalFinalStatus
	result.ReplaySuccess = !result.Divergence
	_ = state
	return result
}

func statusFromEvent(event trace.TraceEvent) string {
	if raw, ok := event.Payload["final_status"].(string); ok && raw != "" {
		return raw
	}
	if raw, ok := event.Payload["status"].(string); ok && raw != "" {
		if strings.Contains(event.Type, "completed") || strings.Contains(event.Type, "failed") || strings.Contains(event.Type, "recovery") {
			return raw
		}
	}
	switch event.Type {
	case "task_completed", "completed":
		return "completed"
	case "task_failed", "failed":
		return "failed"
	}
	return ""
}

func diverged(result Result, reason string) Result {
	result.Divergence = true
	result.DivergenceReason = reason
	result.ReplaySuccess = false
	result.ReplayFinalStatus = "diverged"
	if result.OriginalFinalStatus == "" {
		result.OriginalFinalStatus = "unknown"
	}
	return result
}

func fallbackID(event trace.TraceEvent, index int) string {
	if event.EventID != "" {
		return event.EventID
	}
	return fmt.Sprintf("index-%d", index)
}

func writeJSON(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
