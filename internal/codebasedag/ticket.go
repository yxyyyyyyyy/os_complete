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
	return Ticket{
		ID:            "review-remediation",
		SharedContext: "review-remediation-resource-context-evidence",
		NodePolicies: map[string]NodePolicy{
			"planner": {
				Role:           KindPlanner,
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Plan the three review-remediation changes and identify machine acceptance gates.",
			},
			"resource-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/review/live_resource_hook.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Return replacement_value as a new non-empty string for LiveResourceHook only. Do not invent structs. Prefer replacement_value over a hand-written patch.",
			},
			"context-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/review/live_context_hook.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Return replacement_value as a new non-empty string for LiveContextHook only. Do not invent structs. Prefer replacement_value over a hand-written patch.",
			},
			"evidence-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/review/live_evidence_hook.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Return replacement_value as a new non-empty string for LiveEvidenceHook only. Do not invent structs. Prefer replacement_value over a hand-written patch.",
			},
			"tester": {
				Role:           KindTester,
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Review command outputs and report pass/fix. Machine failures must force fix.",
			},
			"reviewer": {
				Role:           KindReviewer,
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Perform defect-first review. Blocking findings force a fixer.",
			},
			"finalizer": {
				Role:           KindFinalizer,
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Summarize only after all machine gates pass and exact-model calls are validated.",
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
