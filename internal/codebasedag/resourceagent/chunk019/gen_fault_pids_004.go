package chunk019

import (
	"errors"
	"fmt"
)

// PidsMaxConfig_004 holds configuration for pids.max containment.
type PidsMaxConfig_004 struct {
	// DefaultLimit is the baseline pids.max value.
	DefaultLimit int

	// HardLimit is the absolute maximum beyond which containment is forced.
	HardLimit int

	// SoftLimit is the threshold at which warning actions are triggered.
	SoftLimit int

	// BurstMultiplier scales the default limit during fork bursts.
	BurstMultiplier float64

	// RecoveryDelaySec is seconds to wait before reducing limits after containment.
	RecoveryDelaySec int

	// ComplianceMode enables strict enforcement (vs advisory).
	ComplianceMode bool
}

// ForkBombScenario_004 enumerates known fork-bomb patterns.
type ForkBombScenario_004 int

const (
	// ForkBombScenarioUndefined_004 is the zero value.
	ForkBombScenarioUndefined_004 ForkBombScenario_004 = iota

	// ForkBombScenarioSimple_004 simulates rapid fork() without exec.
	ForkBombScenarioSimple_004

	// ForkBombScenarioExecLoop_004 forks and executes a short-lived command in a loop.
	ForkBombScenarioExecLoop_004

	// ForkBombScenarioZombiePile_004 creates many zombies to exhaust PID space.
	ForkBombScenarioZombiePile_004

	// ForkBombScenarioSubCgroup_004 attacks via nested cgroup forks.
	ForkBombScenarioSubCgroup_004

	// ForkBombScenarioContainerEscape_004 attempts to write pids.max to escape.
	ForkBombScenarioContainerEscape_004
)

// scenarioName maps scenario to human readable name.
var scenarioName_004 = map[ForkBombScenario_004]string{
	ForkBombScenarioUndefined_004:       "undefined",
	ForkBombScenarioSimple_004:          "simple-fork",
	ForkBombScenarioExecLoop_004:        "exec-loop",
	ForkBombScenarioZombiePile_004:      "zombie-pile",
	ForkBombScenarioSubCgroup_004:       "sub-cgroup",
	ForkBombScenarioContainerEscape_004: "container-escape",
}

// String returns a human-readable scenario name.
func (s ForkBombScenario_004) String() string {
	if name, ok := scenarioName_004[s]; ok {
		return name
	}
	return fmt.Sprintf("scenario(%d)", int(s))
}

// ContainmentCheckResult_004 holds the outcome of a pids.max containment check.
type ContainmentCheckResult_004 struct {
	// Exceeded is true if the PID count surpassed the soft or hard limit.
	Exceeded bool

	// CurrentPids is the number of PIDs at check time.
	CurrentPids int

	// HardLimitReached indicates the hard limit was hit.
	HardLimitReached bool

	// SoftLimitReached indicates the soft limit was hit.
	SoftLimitReached bool

	// Scenario detected.
	Scenario ForkBombScenario_004

	// RecommendedAction is a brief imperative string.
	RecommendedAction string

	// ProposedNewLimit is the pids.max value to apply (0 means no change).
	ProposedNewLimit int
}

// ContainmentStrategy_004 defines a remedial action for a pids.max scenario.
type ContainmentStrategy_004 int

const (
	// ContainmentStrategyNone_004 indicates no action.
	ContainmentStrategyNone_004 ContainmentStrategy_004 = iota

	// ContainmentStrategyReduce_004 decreases pids.max to the soft limit.
	ContainmentStrategyReduce_004

	// ContainmentStrategyHalt_004 stops all new fork operations.
	ContainmentStrategyHalt_004

	// ContainmentStrategyIsolate_004 moves the process into a separate cgroup.
	ContainmentStrategyIsolate_004

	// ContainmentStrategyKill_004 sends SIGKILL to the offending processes.
	ContainmentStrategyKill_004

	// ContainmentStrategyEscalate_004 reports upward and halts the container.
	ContainmentStrategyEscalate_004
)

// strategyName_004 maps strategy to a readable name.
var strategyName_004 = map[ContainmentStrategy_004]string{
	ContainmentStrategyNone_004:     "none",
	ContainmentStrategyReduce_004:   "reduce",
	ContainmentStrategyHalt_004:     "halt",
	ContainmentStrategyIsolate_004:  "isolate",
	ContainmentStrategyKill_004:     "kill",
	ContainmentStrategyEscalate_004: "escalate",
}

// String returns the strategy name.
func (s ContainmentStrategy_004) String() string {
	if name, ok := strategyName_004[s]; ok {
		return name
	}
	return fmt.Sprintf("strategy(%d)", int(s))
}

