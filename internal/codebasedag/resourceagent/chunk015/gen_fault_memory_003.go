package chunk015

import (
	"errors"
	"math"
	"sync"
	"time"
)

// ResourceAgentMemoryConfig_003 holds configuration for memory pressure detection.
type ResourceAgentMemoryConfig_003 struct {
	// HighPressureThreshold is the fraction of total memory (0.0–1.0) above which pressure is considered high.
	HighPressureThreshold float64 `json:"high_pressure_threshold"`
	// ModeratePressureThreshold is the fraction above which pressure is moderate.
	ModeratePressureThreshold float64 `json:"moderate_pressure_threshold"`
	// CriticalPressureThreshold is the fraction above which pressure is critical.
	CriticalPressureThreshold float64 `json:"critical_pressure_threshold"`
	// SwapUsageThreshold is the fraction of swap used above which it contributes to pressure.
	SwapUsageThreshold float64 `json:"swap_usage_threshold"`
	// WindowSize is the number of recent samples used for moving average calculations.
	WindowSize int `json:"window_size"`
	// PercentileRank determines which percentile of recent samples to use (0.0–1.0).
	PercentileRank float64 `json:"percentile_rank"`
	// EWMAAlpha is the smoothing factor for exponential weighted moving average (0.0–1.0).
	EWMAAlpha float64 `json:"ewma_alpha"`
	// MinSamplesBeforeDetection is the minimum number of samples required before detection begins.
	MinSamplesBeforeDetection int `json:"min_samples_before_detection"`
}

// ResourceAgentMemoryMetric_003 represents a single memory metric sample.
type ResourceAgentMemoryMetric_003 struct {
	Timestamp time.Time `json:"timestamp"`
	// UsedMemoryFraction is used memory / total memory (0.0–1.0).
	UsedMemoryFraction float64 `json:"used_memory_fraction"`
	// SwapUsedFraction is used swap / total swap (0.0–1.0). 0 if no swap.
	SwapUsedFraction float64 `json:"swap_used_fraction"`
	// CommittedMemoryFraction is committed memory / total memory (0.0–1.0).
	CommittedMemoryFraction float64 `json:"committed_memory_fraction"`
	// PageFaultRate is the rate of major page faults per second.
	PageFaultRate float64 `json:"page_fault_rate"`
	// OOMKillRate is the rate of OOM kills per second (typically 0 or small).
	OOMKillRate float64 `json:"oom_kill_rate"`
}

// ResourceAgentMemoryPressureLevel_003 indicates the detected memory pressure level.
type ResourceAgentMemoryPressureLevel_003 int

const (
	MemoryPressureLow_003      ResourceAgentMemoryPressureLevel_003 = iota
	MemoryPressureModerate_003
	MemoryPressureHigh_003
	MemoryPressureCritical_003
)

// String returns the string representation of the pressure level.
func (l ResourceAgentMemoryPressureLevel_003) String() string {
	switch l {
	case MemoryPressureLow_003:
		return "low"
	case MemoryPressureModerate_003:
		return "moderate"
	case MemoryPressureHigh_003:
		return "high"
	case MemoryPressureCritical_003:
		return "critical"
	default:
		return "unknown"
	}
}

// ResourceAgentMemoryDetectionResult_003 contains the outcome of detection.
type ResourceAgentMemoryDetectionResult_003 struct {
	PressureLevel   ResourceAgentMemoryPressureLevel_003 `json:"pressure_level"`
	PressureScore   float64                              `json:"pressure_score"`
	SampleCount     int                                  `json:"sample_count"`
	AvgUsedMemory   float64                              `json:"avg_used_memory"`
	AvgSwapUsed     float64                              `json:"avg_swap_used"`
	EWMAUsedMemory  float64                              `json:"ewma_used_memory"`
	PercentileValue float64                              `json:"percentile_value"`
	DetectedAt      time.Time                            `json:"detected_at"`
}

