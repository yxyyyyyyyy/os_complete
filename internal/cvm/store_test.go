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

func TestPinnedPageIsNotEvicted(t *testing.T) {
	store := NewStore(nil)
	pinned, _ := store.CreatePage(KindSystem, "pinned shared prefix")
	cold, _ := store.CreatePage(KindTask, "cold evictable page")
	if err := store.PinPage(pinned.ID); err != nil {
		t.Fatalf("PinPage: %v", err)
	}
	if err := store.ReleasePage("", pinned.ID); err != nil {
		t.Fatalf("Release pinned: %v", err)
	}
	if err := store.ReleasePage("", cold.ID); err != nil {
		t.Fatalf("Release cold: %v", err)
	}
	evicted, _ := store.EvictColdPages(pinned.Bytes)
	if evicted != 1 {
		t.Fatalf("evicted = %d", evicted)
	}
	if _, ok := store.Page(pinned.ID); !ok {
		t.Fatalf("pinned page should remain")
	}
	if _, ok := store.Page(cold.ID); ok {
		t.Fatalf("cold page should be evicted")
	}
}

func TestCompressionAndMaterializeCorrectness(t *testing.T) {
	store := NewStore(nil)
	page, _ := store.CreatePage(KindProject, "repeat repeat repeat repeat repeat repeat")
	if err := store.MountPage("agent-1", page.ID); err != nil {
		t.Fatalf("MountPage: %v", err)
	}
	if err := store.ReleasePage("", page.ID); err != nil {
		t.Fatalf("Release initial ref: %v", err)
	}
	compressed, saved, err := store.CompressColdPages(2)
	if err != nil {
		t.Fatalf("CompressColdPages: %v", err)
	}
	if compressed != 1 || saved <= 0 {
		t.Fatalf("compressed=%d saved=%d", compressed, saved)
	}
	got, err := store.Materialize("agent-1")
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if got != "repeat repeat repeat repeat repeat repeat" {
		t.Fatalf("materialized = %q", got)
	}
	stats := store.Stats()
	if stats.CompressedPages != 1 || stats.CompressionSavedBytes <= 0 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestHotPageSurvivesLRUEviction(t *testing.T) {
	store := NewStore(nil)
	hot, _ := store.CreatePage(KindProject, "hot page content")
	cold, _ := store.CreatePage(KindTask, "cold page content")
	for i := 0; i < 5; i++ {
		if err := store.TouchPage(hot.ID); err != nil {
			t.Fatalf("TouchPage: %v", err)
		}
	}
	_ = store.ReleasePage("", hot.ID)
	_ = store.ReleasePage("", cold.ID)
	evicted, _ := store.EvictColdPages(hot.Bytes)
	if evicted != 1 {
		t.Fatalf("evicted = %d", evicted)
	}
	if _, ok := store.Page(hot.ID); !ok {
		t.Fatalf("hot page should remain")
	}
	if _, ok := store.Page(cold.ID); ok {
		t.Fatalf("cold page should be evicted")
	}
}
