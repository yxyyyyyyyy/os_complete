package codebasedag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SeedIncompleteJudgeTasks flips judge markers to seed-incomplete in workDir
// while keeping function bodies intact (Ready() gates behavior).
func SeedIncompleteJudgeTasks(workDir string) error {
	if workDir == "" {
		return fmt.Errorf("workDir is required")
	}
	replacements := []struct {
		rel  string
		from string
		to   string
	}{
		{"internal/codebasedag/judge_resource.go", `const ResourceJudgeMarker = "judge-resource-complete"`, `const ResourceJudgeMarker = "seed-incomplete"`},
		{"internal/codebasedag/judge_context.go", `const ContextJudgeMarker = "judge-context-complete"`, `const ContextJudgeMarker = "seed-incomplete"`},
		{"internal/codebasedag/judge_evidence.go", `const EvidenceJudgeMarker = "judge-evidence-complete"`, `const EvidenceJudgeMarker = "seed-incomplete"`},
	}
	for _, item := range replacements {
		path := filepath.Join(workDir, item.rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("seed %s: %w", item.rel, err)
		}
		body := string(data)
		if strings.Contains(body, item.to) {
			continue // already seeded (const line, not comment prose)
		}
		updated := strings.Replace(body, item.from, item.to, 1)
		if updated == body {
			return fmt.Errorf("seed %s: marker assignment %q not found", item.rel, item.from)
		}
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// CompleteJudgeMarkersPatch returns a unified diff restoring one judge marker.
func CompleteJudgeMarkersPatch(nodeID string) (patch string, files []string, err error) {
	switch {
	case nodeID == "resource-coder" || strings.HasPrefix(nodeID, "fixer-"):
		return markerPatch("internal/codebasedag/judge_resource.go", "ResourceJudgeMarker", "seed-incomplete", "judge-resource-complete"),
			[]string{"internal/codebasedag/judge_resource.go"}, nil
	case nodeID == "context-coder":
		return markerPatch("internal/codebasedag/judge_context.go", "ContextJudgeMarker", "seed-incomplete", "judge-context-complete"),
			[]string{"internal/codebasedag/judge_context.go"}, nil
	case nodeID == "evidence-coder":
		return markerPatch("internal/codebasedag/judge_evidence.go", "EvidenceJudgeMarker", "seed-incomplete", "judge-evidence-complete"),
			[]string{"internal/codebasedag/judge_evidence.go"}, nil
	default:
		return "", nil, fmt.Errorf("no judge patch for node %q", nodeID)
	}
}

func markerPatch(path, constName, oldMark, newMark string) string {
	return fmt.Sprintf(`diff --git a/%s b/%s
--- a/%s
+++ b/%s
@@ -1,3 +1,3 @@
 package codebasedag
 
-const %s = %q
+const %s = %q
`, path, path, path, path, constName, oldMark, constName, newMark)
}