// ValidateMemoryConfig_003 validates the configuration and returns an error if invalid.
func ValidateMemoryConfig_003(cfg *ResourceAgentMemoryConfig_003) error {
	if cfg == nil {
		return errors.New("memory config is nil")
	}
	if cfg.HighPressureThreshold <= 0.0 || cfg.HighPressureThreshold > 1.0 {
		return errors.New("high_pressure_threshold must be in (0.0, 1.0]")
	}
	if cfg.ModeratePressureThreshold < 0.0 || cfg.ModeratePressureThreshold >= cfg.HighPressureThreshold {
		return errors.New("moderate_pressure_threshold must be >= 0 and < high_pressure_threshold")
	}
	if cfg.CriticalPressureThreshold <= cfg.HighPressureThreshold || cfg.CriticalPressureThreshold > 1.1 {
		return errors.New("critical_pressure_threshold must be > high_pressure_threshold and <= 1.1")
	}
	if cfg.SwapUsageThreshold < 0.0 || cfg.SwapUsageThreshold > 1.0 {
		return errors.New("swap_usage_threshold must be in [0.0, 1.0]")
	}
	if cfg.WindowSize < 1 {
		return errors.New("window_size must be at least 1")
	}
	if cfg.PercentileRank < 0.0 || cfg.PercentileRank > 1.0 {
		return errors.New("percentile_rank must be in [0.0, 1.0]")
	}
	if cfg.EWMAAlpha <= 0.0 || cfg.EWMAAlpha > 1.0 {
		return errors.New("ewma_alpha must be in (0.0, 1.0]")
	}
	if cfg.MinSamplesBeforeDetection < 1 {
		return errors.New("min_samples_before_detection must be at least 1")
	}
	if cfg.WindowSize < cfg.MinSamplesBeforeDetection {
		return errors.New("window_size must be >= min_samples_before_detection")
	}
	return nil
}

// GetDefaultMemoryConfig_003 returns a sensible default configuration.
func GetDefaultMemoryConfig_003() *ResourceAgentMemoryConfig_003 {
	return &ResourceAgentMemoryConfig_003{
		HighPressureThreshold:      0.8,
		ModeratePressureThreshold:  0.6,
		CriticalPressureThreshold:  0.95,
		SwapUsageThreshold:         0.5,
		WindowSize:                 60,
		PercentileRank:             0.95,
		EWMAAlpha:                  0.3,
		MinSamplesBeforeDetection:  5,
	}
}

// ResourceAgentMemoryDetector_003 performs memory pressure detection using configurable heuristics.
type ResourceAgentMemoryDetector_003 struct {
	mu        sync.Mutex
	config    *ResourceAgentMemoryConfig_003
	samples   []ResourceAgentMemoryMetric_003
	ewmaValue float64
}

// NewMemoryDetector_003 creates a new detector with the given config. Returns error if config invalid.
func NewMemoryDetector_003(cfg *ResourceAgentMemoryConfig_003) (*ResourceAgentMemoryDetector_003, error) {
	if err := ValidateMemoryConfig_003(cfg); err != nil {
		return nil, err
	}
	return &ResourceAgentMemoryDetector_003{
		config:  cfg,
		samples: make([]ResourceAgentMemoryMetric_003, 0, cfg.WindowSize),
	}, nil
}

// AddSample_003 adds a memory metric sample and optionally trims the window.
func (d *ResourceAgentMemoryDetector_003) AddSample_003(m ResourceAgentMemoryMetric_003) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.samples = append(d.samples, m)
	if len(d.samples) > d.config.WindowSize {
		d.samples = d.samples[len(d.samples)-d.config.WindowSize:]
	}

	// Update EWMA if we have samples
	if len(d.samples) == 1 {
		d.ewmaValue = m.UsedMemoryFraction
	} else {
		d.ewmaValue = d.config.EWMAAlpha*m.UsedMemoryFraction + (1-d.config.EWMAAlpha)*d.ewmaValue
	}
}

// DetectPressure_003 runs detection logic on current samples and returns a result.
func (d *ResourceAgentMemoryDetector_003) DetectPressure_003() (*ResourceAgentMemoryDetectionResult_003, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.samples) < d.config.MinSamplesBeforeDetection {
		return nil, errors.New("insufficient samples for detection")
	}

	// Calculate statistics
	n := len(d.samples)
	sumUsed := 0.0
	sumSwap := 0.0
	values := make([]float64, n)
	for i, s := range d.samples {
		sumUsed += s.UsedMemoryFraction
		sumSwap += s.SwapUsedFraction
		values[i] = s.UsedMemoryFraction + 0.2*s.SwapUsedFraction // combined pressure indicator
	}

	avgUsed := sumUsed / float64(n)
	avgSwap := sumSwap / float64(n)

	// Percentile calculation
	percentile := percentile_003(values, d.config.PercentileRank)

	// Compute pressure score as weighted combination
	score := (0.5 * percentile) +
		(0.3 * d.ewmaValue) +
		(0.2 * math.Min(avgSwap, 1.0))

	// Apply swap penalty if swap usage above threshold
	if avgSwap > d.config.SwapUsageThreshold {
		swapExcess := (avgSwap - d.config.SwapUsageThreshold) / (1.0 - d.config.SwapUsageThreshold)
		score += 0.1 * swapExcess
	}

	// Include page fault and OOM kill rates as additional pressure signals
	maxFaultRate := 1000.0 // typical max
	for _, s := range d.samples {
		faultFactor := math.Min(s.PageFaultRate/maxFaultRate, 1.0)
		score += 0.03 * faultFactor
		score += 0.05 * math.Min(s.OOMKillRate*10, 1.0) // OOM kills are severe
	}

	// Clamp score to [0, 1]
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	// Determine level based on thresholds
	var level ResourceAgentMemoryPressureLevel_003
	switch {
	case score >= d.config.CriticalPressureThreshold:
		level = MemoryPressureCritical_003
	case score >= d.config.HighPressureThreshold:
		level = MemoryPressureHigh_003
	case score >= d.config.ModeratePressureThreshold:
		level = MemoryPressureModerate_003
	default:
		level = MemoryPressureLow_003
	}

	result := &ResourceAgentMemoryDetectionResult_003{
		PressureLevel:   level,
		PressureScore:   score,
		SampleCount:     n,
		AvgUsedMemory:   avgUsed,
		AvgSwapUsed:     avgSwap,
		EWMAUsedMemory:  d.ewmaValue,
		PercentileValue: percentile,
		DetectedAt:      time.Now(),
	}
	return result, nil
}

