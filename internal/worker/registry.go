package worker

import (
	"sync"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/events"
)

type EventSink interface {
	Publish(events.Event)
}

type Registry struct {
	mu         sync.RWMutex
	agents     map[string]avp.AVP
	sink       EventSink
	onRegister func(avp.AVP)
}

func NewRegistry(sink EventSink) *Registry {
	return &Registry{
		agents: make(map[string]avp.AVP),
		sink:   sink,
	}
}

func (r *Registry) CreateAgent(agentID, role, taskID string) avp.AVP {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UnixMilli()
	agent := avp.AVP{
		AgentID:   agentID,
		TaskID:    taskID,
		Role:      role,
		State:     avp.StateCreated,
		Priority:  100,
		Weight:    100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.agents[agentID] = agent
	r.publishLocked("agent.created", agent, map[string]any{"role": role})
	return agent
}

func (r *Registry) HandleMessage(message Message) {
	switch message.Type {
	case MessageRegister:
		r.register(message)
	case MessageHeartbeat:
		r.heartbeat(message)
	case MessageReport:
		r.report(message)
	}
}

func (r *Registry) Get(agentID string) (avp.AVP, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[agentID]
	return agent, ok
}

func (r *Registry) List() []avp.AVP {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]avp.AVP, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (r *Registry) ListByTask(taskID string) []avp.AVP {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]avp.AVP, 0)
	for _, agent := range r.agents {
		if agent.TaskID == taskID {
			agents = append(agents, agent)
		}
	}
	return agents
}

func (r *Registry) SetOnRegister(fn func(avp.AVP)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onRegister = fn
}

func (r *Registry) SetCapsule(agentID, cgroupPath, mode string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.agents[agentID]
	agent.CgroupPath = cgroupPath
	agent.CapsuleMode = mode
	agent.UpdatedAt = time.Now().UnixMilli()
	r.agents[agentID] = agent
	r.publishLocked("agent.capsule_attached", agent, map[string]any{"cgroup_path": cgroupPath, "mode": mode})
}

func (r *Registry) SetState(agentID string, state avp.AgentState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.agents[agentID]
	agent.State = state
	agent.UpdatedAt = time.Now().UnixMilli()
	r.agents[agentID] = agent
	r.publishLocked("agent.state_changed", agent, map[string]any{"state": string(state)})
}

func (r *Registry) RestoreAgent(agent avp.AVP) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UnixMilli()
	if agent.CreatedAt == 0 {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt == 0 {
		agent.UpdatedAt = now
	}
	if agent.Priority == 0 {
		agent.Priority = 100
	}
	if agent.Weight == 0 {
		agent.Weight = 100
	}
	r.agents[agent.AgentID] = agent
	r.publishLocked("agent.recovered", agent, map[string]any{
		"state":    string(agent.State),
		"vruntime": agent.VRuntime,
	})
}

func (r *Registry) MarkHeartbeatLost(now time.Time, timeout time.Duration) []avp.AVP {
	r.mu.Lock()
	defer r.mu.Unlock()
	failed := make([]avp.AVP, 0)
	threshold := now.Add(-timeout).UnixMilli()
	for id, agent := range r.agents {
		if agent.PID == 0 || agent.LastSeen == 0 || agent.State == avp.StateFailed || agent.State == avp.StateKilled {
			continue
		}
		if agent.LastSeen < threshold {
			agent.State = avp.StateFailed
			agent.UpdatedAt = now.UnixMilli()
			r.agents[id] = agent
			failed = append(failed, agent)
			r.publishLocked("agent.heartbeat_lost", agent, map[string]any{"last_seen": agent.LastSeen})
			r.publishLocked("agent.state_changed", agent, map[string]any{"state": string(agent.State)})
		}
	}
	return failed
}

func (r *Registry) MarkLastSeenForTest(agentID string, lastSeen time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.agents[agentID]
	agent.LastSeen = lastSeen.UnixMilli()
	r.agents[agentID] = agent
}

func (r *Registry) register(message Message) {
	r.mu.Lock()
	agent := r.agentForMessageLocked(message)
	now := time.Now().UnixMilli()
	agent.PID = message.PID
	agent.State = avp.StateRunning
	agent.LastSeen = now
	agent.UpdatedAt = now
	r.agents[agent.AgentID] = agent
	r.publishLocked("agent.registered", agent, map[string]any{"pid": message.PID})
	r.publishLocked("agent.state_changed", agent, map[string]any{"state": string(agent.State)})
	onRegister := r.onRegister
	r.mu.Unlock()
	if onRegister != nil {
		onRegister(agent)
	}
}

func (r *Registry) heartbeat(message Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.agentForMessageLocked(message)
	now := time.Now().UnixMilli()
	agent.LastSeen = now
	agent.UpdatedAt = now
	if message.PID != 0 {
		agent.PID = message.PID
	}
	r.agents[agent.AgentID] = agent
}

func (r *Registry) report(message Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent := r.agentForMessageLocked(message)
	now := time.Now().UnixMilli()
	if message.Status != "" {
		agent.State = avp.AgentState(message.Status)
	}
	agent.UpdatedAt = now
	agent.LastSeen = now
	r.agents[agent.AgentID] = agent
	r.publishLocked("agent.report", agent, map[string]any{"status": message.Status})
	r.publishLocked("agent.state_changed", agent, map[string]any{"state": string(agent.State)})
}

func (r *Registry) agentForMessageLocked(message Message) avp.AVP {
	agent, ok := r.agents[message.AgentID]
	if ok {
		return agent
	}
	now := time.Now().UnixMilli()
	return avp.AVP{
		AgentID:   message.AgentID,
		TaskID:    message.TaskID,
		Role:      message.Role,
		State:     avp.StateCreated,
		Priority:  100,
		Weight:    100,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (r *Registry) publishLocked(eventType string, agent avp.AVP, payload map[string]any) {
	if r.sink == nil {
		return
	}
	r.sink.Publish(events.New(eventType, agent.TaskID, agent.AgentID, "runtime", payload))
}
