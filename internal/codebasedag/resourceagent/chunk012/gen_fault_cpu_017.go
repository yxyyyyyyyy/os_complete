package chunk012

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ResourceAgentCPUSaturationScenario_017 describes a CPU saturation fault injection test case.
type ResourceAgentCPUSaturationScenario_017 struct {
	CoreCount         int
	LoadPercent       float64 // 0.0–1.0
	Duration          time.Duration
	ThrottleThreshold float64 // 0.0–1.0; fraction of total CPU capacity
	ExpectedThrottle  bool
}

// ResourceAgentThrottleObservation_017 captures measurement results for one scenario.
type ResourceAgentThrottleObservation_017 struct {
	Scenario      ResourceAgentCPUSaturationScenario_017
	ActualLoad    float64
	SaturatedCore int
	Throttled     bool
	ThrottleCount int
	Observation   time.Time
}

// ResourceAgentSaturationReport_017 aggregates multiple observations.
type ResourceAgentSaturationReport_017 struct {
	Scenarios    int
	Passed       int
	Failed       int
	Observations []ResourceAgentThrottleObservation_017
}

const (
	// DefaultCoreCount_017 is the default number of CPU cores used in scenarios.
	DefaultCoreCount_017 = 4
	// DefaultLoadPercent_017 is the default CPU load fraction.
	DefaultLoadPercent_017 = 0.75
	// DefaultThrottleThreshold_017 is the default throttle activation threshold.
	DefaultThrottleThreshold_017 = 0.85
	// DefaultSaturationDuration_017 is the default saturation duration.
	DefaultSaturationDuration_017 = 30 * time.Second
	// MinCoreCount_017 enforces a lower bound on core count.
	MinCoreCount_017 = 1
	// MaxCoreCount_017 enforces an upper bound on core count.
	MaxCoreCount_017 = 1024
	// MaxObservations_017 limits the number of stored observations.
	MaxObservations_017 = 1000
)

// NewDefaultCPUSaturationScenario_017 creates a default scenario with safe parameters.
func NewDefaultCPUSaturationScenario_017() ResourceAgentCPUSaturationScenario_017 {
	return ResourceAgentCPUSaturationScenario_017{
		CoreCount:         DefaultCoreCount_017,
		LoadPercent:       DefaultLoadPercent_017,
		Duration:          DefaultSaturationDuration_017,
		ThrottleThreshold: DefaultThrottleThreshold_017,
		ExpectedThrottle:  false,
	}
}

// ValidateCPUSaturationConfig_017 validates all fields of a single scenario.
// Returns an error if any parameter is out of acceptable range.
func ValidateCPUSaturationConfig_017(cfg ResourceAgentCPUSaturationScenario_017) error {
	if cfg.CoreCount < MinCoreCount_017 || cfg.CoreCount > MaxCoreCount_017 {
		return fmt.Errorf("CoreCount %d out of range [%d, %d]", cfg.CoreCount, MinCoreCount_017, MaxCoreCount_017)
	}
	if cfg.LoadPercent < 0 || cfg.LoadPercent > 1.0 {
		return fmt.Errorf("LoadPercent %f not in [0,1]", cfg.LoadPercent)
	}
	if cfg.Duration <= 0 {
		return fmt.Errorf("Duration %v must be positive", cfg.Duration)
	}
	if cfg.ThrottleThreshold <= 0 || cfg.ThrottleThreshold > 1.0 {
		return fmt.Errorf("ThrottleThreshold %f not in (0,1]", cfg.ThrottleThreshold)
	}
	return nil
}

// ValidateCPUSaturationConfigs_017 validates a slice of scenarios.
func ValidateCPUSaturationConfigs_017(cfgs []ResourceAgentCPUSaturationScenario_017) error {
	if len(cfgs) == 0 {
		return errors.New("empty scenario list")
	}
	if len(cfgs) > MaxObservations_017 {
		return fmt.Errorf("too many scenarios (%d) max allowed is %d", len(cfgs), MaxObservations_017)
	}
	for i, cfg := range cfgs {
		if err := ValidateCPUSaturationConfig_017(cfg); err != nil {
			return fmt.Errorf("scenario index %d: %w", i, err)
		}
	}
	return nil
}

