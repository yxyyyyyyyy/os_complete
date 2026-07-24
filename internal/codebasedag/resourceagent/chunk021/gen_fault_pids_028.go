package chunk021

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// ResourceAgentPidsConfig holds all tunable parameters for PIDS limit enforcement
// and fork bomb detection inside a container or resource group.
type ResourceAgentPidsConfig struct {
	// MaxPids is the absolute upper bound on the number of processes allowed.
	MaxPids int

	// BurstLimit is the maximum number of additional processes allowed momentarily
	// before the enforcement kicks in. If 0, MaxPids is used as the combined limit.
	BurstLimit int

	// CheckInterval defines how often the agent evaluates the process count.
	CheckInterval time.Duration

	// EnableForkBombDetection when true enables rate-based detection of fork bombs.
	EnableForkBombDetection bool

	// ForkBombRateThreshold is the allowed fork rate (processes per second) before
	// an alert is raised. Ignored if EnableForkBombDetection is false.
	ForkBombRateThreshold float64

	// ForkBombWindow is the sliding window over which the fork rate is averaged.
	ForkBombWindow time.Duration

	// PenaltyDuration is the time the agent will fully block new processes after
	// a containment violation is detected.
	PenaltyDuration time.Duration
}

// ResourceAgentForkBombScenario describes a known or testable fork bomb pattern.
type ResourceAgentForkBombScenario struct {
	// Name is a human-readable identifier for this scenario.
	Name string

	// ProcessCount is the steady-state number of processes after the bomb.
	ProcessCount int

	// RatePerSec is the fork rate (processes created per second).
	RatePerSec float64

	// ExpectedAction indicates what the agent should do: "block", "alert", or "none".
	ExpectedAction string

	// MaxPids is the PIDS limit in effect when this scenario is applied.
	MaxPids int

	// Burst is the optional burst limit active during the scenario.
	Burst int
}

// ResourceAgentPidsScenarioResult stores the outcome of evaluating a scenario
// against a given PIDS configuration.
type ResourceAgentPidsScenarioResult struct {
	ScenarioName  string
	ConfigMaxPids int
	ConfigBurst   int
	Violation     bool
	Action        string
	Message       string
}

// ResourceAgentPidsContainmentChecker provides methods to validate and enforce
// PIDS limits, detect fork bombs, and manage penalty periods.
type ResourceAgentPidsContainmentChecker struct {
	mu       sync.Mutex
	config   ResourceAgentPidsConfig
	penalty  time.Time
	counter  int
	lastTime time.Time
}

// NewResourceAgentPidsContainmentChecker_028 creates a new checker with the given config.
func NewResourceAgentPidsContainmentChecker_028(cfg *ResourceAgentPidsConfig) (*ResourceAgentPidsContainmentChecker, error) {
	if err := ValidatePidsConfig_028(cfg); err != nil {
		return nil, err
	}
	return &ResourceAgentPidsContainmentChecker{
		config:   *cfg,
		penalty:  time.Time{},
		counter:  0,
		lastTime: time.Now(),
	}, nil
}

// ValidatePidsConfig_028 validates that the PIDS configuration is self-consistent
// and all numeric values are positive and sensible.
func ValidatePidsConfig_028(cfg *ResourceAgentPidsConfig) error {
	if cfg == nil {
		return errors.New("PIDS config cannot be nil")
	}
	if cfg.MaxPids <= 0 {
		return fmt.Errorf("MaxPids must be positive, got %d", cfg.MaxPids)
	}
	if cfg.MaxPids > math.MaxInt32 {
		return fmt.Errorf("MaxPids exceeds max int32: %d", cfg.MaxPids)
	}
	if cfg.BurstLimit < 0 {
		return fmt.Errorf("BurstLimit must be non-negative, got %d", cfg.BurstLimit)
	}
	if cfg.BurstLimit > 0 && cfg.BurstLimit > cfg.MaxPids {
		return fmt.Errorf("BurstLimit (%d) cannot be greater than MaxPids (%d)", cfg.BurstLimit, cfg.MaxPids)
	}
	if cfg.CheckInterval <= 0 {
		return fmt.Errorf("CheckInterval must be positive, got %v", cfg.CheckInterval)
	}
	if cfg.EnableForkBombDetection {
		if cfg.ForkBombRateThreshold <= 0 {
			return fmt.Errorf("ForkBombRateThreshold must be positive when detection enabled, got %f", cfg.ForkBombRateThreshold)
		}
		if cfg.ForkBombWindow <= 0 {
			return fmt.Errorf("ForkBombWindow must be positive when detection enabled, got %v", cfg.ForkBombWindow)
		}
	}
	if cfg.PenaltyDuration < 0 {
		return fmt.Errorf("PenaltyDuration cannot be negative, got %v", cfg.PenaltyDuration)
	}
	return nil
}

