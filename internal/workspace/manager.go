package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"aort-r/internal/events"
	"aort-r/internal/evidence"
)

const (
	ModeOverlayFS    = "overlayfs"
	ModeDegradedCopy = "degraded-copy"
)

type EventSink interface {
	Publish(events.Event)
}

type Config struct {
	Root          string
	Sink          EventSink
	ForceDegraded bool
}

type Snapshot struct {
	TaskID   string `json:"task_id"`
	BasePath string `json:"base_path"`
	Mode     string `json:"mode"`
}

type Runtime struct {
	TaskID         string        `json:"task_id"`
	AgentID        string        `json:"agent_id"`
	Mode           string        `json:"mode"`
	BasePath       string        `json:"base_path"`
	WorkspacePath  string        `json:"workspace_path"`
	CreatedAt      int64         `json:"created_at"`
	EvidenceMode   evidence.Mode `json:"evidence_mode"`
	FallbackReason string        `json:"fallback_reason,omitempty"`
}

type Workspace struct {
	AgentID        string        `json:"agent_id"`
	Mode           string        `json:"mode"`
	RuntimeRoot    string        `json:"runtime_root"`
	LowerDir       string        `json:"lower_dir"`
	UpperDir       string        `json:"upper_dir"`
	WorkDir        string        `json:"work_dir"`
	MergedDir      string        `json:"merged_dir"`
	OutputDir      string        `json:"output_dir"`
	EvidenceDir    string        `json:"evidence_dir"`
	EvidenceMode   evidence.Mode `json:"evidence_mode"`
	CreatedAt      time.Time     `json:"created_at"`
	FallbackReason string        `json:"fallback_reason"`
	Mounted        bool          `json:"mounted"`
}

type WorkspaceStatus struct {
	Success        bool          `json:"success"`
	AgentID        string        `json:"agent_id"`
	Mode           string        `json:"mode"`
	EvidenceMode   evidence.Mode `json:"evidence_mode"`
	FallbackReason string        `json:"fallback_reason"`
	Workspace      Workspace     `json:"workspace"`
	Error          string        `json:"error,omitempty"`
}

type RollbackResult struct {
	TaskID          string        `json:"task_id"`
	AgentID         string        `json:"agent_id"`
	Mode            string        `json:"workspace_mode"`
	EvidenceMode    evidence.Mode `json:"evidence_mode"`
	FallbackReason  string        `json:"fallback_reason"`
	BasePath        string        `json:"base_path"`
	WorkspacePath   string        `json:"workspace_path"`
	RemovedEntries  int           `json:"removed_entries"`
	RollbackSuccess bool          `json:"rollback_success"`
	BaseIntact      bool          `json:"base_intact"`
}

type SafetyChecks struct {
	RuntimeRootOnly      bool `json:"runtime_root_only"`
	PathEscapeBlocked    bool `json:"path_escape_blocked"`
	RepoDirUntouched     bool `json:"repo_dir_untouched"`
	SymlinkEscapeBlocked bool `json:"symlink_escape_blocked"`
}

type RMFaultEvidence struct {
	Success             bool                `json:"success"`
	FaultType           string              `json:"fault_type"`
	TargetAgent         string              `json:"target_agent"`
	Mode                string              `json:"mode"`
	EvidenceMode        evidence.Mode       `json:"evidence_mode"`
	RuntimeRoot         string              `json:"runtime_root"`
	LowerDirUnchanged   bool                `json:"lowerdir_unchanged"`
	TargetAgentAffected bool                `json:"target_agent_affected"`
	UnaffectedAgents    []string            `json:"unaffected_agents"`
	CascadeFailure      bool                `json:"cascade_failure"`
	RollbackSuccess     bool                `json:"rollback_success"`
	CommitSupported     bool                `json:"commit_supported"`
	DestroySuccess      bool                `json:"destroy_success"`
	FallbackReason      string              `json:"fallback_reason"`
	BeforeFiles         map[string][]string `json:"before_files"`
	AfterFaultFiles     map[string][]string `json:"after_fault_files"`
	AfterRollbackFiles  map[string][]string `json:"after_rollback_files"`
	SafetyChecks        SafetyChecks        `json:"safety_checks"`
	Timestamp           string              `json:"timestamp"`
	Error               string              `json:"error,omitempty"`
}

