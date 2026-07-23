package codebasedag

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ReportDocument is the human-readable final report for a codebase-DAG run.
// Machine gates remain authoritative; this document cannot override them.
type ReportDocument struct {
	Title       string
	RunID       string
	GeneratedAt time.Time
	Status      string
	Sections    []ReportSection
	Limitations []string
}

type ReportSection struct {
	Heading string
	Body    string
}

func BuildReport(summary EvidenceSummary, probe OSProbeResult, journalEvents []EvidenceEvent) ReportDocument {
	status := "failed"
	if summary.AllRequiredPassed {
		status = "passed"
	}
	doc := ReportDocument{
		Title:       "AORT-R Codebase DAG Open World Report",
		RunID:       summary.RunID,
		GeneratedAt: time.Now().UTC(),
		Status:      status,
		Limitations: defaultLimitations(),
	}
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Summary",
		Body: strings.Join([]string{
			fmt.Sprintf("run_id: %s", summary.RunID),
			fmt.Sprintf("schema: %s", summary.SchemaVersion),
			fmt.Sprintf("status: %s", status),
			fmt.Sprintf("provider_model: %s/%s", RequiredDeepSeekProvider, RequiredDeepSeekModel),
			fmt.Sprintf("physical_go_lines: %d", summary.SourceManifest.PhysicalLines),
			fmt.Sprintf("nonblank_go_lines: %d", summary.SourceManifest.NonblankLines),
			fmt.Sprintf("tracked_go_files: %d", summary.SourceManifest.TrackedGoFiles),
			fmt.Sprintf("llm_calls: %d", len(summary.Calls)),
			fmt.Sprintf("nodes: %d", len(summary.Nodes)),
			fmt.Sprintf("tests: %d", len(summary.Tests)),
		}, "\n"),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Open World Environment",
		Body:    formatProbe(probe),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "DAG Nodes",
		Body:    formatNodes(summary.Nodes),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "LLM Calls",
		Body:    formatCalls(summary.Calls),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Patches",
		Body:    formatPatches(summary.Patches),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Tests",
		Body:    formatTests(summary.Tests),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Process Journal",
		Body:    formatJournal(journalEvents),
	})
	doc.Sections = append(doc.Sections, ReportSection{
		Heading: "Acceptance Gates",
		Body:    formatAcceptance(summary),
	})
	return doc
}

