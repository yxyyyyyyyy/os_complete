package worker

const (
	MessageRegister  = "register"
	MessageHeartbeat = "heartbeat"
	MessageReport    = "report"
)

type Message struct {
	Type    string         `json:"type"`
	AgentID string         `json:"agent_id"`
	Role    string         `json:"role,omitempty"`
	TaskID  string         `json:"task_id"`
	PID     int            `json:"pid,omitempty"`
	Status  string         `json:"status,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}