// SimulateCPUSaturation_017 deterministically computes the throttle observation
// for a given scenario without performing actual CPU work.
func SimulateCPUSaturation_017(cfg ResourceAgentCPUSaturationScenario_017) ResourceAgentThrottleObservation_017 {
	obs := ResourceAgentThrottleObservation_017{
		Scenario:    cfg,
		ActualLoad:  cfg.LoadPercent,
		Observation: time.Now().UTC().Truncate(time.Microsecond),
	}
	// The saturation algorithm: the load is distributed across cores.
	// A core is saturated if its share exceeds 1.0 / CoreCount of total capacity.
	coreShare := cfg.LoadPercent / float64(cfg.CoreCount)
	const perCoreCapacity = 1.0
	saturatedCores := 0
	if coreShare > perCoreCapacity {
		saturatedCores = cfg.CoreCount
	} else {
		// Number of saturated cores is the smallest integer such that load on those cores > threshold.
		// Distribute load equally; a core is saturated if its load > perCoreCapacity.
		// For deterministic simulation we cap at CoreCount.
		loadPerCore := cfg.LoadPercent / float64(cfg.CoreCount)
		needed := int(math.Ceil(loadPerCore * float64(cfg.CoreCount)))
		if needed > cfg.CoreCount {
			needed = cfg.CoreCount
		}
		saturatedCores = needed
		if loadPerCore > perCoreCapacity {
			saturatedCores = cfg.CoreCount
		}
	}
	obs.SaturatedCore = saturatedCores

	// Throttle condition: throttled if effective load exceeds threshold.
	effectiveLoad := cfg.LoadPercent // total load ratio
	obs.Throttled = effectiveLoad > cfg.ThrottleThreshold

	// Throttle count is the number of microseconds the system would have been above threshold.
	// Deterministic: floor((load - threshold) * 1000) for typical durations.
	if obs.Throttled {
		excess := effectiveLoad - cfg.ThrottleThreshold
		// Convert to microseconds proportional to duration.
		micro := int64(cfg.Duration.Microseconds())
		count := int(math.Floor(excess * float64(micro)))
		if count < 0 {
			count = 0
		}
		obs.ThrottleCount = count
	} else {
		obs.ThrottleCount = 0
	}
	return obs
}

// RunCPUSaturationTest_017 evaluates a single scenario and returns the observation.
func RunCPUSaturationTest_017(cfg ResourceAgentCPUSaturationScenario_017) (ResourceAgentThrottleObservation_017, error) {
	if err := ValidateCPUSaturationConfig_017(cfg); err != nil {
		return ResourceAgentThrottleObservation_017{}, err
	}
	return SimulateCPUSaturation_017(cfg), nil
}

// RunCPUSaturationSuite_017 runs multiple scenarios and collects observations.
func RunCPUSaturationSuite_017(cfgs []ResourceAgentCPUSaturationScenario_017) (ResourceAgentSaturationReport_017, error) {
	if err := ValidateCPUSaturationConfigs_017(cfgs); err != nil {
		return ResourceAgentSaturationReport_017{}, err
	}
	report := ResourceAgentSaturationReport_017{
		Scenarios:    len(cfgs),
		Observations: make([]ResourceAgentThrottleObservation_017, 0, len(cfgs)),
	}
	for _, cfg := range cfgs {
		obs, _ := RunCPUSaturationTest_017(cfg)
		report.Observations = append(report.Observations, obs)
		if obs.Throttled == cfg.ExpectedThrottle {
			report.Passed++
		} else {
			report.Failed++
		}
	}
	return report, nil
}