func (d ReportDocument) Markdown() string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(d.Title)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("- run_id: `%s`\n", d.RunID))
	b.WriteString(fmt.Sprintf("- generated_at_utc: `%s`\n", d.GeneratedAt.UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- status: **%s**\n\n", d.Status))
	for _, section := range d.Sections {
		b.WriteString("## ")
		b.WriteString(section.Heading)
		b.WriteString("\n\n")
		b.WriteString(section.Body)
		b.WriteString("\n\n")
	}
	if len(d.Limitations) > 0 {
		b.WriteString("## Limitations\n\n")
		for _, item := range d.Limitations {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Authority\n\n")
	b.WriteString("This report is explanatory only. Strict validators, acceptance scripts, and create-exclusive artifacts decide pass/fail.\n")
	return b.String()
}

func defaultLimitations() []string {
	return []string{
		"CVM page reuse is not model KV-cache sharing.",
		"memfd/mmap evidence is not an end-to-end zero-copy claim.",
		"cgroup + OverlayFS is not a full container/VM sandbox.",
		"Single-host Huawei openEuler results do not imply distributed guarantees.",
		"Mock, fallback, simulation, degraded, skipped, or MOOC modes cannot satisfy Open World acceptance.",
	}
}

func formatProbe(probe OSProbeResult) string {
	return strings.Join([]string{
		fmt.Sprintf("os: %s %s (%s)", probe.OS.ID, probe.OS.VersionID, probe.OS.PrettyName),
		fmt.Sprintf("cgroup: fs=%s writable=%t nested=%t mode=%s err=%q", probe.Cgroup.FilesystemType, probe.Cgroup.Writable, probe.Cgroup.NestedCreate, probe.Cgroup.EvidenceMode, probe.Cgroup.Error),
		fmt.Sprintf("overlay: available=%t mount=%t mode=%s err=%q", probe.Overlay.Available, probe.Overlay.MountSucceeded, probe.Overlay.EvidenceMode, probe.Overlay.Error),
		fmt.Sprintf("memfd: available=%t mmap=%t fd_pass=%t mode=%s err=%q", probe.Memfd.Available, probe.Memfd.MmapSucceeded, probe.Memfd.FDPassingSucceeded, probe.Memfd.EvidenceMode, probe.Memfd.Error),
	}, "\n")
}

func formatNodes(nodes []NodeRecord) string {
	if len(nodes) == 0 {
		return "no nodes recorded"
	}
	lines := make([]string, 0, len(nodes))
	ordered := append([]NodeRecord(nil), nodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].NodeID < ordered[j].NodeID })
	for _, node := range ordered {
		lines = append(lines, fmt.Sprintf("- %s kind=%s status=%s call=%s sha=%s err=%q",
			node.NodeID, node.Kind, node.Status, node.LLMCallID, shortHash(node.OutputSHA256), node.Error))
	}
	return strings.Join(lines, "\n")
}

func formatCalls(calls []CallRecord) string {
	if len(calls) == 0 {
		return "no llm calls recorded"
	}
	lines := make([]string, 0, len(calls))
	for i, call := range calls {
		lines = append(lines, fmt.Sprintf(
			"- #%d id=%s node=%s role=%s provider=%s requested=%s actual=%s mode=%s fallback=%t tokens=%d/%d/%d duration_ms=%d status=%s",
			i+1, call.CallID, call.NodeID, call.Role, call.Provider, call.RequestedModel, call.ActualModel,
			call.EvidenceMode, call.Fallback, call.PromptTokens, call.CompletionTokens, call.TotalTokens, call.DurationMS, call.Status,
		))
	}
	return strings.Join(lines, "\n")
}

func formatPatches(patches []PatchRecord) string {
	if len(patches) == 0 {
		return "no patches recorded"
	}
	lines := make([]string, 0, len(patches))
	for _, patch := range patches {
		lines = append(lines, fmt.Sprintf("- node=%s call=%s files=%d sha=%s bytes=%d",
			patch.NodeID, patch.SourceCallID, len(patch.ChangedFiles), shortHash(patch.SHA256), patch.Bytes))
	}
	return strings.Join(lines, "\n")
}

func formatTests(tests []TestRecord) string {
	if len(tests) == 0 {
		return "no tests recorded"
	}
	lines := make([]string, 0, len(tests))
	for _, test := range tests {
		cmd := strings.Join(test.Command, " ")
		lines = append(lines, fmt.Sprintf("- %s cmd=%q exit=%d stdout=%s stderr=%s",
			test.Name, cmd, test.ExitCode, shortHash(test.StdoutSHA256), shortHash(test.StderrSHA256)))
	}
	return strings.Join(lines, "\n")
}

func formatJournal(events []EvidenceEvent) string {
	if len(events) == 0 {
		return "no journal events"
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("- %s type=%s node=%s msg=%q",
			event.At.UTC().Format(time.RFC3339), event.Type, event.NodeID, event.Message))
	}
	return strings.Join(lines, "\n")
}

func formatAcceptance(summary EvidenceSummary) string {
	checks := []string{
		fmt.Sprintf("all_required_passed=%t", summary.AllRequiredPassed),
		fmt.Sprintf("line_gate_physical_ok=%t", summary.SourceManifest.PhysicalLines >= DefaultMinPhysicalLines),
		fmt.Sprintf("line_gate_nonblank_ok=%t", summary.SourceManifest.NonblankLines >= DefaultMinNonblankLines),
		fmt.Sprintf("artifact_count=%d", len(summary.Artifacts)),
	}
	if err := ValidateEvidenceSummary(summary); err != nil {
		checks = append(checks, "validator_error="+err.Error())
	} else {
		checks = append(checks, "validator_error=")
	}
	return strings.Join(checks, "\n")
}

func shortHash(v string) string {
	if len(v) <= 12 {
		return v
	}
	return v[:12]
}