// DefaultPidsMaxProfiles_004 is a table of sensible pids.max profiles
// for different runtime environments. Key is a short descriptor.
var DefaultPidsMaxProfiles_004 = map[string]PidsMaxConfig_004{
	"low-resource":   {DefaultLimit: 256, HardLimit: 512, SoftLimit: 200, BurstMultiplier: 2.0, RecoveryDelaySec: 10, ComplianceMode: true},
	"high-resource":  {DefaultLimit: 4096, HardLimit: 8192, SoftLimit: 3500, BurstMultiplier: 1.5, RecoveryDelaySec: 30, ComplianceMode: false},
	"critical":       {DefaultLimit: 1024, HardLimit: 2048, SoftLimit: 900, BurstMultiplier: 1.2, RecoveryDelaySec: 60, ComplianceMode: true},
	"development":    {DefaultLimit: 10240, HardLimit: 20480, SoftLimit: 8000, BurstMultiplier: 3.0, RecoveryDelaySec: 5, ComplianceMode: false},
	"batch":          {DefaultLimit: 128, HardLimit: 256, SoftLimit: 100, BurstMultiplier: 4.0, RecoveryDelaySec: 15, ComplianceMode: true},
}

// ScenarioDefinitions_004 contains deterministic scenario parameters.
type ScenarioDefinition_004 struct {
	// TypicalForkRate is the expected fork rate in forks/second for this scenario.
	TypicalForkRate int

	// PIDFootprint is the average number of PIDs consumed per fork.
	PIDFootprint int

	// TransientDurationSec is how long the scenario typically lasts.
	TransientDurationSec int

	// Priority determines escalation urgency (higher = more urgent).
	Priority int
}

// scenarioDefinitionsTable_004 maps each scenario to its definition.
var scenarioDefinitionsTable_004 = map[ForkBombScenario_004]ScenarioDefinition_004{
	ForkBombScenarioSimple_004:          {TypicalForkRate: 1000, PIDFootprint: 1, TransientDurationSec: 30, Priority: 3},
	ForkBombScenarioExecLoop_004:        {TypicalForkRate: 500, PIDFootprint: 2, TransientDurationSec: 45, Priority: 4},
	ForkBombScenarioZombiePile_004:      {TypicalForkRate: 200, PIDFootprint: 5, TransientDurationSec: 120, Priority: 5},
	ForkBombScenarioSubCgroup_004:       {TypicalForkRate: 800, PIDFootprint: 1, TransientDurationSec: 60, Priority: 4},
	ForkBombScenarioContainerEscape_004: {TypicalForkRate: 10, PIDFootprint: 10, TransientDurationSec: 10, Priority: 5},
}

// ContainmentActions_004 is a deterministic table mapping scenario and strategy to action description.
var ContainmentActions_004 = map[ForkBombScenario_004]map[ContainmentStrategy_004]string{
	ForkBombScenarioSimple_004: {
		ContainmentStrategyReduce_004: "reduce pids.max to soft limit; monitor fork rate",
		ContainmentStrategyHalt_004:   "suspend all new forks until rate decreases",
		ContainmentStrategyKill_004:   "kill the top forking process tree",
	},
	ForkBombScenarioExecLoop_004: {
		ContainmentStrategyIsolate_004:  "move process into dedicated cgroup with hard cap",
		ContainmentStrategyEscalate_004: "report to security agent and freeze container",
	},
	ForkBombScenarioZombiePile_004: {
		ContainmentStrategyHalt_004:   "pause container and drain zombie processes",
		ContainmentStrategyKill_004:   "SIGKILL zombie parent processes",
	},
	ForkBombScenarioSubCgroup_004: {
		ContainmentStrategyReduce_004:  "recursively apply pids.max to all child cgroups",
		ContainmentStrategyIsolate_004: "deny cgroup creation for the offending process",
	},
	ForkBombScenarioContainerEscape_004: {
		ContainmentStrategyKill_004:   "immediately kill the container PID 1",
		ContainmentStrategyHalt_004:   "freeze the container and disable pids.max write access",
	},
}

// ValidatePidsMaxConfig_004 validates a PidsMaxConfig_004 and returns an error if invalid.
func ValidatePidsMaxConfig_004(cfg *PidsMaxConfig_004) error {
	if cfg == nil {
		return errors.New("pids max config cannot be nil")
	}
	if cfg.DefaultLimit <= 0 {
		return errors.New("default limit must be positive")
	}
	if cfg.HardLimit <= 0 {
		return errors.New("hard limit must be positive")
	}
	if cfg.SoftLimit <= 0 {
		return errors.New("soft limit must be positive")
	}
	if cfg.DefaultLimit >= cfg.HardLimit {
		return errors.New("default limit must be less than hard limit")
	}
	if cfg.SoftLimit >= cfg.HardLimit {
		return errors.New("soft limit must be less than hard limit")
	}
	if cfg.SoftLimit >= cfg.DefaultLimit {
		return errors.New("soft limit must be less than default limit")
	}
	if cfg.BurstMultiplier <= 0 {
		return errors.New("burst multiplier must be positive")
	}
	if cfg.RecoveryDelaySec < 0 {
		return errors.New("recovery delay cannot be negative")
	}
	if cfg.ComplianceMode && cfg.DefaultLimit > cfg.HardLimit/2 {
		return errors.New("in compliance mode, default limit should not exceed half of hard limit")
	}
	return nil
}

