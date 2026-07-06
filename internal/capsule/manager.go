package capsule

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"aort-r/internal/evidence"
)

const (
	ModeReal     = "real"
	ModeDegraded = "degraded"

	KillMethodCgroupKill        = "cgroup.kill"
	KillMethodPidSignalFallback = "pid-signal-fallback"
)

type Config struct {
	Root              string
	ForceReal         bool
	AllowDegraded     bool
	MemoryMax         string
	PidsMax           string
	CPUMax            string
	SignalFunc        func(pid int, signal syscall.Signal) error
	SignalGracePeriod time.Duration
}

type Runtime struct {
	AgentID    string `json:"agent_id"`
	CgroupPath string `json:"cgroup_path"`
	Mode       string `json:"capsule_mode"`
	Error      string `json:"error,omitempty"`
}

type Stats struct {
	MemoryCurrent int64             `json:"memory_current"`
	PidsCurrent   int64             `json:"pids_current"`
	CPUStat       map[string]uint64 `json:"cpu_stat"`
	Events        map[string]uint64 `json:"events"`
	Frozen        bool              `json:"frozen"`
	Mode          string            `json:"capsule_mode"`
	Error         string            `json:"error,omitempty"`
}

type KillResult struct {
	AgentID        string        `json:"agent_id"`
	KillMethod     string        `json:"kill_method"`
	EvidenceMode   evidence.Mode `json:"evidence_mode"`
	FallbackReason string        `json:"fallback_reason,omitempty"`
}

type Manager struct {
	mu       sync.RWMutex
	cfg      Config
	runtimes map[string]Runtime
	pids     map[string]int
}

func NewManager(cfg Config) *Manager {
	if cfg.Root == "" {
		cfg.Root = "/sys/fs/cgroup/aort.slice"
	}
	if cfg.MemoryMax == "" {
		cfg.MemoryMax = "256M"
	}
	if cfg.PidsMax == "" {
		cfg.PidsMax = "64"
	}
	if cfg.CPUMax == "" {
		cfg.CPUMax = "100000 100000"
	}
	if cfg.SignalFunc == nil {
		cfg.SignalFunc = syscall.Kill
	}
	if cfg.SignalGracePeriod == 0 {
		cfg.SignalGracePeriod = 500 * time.Millisecond
	}
	return &Manager{
		cfg:      cfg,
		runtimes: make(map[string]Runtime),
		pids:     make(map[string]int),
	}
}

func (m *Manager) Prepare(agentID string, pid int) (Runtime, error) {
	if err := m.available(); err != nil {
		return m.degradedOrError(agentID, pid, err)
	}

	if err := enableSubtreeControllers(m.cfg.Root, []string{"cpu", "memory", "pids"}); err != nil {
		return m.degradedOrError(agentID, pid, fmt.Errorf("enable cgroup subtree controllers: %w", err))
	}

	path := filepath.Join(m.cfg.Root, safeName(agentID))
	if err := os.MkdirAll(path, 0o755); err != nil {
		return m.degradedOrError(agentID, pid, fmt.Errorf("create cgroup: %w", err))
	}
	files := map[string]string{
		"memory.max":   m.cfg.MemoryMax,
		"pids.max":     m.cfg.PidsMax,
		"cpu.max":      m.cfg.CPUMax,
		"cgroup.procs": strconv.Itoa(pid),
	}
	for name, value := range files {
		if err := os.WriteFile(filepath.Join(path, name), []byte(value+"\n"), 0o644); err != nil {
			return m.degradedOrError(agentID, pid, fmt.Errorf("write %s: %w", name, err))
		}
	}

	rt := Runtime{AgentID: agentID, CgroupPath: path, Mode: ModeReal}
	m.mu.Lock()
	m.runtimes[agentID] = rt
	m.pids[agentID] = pid
	m.mu.Unlock()
	return rt, nil
}

func (m *Manager) degradedOrError(agentID string, pid int, err error) (Runtime, error) {
	if !m.cfg.AllowDegraded {
		return Runtime{}, err
	}
	rt := Runtime{AgentID: agentID, CgroupPath: "degraded://" + safeName(agentID), Mode: ModeDegraded, Error: err.Error()}
	m.mu.Lock()
	m.runtimes[agentID] = rt
	m.pids[agentID] = pid
	m.mu.Unlock()
	return rt, nil
}

func (m *Manager) Runtime(agentID string) (Runtime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rt, ok := m.runtimes[agentID]
	return rt, ok
}

