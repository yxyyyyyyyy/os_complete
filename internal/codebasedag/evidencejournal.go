package codebasedag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type EvidenceEventType string

const (
	EventNodeStart    EvidenceEventType = "node.start"
	EventNodeEnd      EvidenceEventType = "node.end"
	EventLLMCall      EvidenceEventType = "llm.call"
	EventArtifactHash EvidenceEventType = "artifact.hash"
	EventPreflight    EvidenceEventType = "preflight"
	EventProcess      EvidenceEventType = "process"
)

type EvidenceEvent struct {
	SchemaVersion string            `json:"schema_version"`
	RunID         string            `json:"run_id"`
	NodeID        string            `json:"node_id,omitempty"`
	Type          EvidenceEventType `json:"type"`
	At            time.Time         `json:"at"`
	Message       string            `json:"message,omitempty"`
	Call          *CallRecord       `json:"call,omitempty"`
	Artifact      *ArtifactRecord   `json:"artifact,omitempty"`
	Process       *ProcessResult    `json:"process,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
}

type ArtifactRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type EvidenceJournal struct {
	path string
	file *os.File
	mu   sync.Mutex
}

func NewEvidenceJournal(path string) (*EvidenceJournal, error) {
	if path == "" {
		return nil, fmt.Errorf("journal path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	return &EvidenceJournal{path: path, file: file}, nil
}

func (j *EvidenceJournal) Path() string {
	if j == nil {
		return ""
	}
	return j.path
}

func (j *EvidenceJournal) Append(event EvidenceEvent) error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return fmt.Errorf("evidence journal is closed")
	}
	event = sanitizeEvidenceEvent(event)
	if event.SchemaVersion == "" {
		event.SchemaVersion = SchemaVersion
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	} else {
		event.At = event.At.UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := j.file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (j *EvidenceJournal) AppendNode(runID, nodeID string, typ EvidenceEventType, message string, at time.Time) error {
	return j.Append(EvidenceEvent{RunID: runID, NodeID: nodeID, Type: typ, At: at, Message: message})
}

func (j *EvidenceJournal) AppendCall(runID, nodeID string, call CallRecord, at time.Time) error {
	call.Error = sanitizeEvidenceError(call.Error)
	return j.Append(EvidenceEvent{RunID: runID, NodeID: nodeID, Type: EventLLMCall, At: at, Call: &call})
}

func (j *EvidenceJournal) AppendArtifact(runID, nodeID, path, hash string, bytes int64, at time.Time) error {
	if _, err := cleanPolicyPath(path); err != nil {
		return err
	}
	return j.Append(EvidenceEvent{
		RunID:  runID,
		NodeID: nodeID,
		Type:   EventArtifactHash,
		At:     at,
		Artifact: &ArtifactRecord{
			Path:   path,
			SHA256: hash,
			Bytes:  bytes,
		},
	})
}

func (j *EvidenceJournal) Close() error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	return err
}

func sanitizeEvidenceEvent(event EvidenceEvent) EvidenceEvent {
	event.Message = redactSecretLikeText(event.Message)
	if event.Fields != nil {
		copied := make(map[string]string, len(event.Fields))
		for key, value := range event.Fields {
			copied[key] = redactSecretLikeText(value)
		}
		event.Fields = copied
	}
	return event
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+\S+`),
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`),
}

func redactSecretLikeText(text string) string {
	if text == "" {
		return ""
	}
	out := text
	for _, pattern := range secretPatterns {
		out = pattern.ReplaceAllString(out, "[REDACTED]")
	}
	out = strings.ReplaceAll(out, "DEEPSEEK_API_KEY=", "DEEPSEEK_API_KEY=[REDACTED]")
	return out
}
