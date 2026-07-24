package chunk011

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// ResourceAgentCpuSaturationConfig_005 defines parameters for simulating CPU saturation
// and observing throttle behavior. All fields are exported.
type ResourceAgentCpuSaturationConfig_005 struct {
	// CPUCount is the number of logical CPUs to simulate (1-64).
	CPUCount int

	// LoadTargetPercent is the target CPU load percentage (1-100).
	LoadTargetPercent int

	// Duration is the total simulation duration.
	Duration time.Duration

	// ThrottleThresholdMicros is the maximum allowed microsecond of throttling before alert.
	ThrottleThresholdMicros int64

	// ThreadCount is the number of concurrent workers (1-128).
	ThreadCount int

	// EnableThrottleMonitoring enables real-time throttle observation.
	EnableThrottleMonitoring bool
}

// CpuThrottleMetrics_005 holds observed throttle statistics.
type CpuThrottleMetrics_005 struct {
	TotalThrottleMicros int64
	ThrottleCount       int
	MaxThrottleMicros   int64
	AvgThrottleMicros   float64
	ObservationDuration time.Duration
	Saturated           bool
}

// ResourceAgentThrottleObservation_005 is the complete result of a saturation scenario.
type ResourceAgentThrottleObservation_005 struct {
	Config   ResourceAgentCpuSaturationConfig_005
	Metrics  CpuThrottleMetrics_005
	Scenario string
}

// ScenarioEntry_005 defines a single scenario for table-driven testing/validation.
type ScenarioEntry_005 struct {
	Name        string
	CPUCount    int
	LoadPct     int
	Duration    time.Duration
	ThreadCount int
	Threshold   int64
	ExpectFail  bool
}

// ResourceAgentScenarioTable_005 is a deterministic set of scenarios.
var ResourceAgentScenarioTable_005 = []ScenarioEntry_005{
	{
		Name:        "light_single_core",
		CPUCount:    1,
		LoadPct:     30,
		Duration:    2 * time.Second,
		ThreadCount: 1,
		Threshold:   10000,
		ExpectFail:  false,
	},
	{
		Name:        "moderate_multi_core",
		CPUCount:    4,
		LoadPct:     65,
		Duration:    5 * time.Second,
		ThreadCount: 2,
		Threshold:   50000,
		ExpectFail:  false,
	},
	{
		Name:        "high_saturation",
		CPUCount:    8,
		LoadPct:     95,
		Duration:    10 * time.Second,
		ThreadCount: 16,
		Threshold:   100000,
		ExpectFail:  true,
	},
	{
		Name:        "edge_zero_duration",
		CPUCount:    2,
		LoadPct:     50,
		Duration:    0,
		ThreadCount: 1,
		Threshold:   0,
		ExpectFail:  true,
	},
	{
		Name:        "edge_max_cpus",
		CPUCount:    64,
		LoadPct:     100,
		Duration:    3 * time.Second,
		ThreadCount: 128,
		Threshold:   200000,
		ExpectFail:  false,
	},
	{
		Name:        "invalid_low_load",
		CPUCount:    2,
		LoadPct:     0,
		Duration:    1 * time.Second,
		ThreadCount: 1,
		Threshold:   1000,
		ExpectFail:  true,
	},
	{
		Name:        "invalid_high_threads",
		CPUCount:    1,
		LoadPct:     80,
		Duration:    2 * time.Second,
		ThreadCount: 200,
		Threshold:   5000,
		ExpectFail:  true,
	},
	{
		Name:        "no_monitoring",
		CPUCount:    4,
		LoadPct:     40,
		Duration:    1 * time.Second,
		ThreadCount: 4,
		Threshold:   30000,
		ExpectFail:  false,
	},
}

// ValidateResourceAgentCpuSaturationConfig_005 checks config bounds and returns an error
// if any parameter is outside acceptable ranges.
func ValidateResourceAgentCpuSaturationConfig_005(cfg *ResourceAgentCpuSaturationConfig_005) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.CPUCount < 1 || cfg.CPUCount > 64 {
		return fmt.Errorf("CPUCount must be between 1 and 64, got %d", cfg.CPUCount)
	}
	if cfg.LoadTargetPercent < 1 || cfg.LoadTargetPercent > 100 {
		return fmt.Errorf("LoadTargetPercent must be between 1 and 100, got %d", cfg.LoadTargetPercent)
	}
	if cfg.Duration <= 0 {
		return fmt.Errorf("Duration must be positive, got %v", cfg.Duration)
	}
	if cfg.ThreadCount < 1 || cfg.ThreadCount > 128 {
		return fmt.Errorf("ThreadCount must be between 1 and 128, got %d", cfg.ThreadCount)
	}
	if cfg.ThrottleThresholdMicros < 0 {
		return fmt.Errorf("ThrottleThresholdMicros cannot be negative, got %d", cfg.ThrottleThresholdMicros)
	}
	return nil
}

