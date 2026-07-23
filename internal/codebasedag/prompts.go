package codebasedag

import (
	"fmt"
	"sort"
	"strings"
)

type PromptRequest struct {
	NodeID          string
	Role            NodeKind
	Ticket          string
	AllowedFiles    []string
	ImmutableFiles  []string
	SharedContextID string
	PrivateContext  string
	FileContents    map[string]string
}

func BuildPrompt(req PromptRequest) (string, error) {
	if req.NodeID == "" {
		return "", fmt.Errorf("node ID is required")
	}
	if req.Ticket == "" {
		return "", fmt.Errorf("ticket is required")
	}
	schema, err := schemaForRole(req.Role, req.NodeID)
	if err != nil {
		return "", err
	}
	allowed, err := normalizePromptPaths(req.AllowedFiles)
	if err != nil {
		return "", err
	}
	immutable, err := normalizePromptPaths(req.ImmutableFiles)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "node_id: %s\n", req.NodeID)
	fmt.Fprintf(&b, "role: %s\n", req.Role)
	fmt.Fprintf(&b, "ticket: %s\n", req.Ticket)
	fmt.Fprintf(&b, "shared_context_hash: %s\n", req.SharedContextID)
	writeList(&b, "allowed_files", allowed)
	writeList(&b, "immutable_acceptance", immutable)
	if req.PrivateContext != "" {
		fmt.Fprintf(&b, "private_context:\n%s\n", req.PrivateContext)
	}
	if len(req.FileContents) > 0 {
		keys := make([]string, 0, len(req.FileContents))
		for path := range req.FileContents {
			keys = append(keys, path)
		}
		sort.Strings(keys)
		b.WriteString("current_allowed_file_contents:\n")
		for _, path := range keys {
			fmt.Fprintf(&b, "--- BEGIN %s ---\n%s\n--- END %s ---\n", path, truncateEvidence(req.FileContents[path], 4000), path)
		}
		b.WriteString("Unified diffs MUST match these exact current contents. Do not invent types/APIs that are not present.\n")
	}
	fmt.Fprintf(&b, "json_schema:\n%s\n", schema)
	b.WriteString("Return exactly one JSON object. Do not use Markdown fences. Do not include secrets,\n")
	b.WriteString("authorization headers, environment variables, binary patches, absolute paths, or\n")
	b.WriteString("files outside the allowlist. A textual claim cannot override command evidence.\n")
	if req.Role == KindCoder || req.Role == KindFixer {
		b.WriteString("Prefer replacement_value for the Live*Hook string constant. The runtime will synthesize a unified diff from the current file.\n")
		b.WriteString("If you emit patch instead, it must be a JSON string with valid escapes only (\\\\, \\\", \\n, \\t, \\uXXXX).\n")
		b.WriteString("Never invent types or APIs that are not in current_allowed_file_contents.\n")
	}
	return b.String(), nil
}

func schemaForRole(role NodeKind, nodeID string) (string, error) {
	switch role {
	case KindPlanner:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"tasks":[{"id":"","owner":"","dependencies":[],"files":[],"acceptance":[]}],"risks":[],"commands":[["go","test","./..."]]}`, SchemaVersion, nodeID), nil
	case KindCoder, KindFixer:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"summary":"","replacement_value":"hook-v2","changed_files":[],"tests":[["go","test","./internal/..."]]}`, SchemaVersion, nodeID), nil
	case KindTester, KindReviewer:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"verdict":"pass|fix|reject","blocking_findings":[],"non_blocking_findings":[]}`, SchemaVersion, nodeID), nil
	case KindFinalizer:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"status":"passed|failed","summary":"","limitations":[]}`, SchemaVersion, nodeID), nil
	default:
		return "", fmt.Errorf("unknown prompt role %q", role)
	}
}

func normalizePromptPaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		clean, err := cleanPolicyPath(path)
		if err != nil {
			return nil, err
		}
		if clean != path {
			return nil, fmt.Errorf("path %q is not normalized", path)
		}
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func writeList(b *strings.Builder, name string, values []string) {
	fmt.Fprintf(b, "%s:\n", name)
	if len(values) == 0 {
		b.WriteString("- <none>\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
}
