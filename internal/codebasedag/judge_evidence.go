package codebasedag

import "aort-r/internal/cvm"

// EvidenceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const EvidenceJudgeMarker = "judge-evidence-complete"

// CVMMetrics is the evidence-facing projection of cvm.Stats.
type CVMMetrics struct {
	EvidenceMode          string `json:"evidence_mode"`
	TotalPages            int    `json:"total_pages"`
	SharedPages           int    `json:"shared_pages"`
	SavedBytes            int64  `json:"saved_bytes"`
	DedupSavedBytes       int64  `json:"dedup_saved_bytes"`
	CompressionSavedBytes int64  `json:"compression_saved_bytes"`
	HotPages              int    `json:"hot_pages"`
	ColdPages             int    `json:"cold_pages"`
}

// FaultReport captures Fault-Agent isolation outcomes.
type FaultReport struct {
	FaultType             string            `json:"fault_type"`
	FaultAgentNode        string            `json:"fault_agent_node"`
	AffectedAgents        int               `json:"affected_agents"`
	SiblingCompletionRate float64           `json:"sibling_completion_rate"`
	SiblingSuccessRate    float64           `json:"sibling_success_rate"`
	SiblingP50LatencyMS   int64             `json:"sibling_p50_latency_ms,omitempty"`
	SiblingP95LatencyMS   int64             `json:"sibling_p95_latency_ms,omitempty"`
	DetectionMS           int64             `json:"detection_ms"`
	TerminateMS           int64             `json:"terminate_ms"`
	CleanupMS             int64             `json:"cleanup_ms"`
	RecoveryMS            int64             `json:"recovery_ms"`
	MemoryEvents          map[string]string `json:"memory_events,omitempty"`
	WorkspacePollution    int               `json:"workspace_pollution"`
	LowerdirUnchanged     bool              `json:"lowerdir_unchanged"`
	EvidenceMode          string            `json:"evidence_mode"`
}

// ModeComparisonRow is one resource-isolation mode outcome.
type ModeComparisonRow struct {
	Mode                  string  `json:"mode"`
	SiblingCompletionRate float64 `json:"sibling_completion_rate"`
	SiblingSuccessRate    float64 `json:"sibling_success_rate"`
	FaultContainmentScope string  `json:"fault_containment_scope"`
	EvidenceMode          string  `json:"evidence_mode"`
}

// BaselineVsAORTR holds baseline / isolation-only / aort-r comparison rows.
type BaselineVsAORTR struct {
	Rows []ModeComparisonRow `json:"rows"`
}

// EvidenceJudgeReady reports whether the evidence judge task is complete.
func EvidenceJudgeReady() bool {
	return EvidenceJudgeMarker == "judge-evidence-complete"
}

// CVMMetricsFromStats projects cvm.Stats into evidence schema.
func CVMMetricsFromStats(stats cvm.Stats) CVMMetrics {
	return CVMMetrics{
		EvidenceMode:          string(stats.EvidenceMode),
		TotalPages:            stats.TotalPages,
		SharedPages:           stats.SharedPages,
		SavedBytes:            stats.SavedBytes,
		DedupSavedBytes:       stats.DedupSavedBytes,
		CompressionSavedBytes: stats.CompressionSavedBytes,
		HotPages:              stats.HotPages,
		ColdPages:             stats.ColdPages,
	}
}

// AttachJudgeEvidence writes judge evidence fields onto the summary when ready.
func AttachJudgeEvidence(summary *EvidenceSummary, metrics *CVMMetrics, fault *FaultReport, cmp *CommunicationComparison, base *BaselineVsAORTR) {
	if summary == nil || !EvidenceJudgeReady() {
		return
	}
	if metrics != nil {
		summary.CVMMetrics = metrics
	}
	if fault != nil {
		summary.FaultReport = fault
	}
	if cmp != nil {
		summary.CommunicationComparison = cmp
	}
	if base != nil {
		summary.BaselineVsAORTR = base
	}
}