// ResourceAgentNewPidsMaxConfig_004 creates a new PidsMaxConfig_004 with sensible defaults.
func ResourceAgentNewPidsMaxConfig_004() *PidsMaxConfig_004 {
	return &PidsMaxConfig_004{
		DefaultLimit:     1024,
		HardLimit:        4096,
		SoftLimit:        800,
		BurstMultiplier:  2.0,
		RecoveryDelaySec: 30,
		ComplianceMode:   false,
	}
}

// ResourceAgentForkBombContainmentCheck_004 performs a containment check based on configuration and current PID count.
// It returns a result and any validation error.
func ResourceAgentForkBombContainmentCheck_004(cfg *PidsMaxConfig_004, currentPids int, scenario ForkBombScenario_004) (*ContainmentCheckResult_004, error) {
	if err := ValidatePidsMaxConfig_004(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if scenario < ForkBombScenarioSimple_004 || scenario > ForkBombScenarioContainerEscape_004 {
		return nil, fmt.Errorf("unknown scenario: %v", scenario)
	}
	result := &ContainmentCheckResult_004{
		CurrentPids: currentPids,
		Scenario:    scenario,
	}
	def, ok := scenarioDefinitionsTable_004[scenario]
	if ok {
		// adjust soft limit based on scenario priority
		effectiveSoft := cfg.SoftLimit - (def.Priority * 10)
		if effectiveSoft < 10 {
			effectiveSoft = 10
		}
		if currentPids >= effectiveSoft {
			result.SoftLimitReached = true
		}
	} else {
		if currentPids >= cfg.SoftLimit {
			result.SoftLimitReached = true
		}
	}

	if currentPids >= cfg.HardLimit {
		result.HardLimitReached = true
		result.Exceeded = true
	} else if result.SoftLimitReached {
		// only exceed if soft limit is significantly breached
		result.Exceeded = true
	}

	if result.Exceeded {
		strategy := selectStrategy_004(scenario, result.HardLimitReached)
		result.RecommendedAction = "apply " + strategy.String()
		if action, ok := ContainmentActions_004[scenario][strategy]; ok {
			result.RecommendedAction = action
		}
		// propose a new limit: if hard limit reached, go to soft limit; else go to effective soft limit less some margin
		proposed := cfg.DefaultLimit
		if result.HardLimitReached {
			proposed = cfg.SoftLimit
		} else if result.SoftLimitReached {
			proposed = cfg.SoftLimit - 20
		}
		if proposed < 1 {
			proposed = 1
		}
		result.ProposedNewLimit = proposed
	} else {
		result.RecommendedAction = "no immediate action required"
	}

	return result, nil
}

// selectStrategy_004 picks a strategy based on scenario and severity.
func selectStrategy_004(scenario ForkBombScenario_004, hardLimitReached bool) ContainmentStrategy_004 {
	if hardLimitReached {
		switch scenario {
		case ForkBombScenarioContainerEscape_004:
			return ContainmentStrategyKill_004
		case ForkBombScenarioZombiePile_004:
			return ContainmentStrategyHalt_004
		default:
			return ContainmentStrategyReduce_004
		}
	}
	switch scenario {
	case ForkBombScenarioSubCgroup_004:
		return ContainmentStrategyIsolate_004
	case ForkBombScenarioExecLoop_004:
		return ContainmentStrategyEscalate_004
	default:
		return ContainmentStrategyReduce_004
	}
}

// ResourceAgentApplyContainmentStrategy_004 applies a containment strategy to a pids.max limit value.
// It modifies the integer pointer pidsMax to the new limit if applicable.
func ResourceAgentApplyContainmentStrategy_004(strategy ContainmentStrategy_004, pidsMax *int) error {
	if pidsMax == nil {
		return errors.New("pidsMax pointer is nil")
	}
	switch strategy {
	case ContainmentStrategyNone_004:
		// no change
	case ContainmentStrategyReduce_004:
		if *pidsMax > 100 {
			*pidsMax = *pidsMax / 2
		} else {
			*pidsMax = 1
		}
	case ContainmentStrategyHalt_004:
		*pidsMax = 0
	case ContainmentStrategyIsolate_004:
		// isolation does not change the limit; strategy is applied externally
	case ContainmentStrategyKill_004:
		*pidsMax = 0
	case ContainmentStrategyEscalate_004:
		*pidsMax = *pidsMax / 4
		if *pidsMax < 1 {
			*pidsMax = 1
		}
	default:
		return fmt.Errorf("unknown containment strategy: %v", strategy)
	}
	return nil
}
