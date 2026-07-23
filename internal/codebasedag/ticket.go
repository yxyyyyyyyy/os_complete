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
					"internal/review/resource_isolation.go",
					"internal/review/resource_isolation_test.go",
					"internal/experiment/review_scenarios.go",
					"cmd/aortctl/main.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Implement real resource isolation evidence: cgroup v2, overlay workspace checks, scheduler and cleanup evidence.",
			},
			"context-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/review/context_sharing.go",
					"internal/review/context_sharing_test.go",
					"internal/ipc/shm/shm.go",
					"internal/ipc/shm/shm_test.go",
					"cmd/aortctl/main.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Implement real shared-context evidence using memfd/mmap/FD-passing counters and no degraded pass labels.",
			},
			"evidence-coder": {
				Role: KindCoder,
				AllowedFiles: []string{
					"internal/review/review_final.go",
					"internal/review/review_final_test.go",
					"internal/review/metrics.go",
					"internal/review/metrics_test.go",
					"cmd/aortctl/main.go",
				},
				ImmutableFiles: acceptanceScriptNames(),
				PrivateContext: "Implement strict final evidence validation for real modes, call metadata, artifact hashes, and known limits.",
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