// AverageLoad_017 computes the mean actual load from a slice of observations.
func AverageLoad_017(obs []ResourceAgentThrottleObservation_017) float64 {
	if len(obs) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, o := range obs {
		sum += o.ActualLoad
	}
	return sum / float64(len(obs))
}

// CountThrottled_017 returns how many observations report throttling.
func CountThrottled_017(obs []ResourceAgentThrottleObservation_017) int {
	count := 0
	for _, o := range obs {
		if o.Throttled {
			count++
		}
	}
	return count
}

// SortObservationsByActualLoad_017 sorts the observations in ascending order of load.
func SortObservationsByActualLoad_017(obs []ResourceAgentThrottleObservation_017) {
	sort.Slice(obs, func(i, j int) bool {
		return obs[i].ActualLoad < obs[j].ActualLoad
	})
}

// CPUSaturationTable_017 returns a deterministic set of test scenarios.
// These are designed to cover boundary conditions and common cases.
func CPUSaturationTable_017() []ResourceAgentCPUSaturationScenario_017 {
	// Deterministic table – no random values.
	return []ResourceAgentCPUSaturationScenario_017{
		// 1 core, low load
		{CoreCount: 1, LoadPercent: 0.1, Duration: 10 * time.Second, ThrottleThreshold: 0.9, ExpectedThrottle: false},
		// 1 core, high load but under threshold
		{CoreCount: 1, LoadPercent: 0.85, Duration: 5 * time.Second, ThrottleThreshold: 0.9, ExpectedThrottle: false},
		// 1 core, load exceeds threshold
		{CoreCount: 1, LoadPercent: 0.95, Duration: 2 * time.Second, ThrottleThreshold: 0.9, ExpectedThrottle: true},
		// 2 cores, moderate load
		{CoreCount: 2, LoadPercent: 0.5, Duration: 30 * time.Second, ThrottleThreshold: 0.8, ExpectedThrottle: false},
		// 2 cores, load exceeds threshold
		{CoreCount: 2, LoadPercent: 0.9, Duration: 15 * time.Second, ThrottleThreshold: 0.75, ExpectedThrottle: true},
		// 4 cores, high load but threshold high
		{CoreCount: 4, LoadPercent: 0.9, Duration: 20 * time.Second, ThrottleThreshold: 0.95, ExpectedThrottle: false},
		// 4 cores, load just above threshold
		{CoreCount: 4, LoadPercent: 0.85, Duration: 10 * time.Second, ThrottleThreshold: 0.84, ExpectedThrottle: true},
		// 8 cores, uniform distribution
		{CoreCount: 8, LoadPercent: 0.5, Duration: 60 * time.Second, ThrottleThreshold: 0.5, ExpectedThrottle: false},
		// 16 cores, near-threshold
		{CoreCount: 16, LoadPercent: 0.799, Duration: 5 * time.Second, ThrottleThreshold: 0.8, ExpectedThrottle: false},
		// 32 cores, overload
		{CoreCount: 32, LoadPercent: 0.99, Duration: 1 * time.Second, ThrottleThreshold: 0.5, ExpectedThrottle: true},
		// 64 cores, full load
		{CoreCount: 64, LoadPercent: 1.0, Duration: 2 * time.Second, ThrottleThreshold: 0.99, ExpectedThrottle: true},
		// 128 cores, threshold at 1.0 (never throttle)
		{CoreCount: 128, LoadPercent: 0.5, Duration: 3 * time.Second, ThrottleThreshold: 1.0, ExpectedThrottle: false},
		// 256 cores, high threshold, high load
		{CoreCount: 256, LoadPercent: 0.95, Duration: 100 * time.Millisecond, ThrottleThreshold: 0.9, ExpectedThrottle: true},
		// 512 cores, low load, low threshold
		{CoreCount: 512, LoadPercent: 0.05, Duration: 10 * time.Second, ThrottleThreshold: 0.1, ExpectedThrottle: false},
		// 1024 cores, extreme
		{CoreCount: 1024, LoadPercent: 0.2, Duration: 1 * time.Minute, ThrottleThreshold: 0.25, ExpectedThrottle: false},
	}
}

