package chunk013

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// CPULoadPattern_029 defines the shape of CPU load over time.
type CPULoadPattern_029 int

const (
	// CPULoadPatternUndefined_029 is the zero value.
	CPULoadPatternUndefined_029 CPULoadPattern_029 = iota
	// CPULoadPatternSpike_029 applies a short, high-intensity burst of CPU usage.
	CPULoadPatternSpike_029
	// CPULoadPatternSustained_029 maintains a constant CPU usage level.
	CPULoadPatternSustained_029
	// CPULoadPatternPeriodic_029 oscillates between target and idle usage.
	CPULoadPatternPeriodic_029
	// CPULoadPatternRamp_029 gradually increases CPU usage from idle to target.
	CPULoadPatternRamp_029
	// CPULoadPatternRandom_029 varies load unpredictably within a range.
	CPULoadPatternRandom_029
)

// String returns the human-readable name of the pattern.
func (p CPULoadPattern_029) String() string {
	switch p {
	case CPULoadPatternSpike_029:
		return "spike"
	case CPULoadPatternSustained_029:
		return "sustained"
	case CPULoadPatternPeriodic_029:
		return "periodic"
	case CPULoadPatternRamp_029:
		return "ramp"
	case CPULoadPatternRandom_029:
		return "random"
	default:
		return "undefined"
	}
}

// CPUSaturationConfig_029 holds all parameters needed to define a CPU saturation scenario.
type CPUSaturationConfig_029 struct {
	// Pattern selects the load shape.
	Pattern CPULoadPattern_029
	// TargetLoadPercent is the desired CPU usage percentage (0-100).
	TargetLoadPercent float64
	// MinLoadPercent is used for periodic or random patterns (0-100).
	MinLoadPercent float64
	// Duration is the total length of the saturation period.
	Duration time.Duration
	// SpikeDuration is how long a spike lasts (only for spike pattern).
	SpikeDuration time.Duration
	// Period is the oscillation period for periodic pattern.
	Period time.Duration
	// Cores limits the number of CPU cores to saturate (0 means all available).
	Cores int
	// Granularity controls the time resolution of load adjustment (default 10ms).
	Granularity time.Duration
}

// ValidateCPUSaturationConfig_029 checks the configuration for common errors.
func ValidateCPUSaturationConfig_029(cfg *CPUSaturationConfig_029) error {
	if cfg == nil {
		return errors.New("config must not be nil")
	}
	if cfg.Pattern == CPULoadPatternUndefined_029 {
		return errors.New("pattern must be set to a valid value")
	}
	if cfg.TargetLoadPercent < 0 || cfg.TargetLoadPercent > 100 {
		return fmt.Errorf("target load %v must be between 0 and 100", cfg.TargetLoadPercent)
	}
	if cfg.MinLoadPercent < 0 || cfg.MinLoadPercent > 100 {
		return fmt.Errorf("min load %v must be between 0 and 100", cfg.MinLoadPercent)
	}
	if cfg.MinLoadPercent >= cfg.TargetLoadPercent {
		return fmt.Errorf("min load %v must be less than target load %v", cfg.MinLoadPercent, cfg.TargetLoadPercent)
	}
	if cfg.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	if cfg.SpikeDuration < 0 {
		return errors.New("spike duration must be non-negative")
	}
	if cfg.Period < 0 {
		return errors.New("period must be non-negative")
	}
	if cfg.Cores < 0 {
		return errors.New("cores must be non-negative")
	}
	if cfg.Granularity < 0 {
		return errors.New("granularity must be non-negative")
	}
	if cfg.Granularity == 0 {
		cfg.Granularity = 10 * time.Millisecond // default
	}
	// pattern-specific checks
	switch cfg.Pattern {
	case CPULoadPatternSpike_029:
		if cfg.SpikeDuration == 0 {
			return errors.New("spike pattern requires spike_duration > 0")
		}
		if cfg.SpikeDuration > cfg.Duration {
			return errors.New("spike duration cannot exceed total duration")
		}
	case CPULoadPatternPeriodic_029:
		if cfg.Period == 0 {
			return errors.New("periodic pattern requires period > 0")
		}
		if cfg.Duration%cfg.Period != 0 {
			// warn but not mandatory
		}
	case CPULoadPatternRandom_029:
		if cfg.MinLoadPercent == 0 && cfg.TargetLoadPercent == 100 {
			// acceptable
		}
	}
	return nil
}

