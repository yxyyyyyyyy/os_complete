package avp

import "testing"

func TestAVPTransitionCreatedReadyRunningCompleted(t *testing.T) {
	mgr := NewManager()
	a := mgr.Create("task-1", "planner", nil)
	if a.State != StateCreated {
		t.Fatalf("initial state = %s", a.State)
	}
	for _, state := range []AgentState{StateReady, StateRunning, StateCompleted} {
		if err := mgr.Transition(a.AgentID, state); err != nil {
			t.Fatalf("transition to %s failed: %v", state, err)
		}
	}
	got, ok := mgr.Get(a.AgentID)
	if !ok || got.State != StateCompleted {
		t.Fatalf("final AVP = %#v ok=%v", got, ok)
	}
}

func TestAVPRejectsInvalidCompletedToRunning(t *testing.T) {
	mgr := NewManager()
	a := mgr.Create("task-1", "planner", nil)
	_ = mgr.Transition(a.AgentID, StateReady)
	_ = mgr.Transition(a.AgentID, StateRunning)
	_ = mgr.Transition(a.AgentID, StateCompleted)
	if err := mgr.Transition(a.AgentID, StateRunning); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}

func TestAVPCreateStoresDependenciesAndDefaults(t *testing.T) {
	mgr := NewManager()
	a := mgr.Create("task-1", "coder", []string{"planner"})
	if a.AgentID == "" {
		t.Fatalf("AgentID is empty")
	}
	if a.Weight != 100 {
		t.Fatalf("Weight = %d", a.Weight)
	}
	if len(a.Dependencies) != 1 || a.Dependencies[0] != "planner" {
		t.Fatalf("Dependencies = %#v", a.Dependencies)
	}
}
