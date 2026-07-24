package chunk034

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// RecoveryState021 represents the phase of resource recovery.
type RecoveryState021 int

const (
	RecoveryStateUninitialized021 RecoveryState021 = iota
	RecoveryStateKilling021
	RecoveryStateDestroying021
	RecoveryStateRecovering021
	RecoveryStateCleaning021
	RecoveryStateComplete021
	RecoveryStateFailed021
)

// ActionType021 categorizes actions taken during recovery.
type ActionType021 int

const (
	ActionTypeKill021   ActionType021 = iota
	ActionTypeDestroy021
	ActionTypeRecover021
	ActionTypeCleanup021
	ActionTypeVerify021
)

// ResourceAgentRecoveryAction021 is a single event on the timeline.
type ResourceAgentRecoveryAction021 struct {
	Timestamp  time.Time
	Action     ActionType021
	ResourceID string
	State      RecoveryState021
	Detail     string
}

// ResourceAgentRecoveryTimeline021 stores timestamped recovery events.
type ResourceAgentRecoveryTimeline021 struct {
	Actions []ResourceAgentRecoveryAction021
}

// CleanupVerificationResult021 holds the outcome of cleanup checks.
type CleanupVerificationResult021 struct {
	AllDestroyed      bool
	AllRecovered      bool
	PingCount         int
	LastAliveTime     time.Time
	FailedResources   []string
}

// ResourceAgentCleanupVerification021 defines parameters for verification.
type ResourceAgentCleanupVerification021 struct {
	CheckInterval time.Duration
	Timeout       time.Duration
	ExpectedState RecoveryState021
	Resources     []string
	AllowPartial  bool
}

// ResourceAgentRecoveryPlan021 defines a planned set of recovery steps.
type ResourceAgentRecoveryPlan021 struct {
	ResourceID string
	Steps      []ResourceAgentRecoveryAction021
	Deadline   time.Time
	Priority   int
}

// RecoveryError021 is a structured error for recovery operations.
type RecoveryError021 struct {
	ResourceID string
	Phase      RecoveryState021
	Action     ActionType021
	Message    string
}

func (e *RecoveryError021) Error() string {
	return fmt.Sprintf("recovery error on %s at phase %d action %d: %s",
		e.ResourceID, e.Phase, e.Action, e.Message)
}

// String returns a human-readable representation of ActionType021.
func (a ActionType021) String() string {
	switch a {
	case ActionTypeKill021:
		return "kill"
	case ActionTypeDestroy021:
		return "destroy"
	case ActionTypeRecover021:
		return "recover"
	case ActionTypeCleanup021:
		return "cleanup"
	case ActionTypeVerify021:
		return "verify"
	default:
		return "unknown"
	}
}

// String returns a human-readable representation of RecoveryState021.
func (s RecoveryState021) String() string {
	switch s {
	case RecoveryStateUninitialized021:
		return "uninitialized"
	case RecoveryStateKilling021:
		return "killing"
	case RecoveryStateDestroying021:
		return "destroying"
	case RecoveryStateRecovering021:
		return "recovering"
	case RecoveryStateCleaning021:
		return "cleaning"
	case RecoveryStateComplete021:
		return "complete"
	case RecoveryStateFailed021:
		return "failed"
	default:
		return "unknown"
	}
}

// validActionSequenceTable021 defines allowed transitions between actions per resource.
var validActionSequenceTable021 = map[ActionType021][]ActionType021{
	ActionTypeKill021:   {ActionTypeKill021, ActionTypeDestroy021},
	ActionTypeDestroy021: {ActionTypeDestroy021, ActionTypeRecover021},
	ActionTypeRecover021: {ActionTypeRecover021, ActionTypeCleanup021},
	ActionTypeCleanup021: {ActionTypeCleanup021, ActionTypeVerify021},
	ActionTypeVerify021:  {ActionTypeVerify021},
}

