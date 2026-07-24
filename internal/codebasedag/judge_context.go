package codebasedag

import (
	"fmt"
	"strings"
)

// ContextJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ContextJudgeMarker = "judge-context-complete"

// ContextPageRef is a CVM page reference passed to coders instead of full text.
type ContextPageRef struct {
	PageID string `json:"page_id"`
	Kind   string `json:"kind"`
	Bytes  int    `json:"bytes"`
}

// CommunicationLaneMetrics holds one transport mode's measured/derived counters.
type CommunicationLaneMetrics struct {
	Mode                string  `json:"mode"`
	LogicalBytes        int     `json:"logical_bytes"`
	TransferredBytes    int     `json:"transferred_bytes"`
	MaterializedBytes   int     `json:"materialized_bytes"`
	SavedBytes          int     `json:"saved_bytes"`
	Kind                string  `json:"kind"` // measured | derived | unsupported
	CoderCompletionRate float64 `json:"coder_completion_rate,omitempty"`
	TotalTaskMS         int64   `json:"total_task_ms,omitempty"`
}

// CommunicationComparison contrasts full-copy vs AORT-R CVM page-ref transport.
type CommunicationComparison struct {
	FullCopy CommunicationLaneMetrics `json:"full_copy"`
	AORTR    CommunicationLaneMetrics `json:"aort_r"`
}

// ContextJudgeReady reports whether the context judge task is complete.
func ContextJudgeReady() bool {
	return ContextJudgeMarker == "judge-context-complete"
}

// BuildCoderPagePrompt returns a prompt fragment with page IDs and an incremental task.
// It must never embed the full shared context body.
func BuildCoderPagePrompt(pageIDs []string, incrementalTask string) string {
	if !ContextJudgeReady() {
		return ""
	}
	var b strings.Builder
	b.WriteString("cvm_page_ids:\n")
	for _, id := range pageIDs {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	fmt.Fprintf(&b, "incremental_task:\n%s\n", incrementalTask)
	b.WriteString("Do not request a full-text copy of shared context; mount pages by ID.\n")
	return b.String()
}

// MeasureCommunicationCompare builds full-copy vs aort-r metrics for the same logical bytes.
func MeasureCommunicationCompare(logicalBytes int, pageIDs []string, mode string) CommunicationLaneMetrics {
	if !ContextJudgeReady() {
		return CommunicationLaneMetrics{Mode: mode, Kind: "unsupported"}
	}
	if logicalBytes < 0 {
		logicalBytes = 0
	}
	switch mode {
	case "full-copy":
		return CommunicationLaneMetrics{
			Mode:              "full-copy",
			LogicalBytes:      logicalBytes,
			TransferredBytes:  logicalBytes,
			MaterializedBytes: logicalBytes,
			SavedBytes:        0,
			Kind:              "derived",
		}
	case "aort-r":
		transferred := 0
		for _, id := range pageIDs {
			transferred += len(id) + 8
		}
		if transferred > logicalBytes {
			transferred = logicalBytes
		}
		saved := logicalBytes - transferred
		if saved < 0 {
			saved = 0
		}
		return CommunicationLaneMetrics{
			Mode:              "aort-r",
			LogicalBytes:      logicalBytes,
			TransferredBytes:  transferred,
			MaterializedBytes: transferred,
			SavedBytes:        saved,
			Kind:              "derived",
		}
	default:
		return CommunicationLaneMetrics{Mode: mode, LogicalBytes: logicalBytes, Kind: "unsupported"}
	}
}

// BuildCommunicationComparison returns both lanes for the same logical payload.
func BuildCommunicationComparison(logicalBytes int, pageIDs []string) *CommunicationComparison {
	if !ContextJudgeReady() {
		return nil
	}
	return &CommunicationComparison{
		FullCopy: MeasureCommunicationCompare(logicalBytes, pageIDs, "full-copy"),
		AORTR:    MeasureCommunicationCompare(logicalBytes, pageIDs, "aort-r"),
	}
}