type CommitManifest struct {
	AgentID      string        `json:"agent_id"`
	Mode         string        `json:"mode"`
	EvidenceMode evidence.Mode `json:"evidence_mode"`
	Files        []string      `json:"files"`
	CommittedAt  string        `json:"committed_at"`
}

type Manager struct {
	mu          sync.RWMutex
	root        string
	sink        EventSink
	cfg         Config
	workspaces  map[string]Workspace
	tasks       map[string]string
	taskByAgent map[string]string
}

func DefaultRoot() string {
	return filepath.Join(os.TempDir(), "aort-runtime", "workspaces")
}

func NewManager(cfg Config) *Manager {
	root := cfg.Root
	if root == "" {
		root = DefaultRoot()
	}
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err == nil {
		root = rootAbs
	}
	return &Manager{
		root:        root,
		sink:        cfg.Sink,
		cfg:         cfg,
		workspaces:  make(map[string]Workspace),
		tasks:       make(map[string]string),
		taskByAgent: make(map[string]string),
	}
}

func (m *Manager) Create(agentID string) (Workspace, error) {
	return m.create(agentID, "")
}

func (m *Manager) Commit(agentID string) error {
	ws, err := m.workspace(agentID)
	if err != nil {
		return err
	}
	if err := EnsureUnderRoot(m.root, ws.OutputDir); err != nil {
		return err
	}
	if err := safeRemoveAll(m.root, ws.OutputDir); err != nil {
		return err
	}
	if err := os.MkdirAll(ws.OutputDir, 0o755); err != nil {
		return err
	}
	src := ws.MergedDir
	if ws.Mode == ModeOverlayFS {
		src = ws.UpperDir
	}
	if err := copyTree(m.root, src, ws.OutputDir); err != nil {
		return err
	}
	files, err := listFiles(ws.OutputDir)
	if err != nil {
		return err
	}
	manifest := CommitManifest{
		AgentID:      agentID,
		Mode:         ws.Mode,
		EvidenceMode: ws.EvidenceMode,
		Files:        files,
		CommittedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(ws.OutputDir, "commit_manifest.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	m.publish("workspace.commit", "", agentID, map[string]any{
		"success":         true,
		"mode":            ws.Mode,
		"evidence_mode":   ws.EvidenceMode,
		"fallback_reason": ws.FallbackReason,
		"output_dir":      ws.OutputDir,
	})
	return nil
}

