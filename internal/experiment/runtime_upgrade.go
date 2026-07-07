package experiment

import (
	"path/filepath"
	"strings"

	"aort-r/internal/cvm"
	"aort-r/internal/evidence"
	"aort-r/internal/ipc/shm"
	"aort-r/internal/observer/ebpf"
	"aort-r/internal/replay"
)

type CVMMemorySmokeResult struct {
	Experiment            string        `json:"experiment"`
	EvidenceMode          evidence.Mode `json:"evidence_mode"`
	TotalPages            int           `json:"total_pages"`
	DedupPages            int           `json:"dedup_pages"`
	HotPages              int           `json:"hot_pages"`
	ColdPages             int           `json:"cold_pages"`
	CompressedPages       int           `json:"compressed_pages"`
	EvictedPages          int           `json:"evicted_pages"`
	PinnedPages           int           `json:"pinned_pages"`
	RefCountedPages       int           `json:"ref_counted_pages"`
	MaterializeSuccess    bool          `json:"materialize_success"`
	MemorySavedBytes      int64         `json:"memory_saved_bytes"`
	CompressionSavedBytes int64         `json:"compression_saved_bytes"`
	DedupSavedBytes       int64         `json:"dedup_saved_bytes"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
}

func RunEBPFSmoke(outDir string) (ebpf.SmokeResult, error) {
	return ebpf.RunSmoke(outDir)
}

func RunIPCShmSmoke(outDir string) (shm.SmokeResult, error) {
	return shm.RunSmoke(outDir)
}

func RunCVMMemorySmoke(outDir string) (CVMMemorySmokeResult, error) {
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "cvm_memory")
	}
	store := cvm.NewStore(nil)
	system, _ := store.CreatePage(cvm.KindSystem, strings.Repeat("system prefix ", 32))
	_ = store.PinPage(system.ID)
	shared, _ := store.CreatePage(cvm.KindProject, strings.Repeat("shared project prefix ", 64))
	duplicate, _ := store.CreatePage(cvm.KindProject, strings.Repeat("shared project prefix ", 64))
	hot, _ := store.CreatePage(cvm.KindTask, strings.Repeat("hot task context ", 32))
	cold, _ := store.CreatePage(cvm.KindSummary, strings.Repeat("cold summary context ", 64))
	agentID := "agent-cvm-smoke"
	_ = store.MountPage(agentID, system.ID)
	_ = store.MountPage(agentID, shared.ID)
	_ = store.MountPage(agentID, hot.ID)
	for i := 0; i < 5; i++ {
		_ = store.TouchPage(hot.ID)
	}
	_ = store.ReleasePage("", cold.ID)
	_, _, _ = store.CompressColdPages(2)
	_, _ = store.EvictColdPages(1)
	materialized, err := store.Materialize(agentID)
	materializeSuccess := err == nil && strings.Contains(materialized, "shared project prefix") && duplicate.ID == shared.ID
	stats := store.Stats()
	totalBeforeDedup := 5
	cacheHitRate := 0.0
	if totalBeforeDedup > 0 {
		cacheHitRate = float64(maxInt(0, totalBeforeDedup-stats.TotalPages)) / float64(totalBeforeDedup)
	}
	result := CVMMemorySmokeResult{
		Experiment:            "cvm_memory_smoke",
		EvidenceMode:          evidence.ModeRealPartial,
		TotalPages:            stats.TotalPages,
		DedupPages:            maxInt(0, totalBeforeDedup-stats.TotalPages),
		HotPages:              stats.HotPages,
		ColdPages:             stats.ColdPages,
		CompressedPages:       stats.CompressedPages,
		EvictedPages:          stats.EvictedPages,
		PinnedPages:           stats.PinnedPages,
		RefCountedPages:       stats.RefCountedPages,
		MaterializeSuccess:    materializeSuccess,
		MemorySavedBytes:      stats.MemorySavedBytes,
		CompressionSavedBytes: stats.CompressionSavedBytes,
		DedupSavedBytes:       stats.DedupSavedBytes,
		CacheHitRate:          cacheHitRate,
	}
	return result, WriteJSON(filepath.Join(outDir, "cvm_memory_smoke.json"), result)
}

func RunReplay(tracePath, outDir string) (replay.Result, error) {
	return replay.Run(tracePath, outDir)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
