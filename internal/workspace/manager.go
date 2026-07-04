package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aort-r/internal/events"
)

const (
	ModeDegradedCopy = "degraded-copy"
)

type EventSink interface {
	Publish(events.Event)
}

type Config struct {
	Root string
	Sink EventSink
}

type Snapshot struct {
	TaskID   string `json:"task_id"`
	BasePath string `json:"base_path"`
	Mode     string `json:"mode"`
}

type Runtime struct {
	TaskID        string `json:"task_id"`
	AgentID       string `json:"agent_id"`
	Mode          string `json:"mode"`
	BasePath      string `json:"base_path"`
	WorkspacePath string `json:"workspace_path"`
	CreatedAt     int64  `json:"created_at"`
}

type RollbackResult struct {
	TaskID          string `json:"task_id"`
	AgentID         string `json:"agent_id"`
	Mode            string `json:"workspace_mode"`
	BasePath        string `json:"base_path"`
	WorkspacePath   string `json:"workspace_path"`
	RemovedEntries  int    `json:"removed_entries"`
	RollbackSuccess bool   `json:"rollback_success"`
	BaseIntact      bool   `json:"base_intact"`
}

type Manager struct {
	mu       sync.RWMutex
	root     string
	sink     EventSink
	runtimes map[string]Runtime
}

func NewManager(cfg Config) *Manager {
	root := cfg.Root
	if root == "" {
		root = filepath.Join(os.TempDir(), "aort-workspaces")
	}
	return &Manager{
		root:     root,
		sink:     cfg.Sink,
		runtimes: make(map[string]Runtime),
	}
}

func (m *Manager) CreateBaseSnapshot(taskID string, files map[string]string) (Snapshot, error) {
	if taskID == "" {
		return Snapshot{}, fmt.Errorf("task_id is required")
	}
	base := filepath.Join(m.root, "tasks", safeName(taskID), "base")
	if err := os.RemoveAll(base); err != nil {
		return Snapshot{}, err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return Snapshot{}, err
	}
	for name, content := range files {
		path, err := confinedPath(base, name)
		if err != nil {
			return Snapshot{}, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Snapshot{}, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return Snapshot{}, err
		}
	}
	snapshot := Snapshot{TaskID: taskID, BasePath: base, Mode: ModeDegradedCopy}
	m.publish("workspace.snapshot.created", taskID, "", map[string]any{
		"base_path": base,
		"mode":      snapshot.Mode,
	})
	return snapshot, nil
}

func (m *Manager) PrepareAgent(taskID, agentID string) (Runtime, error) {
	if taskID == "" || agentID == "" {
		return Runtime{}, fmt.Errorf("task_id and agent_id are required")
	}
	base := filepath.Join(m.root, "tasks", safeName(taskID), "base")
	if _, err := os.Stat(base); err != nil {
		return Runtime{}, fmt.Errorf("base snapshot missing: %w", err)
	}
	workspace := filepath.Join(m.root, "agents", safeName(agentID), "workspace")
	if err := os.RemoveAll(workspace); err != nil {
		return Runtime{}, err
	}
	if err := copyTree(base, workspace); err != nil {
		return Runtime{}, err
	}
	runtime := Runtime{
		TaskID:        taskID,
		AgentID:       agentID,
		Mode:          ModeDegradedCopy,
		BasePath:      base,
		WorkspacePath: workspace,
		CreatedAt:     time.Now().UnixMilli(),
	}
	m.mu.Lock()
	m.runtimes[runtimeKey(taskID, agentID)] = runtime
	m.mu.Unlock()
	m.publish("workspace.created", taskID, agentID, map[string]any{
		"workspace_mode": runtime.Mode,
		"base_path":      runtime.BasePath,
		"workspace_path": runtime.WorkspacePath,
	})
	return runtime, nil
}

func (m *Manager) Rollback(taskID, agentID string) (RollbackResult, error) {
	runtime, err := m.runtime(taskID, agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	removed, err := removeChildren(runtime.WorkspacePath)
	if err != nil {
		return RollbackResult{}, err
	}
	if err := copyTree(runtime.BasePath, runtime.WorkspacePath); err != nil {
		return RollbackResult{}, err
	}
	baseIntact := directoryHasFiles(runtime.BasePath)
	result := RollbackResult{
		TaskID:          taskID,
		AgentID:         agentID,
		Mode:            runtime.Mode,
		BasePath:        runtime.BasePath,
		WorkspacePath:   runtime.WorkspacePath,
		RemovedEntries:  removed,
		RollbackSuccess: baseIntact && directoryHasFiles(runtime.WorkspacePath),
		BaseIntact:      baseIntact,
	}
	m.publish("workspace.rollback", taskID, agentID, map[string]any{
		"workspace_mode":   result.Mode,
		"rollback_success": result.RollbackSuccess,
		"base_intact":      result.BaseIntact,
		"removed_entries":  result.RemovedEntries,
		"workspace_path":   result.WorkspacePath,
	})
	return result, nil
}

func (m *Manager) InjectRMAndRollback(taskID, agentID string) (RollbackResult, error) {
	runtime, err := m.PrepareAgent(taskID, agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	removed, err := removeChildren(runtime.WorkspacePath)
	if err != nil {
		return RollbackResult{}, err
	}
	m.publish("workspace.rmrf", taskID, agentID, map[string]any{
		"workspace_mode":  runtime.Mode,
		"removed_entries": removed,
		"workspace_path":  runtime.WorkspacePath,
	})
	result, err := m.Rollback(taskID, agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	result.RemovedEntries = removed
	return result, nil
}

func (m *Manager) runtime(taskID, agentID string) (Runtime, error) {
	m.mu.RLock()
	runtime, ok := m.runtimes[runtimeKey(taskID, agentID)]
	m.mu.RUnlock()
	if !ok {
		return Runtime{}, fmt.Errorf("workspace runtime missing for %s/%s", taskID, agentID)
	}
	return runtime, nil
}

func (m *Manager) publish(eventType, taskID, agentID string, payload map[string]any) {
	if m.sink == nil {
		return
	}
	m.sink.Publish(events.New(eventType, taskID, agentID, "workspace", payload))
}

func runtimeKey(taskID, agentID string) string {
	return taskID + "/" + agentID
}

func safeName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(value)
}

func confinedPath(root, requested string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(rootAbs, requested)
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q escapes workspace", requested)
	}
	return candidate, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func removeChildren(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func directoryHasFiles(path string) bool {
	hasFiles := false
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		hasFiles = true
		return filepath.SkipAll
	})
	return hasFiles
}