// percentile_003 calculates the given percentile (0.0–1.0) of a sorted slice of float64.
func percentile_003(data []float64, rank float64) float64 {
	if len(data) == 0 {
		return 0.0
	}
	if rank < 0.0 || rank > 1.0 {
		rank = 0.5
	}
	// Sort a copy
	sorted := make([]float64, len(data))
	copy(sorted, data)
	// Simple insertion sort for small slices (deterministic, no random)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	index := rank * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return sorted[lower]
	}
	// Linear interpolation
	frac := index - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

// MemoryPressureThresholdTable_003 provides a deterministic table of thresholds and their descriptions.
func MemoryPressureThresholdTable_003() map[string]float64 {
	return map[string]float64{
		"low_to_moderate":      0.6,
		"moderate_to_high":     0.8,
		"high_to_critical":     0.95,
		"swap_warning":         0.5,
		"oom_kill_rate_alert":  0.1,
		"page_fault_rate_high": 500.0,
	}
}

// EvaluateMemoryScenario_003 evaluates a single scenario (set of metrics) and returns a pressure level and score.
func EvaluateMemoryScenario_003(metrics ResourceAgentMemoryMetric_003, thresholds map[string]float64) (ResourceAgentMemoryPressureLevel_003, float64) {
	if thresholds == nil {
		thresholds = MemoryPressureThresholdTable_003()
	}
	score := metrics.UsedMemoryFraction +
		0.2*metrics.SwapUsedFraction +
		0.05*math.Min(metrics.PageFaultRate/thresholds["page_fault_rate_high"], 1.0) +
		0.1*math.Min(metrics.OOMKillRate/thresholds["oom_kill_rate_alert"], 1.0)

	if score > 1.0 {
		score = 1.0
	}

	level := MemoryPressureLow_003
	if score >= thresholds["high_to_critical"] {
		level = MemoryPressureCritical_003
	} else if score >= thresholds["moderate_to_high"] {
		level = MemoryPressureHigh_003
	} else if score >= thresholds["low_to_moderate"] {
		level = MemoryPressureModerate_003
	}
	return level, score
}

// IsMemoryPressureCritical_003 is a quick check using thresholds.
func IsMemoryPressureCritical_003(usedFraction, swapFraction, pageFaultRate, oomKillRate float64) bool {
	metrics := ResourceAgentMemoryMetric_003{
		UsedMemoryFraction: usedFraction,
		SwapUsedFraction:   swapFraction,
		PageFaultRate:      pageFaultRate,
		OOMKillRate:        oomKillRate,
	}
	level, _ := EvaluateMemoryScenario_003(metrics, nil)
	return level == MemoryPressureCritical_003
}

// ScenarioTemplate_003 defines a named memory pressure scenario with expected outcome.
type ScenarioTemplate_003 struct {
	Name          string
	Metrics       ResourceAgentMemoryMetric_003
	ExpectedLevel ResourceAgentMemoryPressureLevel_003
}

// GetPressureScenarioTable_003 returns a deterministic table of test scenarios.
func GetPressureScenarioTable_003() []ScenarioTemplate_003 {
	return []ScenarioTemplate_003{
		{Name: "normal_low", ExpectedLevel: MemoryPressureLow_003, Metrics: ResourceAgentMemoryMetric_003{
			UsedMemoryFraction: 0.4, SwapUsedFraction: 0.1, PageFaultRate: 10, OOMKillRate: 0,
		}},
		{Name: "moderate_swap", ExpectedLevel: MemoryPressureModerate_003, Metrics: ResourceAgentMemoryMetric_003{
			UsedMemoryFraction: 0.65, SwapUsedFraction: 0.55, PageFaultRate: 100, OOMKillRate: 0,
		}},
		{Name: "high_memory", ExpectedLevel: MemoryPressureHigh_003, Metrics: ResourceAgentMemoryMetric_003{
			UsedMemoryFraction: 0.85, SwapUsedFraction: 0.3, PageFaultRate: 200, OOMKillRate: 0,
		}},
		{Name: "critical_oom", ExpectedLevel: MemoryPressureCritical_003, Metrics: ResourceAgentMemoryMetric_003{
			UsedMemoryFraction: 0.95, SwapUsedFraction: 0.9, PageFaultRate: 1000, OOMKillRate: 0.2,
		}},
	}
}

