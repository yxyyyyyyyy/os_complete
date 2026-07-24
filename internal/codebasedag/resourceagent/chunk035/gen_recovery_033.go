package chunk035

import (
	"errors"
	"fmt"
	"time"
)

// RecoveryActionType represents the type of recovery action.
type RecoveryActionType_033 int

const (
	ActionKill_033    RecoveryActionType_033 = iota // Kill the target
	ActionDestroy_033                                // Destroy the target
	ActionRecover_033                                // Recover the target
	ActionVerify_033                                 // Verify cleanup
)

// RecoveryTargetType indicates what kind of resource the action targets.
type RecoveryTargetType_033 int

const (
	TargetPod_033       RecoveryTargetType_033 = iota // Pod
	TargetContainer_033                               // Container
	TargetNode_033                                    // Node
	TargetVolume_033                                  // Volume
	TargetNetwork_033                                 // Network
)

// RecoveryStatus represents the current status of a recovery timeline.
type RecoveryStatus_033 int

const (
	StatusPending_033     RecoveryStatus_033 = iota // Awaiting execution
	StatusInProgress_033                            // Under execution
	StatusSucceeded_033                             // Completed successfully
	StatusFailed_033                                // Completed with failure
	StatusPartial_033                               // Some actions succeeded, some failed
)

func (s RecoveryStatus_033) String() string {
	switch s {
	case StatusPending_033:
		return "pending"
	case StatusInProgress_033:
		return "in_progress"
	case StatusSucceeded_033:
		return "succeeded"
	case StatusFailed_033:
		return "failed"
	case StatusPartial_033:
		return "partial"
	default:
		return "unknown"
	}
}

// RecoveryAction_033 represents a single action in the recovery timeline.
type RecoveryAction_033 struct {
	ActionType RecoveryActionType_033
	TargetType RecoveryTargetType_033
	TargetID   string
	Timestamp  time.Time
	Duration   time.Duration
	Error      string
	Succeeded  bool
}

// CleanupCheck_033 holds the verification status of a single cleanup item.
type CleanupCheck_033 struct {
	ResourceID string
	Cleaned    bool
	Detail     string
}

// CleanupVerificationResult_033 aggregates cleanup verification for multiple resources.
type CleanupVerificationResult_033 struct {
	Checks    []CleanupCheck_033
	AllPassed bool
	CreatedAt time.Time
}

// RecoveryTimeline_033 records the sequence of kill, destroy, recover, and verify actions.
type RecoveryTimeline_033 struct {
	ID            string
	Actions       []RecoveryAction_033
	StartTime     time.Time
	EndTime       time.Time
	Status        RecoveryStatus_033
	Label         string
	Priority      int
	CleanupResult *CleanupVerificationResult_033
}

// NewRecoveryTimeline_033 creates a new timeline with the given ID and label.
func NewRecoveryTimeline_033(id, label string) *RecoveryTimeline_033 {
	return &RecoveryTimeline_033{
		ID:        id,
		Label:     label,
		Actions:   make([]RecoveryAction_033, 0),
		StartTime: time.Now(),
		Status:    StatusPending_033,
	}
}

// AddAction_033 appends an action to the timeline after validation.
func (t *RecoveryTimeline_033) AddAction_033(action RecoveryAction_033) error {
	if t.Status != StatusPending_033 && t.Status != StatusInProgress_033 {
		return errors.New("cannot add action to completed timeline")
	}
	if action.ActionType == ActionVerify_033 && t.Status != StatusInProgress_033 {
		return errors.New("verify action requires in_progress status")
	}
	if action.TargetID == "" {
		return errors.New("action target ID must not be empty")
	}
	t.Actions = append(t.Actions, action)
	t.Status = StatusInProgress_033
	return nil
}

// Complete_033 marks the timeline as finished and sets the status based on action outcomes.
func (t *RecoveryTimeline_033) Complete_033() {
	t.EndTime = time.Now()
	allSucceeded := true
	anyFailed := false
	for _, a := range t.Actions {
		if !a.Succeeded {
			allSucceeded = false
			anyFailed = true
		}
	}
	switch {
	case allSucceeded:
		t.Status = StatusSucceeded_033
	case anyFailed:
		t.Status = StatusPartial_033
	default:
		t.Status = StatusFailed_033
	}
}

