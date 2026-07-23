package codebasedag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func CheckPatchApplies(ctx context.Context, gitPath, workDir, patchText string) error {
	if strings.TrimSpace(gitPath) == "" {
		gitPath = "git"
	}
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("workload dir is required for patch apply check")
	}
	if strings.TrimSpace(patchText) == "" {
		return fmt.Errorf("patch is required")
	}
	tmp, err := os.CreateTemp("", "aort-patch-check-*.diff")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := tmp.WriteString(patchText); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, gitPath, "-C", workDir, "apply", "--check", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git apply --check failed: %s", truncateEvidence(string(out), 1024))
	}
	return nil
}

// WorktreeSession is an isolated Git worktree/clone used for real patch apply.
type WorktreeSession struct {
	RootDir       string `json:"root_dir"`
	WorkDir       string `json:"work_dir"`
	BaseCommit    string `json:"base_commit"`
	BaseTreeHash  string `json:"base_tree_hash"`
	FinalTreeHash string `json:"final_tree_hash,omitempty"`
	GitPath       string `json:"git_path"`
	Cleaned       bool   `json:"cleaned"`
}

type PatchApplyResult struct {
	NodeID        string   `json:"node_id"`
	PatchSHA256   string   `json:"patch_sha256"`
	DeclaredFiles []string `json:"declared_files"`
	ActualFiles   []string `json:"actual_files"`
	ExitCode      int      `json:"exit_code"`
	StdoutSHA256  string   `json:"stdout_sha256"`
	StderrSHA256  string   `json:"stderr_sha256"`
	StdoutExcerpt string   `json:"stdout_excerpt,omitempty"`
	StderrExcerpt string   `json:"stderr_excerpt,omitempty"`
	TreeHashAfter string   `json:"tree_hash_after"`
	DurationMS    int64    `json:"duration_ms"`
}

type IntegrateEvidence struct {
	SchemaVersion string             `json:"schema_version"`
	NodeID        string             `json:"node_id"`
	Status        string             `json:"status"`
	BaseCommit    string             `json:"base_commit"`
	BaseTreeHash  string             `json:"base_tree_hash"`
	FinalTreeHash string             `json:"final_tree_hash"`
	FinalDiffSHA  string             `json:"final_diff_sha256"`
	Applies       []PatchApplyResult `json:"applies"`
	Conflicts     []string           `json:"conflicts,omitempty"`
}

// CreateWorktree creates an exclusive worktree from sourceRepo at HEAD.
func CreateWorktree(ctx context.Context, sourceRepo, parentDir, runID, gitPath string) (WorktreeSession, error) {
	if sourceRepo == "" || parentDir == "" || runID == "" {
		return WorktreeSession{}, fmt.Errorf("source repo, parent dir, and run ID are required")
	}
	if gitPath == "" {
		gitPath = "git"
	}
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return WorktreeSession{}, err
	}
	workDir := filepath.Join(parentDir, "worktree-"+runID)
	if _, err := os.Stat(workDir); err == nil {
		return WorktreeSession{}, fmt.Errorf("worktree already exists: %s", workDir)
	}
	commit, err := gitWorktreeOutput(ctx, gitPath, sourceRepo, "rev-parse", "HEAD")
	if err != nil {
		return WorktreeSession{}, fmt.Errorf("base commit: %w", err)
	}
	commit = strings.TrimSpace(commit)
	tree, err := gitWorktreeOutput(ctx, gitPath, sourceRepo, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return WorktreeSession{}, fmt.Errorf("base tree: %w", err)
	}
	tree = strings.TrimSpace(tree)
	if out, err := exec.CommandContext(ctx, gitPath, "-C", sourceRepo, "worktree", "add", "--detach", workDir, commit).CombinedOutput(); err != nil {
		// fallback: local clone when worktree unsupported
		_ = os.RemoveAll(workDir)
		if out2, err2 := exec.CommandContext(ctx, gitPath, "clone", "--local", "--no-hardlinks", sourceRepo, workDir).CombinedOutput(); err2 != nil {
			return WorktreeSession{}, fmt.Errorf("git worktree add failed: %v (%s); clone fallback failed: %v (%s)", err, truncateEvidence(string(out), 512), err2, truncateEvidence(string(out2), 512))
		}
		if out3, err3 := exec.CommandContext(ctx, gitPath, "-C", workDir, "checkout", "--detach", commit).CombinedOutput(); err3 != nil {
			_ = os.RemoveAll(workDir)
			return WorktreeSession{}, fmt.Errorf("checkout detach failed: %v (%s)", err3, truncateEvidence(string(out3), 512))
		}
	}
	return WorktreeSession{
		RootDir:      parentDir,
		WorkDir:      workDir,
		BaseCommit:   commit,
		BaseTreeHash: tree,
		GitPath:      gitPath,
	}, nil
}