// MultipliedCPUSaturationTable_017 returns scenarios derived from the default table
// with a multiplicative factor applied to load and threshold (clamped to [0,1]).
func MultipliedCPUSaturationTable_017(factor float64) []ResourceAgentCPUSaturationScenario_017 {
	base := CPUSaturationTable_017()
	mul := factor
	if mul < 0 {
		mul = 0
	}
	result := make([]ResourceAgentCPUSaturationScenario_017, len(base))
	for i, s := range base {
		newLoad := s.LoadPercent * mul
		if newLoad > 1.0 {
			newLoad = 1.0
		}
		newThresh := s.ThrottleThreshold * mul
		if newThresh > 1.0 {
			newThresh = 1.0
		}
		result[i] = ResourceAgentCPUSaturationScenario_017{
			CoreCount:         s.CoreCount,
			LoadPercent:       newLoad,
			Duration:          s.Duration,
			ThrottleThreshold: newThresh,
			ExpectedThrottle:  newLoad > newThresh,
		}
	}
	return result
}

// FilterThrottledScenarios_017 returns only those scenarios that would trigger throttling
// under the given threshold.
func FilterThrottledScenarios_017(cfgs []ResourceAgentCPUSaturationScenario_017) []ResourceAgentCPUSaturationScenario_017 {
	var out []ResourceAgentCPUSaturationScenario_017
	for _, c := range cfgs {
		if c.LoadPercent > c.ThrottleThreshold {
			out = append(out, c)
		}
	}
	return out
}

// MergeObservations_017 combines two observation slices into one, deduplicating by scenario identity.
func MergeObservations_017(a, b []ResourceAgentThrottleObservation_017) []ResourceAgentThrottleObservation_017 {
	seen := make(map[ResourceAgentCPUSaturationScenario_017]bool)
	var merged []ResourceAgentThrottleObservation_017
	for _, o := range a {
		if !seen[o.Scenario] {
			seen[o.Scenario] = true
			merged = append(merged, o)
		}
	}
	for _, o := range b {
		if !seen[o.Scenario] {
			seen[o.Scenario] = true
			merged = append(merged, o)
		}
	}
	return merged
}

// MaxThrottleCount_017 returns the highest throttle count from a set of observations.
func MaxThrottleCount_017(obs []ResourceAgentThrottleObservation_017) int {
	max := 0
	for _, o := range obs {
		if o.ThrottleCount > max {
			max = o.ThrottleCount
		}
	}
	return max
}

// MinCoreCountInScenarios_017 finds the smallest CoreCount among a list of scenarios.
func MinCoreCountInScenarios_017(cfgs []ResourceAgentCPUSaturationScenario_017) int {
	if len(cfgs) == 0 {
		return 0
	}
	min := cfgs[0].CoreCount
	for _, c := range cfgs[1:] {
		if c.CoreCount < min {
			min = c.CoreCount
		}
	}
	return min
}

// MaxCoreCountInScenarios_017 finds the largest CoreCount among a list of scenarios.
func MaxCoreCountInScenarios_017(cfgs []ResourceAgentCPUSaturationScenario_017) int {
	if len(cfgs) == 0 {
		return 0
	}
	max := cfgs[0].CoreCount
	for _, c := range cfgs[1:] {
		if c.CoreCount > max {
			max = c.CoreCount
		}
	}
	return max
}

// ThrottleFraction_017 returns the fraction of observations that were throttled.
func ThrottleFraction_017(obs []ResourceAgentThrottleObservation_017) float64 {
	if len(obs) == 0 {
		return 0.0
	}
	return float64(CountThrottled_017(obs)) / float64(len(obs))
}
