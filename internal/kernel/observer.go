package kernel

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"aort-r/internal/events"
)

const (
	ModeDegradedProxy        = "degraded-proxy"
	ProbeSyscallGatewayProxy = "syscall-gateway-proxy"
)

type EventSink interface {
	Publish(events.Event)
}

type Config struct {
	Sink    EventSink
	BTFPath string
	BPFFS   string
}

type Status struct {
	Enabled      bool   `json:"enabled"`
	Mode         string `json:"mode"`
	Probe        string `json:"probe"`
	Reason       string `json:"reason"`
	BTFAvailable bool   `json:"btf_available"`
	BPFFSReady   bool   `json:"bpffs_ready"`
	EventCount   int    `json:"event_count"`
}

type ExecObservation struct {
	TaskID     string   `json:"task_id"`
	AgentID    string   `json:"agent_id"`
	PID        int      `json:"pid"`
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	CgroupPath string   `json:"cgroup_path,omitempty"`
	Workspace  string   `json:"workspace,omitempty"`
	Status     string   `json:"status"`
}

type ExecEvent struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Source     string   `json:"source"`
	TaskID     string   `json:"task_id"`
	AgentID    string   `json:"agent_id"`
	PID        int      `json:"pid"`
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	CgroupPath string   `json:"cgroup_path,omitempty"`
	Workspace  string   `json:"workspace,omitempty"`
	Status     string   `json:"status"`
	Mode       string   `json:"mode"`
	Probe      string   `json:"probe"`
	Timestamp  int64    `json:"timestamp"`
}

type Observer struct {
	mu     sync.RWMutex
	cfg    Config
	status Status
	events []ExecEvent
}

func NewObserver(cfg Config) *Observer {
	if cfg.BTFPath == "" {
		cfg.BTFPath = "/sys/kernel/btf/vmlinux"
	}
	if cfg.BPFFS == "" {
		cfg.BPFFS = "/sys/fs/bpf"
	}
	return &Observer{
		cfg: cfg,
		status: Status{
			Mode:  ModeDegradedProxy,
			Probe: ProbeSyscallGatewayProxy,
		},
	}
}

func (o *Observer) Start(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	btf := fileExists(o.cfg.BTFPath)
	bpffs := dirExists(o.cfg.BPFFS)
	reason := fmt.Sprintf("real eBPF sched_process_exec observer is not attached in this build; using %s", ProbeSyscallGatewayProxy)
	if runtime.GOOS != "linux" {
		reason = "real eBPF requires Linux; using syscall gateway proxy events"
	} else if !btf {
		reason = "kernel BTF vmlinux is unavailable; using syscall gateway proxy events"
	} else if !bpffs {
		reason = "bpffs is unavailable; using syscall gateway proxy events"
	}
	o.mu.Lock()
	o.status.Enabled = false
	o.status.Mode = ModeDegradedProxy
	o.status.Probe = ProbeSyscallGatewayProxy
	o.status.Reason = reason
	o.status.BTFAvailable = btf
	o.status.BPFFSReady = bpffs
	count := len(o.events)
	o.status.EventCount = count
	status := o.status
	o.mu.Unlock()
	o.publish(events.New("kernel.observer_disabled", "", "", "kernel", map[string]any{
		"enabled":       status.Enabled,
		"mode":          status.Mode,
		"probe":         status.Probe,
		"reason":        status.Reason,
		"btf_available": status.BTFAvailable,
		"bpffs_ready":   status.BPFFSReady,
	}))
	return nil
}

func (o *Observer) ObserveExec(observation ExecObservation) events.Event {
	now := time.Now().UnixMilli()
	args := append([]string(nil), observation.Args...)
	if args == nil {
		args = []string{}
	}
	record := ExecEvent{
		ID:         time.Now().Format("20060102150405.000000000"),
		Type:       "kernel.exec",
		Source:     "kernel",
		TaskID:     observation.TaskID,
		AgentID:    observation.AgentID,
		PID:        observation.PID,
		Command:    observation.Command,
		Args:       args,
		CgroupPath: observation.CgroupPath,
		Workspace:  observation.Workspace,
		Status:     observation.Status,
		Mode:       ModeDegradedProxy,
		Probe:      ProbeSyscallGatewayProxy,
		Timestamp:  now,
	}
	o.mu.Lock()
	o.events = append(o.events, record)
	o.status.EventCount = len(o.events)
	o.mu.Unlock()
	event := events.Event{
		ID:        record.ID,
		TaskID:    record.TaskID,
		AgentID:   record.AgentID,
		Type:      record.Type,
		Source:    record.Source,
		Timestamp: record.Timestamp,
		Payload: map[string]any{
			"pid":         record.PID,
			"command":     record.Command,
			"args":        record.Args,
			"cgroup_path": record.CgroupPath,
			"workspace":   record.Workspace,
			"status":      record.Status,
			"mode":        record.Mode,
			"probe":       record.Probe,
		},
	}
	o.publish(event)
	return event
}

func (o *Observer) Status() Status {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

func (o *Observer) Events() []ExecEvent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make([]ExecEvent, len(o.events))
	copy(out, o.events)
	return out
}

func (o *Observer) publish(event events.Event) {
	if o.cfg.Sink != nil {
		o.cfg.Sink.Publish(event)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