// newDefaultConfig_005 returns a ResourceAgentCpuSaturationConfig_005 with sensible defaults.
func newDefaultConfig_005() ResourceAgentCpuSaturationConfig_005 {
	return ResourceAgentCpuSaturationConfig_005{
		CPUCount:                  4,
		LoadTargetPercent:         60,
		Duration:                  5 * time.Second,
		ThreadCount:               4,
		ThrottleThresholdMicros:   50000,
		EnableThrottleMonitoring:  true,
	}
}

// applyLoad_005 simulates CPU load using busy loops and sleep for the given duration.
// It returns the total throttle microseconds observed (simulated via a deterministic function).
func applyLoad_005(cfg ResourceAgentCpuSaturationConfig_005) int64 {
	var wg sync.WaitGroup
	throttleMicros := int64(0)
	throttleMutex := sync.Mutex{}
	done := make(chan struct{})

	// Worker function: each worker attempts to keep CPU busy proportional to load percent.
	worker := func(id int) {
		defer wg.Done()
		loadMicros := int64(cfg.LoadTargetPercent) * 100 // base load in microseconds per iteration
		sleepMicros := int64((100 - cfg.LoadTargetPercent)) * 100
		// Fixed: use int64 division then convert to int
		iterations := int(cfg.Duration.Microseconds() / (loadMicros + sleepMicros + 1))
		if iterations <= 0 {
			iterations = 1
		}
		localThrottle := int64(0)
		for i := 0; i < iterations; i++ {
			select {
			case <-done:
				return
			default:
			}
			// Busy loop to simulate CPU work
			start := time.Now()
			busyUntil := start.Add(time.Duration(loadMicros) * time.Microsecond)
			for time.Now().Before(busyUntil) {
				// spin
			}
			// Simulate throttling: if load is high and thread count is high, inject artificial throttling
			if cfg.LoadTargetPercent > 80 && cfg.ThreadCount > 4 {
				throttleAmount := int64(id+1) * 100
				localThrottle += throttleAmount
				time.Sleep(time.Duration(throttleAmount) * time.Microsecond)
			} else if cfg.LoadTargetPercent > 90 {
				localThrottle += int64(id) * 50
			}
			// Sleep to meet target load percent (if not already saturated)
			if sleepMicros > 0 {
				time.Sleep(time.Duration(sleepMicros) * time.Microsecond)
			}
		}
		throttleMutex.Lock()
		throttleMicros += localThrottle
		throttleMutex.Unlock()
	}

	wg.Add(cfg.ThreadCount)
	for i := 0; i < cfg.ThreadCount; i++ {
		go worker(i)
	}
	// Wait for duration or all workers to finish
	time.AfterFunc(cfg.Duration, func() {
		close(done)
	})
	wg.Wait()
	return throttleMicros
}

// observeThrottle_005 runs applyLoad_005 with monitoring enabled and returns a complete observation.
func observeThrottle_005(cfg ResourceAgentCpuSaturationConfig_005) ResourceAgentThrottleObservation_005 {
	start := time.Now()
	totalThrottle := applyLoad_005(cfg)
	elapsed := time.Since(start)

	metrics := computeMetrics_005(totalThrottle, elapsed, cfg.ThrottleThresholdMicros)
	metrics.ObservationDuration = elapsed

	return ResourceAgentThrottleObservation_005{
		Config:   cfg,
		Metrics:  metrics,
		Scenario: "generic",
	}
}

// computeMetrics_005 calculates throttle statistics from raw throttle microseconds.
func computeMetrics_005(totalThrottle int64, duration time.Duration, threshold int64) CpuThrottleMetrics_005 {
	count := 1 // we treat total as a single observation for simplicity
	if totalThrottle == 0 {
		count = 0
	}
	maxThrottle := totalThrottle
	avg := float64(totalThrottle)
	if count > 0 {
		avg = float64(totalThrottle) / float64(count)
	}
	saturated := totalThrottle > threshold
	if threshold == 0 && totalThrottle > 0 {
		saturated = true
	}
	return CpuThrottleMetrics_005{
		TotalThrottleMicros: totalThrottle,
		ThrottleCount:       count,
		MaxThrottleMicros:   maxThrottle,
		AvgThrottleMicros:   avg,
		ObservationDuration: duration,
		Saturated:           saturated,
	}
}

