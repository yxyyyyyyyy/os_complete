package codebasedag

// ResourceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.
const ResourceJudgeMarker = "judge-resource-complete"

// CommandAudit records the exact worker command that ran inside a cgroup.
type CommandAudit struct {
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	NodeID     string   `json:"node_id"`
	RunID      string   `json:"run_id,omitempty"`
}

// EnrichProcessResult attaches configured limits and command audit to a process result.
// Live seed mode replaces this body with a no-op until agents restore it.
func EnrichProcessResult(result *ProcessResult, cfg ProcessConfig) {
	if result == nil {
		return
	}
	if !ResourceJudgeReady() {
		return
	}
	result.NodeID = cfg.NodeID
	result.Limits = cfg.Limits
	result.CommandAudit = CommandAudit{
		Command:    cfg.Worker.Command,
		Args:       append([]string(nil), cfg.Worker.Args...),
		WorkingDir: cfg.Worker.Dir,
		NodeID:     cfg.NodeID,
		RunID:      cfg.RunID,
	}
}

// ResourceJudgeReady reports whether the resource judge task is complete.
func ResourceJudgeReady() bool {
	return ResourceJudgeMarker == "judge-resource-complete"
}

// ValidateCommandAudit rejects empty audits when the judge task is complete.
func ValidateCommandAudit(audit CommandAudit) error {
	if !ResourceJudgeReady() {
		return nil
	}
	if audit.Command == "" {
		return errJudge("command audit requires command")
	}
	if audit.NodeID == "" {
		return errJudge("command audit requires node_id")
	}
	return nil
}
