package codebasedag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type PatchApplicationPlan struct {
	Ordered       []PatchRecord     `json:"ordered"`
	Collisions    []PatchCollision  `json:"collisions,omitempty"`
	RequiresFixer bool              `json:"requires_fixer"`
	FileOwners    map[string]string `json:"file_owners"`
}

type PatchCollision struct {
	Path  string   `json:"path"`
	Nodes []string `json:"nodes"`
}

type PatchBundleManifest struct {
	SchemaVersion string        `json:"schema_version"`
	BundleSHA256  string        `json:"bundle_sha256"`
	Patches       []PatchRecord `json:"patches"`
	ChangedFiles  []string      `json:"changed_files"`
}

func PlanPatchApplication(records []PatchRecord) PatchApplicationPlan {
	ordered := append([]PatchRecord(nil), records...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].NodeID == ordered[j].NodeID {
			return ordered[i].SHA256 < ordered[j].SHA256
		}
		return ordered[i].NodeID < ordered[j].NodeID
	})
	owners := make(map[string][]string)
	for _, record := range ordered {
		for _, path := range record.ChangedFiles {
			owners[path] = append(owners[path], record.NodeID)
		}
	}
	fileOwners := make(map[string]string, len(owners))
	var collisions []PatchCollision
	for path, nodes := range owners {
		sort.Strings(nodes)
		fileOwners[path] = nodes[0]
		if len(nodes) > 1 {
			collisions = append(collisions, PatchCollision{Path: path, Nodes: append([]string(nil), nodes...)})
		}
	}
	sort.Slice(collisions, func(i, j int) bool { return collisions[i].Path < collisions[j].Path })
	return PatchApplicationPlan{
		Ordered:       ordered,
		Collisions:    collisions,
		RequiresFixer: len(collisions) > 0,
		FileOwners:    fileOwners,
	}
}

func NewPatchBundleManifest(records []PatchRecord) (PatchBundleManifest, error) {
	ordered := PlanPatchApplication(records).Ordered
	changedSet := make(map[string]struct{})
	hash := sha256.New()
	for _, record := range ordered {
		if record.NodeID == "" {
			return PatchBundleManifest{}, fmt.Errorf("patch node ID is required")
		}
		if len(record.SHA256) != 64 {
			return PatchBundleManifest{}, fmt.Errorf("patch %q hash must be sha256 hex", record.NodeID)
		}
		hash.Write([]byte(record.NodeID))
		hash.Write([]byte{0})
		hash.Write([]byte(record.SHA256))
		hash.Write([]byte{0})
		for _, path := range record.ChangedFiles {
			clean, err := cleanPolicyPath(path)
			if err != nil {
				return PatchBundleManifest{}, err
			}
			changedSet[clean] = struct{}{}
			hash.Write([]byte(clean))
			hash.Write([]byte{0})
		}
	}
	changed := make([]string, 0, len(changedSet))
	for path := range changedSet {
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return PatchBundleManifest{
		SchemaVersion: SchemaVersion,
		BundleSHA256:  hex.EncodeToString(hash.Sum(nil)),
		Patches:       ordered,
		ChangedFiles:  changed,
	}, nil
}

func FormatPatchCollisions(collisions []PatchCollision) string {
	if len(collisions) == 0 {
		return ""
	}
	var parts []string
	for _, collision := range collisions {
		parts = append(parts, collision.Path+":"+strings.Join(collision.Nodes, ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}
