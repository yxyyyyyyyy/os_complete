package codebasedag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProcessEvidenceBundle is an immutable stamped archive of intermediate
// artifacts produced while executing Open World phases.
type ProcessEvidenceBundle struct {
	SchemaVersion string                 `json:"schema_version"`
	Phase         string                 `json:"phase"`
	CapturedAt    time.Time              `json:"captured_at_utc"`
	Host          string                 `json:"host,omitempty"`
	RemoteDir     string                 `json:"remote_dir,omitempty"`
	Notes         []string               `json:"notes,omitempty"`
	Files         []ProcessEvidenceFile  `json:"files"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

type ProcessEvidenceFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

const ProcessEvidenceSchema = "aort.process_evidence.v1"

func NewProcessEvidenceBundle(phase string) ProcessEvidenceBundle {
	return ProcessEvidenceBundle{
		SchemaVersion: ProcessEvidenceSchema,
		Phase:         phase,
		CapturedAt:    time.Now().UTC(),
		Files:         []ProcessEvidenceFile{},
		Extra:         map[string]interface{}{},
	}
}

func (b *ProcessEvidenceBundle) AddFile(path string, data []byte) error {
	clean, err := cleanPolicyPath(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	b.Files = append(b.Files, ProcessEvidenceFile{
		Path:   clean,
		Bytes:  int64(len(data)),
		SHA256: hex.EncodeToString(sum[:]),
	})
	return nil
}

func (b *ProcessEvidenceBundle) AddPath(root, rel string) error {
	clean, err := cleanPolicyPath(rel)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return err
	}
	return b.AddFile(clean, data)
}

func (b ProcessEvidenceBundle) Validate() error {
	if b.SchemaVersion != ProcessEvidenceSchema {
		return fmt.Errorf("process evidence schema %q invalid", b.SchemaVersion)
	}
	if strings.TrimSpace(b.Phase) == "" {
		return fmt.Errorf("phase is required")
	}
	if b.CapturedAt.IsZero() {
		return fmt.Errorf("captured_at is required")
	}
	if len(b.Files) == 0 {
		return fmt.Errorf("at least one evidence file is required")
	}
	seen := make(map[string]struct{}, len(b.Files))
	for _, file := range b.Files {
		if _, err := cleanPolicyPath(file.Path); err != nil {
			return fmt.Errorf("invalid evidence path %q: %w", file.Path, err)
		}
		if file.Bytes < 0 {
			return fmt.Errorf("negative bytes for %q", file.Path)
		}
		if len(file.SHA256) != 64 {
			return fmt.Errorf("invalid sha256 for %q", file.Path)
		}
		if _, ok := seen[file.Path]; ok {
			return fmt.Errorf("duplicate evidence path %q", file.Path)
		}
		seen[file.Path] = struct{}{}
	}
	return nil
}

func WriteProcessEvidenceBundle(dir string, bundle ProcessEvidenceBundle) error {
	if err := bundle.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	manifestPath := filepath.Join(dir, "EVIDENCE_MANIFEST.json")
	if _, err := os.Lstat(manifestPath); err == nil {
		return fmt.Errorf("refusing to overwrite %s", manifestPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	sort.Slice(bundle.Files, func(i, j int) bool { return bundle.Files[i].Path < bundle.Files[j].Path })
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(manifestPath, data, 0o644)
}

func LoadProcessEvidenceBundle(dir string) (ProcessEvidenceBundle, error) {
	data, err := os.ReadFile(filepath.Join(dir, "EVIDENCE_MANIFEST.json"))
	if err != nil {
		return ProcessEvidenceBundle{}, err
	}
	var bundle ProcessEvidenceBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return ProcessEvidenceBundle{}, err
	}
	if err := bundle.Validate(); err != nil {
		return ProcessEvidenceBundle{}, err
	}
	return bundle, nil
}

func IndexProcessEvidenceRoots(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func VerifyProcessEvidenceBundleFiles(dir string, bundle ProcessEvidenceBundle) error {
	for _, file := range bundle.Files {
		path := filepath.Join(dir, filepath.FromSlash(file.Path))
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if int64(len(data)) != file.Bytes {
			return fmt.Errorf("%s size %d want %d", file.Path, len(data), file.Bytes)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			return fmt.Errorf("%s hash mismatch", file.Path)
		}
	}
	return nil
}
