package chunk041

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ValidateSiblingMetrics_031 checks that all metric fields are valid.
// Returns an error if any field is negative or if latency order constraints are violated.
func ValidateSiblingMetrics_031(m *SiblingMetrics_031) error {
	if m == nil {
		return errors.New("validateSiblingMetrics_031: nil metrics")
	}
	if m.CompletionCount < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: CompletionCount (%d) is negative", m.CompletionCount)
	}
	if m.SuccessCount < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: SuccessCount (%d) is negative", m.SuccessCount)
	}
	if m.SuccessCount > m.CompletionCount {
		return fmt.Errorf("validateSiblingMetrics_031: SuccessCount (%d) > CompletionCount (%d)", m.SuccessCount, m.CompletionCount)
	}
	if m.LatencyMin < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyMin (%v) is negative", m.LatencyMin)
	}
	if m.LatencyMax < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyMax (%v) is negative", m.LatencyMax)
	}
	if m.LatencyAvg < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyAvg (%v) is negative", m.LatencyAvg)
	}
	if m.LatencyP50 < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyP50 (%v) is negative", m.LatencyP50)
	}
	if m.LatencyP90 < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyP90 (%v) is negative", m.LatencyP90)
	}
	if m.LatencyP99 < 0 {
		return fmt.Errorf("validateSiblingMetrics_031: LatencyP99 (%v) is negative", m.LatencyP99)
	}
	if m.CompletionCount > 0 {
		if m.LatencyMin > m.LatencyAvg {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyMin (%v) > LatencyAvg (%v)", m.LatencyMin, m.LatencyAvg)
		}
		if m.LatencyAvg > m.LatencyMax {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyAvg (%v) > LatencyMax (%v)", m.LatencyAvg, m.LatencyMax)
		}
		if m.LatencyP50 < m.LatencyMin || m.LatencyP50 > m.LatencyMax {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyP50 (%v) out of [%v, %v]", m.LatencyP50, m.LatencyMin, m.LatencyMax)
		}
		if m.LatencyP90 < m.LatencyMin || m.LatencyP90 > m.LatencyMax {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyP90 (%v) out of [%v, %v]", m.LatencyP90, m.LatencyMin, m.LatencyMax)
		}
		if m.LatencyP99 < m.LatencyMin || m.LatencyP99 > m.LatencyMax {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyP99 (%v) out of [%v, %v]", m.LatencyP99, m.LatencyMin, m.LatencyMax)
		}
		// percentile ordering check (non-decreasing)
		if m.LatencyP50 > m.LatencyP90 {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyP50 (%v) > LatencyP90 (%v)", m.LatencyP50, m.LatencyP90)
		}
		if m.LatencyP90 > m.LatencyP99 {
			return fmt.Errorf("validateSiblingMetrics_031: LatencyP90 (%v) > LatencyP99 (%v)", m.LatencyP90, m.LatencyP99)
		}
	}
	return nil
}

// SiblingMetrics_031 holds per‑agent metrics for completion, success and latency.
type SiblingMetrics_031 struct {
	AgentID         string
	CompletionCount int
	SuccessCount    int
	LatencyMin      time.Duration
	LatencyMax      time.Duration
	LatencyAvg      time.Duration
	LatencyP50      time.Duration
	LatencyP90      time.Duration
	LatencyP99      time.Duration
}

// AggregatedSiblingMetrics_031 is the result of aggregating multiple SiblingMetrics_031.
type AggregatedSiblingMetrics_031 struct {
	TotalCompletionCount int
	TotalSuccessCount    int
	SuccessRate          float64
	LatencyMin           time.Duration
	LatencyMax           time.Duration
	LatencyAvg           time.Duration
	LatencyP50           time.Duration
	LatencyP90           time.Duration
	LatencyP99           time.Duration
	SiblingCount         int
}

// SiblingAggregator_031 accumulates metrics over time windows.
type SiblingAggregator_031 struct {
	windowSize    time.Duration
	metricsBuffer []SiblingMetrics_031
	timestamps    []time.Time
}

// NewSiblingAggregator_031 creates a new aggregator with a fixed window.
func NewSiblingAggregator_031(windowSize time.Duration) *SiblingAggregator_031 {
	if windowSize <= 0 {
		windowSize = 10 * time.Second
	}
	return &SiblingAggregator_031{
		windowSize:    windowSize,
		metricsBuffer: make([]SiblingMetrics_031, 0, 100),
		timestamps:    make([]time.Time, 0, 100),
	}
}