// ResourceAgentValidateTimeline_033 validates the entire timeline for consistency.
func ResourceAgentValidateTimeline_033(t *RecoveryTimeline_033) error {
	if t == nil {
		return errors.New("timeline is nil")
	}
	if t.ID == "" {
		return errors.New("timeline ID is required")
	}
	if len(t.Actions) == 0 {
		return errors.New("timeline must have at least one action")
	}
	if t.StartTime.IsZero() {
		return errors.New("start time must be set")
	}
	if t.EndTime.IsZero() && t.Status != StatusPending_033 && t.Status != StatusInProgress_033 {
		return errors.New("end time must be set for non-pending/in-progress timeline")
	}
	if t.Status < StatusPending_033 || t.Status > StatusPartial_033 {
		return fmt.Errorf("invalid status value: %d", t.Status)
	}
	for i, a := range t.Actions {
		if a.TargetID == "" {
			return fmt.Errorf("action %d: target ID is empty", i)
		}
		if a.ActionType < ActionKill_033 || a.ActionType > ActionVerify_033 {
			return fmt.Errorf("action %d: invalid action type", i)
		}
		if a.TargetType < TargetPod_033 || a.TargetType > TargetNetwork_033 {
			return fmt.Errorf("action %d: invalid target type", i)
		}
		if a.Timestamp.Before(t.StartTime) {
			return fmt.Errorf("action %d: timestamp before timeline start", i)
		}
	}
	return nil
}

// ResourceAgentValidateCleanup_033 validates the cleanup verification result.
func ResourceAgentValidateCleanup_033(result *CleanupVerificationResult_033) error {
	if result == nil {
		return errors.New("cleanup result is nil")
	}
	if len(result.Checks) == 0 {
		return errors.New("cleanup result must contain at least one check")
	}
	if result.CreatedAt.IsZero() {
		return errors.New("cleanup result creation timestamp is required")
	}
	for i, check := range result.Checks {
		if check.ResourceID == "" {
			return fmt.Errorf("check %d: resource ID is empty", i)
		}
	}
	return nil
}

// ResourceAgentVerifyCleanup_033 creates a verification result from a map of resource IDs to cleanup status.
func ResourceAgentVerifyCleanup_033(statuses map[string]bool) *CleanupVerificationResult_033 {
	result := &CleanupVerificationResult_033{
		Checks:    make([]CleanupCheck_033, 0, len(statuses)),
		AllPassed: true,
		CreatedAt: time.Now(),
	}
	for id, cleaned := range statuses {
		check := CleanupCheck_033{
			ResourceID: id,
			Cleaned:    cleaned,
		}
		if cleaned {
			check.Detail = "cleaned"
		} else {
			check.Detail = "not cleaned"
			result.AllPassed = false
		}
		result.Checks = append(result.Checks, check)
	}
	return result
}

// ResourceAgentKillSequence_033 generates an ordered slice of actions for a kill sequence.
// It expects a list of target IDs and returns kill actions ordered by priority (nodes first, then containers, then pods).
func ResourceAgentKillSequence_033(targets map[RecoveryTargetType_033][]string) []RecoveryAction_033 {
	order := []RecoveryTargetType_033{TargetNode_033, TargetContainer_033, TargetPod_033}
	actions := make([]RecoveryAction_033, 0)
	for _, t := range order {
		ids, ok := targets[t]
		if !ok {
			continue
		}
		for _, id := range ids {
			actions = append(actions, RecoveryAction_033{
				ActionType: ActionKill_033,
				TargetType: t,
				TargetID:   id,
				Timestamp:  time.Now(),
			})
		}
	}
	return actions
}