// ValidateForkBombScenario_028 validates a fork bomb scenario definition.
func ValidateForkBombScenario_028(s *ResourceAgentForkBombScenario) error {
	if s == nil {
		return errors.New("scenario cannot be nil")
	}
	if s.Name == "" {
		return errors.New("scenario Name must not be empty")
	}
	if s.ProcessCount < 0 {
		return fmt.Errorf("ProcessCount must be >= 0, got %d", s.ProcessCount)
	}
	if s.RatePerSec < 0 {
		return fmt.Errorf("RatePerSec must be >= 0, got %f", s.RatePerSec)
	}
	validActions := map[string]bool{"block": true, "alert": true, "none": true}
	if !validActions[s.ExpectedAction] {
		return fmt.Errorf("ExpectedAction must be one of 'block', 'alert', 'none', got %q", s.ExpectedAction)
	}
	if s.MaxPids <= 0 {
		return fmt.Errorf("MaxPids must be positive, got %d", s.MaxPids)
	}
	if s.Burst < 0 {
		return fmt.Errorf("Burst must be >= 0, got %d", s.Burst)
	}
	return nil
}

// CheckContainment_028 evaluates the current process count (provided externally)
// against the PIDS limits and fork bomb rate. It returns an error if a containment
// action is needed.
func (cc *ResourceAgentPidsContainmentChecker) CheckContainment_028(currentPids int) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// If we are in a penalty period, reject any new processes.
	if !cc.penalty.IsZero() && time.Now().Before(cc.penalty) {
		return fmt.Errorf("containment penalty active until %v, blocking process creation", cc.penalty)
	}

	// Absolute limit check.
	effectiveLimit := cc.config.MaxPids
	if cc.config.BurstLimit > 0 {
		effectiveLimit += cc.config.BurstLimit
	}
	if currentPids > effectiveLimit {
		cc.startPenaltyLocked()
		return fmt.Errorf("process count %d exceeds effective limit %d (max=%d, burst=%d)", currentPids, effectiveLimit, cc.config.MaxPids, cc.config.BurstLimit)
	}

	// Fork bomb rate detection.
	if cc.config.EnableForkBombDetection {
		now := time.Now()
		elapsed := now.Sub(cc.lastTime)
		// Update rate counter based on process increases since last check.
		// For simplicity, we assume we are called at every check interval.
		if elapsed > 0 && elapsed < cc.config.ForkBombWindow {
			rate := float64(currentPids-cc.counter) / elapsed.Seconds()
			if rate > cc.config.ForkBombRateThreshold {
				cc.startPenaltyLocked()
				return fmt.Errorf("fork bomb detected: fork rate %.2f pids/s exceeds threshold %.2f pids/s", rate, cc.config.ForkBombRateThreshold)
			}
		}
		cc.counter = currentPids
		cc.lastTime = now
	}

	return nil
}

// startPenaltyLocked sets the penalty period (must hold lock).
func (cc *ResourceAgentPidsContainmentChecker) startPenaltyLocked() {
	if cc.config.PenaltyDuration > 0 {
		cc.penalty = time.Now().Add(cc.config.PenaltyDuration)
	} else {
		cc.penalty = time.Now().Add(10 * time.Second) // default short penalty
	}
}

// ResetPenalty_028 clears any active penalty.
func (cc *ResourceAgentPidsContainmentChecker) ResetPenalty_028() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.penalty = time.Time{}
	cc.counter = 0
	cc.lastTime = time.Now()
}

// DefaultResourceAgentPidsConfigs_028 is a table of common PIDS configurations
// used in container runtimes. Each entry corresponds to a known environment.
var DefaultResourceAgentPidsConfigs_028 = []ResourceAgentPidsConfig{
	{
		MaxPids:                100,
		BurstLimit:             10,
		CheckInterval:          1 * time.Second,
		EnableForkBombDetection: true,
		ForkBombRateThreshold:  50.0,
		ForkBombWindow:         1 * time.Second,
		PenaltyDuration:        5 * time.Second,
	},
	{
		MaxPids:                200,
		BurstLimit:             20,
		CheckInterval:          500 * time.Millisecond,
		EnableForkBombDetection: true,
		ForkBombRateThreshold:  100.0,
		ForkBombWindow:         2 * time.Second,
		PenaltyDuration:        10 * time.Second,
	},
	{
		MaxPids:                500,
		BurstLimit:             0,
		CheckInterval:          2 * time.Second,
		EnableForkBombDetection: false,
		ForkBombRateThreshold:  0,
		ForkBombWindow:         0,
		PenaltyDuration:        0,
	},
}