// ThrottleObservation_029 captures statistics about CPU throttling observed during an experiment.
type ThrottleObservation_029 struct {
	// TotalThrottledDuration is the cumulative time the CPU was throttled.
	TotalThrottledDuration time.Duration
	// ThrottleEvents is the number of times throttling activated.
	ThrottleEvents int
	// AverageThrottleDepth is the average percentage of throttling (0-100).
	AverageThrottleDepth float64
	// MaxThrottleDepth is the highest observed throttling percentage.
	MaxThrottleDepth float64
	// MinThrottleDepth is the lowest observed throttling percentage (non-zero).
	MinThrottleDepth float64
	// SampleCount is the number of measurement samples.
	SampleCount int
	// Timestamps for start and end of observation.
	StartTime time.Time
	EndTime   time.Time
}

// ResourceAgentThrottleCompare_029 returns a similarity score (0.0 = identical, higher = more divergent).
func ResourceAgentThrottleCompare_029(a, b ThrottleObservation_029) float64 {
	diff := 0.0
	diff += math.Abs(a.TotalThrottledDuration.Seconds() - b.TotalThrottledDuration.Seconds())
	diff += math.Abs(float64(a.ThrottleEvents - b.ThrottleEvents))
	diff += math.Abs(a.AverageThrottleDepth - b.AverageThrottleDepth)
	diff += math.Abs(a.MaxThrottleDepth - b.MaxThrottleDepth)
	diff += math.Abs(a.MinThrottleDepth - b.MinThrottleDepth)
	return diff
}

// CPUSaturationScenario_029 bundles a config, expected observation, and metadata.
type CPUSaturationScenario_029 struct {
	Name        string
	Description string
	Config      CPUSaturationConfig_029
	Expected    ThrottleObservation_029
	Tags        []string
}

// ResourceAgentNewScenarioFromConfig_029 creates a scenario from a validated config.
func ResourceAgentNewScenarioFromConfig_029(name string, cfg *CPUSaturationConfig_029) (*CPUSaturationScenario_029, error) {
	if err := ValidateCPUSaturationConfig_029(cfg); err != nil {
		return nil, fmt.Errorf("invalid config for scenario %q: %w", name, err)
	}
	return &CPUSaturationScenario_029{
		Name:   name,
		Config: *cfg,
	}, nil
}

// PredefinedScenarios_029 is a deterministic table mapping scenario names to their definitions.
var PredefinedScenarios_029 = map[string]CPUSaturationScenario_029{
	"spike_heavy": {
		Name:        "spike_heavy",
		Description: "A short, intense CPU spike to trigger immediate throttling.",
		Config: CPUSaturationConfig_029{
			Pattern:          CPULoadPatternSpike_029,
			TargetLoadPercent: 100,
			Duration:         5 * time.Second,
			SpikeDuration:    3 * time.Second,
			Cores:            1,
			Granularity:      10 * time.Millisecond,
		},
		Expected: ThrottleObservation_029{
			TotalThrottledDuration: 2 * time.Second,
			ThrottleEvents:         1,
			AverageThrottleDepth:   50.0,
			MaxThrottleDepth:       80.0,
			MinThrottleDepth:       10.0,
			SampleCount:            500,
		},
		Tags: []string{"cpu", "throttle", "spike"},
	},
	"sustained_moderate": {
		Name:        "sustained_moderate",
		Description: "Sustained 70% CPU load for 30 seconds to measure steady-state throttling.",
		Config: CPUSaturationConfig_029{
			Pattern:          CPULoadPatternSustained_029,
			TargetLoadPercent: 70,
			Duration:         30 * time.Second,
			Cores:            2,
			Granularity:      100 * time.Millisecond,
		},
		Expected: ThrottleObservation_029{
			TotalThrottledDuration: 5 * time.Second,
			ThrottleEvents:         3,
			AverageThrottleDepth:   30.0,
			MaxThrottleDepth:       45.0,
			MinThrottleDepth:       15.0,
			SampleCount:            300,
		},
		Tags: []string{"cpu", "throttle", "sustained"},
	},
	"periodic_wave": {
		Name:        "periodic_wave",
		Description: "Periodic load between 20% and 80% every 10 seconds for 1 minute.",
		Config: CPUSaturationConfig_029{
			Pattern:          CPULoadPatternPeriodic_029,
			TargetLoadPercent: 80,
			MinLoadPercent:    20,
			Duration:          60 * time.Second,
			Period:            10 * time.Second,
			Cores:             4,
			Granularity:       50 * time.Millisecond,
		},
		Expected: ThrottleObservation_029{
			TotalThrottledDuration: 12 * time.Second,
			ThrottleEvents:         6,
			AverageThrottleDepth:   35.0,
			MaxThrottleDepth:       55.0,
			MinThrottleDepth:       5.0,
			SampleCount:            1200,
		},
		Tags: []string{"cpu", "throttle", "periodic"},
	},
	"ramp_up": {
		Name:        "ramp_up",
		Description: "Gradual increase from 10% to 90% over 20 seconds.",
		Config: CPUSaturationConfig_029{
			Pattern:          CPULoadPatternRamp_029,
			TargetLoadPercent: 90,
			MinLoadPercent:    10,
			Duration:          20 * time.Second,
			Cores:             1,
			Granularity:       20 * time.Millisecond,
		},
		Expected: ThrottleObservation_029{
			TotalThrottledDuration: 3 * time.Second,
			ThrottleEvents:         4,
			AverageThrottleDepth:   25.0,
			MaxThrottleDepth:       60.0,
			MinThrottleDepth:       2.0,
			SampleCount:            1000,
		},
		Tags: []string{"cpu", "throttle", "ramp"},
	},
	"random_burst": {
		Name:        "random_burst",
		Description: "Random load between 0% and 100% for 15 seconds.",
		Config: CPUSaturationConfig_029{
			Pattern:          CPULoadPatternRandom_029,
			TargetLoadPercent: 100,
			MinLoadPercent:    0,
			Duration:          15 * time.Second,
			Cores:             2,
			Granularity:       50 * time.Millisecond,
		},
		Expected: ThrottleObservation_029{
			TotalThrottledDuration: 7 * time.Second,
			ThrottleEvents:         10,
			AverageThrottleDepth:   40.0,
			MaxThrottleDepth:       90.0,
			MinThrottleDepth:       1.0,
			SampleCount:            300,
		},
		Tags: []string{"cpu", "throttle", "random"},
	},
}