// ResourceAgentDestroySequence_033 generates destroy actions from a list of target IDs.
func ResourceAgentDestroySequence_033(targets map[RecoveryTargetType_033][]string) []RecoveryAction_033 {
	actions := make([]RecoveryAction_033, 0)
	for tType, ids := range targets {
		for _, id := range ids {
			actions = append(actions, RecoveryAction_033{
				ActionType: ActionDestroy_033,
				TargetType: tType,
				TargetID:   id,
				Timestamp:  time.Now(),
			})
		}
	}
	return actions
}

// ResourceAgentRecoverSequence_033 generates recover actions from a list of target IDs.
func ResourceAgentRecoverSequence_033(targets map[RecoveryTargetType_033][]string) []RecoveryAction_033 {
	actions := make([]RecoveryAction_033, 0)
	for tType, ids := range targets {
		for _, id := range ids {
			actions = append(actions, RecoveryAction_033{
				ActionType: ActionRecover_033,
				TargetType: tType,
				TargetID:   id,
				Timestamp:  time.Now(),
			})
		}
	}
	return actions
}

// recoveryActionTable_033 is a deterministic table of valid action combinations and their expected durations.
var recoveryActionTable_033 = []struct {
	ActionType RecoveryActionType_033
	TargetType RecoveryTargetType_033
	MinDuration time.Duration
	MaxDuration time.Duration
}{
	{ActionKill_033, TargetPod_033, 100 * time.Millisecond, 5 * time.Second},
	{ActionKill_033, TargetContainer_033, 50 * time.Millisecond, 3 * time.Second},
	{ActionKill_033, TargetNode_033, 500 * time.Millisecond, 30 * time.Second},
	{ActionDestroy_033, TargetPod_033, 200 * time.Millisecond, 10 * time.Second},
	{ActionDestroy_033, TargetContainer_033, 100 * time.Millisecond, 5 * time.Second},
	{ActionDestroy_033, TargetNode_033, 1 * time.Second, 60 * time.Second},
	{ActionDestroy_033, TargetVolume_033, 300 * time.Millisecond, 15 * time.Second},
	{ActionDestroy_033, TargetNetwork_033, 500 * time.Millisecond, 20 * time.Second},
	{ActionRecover_033, TargetPod_033, 1 * time.Second, 30 * time.Second},
	{ActionRecover_033, TargetContainer_033, 500 * time.Millisecond, 20 * time.Second},
	{ActionRecover_033, TargetNode_033, 5 * time.Second, 120 * time.Second},
	{ActionVerify_033, TargetPod_033, 100 * time.Millisecond, 2 * time.Second},
	{ActionVerify_033, TargetContainer_033, 50 * time.Millisecond, 1 * time.Second},
	{ActionVerify_033, TargetNode_033, 200 * time.Millisecond, 5 * time.Second},
}

// cleanupVerificationTable_033 provides deterministic expected results for cleanup checks.
var cleanupVerificationTable_033 = []struct {
	ResourceID string
	Expected   bool
	Detail     string
}{
	{"pod-abc", true, "cleaned"},
	{"container-123", true, "cleaned"},
	{"node-node1", false, "not cleaned"},
	{"vol-xyz", true, "cleaned"},
	{"net-net1", true, "cleaned"},
}

// ResourceAgentCheckActionDuration_033 validates that an action's duration falls within the expected range from the table.
func ResourceAgentCheckActionDuration_033(action RecoveryAction_033) error {
	for _, entry := range recoveryActionTable_033 {
		if entry.ActionType == action.ActionType && entry.TargetType == action.TargetType {
			if action.Duration < entry.MinDuration {
				return fmt.Errorf("duration %v is less than minimum %v for %v/%v", action.Duration, entry.MinDuration, action.ActionType, action.TargetType)
			}
			if action.Duration > entry.MaxDuration {
				return fmt.Errorf("duration %v exceeds maximum %v for %v/%v", action.Duration, entry.MaxDuration, action.ActionType, action.TargetType)
			}
			return nil
		}
	}
	return fmt.Errorf("unknown action type %v and target type %v combination", action.ActionType, action.TargetType)
}

