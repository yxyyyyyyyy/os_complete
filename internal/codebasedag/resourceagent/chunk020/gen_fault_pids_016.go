package chunk020

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ---------- Constants ----------

const (
	DefaultPidMax_016          = 32768
	DefaultForkBombThreshold_016 = 80  // percentage of PidMax
	DefaultGracePeriod_016     = 5 * time.Second
	DefaultContainmentAction_016 = "alert"
)

var allowedActions_016 = []string{"alert", "contain", "kill", "none"}

// ---------- Types ----------

// ResourceAgentPidConfig_016 holds configuration for PID monitoring and fork bomb detection.
type ResourceAgentPidConfig_016 struct {
	PidMax        int           `json:"pid_max"`
	Threshold     int           `json:"threshold"`      // percentage (0-100)
	Action        string        `json:"action"`
	GracePeriod   time.Duration `json:"grace_period"`
	EnableMonitor bool          `json:"enable_monitor"`
}

// ResourceAgentForkBombEvent_016 represents a detected fork bomb event.
type ResourceAgentForkBombEvent_016 struct {
	Timestamp   time.Time `json:"timestamp"`
	PidCount    int       `json:"pid_count"`
	PidMax      int       `json:"pid_max"`
	Threshold   int       `json:"threshold"`
	ActionTaken string    `json:"action_taken"`
}

// ResourceAgentContainmentState_016 tracks containment actions.
type ResourceAgentContainmentState_016 struct {
	mu           sync.Mutex
	Active       bool          `json:"active"`
	StartedAt    time.Time     `json:"started_at"`
	Duration     time.Duration `json:"duration"`
	Action       string        `json:"action"`
	PidCountAt   int           `json:"pid_count_at"`
	EventHistory []ResourceAgentForkBombEvent_016 `json:"event_history"`
}

// ResourceAgentForkBombScenario_016 is a table-driven scenario for testing detection.
type ResourceAgentForkBombScenario_016 struct {
	Name           string
	PidMax         int
	Threshold      int
	CurrentPids    int
	ExpectAlert    bool
	ExpectAction   string
	ExpectError    bool
}

// ---------- Constructor & Validation ----------