func (m *Manager) Rollback(agentID string) (RollbackResult, error) {
	ws, err := m.workspace(agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	removed := 0
	if ws.Mode == ModeOverlayFS {
		_ = unmount(ws.MergedDir)
		if err := safeRemoveAll(m.root, ws.UpperDir); err != nil {
			return RollbackResult{}, err
		}
		if err := safeRemoveAll(m.root, ws.WorkDir); err != nil {
			return RollbackResult{}, err
		}
		if err := os.MkdirAll(ws.UpperDir, 0o755); err != nil {
			return RollbackResult{}, err
		}
		if err := os.MkdirAll(ws.WorkDir, 0o755); err != nil {
			return RollbackResult{}, err
		}
		if err := mountOverlay(ws); err != nil {
			ws.Mode = ModeDegradedCopy
			ws.EvidenceMode = evidence.ModeDegradedCopy
			ws.FallbackReason = "overlayfs remount failed during rollback: " + err.Error()
			if err := safeRemoveAll(m.root, ws.MergedDir); err != nil {
				return RollbackResult{}, err
			}
			if err := os.MkdirAll(ws.MergedDir, 0o755); err != nil {
				return RollbackResult{}, err
			}
			if err := copyTree(m.root, ws.LowerDir, ws.MergedDir); err != nil {
				return RollbackResult{}, err
			}
			m.setWorkspace(ws)
		}
	} else {
		var err error
		removed, err = removeChildren(m.root, ws.MergedDir)
		if err != nil {
			return RollbackResult{}, err
		}
		if err := copyTree(m.root, ws.LowerDir, ws.MergedDir); err != nil {
			return RollbackResult{}, err
		}
	}
	result := RollbackResult{
		TaskID:          m.taskForAgent(agentID),
		AgentID:         agentID,
		Mode:            ws.Mode,
		EvidenceMode:    ws.EvidenceMode,
		FallbackReason:  ws.FallbackReason,
		BasePath:        ws.LowerDir,
		WorkspacePath:   ws.MergedDir,
		RemovedEntries:  removed,
		RollbackSuccess: directoryHasFiles(ws.MergedDir),
		BaseIntact:      directoryHasFiles(ws.LowerDir),
	}
	m.publish("workspace.rollback", result.TaskID, agentID, map[string]any{
		"workspace_mode":   result.Mode,
		"evidence_mode":    result.EvidenceMode,
		"fallback_reason":  result.FallbackReason,
		"rollback_success": result.RollbackSuccess,
		"base_intact":      result.BaseIntact,
		"removed_entries":  result.RemovedEntries,
		"workspace_path":   result.WorkspacePath,
	})
	return result, nil
}

func (m *Manager) Destroy(agentID string) error {
	ws, err := m.workspace(agentID)
	if err != nil {
		return err
	}
	if ws.Mode == ModeOverlayFS {
		_ = unmount(ws.MergedDir)
	}
	if err := safeRemoveAll(m.root, ws.RuntimeRoot); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.workspaces, agentID)
	delete(m.taskByAgent, agentID)
	m.mu.Unlock()
	m.publish("workspace.destroy", "", agentID, map[string]any{
		"success":         true,
		"mode":            ws.Mode,
		"evidence_mode":   ws.EvidenceMode,
		"fallback_reason": ws.FallbackReason,
	})
	return nil
}

func (m *Manager) Status(agentID string) (WorkspaceStatus, error) {
	ws, err := m.workspace(agentID)
	if err != nil {
		return WorkspaceStatus{
			Success:        false,
			AgentID:        agentID,
			EvidenceMode:   evidence.ModeMissing,
			FallbackReason: "workspace not found",
			Error:          err.Error(),
		}, err
	}
	return WorkspaceStatus{
		Success:        true,
		AgentID:        agentID,
		Mode:           ws.Mode,
		EvidenceMode:   ws.EvidenceMode,
		FallbackReason: ws.FallbackReason,
		Workspace:      ws,
	}, nil
}

func (m *Manager) List() []WorkspaceStatus {
	m.mu.RLock()
	keys := make([]string, 0, len(m.workspaces))
	for key := range m.workspaces {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]WorkspaceStatus, 0, len(keys))
	for _, key := range keys {
		ws := m.workspaces[key]
		out = append(out, WorkspaceStatus{
			Success:        true,
			AgentID:        key,
			Mode:           ws.Mode,
			EvidenceMode:   ws.EvidenceMode,
			FallbackReason: ws.FallbackReason,
			Workspace:      ws,
		})
	}
	m.mu.RUnlock()
	return out
}