// ResourceAgentVerifyCleanupFromTable_033 uses cleanupVerificationTable_033 to verify the given checks.
// It returns the result and an error if any check does not match expected.
func ResourceAgentVerifyCleanupFromTable_033(checks []CleanupCheck_033) (*CleanupVerificationResult_033, error) {
	result := &CleanupVerificationResult_033{
		Checks:    make([]CleanupCheck_033, 0),
		AllPassed: true,
		CreatedAt: time.Now(),
	}
	expectedMap := make(map[string]bool)
	for _, e := range cleanupVerificationTable_033 {
		expectedMap[e.ResourceID] = e.Expected
	}
	for _, c := range checks {
		expected, exists := expectedMap[c.ResourceID]
		if !exists {
			return nil, fmt.Errorf("unexpected resource ID %s in table", c.ResourceID)
		}
		if c.Cleaned != expected {
			result.AllPassed = false
			result.Checks = append(result.Checks, CleanupCheck_033{
				ResourceID: c.ResourceID,
				Cleaned:    c.Cleaned,
				Detail:     fmt.Sprintf("expected %v, got %v", expected, c.Cleaned),
			})
		} else {
			result.Checks = append(result.Checks, c)
		}
	}
	return result, nil
}

// helperActionValidator_033 checks that an action has valid type and target.
func helperActionValidator_033(a RecoveryAction_033) bool {
	return a.ActionType >= ActionKill_033 && a.ActionType <= ActionVerify_033 &&
		a.TargetType >= TargetPod_033 && a.TargetType <= TargetNetwork_033
}

// ResourceAgentMergeTimelines_033 combines multiple timelines into one, preserving order.
func ResourceAgentMergeTimelines_033(timelines []*RecoveryTimeline_033) (*RecoveryTimeline_033, error) {
	if len(timelines) == 0 {
		return nil, errors.New("at least one timeline required to merge")
	}
	merged := NewRecoveryTimeline_033("merged-"+timelines[0].ID, timelines[0].Label)
	for _, t := range timelines {
		for _, a := range t.Actions {
			if err := merged.AddAction_033(a); err != nil {
				return nil, fmt.Errorf("failed to add action from timeline %s: %w", t.ID, err)
			}
		}
	}
	merged.StartTime = time.Now()
	return merged, nil
}

// ResourceAgentIsTimelineComplete_033 checks if all actions in the timeline have been executed.
func ResourceAgentIsTimelineComplete_033(t *RecoveryTimeline_033) bool {
	if t == nil {
		return false
	}
	for _, a := range t.Actions {
		if a.Timestamp.IsZero() {
			return false
		}
	}
	return true
}

// ResourceAgentGenerateTimelineSummary_033 produces a summary string from a timeline.
func ResourceAgentGenerateTimelineSummary_033(t *RecoveryTimeline_033) string {
	if t == nil {
		return "no timeline"
	}
	return fmt.Sprintf("ID=%s Label=%s Status=%s Actions=%d Start=%s End=%s",
		t.ID, t.Label, t.Status.String(), len(t.Actions),
		t.StartTime.Format(time.RFC3339), t.EndTime.Format(time.RFC3339))
}

// ResourceAgentValidateTimelineActionSequence_033 ensures that actions follow a valid pattern:
// kill and destroy actions may be interleaved, but recover actions should come after destroy actions.
func ResourceAgentValidateTimelineActionSequence_033(t *RecoveryTimeline_033) error {
	if t == nil {
		return errors.New("timeline is nil")
	}
	hasDestroy := false
	for i, a := range t.Actions {
		if a.ActionType == ActionDestroy_033 {
			hasDestroy = true
		}
		if a.ActionType == ActionRecover_033 && !hasDestroy {
			return fmt.Errorf("action %d: recover before any destroy action", i)
		}
	}
	return nil
}

// ResourceAgentGetActionByID_033 returns the index and action for a given target ID.
func ResourceAgentGetActionByID_033(t *RecoveryTimeline_033, targetID string) (int, RecoveryAction_033, bool) {
	for i, a := range t.Actions {
		if a.TargetID == targetID {
			return i, a, true
		}
	}
	return -1, RecoveryAction_033{}, false
}

// cleanupResourceSet_033 is a deterministic set of resource IDs used for verification.
var cleanupResourceSet_033 = []string{
	"res-001",
	"res-002",
	"res-003",
	"res-004",
	"res-005",
}

