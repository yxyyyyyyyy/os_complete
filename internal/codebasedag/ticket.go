package codebasedag

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type NodePolicy struct {
	Role           NodeKind `json:"role"`
	AllowedFiles   []string `json:"allowed_files"`
	ImmutableFiles []string `json:"immutable_files"`
	PrivateContext string   `json:"private_context"`
}

type Ticket struct {
	ID            string                `json:"id"`
	SharedContext string                `json:"shared_context"`
	NodePolicies  map[string]NodePolicy `json:"node_policies"`
}

func ReviewRemediationTicket() Ticket {
	immutable := acceptanceScriptNames()
	return Ticket{
		ID:            "review-remediation",
		SharedContext: "review-remediation-resource-context-evidence-judge-full",
		NodePolicies: map[string]NodePolicy{
			"planner": {
				Role:           KindPlanner,
				ImmutableFiles: immutable,
				PrivateContext: "Plan resource audit, CVM page-ref context, and unified evidence summary tasks. List allowlisted production files.",
			},
			"resource-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/codebasedag/judge_resource.go",
					"internal/codebasedag/judge_resource_test.go",
					"internal/codebasedag/process.go",
					"internal/codebasedag/resourceagent",
					"internal/resource/sampler.go",
					"internal/review/resource_isolation.go",
				},
				ImmutableFiles: immutable,
				PrivateContext: "You own internal/codebasedag/resourceagent — a real DeepSeek-authored corpus (≥20000 physical Go lines). Live runs append AORT_SEED_BROKEN trailers to a few gen_*.go files and set ResourceJudgeMarker to seed-incomplete. Preferred response: set seed_restore=true with a non-empty summary so the runtime materializes the exact restore patch for your allowlisted files. Alternatively emit the REFERENCE_RESTORE_PATCH from the prompt. Do not invent process.go edits. Broken builds must be fixed via coder/fixer — never fake success with hooks.",
			},
			"context-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/codebasedag/judge_context.go",
					"internal/codebasedag/judge_context_test.go",
					"internal/codebasedag/live_cvm.go",
					"internal/cvm/store.go",
					"internal/review/context_sharing.go",
				},
				ImmutableFiles: immutable,
				PrivateContext: "Restore ContextJudgeMarker to judge-context-complete (prefer seed_restore=true). Ensure BuildCoderPagePrompt emits page IDs only and MeasureCommunicationCompare distinguishes full-copy vs aort-r.",
			},
			"evidence-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/codebasedag/judge_evidence.go",
					"internal/codebasedag/judge_evidence_test.go",
					"internal/codebasedag/types.go",
					"internal/codebasedag/validate.go",
					"internal/codebasedag/report.go",
					"internal/codebasedag/evidence_bundle.go",
				},
				ImmutableFiles: immutable,
				PrivateContext: "Restore EvidenceJudgeMarker to judge-evidence-complete (prefer seed_restore=true). AttachJudgeEvidence must wire CVMMetrics, FaultReport, and CommunicationComparison.",
			},
			"fault-agent": {
				Role:           KindCoder,
				ImmutableFiles: immutable,
				PrivateContext: "Fault-Agent is a machine node; no LLM patch required.",
			},
			"tester": {
				Role:           KindTester,
				ImmutableFiles: immutable,
				PrivateContext: "Review command outputs and report pass/fix. Machine failures must force fix.",
			},
			"reviewer": {
				Role:           KindReviewer,
				ImmutableFiles: immutable,
				PrivateContext: "Perform defect-first review of real multi-file diffs and test evidence. Blocking findings force a fixer.",
			},
			"finalizer": {
				Role:           KindFinalizer,
				ImmutableFiles: immutable,
				PrivateContext: "Summarize resource audit, CVM comparison, fault isolation, and machine gates. Only after review pass.",
			},
		},
	}
}

func LoadTicketFile(path string) (Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Ticket{}, err
	}
	var ticket Ticket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return Ticket{}, err
	}
	if err := ticket.Validate(); err != nil {
		return Ticket{}, err
	}
	return ticket, nil
}

func (t Ticket) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if t.SharedContext == "" {
		return fmt.Errorf("shared context is required")
	}
	if len(t.NodePolicies) == 0 {
		return fmt.Errorf("node policies are required")
	}
	for nodeID, policy := range t.NodePolicies {
		if policy.Role == "" {
			return fmt.Errorf("node %q role is required", nodeID)
		}
		allowed, err := normalizePromptPaths(policy.AllowedFiles)
		if err != nil {
			return err
		}
		immutable, err := normalizePromptPaths(policy.ImmutableFiles)
		if err != nil {
			return err
		}
		policy.AllowedFiles = allowed
		policy.ImmutableFiles = immutable
		t.NodePolicies[nodeID] = policy
	}
	return nil
}

func acceptanceScriptNames() []string {
	names := []string{
		"internal/codebasedag/acceptance/context_real.sh",
		"internal/codebasedag/acceptance/resource_real.sh",
		"internal/codebasedag/acceptance/review_final_strict.sh",
	}
	sort.Strings(names)
	return names
}
