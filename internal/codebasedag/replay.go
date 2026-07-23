package codebasedag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// ReplayGuard ensures a failed-once worker replay reuses checkpointed model
// output instead of issuing a duplicate DeepSeek completion call.
type ReplayGuard struct {
	mu       sync.Mutex
	outputs  map[string]ReplayRecord
	failOnce map[string]bool
	replays  map[string]int
}

type ReplayRecord struct {
	NodeID       string
	CallID       string
	OutputSHA256 string
	Payload      []byte
}

func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{
		outputs:  make(map[string]ReplayRecord),
		failOnce: make(map[string]bool),
		replays:  make(map[string]int),
	}
}

func (g *ReplayGuard) Remember(nodeID, callID string, payload []byte) (ReplayRecord, error) {
	if nodeID == "" {
		return ReplayRecord{}, fmt.Errorf("node ID is required")
	}
	if callID == "" {
		return ReplayRecord{}, fmt.Errorf("call ID is required")
	}
	if len(payload) == 0 {
		return ReplayRecord{}, fmt.Errorf("payload is required")
	}
	sum := sha256.Sum256(payload)
	rec := ReplayRecord{
		NodeID:       nodeID,
		CallID:       callID,
		OutputSHA256: hex.EncodeToString(sum[:]),
		Payload:      append([]byte(nil), payload...),
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if existing, ok := g.outputs[nodeID]; ok {
		if existing.OutputSHA256 != rec.OutputSHA256 || existing.CallID != callID {
			return ReplayRecord{}, fmt.Errorf("node %q already has conflicting checkpointed output", nodeID)
		}
		return existing, nil
	}
	g.outputs[nodeID] = rec
	return rec, nil
}

func (g *ReplayGuard) MarkFailOnce(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.failOnce[nodeID] {
		return fmt.Errorf("node %q already consumed fail-once path", nodeID)
	}
	if _, ok := g.outputs[nodeID]; !ok {
		return fmt.Errorf("node %q has no checkpointed output for fail-once", nodeID)
	}
	g.failOnce[nodeID] = true
	return nil
}

func (g *ReplayGuard) Replay(nodeID string) (ReplayRecord, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	rec, ok := g.outputs[nodeID]
	if !ok {
		return ReplayRecord{}, fmt.Errorf("node %q has no checkpointed output", nodeID)
	}
	if !g.failOnce[nodeID] {
		return ReplayRecord{}, fmt.Errorf("node %q cannot replay without fail-once mark", nodeID)
	}
	g.replays[nodeID]++
	out := rec
	out.Payload = append([]byte(nil), rec.Payload...)
	return out, nil
}

func (g *ReplayGuard) VerifyUnchanged(nodeID string, payload []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	rec, ok := g.outputs[nodeID]
	if !ok {
		return fmt.Errorf("node %q has no checkpointed output", nodeID)
	}
	sum := sha256.Sum256(payload)
	got := hex.EncodeToString(sum[:])
	if got != rec.OutputSHA256 {
		return fmt.Errorf("replay output hash changed for %q: got %s want %s", nodeID, got, rec.OutputSHA256)
	}
	return nil
}

type ReplaySnapshot struct {
	CheckpointedNodes int            `json:"checkpointed_nodes"`
	FailOnceNodes     int            `json:"fail_once_nodes"`
	ReplayCounts      map[string]int `json:"replay_counts"`
}

func (g *ReplayGuard) Snapshot() ReplaySnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	counts := make(map[string]int, len(g.replays))
	for k, v := range g.replays {
		counts[k] = v
	}
	return ReplaySnapshot{
		CheckpointedNodes: len(g.outputs),
		FailOnceNodes:     len(g.failOnce),
		ReplayCounts:      counts,
	}
}