// defaultRecoveryTimelineTable021 provides deterministic test data.
var defaultRecoveryTimelineTable021 = []struct {
	Sequence int
	Action   ActionType021
	Resource string
	State    RecoveryState021
	Duration time.Duration // delay from previous entry
	Detail   string
}{
	{1, ActionTypeKill021, "res-a", RecoveryStateKilling021, 0, "initiate kill on res-a"},
	{2, ActionTypeKill021, "res-b", RecoveryStateKilling021, 100 * time.Millisecond, "initiate kill on res-b"},
	{3, ActionTypeDestroy021, "res-a", RecoveryStateDestroying021, 50 * time.Millisecond, "destroy after kill"},
	{4, ActionTypeDestroy021, "res-b", RecoveryStateDestroying021, 75 * time.Millisecond, "destroy after kill"},
	{5, ActionTypeRecover021, "res-a", RecoveryStateRecovering021, 200 * time.Millisecond, "recover after destroy"},
	{6, ActionTypeRecover021, "res-b", RecoveryStateRecovering021, 150 * time.Millisecond, "recover after destroy"},
	{7, ActionTypeCleanup021, "res-a", RecoveryStateCleaning021, 100 * time.Millisecond, "cleanup res-a"},
	{8, ActionTypeCleanup021, "res-b", RecoveryStateCleaning021, 100 * time.Millisecond, "cleanup res-b"},
	{9, ActionTypeVerify021, "res-a", RecoveryStateComplete021, 50 * time.Millisecond, "verify res-a"},
	{10, ActionTypeVerify021, "res-b", RecoveryStateComplete021, 50 * time.Millisecond, "verify res-b"},
}

// recoveryPriorityTable021 defines default priorities for action types.
var recoveryPriorityTable021 = map[ActionType021]int{
	ActionTypeKill021:   0,
	ActionTypeDestroy021: 1,
	ActionTypeRecover021: 2,
	ActionTypeCleanup021: 3,
	ActionTypeVerify021:  4,
}

// ValidateResourceAgentRecoveryTimeline_021 checks the timeline for consistency.
func ValidateResourceAgentRecoveryTimeline_021(tl *ResourceAgentRecoveryTimeline021) error {
	if tl == nil {
		return errors.New("recovery timeline is nil")
	}
	if len(tl.Actions) == 0 {
		return errors.New("timeline has no actions")
	}
	for i, a := range tl.Actions {
		if a.ResourceID == "" {
			return fmt.Errorf("action %d: empty resource ID", i)
		}
		if a.Action < ActionTypeKill021 || a.Action > ActionTypeVerify021 {
			return fmt.Errorf("action %d: invalid action type", i)
		}
		if a.State < RecoveryStateUninitialized021 || a.State > RecoveryStateFailed021 {
			return fmt.Errorf("action %d: invalid state", i)
		}
		if i > 0 && a.Timestamp.Before(tl.Actions[i-1].Timestamp) {
			return fmt.Errorf("action %d: timestamp out of order", i)
		}
	}
	// Validate per-resource action sequence.
	resMap := make(map[string][]ResourceAgentRecoveryAction021)
	for _, a := range tl.Actions {
		resMap[a.ResourceID] = append(resMap[a.ResourceID], a)
	}
	for rid, acts := range resMap {
		sort.Slice(acts, func(i, j int) bool { return acts[i].Timestamp.Before(acts[j].Timestamp) })
		seen := make(map[ActionType021]bool)
		for j, a := range acts {
			if j == 0 && a.Action != ActionTypeKill021 {
				return &RecoveryError021{
					ResourceID: rid,
					Phase:      a.State,
					Action:     a.Action,
					Message:    fmt.Sprintf("first action for resource %s must be kill, got %s", rid, a.Action),
				}
			}
			if a.Action == ActionTypeKill021 && seen[ActionTypeDestroy021] {
				return &RecoveryError021{
					ResourceID: rid,
					Phase:      a.State,
					Action:     a.Action,
					Message:    fmt.Sprintf("kill after destroy not allowed for %s", rid),
				}
			}
			if a.Action == ActionTypeDestroy021 && !seen[ActionTypeKill021] && j > 0 {
				return &RecoveryError021{
					ResourceID: rid,
					Phase:      a.State,
					Action:     a.Action,
					Message:    fmt.Sprintf("destroy without prior kill for %s", rid),
				}
			}
			if a.Action == ActionTypeRecover021 && !seen[ActionTypeDestroy021] {
				return &RecoveryError021{
					ResourceID: rid,
					Phase:      a.State,
					Action:     a.Action,
					Message:    fmt.Sprintf("recover without prior destroy for %s", rid),
				}
			}
			if a.Action == ActionTypeCleanup021 && !seen[ActionTypeRecover021] {
				return &RecoveryError021{
					ResourceID: rid,
					Phase:      a.State,
					Action:     a.Action,
					Message:    fmt.Sprintf("cleanup without prior recover for %s", rid),
				}
			}
			// Validate transition table.
			if j > 0 {
				prevAction := acts[j-1].Action
				allowed, ok := validActionSequenceTable021[prevAction]
				if !ok {
					return fmt.Errorf("no transition rule for action %s", prevAction)
				}
				found := false
				for _, allowedAct := range allowed {
					if a.Action == allowedAct {
						found = true
						break
					}
				}
				if !found {
					return &RecoveryError021{
						ResourceID: rid,
						Phase:      a.State,
						Action:     a.Action,
						Message:    fmt.Sprintf("invalid transition from %s to %s for %s", prevAction, a.Action, rid),
					}
				}
			}
			seen[a.Action] = true
		}
		// Final state check.
		lastAction := acts[len(acts)-1]
		if lastAction.Action == ActionTypeRecover021 &&
			lastAction.State != RecoveryStateRecovering021 &&
			lastAction.State != RecoveryStateComplete021 &&
			lastAction.State != RecoveryStateFailed021 {
			return fmt.Errorf("resource %s: unexpected state after recover: %s", rid, lastAction.State)
		}
		if lastAction.Action == ActionTypeVerify021 && lastAction.State != RecoveryStateComplete021 && lastAction.State != RecoveryStateFailed021 {
			return fmt.Errorf("resource %s: verify must end in complete or failed", rid)
		}
	}
	return nil
}