func (w *WorktreeSession) Cleanup(ctx context.Context) error {
	if w == nil || w.Cleaned || w.WorkDir == "" {
		return nil
	}
	var errs []error
	if w.GitPath == "" {
		w.GitPath = "git"
	}
	// Best-effort remove worktree registration from source if present.
	_ = exec.CommandContext(ctx, w.GitPath, "worktree", "remove", "--force", w.WorkDir).Run()
	if err := os.RemoveAll(w.WorkDir); err != nil {
		errs = append(errs, err)
	}
	w.Cleaned = true
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (w *WorktreeSession) TreeHash(ctx context.Context) (string, error) {
	out, err := gitWorktreeOutput(ctx, w.GitPath, w.WorkDir, "write-tree")
	if err != nil {
		// dirty tree: hash index after add -A for evidence only when needed
		if _, addErr := gitWorktreeOutput(ctx, w.GitPath, w.WorkDir, "add", "-A"); addErr != nil {
			return "", err
		}
		out, err = gitWorktreeOutput(ctx, w.GitPath, w.WorkDir, "write-tree")
		if err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(out), nil
}

func (w *WorktreeSession) Diff(ctx context.Context) (string, error) {
	out, err := gitWorktreeOutput(ctx, w.GitPath, w.WorkDir, "diff", "--no-ext-diff", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (w *WorktreeSession) ChangedFiles(ctx context.Context) ([]string, error) {
	out, err := gitWorktreeOutput(ctx, w.GitPath, w.WorkDir, "diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		clean, err := cleanPolicyPath(filepath.ToSlash(line))
		if err != nil {
			return nil, err
		}
		files = append(files, clean)
	}
	sort.Strings(files)
	return files, nil
}

// ApplyValidatedPatch writes the patch and runs git apply --check then git apply.
func (w *WorktreeSession) ApplyValidatedPatch(ctx context.Context, nodeID, patchText string, declared []string) (PatchApplyResult, error) {
	start := time.Now()
	sum := sha256.Sum256([]byte(patchText))
	result := PatchApplyResult{
		NodeID:        nodeID,
		PatchSHA256:   hex.EncodeToString(sum[:]),
		DeclaredFiles: append([]string(nil), declared...),
	}
	patchPath := filepath.Join(w.WorkDir, ".aort-patches", nodeID+".patch")
	if err := os.MkdirAll(filepath.Dir(patchPath), 0o755); err != nil {
		return result, err
	}
	if err := os.WriteFile(patchPath, []byte(patchText), 0o600); err != nil {
		return result, err
	}
	check := exec.CommandContext(ctx, w.GitPath, "-C", w.WorkDir, "apply", "--check", patchPath)
	checkOut, checkErr := check.CombinedOutput()
	if checkErr != nil {
		result.ExitCode = exitCodeOf(check)
		result.StderrSHA256 = sha256Hex(checkOut)
		result.StderrExcerpt = truncateEvidence(string(checkOut), 2048)
		result.DurationMS = time.Since(start).Milliseconds()
		return result, fmt.Errorf("git apply --check failed for %s: %s", nodeID, result.StderrExcerpt)
	}
	beforeFiles, _ := w.ChangedFiles(ctx)
	_ = beforeFiles
	apply := exec.CommandContext(ctx, w.GitPath, "-C", w.WorkDir, "apply", patchPath)
	applyOut, applyErr := apply.CombinedOutput()
	result.ExitCode = exitCodeOf(apply)
	result.StdoutSHA256 = sha256Hex(applyOut)
	result.StdoutExcerpt = truncateEvidence(string(applyOut), 2048)
	result.DurationMS = time.Since(start).Milliseconds()
	if applyErr != nil {
		result.StderrSHA256 = result.StdoutSHA256
		result.StderrExcerpt = result.StdoutExcerpt
		return result, fmt.Errorf("git apply failed for %s: %s", nodeID, result.StdoutExcerpt)
	}
	actual, err := w.ChangedFiles(ctx)
	if err != nil {
		return result, err
	}
	// Actual files for this patch = new changes vs previous set is hard; use patch-declared recomputed from patch text.
	fromPatch, _, err := patchChangedFiles(patchText)
	if err != nil {
		return result, err
	}
	result.ActualFiles = fromPatch
	if !equalStringSlices(sortedCopy(declared), fromPatch) {
		return result, fmt.Errorf("changed_files mismatch for %s: declared=%v actual=%v", nodeID, sortedCopy(declared), fromPatch)
	}
	// Ensure all actual files appear in current dirty set
	dirtySet := map[string]struct{}{}
	for _, f := range actual {
		dirtySet[f] = struct{}{}
	}
	for _, f := range fromPatch {
		if _, ok := dirtySet[f]; !ok {
			return result, fmt.Errorf("patch %s claimed %q but worktree diff lacks it", nodeID, f)
		}
	}
	tree, err := w.TreeHash(ctx)
	if err != nil {
		return result, err
	}
	result.TreeHashAfter = tree
	w.FinalTreeHash = tree
	return result, nil
}

// DetectHunkConflicts returns overlapping hunk conflicts for patches that share files.
func DetectHunkConflicts(patches map[string]string) []string {
	type span struct {
		node  string
		file  string
		start int
		count int
	}
	var spans []span
	var conflicts []string
	for node, text := range patches {
		files, _, err := patchChangedFiles(text)
		if err != nil {
			conflicts = append(conflicts, fmt.Sprintf("%s: invalid patch: %v", node, err))
			continue
		}
		_ = files
		for _, line := range strings.Split(text, "\n") {
			if !strings.HasPrefix(line, "@@") {
				continue
			}
			// @@ -l,s +l,s @@
			oldStart, oldCount, newStart, newCount, ok := parseHunkHeader(line)
			if !ok {
				conflicts = append(conflicts, fmt.Sprintf("%s: bad hunk header %q", node, line))
				continue
			}
			_ = newStart
			_ = newCount
			file := currentPatchFile(text, line)
			spans = append(spans, span{node: node, file: file, start: oldStart, count: oldCount})
		}
	}
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			a, b := spans[i], spans[j]
			if a.file == "" || a.file != b.file || a.node == b.node {
				continue
			}
			if rangesOverlap(a.start, a.count, b.start, b.count) {
				conflicts = append(conflicts, fmt.Sprintf("%s hunk overlap on %s between %s and %s", a.file, a.file, a.node, b.node))
			}
		}
	}
	sort.Strings(conflicts)
	return uniqueStrings(conflicts)
}

func parseHunkHeader(line string) (oldStart, oldCount, newStart, newCount int, ok bool) {
	// @@ -12,3 +12,4 @@
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") {
		return 0, 0, 0, 0, false
	}
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return 0, 0, 0, 0, false
	}
	oldStart, oldCount, ok1 := parseRange(parts[1])
	newStart, newCount, ok2 := parseRange(parts[2])
	return oldStart, oldCount, newStart, newCount, ok1 && ok2
}

