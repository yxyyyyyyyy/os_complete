package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aort-r/internal/avp"
)

type Snapshot struct {
	TaskID            string              `json:"task_id"`
	Sequence          int                 `json:"sequence"`
	Agents            []avp.AVP           `json:"agents"`
	PageTables        map[string][]string `json:"page_tables"`
	SchedulerVRuntime map[string]uint64   `json:"scheduler_vruntime,omitempty"`
	TraceOffset       int64               `json:"trace_offset,omitempty"`
	Mode              string              `json:"mode"`
	CreatedAt         int64               `json:"created_at"`
}

type RecoveryReport struct {
	Mode           string          `json:"mode"`
	Degraded       bool            `json:"degraded"`
	Reason         string          `json:"reason"`
	TaskCount      int             `json:"task_count"`
	RecoveredAt    int64           `json:"recovered_at"`
	RecoveredTasks []RecoveredTask `json:"recovered_tasks"`
}

type RecoveredTask struct {
	TaskID            string            `json:"task_id"`
	Sequence          int               `json:"sequence"`
	Status            string            `json:"status"`
	AgentCount        int               `json:"agent_count"`
	CompletedAgents   []string          `json:"completed_agents"`
	ReadyAgents       []string          `json:"ready_agents"`
	PageTableRefs     int               `json:"page_table_refs"`
	SchedulerVRuntime map[string]uint64 `json:"scheduler_vruntime"`
	CreatedAt         int64             `json:"created_at"`
}

type Store struct {
	root string
}

func NewStore(root string) *Store {
	if root == "" {
		root = filepath.Join(os.TempDir(), "aort-checkpoints")
	}
	return &Store{root: root}
}

func (s *Store) Save(snapshot Snapshot) error {
	if snapshot.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if snapshot.Sequence <= 0 {
		snapshot.Sequence = 1
	}
	if snapshot.Mode == "" {
		snapshot.Mode = "runtime-state"
	}
	if snapshot.CreatedAt == 0 {
		snapshot.CreatedAt = time.Now().UnixMilli()
	}
	dir := filepath.Join(s.root, safeName(snapshot.TaskID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("%06d.json", snapshot.Sequence))
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) LoadLatest(taskID string) (Snapshot, error) {
	if taskID == "" {
		return Snapshot{}, fmt.Errorf("task_id is required")
	}
	snapshots, err := s.listTask(taskID)
	if err != nil {
		return Snapshot{}, err
	}
	if len(snapshots) == 0 {
		return Snapshot{}, fmt.Errorf("no checkpoint for task %q", taskID)
	}
	return snapshots[len(snapshots)-1], nil
}

func (s *Store) List() ([]Snapshot, error) {
	var snapshots []Snapshot
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		snapshot, err := readSnapshot(path)
		if err != nil {
			return err
		}
		snapshots = append(snapshots, snapshot)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sortSnapshots(snapshots)
	return snapshots, nil
}

func (s *Store) RecoverAll() (RecoveryReport, error) {
	snapshots, err := s.List()
	if err != nil {
		return RecoveryReport{}, err
	}
	report := RecoveryReport{
		Mode:           "checkpoint-light",
		Degraded:       true,
		Reason:         "restored AVP table, scheduler vruntime, and CVM page references; page contents require the live CVM store or future durable page backing",
		RecoveredAt:    time.Now().UnixMilli(),
		RecoveredTasks: []RecoveredTask{},
	}
	latest := latestByTask(snapshots)
	taskIDs := make([]string, 0, len(latest))
	for taskID := range latest {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)
	for _, taskID := range taskIDs {
		report.RecoveredTasks = append(report.RecoveredTasks, summarizeRecovery(latest[taskID]))
	}
	report.TaskCount = len(report.RecoveredTasks)
	return report, nil
}

func (s *Store) listTask(taskID string) ([]Snapshot, error) {
	dir := filepath.Join(s.root, safeName(taskID))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	snapshots := make([]Snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		snapshot, err := readSnapshot(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	sortSnapshots(snapshots)
	return snapshots, nil
}

func readSnapshot(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func sortSnapshots(snapshots []Snapshot) {
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].TaskID != snapshots[j].TaskID {
			return snapshots[i].TaskID < snapshots[j].TaskID
		}
		if snapshots[i].Sequence != snapshots[j].Sequence {
			return snapshots[i].Sequence < snapshots[j].Sequence
		}
		return snapshots[i].CreatedAt < snapshots[j].CreatedAt
	})
}

func latestByTask(snapshots []Snapshot) map[string]Snapshot {
	latest := make(map[string]Snapshot)
	for _, snapshot := range snapshots {
		if snapshot.TaskID == "" {
			continue
		}
		latest[snapshot.TaskID] = snapshot
	}
	return latest
}

func summarizeRecovery(snapshot Snapshot) RecoveredTask {
	task := RecoveredTask{
		TaskID:            snapshot.TaskID,
		Sequence:          snapshot.Sequence,
		Status:            "completed",
		AgentCount:        len(snapshot.Agents),
		CompletedAgents:   []string{},
		ReadyAgents:       []string{},
		SchedulerVRuntime: copyVRuntime(snapshot.SchedulerVRuntime),
		CreatedAt:         snapshot.CreatedAt,
	}
	for agentID, pageIDs := range snapshot.PageTables {
		if agentID == "" {
			continue
		}
		task.PageTableRefs += len(pageIDs)
	}
	for _, agent := range snapshot.Agents {
		if isTerminal(agent.State) {
			if agent.State == avp.StateCompleted {
				task.CompletedAgents = append(task.CompletedAgents, agent.AgentID)
			}
			continue
		}
		task.Status = "recovered"
		task.ReadyAgents = append(task.ReadyAgents, agent.AgentID)
	}
	sort.Strings(task.CompletedAgents)
	sort.Strings(task.ReadyAgents)
	return task
}

func copyVRuntime(input map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(input))
	for agentID, vruntime := range input {
		out[agentID] = vruntime
	}
	return out
}

func isTerminal(state avp.AgentState) bool {
	return state == avp.StateCompleted || state == avp.StateFailed || state == avp.StateKilled
}

func safeName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(value)
}
