package codebasedag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// IntegrationPlan describes how validated coder patches are applied into one
// integration workspace before tester/reviewer gates run.
type IntegrationPlan struct {
	RunID          string              `json:"run_id"`
	OrderedPatches []PatchRecord       `json:"ordered_patches"`
	FileOwners     map[string]string   `json:"file_owners"`
	Conflicts      []IntegrationConflict `json:"conflicts,omitempty"`
}

type IntegrationConflict struct {
	Path   string   `json:"path"`
	Nodes  []string `json:"nodes"`
	Reason string   `json:"reason"`
}

func BuildIntegrationPlan(runID string, patches []PatchRecord) (IntegrationPlan, error) {
	if runID == "" {
		return IntegrationPlan{}, fmt.Errorf("run ID is required")
	}
	if len(patches) == 0 {
		return IntegrationPlan{}, fmt.Errorf("patches are required")
	}
	order := []string{"resource-coder", "context-coder", "evidence-coder"}
	byNode := make(map[string]PatchRecord, len(patches))
	for _, patch := range patches {
		if patch.NodeID == "" {
			return IntegrationPlan{}, fmt.Errorf("patch missing node ID")
		}
		if _, ok := byNode[patch.NodeID]; ok {
			return IntegrationPlan{}, fmt.Errorf("duplicate patch for node %q", patch.NodeID)
		}
		byNode[patch.NodeID] = patch
	}
	plan := IntegrationPlan{
		RunID:      runID,
		FileOwners: map[string]string{},
	}
	for _, node := range order {
		patch, ok := byNode[node]
		if !ok {
			// allow fixer-only supplemental patches after required coders
			continue
		}
		for _, path := range patch.ChangedFiles {
			clean, err := cleanPolicyPath(path)
			if err != nil {
				return IntegrationPlan{}, err
			}
			if owner, exists := plan.FileOwners[clean]; exists && owner != node {
				plan.Conflicts = append(plan.Conflicts, IntegrationConflict{
					Path:   clean,
					Nodes:  []string{owner, node},
					Reason: "same path claimed by multiple coder nodes",
				})
			} else {
				plan.FileOwners[clean] = node
			}
		}
		plan.OrderedPatches = append(plan.OrderedPatches, patch)
	}
	// append any non-coder patches (fixer-*) stably
	extra := make([]PatchRecord, 0)
	for _, patch := range patches {
		if patch.NodeID == "resource-coder" || patch.NodeID == "context-coder" || patch.NodeID == "evidence-coder" {
			continue
		}
		extra = append(extra, patch)
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i].NodeID < extra[j].NodeID })
	plan.OrderedPatches = append(plan.OrderedPatches, extra...)
	if len(plan.Conflicts) > 0 {
		return plan, fmt.Errorf("integration conflicts: %d", len(plan.Conflicts))
	}
	if len(plan.OrderedPatches) == 0 {
		return IntegrationPlan{}, fmt.Errorf("no ordered patches")
	}
	return plan, nil
}

func (p IntegrationPlan) ValidateAgainstContract(contract ReviewRemediationContract) error {
	if err := contract.Validate(); err != nil {
		return err
	}
	for _, patch := range p.OrderedPatches {
		if patch.NodeID == "resource-coder" || patch.NodeID == "context-coder" || patch.NodeID == "evidence-coder" {
			for _, path := range patch.ChangedFiles {
				if err := contract.AllowPath(patch.NodeID, path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ArtifactIndex is a deterministic map of relative artifact path -> sha256.
type ArtifactIndex struct {
	items map[string]string
}

func NewArtifactIndex() *ArtifactIndex {
	return &ArtifactIndex{items: map[string]string{}}
}

func (a *ArtifactIndex) Put(path string, data []byte) error {
	clean, err := cleanPolicyPath(path)
	if err != nil {
		return err
	}
	if _, exists := a.items[clean]; exists {
		return fmt.Errorf("artifact %q already recorded", clean)
	}
	sum := sha256.Sum256(data)
	a.items[clean] = hex.EncodeToString(sum[:])
	return nil
}

func (a *ArtifactIndex) PutHash(path, hash string) error {
	clean, err := cleanPolicyPath(path)
	if err != nil {
		return err
	}
	if len(hash) != 64 {
		return fmt.Errorf("artifact hash for %q must be sha256 hex", clean)
	}
	if _, exists := a.items[clean]; exists {
		return fmt.Errorf("artifact %q already recorded", clean)
	}
	a.items[clean] = hash
	return nil
}

func (a *ArtifactIndex) Map() map[string]string {
	out := make(map[string]string, len(a.items))
	for k, v := range a.items {
		out[k] = v
	}
	return out
}

func (a *ArtifactIndex) SortedPaths() []string {
	paths := make([]string, 0, len(a.items))
	for path := range a.items {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (a *ArtifactIndex) MerkleRoot() string {
	paths := a.SortedPaths()
	var b strings.Builder
	for _, path := range paths {
		b.WriteString(path)
		b.WriteByte(0)
		b.WriteString(a.items[path])
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// TraceEvent is a compact timeline event for dashboard/debug export.
type TraceEvent struct {
	Seq     int               `json:"seq"`
	RunID   string            `json:"run_id"`
	NodeID  string            `json:"node_id,omitempty"`
	Type    string            `json:"type"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type TraceBuilder struct {
	runID  string
	events []TraceEvent
}

func NewTraceBuilder(runID string) (*TraceBuilder, error) {
	if runID == "" {
		return nil, fmt.Errorf("run ID is required")
	}
	return &TraceBuilder{runID: runID}, nil
}

func (t *TraceBuilder) Add(nodeID, typ, message string, fields map[string]string) {
	copied := map[string]string{}
	for k, v := range fields {
		copied[k] = v
	}
	t.events = append(t.events, TraceEvent{
		Seq:     len(t.events) + 1,
		RunID:   t.runID,
		NodeID:  nodeID,
		Type:    typ,
		Message: message,
		Fields:  copied,
	})
}

func (t *TraceBuilder) Events() []TraceEvent {
	out := make([]TraceEvent, len(t.events))
	copy(out, t.events)
	return out
}

func (t *TraceBuilder) Filter(typ string) []TraceEvent {
	var out []TraceEvent
	for _, event := range t.events {
		if event.Type == typ {
			out = append(out, event)
		}
	}
	return out
}