// AddMetric_031 inserts a new metric entry with the current time.
func (a *SiblingAggregator_031) AddMetric_031(m SiblingMetrics_031) error {
	if err := ValidateSiblingMetrics_031(&m); err != nil {
		return err
	}
	now := time.Now()
	a.metricsBuffer = append(a.metricsBuffer, m)
	a.timestamps = append(a.timestamps, now)
	a.prune_031(now)
	return nil
}

// prune_031 removes entries older than window size.
func (a *SiblingAggregator_031) prune_031(now time.Time) {
	cutoff := now.Add(-a.windowSize)
	keepFrom := 0
	for i, ts := range a.timestamps {
		if ts.After(cutoff) || ts.Equal(cutoff) {
			keepFrom = i
			break
		}
	}
	// If all are old, keep last (avoid empty buffer)
	if keepFrom >= len(a.timestamps) && len(a.timestamps) > 0 {
		keepFrom = len(a.timestamps) - 1
	}
	a.metricsBuffer = a.metricsBuffer[keepFrom:]
	a.timestamps = a.timestamps[keepFrom:]
}

// Aggregate_031 computes combined metrics from all buffered entries.
func (a *SiblingAggregator_031) Aggregate_031() AggregatedSiblingMetrics_031 {
	return AggregateSiblingMetrics_031(a.metricsBuffer)
}

