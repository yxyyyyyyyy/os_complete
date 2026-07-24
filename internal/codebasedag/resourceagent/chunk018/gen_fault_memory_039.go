package chunk018

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ResourceAgentMemoryPressureLevel_039 represents the severity of memory pressure.
type ResourceAgentMemoryPressureLevel_039 int

const (
	PressureNone_039     ResourceAgentMemoryPressureLevel_039 = iota
	PressureLow_039
	PressureMedium_039
	PressureHigh_039
	PressureCritical_039
)

// ResourceAgentMemoryPressureScenario_039 defines a specific memory pressure scenario
// with thresholds and probability of encountering an OOM event.
type ResourceAgentMemoryPressureScenario_039 struct {
	Name             string
	ThresholdPercent float64
	OOMProbability   float64
	SwapUsagePercent float64
	Description      string
}

// ResourceAgentMemoryPressureConfig_039 holds configuration parameters for the
// memory pressure detector.
type ResourceAgentMemoryPressureConfig_039 struct {
	CheckInterval     time.Duration
	WarningThreshold  float64
	CriticalThreshold float64
	Scenarios         []ResourceAgentMemoryPressureScenario_039
}

// ResourceAgentMemoryPressureMetrics_039 captures a snapshot of memory-related
// system metrics at a particular time.
type ResourceAgentMemoryPressureMetrics_039 struct {
	Timestamp         time.Time
	MemoryUsedPercent float64
	SwapUsedPercent   float64
	OOMKillCount      int
	PageFaultRate     float64
}

// ResourceAgentMemoryPressureDetector_039 monitors memory metrics, evaluates them
// against configured thresholds and scenarios, and produces detection results.
type ResourceAgentMemoryPressureDetector_039 struct {
	mu      sync.RWMutex
	config  ResourceAgentMemoryPressureConfig_039
	history []ResourceAgentMemoryPressureMetrics_039
	alert   bool
}

// ResourceAgentMemoryPressureResult_039 contains the outcome of a detection cycle.
type ResourceAgentMemoryPressureResult_039 struct {
	Level     ResourceAgentMemoryPressureLevel_039
	Scenario  *ResourceAgentMemoryPressureScenario_039
	Metrics   ResourceAgentMemoryPressureMetrics_039
	Diagnosis string
}

// ValidateMemoryPressureConfig_039 checks that the supplied configuration is
// valid. It returns an error describing the first problem encountered.
func ValidateMemoryPressureConfig_039(cfg ResourceAgentMemoryPressureConfig_039) error {
	if cfg.CheckInterval <= 0 {
		return errors.New("CheckInterval must be positive")
	}
	if cfg.WarningThreshold < 0 || cfg.WarningThreshold > 100 {
		return errors.New("WarningThreshold must be between 0 and 100")
	}
	if cfg.CriticalThreshold < 0 || cfg.CriticalThreshold > 100 {
		return errors.New("CriticalThreshold must be between 0 and 100")
	}
	if cfg.WarningThreshold >= cfg.CriticalThreshold {
		return errors.New("WarningThreshold must be strictly less than CriticalThreshold")
	}
	for i, sc := range cfg.Scenarios {
		if sc.ThresholdPercent < 0 || sc.ThresholdPercent > 100 {
			return fmt.Errorf("Scenario[%d] ThresholdPercent out of range [0,100]", i)
		}
		if sc.OOMProbability < 0 || sc.OOMProbability > 1 {
			return fmt.Errorf("Scenario[%d] OOMProbability out of range [0,1]", i)
		}
		if sc.SwapUsagePercent < 0 || sc.SwapUsagePercent > 100 {
			return fmt.Errorf("Scenario[%d] SwapUsagePercent out of range [0,100]", i)
		}
	}
	return nil
}

// NewMemoryPressureDetector_039 creates a new detector after validating the
// configuration. Returns an error if the configuration is invalid.
func NewMemoryPressureDetector_039(cfg ResourceAgentMemoryPressureConfig_039) (*ResourceAgentMemoryPressureDetector_039, error) {
	if err := ValidateMemoryPressureConfig_039(cfg); err != nil {
		return nil, err
	}
	return &ResourceAgentMemoryPressureDetector_039{
		config:  cfg,
		history: make([]ResourceAgentMemoryPressureMetrics_039, 0, 100),
	}, nil
}

// pressureLevelFromPercent_039 determines the pressure level based on the
// memory usage percentage and configured warning/critical thresholds.
func pressureLevelFromPercent_039(percent, warn, crit float64) ResourceAgentMemoryPressureLevel_039 {
	if percent >= crit {
		return PressureCritical_039
	}
	if percent >= warn {
		return PressureHigh_039
	}
	if percent >= warn*0.8 {
		return PressureMedium_039
	}
	if percent > 10 {
		return PressureLow_039
	}
	return PressureNone_039
}