func (m *Manager) Stats(agentID string) Stats {
	rt, ok := m.Runtime(agentID)
	if !ok {
		return Stats{Mode: ModeDegraded, Error: "capsule not prepared"}
	}
	if rt.Mode != ModeReal {
		return Stats{Mode: rt.Mode, Error: rt.Error}
	}
	return Stats{
		MemoryCurrent: readInt(filepath.Join(rt.CgroupPath, "memory.current")),
		PidsCurrent:   readInt(filepath.Join(rt.CgroupPath, "pids.current")),
		CPUStat:       readKV(filepath.Join(rt.CgroupPath, "cpu.stat")),
		Events:        readKV(filepath.Join(rt.CgroupPath, "cgroup.events")),
		Frozen:        readInt(filepath.Join(rt.CgroupPath, "cgroup.freeze")) == 1,
		Mode:          rt.Mode,
	}
}

func (m *Manager) Freeze(agentID string) error {
	return m.writeFreeze(agentID, "1")
}

func (m *Manager) Unfreeze(agentID string) error {
	return m.writeFreeze(agentID, "0")
}

func (m *Manager) Kill(agentID string) (KillResult, error) {
	result := KillResult{
		AgentID:      agentID,
		KillMethod:   KillMethodPidSignalFallback,
		EvidenceMode: evidence.ModeDegraded,
	}
	rt, ok := m.Runtime(agentID)
	if ok && rt.Mode == ModeReal {
		killFile := filepath.Join(rt.CgroupPath, "cgroup.kill")
		if err := os.WriteFile(killFile, []byte("1\n"), 0o644); err == nil {
			result.KillMethod = KillMethodCgroupKill
			result.EvidenceMode = evidence.ModeRealCgroupV2
			return result, nil
		} else {
			result.FallbackReason = "cgroup.kill failed: " + err.Error()
		}
	}
	m.mu.RLock()
	pid := m.pids[agentID]
	m.mu.RUnlock()
	if pid == 0 {
		return result, fmt.Errorf("agent %q has no pid", agentID)
	}
	if err := m.cfg.SignalFunc(pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return result, err
	}
	if m.cfg.SignalGracePeriod > 0 {
		time.Sleep(m.cfg.SignalGracePeriod)
	}
	_ = m.cfg.SignalFunc(pid, syscall.SIGKILL)
	return result, nil
}

func (m *Manager) Destroy(agentID string) error {
	rt, ok := m.Runtime(agentID)
	if !ok {
		return fmt.Errorf("agent %q has no capsule", agentID)
	}
	m.mu.Lock()
	delete(m.runtimes, agentID)
	delete(m.pids, agentID)
	m.mu.Unlock()
	if rt.Mode != ModeReal {
		return nil
	}
	if err := os.Remove(rt.CgroupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("destroy cgroup: %w", err)
	}
	return nil
}

func (m *Manager) writeFreeze(agentID string, value string) error {
	rt, ok := m.Runtime(agentID)
	if !ok {
		return fmt.Errorf("agent %q has no capsule", agentID)
	}
	if rt.Mode != ModeReal {
		return fmt.Errorf("capsule degraded: %s", rt.Error)
	}
	return os.WriteFile(filepath.Join(rt.CgroupPath, "cgroup.freeze"), []byte(value+"\n"), 0o644)
}

func (m *Manager) available() error {
	if m.cfg.ForceReal {
		return nil
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("cgroup v2 requires linux, got %s", runtime.GOOS)
	}
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		return fmt.Errorf("cgroup v2 unavailable: %w", err)
	}
	if err := os.MkdirAll(m.cfg.Root, 0o755); err != nil {
		return fmt.Errorf("cannot create cgroup root: %w", err)
	}
	return nil
}

func enableSubtreeControllers(root string, wanted []string) error {
	controllersPath := filepath.Join(root, "cgroup.controllers")
	subtreePath := filepath.Join(root, "cgroup.subtree_control")
	controllersData, err := os.ReadFile(controllersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", controllersPath, err)
	}
	if _, err := os.Stat(subtreePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", subtreePath, err)
	}

	available := map[string]bool{}
	for _, controller := range strings.Fields(string(controllersData)) {
		available[controller] = true
	}
	commands := make([]string, 0, len(wanted))
	for _, controller := range wanted {
		if available[controller] {
			commands = append(commands, "+"+controller)
		}
	}
	if len(commands) == 0 {
		return nil
	}
	if err := os.WriteFile(subtreePath, []byte(strings.Join(commands, " ")+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", subtreePath, err)
	}
	return nil
}

func safeName(value string) string {
	return strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(value)
}

func readInt(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func readKV(path string) map[string]uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]uint64{}
	}
	out := make(map[string]uint64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			out[fields[0]] = value
		}
	}
	return out
}
