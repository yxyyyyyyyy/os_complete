package codebasedag

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvidenceJournalCreatesExclusiveJSONLAndSanitizesEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	journal, err := NewEvidenceJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	event := EvidenceEvent{
		SchemaVersion: SchemaVersion,
		RunID:         "run",
		NodeID:        "planner",
		Type:          EventLLMCall,
		At:            time.Unix(1, 0).UTC(),
		Message:       "Authorization: Bearer sk-secret should not survive",
		Call: &CallRecord{
			CallID:           "call-1",
			Provider:         RequiredDeepSeekProvider,
			RequestedModel:   RequiredDeepSeekModel,
			ActualModel:      RequiredDeepSeekModel,
			EvidenceMode:     "real-api",
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
			OutputSHA256:     strings.Repeat("a", 64),
			Status:           "succeeded",
		},
	}
	if err := journal.Append(event); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewEvidenceJournal(path); err == nil {
		t.Fatal("journal path must be create-exclusive")
	}
	events := readJournalEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	if strings.Contains(events[0].Message, "sk-secret") || strings.Contains(events[0].Message, "Bearer") {
		t.Fatalf("secret-like text leaked: %#v", events[0])
	}
	if events[0].Call.CallID != "call-1" {
		t.Fatalf("call metadata missing: %#v", events[0])
	}
}

func TestEvidenceJournalAppendsNodeAndArtifactEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	journal, err := NewEvidenceJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendNode("run", "planner", EventNodeStart, "started", time.Unix(2, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendArtifact("run", "planner", "outputs/planner.json", strings.Repeat("b", 64), 128, time.Unix(3, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	events := readJournalEvents(t, path)
	if len(events) != 2 || events[0].Type != EventNodeStart || events[1].Type != EventArtifactHash {
		t.Fatalf("events = %#v", events)
	}
	if events[1].Artifact == nil || events[1].Artifact.Path != "outputs/planner.json" {
		t.Fatalf("artifact event = %#v", events[1])
	}
}

func readJournalEvents(t *testing.T, path string) []EvidenceEvent {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var events []EvidenceEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event EvidenceEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode journal line: %v", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}