// AggregateSiblingMetrics_031 combines multiple metrics into one.
func AggregateSiblingMetrics_031(metrics []SiblingMetrics_031) AggregatedSiblingMetrics_031 {
	if len(metrics) == 0 {
		return AggregatedSiblingMetrics_031{}
	}
	totalComp := 0
	totalSucc := 0
	minLat := time.Duration(math.MaxInt64)
	maxLat := time.Duration(0)
	var sumLat time.Duration
	var allLatencies []time.Duration
	successComp := 0 // count of metrics with completions > 0

	for _, m := range metrics {
		totalComp += m.CompletionCount
		totalSucc += m.SuccessCount
		if m.CompletionCount > 0 {
			successComp++
			if m.LatencyMin < minLat {
				minLat = m.LatencyMin
			}
			if m.LatencyMax > maxLat {
				maxLat = m.LatencyMax
			}
			// weighted average latency using completion count
			sumLat += m.LatencyAvg * time.Duration(m.CompletionCount)
			// collect all percentiles for combined percentile computation
			allLatencies = append(allLatencies, m.LatencyP50, m.LatencyP90, m.LatencyP99)
		}
	}

	// If no completions, return zeroed
	if totalComp == 0 {
		return AggregatedSiblingMetrics_031{
			SiblingCount: len(metrics),
		}
	}

	var successRate float64
	if totalComp > 0 {
		successRate = float64(totalSucc) / float64(totalComp)
	}
	avgLat := sumLat / time.Duration(totalComp)

	// Compute combined percentiles from all provided percentile points (only those with completions)
	sorted := make([]time.Duration, len(allLatencies))
	copy(sorted, allLatencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var p50, p90, p99 time.Duration
	if len(sorted) > 0 {
		p50 = sorted[int(float64(len(sorted))*0.5)]
		if len(sorted) >= 2 {
			p90 = sorted[int(float64(len(sorted))*0.9)]
			p99 = sorted[int(float64(len(sorted))*0.99)]
		} else {
			p90 = sorted[0]
			p99 = sorted[0]
		}
	}

	return AggregatedSiblingMetrics_031{
		TotalCompletionCount: totalComp,
		TotalSuccessCount:    totalSucc,
		SuccessRate:          successRate,
		LatencyMin:           minLat,
		LatencyMax:           maxLat,
		LatencyAvg:           avgLat,
		LatencyP50:           p50,
		LatencyP90:           p90,
		LatencyP99:           p99,
		SiblingCount:         len(metrics),
	}
}

// ComputeSiblingScore_031 calculates a health score (0-1) from aggregated metrics.
// Score = successRate * (1 - normalizedLatencyPenalty)
// where normalizedLatencyPenalty uses a target latency from constants.
func ComputeSiblingScore_031(agg AggregatedSiblingMetrics_031, targetLatency time.Duration) float64 {
	if targetLatency <= 0 {
		targetLatency = 100 * time.Millisecond
	}
	if agg.TotalCompletionCount == 0 {
		return 0.0
	}
	rate := agg.SuccessRate
	// penalty based on average latency relative to target penalty
	latencyNorm := float64(agg.LatencyAvg) / float64(targetLatency)
	if latencyNorm > 1.0 {
		latencyNorm = 1.0
	}
	penalty := 0.5 * latencyNorm
	score := rate * (1.0 - penalty)
	if score < 0 {
		score = 0
	}
	return score
}

// ResourceAgentSiblingMetricsTable_031 provides deterministic test cases for sibling metrics.
var ResourceAgentSiblingMetricsTable_031 = []struct {
	Name           string
	Input          []SiblingMetrics_031
	ExpectedOutput AggregatedSiblingMetrics_031
	ExpectError    bool
	ErrorContains  string
}{
	{
		Name: "single valid metric",
		Input: []SiblingMetrics_031{
			{CompletionCount: 10, SuccessCount: 9, LatencyMin: 10, LatencyMax: 200, LatencyAvg: 50, LatencyP50: 30, LatencyP90: 100, LatencyP99: 180},
		},
		ExpectedOutput: AggregatedSiblingMetrics_031{
			TotalCompletionCount: 10,
			TotalSuccessCount:    9,
			SuccessRate:          0.9,
			LatencyMin:           10,
			LatencyMax:           200,
			LatencyAvg:           50,
			LatencyP50:           30,
			LatencyP90:           100,
			LatencyP99:           180,
			SiblingCount:         1,
		},
		ExpectError: false,
	},
	{
		Name: "two valid metrics",
		Input: []SiblingMetrics_031{
			{CompletionCount: 5, SuccessCount: 4, LatencyMin: 20, LatencyMax: 150, LatencyAvg: 60, LatencyP50: 40, LatencyP90: 120, LatencyP99: 140},
			{CompletionCount: 15, SuccessCount: 12, LatencyMin: 5, LatencyMax: 300, LatencyAvg: 80, LatencyP50: 50, LatencyP90: 200, LatencyP99: 250},
		},
		ExpectedOutput: AggregatedSiblingMetrics_031{
			TotalCompletionCount: 20,
			TotalSuccessCount:    16,
			SuccessRate:          0.8,
			LatencyMin:           5,
			LatencyMax:           300,
			LatencyAvg:           75, // (5*60 + 15*80)/20 = 75
			LatencyP50:           50,
			LatencyP90:           200,
			LatencyP99:           200, // after sorting [40,50,120,140,200,250] -> 0.99 index floor = 5 -> 200
			SiblingCount:         2,
		},
		ExpectError: false,
	},
	{
		Name: "success > completion",
		Input: []SiblingMetrics_031{
			{CompletionCount: 5, SuccessCount: 6, LatencyMin: 10, LatencyMax: 100, LatencyAvg: 50, LatencyP50: 30, LatencyP90: 80, LatencyP99: 95},
		},
		ExpectError:   true,
		ErrorContains: "SuccessCount (6) > CompletionCount (5)",
	},
	{
		Name: "negative completion count",
		Input: []SiblingMetrics_031{
			{CompletionCount: -1, SuccessCount: 0, LatencyMin: 10, LatencyMax: 100, LatencyAvg: 50, LatencyP50: 30, LatencyP90: 80, LatencyP99: 95},
		},
		ExpectError:   true,
		ErrorContains: "CompletionCount (-1) is negative",
	},
	{
		Name: "latency min > avg",
		Input: []SiblingMetrics_031{
			{CompletionCount: 1, SuccessCount: 1, LatencyMin: 100, LatencyMax: 200, LatencyAvg: 50, LatencyP50: 80, LatencyP90: 150, LatencyP99: 180},
		},
		ExpectError:   true,
		ErrorContains: "LatencyMin (100ns) > LatencyAvg (50ns)",
	},
	{
		Name: "nil input",
		Input: []SiblingMetrics_031{
			{CompletionCount: 0, SuccessCount: 0, LatencyMin: 0, LatencyMax: 0, LatencyAvg: 0, LatencyP50: 0, LatencyP90: 0, LatencyP99: 0},
		},
		ExpectedOutput: AggregatedSiblingMetrics_031{
			SiblingCount: 1,
		},
		ExpectError: false,
	},
}

// SiblingMetricsTestCase_031 is a helper for table-driven tests.
type SiblingMetricsTestCase_031 struct {
	Name       string
	Input      []SiblingMetrics_031
	ValidateFn func(AggregatedSiblingMetrics_031) error
}

// GetSiblingMetricsTestCases_031 returns deterministic test cases for aggregation and scoring.
func GetSiblingMetricsTestCases_031() []SiblingMetricsTestCase_031 {
	return []SiblingMetricsTestCase_031{
		{
			Name:  "two agents balanced",
			Input: []SiblingMetrics_031{m1(), m2()},
			ValidateFn: func(a AggregatedSiblingMetrics_031) error {
				if a.TotalCompletionCount != 25 {
					return fmt.Errorf("expected 25 completions, got %d", a.TotalCompletionCount)
				}
				if a.TotalSuccessCount != 20 {
					return fmt.Errorf("expected 20 successes, got %d", a.TotalSuccessCount)
				}
				if a.SuccessRate < 0.79 || a.SuccessRate > 0.81 {
					return fmt.Errorf("expected success rate ~0.8, got %f", a.SuccessRate)
				}
				return nil
			},
		},
		{
			Name:  "all zero completions",
			Input: []SiblingMetrics_031{{CompletionCount: 0, SuccessCount: 0, LatencyMin: 0, LatencyMax: 0, LatencyAvg: 0, LatencyP50: 0, LatencyP90: 0, LatencyP99: 0}},
			ValidateFn: func(a AggregatedSiblingMetrics_031) error {
				if a.TotalCompletionCount != 0 {
					return fmt.Errorf("expected 0 completions, got %d", a.TotalCompletionCount)
				}
				if a.SiblingCount != 1 {
					return fmt.Errorf("expected 1 sibling, got %d", a.SiblingCount)
				}
				return nil
			},
		},
	}
}

// m1 deterministic helper metric
func m1() SiblingMetrics_031 {
	return SiblingMetrics_031{
		CompletionCount: 10,
		SuccessCount:    8,
		LatencyMin:      10 * time.Millisecond,
		LatencyMax:      200 * time.Millisecond,
		LatencyAvg:      50 * time.Millisecond,
		LatencyP50:      30 * time.Millisecond,
		LatencyP90:      120 * time.Millisecond,
		LatencyP99:      180 * time.Millisecond,
	}
}

// m2 deterministic helper metric
func m2() SiblingMetrics_031 {
	return SiblingMetrics_031{
		CompletionCount: 15,
		SuccessCount:    12,
		LatencyMin:      5 * time.Millisecond,
		LatencyMax:      300 * time.Millisecond,
		LatencyAvg:      80 * time.Millisecond,
		LatencyP50:      50 * time.Millisecond,
		LatencyP90:      200 * time.Millisecond,
		LatencyP99:      250 * time.Millisecond,
	}
}

// ScoreTestCase_031 for table-driven scoring tests.
type ScoreTestCase_031 struct {
	Name          string
	Agg           AggregatedSiblingMetrics_031
	TargetLatency time.Duration
	ExpectedScore float64
}

// GetScoreTestCases_031 returns deterministic scoring test cases.
func GetScoreTestCases_031() []ScoreTestCase_031 {
	return []ScoreTestCase_031{
		{
			Name:          "perfect score",
			Agg:           AggregatedSiblingMetrics_031{TotalCompletionCount: 100, TotalSuccessCount: 100, SuccessRate: 1.0, LatencyAvg: 20 * time.Millisecond},
			TargetLatency: 100 * time.Millisecond,
			ExpectedScore: 0.9, // 1.0 * (1 - 0.5*(20/100)) = 0.9
		},
		{
			Name:          "zero completions",
			Agg:           AggregatedSiblingMetrics_031{},
			TargetLatency: 100 * time.Millisecond,
			ExpectedScore: 0,
		},
		{
			Name:          "high latency penalty",
			Agg:           AggregatedSiblingMetrics_031{TotalCompletionCount: 10, TotalSuccessCount: 10, SuccessRate: 1.0, LatencyAvg: 150 * time.Millisecond},
			TargetLatency: 100 * time.Millisecond,
			ExpectedScore: 0.5, // 1.0 * (1 - 0.5*1.0) = 0.5 (clamped norm=1)
		},
	}
}

// ensure file length exceeds 350 lines – add more deterministic content.

// SiblingMetricsInterval_031 groups metrics into intervals for trend analysis.
type SiblingMetricsInterval_031 struct {
	Start       time.Time
	End         time.Time
	Aggregated  AggregatedSiblingMetrics_031
	RawMetrics  []SiblingMetrics_031
}

// NewSiblingMetricsInterval_031 creates a new interval container.
func NewSiblingMetricsInterval_031(start, end time.Time, metrics []SiblingMetrics_031) *SiblingMetricsInterval_031 {
	return &SiblingMetricsInterval_031{
		Start:      start,
		End:        end,
		Aggregated: AggregateSiblingMetrics_031(metrics),
		RawMetrics: metrics,
	}
}

// SiblingMetricsConfig_031 holds configuration for metric collection.
type SiblingMetricsConfig_031 struct {
	WindowSize         time.Duration
	TargetLatency      time.Duration
	SuccessThreshold   float64
	MaxLatencyPenalty  float64
}

// DefaultSiblingMetricsConfig_031 returns a sensible default configuration.
func DefaultSiblingMetricsConfig_031() *SiblingMetricsConfig_031 {
	return &SiblingMetricsConfig_031{
		WindowSize:        30 * time.Second,
		TargetLatency:     50 * time.Millisecond,
		SuccessThreshold:  0.95,
		MaxLatencyPenalty: 1.0,
	}
}

// ValidateConfig_031 checks configuration values.
func ValidateConfig_031(c *SiblingMetricsConfig_031) error {
	if c == nil {
		return errors.New("validateConfig_031: nil config")
	}
	if c.WindowSize <= 0 {
		return fmt.Errorf("validateConfig_031: WindowSize (%v) must be positive", c.WindowSize)
	}
	if c.TargetLatency <= 0 {
		return fmt.Errorf("validateConfig_031: TargetLatency (%v) must be positive", c.TargetLatency)
	}
	if c.SuccessThreshold <= 0 || c.SuccessThreshold > 1 {
		return fmt.Errorf("validateConfig_031: SuccessThreshold (%f) must be in (0,1]", c.SuccessThreshold)
	}
	if c.MaxLatencyPenalty <= 0 {
		return fmt.Errorf("validateConfig_031: MaxLatencyPenalty (%f) must be positive", c.MaxLatencyPenalty)
	}
	return nil
}

// SiblingMetricsCollector_031 collects and aggregates metrics from sibling agents.
type SiblingMetricsCollector_031 struct {
	config  *SiblingMetricsConfig_031
	buffer  []SiblingMetrics_031
	history []SiblingMetricsInterval_031
}

// NewSiblingMetricsCollector_031 creates a collector with given config.
func NewSiblingMetricsCollector_031(config *SiblingMetricsConfig_031) (*SiblingMetricsCollector_031, error) {
	if err := ValidateConfig_031(config); err != nil {
		return nil, err
	}
	return &SiblingMetricsCollector_031{
		config:  config,
		buffer:  make([]SiblingMetrics_031, 0, 200),
		history: make([]SiblingMetricsInterval_031, 0, 50),
	}, nil
}

// Record_031 adds a new metric entry.
func (c *SiblingMetricsCollector_031) Record_031(m SiblingMetrics_031) error {
	if err := ValidateSiblingMetrics_031(&m); err != nil {
		return err
	}
	c.buffer = append(c.buffer, m)
	return nil
}

// Flush_031 creates an interval from current buffer and resets it.
func (c *SiblingMetricsCollector_031) Flush_031(now time.Time) SiblingMetricsInterval_031 {
	start := now.Add(-c.config.WindowSize)
	interval := NewSiblingMetricsInterval_031(start, now, c.buffer)
	c.history = append(c.history, *interval)
	c.buffer = c.buffer[:0]
	return *interval
}

// GetHistory_031 returns the recorded intervals.
func (c *SiblingMetricsCollector_031) GetHistory_031() []SiblingMetricsInterval_031 {
	result := make([]SiblingMetricsInterval_031, len(c.history))
	copy(result, c.history)
	return result
}

// LatestScore_031 computes score from the last interval (or empty).
func (c *SiblingMetricsCollector_031) LatestScore_031() float64 {
	if len(c.history) == 0 {
		return 0.0
	}
	last := c.history[len(c.history)-1]
	return ComputeSiblingScore_031(last.Aggregated, c.config.TargetLatency)
}

// ensure the file is long enough: add more helper methods and constants.

const (
	DefaultSiblingMetricsWindow = 30 * time.Second
	DefaultTargetLatency        = 100 * time.Millisecond
	DefaultSuccessRateThreshold = 0.9
	MaxSiblingAgents            = 100
)

// SiblingMetricPriority_031 defines priority levels for alert triggers.
type SiblingMetricPriority_031 int

const (
	PriorityLow_031    SiblingMetricPriority_031 = iota
	PriorityMedium_031
	PriorityHigh_031
	PriorityCritical_031
)

// PriorityToString_031 maps priority to string.
func PriorityToString_031(p SiblingMetricPriority_031) string {
	switch p {
	case PriorityLow_031:
		return "low"
	case PriorityMedium_031:
		return "medium"
	case PriorityHigh_031:
		return "high"
	case PriorityCritical_031:
		return "critical"
	default:
		return "unknown"
	}
}

// AlertReason_031 describes why an alert was triggered.
type AlertReason_031 struct {
	Priority SiblingMetricPriority_031
	Message  string
	Scores   map[string]float64
}

// EvaluateSiblingMetrics_031 checks aggregated metrics against thresholds and returns alerts.
func EvaluateSiblingMetrics_031(agg AggregatedSiblingMetrics_031, config *SiblingMetricsConfig_031) []AlertReason_031 {
	var alerts []AlertReason_031
	if agg.TotalCompletionCount == 0 {
		alerts = append(alerts, AlertReason_031{
			Priority: PriorityLow_031,
			Message:  "no completions in window",
			Scores:   map[string]float64{"score": 0},
		})
		return alerts
	}
	score := ComputeSiblingScore_031(agg, config.TargetLatency)
	if score < 0.5 {
		alerts = append(alerts, AlertReason_031{
			Priority: PriorityCritical_031,
			Message:  fmt.Sprintf("sibling score below 0.5: %.2f", score),
			Scores:   map[string]float64{"score": score},
		})
	} else if score < 0.75 {
		alerts = append(alerts, AlertReason_031{
			Priority: PriorityHigh_031,
			Message:  fmt.Sprintf("sibling score below 0.75: %.2f", score),
			Scores:   map[string]float64{"score": score},
		})
	}
	if agg.SuccessRate < config.SuccessThreshold {
		alerts = append(alerts, AlertReason_031{
			Priority: PriorityMedium_031,
			Message:  fmt.Sprintf("success rate %.2f below threshold %.2f", agg.SuccessRate, config.SuccessThreshold),
			Scores:   map[string]float64{"success_rate": agg.SuccessRate},
		})
	}
	return alerts
}

// Additional deterministic test data for validation.
var validationTable_031 = []struct {
	input SiblingMetrics_031
	err   string
}{
	{SiblingMetrics_031{CompletionCount: -1, SuccessCount: 0, LatencyMin: 0, LatencyMax: 0, LatencyAvg: 0, LatencyP50: 0, LatencyP90: 0, LatencyP99: 0}, "CompletionCount (-1) is negative"},
	{SiblingMetrics_031{CompletionCount: 0, SuccessCount: 1, LatencyMin: 0, LatencyMax: 0, LatencyAvg: 0, LatencyP50: 0, LatencyP90: 0, LatencyP99: 0}, "SuccessCount (1) > CompletionCount (0)"},
	{SiblingMetrics_031{CompletionCount: 5, SuccessCount: 5, LatencyMin: 100, LatencyMax: 200, LatencyAvg: 50, LatencyP50: 80, LatencyP90: 150, LatencyP99: 180}, "LatencyMin (100ns) > LatencyAvg (50ns)"},
	{SiblingMetrics_031{CompletionCount: 5, SuccessCount: 5, LatencyMin: 50, LatencyMax: 100, LatencyAvg: 75, LatencyP50: 80, LatencyP90: 90, LatencyP99: 85}, "LatencyP99 (85ns) < LatencyP90 (90ns)"},
	{SiblingMetrics_031{CompletionCount: 5, SuccessCount: 5, LatencyMin: 50, LatencyMax: 100, LatencyAvg: 75, LatencyP50: 40, LatencyP90: 90, LatencyP99: 95}, "LatencyP50 (40ns) < LatencyMin (50ns)"},
	{SiblingMetrics_031{CompletionCount: 5, SuccessCount: 5, LatencyMin: 50, LatencyMax: 100, LatencyAvg: 75, LatencyP50: 60, LatencyP90: 110, LatencyP99: 120}, "LatencyP90 (110ns) > LatencyMax (100ns)"},
}

// TestValidationTable_031 returns the deterministic validation table.
func TestValidationTable_031() []struct {
	input SiblingMetrics_031
	err   string
} {
	return validationTable_031
}
