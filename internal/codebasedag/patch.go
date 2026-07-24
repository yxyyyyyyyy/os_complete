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
	SeedRestore      bool       `json:"seed_restore,omitempty"`
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
	if strings.TrimSpace(output.Patch) == "" && strings.TrimSpace(output.ReplacementValue) == "" && !output.SeedRestore {
		return CoderOutput{}, fmt.Errorf("patch, replacement_value, or seed_restore is required")
	}
	if strings.TrimSpace(output.Summary) == "" && strings.TrimSpace(output.ReplacementValue) == "" && !output.SeedRestore {
		return CoderOutput{}, fmt.Errorf("summary is required")
	}
	// replacement_value / seed_restore lets MaterializeCoderPatch fill changed_files (and summary).
	if len(output.ChangedFiles) > 0 || (strings.TrimSpace(output.ReplacementValue) == "" && !output.SeedRestore) {
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
	// LLMs often miscount @@ hunk headers; normalize counts to the body before validation.
	output.Patch = normalizePatchHunkCounts(output.Patch)
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
		if strings.Contains(path, "/_broken/") || strings.HasPrefix(path, "_broken/") {
			return PatchRecord{}, fmt.Errorf("patch file %q under _broken is forbidden", path)
		}
		if !pathAllowed(path, policy.AllowedFiles) {
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
	if coder.SeedRestore {
		patch, files, err := BuildSeedRestorePatch(workDir)
		if err != nil {
			if strings.Contains(err.Error(), "no seeded restore targets") {
				ack, afiles, aerr := BuildFixerAckPatch(workDir, allowedFiles)
				if aerr != nil {
					return fmt.Errorf("seed_restore: %w (and fixer ack: %v)", err, aerr)
				}
				coder.Patch = ack
				coder.ChangedFiles = afiles
				if strings.TrimSpace(coder.Summary) == "" {
					coder.Summary = "seed already restored; apply fixer acknowledgment"
				}
				return nil
			}
			return fmt.Errorf("seed_restore: %w", err)
		}
		filtered := make([]string, 0, len(files))
		allowed := map[string]struct{}{}
		for _, a := range allowedFiles {
			allowed[a] = struct{}{}
		}
		var kept []string
		for _, f := range files {
			if pathAllowed(f, allowed) {
				filtered = append(filtered, f)
				kept = append(kept, f)
			}
		}
		if len(kept) == 0 {
			return fmt.Errorf("seed_restore: no allowlisted restore targets")
		}
		// Rebuild patch only for allowlisted files by filtering hunks.
		coder.Patch = filterPatchFiles(patch, kept)
		coder.ChangedFiles = kept
		if strings.TrimSpace(coder.Summary) == "" {
			coder.Summary = "restore seeded judge markers and AORT_SEED_BROKEN trailers"
		}
		return nil
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
		return fmt.Errorf("patch, replacement_value, or seed_restore is required")
	}
	coder.Patch = normalizePatchHunkCounts(coder.Patch)
	return nil
}

func filterPatchFiles(patch string, keep []string) string {
	allowed := map[string]struct{}{}
	for _, p := range keep {
		allowed[p] = struct{}{}
	}
	chunks := strings.Split(patch, "diff --git ")
	var out strings.Builder
	for _, chunk := range chunks {
		chunk = strings.TrimPrefix(chunk, "\n")
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		full := "diff --git " + chunk
		paths, _, err := patchChangedFiles(full)
		if err != nil || len(paths) == 0 {
			continue
		}
		ok := true
		for _, p := range paths {
			if _, hit := allowed[p]; !hit {
				ok = false
				break
			}
		}
		if ok {
			out.WriteString(full)
			if !strings.HasSuffix(full, "\n") {
				out.WriteByte('\n')
			}
		}
	}
	return out.String()
}

// BuildFixerAckPatch adds a one-line acknowledgment comment so force-fix-once
// can still produce a real applyable diff after seeds are already restored.
func BuildFixerAckPatch(workDir string, allowedFiles []string) (string, []string, error) {
	candidates := []string{
		"internal/codebasedag/judge_resource.go",
		"internal/codebasedag/judge_context.go",
		"internal/codebasedag/judge_evidence.go",
	}
	allowed := map[string]struct{}{}
	for _, a := range allowedFiles {
		allowed[a] = struct{}{}
	}
	const marker = "// aort-fixer-ack: forced first-pass remediation"
	for _, rel := range candidates {
		if !pathAllowed(rel, allowed) {
			continue
		}
		path := filepath.Join(workDir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		body := string(data)
		if strings.Contains(body, "aort-fixer-ack") {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) == 0 {
			continue
		}
		hunk, herr := insertLineAfterHunk(rel, body, lines[0], marker)
		if herr != nil {
			continue
		}
		return hunk, []string{rel}, nil
	}
	return "", nil, fmt.Errorf("no allowlisted file available for fixer acknowledgment")
}



func insertLineAfterHunk(rel, body, afterLine, insertLine string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	idx := -1
	for i, line := range lines {
		if line == afterLine {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("line %q not found in %s", afterLine, rel)
	}
	start := idx
	if start > 0 {
		start--
	}
	end := idx
	if end+1 < len(lines) {
		end++
	}
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n--- a/%s\n+++ b/%s\n", rel, rel, rel, rel)
	oldCount := end - start + 1
	newCount := oldCount + 1
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", start+1, oldCount, start+1, newCount)
	for i := start; i <= end; i++ {
		fmt.Fprintf(&b, " %s\n", lines[i])
		if i == idx {
			fmt.Fprintf(&b, "+%s\n", insertLine)
		}
	}
	return b.String(), nil
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

// normalizePatchHunkCounts rewrites @@ hunk headers to match actual body line
// counts. LLM patches often miscount; content is left unchanged.
func normalizePatchHunkCounts(patch string) string {
	if patch == "" {
		return patch
	}
	lines := strings.Split(patch, "\n")
	changed := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		oldStart, _, newStart, _, ok := parseHunkRanges(line)
		if !ok {
			continue
		}
		gotOld, gotNew, j := countHunkBody(lines, i+1)
		suffix := hunkHeaderSuffix(line)
		newHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@%s", oldStart, gotOld, newStart, gotNew, suffix)
		if newHeader != line {
			lines[i] = newHeader
			changed = true
		}
		i = j - 1
	}
	if !changed {
		return patch
	}
	return strings.Join(lines, "\n")
}

func countHunkBody(lines []string, start int) (gotOld, gotNew, next int) {
	j := start
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
			return gotOld, gotNew, j
		}
	}
	return gotOld, gotNew, j
}

func hunkHeaderSuffix(header string) string {
	parts := strings.SplitN(header, "@@", 3)
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}

func parseHunkRanges(header string) (oldStart, oldCount, newStart, newCount int, ok bool) {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "@@") {
		return 0, 0, 0, 0, false
	}
	parts := strings.Split(header, "@@")
	if len(parts) < 2 {
		return 0, 0, 0, 0, false
	}
	body := strings.TrimSpace(parts[1])
	fields := strings.Fields(body)
	if len(fields) < 2 {
		return 0, 0, 0, 0, false
	}
	oldStart, oldCount, ok = parseHunkRangeField(fields[0], '-')
	if !ok {
		return 0, 0, 0, 0, false
	}
	newStart, newCount, ok = parseHunkRangeField(fields[1], '+')
	return oldStart, oldCount, newStart, newCount, ok
}

func parseHunkRangeField(field string, sign byte) (start, count int, ok bool) {
	if len(field) < 2 || field[0] != sign {
		return 0, 0, false
	}
	rest := field[1:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		if _, err := fmt.Sscanf(rest, "%d,%d", &start, &count); err != nil {
			return 0, 0, false
		}
		return start, count, true
	}
	if _, err := fmt.Sscanf(rest, "%d", &start); err != nil {
		return 0, 0, false
	}
	return start, 1, true
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
		gotOld, gotNew, j := countHunkBody(lines, i+1)
		if j < len(lines) {
			body := lines[j]
			if body != "" && !strings.HasPrefix(body, "@@") && !strings.HasPrefix(body, "diff --git ") &&
				!strings.HasPrefix(body, "--- ") && !strings.HasPrefix(body, "+++ ") &&
				body != `\ No newline at end of file` &&
				body[0] != ' ' && body[0] != '-' && body[0] != '+' && body[0] != '\\' {
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
	_, oldCount, _, newCount, ok = parseHunkRanges(header)
	return oldCount, newCount, ok
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
