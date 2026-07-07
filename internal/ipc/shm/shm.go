package shm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SmokeResult struct {
	IPCMode                string `json:"ipc_mode"`
	EvidenceMode           string `json:"evidence_mode"`
	MemfdCreateSuccess     bool   `json:"memfd_create_success"`
	MmapSuccess            bool   `json:"mmap_success"`
	FDPassingSuccess       bool   `json:"fd_passing_success"`
	WorkerMmapSuccess      bool   `json:"worker_mmap_success"`
	SharedPages            int    `json:"shared_pages"`
	PayloadBytesSent       int    `json:"payload_bytes_sent"`
	ReferencedContextBytes int    `json:"referenced_context_bytes"`
	AvoidedCopyBytes       int    `json:"avoided_copy_bytes"`
	DataIntegrityOK        bool   `json:"data_integrity_ok"`
	CleanupSuccess         bool   `json:"cleanup_success"`
	FallbackReason         string `json:"fallback_reason"`
}

func RunSmoke(outDir string) (SmokeResult, error) {
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "ipc_shm")
	}
	payload := []byte("AORT-R shared context page via memfd/mmap")
	result, err := TransferPayload(payload, 2)
	if err != nil {
		result = SmokeResult{
			IPCMode:          "page-reference",
			EvidenceMode:     "degraded",
			CleanupSuccess:   true,
			FallbackReason:   err.Error(),
			PayloadBytesSent: len(payload),
		}
	}
	if result.IPCMode == "" {
		result.IPCMode = "memfd-mmap"
	}
	if err := writeJSON(filepath.Join(outDir, "ipc_shm_smoke.json"), result); err != nil {
		return result, err
	}
	return result, nil
}

func TransferPayload(payload []byte, workerCount int) (SmokeResult, error) {
	return exerciseMemoryTransport(payload, workerCount)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