// formatDiagnosis_039 creates a human-readable diagnosis string from the
// detection result.
func formatDiagnosis_039(metrics ResourceAgentMemoryPressureMetrics_039, level ResourceAgentMemoryPressureLevel_039, sc *ResourceAgentMemoryPressureScenario_039) string {
	diag := fmt.Sprintf("Memory: %.1f%% used, Swap: %.1f%%, OOM: %d, PF/s: %.1f",
		metrics.MemoryUsedPercent, metrics.SwapUsedPercent, metrics.OOMKillCount, metrics.PageFaultRate)
	if sc != nil {
		diag += " | Scenario: " + sc.Name
	}
	diag += " | Level: " + level.String()
	return diag
}

// String returns a textual representation of the pressure level.
func (l ResourceAgentMemoryPressureLevel_039) String() string {
	switch l {
	case PressureNone_039:
		return "none"
	case PressureLow_039:
		return "low"
	case PressureMedium_039:
		return "medium"
	case PressureHigh_039:
		return "high"
	case PressureCritical_039:
		return "critical"
	default:
		return "unknown"
	}
}

// DefaultMemoryPressureScenarios_039 returns a set of predefined scenarios that
// represent common memory pressure patterns.
func DefaultMemoryPressureScenarios_039() []ResourceAgentMemoryPressureScenario_039 {
	return []ResourceAgentMemoryPressureScenario_039{
		{
			Name:             "LowMemoryStress",
			ThresholdPercent: 60,
			OOMProbability:   0.01,
			SwapUsagePercent: 10,
			Description:      "Memory usage above 60%, low swap usage, and low OOM risk.",
		},
		{
			Name:             "ModerateMemoryPressure",
			ThresholdPercent: 75,
			OOMProbability:   0.05,
			SwapUsagePercent: 30,
			Description:      "Memory above 75% and swap above 30%.",
		},
		{
			Name:             "HighMemoryPressure",
			ThresholdPercent: 85,
			OOMProbability:   0.20,
			SwapUsagePercent: 50,
			Description:      "Memory above 85% and swap above 50%.",
		},
		{
			Name:             "CriticalMemoryExhaustion",
			ThresholdPercent: 95,
			OOMProbability:   0.50,
			SwapUsagePercent: 80,
			Description:      "Memory near exhaustion, heavy swapping, high OOM risk.",
		},
	}
}

// SortMemoryPressureScenariosByThreshold_039 sorts a slice of scenarios in
// ascending order of their ThresholdPercent.
func SortMemoryPressureScenariosByThreshold_039(scenarios []ResourceAgentMemoryPressureScenario_039) {
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].ThresholdPercent < scenarios[j].ThresholdPercent
	})
}

// ComputeMemoryPressureMetrics_039 builds a ResourceAgentMemoryPressureMetrics_039
// struct from raw system counters and returns it.
func ComputeMemoryPressureMetrics_039(memTotal, memUsed, swapTotal, swapUsed uint64, oomKills int, pageFaultsPerSec float64) ResourceAgentMemoryPressureMetrics_039 {
	memPct := (float64(memUsed) / float64(memTotal)) * 100
	var swapPct float64
	if swapTotal > 0 {
		swapPct = (float64(swapUsed) / float64(swapTotal)) * 100
	}
	return ResourceAgentMemoryPressureMetrics_039{
		Timestamp:         time.Now(),
		MemoryUsedPercent: memPct,
		SwapUsedPercent:   swapPct,
		OOMKillCount:      oomKills,
		PageFaultRate:     pageFaultsPerSec,
	}
}

// DetectMemoryPressure_039 evaluates the provided metrics against the
// detector's configuration and returns a detection result. A nil result
// is returned together with an error if the metrics are invalid.
func (d *ResourceAgentMemoryPressureDetector_039) DetectMemoryPressure_039(metrics ResourceAgentMemoryPressureMetrics_039) (ResourceAgentMemoryPressureResult_039, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if metrics.MemoryUsedPercent < 0 || metrics.MemoryUsedPercent > 100 {
		return ResourceAgentMemoryPressureResult_039{}, errors.New("invalid MemoryUsedPercent")
	}
	if metrics.SwapUsedPercent < 0 || metrics.SwapUsedPercent > 100 {
		return ResourceAgentMemoryPressureResult_039{}, errors.New("invalid SwapUsedPercent")
	}
	if metrics.PageFaultRate < 0 {
		return ResourceAgentMemoryPressureResult_039{}, errors.New("invalid PageFaultRate")
	}

	d.history = append(d.history, metrics)
	if len(d.history) > 100 {
		d.history = d.history[1:]
	}

	level := pressureLevelFromPercent_039(metrics.MemoryUsedPercent, d.config.WarningThreshold, d.config.CriticalThreshold)

	var scenario *ResourceAgentMemoryPressureScenario_039
	for i := range d.config.Scenarios {
		sc := &d.config.Scenarios[i]
		if metrics.MemoryUsedPercent >= sc.ThresholdPercent && metrics.SwapUsedPercent >= sc.SwapUsagePercent {
			scenario = sc
			break
		}
	}

	if metrics.PageFaultRate > 1000 && metrics.OOMKillCount > 0 && level < PressureHigh_039 {
		level = PressureHigh_039
	}
	if metrics.SwapUsedPercent > 50 && level < PressureMedium_039 {
		level = PressureMedium_039
	}

	diagnosis := formatDiagnosis_039(metrics, level, scenario)
	d.alert = (level >= PressureHigh_039)

	return ResourceAgentMemoryPressureResult_039{
		Level:     level,
		Scenario:  scenario,
		Metrics:   metrics,
		Diagnosis: diagnosis,
	}, nil
}