// ResourceAgentLookupScenario_029 returns a copy of the scenario by name.
func ResourceAgentLookupScenario_029(name string) (*CPUSaturationScenario_029, error) {
	sc, ok := PredefinedScenarios_029[name]
	if !ok {
		keys := make([]string, 0, len(PredefinedScenarios_029))
		for k := range PredefinedScenarios_029 {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("scenario %q not found; available: %s", name, strings.Join(keys, ", "))
	}
	return &sc, nil
}

// ResourceAgentListScenarios_029 returns all predefined scenario names.
func ResourceAgentListScenarios_029() []string {
	names := make([]string, 0, len(PredefinedScenarios_029))
	for k := range PredefinedScenarios_029 {
		names = append(names, k)
	}
	return names
}

// ResourceAgentGenerateLoadProfile_029 produces a time-ordered list of load targets based on configuration.
func ResourceAgentGenerateLoadProfile_029(cfg CPUSaturationConfig_029) ([]time.Duration, []float64, error) {
	if err := ValidateCPUSaturationConfig_029(&cfg); err != nil {
		return nil, nil, err
	}
	step := cfg.Granularity
	steps := int(cfg.Duration / step)
	offsets := make([]time.Duration, steps)
	targets := make([]float64, steps)

	totalSteps := float64(steps)
	switch cfg.Pattern {
	case CPULoadPatternSustained_029:
		for i := 0; i < steps; i++ {
			offsets[i] = time.Duration(i) * step
			targets[i] = cfg.TargetLoadPercent
		}
	case CPULoadPatternSpike_029:
		spikeEndSteps := int(cfg.SpikeDuration / step)
		for i := 0; i < steps; i++ {
			offsets[i] = time.Duration(i) * step
			if i < spikeEndSteps {
				targets[i] = cfg.TargetLoadPercent
			} else {
				targets[i] = cfg.MinLoadPercent
			}
		}
	case CPULoadPatternPeriodic_029:
		periodSteps := int(cfg.Period / step)
		if periodSteps == 0 {
			return nil, nil, errors.New("period too small for granularity")
		}
		for i := 0; i < steps; i++ {
			offsets[i] = time.Duration(i) * step
			pos := float64(i%periodSteps) / float64(periodSteps)
			// sine wave between min and target
			amp := (cfg.TargetLoadPercent - cfg.MinLoadPercent) / 2.0
			mid := (cfg.TargetLoadPercent + cfg.MinLoadPercent) / 2.0
			targets[i] = mid + amp*math.Sin(2.0*math.Pi*pos)
		}
	case CPULoadPatternRamp_029:
		for i := 0; i < steps; i++ {
			offsets[i] = time.Duration(i) * step
			frac := float64(i) / totalSteps
			targets[i] = cfg.MinLoadPercent + frac*(cfg.TargetLoadPercent-cfg.MinLoadPercent)
		}
	case CPULoadPatternRandom_029:
		// Use a deterministic pseudo-random generator for reproducibility.
		rng := rand.New(rand.NewSource(42))
		lower := cfg.MinLoadPercent
		upper := cfg.TargetLoadPercent
		range_ := upper - lower
		for i := 0; i < steps; i++ {
			offsets[i] = time.Duration(i) * step
			targets[i] = lower + rng.Float64()*range_
		}
	}
	return offsets, targets, nil
}
