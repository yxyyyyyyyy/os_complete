package chunk016

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Memory Pressure Detection Configuration and Types
// ---------------------------------------------------------------------------

// ResourceAgentMemoryPressureSeverity_015 defines the severity level of memory pressure.
type ResourceAgentMemoryPressureSeverity_015 int

const (
	MemoryPressureNone_015     ResourceAgentMemoryPressureSeverity_015 = 0
	MemoryPressureWarning_015  ResourceAgentMemoryPressureSeverity_015 = 1
	MemoryPressureCritical_015 ResourceAgentMemoryPressureSeverity_015 = 2
	MemoryPressureFatal_015    ResourceAgentMemoryPressureSeverity_015 = 3
)

// String returns a human-readable representation of the severity.
func (s ResourceAgentMemoryPressureSeverity_015) String() string {
	switch s {
	case MemoryPressureNone_015:
		return "none"
	case MemoryPressureWarning_015:
		return "warning"
	case MemoryPressureCritical_015:
		return "critical"
	case MemoryPressureFatal_015:
		return "fatal"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// ResourceAgentMemoryPressureScenario_015 categorizes the type of memory fault.
type ResourceAgentMemoryPressureScenario_015 int

const (
	MemoryScenarioNone_015            ResourceAgentMemoryPressureScenario_015 = 0
	MemoryScenarioLeak_015            ResourceAgentMemoryPressureScenario_015 = 1
	MemoryScenarioBloat_015           ResourceAgentMemoryPressureScenario_015 = 2
	MemoryScenarioOOM_015             ResourceAgentMemoryPressureScenario_015 = 3
	MemoryScenarioSwapPressure_015    ResourceAgentMemoryPressureScenario_015 = 4
	MemoryScenarioCgroupThrottle_015  ResourceAgentMemoryPressureScenario_015 = 5
)

// String returns a human-readable scenario name.
func (s ResourceAgentMemoryPressureScenario_015) String() string {
	switch s {
	case MemoryScenarioNone_015:
		return "none"
	case MemoryScenarioLeak_015:
		return "memory-leak"
	case MemoryScenarioBloat_015:
		return "memory-bloat"
	case MemoryScenarioOOM_015:
		return "out-of-memory"
	case MemoryScenarioSwapPressure_015:
		return "swap-pressure"
	case MemoryScenarioCgroupThrottle_015:
		return "cgroup-throttle"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// ResourceAgentMemoryFaultConfig_015 holds user-configurable parameters for memory pressure detection.
type ResourceAgentMemoryFaultConfig_015 struct {
	// Enabled indicates whether memory pressure detection is active.
	Enabled bool

	// SampleInterval is the time between consecutive memory metric samples.
	SampleInterval time.Duration

	// HistorySize is the number of samples kept for trend analysis.
	HistorySize int

	// ThresholdWarning is the percentage of used memory (0-100) that triggers a warning.
	ThresholdWarning float64

	// ThresholdCritical is the percentage that triggers critical severity.
	ThresholdCritical float64

	// ThresholdFatal is the percentage that triggers fatal severity.
	ThresholdFatal float64

	// SwapUsageWarning is the percentage of swap used (0-100) for a warning.
	SwapUsageWarning float64

	// SwapUsageCritical is the percentage of swap used for critical.
	SwapUsageCritical float64

	// LeakRateMBPerSec is the sustained memory growth in MB/s that indicates a leak.
	LeakRateMBPerSec float64

	// LeakDurationSec is the minimum duration (in seconds) of sustained growth.
	LeakDurationSec int

	// BloatThresholdMB is the increase in RSS over baseline (MB) indicating bloat.
	BloatThresholdMB float64

	// OOMKillHistoryWindow is how far back to look for OOM kill events.
	OOMKillHistoryWindow time.Duration

	// CgroupThrottleThreshold is the number of memory throttle events per minute.
	CgroupThrottleThreshold int
}

// DefaultMemoryFaultConfig_015 returns a sensible default configuration.
func DefaultMemoryFaultConfig_015() ResourceAgentMemoryFaultConfig_015 {
	return ResourceAgentMemoryFaultConfig_015{
		Enabled:                 true,
		SampleInterval:          5 * time.Second,
		HistorySize:             60,
		ThresholdWarning:        70.0,
		ThresholdCritical:       85.0,
		ThresholdFatal:          95.0,
		SwapUsageWarning:        50.0,
		SwapUsageCritical:       80.0,
		LeakRateMBPerSec:        10.0,
		LeakDurationSec:         30,
		BloatThresholdMB:        200.0,
		OOMKillHistoryWindow:    2 * time.Minute,
		CgroupThrottleThreshold: 5,
	}
}

// ValidateMemoryFaultConfig_015 validates the configuration and returns an error if invalid.
func ValidateMemoryFaultConfig_015(cfg *ResourceAgentMemoryFaultConfig_015) error {
	if cfg == nil {
		return errors.New("memory fault config is nil")
	}
	if cfg.SampleInterval <= 0 {
		return errors.New("sample interval must be positive")
	}
	if cfg.HistorySize < 10 {
		return errors.New("history size must be at least 10")
	}
	if cfg.ThresholdWarning < 0 || cfg.ThresholdWarning > 100 {
		return fmt.Errorf("warning threshold out of range [0,100]: %f", cfg.ThresholdWarning)
	}
	if cfg.ThresholdCritical < 0 || cfg.ThresholdCritical > 100 {
		return fmt.Errorf("critical threshold out of range [0,100]: %f", cfg.ThresholdCritical)
	}
	if cfg.ThresholdFatal < 0 || cfg.ThresholdFatal > 100 {
		return fmt.Errorf("fatal threshold out of range [0,100]: %f", cfg.ThresholdFatal)
	}
	if cfg.ThresholdWarning >= cfg.ThresholdCritical {
		return errors.New("warning threshold must be less than critical threshold")
	}
	if cfg.ThresholdCritical >= cfg.ThresholdFatal {
		return errors.New("critical threshold must be less than fatal threshold")
	}
	if cfg.SwapUsageWarning < 0 || cfg.SwapUsageWarning > 100 {
		return fmt.Errorf("swap warning threshold out of range [0,100]: %f", cfg.SwapUsageWarning)
	}
	if cfg.SwapUsageCritical < 0 || cfg.SwapUsageCritical > 100 {
		return fmt.Errorf("swap critical threshold out of range [0,100]: %f", cfg.SwapUsageCritical)
	}
	if cfg.LeakRateMBPerSec <= 0 {
		return errors.New("leak rate must be positive")
	}
	if cfg.LeakDurationSec <= 0 {
		return errors.New("leak duration must be positive")
	}
	if cfg.BloatThresholdMB <= 0 {
		return errors.New("bloat threshold must be positive")
	}
	if cfg.OOMKillHistoryWindow <= 0 {
		return errors.New("OOM kill history window must be positive")
	}
	if cfg.CgroupThrottleThreshold <= 0 {
		return errors.New("cgroup throttle threshold must be positive")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Memory Metrics and Sample
// ---------------------------------------------------------------------------

// ResourceAgentMemorySample_015 represents a single snapshot of memory metrics.
type ResourceAgentMemorySample_015 struct {
	Timestamp           time.Time
	UsedPercent         float64
	UsedBytes           uint64
	TotalBytes          uint64
	SwapUsedBytes       uint64
	SwapTotalBytes      uint64
	RSSBytes            uint64
	OOMKills            int
	CgroupThrottleEvents int
}

// ResourceAgentMemoryFaultState_015 holds the runtime state for memory pressure detection.
type ResourceAgentMemoryFaultState_015 struct {
	mu sync.Mutex

	config  ResourceAgentMemoryFaultConfig_015
	history []ResourceAgentMemorySample_015
	head    int

	currentScenario ResourceAgentMemoryPressureScenario_015
	currentSeverity ResourceAgentMemoryPressureSeverity_015
	leakStartTime   time.Time
	leakStartBytes  uint64

	oomKillWindow  []time.Time
	throttleEvents []time.Time
	lastSampleTime time.Time
}

// NewMemoryFaultState_015 creates a new state initialized with the given config.
func NewMemoryFaultState_015(cfg ResourceAgentMemoryFaultConfig_015) (*ResourceAgentMemoryFaultState_015, error) {
	if err := ValidateMemoryFaultConfig_015(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	s := &ResourceAgentMemoryFaultState_015{
		config:          cfg,
		history:         make([]ResourceAgentMemorySample_015, cfg.HistorySize),
		head:            0,
		currentScenario: MemoryScenarioNone_015,
		currentSeverity: MemoryPressureNone_015,
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// Circular Buffer for History
// ---------------------------------------------------------------------------

// pushSample_015 adds a sample to the circular history buffer.
func (s *ResourceAgentMemoryFaultState_015) pushSample_015(sample ResourceAgentMemorySample_015) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history[s.head] = sample
	s.head = (s.head + 1) % len(s.history)
	s.lastSampleTime = sample.Timestamp
}

// getRecentSamples_015 returns the N most recent samples in chronological order.
func (s *ResourceAgentMemoryFaultState_015) getRecentSamples_015(n int) []ResourceAgentMemorySample_015 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n > len(s.history) {
		n = len(s.history)
	}
	res := make([]ResourceAgentMemorySample_015, 0, n)
	idx := (s.head - n + len(s.history)) % len(s.history)
	for i := 0; i < n; i++ {
		p := (idx + i) % len(s.history)
		if s.history[p].Timestamp.IsZero() {
			continue
		}
		res = append(res, s.history[p])
	}
	return res
}

// ---------------------------------------------------------------------------
// Detection Heuristics (Table-Driven)
// ---------------------------------------------------------------------------

// pressureThresholdTable_015 maps severity to memory usage thresholds.
var pressureThresholdTable_015 = []struct {
	Severity ResourceAgentMemoryPressureSeverity_015
	MinPct   float64
	MaxPct   float64
}{
	{MemoryPressureNone_015, 0, 70},
	{MemoryPressureWarning_015, 70, 85},
	{MemoryPressureCritical_015, 85, 95},
	{MemoryPressureFatal_015, 95, 101},
}

// severityFromMemoryPercent_015 returns the severity based on used memory percentage.
func severityFromMemoryPercent_015(usedPct float64) ResourceAgentMemoryPressureSeverity_015 {
	for _, entry := range pressureThresholdTable_015 {
		if usedPct >= entry.MinPct && usedPct < entry.MaxPct {
			return entry.Severity
		}
	}
	return MemoryPressureFatal_015
}

// swapPressureTable_015 defines swap usage severity thresholds.
var swapPressureTable_015 = []struct {
	Severity ResourceAgentMemoryPressureSeverity_015
	MinPct   float64
	MaxPct   float64
}{
	{MemoryPressureNone_015, 0, 50},
	{MemoryPressureWarning_015, 50, 80},
	{MemoryPressureCritical_015, 80, 101},
}

// severityFromSwapPercent_015 returns swap-based severity.
func severityFromSwapPercent_015(swapPct float64) ResourceAgentMemoryPressureSeverity_015 {
	for _, entry := range swapPressureTable_015 {
		if swapPct >= entry.MinPct && swapPct < entry.MaxPct {
			return entry.Severity
		}
	}
	return MemoryPressureCritical_015
}

// detectLeakScenario_015 determines if a memory leak is in progress.
func (s *ResourceAgentMemoryFaultState_015) detectLeakScenario_015(samples []ResourceAgentMemorySample_015) (bool, float64) {
	if len(samples) < 2 {
		return false, 0
	}
	first := samples[0]
	last := samples[len(samples)-1]
	duration := last.Timestamp.Sub(first.Timestamp).Seconds()
	if duration < float64(s.config.LeakDurationSec) {
		return false, 0
	}
	growthBytes := int64(last.RSSBytes) - int64(first.RSSBytes)
	if growthBytes <= 0 {
		return false, 0
	}
	growthMB := float64(growthBytes) / (1024 * 1024)
	rateMBPerSec := growthMB / duration
	return rateMBPerSec >= s.config.LeakRateMBPerSec && duration >= float64(s.config.LeakDurationSec), rateMBPerSec
}

// detectBloatScenario_015 detects RSS bloat above baseline.
func (s *ResourceAgentMemoryFaultState_015) detectBloatScenario_015(samples []ResourceAgentMemorySample_015) bool {
	if len(samples) < 2 {
		return false
	}
	var baseline uint64
	baseline = samples[0].RSSBytes
	if baseline == 0 {
		return false
	}
	lastRSS := samples[len(samples)-1].RSSBytes
	diffMB := float64(int64(lastRSS)-int64(baseline)) / (1024 * 1024)
	return diffMB >= s.config.BloatThresholdMB
}

// detectOOMScenario_015 checks if OOM kills occurred in the history window.
func (s *ResourceAgentMemoryFaultState_015) detectOOMScenario_015(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := now.Add(-s.config.OOMKillHistoryWindow)
	// Remove old OOM events outside the window
	var recent []time.Time
	for _, t := range s.oomKillWindow {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	s.oomKillWindow = recent
	return len(recent) > 0
}

// detectCgroupThrottleScenario_015 checks if cgroup throttle events exceed threshold.
func (s *ResourceAgentMemoryFaultState_015) detectCgroupThrottleScenario_015(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Count events in the last minute
	cutoff := now.Add(-1 * time.Minute)
	var count int
	var recent []time.Time
	for _, t := range s.throttleEvents {
		if t.After(cutoff) {
			recent = append(recent, t)
			count++
		}
	}
	s.throttleEvents = recent
	return count >= s.config.CgroupThrottleThreshold
}

// EvaluateMemoryPressure_015 runs all heuristics and updates the current scenario/severity.
func (s *ResourceAgentMemoryFaultState_015) EvaluateMemoryPressure_015(sample ResourceAgentMemorySample_015) (ResourceAgentMemoryPressureScenario_015, ResourceAgentMemoryPressureSeverity_015) {
	s.pushSample_015(sample)
	recent := s.getRecentSamples_015(s.config.HistorySize)

	// Determine severity from memory usage
	sev := severityFromMemoryPercent_015(sample.UsedPercent)

	// Determine scenario heuristics
	scenario := MemoryScenarioNone_015
	if leak, _ := s.detectLeakScenario_015(recent); leak {
		scenario = MemoryScenarioLeak_015
	} else if s.detectBloatScenario_015(recent) {
		scenario = MemoryScenarioBloat_015
	} else if s.detectOOMScenario_015(sample.Timestamp) {
		scenario = MemoryScenarioOOM_015
	} else if severityFromSwapPercent_015(float64(sample.SwapUsedBytes)*100/float64(sample.SwapTotalBytes+1)) > MemoryPressureNone_015 {
		scenario = MemoryScenarioSwapPressure_015
	} else if s.detectCgroupThrottleScenario_015(sample.Timestamp) {
		scenario = MemoryScenarioCgroupThrottle_015
	}

	// For OOM and cgroup throttle, escalate severity if needed
	if scenario == MemoryScenarioOOM_015 && sev < MemoryPressureCritical_015 {
		sev = MemoryPressureCritical_015
	}
	if scenario == MemoryScenarioCgroupThrottle_015 && sev < MemoryPressureWarning_015 {
		sev = MemoryPressureWarning_015
	}

	s.mu.Lock()
	s.currentScenario = scenario
	s.currentSeverity = sev
	s.mu.Unlock()

	return scenario, sev
}

// GetCurrentPressure_015 returns the latest evaluated scenario and severity.
func (s *ResourceAgentMemoryFaultState_015) GetCurrentPressure_015() (ResourceAgentMemoryPressureScenario_015, ResourceAgentMemoryPressureSeverity_015) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentScenario, s.currentSeverity
}
