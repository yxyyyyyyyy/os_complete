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
	fmt.Fprintf(&b, "json_schema:\n%s\n", schema)
	b.WriteString("Return exactly one JSON object. Do not use Markdown fences. Do not include secrets,\n")
	b.WriteString("authorization headers, environment variables, binary patches, absolute paths, or\n")
	b.WriteString("files outside the allowlist. A textual claim cannot override command evidence.\n")
	return b.String(), nil
}

func schemaForRole(role NodeKind, nodeID string) (string, error) {
	switch role {
	case KindPlanner:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"tasks":[{"id":"","owner":"","dependencies":[],"files":[],"acceptance":[]}],"risks":[],"commands":[["go","test","./..."]]}`, SchemaVersion, nodeID), nil
	case KindCoder, KindFixer:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"summary":"","patch":"diff --git ...","changed_files":[],"tests":[["go","test","./internal/..."]]}`, SchemaVersion, nodeID), nil
	case KindTester, KindReviewer:
		return fmt.Sprintf(`{"schema_version":%q,"node_id":%q,"verdict":"pass|fix","blocking_findings":[],"non_blocking_findings":[]}`, SchemaVersion, nodeID), nil
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