// NewResourceAgentPidConfig_016 creates a validated PID config.
func NewResourceAgentPidConfig_016(pidMax int, threshold int, action string, grace time.Duration) (*ResourceAgentPidConfig_016, error) {
	cfg := &ResourceAgentPidConfig_016{
		PidMax:        pidMax,
		Threshold:     threshold,
		Action:        action,
		GracePeriod:   grace,
		EnableMonitor: true,
	}
	if err := ValidateResourceAgentPidConfig_016(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ValidateResourceAgentPidConfig_016 validates all fields of the config.
func ValidateResourceAgentPidConfig_016(cfg *ResourceAgentPidConfig_016) error {
	if cfg == nil {
		return errors.New("pid config is nil")
	}
	if cfg.PidMax <= 0 {
		return fmt.Errorf("pid_max must be positive: %d", cfg.PidMax)
	}
	if cfg.PidMax > 4194304 { // reasonable upper bound
		return fmt.Errorf("pid_max too large: %d (max 4194304)", cfg.PidMax)
	}
	if cfg.Threshold < 0 || cfg.Threshold > 100 {
		return fmt.Errorf("threshold must be 0-100: %d", cfg.Threshold)
	}
	if cfg.GracePeriod < 0 {
		return fmt.Errorf("grace_period cannot be negative: %v", cfg.GracePeriod)
	}
	validAction := false
	for _, a := range allowedActions_016 {
		if cfg.Action == a {
			validAction = true
			break
		}
	}
	if !validAction {
		return fmt.Errorf("action must be one of [%s], got: %s", strings.Join(allowedActions_016, ", "), cfg.Action)
	}
	return nil
}

// ---------- Detection & Containment ----------

// ResourceAgentDetectForkBomb_016 returns true if current PID count exceeds the threshold.
func ResourceAgentDetectForkBomb_016(currentPids int, config *ResourceAgentPidConfig_016) (bool, error) {
	if config == nil {
		return false, errors.New("config is nil")
	}
	if err := ValidateResourceAgentPidConfig_016(config); err != nil {
		return false, err
	}
	if currentPids < 0 {
		return false, fmt.Errorf("currentPids cannot be negative: %d", currentPids)
	}
	threshold := (config.PidMax * config.Threshold) / 100
	return currentPids >= threshold, nil
}

// ResourceAgentCheckContainment_016 determines the appropriate action based on current PIDs.
func ResourceAgentCheckContainment_016(currentPids int, config *ResourceAgentPidConfig_016) (string, error) {
	alert, err := ResourceAgentDetectForkBomb_016(currentPids, config)
	if err != nil {
		return "", err
	}
	if !alert {
		return "none", nil
	}
	return config.Action, nil
}

// ResourceAgentContainmentActionToString_016 returns a human-readable description.
func ResourceAgentContainmentActionToString_016(action string) string {
	switch action {
	case "alert":
		return "alerting system administrator"
	case "contain":
		return "containing fork bomb via cgroup pids.max"
	case "kill":
		return "killing offending processes"
	case "none":
		return "no action taken"
	default:
		return fmt.Sprintf("unknown action: %s", action)
	}
}

// ---------- Table-Driven Helpers ----------

// ResourceAgentForkBombScenarios_016 returns a list of deterministic scenarios for validation and detection.
func ResourceAgentForkBombScenarios_016() []ResourceAgentForkBombScenario_016 {
	return []ResourceAgentForkBombScenario_016{
		{
			Name:         "under threshold – no alert",
			PidMax:       100,
			Threshold:    80,
			CurrentPids:  50,
			ExpectAlert:  false,
			ExpectAction: "none",
			ExpectError:  false,
		},
		{
			Name:         "exact threshold – alert",
			PidMax:       100,
			Threshold:    80,
			CurrentPids:  80,
			ExpectAlert:  true,
			ExpectAction: "alert",
			ExpectError:  false,
		},
		{
			Name:         "above threshold – alert",
			PidMax:       200,
			Threshold:    50,
			CurrentPids:  150,
			ExpectAlert:  true,
			ExpectAction: "alert",
			ExpectError:  false,
		},
		{
			Name:         "zero current – no alert",
			PidMax:       1000,
			Threshold:    90,
			CurrentPids:  0,
			ExpectAlert:  false,
			ExpectAction: "none",
			ExpectError:  false,
		},
		{
			Name:         "negative current – error",
			PidMax:       100,
			Threshold:    80,
			CurrentPids:  -1,
			ExpectAlert:  false,
			ExpectAction: "none",
			ExpectError:  true,
		},
		{
			Name:         "threshold zero – no alert (unless pids also zero)",
			PidMax:       100,
			Threshold:    0,
			CurrentPids:  50,
			ExpectAlert:  false,
			ExpectAction: "none",
			ExpectError:  false,
		},
		{
			Name:         "threshold 100 – alert only at pid_max",
			PidMax:       32768,
			Threshold:    100,
			CurrentPids:  32768,
			ExpectAlert:  true,
			ExpectAction: "alert",
			ExpectError:  false,
		},
		{
			Name:         "pid_max 1 – minimal",
			PidMax:       1,
			Threshold:    100,
			CurrentPids:  1,
			ExpectAlert:  true,
			ExpectAction: "alert",
			ExpectError:  false,
		},
	}
}

// ResourceAgentValidationScenarios_016 returns configs for validation tests.
func ResourceAgentValidationScenarios_016() []struct {
	PidMax      int
	Threshold   int
	Action      string
	GracePeriod time.Duration
	ExpectError bool
} {
	return []struct {
		PidMax      int
		Threshold   int
		Action      string
		GracePeriod time.Duration
		ExpectError bool
	}{
		{PidMax: 32768, Threshold: 80, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: false},
		{PidMax: 0, Threshold: 80, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: true},
		{PidMax: -1, Threshold: 80, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: true},
		{PidMax: 32768, Threshold: -1, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: true},
		{PidMax: 32768, Threshold: 101, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: true},
		{PidMax: 32768, Threshold: 80, Action: "kill", GracePeriod: 5 * time.Second, ExpectError: false},
		{PidMax: 32768, Threshold: 80, Action: "contain", GracePeriod: 5 * time.Second, ExpectError: false},
		{PidMax: 32768, Threshold: 80, Action: "none", GracePeriod: 5 * time.Second, ExpectError: false},
		{PidMax: 32768, Threshold: 80, Action: "unknown", GracePeriod: 5 * time.Second, ExpectError: true},
		{PidMax: 32768, Threshold: 80, Action: "alert", GracePeriod: -1 * time.Second, ExpectError: true},
		{PidMax: 4194305, Threshold: 80, Action: "alert", GracePeriod: 5 * time.Second, ExpectError: true}, // exceeds max
	}
}

// ResourceAgentPidMaxRecommendations_016 returns table of recommended PidMax values per system type.
func ResourceAgentPidMaxRecommendations_016() map[string]int {
	return map[string]int{
		"tiny":    1024,
		"small":   8192,
		"medium":  32768,
		"large":   131072,
		"huge":    4194304,
	}
}

// ---------- Helper Functions ----------

// ResourceAgentThresholdPids_016 computes the PID count at which alert triggers.
func ResourceAgentThresholdPids_016(config *ResourceAgentPidConfig_016) (int, error) {
	if config == nil {
		return 0, errors.New("config is nil")
	}
	if err := ValidateResourceAgentPidConfig_016(config); err != nil {
		return 0, err
	}
	return (config.PidMax * config.Threshold) / 100, nil
}

// ResourceAgentIsForkBombActive_016 checks if containment is currently active.
func ResourceAgentIsForkBombActive_016(state *ResourceAgentContainmentState_016) bool {
	if state == nil {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.Active
}

// ResourceAgentStartContainment_016 records containment start.
func ResourceAgentStartContainment_016(state *ResourceAgentContainmentState_016, pidCount int, action string) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.Active = true
	state.StartedAt = time.Now()
	state.PidCountAt = pidCount
	state.Action = action
}

// ResourceAgentEndContainment_016 records containment end and logs event.
func ResourceAgentEndContainment_016(state *ResourceAgentContainmentState_016, pidCount int) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.Active {
		return
	}
	duration := time.Since(state.StartedAt)
	state.Duration = duration
	event := ResourceAgentForkBombEvent_016{
		Timestamp:   time.Now(),
		PidCount:    pidCount,
		PidMax:      0, // not stored in state; could be passed separately
		Threshold:   0,
		ActionTaken: state.Action,
	}
	state.EventHistory = append(state.EventHistory, event)
	state.Active = false
}

// ResourceAgentNewContainmentState_016 creates a fresh containment state.
func ResourceAgentNewContainmentState_016() *ResourceAgentContainmentState_016 {
	return &ResourceAgentContainmentState_016{
		EventHistory: make([]ResourceAgentForkBombEvent_016, 0),
	}
}

// ResourceAgentCleanEventHistory_016 removes events older than maxAge.
func ResourceAgentCleanEventHistory_016(state *ResourceAgentContainmentState_016, maxAge time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	var kept []ResourceAgentForkBombEvent_016
	for _, e := range state.EventHistory {
		if e.Timestamp.After(cutoff) {
			kept = append(kept, e)
		}
	}
	state.EventHistory = kept
}

// ResourceAgentEventCount_016 returns the number of recorded events.
func ResourceAgentEventCount_016(state *ResourceAgentContainmentState_016) int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return len(state.EventHistory)
}

