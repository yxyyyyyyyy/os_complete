package codebasedag

import "testing"

func TestEnrichProcessResultFillsAudit(t *testing.T) {
	if !ResourceJudgeReady() {
		t.Fatal("resource judge marker must be complete in main tree")
	}
	var result ProcessResult
	EnrichProcessResult(&result, ProcessConfig{
		RunID:  "r1",
		NodeID: "resource-coder-worker",
		Worker: WorkerSpec{Command: "/bin/true", Args: []string{"--mode", "sidecar"}, Dir: "/tmp"},
		Limits: DefaultResourceLimits(),
	})
	if result.CommandAudit.Command != "/bin/true" || result.Limits.MemoryMax == "" {
		t.Fatalf("enrich incomplete: %#v", result)
	}
	if err := ValidateCommandAudit(result.CommandAudit); err != nil {
		t.Fatal(err)
	}
}

func TestBuildCoderPagePromptOmitsFullText(t *testing.T) {
	if !ContextJudgeReady() {
		t.Fatal("context judge incomplete")
	}
	out := BuildCoderPagePrompt([]string{"abc", "def"}, "increment")
	if out == "" || containsFullDump(out) {
		t.Fatalf("bad prompt: %q", out)
	}
	cmp := BuildCommunicationComparison(4096, []string{"abc", "def"})
	if cmp == nil || cmp.AORTR.SavedBytes <= 0 || cmp.FullCopy.TransferredBytes != 4096 {
		t.Fatalf("bad comparison: %#v", cmp)
	}
}

func TestAttachJudgeEvidence(t *testing.T) {
	if !EvidenceJudgeReady() {
		t.Fatal("evidence judge incomplete")
	}
	summary := &EvidenceSummary{}
	metrics := &CVMMetrics{TotalPages: 2, SharedPages: 1}
	fault := &FaultReport{FaultType: "nonzero-exit", EvidenceMode: "measured"}
	cmp := BuildCommunicationComparison(1024, []string{"p1"})
	base := BuildBaselineComparison(fault)
	AttachJudgeEvidence(summary, metrics, fault, cmp, base)
	if summary.CVMMetrics == nil || summary.FaultReport == nil || summary.CommunicationComparison == nil || summary.BaselineVsAORTR == nil {
		t.Fatalf("attach failed: %#v", summary)
	}
}

func containsFullDump(s string) bool {
	return len(s) > 0 && (contains(s, "BEGIN SHARED") || contains(s, "full shared context body"))
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
