package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"aort-r/internal/events"
)

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