// ResourceAgentIsPidMaxViolation_016 checks if the current PID count exceeds PidMax.
func ResourceAgentIsPidMaxViolation_016(currentPids int, config *ResourceAgentPidConfig_016) (bool, error) {
	if config == nil {
		return false, errors.New("config is nil")
	}
	if err := ValidateResourceAgentPidConfig_016(config); err != nil {
		return false, err
	}
	return currentPids > config.PidMax, nil
}

// ResourceAgentFormatPidSummary_016 returns a formatted string for logging.
func ResourceAgentFormatPidSummary_016(currentPids int, config *ResourceAgentPidConfig_016) string {
	if config == nil {
		return "pid config missing"
	}
	threshold, _ := ResourceAgentThresholdPids_016(config) // error already validated
	return fmt.Sprintf("pids=%d/%d (threshold=%d, action=%s)", currentPids, config.PidMax, threshold, config.Action)
}

// ---------- Additional Validation Variants ----------

// ValidateResourceAgentForkBombConfigSlice_016 validates a slice of configs.
func ValidateResourceAgentForkBombConfigSlice_016(cfgs []*ResourceAgentPidConfig_016) error {
	for i, cfg := range cfgs {
		if err := ValidateResourceAgentPidConfig_016(cfg); err != nil {
			return fmt.Errorf("config[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateResourceAgentContainmentState_016 checks containment state invariants.
func ValidateResourceAgentContainmentState_016(state *ResourceAgentContainmentState_016) error {
	if state == nil {
		return errors.New("containment state is nil")
	}
	if state.Duration < 0 {
		return fmt.Errorf("state duration cannot be negative: %v", state.Duration)
	}
	if state.PidCountAt < 0 {
		return fmt.Errorf("state pid_count_at cannot be negative: %d", state.PidCountAt)
	}
	return nil
}

// ---------- End of File ----------
