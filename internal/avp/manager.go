package avp

import (
	"fmt"
	"sync"
)

type Manager struct {
	mu      sync.RWMutex
	nextID  int
	agents map[string]AVP
}

func NewManager() *Manager {
	return &Manager{agents: make(map[string]AVP)}
}

func (m *Manager) Create(taskID, role string, deps []string) AVP {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	agentID := fmt.Sprintf("%s-%d", role, m.nextID)
	agent := AVP{
		AgentID:      agentID,
		TaskID:       taskID,
		Role:         role,
		State:        StateCreated,
		Weight:       100,
		Dependencies: append([]string(nil), deps...),
	}
	m.agents[agentID] = agent
	return agent
}

func (m *Manager) Get(agentID string) (AVP, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.agents[agentID]
	return agent, ok
}

func (m *Manager) List() []AVP {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agents := make([]AVP, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (m *Manager) Transition(agentID string, next AgentState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("unknown agent %q", agentID)
	}
	if !validTransition(agent.State, next) {
		return fmt.Errorf("invalid transition %s -> %s", agent.State, next)
	}
	agent.State = next
	m.agents[agentID] = agent
	return nil
}

func validTransition(from, to AgentState) bool {
	valid := map[AgentState][]AgentState{
		StateCreated:     {StateReady},
		StateReady:       {StateRunning},
		StateRunning:     {StateWaitingLLM, StateWaitingTool, StateWaitingIPC, StateSuspended, StateCompleted, StateFailed},
		StateWaitingLLM:  {StateReady},
		StateWaitingTool: {StateReady},
		StateWaitingIPC:  {StateReady},
		StateSuspended:   {StateReady},
		StateFailed:      {StateReady, StateKilled},
	}
	for _, candidate := range valid[from] {
		if candidate == to {
			return true
		}
	}
	return false
}
