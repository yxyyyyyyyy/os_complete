package codebasedag

import (
	"fmt"
	"sort"
	"strings"
)

// ReviewRemediationContract describes the Open World engineering ticket that
// the large codebase DAG must complete using real DeepSeek calls only.
type ReviewRemediationContract struct {
	TicketID           string              `json:"ticket_id"`
	Title              string              `json:"title"`
	Objectives         []string            `json:"objectives"`
	CoderScopes        map[string][]string `json:"coder_scopes"`
	ForbiddenPaths     []string            `json:"forbidden_paths"`
	RequiredSmoke      []string            `json:"required_smoke"`
	RequiredEvidence   []string            `json:"required_evidence"`
	AcceptanceFailures []string            `json:"baseline_acceptance_failures"`
}

func DefaultReviewRemediationContract() ReviewRemediationContract {
	return ReviewRemediationContract{
		TicketID: "review-remediation",
		Title:    "Connect real OS resource, context, and evidence paths",
		Objectives: []string{
			"Connect resource-isolation to real worker processes, cgroup capsules, sampling, scheduling, kill/destroy, OverlayFS, and checkpoint/replay.",
			"Make context-sharing execute real memfd/mmap and FD passing with counters derived from instrumented transport operations.",
			"Make review-final reject incomplete, fabricated, overwritten, wrong-mode, or insufficient-run evidence.",
		},
		CoderScopes: map[string][]string{
			"resource-coder": {
				"internal/review/resource_isolation.go",
				"internal/review/resource_isolation_test.go",
				"internal/capsule/",
				"internal/scheduler/",
				"internal/workspace/",
				"internal/checkpoint/",
			},
			"context-coder": {
				"internal/review/context_sharing.go",
				"internal/review/context_sharing_test.go",
				"internal/ipc/shm/",
				"internal/cvm/",
			},
			"evidence-coder": {
				"internal/review/review_final.go",
				"internal/review/review_final_test.go",
				"internal/evidence/",
				"internal/verify/",
			},
		},
		ForbiddenPaths: []string{
			"internal/codebasedag/acceptance/",
			"scripts/competition_verify_real.sh",
			".env",
			".env.local",
			"experiments/results/_process_evidence/",
		},
		RequiredSmoke: []string{
			"scripts/check_openeuler_env.sh",
			"scripts/smoke_openeuler.sh",
			"scripts/smoke_cgroupv2_multi_agent.sh",
			"scripts/smoke_cgroupv2_limits.sh",
			"scripts/smoke_software_real_openeuler.sh",
			"scripts/smoke_deepseek.sh",
		},
		RequiredEvidence: []string{
			"summary.json",
			"report.md",
			"preflight.json",
			"source_manifest.json",
			"dag.json",
			"llm_calls.jsonl",
			"capsules.jsonl",
			"patches/accepted.diff",
			"secret_scan.txt",
		},
		AcceptanceFailures: []string{
			"resource path not connected to real capsules/sampler/scheduler",
			"context path not using measured memfd/mmap transport",
			"review-final accepts wrong-mode or incomplete evidence",
		},
	}
}

func (c ReviewRemediationContract) Validate() error {
	if c.TicketID == "" || c.Title == "" {
		return fmt.Errorf("ticket id and title are required")
	}
	if len(c.Objectives) < 3 {
		return fmt.Errorf("at least 3 objectives required")
	}
	for _, role := range []string{"resource-coder", "context-coder", "evidence-coder"} {
		if len(c.CoderScopes[role]) == 0 {
			return fmt.Errorf("coder scope for %q is required", role)
		}
	}
	if len(c.ForbiddenPaths) == 0 {
		return fmt.Errorf("forbidden paths are required")
	}
	if len(c.RequiredSmoke) == 0 || len(c.RequiredEvidence) == 0 {
		return fmt.Errorf("required smoke and evidence lists are required")
	}
	if len(c.AcceptanceFailures) < 3 {
		return fmt.Errorf("baseline acceptance failures must list the three defects")
	}
	return nil
}

func (c ReviewRemediationContract) AllowPath(nodeID, path string) error {
	clean, err := cleanPolicyPath(path)
	if err != nil {
		return err
	}
	for _, forbidden := range c.ForbiddenPaths {
		if clean == strings.TrimSuffix(forbidden, "/") || strings.HasPrefix(clean, strings.TrimSuffix(forbidden, "/")+"/") {
			return fmt.Errorf("path %q is forbidden by ticket policy", clean)
		}
	}
	scopes := c.CoderScopes[nodeID]
	if len(scopes) == 0 {
		return fmt.Errorf("unknown coder node %q", nodeID)
	}
	for _, scope := range scopes {
		scope = strings.TrimSuffix(scope, "/")
		if clean == scope || strings.HasPrefix(clean, scope+"/") {
			return nil
		}
	}
	return fmt.Errorf("path %q is outside %s scope", clean, nodeID)
}

func (c ReviewRemediationContract) PromptBlock() string {
	var b strings.Builder
	b.WriteString("TICKET ")
	b.WriteString(c.TicketID)
	b.WriteString(": ")
	b.WriteString(c.Title)
	b.WriteString("\n\nOBJECTIVES:\n")
	for i, obj := range c.Objectives {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, obj))
	}
	b.WriteString("\nCODER SCOPES:\n")
	roles := make([]string, 0, len(c.CoderScopes))
	for role := range c.CoderScopes {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		b.WriteString("- ")
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(strings.Join(c.CoderScopes[role], ", "))
		b.WriteString("\n")
	}
	b.WriteString("\nFORBIDDEN:\n")
	for _, path := range c.ForbiddenPaths {
		b.WriteString("- ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	b.WriteString("\nBASELINE ACCEPTANCE MUST FAIL FOR:\n")
	for _, item := range c.AcceptanceFailures {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\nRULES:\n")
	b.WriteString("- provider must be deepseek and model must be deepseek-v4-flash\n")
	b.WriteString("- no mock, fallback, MOOC, simulation, or degraded evidence may pass\n")
	b.WriteString("- return schema-valid JSON only\n")
	return b.String()
}
