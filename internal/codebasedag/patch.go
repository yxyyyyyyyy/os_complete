package codebasedag

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	SchemaVersion    string     `json:"schema_version"`
	NodeID           string     `json:"node_id"`
	Summary          string     `json:"summary"`
	Patch            string     `json:"patch"`
	ReplacementValue string     `json:"replacement_value,omitempty"`
	ChangedFiles     []string   `json:"changed_files"`
	Tests            [][]string `json:"tests"`
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
	if strings.TrimSpace(output.Patch) == "" && strings.TrimSpace(output.ReplacementValue) == "" {
		return CoderOutput{}, fmt.Errorf("patch or replacement_value is required")
	}
	if strings.TrimSpace(output.Summary) == "" && strings.TrimSpace(output.ReplacementValue) == "" {
		return CoderOutput{}, fmt.Errorf("summary is required")
	}
	// replacement_value lets MaterializeCoderPatch fill changed_files (and summary).
	if len(output.ChangedFiles) > 0 || strings.TrimSpace(output.ReplacementValue) == "" {
		if err := validateUniquePaths(output.ChangedFiles); err != nil {
			return CoderOutput{}, err
		}
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
	switch output.Verdict {
	case "pass", "fix", "reject":
	default:
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
	if err := validatePatchHunkCompleteness(output.Patch); err != nil {
		return PatchRecord{}, err
	}
	paths, deletions, err := patchChangedFiles(output.Patch)
	if err != nil {
		return PatchRecord{}, err
	}
	if len(paths) == 0 {
		return PatchRecord{}, fmt.Errorf("patch does not change any files")
	}
	// Authoritative changed_files come from the patch body, never the model declaration.
	// Keep a strict failure only when the model omits every patch path (empty attribution).
	if len(output.ChangedFiles) == 0 {
		return PatchRecord{}, fmt.Errorf("changed_files is required")
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

func MaterializeCoderPatch(workDir string, allowedFiles []string, coder *CoderOutput) error {
	if coder == nil {
		return fmt.Errorf("coder output is required")
	}
	if value := strings.TrimSpace(coder.ReplacementValue); value != "" {
		if strings.ContainsAny(value, "\"\n\r") {
			return fmt.Errorf("replacement_value must be a single-line string without quotes")
		}
		target := ""
		switch {
		case len(allowedFiles) == 1:
			target = allowedFiles[0]
		case len(coder.ChangedFiles) == 1:
			for _, allowed := range allowedFiles {
				if allowed == coder.ChangedFiles[0] {
					target = allowed
					break
				}
			}
			if target == "" {
				return fmt.Errorf("changed_files %q is outside allowlist", coder.ChangedFiles[0])
			}
		default:
			for _, allowed := range allowedFiles {
				if strings.Contains(filepath.Base(allowed), "live_") && strings.HasSuffix(allowed, "_hook.go") {
					if target != "" {
						return fmt.Errorf("replacement_value is ambiguous across allowlisted hook files")
					}
					target = allowed
				}
			}
			if target == "" {
				return fmt.Errorf("replacement_value requires one allowlisted hook file")
			}
		}
		patch, constName, err := SynthesizeQuotedConstPatch(workDir, target, value)
		if err != nil {
			return err
		}
		coder.Patch = patch
		coder.ChangedFiles = []string{target}
		if strings.TrimSpace(coder.Summary) == "" {
			coder.Summary = fmt.Sprintf("update %s", constName)
		}
		return nil
	}
	if strings.TrimSpace(coder.Patch) == "" {
		return fmt.Errorf("patch or replacement_value is required")
	}
	return nil
}

func SynthesizeQuotedConstPatch(workDir, relPath, newValue string) (string, string, error) {
	clean, err := cleanPolicyPath(relPath)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(filepath.Join(workDir, clean))
	if err != nil {
		return "", "", err
	}
	content := string(data)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	constRe := regexp.MustCompile(`^(const\s+)([A-Za-z_][A-Za-z0-9_]*)(\s*=\s*")([^"]*)(".*)$`)
	idx := -1
	var constName, oldLine, newLine string
	for i, line := range lines {
		m := constRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if idx != -1 {
			return "", "", fmt.Errorf("file %s has multiple const string assignments", clean)
		}
		idx = i
		constName = m[2]
		if m[4] == newValue {
			return "", "", fmt.Errorf("replacement_value %q equals current value", newValue)
		}
		oldLine = line
		newLine = m[1] + m[2] + m[3] + newValue + m[5]
	}
	if idx < 0 {
		return "", "", fmt.Errorf("no quoted const assignment found in %s", clean)
	}
	var body strings.Builder
	for i, line := range lines {
		if i == idx {
			fmt.Fprintf(&body, "-%s\n", oldLine)
			fmt.Fprintf(&body, "+%s\n", newLine)
			continue
		}
		fmt.Fprintf(&body, " %s\n", line)
	}
	count := len(lines)
	var patch strings.Builder
	fmt.Fprintf(&patch, "diff --git a/%s b/%s\n", clean, clean)
	fmt.Fprintf(&patch, "--- a/%s\n", clean)
	fmt.Fprintf(&patch, "+++ b/%s\n", clean)
	fmt.Fprintf(&patch, "@@ -1,%d +1,%d @@\n", count, count)
	patch.WriteString(body.String())
	return patch.String(), constName, nil
}

func validatePatchHunkCompleteness(patch string) error {
	lines := strings.Split(patch, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		oldCount, newCount, ok := parseHunkCounts(line)
		if !ok {
			return fmt.Errorf("corrupt hunk header %q", line)
		}
		gotOld, gotNew := 0, 0
		j := i + 1
		for ; j < len(lines); j++ {
			body := lines[j]
			if strings.HasPrefix(body, "@@") || strings.HasPrefix(body, "diff --git ") || strings.HasPrefix(body, "--- ") || strings.HasPrefix(body, "+++ ") {
				break
			}
			if body == `\ No newline at end of file` {
				continue
			}
			if body == "" {
				break
			}
			switch body[0] {
			case ' ':
				gotOld++
				gotNew++
			case '-':
				gotOld++
			case '+':
				gotNew++
			case '\\':
				continue
			default:
				return fmt.Errorf("corrupt hunk body line %q under %q", body, line)
			}
		}
		if gotOld != oldCount || gotNew != newCount {
			return fmt.Errorf("incomplete hunk %q: want old=%d new=%d got old=%d new=%d", line, oldCount, newCount, gotOld, gotNew)
		}
		i = j - 1
	}
	return nil
}

func parseHunkCounts(header string) (oldCount, newCount int, ok bool) {
	// @@ -l,s +l,s @@ or @@ -l +l @@
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "@@") {
		return 0, 0, false
	}
	parts := strings.Split(header, "@@")
	if len(parts) < 2 {
		return 0, 0, false
	}
	body := strings.TrimSpace(parts[1])
	fields := strings.Fields(body)
	if len(fields) < 2 {
		return 0, 0, false
	}
	oldCount, ok = parseRangeCount(fields[0], '-')
	if !ok {
		return 0, 0, false
	}
	newCount, ok = parseRangeCount(fields[1], '+')
	return oldCount, newCount, ok
}

func parseRangeCount(field string, sign byte) (int, bool) {
	if len(field) < 2 || field[0] != sign {
		return 0, false
	}
	rest := field[1:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		var start, count int
		if _, err := fmt.Sscanf(rest, "%d,%d", &start, &count); err != nil {
			return 0, false
		}
		return count, true
	}
	var start int
	if _, err := fmt.Sscanf(rest, "%d", &start); err != nil {
		return 0, false
	}
	return 1, true
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