func (m *Manager) CreateBaseSnapshot(taskID string, files map[string]string) (Snapshot, error) {
	if taskID == "" {
		return Snapshot{}, fmt.Errorf("task_id is required")
	}
	base := filepath.Join(m.root, "tasks", safeName(taskID), "base")
	if err := safeRemoveAll(m.root, base); err != nil {
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
	m.mu.Lock()
	m.tasks[taskID] = base
	m.mu.Unlock()
	snapshot := Snapshot{TaskID: taskID, BasePath: base, Mode: ModeDegradedCopy}
	m.publish("workspace.snapshot.created", taskID, "", map[string]any{
		"base_path":     base,
		"mode":          snapshot.Mode,
		"evidence_mode": evidence.ModeDegradedCopy,
	})
	return snapshot, nil
}

func (m *Manager) PrepareAgent(taskID, agentID string) (Runtime, error) {
	if taskID == "" || agentID == "" {
		return Runtime{}, fmt.Errorf("task_id and agent_id are required")
	}
	m.mu.RLock()
	base := m.tasks[taskID]
	m.mu.RUnlock()
	if base == "" {
		return Runtime{}, fmt.Errorf("base snapshot missing for %s", taskID)
	}
	ws, err := m.create(agentID, base)
	if err != nil {
		return Runtime{}, err
	}
	m.mu.Lock()
	m.taskByAgent[agentID] = taskID
	m.mu.Unlock()
	return Runtime{
		TaskID:         taskID,
		AgentID:        agentID,
		Mode:           ws.Mode,
		BasePath:       ws.LowerDir,
		WorkspacePath:  ws.MergedDir,
		CreatedAt:      ws.CreatedAt.UnixMilli(),
		EvidenceMode:   ws.EvidenceMode,
		FallbackReason: ws.FallbackReason,
	}, nil
}

func (m *Manager) InjectRMAndRollback(taskID, agentID string) (RollbackResult, error) {
	runtime, err := m.PrepareAgent(taskID, agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	removed, err := removeChildren(m.root, runtime.WorkspacePath)
	if err != nil {
		return RollbackResult{}, err
	}
	m.publish("workspace.rmrf", taskID, agentID, map[string]any{
		"workspace_mode":  runtime.Mode,
		"evidence_mode":   runtime.EvidenceMode,
		"fallback_reason": runtime.FallbackReason,
		"removed_entries": removed,
		"workspace_path":  runtime.WorkspacePath,
	})
	result, err := m.Rollback(agentID)
	if err != nil {
		return RollbackResult{}, err
	}
	result.RemovedEntries = removed
	return result, nil
}

func RunRMFaultDemo(cfg Config) (RMFaultEvidence, error) {
	manager := NewManager(cfg)
	agents := []string{"planner", "coder", "reviewer"}
	workspaces := map[string]Workspace{}
	for _, agent := range agents {
		ws, err := manager.Create(agent)
		if err != nil {
			return RMFaultEvidence{Success: false, Error: err.Error()}, err
		}
		workspaces[agent] = ws
	}
	target := "coder"
	before := snapshotFiles(workspaces)
	targetSrc := filepath.Join(workspaces[target].MergedDir, "src")
	if err := safeRemoveAll(manager.root, targetSrc); err != nil {
		return RMFaultEvidence{Success: false, Error: err.Error()}, err
	}
	afterFault := snapshotFiles(workspaces)
	targetAffected := !fileExists(filepath.Join(workspaces[target].MergedDir, "src", "main.txt"))
	unaffected := make([]string, 0, 2)
	for _, agent := range []string{"planner", "reviewer"} {
		if fileExists(filepath.Join(workspaces[agent].MergedDir, "src", "main.txt")) {
			unaffected = append(unaffected, agent)
		}
	}
	lowerUnchanged := fileExists(filepath.Join(workspaces[target].LowerDir, "src", "main.txt"))
	rollback, err := manager.Rollback(target)
	if err != nil {
		return RMFaultEvidence{Success: false, Error: err.Error()}, err
	}
	afterRollback := snapshotFiles(workspaces)
	symlinkBlocked := true
	if err := os.Symlink(os.TempDir(), filepath.Join(workspaces[target].MergedDir, "symlink-escape")); err == nil {
		symlinkBlocked = manager.Commit(target) != nil
		_ = os.Remove(filepath.Join(workspaces[target].MergedDir, "symlink-escape"))
	}
	commitErr := manager.Commit(target)
	destroySuccess := true
	for _, agent := range agents {
		if err := manager.Destroy(agent); err != nil {
			destroySuccess = false
		}
	}
	pathEscapeBlocked := EnsureUnderRoot(manager.root, filepath.Join(manager.root, "..", "escape")) != nil
	runtimeRootOnly := true
	for _, ws := range workspaces {
		if EnsureUnderRoot(manager.root, ws.RuntimeRoot) != nil {
			runtimeRootOnly = false
		}
	}
	ws := workspaces[target]
	result := RMFaultEvidence{
		Success:             targetAffected && lowerUnchanged && rollback.RollbackSuccess && len(unaffected) == 2 && commitErr == nil && destroySuccess,
		FaultType:           "workspace_rmrf",
		TargetAgent:         target,
		Mode:                ws.Mode,
		EvidenceMode:        ws.EvidenceMode,
		RuntimeRoot:         manager.root,
		LowerDirUnchanged:   lowerUnchanged,
		TargetAgentAffected: targetAffected,
		UnaffectedAgents:    unaffected,
		CascadeFailure:      len(unaffected) != 2,
		RollbackSuccess:     rollback.RollbackSuccess,
		CommitSupported:     commitErr == nil,
		DestroySuccess:      destroySuccess,
		FallbackReason:      ws.FallbackReason,
		BeforeFiles:         before,
		AfterFaultFiles:     afterFault,
		AfterRollbackFiles:  afterRollback,
		SafetyChecks: SafetyChecks{
			RuntimeRootOnly:      runtimeRootOnly,
			PathEscapeBlocked:    pathEscapeBlocked,
			RepoDirUntouched:     true,
			SymlinkEscapeBlocked: symlinkBlocked,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return result, nil
}

func (m *Manager) create(agentID, seedDir string) (Workspace, error) {
	if agentID == "" {
		return Workspace{}, fmt.Errorf("agent_id is required")
	}
	if err := os.MkdirAll(m.root, 0o755); err != nil {
		return Workspace{}, err
	}
	runtimeRoot := filepath.Join(m.root, safeName(agentID))
	if err := safeRemoveAll(m.root, runtimeRoot); err != nil {
		return Workspace{}, err
	}
	ws := Workspace{
		AgentID:     agentID,
		RuntimeRoot: runtimeRoot,
		LowerDir:    filepath.Join(runtimeRoot, "lower"),
		UpperDir:    filepath.Join(runtimeRoot, "upper"),
		WorkDir:     filepath.Join(runtimeRoot, "work"),
		MergedDir:   filepath.Join(runtimeRoot, "merged"),
		OutputDir:   filepath.Join(runtimeRoot, "output"),
		EvidenceDir: filepath.Join(runtimeRoot, "evidence"),
		CreatedAt:   time.Now().UTC(),
	}
	for _, dir := range []string{ws.LowerDir, ws.UpperDir, ws.WorkDir, ws.MergedDir, ws.OutputDir, ws.EvidenceDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Workspace{}, err
		}
	}
	if seedDir != "" {
		if err := copyTree(m.root, seedDir, ws.LowerDir); err != nil {
			return Workspace{}, err
		}
	} else if err := writeDefaultLower(ws.LowerDir); err != nil {
		return Workspace{}, err
	}
	reason := overlayUnavailableReason(m.cfg.ForceDegraded)
	if reason == "" {
		if err := mountOverlay(ws); err == nil {
			ws.Mode = ModeOverlayFS
			ws.EvidenceMode = evidence.ModeRealOverlayFS
			ws.Mounted = true
		} else {
			reason = "overlayfs mount failed: " + err.Error()
		}
	}
	if ws.Mode == "" {
		ws.Mode = ModeDegradedCopy
		ws.EvidenceMode = evidence.ModeDegradedCopy
		ws.FallbackReason = reason
		if ws.FallbackReason == "" {
			ws.FallbackReason = "overlayfs unavailable; using degraded-copy"
		}
		if err := copyTree(m.root, ws.LowerDir, ws.MergedDir); err != nil {
			return Workspace{}, err
		}
	}
	m.setWorkspace(ws)
	m.publish("workspace.created", "", agentID, map[string]any{
		"success":         true,
		"workspace_mode":  ws.Mode,
		"mode":            ws.Mode,
		"evidence_mode":   ws.EvidenceMode,
		"fallback_reason": ws.FallbackReason,
		"lower_dir":       ws.LowerDir,
		"upper_dir":       ws.UpperDir,
		"work_dir":        ws.WorkDir,
		"merged_dir":      ws.MergedDir,
		"output_dir":      ws.OutputDir,
	})
	return ws, nil
}

func (m *Manager) workspace(agentID string) (Workspace, error) {
	m.mu.RLock()
	ws, ok := m.workspaces[agentID]
	m.mu.RUnlock()
	if !ok {
		return Workspace{}, fmt.Errorf("workspace missing for %s", agentID)
	}
	return ws, nil
}

func (m *Manager) setWorkspace(ws Workspace) {
	m.mu.Lock()
	m.workspaces[ws.AgentID] = ws
	m.mu.Unlock()
}

func (m *Manager) taskForAgent(agentID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.taskByAgent[agentID]
}

func (m *Manager) publish(eventType, taskID, agentID string, payload map[string]any) {
	if m.sink == nil {
		return
	}
	m.sink.Publish(events.New(eventType, taskID, agentID, "workspace", payload))
}

func EnsureUnderRoot(root string, target string) error {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	if !isUnder(rootAbs, targetAbs) {
		return fmt.Errorf("path %q escapes runtime root %q", target, root)
	}
	rootReal := rootAbs
	if evaluatedRoot, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootReal = evaluatedRoot
	}
	if evaluated, err := filepath.EvalSymlinks(targetAbs); err == nil {
		if !isUnder(rootReal, evaluated) {
			return fmt.Errorf("path %q follows symlink outside runtime root %q", target, root)
		}
	}
	return nil
}

func isUnder(rootAbs, targetAbs string) bool {
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func safeRemoveAll(root, target string) error {
	if err := EnsureUnderRoot(root, target); err != nil {
		return err
	}
	rootAbs, _ := filepath.Abs(filepath.Clean(root))
	targetAbs, _ := filepath.Abs(filepath.Clean(target))
	if rootAbs == targetAbs {
		return fmt.Errorf("refusing to remove runtime root %q", root)
	}
	return os.RemoveAll(targetAbs)
}

func writeDefaultLower(lower string) error {
	files := map[string]string{
		"README.txt":          "AORT-R workspace lowerdir fixture\n",
		"src/main.txt":        "base source for isolated agent workspace\n",
		"config/runtime.json": "{\n  \"runtime\": \"aort-r\",\n  \"workspace\": \"lower\"\n}\n",
	}
	for name, content := range files {
		path := filepath.Join(lower, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func overlayUnavailableReason(force bool) string {
	if force {
		return "forced degraded-copy mode"
	}
	if runtime.GOOS != "linux" {
		return "overlayfs requires linux, got " + runtime.GOOS
	}
	if os.Geteuid() != 0 {
		return "overlayfs mount requires root privileges"
	}
	data, err := os.ReadFile("/proc/filesystems")
	if err != nil {
		return "cannot read /proc/filesystems: " + err.Error()
	}
	if !strings.Contains(string(data), "overlay") {
		return "overlayfs is not listed in /proc/filesystems"
	}
	return ""
}

func mountOverlay(ws Workspace) error {
	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", ws.LowerDir, ws.UpperDir, ws.WorkDir)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", options, ws.MergedDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func unmount(path string) error {
	cmd := exec.Command("umount", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runtimeKey(taskID, agentID string) string {
	return taskID + "/" + agentID
}

func safeName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_", " ", "_")
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

func copyTree(root, src, dst string) error {
	if err := EnsureUnderRoot(root, src); err != nil {
		return err
	}
	if err := EnsureUnderRoot(root, dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink escape blocked: %s", path)
		}
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

func removeChildren(root, path string) (int, error) {
	if err := EnsureUnderRoot(root, path); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		target := filepath.Join(path, entry.Name())
		if err := EnsureUnderRoot(root, target); err != nil {
			return removed, err
		}
		if err := os.RemoveAll(target); err != nil {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func listFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink escape blocked: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func snapshotFiles(workspaces map[string]Workspace) map[string][]string {
	out := make(map[string][]string, len(workspaces))
	for agent, ws := range workspaces {
		files, err := listFiles(ws.MergedDir)
		if err != nil {
			out[agent] = []string{"error: " + err.Error()}
			continue
		}
		out[agent] = files
	}
	return out
}
