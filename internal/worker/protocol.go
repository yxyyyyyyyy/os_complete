package worker

const (
	MessageRegister      = "register"
	MessageHeartbeat     = "heartbeat"
	MessageReport        = "report"
	MessageSyscall       = "syscall"
	MessageSyscallResult = "syscall.result"
)

type Message struct {
	Type      string         `json:"type"`
	RequestID string         `json:"request_id,omitempty"`
	AgentID   string         `json:"agent_id"`
	Role      string         `json:"role,omitempty"`
	TaskID    string         `json:"task_id"`
	PID       int            `json:"pid,omitempty"`
	Status    string         `json:"status,omitempty"`
	Name      string         `json:"name,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type Response struct {
	Type      string         `json:"type"`
	RequestID string         `json:"request_id,omitempty"`
	AgentID   string         `json:"agent_id"`
	TaskID    string         `json:"task_id"`
	Status    string         `json:"status"`
	Error     string         `json:"error,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}