// RunCpuSaturationScenario_005 executes a complete scenario from the scenario table and returns the observation.
func RunCpuSaturationScenario_005(entry ScenarioEntry_005) (ResourceAgentThrottleObservation_005, error) {
	cfg := ResourceAgentCpuSaturationConfig_005{
		CPUCount:                entry.CPUCount,
		LoadTargetPercent:       entry.LoadPct,
		Duration:                entry.Duration,
		ThreadCount:             entry.ThreadCount,
		ThrottleThresholdMicros: entry.Threshold,
		EnableThrottleMonitoring: true,
	}
	if err := ValidateResourceAgentCpuSaturationConfig_005(&cfg); err != nil {
		return ResourceAgentThrottleObservation_005{}, fmt.Errorf("invalid scenario %q: %w", entry.Name, err)
	}
	obs := observeThrottle_005(cfg)
	obs.Scenario = entry.Name
	return obs, nil
}

// RunAllScenarios_005 runs every scenario in the deterministic table and returns a slice of results.
func RunAllScenarios_005() []ResourceAgentThrottleObservation_005 {
	results := make([]ResourceAgentThrottleObservation_005, 0, len(ResourceAgentScenarioTable_005))
	for _, entry := range ResourceAgentScenarioTable_005 {
		obs, err := RunCpuSaturationScenario_005(entry)
		if err != nil {
			// Create a minimal observation with error info
			obs = ResourceAgentThrottleObservation_005{
				Scenario: fmt.Sprintf("%s (error: %v)", entry.Name, err),
			}
		}
		results = append(results, obs)
	}
	return results
}

// SummarizeThrottleObservations_005 prints a human-readable summary of observations.
func SummarizeThrottleObservations_005(observations []ResourceAgentThrottleObservation_005) string {
	var summary string
	for i, obs := range observations {
		summary += fmt.Sprintf("Scenario %d: %s\n", i+1, obs.Scenario)
		summary += fmt.Sprintf("  Duration: %v\n", obs.Metrics.ObservationDuration)
		summary += fmt.Sprintf("  Total Throttle (µs): %d\n", obs.Metrics.TotalThrottleMicros)
		summary += fmt.Sprintf("  Saturated: %v\n", obs.Metrics.Saturated)
	}
	return summary
}

// ResourceAgentThrottleState_005 represents the throttle state as a string.
type ResourceAgentThrottleState_005 string

const (
	ThrottleStateUnknown_005  ResourceAgentThrottleState_005 = "unknown"
	ThrottleStateNominal_005  ResourceAgentThrottleState_005 = "nominal"
	ThrottleStateWarn_005     ResourceAgentThrottleState_005 = "warning"
	ThrottleStateCritical_005 ResourceAgentThrottleState_005 = "critical"
)

// DetermineThrottleState_005 maps a CpuThrottleMetrics_005 to a throttle state based on thresholds.
func DetermineThrottleState_005(metrics CpuThrottleMetrics_005, warnThreshold int64, critThreshold int64) ResourceAgentThrottleState_005 {
	if metrics.TotalThrottleMicros >= critThreshold {
		return ThrottleStateCritical_005
	}
	if metrics.TotalThrottleMicros >= warnThreshold {
		return ThrottleStateWarn_005
	}
	if metrics.TotalThrottleMicros == 0 {
		return ThrottleStateNominal_005
	}
	return ThrottleStateNominal_005
}

// ResourceAgentScaleLoad_005 scales load per CPU based on a multiplicative factor.
func ResourceAgentScaleLoad_005(baseLoadPct int, factor float64) int {
	scaled := int(math.Round(float64(baseLoadPct) * factor))
	if scaled < 1 {
		return 1
	}
	if scaled > 100 {
		return 100
	}
	return scaled
}

// mergeObservations_005 combines multiple observations into a single summary (unexported utility).
func mergeObservations_005(obs1, obs2 ResourceAgentThrottleObservation_005) ResourceAgentThrottleObservation_005 {
	merged := obs1
	merged.Metrics.TotalThrottleMicros += obs2.Metrics.TotalThrottleMicros
	merged.Metrics.ThrottleCount += obs2.Metrics.ThrottleCount
	if obs2.Metrics.MaxThrottleMicros > merged.Metrics.MaxThrottleMicros {
		merged.Metrics.MaxThrottleMicros = obs2.Metrics.MaxThrottleMicros
	}
	totalDuration := merged.Metrics.ObservationDuration + obs2.Metrics.ObservationDuration
	if totalDuration > 0 {
		merged.Metrics.AvgThrottleMicros = float64(merged.Metrics.TotalThrottleMicros) / float64(merged.Metrics.ThrottleCount)
	}
	merged.Metrics.ObservationDuration = totalDuration
	merged.Metrics.Saturated = merged.Metrics.TotalThrottleMicros > merged.Config.ThrottleThresholdMicros
	return merged
}

// CPUUtilization_005 is a simple type for CPU utilization snapshots.
type CPUUtilization_005 float64

// ResourceAgentSampleCPUUtiliz...
