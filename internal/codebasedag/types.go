package codebasedag

import (
	"os"
	"time"
)

const SchemaVersion = "codebase-dag/v1"

type SeedFile struct {
	Path string      `json:"path"`
	Mode os.FileMode `json:"-"`
	Data []byte      `json:"-"`
}

type SourceFile struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Bytes         int64  `json:"bytes"`
	PhysicalLines int    `json:"physical_lines"`
	NonblankLines int    `json:"nonblank_lines"`
}

type SourceManifest struct {
	SchemaVersion  string       `json:"schema_version"`
	GitCommit      string       `json:"git_commit"`
	GitDirty       bool         `json:"git_dirty"`
	TreeHash       string       `json:"tree_hash"`
	PhysicalLines  int          `json:"physical_go_lines"`
	NonblankLines  int          `json:"nonblank_go_lines"`
	TrackedGoFiles int          `json:"tracked_go_files"`
	Files          []SourceFile `json:"files"`
}

type RunRecord struct {
	SchemaVersion string    `json:"schema_version"`
	RunID         string    `json:"run_id"`
	WorkloadDir   string    `json:"workload_dir"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at,omitempty"`
	Status        string    `json:"status"`
}

type NodeKind string

const (
	KindPreflight NodeKind = "preflight"
	KindPlanner   NodeKind = "planner"
	KindCoder     NodeKind = "coder"
	KindIntegrate NodeKind = "integrate"
	KindTester    NodeKind = "tester"
	KindReviewer  NodeKind = "reviewer"
	KindFixer     NodeKind = "fixer"
	KindFinalizer NodeKind = "finalizer"
)

type NodeStatus string

const (
	NodePending   NodeStatus = "pending"
	NodeReady     NodeStatus = "ready"
	NodeRunning   NodeStatus = "running"
	NodeSucceeded NodeStatus = "succeeded"
	NodeFailed    NodeStatus = "failed"
	NodeReplaying NodeStatus = "replaying"
)

type NodeRecord struct {
	SchemaVersion string     `json:"schema_version"`
	NodeID        string     `json:"node_id"`
	Kind          NodeKind   `json:"kind"`
	Status        NodeStatus `json:"status"`
	Dependencies  []string   `json:"dependencies,omitempty"`
	StartedAt     time.Time  `json:"started_at,omitempty"`
	FinishedAt    time.Time  `json:"finished_at,omitempty"`
	OutputSHA256  string     `json:"output_sha256,omitempty"`
	LLMCallID     string     `json:"llm_call_id,omitempty"`
	Error         string     `json:"error,omitempty"`
}

type CallRecord struct {
	SchemaVersion    string    `json:"schema_version"`
	CallID           string    `json:"call_id"`
	NodeID           string    `json:"node_id"`
	Role             string    `json:"role"`
	Provider         string    `json:"provider"`
	RequestedModel   string    `json:"requested_model"`
	ActualModel      string    `json:"actual_model"`
	EvidenceMode     string    `json:"evidence_mode"`
	Fallback         bool      `json:"fallback"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	DurationMS       int64     `json:"duration_ms"`
	OutputSHA256     string    `json:"output_sha256"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
}

type TestRecord struct {
	SchemaVersion string    `json:"schema_version"`
	Name          string    `json:"name"`
	Command       []string  `json:"command"`
	ExitCode      int       `json:"exit_code"`
	StdoutSHA256  string    `json:"stdout_sha256,omitempty"`
	StderrSHA256  string    `json:"stderr_sha256,omitempty"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	FinishedAt    time.Time `json:"finished_at,omitempty"`
}

type EvidenceSummary struct {
	SchemaVersion              string                   `json:"schema_version"`
	RunID                      string                   `json:"run_id"`
	SourceManifest             SourceManifest           `json:"source_manifest"`
	Nodes                      []NodeRecord             `json:"nodes"`
	Calls                      []CallRecord             `json:"calls"`
	Patches                    []PatchRecord            `json:"patches,omitempty"`
	Tests                      []TestRecord             `json:"tests"`
	Acceptance                 []AcceptanceScript       `json:"acceptance_scripts,omitempty"`
	Processes                  []ProcessResult          `json:"processes,omitempty"`
	PageIDs                    []string                 `json:"page_ids,omitempty"`
	CVMMetrics                 *CVMMetrics              `json:"cvm_metrics,omitempty"`
	FaultReport                *FaultReport             `json:"fault_report,omitempty"`
	CommunicationComparison    *CommunicationComparison `json:"communication_comparison,omitempty"`
	BaselineVsAORTR            *BaselineVsAORTR         `json:"baseline_vs_aort_r,omitempty"`
	JudgeMode                  string                   `json:"judge_mode,omitempty"`
	ResourceAgentPhysicalLines int                      `json:"resourceagent_physical_lines"`
	Artifacts                  map[string]string        `json:"artifacts_sha256"`
	AllRequiredPassed          bool                     `json:"all_required_passed"`
	HumanFunctionalEdits       int                      `json:"human_functional_edits"`
	MinPhysicalLines           int                      `json:"min_physical_go_lines,omitempty"`
	MinNonblankLines           int                      `json:"min_nonblank_go_lines,omitempty"`
}