// ValidateResourceAgentCleanupVerification_021 checks the cleanup verification parameters.
func ValidateResourceAgentCleanupVerification_021(cv *ResourceAgentCleanupVerification021) error {
	if cv == nil {
		return errors.New("cleanup verification is nil")
	}
	if cv.CheckInterval <= 0 {
		return errors.New("check interval must be positive")
	}
	if cv.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if cv.Timeout < cv.CheckInterval {
		return errors.New("timeout must be at least check interval")
	}
	if len(cv.Resources) == 0 {
		return errors.New("at least one resource must be provided")
	}
	if cv.ExpectedState < RecoveryStateUninitialized021 || cv.ExpectedState > RecoveryStateFailed021 {
		return errors.New("invalid expected state")
	}
	return nil
}

// ValidateResourceAgentRecoveryPlan_021 checks a recovery plan for validity.
func ValidateResourceAgentRecoveryPlan_021(plan *ResourceAgentRecoveryPlan021) error {
	if plan == nil {
		return errors.New("recovery plan is nil")
	}
	if plan.ResourceID == "" {
		return errors.New("plan must have a resource ID")
	}
	if len(plan.Steps) == 0 {
		return errors.New("plan has no steps")
	}
	if plan.Deadline.IsZero() {
		return errors.New("plan deadline must be set")
	}
	if plan.Priority < 0 || plan.Priority > 100 {
		return errors.New("priority must be between 0 and 100")
	}
	for i, step := range plan.Steps {
		if step.Action < ActionTypeKill021 || step.Action > ActionTypeVerify021 {
			return fmt.Errorf("step %d: invalid action type", i)
		}
	}
	return nil
}

// ResourceAgentGenerateTimelineFromTable_021 creates a timeline from table entries.
func ResourceAgentGenerateTimelineFromTable_021(table []struct {
	Sequence int
	Action   ActionType021
	Resource string
	State    RecoveryState021
	Duration time.Duration
	Detail   string
}) *ResourceAgentRecoveryTimeline021 {
	if table == nil {
		// Use default table for determinism.
		table = defaultRecoveryTimelineTable021
	}
	tl := &ResourceAgentRecoveryTimeline021{}
	var base time.Time
	for _, entry := range table {
		base = base.Add(entry.Duration)
		tl.Actions = append(tl.Actions, ResourceAgentRecoveryAction021{
			Timestamp:  base,
			Action:     entry.Action,
			ResourceID: entry.Resource,
			State:      entry.State,
			Detail:     entry.Detail,
		})
	}
	return tl
}

// ResourceAgentNewTimeline_021 creates a new timeline with given actions.
func ResourceAgentNewTimeline_021(actions []ResourceAgentRecoveryAction021) *ResourceAgentRecoveryTimeline021 {
	return &ResourceAgentRecoveryTimeline021{Actions: actions}
}

// ResourceAgentPerformCleanupVerify_021 simulates cleanup verification.
func ResourceAgentPerformCleanupVerify_021(cv *ResourceAgentCleanupVerification021) (*CleanupVerificationResult021, error) {
	if err := ValidateResourceAgentCleanupVerification_021(cv); err != nil {
		return nil, err
	}
	// Deterministic simulation: assume all resources destroyed and recovered after timeout.
	result := &CleanupVerificationResult021{
		AllDestroyed:    true,
		AllRecovered:    true,
		PingCount:       int(cv.Timeout / cv.CheckInterval),
		LastAliveTime:   time.Now().Add(-cv.Timeout),
	}
	return result, nil
}
