package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"aort-r/internal/events"
)

type TraceEvent struct {
	EventID   string         `json:"event_id"`
	Timestamp string         `json:"timestamp"`
	Type      string         `json:"type"`
	AgentID   string         `json:"agent_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	Payload   map[string]any `json:"payload"`
}

type Recorder struct {
	mu  sync.Mutex
	dir string
}

func NewRecorder(dir string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Recorder{dir: dir}, nil
}

func (r *Recorder) Append(event events.Event) error {
	if event.TaskID == "" {
		return fmt.Errorf("trace event missing task id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	path := filepath.Join(r.dir, event.TaskID+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func WriteTrace(path string, events []TraceEvent) error {
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
	return encoder.Encode(events)
}

func ReadTrace(path string) ([]TraceEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("trace not found: %s", path)
		}
		return nil, err
	}
	var events []TraceEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	for i := range events {
		if events[i].Payload == nil {
			events[i].Payload = map[string]any{}
		}
	}
	return events, nil
}