// EvaluateOOMRisk_039 computes a risk score (0.0 – 1.0) of an OOM event based
// on recent metric trends. It requires at least two history entries.
func (d *ResourceAgentMemoryPressureDetector_039) EvaluateOOMRisk_039() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.history) < 2 {
		return 0
	}
	last := d.history[len(d.history)-1]
	prev := d.history[len(d.history)-2]
	deltaMem := last.MemoryUsedPercent - prev.MemoryUsedPercent
	deltaSwap := last.SwapUsedPercent - prev.SwapUsedPercent
	deltaOOM := last.OOMKillCount - prev.OOMKillCount
	deltaPF := last.PageFaultRate - prev.PageFaultRate

	risk := 0.0
	if deltaMem > 5 && last.MemoryUsedPercent > 80 {
		risk += 0.3
	}
	if deltaSwap > 10 && last.SwapUsedPercent > 50 {
		risk += 0.3
	}
	if deltaOOM > 0 {
		risk += 0.2
	}
	if deltaPF > 500 {
		risk += 0.2
	}
	if risk > 1.0 {
		risk = 1.0
	}
	return risk
}

// GetHistory_039 returns a copy of the detector's metric history.
func (d *ResourceAgentMemoryPressureDetector_039) GetHistory_039() []ResourceAgentMemoryPressureMetrics_039 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cp := make([]ResourceAgentMemoryPressureMetrics_039, len(d.history))
	copy(cp, d.history)
	return cp
}

// ClearHistory_039 removes all previously recorded metrics.
func (d *ResourceAgentMemoryPressureDetector_039) ClearHistory_039() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = d.history[:0]
}

// IsAlertActive_039 reports whether the detector has raised an alert since the
// last detection.
func (d *ResourceAgentMemoryPressureDetector_039) IsAlertActive_039() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.alert
}

// MemoryPressureTestSample_039 is a helper struct for table‑driven validation of
// pressure level classification.
type MemoryPressureTestSample_039 struct {
	Name           string
	Metrics        ResourceAgentMemoryPressureMetrics_039
	WarnThreshold  float64
	CritThreshold  float64
	ExpectedLevel  ResourceAgentMemoryPressureLevel_039
}

// SampleMemoryPressureData_039 returns a deterministic set of test samples
// covering a variety of memory pressure scenarios.
func SampleMemoryPressureData_039() []MemoryPressureTestSample_039 {
	now := time.Now()
	return []MemoryPressureTestSample_039{
		{
			Name: "Normal operation",
			Metrics: ResourceAgentMemoryPressureMetrics_039{
				Timestamp:         now.Add(-10 * time.Second),
				MemoryUsedPercent: 25.0,
				SwapUsedPercent:   5.0,
				OOMKillCount:      0,
				PageFaultRate:     50.0,
			},
			WarnThreshold: 80,
			CritThreshold: 95,
			ExpectedLevel: PressureLow_039,
		},
		{
			Name: "Elevated but below warning",
			Metrics: ResourceAgentMemoryPressureMetrics_039{
				Timestamp:         now.Add(-8 * time.Second),
				MemoryUsedPercent: 60.0,
				SwapUsedPercent:   15.0,
				OOMKillCount:      0,
				PageFaultRate:     200.0,
			},
			WarnThreshold: 80,
			CritThreshold: 95,
			ExpectedLevel: PressureMedium_039,
		},
		{
			Name: "At warning threshold",
			Metrics: ResourceAgentMemoryPressureMetrics_039{
				Timestamp:         now.Add(-6 * time.Second),
				MemoryUsedPercent: 80.0,
				SwapUsedPercent:   25.0,
				OOMKillCount:      0,
				PageFaultRate:     400.0,
			},
			WarnThreshold: 80,
			CritThreshold: 95,
			ExpectedLevel: PressureHigh_039,
		},
		{
			Name: "At critical threshold",
			Metrics: ResourceAgentMemoryPressureMetrics_039{
				Timestamp:         now.Add(-4 * time.Second),
				MemoryUsedPercent: 95.0,
				SwapUsedPercent:   60.0,
				OOMKillCount:      1,
				PageFaultRate:     1200.0,
			},
			WarnThreshold: 80,
			CritThreshold: 95,
			ExpectedLevel: PressureCritical_039,
		},
	}
}