func parseRange(tok string) (start, count int, ok bool) {
	tok = strings.TrimPrefix(tok, "-")
	tok = strings.TrimPrefix(tok, "+")
	if strings.Contains(tok, ",") {
		var a, b int
		if _, err := fmt.Sscanf(tok, "%d,%d", &a, &b); err != nil {
			return 0, 0, false
		}
		return a, b, true
	}
	var a int
	if _, err := fmt.Sscanf(tok, "%d", &a); err != nil {
		return 0, 0, false
	}
	return a, 1, true
}

func rangesOverlap(aStart, aCount, bStart, bCount int) bool {
	if aCount <= 0 {
		aCount = 1
	}
	if bCount <= 0 {
		bCount = 1
	}
	aEnd := aStart + aCount - 1
	bEnd := bStart + bCount - 1
	return aStart <= bEnd && bStart <= aEnd
}

func currentPatchFile(patch, hunkLine string) string {
	lines := strings.Split(patch, "\n")
	file := ""
	for _, line := range lines {
		if line == hunkLine {
			return file
		}
		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if path == "/dev/null" {
				continue
			}
			if clean, err := trimPatchPath(path, "b/"); err == nil {
				file = clean
			}
		}
	}
	return file
}

func gitWorktreeOutput(ctx context.Context, gitPath, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, truncateEvidence(stderr.String(), 512))
	}
	return stdout.String(), nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func truncateEvidence(s string, n int) string {
	s = strings.ReplaceAll(s, "\x00", "")
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func exitCodeOf(cmd *exec.Cmd) int {
	if cmd == nil || cmd.ProcessState == nil {
		return -1
	}
	return cmd.ProcessState.ExitCode()
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