// ResourceAgentForkBombScenarios_028 is a deterministic table of fork bomb
// scenarios used for testing and validation.
var ResourceAgentForkBombScenarios_028 = []ResourceAgentForkBombScenario{
	{
		Name:           "slow_creep",
		ProcessCount:   150,
		RatePerSec:     5.0,
		ExpectedAction: "none",
		MaxPids:        200,
		Burst:          20,
	},
	{
		Name:           "rapid_burst",
		ProcessCount:   300,
		RatePerSec:     200.0,
		ExpectedAction: "block",
		MaxPids:        100,
		Burst:          50,
	},
	{
		Name:           "overflow_burst",
		ProcessCount:   160,
		RatePerSec:     10.0,
		ExpectedAction: "block",
		MaxPids:        100,
		Burst:          50,
	},
	{
		Name:           "within_limits",
		ProcessCount:   50,
		RatePerSec:     2.0,
		ExpectedAction: "none",
		MaxPids:        200,
		Burst:          10,
	},
	{
		Name:           "extreme_fork",
		ProcessCount:   1000,
		RatePerSec:     500.0,
		ExpectedAction: "block",
		MaxPids:        100,
		Burst:          0,
	},
}

// EvaluateScenarios_028 runs each scenario against a given config and returns
// results. It is used for deterministic validation.
func EvaluateScenarios_028(cfg *ResourceAgentPidsConfig, scenarios []ResourceAgentForkBombScenario) ([]ResourceAgentPidsScenarioResult, error) {
	if err := ValidatePidsConfig_028(cfg); err != nil {
		return nil, err
	}
	results := make([]ResourceAgentPidsScenarioResult, 0, len(scenarios))
	for _, s := range scenarios {
		if err := ValidateForkBombScenario_028(&s); err != nil {
			return nil, fmt.Errorf("invalid scenario %q: %w", s.Name, err)
		}
		// Simulate containment check.
		checker, err := NewResourceAgentPidsContainmentChecker_028(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create checker for scenario %q: %w", s.Name, err)
		}
		// We set the counter to a reasonable starting point to simulate previous state.
		checker.counter = 0
		checker.lastTime = time.Now().Add(-1 * time.Second)
		// Apply the scenario's fork rate by simulating process count increase.
		// We call check multiple times to allow rate detection.
		var violation bool
		var message string
		for i := 0; i < 5; i++ {
			// Simulate process count climbing toward scenario ProcessCount.
			pids := int(float64(s.ProcessCount) * float64(i+1) / 5.0)
			err := checker.CheckContainment_028(pids)
			if err != nil {
				violation = true
				message = err.Error()
				break
			}
			time.Sleep(50 * time.Millisecond) // simulate interval
		}
		action := s.ExpectedAction
		if violation {
			action = "block"
		} else {
			action = "none"
		}
		results = append(results, ResourceAgentPidsScenarioResult{
			ScenarioName:  s.Name,
			ConfigMaxPids: cfg.MaxPids,
			ConfigBurst:   cfg.BurstLimit,
			Violation:     violation,
			Action:        action,
			Message:       message,
		})
	}
	return results, nil
}

// NewDefaultPidsConfig_028 returns a reasonable default PIDS configuration.
func NewDefaultPidsConfig_028() *ResourceAgentPidsConfig {
	return &ResourceAgentPidsConfig{
		MaxPids:                100,
		BurstLimit:             10,
		CheckInterval:          1 * time.Second,
		EnableForkBombDetection: true,
		ForkBombRateThreshold:  50.0,
		ForkBombWindow:         1 * time.Second,
		PenaltyDuration:        5 * time.Second,
	}
}

// NewStrictPidsConfig_028 returns a very restrictive PIDS configuration.
func NewStrictPidsConfig_028() *ResourceAgentPidsConfig {
	return &ResourceAgentPidsConfig{
		MaxPids:                20,
		BurstLimit:             5,
		CheckInterval:          100 * time.Millisecond,
		EnableForkBombDetection: true,
		ForkBombRateThreshold:  10.0,
		ForkBombWindow:         500 * time.Millisecond,
		PenaltyDuration:        30 * time.Second,
	}
}

// NewPermissivePidsConfig_028 returns a loose PIDS configuration suitable
// for development or non-critical workloads.
func NewPermissivePidsConfig_028() *ResourceAgentPidsConfig {
	return &ResourceAgentPidsConfig{
		MaxPids:                1000,
		BurstLimit:             200,
		CheckInterval:          5 * time.Second,
		EnableForkBombDetection: false,
		ForkBombRateThreshold:  0,
		ForkBombWindow:         0,
		PenaltyDuration:        0,
	}
}

// ComputeBurstLimit_028 returns the burst allowance based on MaxPids and a scaling factor.
func ComputeBurstLimit_028(maxPids int, scale float64) int {
	if scale <= 0 {
		return 0
	}
	burst := int(math.Ceil(float64(maxPids) * scale))
	if burst < 0 {
		return 0
	}
	return burst
}

// IsWithinPidsLimit_028 is a simple helper that checks if a given process count
// is within the configured limit (including burst).
func IsWithinPidsLimit_028(count int, cfg *ResourceAgentPidsConfig) bool {
	if cfg == nil {
		return true
	}
	limit := cfg.MaxPids
	if cfg.BurstLimit > 0 {
		limit += cfg.BurstLimit
	}
	return count <= limit
}
