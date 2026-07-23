package codebasedag

import (
	"fmt"
	"strings"
)

// PromptCatalog stores the immutable role prompt templates used by the DAG.
// Templates are intentionally verbose so model instructions remain explicit and auditable.
type PromptCatalog struct {
	Version string
	Roles   map[string]PromptTemplate
}

type PromptTemplate struct {
	Role            string
	System          string
	UserSkeleton    string
	RequiredKeys    []string
	ForbiddenPhrases []string
	MaxTokens       int
}

func DefaultPromptCatalog() PromptCatalog {
	return PromptCatalog{
		Version: SchemaVersion,
		Roles: map[string]PromptTemplate{
			"planner": {
				Role: "planner",
				System: strings.Join([]string{
					"You are the planner node of AORT-R codebase-dag.",
					"Return schema-valid JSON only.",
					"Decompose the review-remediation ticket into ownership for resource-coder, context-coder, and evidence-coder.",
					"Never request mock, fallback, MOOC, or degraded evidence modes.",
					"Never ask to edit acceptance scripts or process evidence archives.",
				}, " "),
				UserSkeleton: "TICKET\n{{ticket}}\n\nMANIFEST\n{{manifest}}\n\nCONTRACT\n{{contract}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "tasks", "risks", "verification"},
				ForbiddenPhrases: []string{"use mock", "mooc mode", "skip real cgroup"},
				MaxTokens: 4096,
			},
			"resource-coder": {
				Role: "resource-coder",
				System: strings.Join([]string{
					"You are resource-coder.",
					"Produce a unified diff that connects resource-isolation to real worker/cgroup/sampler/scheduler/overlay/checkpoint paths.",
					"Stay inside your file ownership allowlist.",
					"Do not invent fake /sys/fs/cgroup paths.",
				}, " "),
				UserSkeleton: "SCOPE\n{{scope}}\n\nFILES\n{{files}}\n\nFINDINGS\n{{findings}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "patch", "changed_files", "explanation"},
				ForbiddenPhrases: []string{"degraded-ok", "simulate cgroup"},
				MaxTokens: 8192,
			},
			"context-coder": {
				Role: "context-coder",
				System: strings.Join([]string{
					"You are context-coder.",
					"Produce a unified diff that executes real memfd/mmap and FD-passing transport with measured counters.",
					"Do not claim KV-cache sharing or end-to-end zero-copy.",
				}, " "),
				UserSkeleton: "SCOPE\n{{scope}}\n\nFILES\n{{files}}\n\nFINDINGS\n{{findings}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "patch", "changed_files", "explanation"},
				ForbiddenPhrases: []string{"kv cache sharing", "zero-copy guaranteed"},
				MaxTokens: 8192,
			},
			"evidence-coder": {
				Role: "evidence-coder",
				System: strings.Join([]string{
					"You are evidence-coder.",
					"Produce a unified diff that makes review-final reject incomplete, fabricated, overwritten, wrong-mode, or insufficient evidence.",
					"Preserve create-exclusive artifact semantics.",
				}, " "),
				UserSkeleton: "SCOPE\n{{scope}}\n\nFILES\n{{files}}\n\nFINDINGS\n{{findings}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "patch", "changed_files", "explanation"},
				ForbiddenPhrases: []string{"overwrite run", "accept degraded as real"},
				MaxTokens: 8192,
			},
			"tester": {
				Role: "tester",
				System: strings.Join([]string{
					"You are tester.",
					"Assess command results and request missing tests only when justified.",
					"Tool execution remains under Gateway bounds.",
				}, " "),
				UserSkeleton: "DIFF\n{{diff}}\n\nTEST_RESULTS\n{{tests}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "status", "missing_tests", "notes"},
				ForbiddenPhrases: []string{"ignore failing tests"},
				MaxTokens: 4096,
			},
			"reviewer": {
				Role: "reviewer",
				System: strings.Join([]string{
					"You are reviewer.",
					"Check correctness, safety boundaries, evidence semantics, and whether real OS paths are connected.",
					"Blocking findings must force fixer.",
				}, " "),
				UserSkeleton: "DIFF\n{{diff}}\n\nEVIDENCE\n{{evidence}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "blocking_findings", "status"},
				ForbiddenPhrases: []string{"looks fine without evidence"},
				MaxTokens: 4096,
			},
			"fixer": {
				Role: "fixer",
				System: strings.Join([]string{
					"You are fixer.",
					"Repair only the blocking findings with a unified diff.",
					"At most two fixer attempts exist in the call budget.",
				}, " "),
				UserSkeleton: "FINDINGS\n{{findings}}\n\nDIFF\n{{diff}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "patch", "changed_files", "explanation"},
				ForbiddenPhrases: []string{"rewrite entire repository"},
				MaxTokens: 8192,
			},
			"finalizer": {
				Role: "finalizer",
				System: strings.Join([]string{
					"You are finalizer.",
					"Summarize accepted patch, tests, usage, and limitations.",
					"You cannot override a failed machine gate.",
				}, " "),
				UserSkeleton: "SUMMARY\n{{summary}}\n\nGATES\n{{gates}}\n",
				RequiredKeys: []string{"schema_version", "node_id", "status", "limitations", "usage"},
				ForbiddenPhrases: []string{"mark failed gates as passed"},
				MaxTokens: 4096,
			},
		},
	}
}

func (c PromptCatalog) Validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("prompt catalog version %q invalid", c.Version)
	}
	required := []string{"planner", "resource-coder", "context-coder", "evidence-coder", "tester", "reviewer", "fixer", "finalizer"}
	for _, role := range required {
		tpl, ok := c.Roles[role]
		if !ok {
			return fmt.Errorf("missing prompt template for %q", role)
		}
		if err := tpl.Validate(); err != nil {
			return fmt.Errorf("%s: %w", role, err)
		}
	}
	return nil
}

func (t PromptTemplate) Validate() error {
	if t.Role == "" || t.System == "" || t.UserSkeleton == "" {
		return fmt.Errorf("role/system/user skeleton required")
	}
	if len(t.RequiredKeys) == 0 {
		return fmt.Errorf("required keys missing")
	}
	if t.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	return nil
}

func (t PromptTemplate) Render(vars map[string]string) (system, user string, err error) {
	if err := t.Validate(); err != nil {
		return "", "", err
	}
	user = t.UserSkeleton
	for key, value := range vars {
		user = strings.ReplaceAll(user, "{{"+key+"}}", value)
	}
	if strings.Contains(user, "{{") {
		return "", "", fmt.Errorf("unresolved template placeholders remain")
	}
	lower := strings.ToLower(system + "\n" + user)
	for _, bad := range t.ForbiddenPhrases {
		if strings.Contains(lower, strings.ToLower(bad)) {
			return "", "", fmt.Errorf("forbidden phrase %q present in rendered prompt", bad)
		}
	}
	return t.System, user, nil
}

func (c PromptCatalog) RenderRole(role string, vars map[string]string) (system, user string, err error) {
	tpl, ok := c.Roles[role]
	if !ok {
		return "", "", fmt.Errorf("unknown role %q", role)
	}
	return tpl.Render(vars)
}
