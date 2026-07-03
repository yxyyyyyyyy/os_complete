package avp

type AgentState string

const (
	StateCreated     AgentState = "CREATED"
	StateReady       AgentState = "READY"
	StateRunning     AgentState = "RUNNING"
	StateWaitingLLM  AgentState = "WAITING_LLM"
	StateWaitingTool AgentState = "WAITING_TOOL"
	StateWaitingIPC  AgentState = "WAITING_IPC"
	StateSuspended   AgentState = "SUSPENDED"
	StateCompleted   AgentState = "COMPLETED"
	StateFailed      AgentState = "FAILED"
	StateKilled      AgentState = "KILLED"
)

type AVP struct {
	AgentID      string     `json:"agent_id"`
	TaskID       string     `json:"task_id"`
	Role         string     `json:"role"`
	State        AgentState `json:"state"`
	Weight       int        `json:"weight"`
	VRuntime     uint64     `json:"vruntime"`
	Dependencies []string   `json:"dependencies"`
	PageTable    []string   `json:"page_table"`
	PID          int        `json:"pid"`
	CgroupPath   string     `json:"cgroup_path"`
	RetryCount   int        `json:"retry_count"`
}
