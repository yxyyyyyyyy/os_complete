package supervisor

import (
	"sync"
	"time"

	"aort-r/internal/events"
)

const (
	FaultToolTimeout        = "TOOL_TIMEOUT"
	FaultToolExitNonZero    = "TOOL_EXIT_NONZERO"
	FaultAgentHeartbeatLost = "AGENT_HEARTBEAT_LOST"
	FaultPidsLimit          = "PIDS_LIMIT"
	FaultWorkspaceRollback  = "WORKSPACE_ROLLBACK"
	FaultRetryLimit         = "RETRY_LIMIT"

	StatusRecovered = "RECOVERED"
	StatusOpen      = "OPEN"
)

type EventSink interface {
	Publish(events.Event)
}

type Fault struct {
	Type           string         `json:"type"`
	TaskID         string         `json:"task_id"`
	AgentID        string         `json:"agent_id"`
	RecoveryAction string         `json:"recovery_action"`
	Details        map[string]any `json:"details,omitempty"`
}

type Record struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	TaskID         string         `json:"task_id"`
	AgentID        string         `json:"agent_id"`
	Status         string         `json:"status"`
	RecoveryAction string         `json:"recovery_action"`
	Details        map[string]any `json:"details,omitempty"`
	DetectedAt     int64          `json:"detected_at"`
}

type Manager struct {
	mu      sync.RWMutex
	sink    EventSink
	records []Record
}

func NewManager(sink EventSink) *Manager {
	return &Manager{sink: sink}
}

func (m *Manager) Record(fault Fault) Record {
	now := time.Now()
	status := StatusRecovered
	if fault.RecoveryAction == "" {
		status = StatusOpen
	}
	record := Record{
		ID:             now.Format("20060102150405.000000000"),
		Type:           fault.Type,
		TaskID:         fault.TaskID,
		AgentID:        fault.AgentID,
		Status:         status,
		RecoveryAction: fault.RecoveryAction,
		Details:        fault.Details,
		DetectedAt:     now.UnixMilli(),
	}
	m.mu.Lock()
	m.records = append(m.records, record)
	m.mu.Unlock()
	if m.sink != nil {
		m.sink.Publish(events.New("supervisor.detected", record.TaskID, record.AgentID, "supervisor", map[string]any{
			"fault_type":      record.Type,
			"status":          record.Status,
			"recovery_action": record.RecoveryAction,
			"details":         record.Details,
		}))
	}
	return record
}

func (m *Manager) Records() []Record {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Record, len(m.records))
	copy(out, m.records)
	return out
}
