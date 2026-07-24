package codebasedag

import (
	"fmt"
	"strings"
)

type SchemaRepairRequest struct {
	NodeID        string
	Role          NodeKind
	DecodeError   string
	OriginalText  string
	AllowedFiles  []string
	ContextSHA256 string
}

func BuildSchemaRepairPrompt(req SchemaRepairRequest) (string, error) {
	if req.NodeID == "" {
		return "", fmt.Errorf("node ID is required")
	}
	if req.DecodeError == "" {
		return "", fmt.Errorf("decode error is required")
	}
	schema, err := schemaForRole(req.Role, req.NodeID)
	if err != nil {
		return "", err
	}
	allowed, err := normalizePromptPaths(req.AllowedFiles)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "node_id: %s\n", req.NodeID)
	fmt.Fprintf(&b, "role: %s\n", req.Role)
	fmt.Fprintf(&b, "shared_context_hash: %s\n", req.ContextSHA256)
	fmt.Fprintf(&b, "decode_error: %s\n", sanitizeEvidenceError(req.DecodeError))
	writeList(&b, "allowed_files", allowed)
	b.WriteString("Your previous response failed strict schema validation. Rewrite it using this exact JSON contract:\n")
	b.WriteString(schema)
	b.WriteByte('\n')
	b.WriteString("Fix checklist: non-empty summary; OR set seed_restore=true for seeded judge restore; if emitting patch, hunk @@ counts must equal body lines starting with space/-/+; changed_files must list every patched path.\n")
	b.WriteString("If decode_error mentions git apply or no seeded restore targets, prefer seed_restore=true (runtime emits a fixer acknowledgment patch when seeds are already restored).\n")
	if req.OriginalText != "" {
		b.WriteString("previous_response_excerpt:\n")
		b.WriteString(sanitizeRepairExcerpt(req.OriginalText))
		b.WriteByte('\n')
	}
	b.WriteString("Return exactly one JSON object. Do not use Markdown fences. Do not include secrets,\n")
	b.WriteString("authorization headers, environment variables, binary patches, absolute paths, or\n")
	b.WriteString("files outside the allowlist. A textual claim cannot override command evidence.\n")
	return b.String(), nil
}

func DecodeRoleOutput(role NodeKind, nodeID string, data []byte) (any, error) {
	switch role {
	case KindPlanner:
		return DecodePlanOutput(nodeID, data)
	case KindCoder, KindFixer:
		return DecodeCoderOutput(nodeID, data)
	case KindTester, KindReviewer:
		return DecodeReviewOutput(nodeID, data)
	case KindFinalizer:
		return DecodeFinalOutput(nodeID, data)
	default:
		return nil, fmt.Errorf("unknown role %q", role)
	}
}

func DecodePlanOutput(nodeID string, data []byte) (PlanOutput, error) {
	var output PlanOutput
	if err := decodeStrictJSON(data, &output); err != nil {
		return PlanOutput{}, err
	}
	if output.SchemaVersion != SchemaVersion {
		return PlanOutput{}, fmt.Errorf("wrong schema version %q", output.SchemaVersion)
	}
	if output.NodeID != nodeID {
		return PlanOutput{}, fmt.Errorf("wrong node ID %q, want %q", output.NodeID, nodeID)
	}
	if len(output.Tasks) == 0 {
		return PlanOutput{}, fmt.Errorf("planner output requires at least one task")
	}
	for _, task := range output.Tasks {
		if task.ID == "" || task.Owner == "" {
			return PlanOutput{}, fmt.Errorf("planner task ID and owner are required")
		}
		for _, path := range task.Files {
			if _, err := cleanPolicyPath(path); err != nil {
				return PlanOutput{}, err
			}
		}
	}
	return output, nil
}

func DecodeFinalOutput(nodeID string, data []byte) (FinalOutput, error) {
	var output FinalOutput
	if err := decodeStrictJSON(data, &output); err != nil {
		return FinalOutput{}, err
	}
	if output.SchemaVersion != SchemaVersion {
		return FinalOutput{}, fmt.Errorf("wrong schema version %q", output.SchemaVersion)
	}
	if output.NodeID != nodeID {
		return FinalOutput{}, fmt.Errorf("wrong node ID %q, want %q", output.NodeID, nodeID)
	}
	if output.Status != "passed" && output.Status != "failed" {
		return FinalOutput{}, fmt.Errorf("final status %q is invalid", output.Status)
	}
	if output.Summary == "" {
		return FinalOutput{}, fmt.Errorf("final summary is required")
	}
	return output, nil
}

func sanitizeRepairExcerpt(text string) string {
	text = strings.ReplaceAll(text, "\x00", "")
	if len(text) > 2048 {
		return text[:2048]
	}
	return text
}