// ResourceAgentBuildCleanupChecks_033 creates a full set of CleanupCheck from a list of resource IDs, all marked as not cleaned.
func ResourceAgentBuildCleanupChecks_033(ids []string) []CleanupCheck_033 {
	checks := make([]CleanupCheck_033, len(ids))
	for i, id := range ids {
		checks[i] = CleanupCheck_033{
			ResourceID: id,
			Cleaned:    false,
			Detail:     "pending verification",
		}
	}
	return checks
}

// killDestroyRecoverOrder_033 defines the preferred order of action types in a timeline.
var killDestroyRecoverOrder_033 = []RecoveryActionType_033{
	ActionKill_033,
	ActionDestroy_033,
	ActionRecover_033,
	ActionVerify_033,
}

// ResourceAgentSortActionsByOrder_033 sorts actions in place based on killDestroyRecoverOrder_033.
func ResourceAgentSortActionsByOrder_033(actions []RecoveryAction_033) {
	n := len(actions)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			oi := indexInOrder_033(actions[i].ActionType)
			oj := indexInOrder_033(actions[j].ActionType)
			if oi > oj {
				actions[i], actions[j] = actions[j], actions[i]
			}
		}
	}
}

func indexInOrder_033(t RecoveryActionType_033) int {
	for i, v := range killDestroyRecoverOrder_033 {
		if v == t {
			return i
		}
	}
	return len(killDestroyRecoverOrder_033)
}

// timelineIDPrefixes_033 is a mapping of action type to preferred ID prefix for generated timelines.
var timelineIDPrefixes_033 = map[RecoveryActionType_033]string{
	ActionKill_033:    "kill",
	ActionDestroy_033: "destroy",
	ActionRecover_033: "recover",
	ActionVerify_033:  "verify",
}

// ResourceAgentGenerateTimelineID_033 creates a deterministic timeline ID based on action type and a counter.
func ResourceAgentGenerateTimelineID_033(prefix string, counter int) string {
	if prefix == "" {
		prefix = "timeline"
	}
	return fmt.Sprintf("%s-%05d", prefix, counter)
}

// simulateActionExecution_033 updates an action's timestamp and duration as if it was executed.
func simulateActionExecution_033(action *RecoveryAction_033, executionTime time.Duration) {
	action.Duration = executionTime
	action.Timestamp = time.Now()
	action.Succeeded = true
}

// ResourceAgentExecuteAndVerify_033 simulates execution of all actions in a timeline and then performs cleanup verification.
func ResourceAgentExecuteAndVerify_033(t *RecoveryTimeline_033) (*CleanupVerificationResult_033, error) {
	if t == nil {
		return nil, errors.New("timeline is nil")
	}
	for i := range t.Actions {
		entry := lookupDurationEntry_033(t.Actions[i].ActionType, t.Actions[i].TargetType)
		if entry == nil {
			return nil, fmt.Errorf("action %d: no duration entry found", i)
		}
		simulateActionExecution_033(&t.Actions[i], entry.MinDuration+(entry.MaxDuration-entry.MinDuration)/2)
	}
	t.Complete_033()
	// Build verification from all target IDs
	ids := make([]string, len(t.Actions))
	for i, a := range t.Actions {
		ids[i] = a.TargetID
	}
	checks := ResourceAgentBuildCleanupChecks_033(ids)
	result := &CleanupVerificationResult_033{
		Checks:    checks,
		AllPassed: false,
		CreatedAt: time.Now(),
	}
	t.CleanupResult = result
	return result, nil
}

func lookupDurationEntry_033(at RecoveryActionType_033, tt RecoveryTargetType_033) *struct {
	ActionType RecoveryActionType_033
	TargetType RecoveryTargetType_033
	MinDuration time.Duration
	MaxDuration time.Duration
} {
	for i := range recoveryActionTable_033 {
		if recoveryActionTable_033[i].ActionType == at && recoveryActionTable_033[i].TargetType == tt {
			return &recoveryActionTable_033[i]
		}
	}
	return nil
}
