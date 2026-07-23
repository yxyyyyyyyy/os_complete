package codebasedag

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type PlanTask struct {
	ID           string   `json:"id"`
	Owner        string   `json:"owner"`
	Dependencies []string `json:"dependencies"`
	Files        []string `json:"files"`
	Acceptance   []string `json:"acceptance"`
}

type PlanOutput struct {
	SchemaVersion string     `json:"schema_version"`
	NodeID        string     `json:"node_id"`
	Tasks         []PlanTask `json:"tasks"`
	Risks         []string   `json:"risks"`
	Commands      [][]string `json:"commands"`
}

type CoderOutput struct {
	SchemaVersion string     `json:"schema_version"`
	NodeID        string     `json:"node_id"`
	Summary       string     `json:"summary"`
	Patch         string     `json:"patch"`
	ChangedFiles  []string   `json:"changed_files"`
	Tests         [][]string `json:"tests"`
}

type ReviewOutput struct {
	SchemaVersion string   `json:"schema_version"`
	NodeID        string   `json:"node_id"`
	Verdict       string   `json:"verdict"`
	Blocking      []string `json:"blocking_findings"`
	NonBlocking   []string `json:"non_blocking_findings"`
}

type FinalOutput struct {
	SchemaVersion string   `json:"schema_version"`
	NodeID        string   `json:"node_id"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	Limitations   []string `json:"limitations"`
}

type PatchPolicy struct {
	NodeID         string
	AllowedFiles   map[string]struct{}
	ImmutableFiles map[string]string
	MaxBytes       int
}

type PatchRecord struct {
	NodeID       string   `json:"node_id"`
	SHA256       string   `json:"sha256"`
	Bytes        int      `json:"bytes"`
	ChangedFiles []string `json:"changed_files"`
	SourceCallID string   `json:"source_call_id"`
}

func DecodeCoderOutput(nodeID string, data []byte) (CoderOutput, error) {
	var output CoderOutput
	if err := decodeStrictJSON(data, &output); err != nil {
		return CoderOutput{}, err
	}
	if output.SchemaVersion != SchemaVersion {
		return CoderOutput{}, fmt.Errorf("wrong schema version %q", output.SchemaVersion)
	}
	if output.NodeID != nodeID {
		return CoderOutput{}, fmt.Errorf("wrong node ID %q, want %q", output.NodeID, nodeID)
	}
	if strings.TrimSpace(output.Patch) == "" {
		return CoderOutput{}, fmt.Errorf("patch is required")
	}
	if err := validateUniquePaths(output.ChangedFiles); err != nil {
		return CoderOutput{}, err
	}
	return output, nil
}

func DecodeReviewOutput(nodeID string, data []byte) (ReviewOutput, error) {
	var output ReviewOutput
	if err := decodeStrictJSON(data, &output); err != nil {
		return ReviewOutput{}, err
	}
	if output.SchemaVersion != SchemaVersion || output.NodeID != nodeID {
		return ReviewOutput{}, fmt.Errorf("invalid review output identity")
	}
	if output.Verdict != "pass" && output.Verdict != "fix" {
		return ReviewOutput{}, fmt.Errorf("invalid review verdict %q", output.Verdict)
	}
	return output, nil
}

func ValidatePatch(policy PatchPolicy, output CoderOutput, sourceCallID string) (PatchRecord, error) {
	if output.NodeID != policy.NodeID {
		return PatchRecord{}, fmt.Errorf("patch node %q does not match policy %q", output.NodeID, policy.NodeID)
	}
	maxBytes := policy.MaxBytes
	if maxBytes == 0 {
		maxBytes = 256 << 10
	}
	if len(output.Patch) > maxBytes {
		return PatchRecord{}, fmt.Errorf("patch is %d bytes, above max %d", len(output.Patch), maxBytes)
	}
	if strings.Contains(output.Patch, "\r\n") || strings.Contains(output.Patch, "GIT binary patch") {
		return PatchRecord{}, fmt.Errorf("binary or ambiguous patch format is forbidden")
	}
	paths, deletions, err := patchChangedFiles(output.Patch)
	if err != nil {
		return PatchRecord{}, err
	}
	if !reflect.DeepEqual(paths, sortedCopy(output.ChangedFiles)) {
		return PatchRecord{}, fmt.Errorf("declared changed files %v do not match patch files %v", sortedCopy(output.ChangedFiles), paths)
	}
	for _, path := range paths {
		if _, ok := policy.AllowedFiles[path]; !ok {
			return PatchRecord{}, fmt.Errorf("patch file %q outside allowlist", path)
		}
		if _, ok := policy.ImmutableFiles[path]; ok && deletions[path] {
			return PatchRecord{}, fmt.Errorf("immutable file %q cannot be deleted", path)
		}
	}
	sum := sha256.Sum256([]byte(output.Patch))
	return PatchRecord{
		NodeID:       output.NodeID,
		SHA256:       hex.EncodeToString(sum[:]),
		Bytes:        len(output.Patch),
		ChangedFiles: paths,
		SourceCallID: sourceCallID,
	}, nil
}

func decodeStrictJSON(data []byte, out any) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("expected one bare JSON object")
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing data after JSON object")
	}
	return nil
}

func validateUniquePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("changed_files is required")
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		clean, err := cleanPolicyPath(path)
		if err != nil {
			return err
		}
		if clean != path {
			return fmt.Errorf("path %q is not normalized", path)
		}
		if _, ok := seen[path]; ok {
			return fmt.Errorf("duplicate changed file %q", path)
		}
		seen[path] = struct{}{}
	}
	return nil
}

func patchChangedFiles(patch string) ([]string, map[string]bool, error) {
	lines := strings.Split(patch, "\n")
	files := make(map[string]struct{})
	deletions := make(map[string]bool)
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "--- ") {
			continue
		}
		if i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], "+++ ") {
			return nil, nil, fmt.Errorf("patch has unpaired file headers")
		}
		oldPath := strings.TrimSpace(strings.TrimPrefix(lines[i], "--- "))
		newPath := strings.TrimSpace(strings.TrimPrefix(lines[i+1], "+++ "))
		path, deleted, err := normalizePatchPair(oldPath, newPath)
		if err != nil {
			return nil, nil, err
		}
		files[path] = struct{}{}
		deletions[path] = deleted
	}
	if len(files) == 0 || !strings.Contains(patch, "@@") {
		return nil, nil, fmt.Errorf("patch must contain file headers and hunks")
	}
	out := make([]string, 0, len(files))
	for path := range files {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, deletions, nil
}

func normalizePatchPair(oldPath, newPath string) (string, bool, error) {
	if newPath == "/dev/null" {
		path, err := trimPatchPath(oldPath, "a/")
		return path, true, err
	}
	path, err := trimPatchPath(newPath, "b/")
	if err != nil {
		return "", false, err
	}
	if oldPath != "/dev/null" {
		oldClean, err := trimPatchPath(oldPath, "a/")
		if err != nil {
			return "", false, err
		}
		if oldClean != path {
			return "", false, fmt.Errorf("renames are not allowed: %q -> %q", oldClean, path)
		}
	}
	return path, false, nil
}

func trimPatchPath(path, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("patch path %q missing %q prefix", path, prefix)
	}
	return cleanPolicyPath(strings.TrimPrefix(path, prefix))
}

func cleanPolicyPath(path string) (string, error) {
	if path == "" || strings.Contains(path, `\`) || filepath.IsAbs(path) {
		return "", fmt.Errorf("invalid path %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid path %q", path)
	}
	return clean, nil
}

func sortedCopy(paths []string) []string {
	out := append([]string(nil), paths...)
	sort.Strings(out)
	return out
}
