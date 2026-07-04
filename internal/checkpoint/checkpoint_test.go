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
