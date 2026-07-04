package checkpoint

import (
	"testing"

	"aort-r/internal/avp"
)

func TestStoreSavesAndLoadsLatestSnapshot(t *testing.T) {
	store := NewStore(t.TempDir())
	first := Snapshot{
		TaskID:   "task-1",
		Sequence: 1,
		Agents: []avp.AVP{
			{AgentID: "planner-1", TaskID: "task-1", Role: "planner", State: avp.StateCompleted, VRuntime: 12},
		},
		PageTables: map[string][]string{"planner-1": {"page-a", "page-b"}},
	}
	second := first
	second.Sequence = 2
	second.Agents[0].VRuntime = 20

	if err := store.Save(first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	if err := store.Save(second); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	latest, err := store.LoadLatest("task-1")
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}
	if latest.Sequence != 2 {
		t.Fatalf("latest sequence = %d", latest.Sequence)
	}
	if latest.Agents[0].VRuntime != 20 {
		t.Fatalf("latest agents = %#v", latest.Agents)
	}
	if latest.PageTables["planner-1"][1] != "page-b" {
		t.Fatalf("page tables = %#v", latest.PageTables)
	}
}

func TestStoreListsSnapshotsByTask(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Save(Snapshot{TaskID: "task-1", Sequence: 1}); err != nil {
		t.Fatalf("Save task-1: %v", err)
	}
	if err := store.Save(Snapshot{TaskID: "task-2", Sequence: 1}); err != nil {
		t.Fatalf("Save task-2: %v", err)
	}

	snapshots, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshots = %#v", snapshots)
	}
}

func TestStoreRecoverAllSummarizesLatestCheckpoint(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Save(Snapshot{
		TaskID:   "task-1",
		Sequence: 1,
		Agents: []avp.AVP{
			{AgentID: "planner-1", TaskID: "task-1", Role: "planner", State: avp.StateCompleted, VRuntime: 10},
			{AgentID: "coder-1", TaskID: "task-1", Role: "coder", State: avp.StateRunning, VRuntime: 7},
		},
		PageTables: map[string][]string{
			"planner-1": {"page-a"},
			"coder-1":   {"page-a", "page-b"},
		},
		SchedulerVRuntime: map[string]uint64{"planner-1": 10, "coder-1": 7},
	}); err != nil {
		t.Fatalf("Save sequence 1: %v", err)
	}
	if err := store.Save(Snapshot{
		TaskID:   "task-1",
		Sequence: 2,
		Agents: []avp.AVP{
			{AgentID: "planner-1", TaskID: "task-1", Role: "planner", State: avp.StateCompleted, VRuntime: 10},
			{AgentID: "coder-1", TaskID: "task-1", Role: "coder", State: avp.StateWaitingTool, VRuntime: 9},
			{AgentID: "tester-1", TaskID: "task-1", Role: "tester", State: avp.StateCreated, VRuntime: 0},
		},
		PageTables: map[string][]string{
			"planner-1": {"page-a"},
			"coder-1":   {"page-a", "page-b"},
			"tester-1":  {"page-a"},
		},
		SchedulerVRuntime: map[string]uint64{"planner-1": 10, "coder-1": 9, "tester-1": 0},
	}); err != nil {
		t.Fatalf("Save sequence 2: %v", err)
	}

	report, err := store.RecoverAll()
	if err != nil {
		t.Fatalf("RecoverAll: %v", err)
	}
	if report.Mode != "checkpoint-light" || !report.Degraded {
		t.Fatalf("report mode/degraded = %#v", report)
	}
	if report.TaskCount != 1 || len(report.RecoveredTasks) != 1 {
		t.Fatalf("report tasks = %#v", report)
	}
	task := report.RecoveredTasks[0]
	if task.TaskID != "task-1" || task.Sequence != 2 || task.Status != "recovered" {
		t.Fatalf("task summary = %#v", task)
	}
	if task.PageTableRefs != 4 {
		t.Fatalf("page refs = %d", task.PageTableRefs)
	}
	if task.SchedulerVRuntime["coder-1"] != 9 {
		t.Fatalf("scheduler vruntime = %#v", task.SchedulerVRuntime)
	}
	if len(task.CompletedAgents) != 1 || task.CompletedAgents[0] != "planner-1" {
		t.Fatalf("completed = %#v", task.CompletedAgents)
	}
	if len(task.ReadyAgents) != 2 || task.ReadyAgents[0] != "coder-1" || task.ReadyAgents[1] != "tester-1" {
		t.Fatalf("ready = %#v", task.ReadyAgents)
	}
}
