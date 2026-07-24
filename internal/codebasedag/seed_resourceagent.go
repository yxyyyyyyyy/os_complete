package codebasedag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MinResourceAgentPhysicalLines = 20000
const MaxSeedBrokenResourceAgentFiles = 3

// SeedBrokenResourceAgent injects compile-breaking markers into DeepSeek-authored
// resourceagent sources inside workDir so the live resource-coder must restore them.
func SeedBrokenResourceAgent(workDir string) (int, error) {
	root := filepath.Join(workDir, "internal", "codebasedag", "resourceagent")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return 0, fmt.Errorf("resourceagent package missing under workDir: %w", err)
	}
	broken := 0
	var candidates []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "_broken" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "gen_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, fmt.Errorf("no resourceagent gen_*.go files found to break")
	}
	step := 7
	if len(candidates) < 14 {
		step = 1
	}
	for i, path := range candidates {
		if broken >= MaxSeedBrokenResourceAgentFiles {
			break
		}
		if i%step != 0 {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return broken, err
		}
		text := string(data)
		if strings.Contains(text, "AORT_SEED_BROKEN") {
			continue
		}
		text += "\n// AORT_SEED_BROKEN: intentional compile break for live restore\nvar _ = __aort_seed_broken_undefined__\n"
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return broken, err
		}
		broken++
	}
	if broken == 0 {
		return 0, fmt.Errorf("no resourceagent files were broken for seeding")
	}
	return broken, nil
}

// BuildSeedRestorePatch emits a minimal unified diff that undoes SeedIncompleteJudgeTasks
// and SeedBrokenResourceAgent for the given workDir. Used as a reference for DeepSeek and
// when CoderOutput.SeedRestore is true.
func BuildSeedRestorePatch(workDir string) (patch string, files []string, err error) {
	if workDir == "" {
		return "", nil, fmt.Errorf("workDir is required")
	}
	var parts []string
	markerJobs := []struct {
		rel, from, to string
	}{
		{"internal/codebasedag/judge_resource.go", `const ResourceJudgeMarker = "seed-incomplete"`, `const ResourceJudgeMarker = "judge-resource-complete"`},
		{"internal/codebasedag/judge_context.go", `const ContextJudgeMarker = "seed-incomplete"`, `const ContextJudgeMarker = "judge-context-complete"`},
		{"internal/codebasedag/judge_evidence.go", `const EvidenceJudgeMarker = "seed-incomplete"`, `const EvidenceJudgeMarker = "judge-evidence-complete"`},
	}
	for _, job := range markerJobs {
		path := filepath.Join(workDir, job.rel)
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			continue
		}
		body := string(data)
		if !strings.Contains(body, job.from) {
			continue
		}
		hunk, herr := singleLineReplaceHunk(job.rel, body, job.from, job.to)
		if herr != nil {
			return "", nil, herr
		}
		parts = append(parts, hunk)
		files = append(files, job.rel)
	}
	root := filepath.Join(workDir, "internal", "codebasedag", "resourceagent")
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			if d != nil && d.IsDir() && d.Name() == "_broken" {
				return filepath.SkipDir
			}
			return werr
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		body := string(data)
		if !strings.Contains(body, "AORT_SEED_BROKEN") {
			return nil
		}
		rel, _ := filepath.Rel(workDir, path)
		rel = filepath.ToSlash(rel)
		hunk, herr := removeSeedBrokenTrailerHunk(rel, body)
		if herr != nil {
			err = herr
			return herr
		}
		parts = append(parts, hunk)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("no seeded restore targets found under %s", workDir)
	}
	return strings.Join(parts, ""), files, nil
}

func singleLineReplaceHunk(rel, body, oldLine, newLine string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	idx := -1
	for i, line := range lines {
		if line == oldLine {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("line %q not found in %s", oldLine, rel)
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
	newCount := oldCount
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", start+1, oldCount, start+1, newCount)
	for i := start; i <= end; i++ {
		if i == idx {
			fmt.Fprintf(&b, "-%s\n", oldLine)
			fmt.Fprintf(&b, "+%s\n", newLine)
			continue
		}
		fmt.Fprintf(&b, " %s\n", lines[i])
	}
	return b.String(), nil
}

func removeSeedBrokenTrailerHunk(rel, body string) (string, error) {
	const trailer = "\n// AORT_SEED_BROKEN: intentional compile break for live restore\nvar _ = __aort_seed_broken_undefined__\n"
	if !strings.HasSuffix(body, trailer) && !strings.Contains(body, "AORT_SEED_BROKEN") {
		return "", fmt.Errorf("%s missing seed trailer", rel)
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Drop trailing seed lines (comment + var).
	cut := 0
	for i := len(lines) - 1; i >= 0 && cut < 2; i-- {
		if strings.Contains(lines[i], "AORT_SEED_BROKEN") || strings.Contains(lines[i], "__aort_seed_broken_undefined__") {
			cut++
			continue
		}
		break
	}
	if cut == 0 {
		return "", fmt.Errorf("%s: could not locate seed trailer lines", rel)
	}
	keep := lines[:len(lines)-cut]
	ctx := 2
	if len(keep) < ctx {
		ctx = len(keep)
	}
	start := len(keep) - ctx
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n--- a/%s\n+++ b/%s\n", rel, rel, rel, rel)
	oldCount := ctx + cut
	newCount := ctx
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", start+1, oldCount, start+1, newCount)
	for i := start; i < len(keep); i++ {
		fmt.Fprintf(&b, " %s\n", keep[i])
	}
	for i := len(keep); i < len(lines); i++ {
		fmt.Fprintf(&b, "-%s\n", lines[i])
	}
	return b.String(), nil
}

func hashRel(s string) int {
	h := 0
	for _, c := range s {
		h = (h*31 + int(c)) & 0x7fffffff
	}
	return h
}

// CountResourceAgentPhysicalLines counts Go physical lines owned by resource-coder.
func CountResourceAgentPhysicalLines(workDir string) (int, int, error) {
	root := filepath.Join(workDir, "internal", "codebasedag", "resourceagent")
	phys, files := 0, 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "_broken" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b := string(data)
		phys += strings.Count(b, "\n")
		if len(b) > 0 && !strings.HasSuffix(b, "\n") {
			phys++
		}
		files++
		return nil
	})
	return phys, files, err
}

// pathAllowed reports whether path is exactly allowlisted or under an allowlisted directory.
func pathAllowed(path string, allowed map[string]struct{}) bool {
	if _, ok := allowed[path]; ok {
		return true
	}
	for rule := range allowed {
		rule = strings.TrimSuffix(rule, "/")
		if strings.HasPrefix(path, rule+"/") {
			return true
		}
	}
	return false
}