// ComputePressureTrend_003 computes the trend direction based on recent scores.
func ComputePressureTrend_003(scores []float64) string {
	if len(scores) < 2 {
		return "stable"
	}
	// Use linear regression slope
	n := float64(len(scores))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	for i, y := range scores {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX + 1e-10) // avoid division by zero
	switch {
	case slope > 0.01:
		return "increasing"
	case slope < -0.01:
		return "decreasing"
	default:
		return "stable"
	}
}

// AggregateMemoryMetrics_003 returns the average of a slice of metrics (deterministic).
func AggregateMemoryMetrics_003(metrics []ResourceAgentMemoryMetric_003) ResourceAgentMemoryMetric_003 {
	if len(metrics) == 0 {
		return ResourceAgentMemoryMetric_003{}
	}
	var sum ResourceAgentMemoryMetric_003
	for _, m := range metrics {
		sum.UsedMemoryFraction += m.UsedMemoryFraction
		sum.SwapUsedFraction += m.SwapUsedFraction
		sum.CommittedMemoryFraction += m.CommittedMemoryFraction
		sum.PageFaultRate += m.PageFaultRate
		sum.OOMKillRate += m.OOMKillRate
	}
	n := float64(len(metrics))
	return ResourceAgentMemoryMetric_003{
		UsedMemoryFraction:     sum.UsedMemoryFraction / n,
		SwapUsedFraction:       sum.SwapUsedFraction / n,
		CommittedMemoryFraction: sum.CommittedMemoryFraction / n,
		PageFaultRate:          sum.PageFaultRate / n,
		OOMKillRate:            sum.OOMKillRate / n,
	}
}

// ResourceAgentMemoryScenario_003 groups a scenario with its expected outcome for validation.
type ResourceAgentMemoryScenario_003 struct {
	Name        string
	Config      *ResourceAgentMemoryConfig_003
	Metrics     []ResourceAgentMemoryMetric_003
	ExpectedResult error // nil if valid, non-nil if validation should fail
}

// GetMemoryScenarioValidationTable_003 returns scenarios to test config validation.
func GetMemoryScenarioValidationTable_003() []ResourceAgentMemoryScenario_003 {
	return []ResourceAgentMemoryScenario_003{
		{Name: "valid_default", Config: GetDefaultMemoryConfig_003(), ExpectedResult: nil},
		{Name: "invalid_threshold_high_too_small", Config: &ResourceAgentMemoryConfig_003{
			HighPressureThreshold: 0.0,
		}, ExpectedResult: errors.New("high_pressure_threshold must be in (0.0, 1.0]")},
		{Name: "invalid_window_zero", Config: &ResourceAgentMemoryConfig_003{
			HighPressureThreshold:     0.8,
			ModeratePressureThreshold: 0.5,
			CriticalPressureThreshold: 0.95,
			SwapUsageThreshold:        0.5,
			WindowSize:                0,
			PercentileRank:            0.95,
			EWMAAlpha:                 0.3,
			MinSamplesBeforeDetection: 1,
		}, ExpectedResult: errors.New("window_size must be at least 1")},
		{Name: "invalid_ewma_alpha_too_low", Config: &ResourceAgentMemoryConfig_003{
			HighPressureThreshold:     0.8,
			ModeratePressureThreshold: 0.5,
			CriticalPressureThreshold: 0.95,
			SwapUsageThreshold:        0.5,
			WindowSize:                10,
			PercentileRank:            0.95,
			EWMAAlpha:                 -0.1,
			MinSamplesBeforeDetection: 5,
		}, ExpectedResult: errors.New("ewma_alpha must be in (0.0, 1.0]")},
	}
}

// RunDetectionOnScenarios_003 runs detection on a set of metrics and returns results.
func RunDetectionOnScenarios_003(detector *ResourceAgentMemoryDetector_003, scenarios []ResourceAgentMemoryScenario_003) ([]*ResourceAgentMemoryDetectionResult_003, []error) {
	var results []*ResourceAgentMemoryDetectionResult_003
	var errs []error
	for _, sc := range scenarios {
		for _, m := range sc.Metrics {
			detector.AddSample_003(m)
		}
		r, err := detector.DetectPressure_003()
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, r)
		}
	}
	return results, errs
}
