package cvm

import "testing"

func TestCreatePageDeduplicatesByContent(t *testing.T) {
	store := NewStore(nil)
	first, err := store.CreatePage(KindProject, "shared project")
	if err != nil {
		t.Fatalf("CreatePage first: %v", err)
	}
	second, err := store.CreatePage(KindProject, "shared project")
	if err != nil {
		t.Fatalf("CreatePage second: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids differ: %s %s", first.ID, second.ID)
	}
	page, ok := store.Page(first.ID)
	if !ok {
		t.Fatalf("page not found")
	}
	if page.RefCount != 2 {
		t.Fatalf("ref_count = %d", page.RefCount)
	}
	stats := store.Stats()
	if stats.TotalPages != 1 || stats.SavedBytes == 0 || stats.SavedTokens == 0 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestMountWriteDeltaAndMaterialize(t *testing.T) {
	store := NewStore(nil)
	system, _ := store.CreatePage(KindSystem, "system\n")
	task, _ := store.CreatePage(KindTask, "task\n")
	if err := store.MountPage("agent-1", system.ID); err != nil {
		t.Fatalf("Mount system: %v", err)
	}
	if err := store.MountPage("agent-1", task.ID); err != nil {
		t.Fatalf("Mount task: %v", err)
	}
	delta, err := store.WriteDelta("agent-1", "delta\n")
	if err != nil {
		t.Fatalf("WriteDelta: %v", err)
	}
	if delta.Kind != KindDelta {
		t.Fatalf("delta kind = %s", delta.Kind)
	}
	got, err := store.Materialize("agent-1")
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if got != "system\ntask\ndelta\n" {
		t.Fatalf("materialized = %q", got)
	}
	table := store.PageTable("agent-1")
	if len(table.PageIDs) != 3 {
		t.Fatalf("page table = %#v", table)
	}
}

func TestMountingSharedPageCountsAvoidedCopies(t *testing.T) {
	store := NewStore(nil)
	page, err := store.CreatePage(KindProject, "shared context page")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if err := store.MountPage("agent-1", page.ID); err != nil {
		t.Fatalf("Mount agent-1: %v", err)
	}
	if stats := store.Stats(); stats.SavedBytes != 0 || stats.SavedTokens != 0 {
		t.Fatalf("first mount should not count as avoided copy: %#v", stats)
	}
	if err := store.MountPage("agent-2", page.ID); err != nil {
		t.Fatalf("Mount agent-2: %v", err)
	}
	stats := store.Stats()
	if stats.SavedBytes != int64(page.Bytes) || stats.SavedTokens != int64(page.TokenCount) {
		t.Fatalf("stats = %#v, page = %#v", stats, page)
	}
}

func TestMountRejectsUnknownPage(t *testing.T) {
	store := NewStore(nil)
	if err := store.MountPage("agent-1", "missing"); err == nil {
		t.Fatalf("expected unknown page error")
	}
}
