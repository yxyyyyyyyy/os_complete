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

func safeName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(value)
}
