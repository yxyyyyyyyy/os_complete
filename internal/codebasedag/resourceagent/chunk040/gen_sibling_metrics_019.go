package chunk040

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// SiblingCompletionMetric_019 captures completion counts for a sibling agent.
type SiblingCompletionMetric_019 struct {
	// TotalAttempts is the total number of completion attempts.
	TotalAttempts uint64
	// CompletedCount is the number of successful completions.
	CompletedCount uint64
	// FailedCount is the number of failed completions.
	FailedCount uint64
	// PartialCount is the number of partial completions (e.g., timeouts).
	PartialCount uint64
}

// SiblingSuccessMetric_019 tracks success/failure ratios for a sibling agent.
type SiblingSuccessMetric_019 struct {
	// SuccessRatio is the fraction of attempts that succeeded (0.0 to 1.0).
	SuccessRatio float64
	// ConsecutiveSuccesses counts consecutive successful completions.
	ConsecutiveSuccesses uint64
	// ConsecutiveFailures counts consecutive failed completions.
	ConsecutiveFailures uint64
}

// SiblingLatencyMetric_019 stores latency statistics for a sibling agent.
type SiblingLatencyMetric_019 struct {
	// MinLatency is the minimum observed latency (in nanoseconds).
	MinLatency time.Duration
	// MaxLatency is the maximum observed latency (in nanoseconds).
	MaxLatency time.Duration
	// AvgLatency is the average latency (in nanoseconds).
	AvgLatency time.Duration
	// P50Latency is the median latency (50th percentile, in nanoseconds).
	P50Latency time.Duration
	// P90Latency is the 90th percentile latency (in nanoseconds).
	P90Latency time.Duration
	// P99Latency is the 99th percentile latency (in nanoseconds).
	P99Latency time.Duration
	// SampleCount is the number of latency measurements.
	SampleCount uint64
}

// SiblingMetricsReport_019 is a comprehensive report for a single sibling agent.
type SiblingMetricsReport_019 struct {
	// AgentID identifies the sibling agent.
	AgentID string
	// Timestamp is when this report was generated.
	Timestamp time.Time
	// Completion holds completion statistics.
	Completion SiblingCompletionMetric_019
	// Success holds success/failure statistics.
	Success SiblingSuccessMetric_019
	// Latency holds latency statistics.
	Latency SiblingLatencyMetric_019
	// Tags are optional key-value metadata.
	Tags map[string]string
}

// SiblingMetricsAggregator_019 aggregates multiple SiblingMetricsReport_019 into
// a combined summary. It is safe for concurrent use.
type SiblingMetricsAggregator_019 struct {
	mu sync.Mutex

	// agentID is the target agent for this aggregator.
	agentID string

	// reports stores all individual reports for later aggregation.
	reports []SiblingMetricsReport_019

	// rawLatencies stores all observed latencies for percentile computation.
	rawLatencies []time.Duration

	// totalAttempts, completed, failed, partial sums.
	totalAttempts    uint64
	completedCount   uint64
	failedCount      uint64
	partialCount     uint64

	// success counts.
	totalSuccesses uint64
	totalFailures  uint64

	// consecutive counters (tracked from last report order).
	lastWasSuccess bool
	consecutiveSuccesses uint64
	consecutiveFailures  uint64

	// latency extremes.
	minLatency time.Duration
	maxLatency time.Duration
	latencySum time.Duration
	latencyN   uint64
}

// NewSiblingMetricsAggregator_019 creates a new aggregator for a given agent ID.
func NewSiblingMetricsAggregator_019(agentID string) *SiblingMetricsAggregator_019 {
	return &SiblingMetricsAggregator_019{
		agentID:    agentID,
		minLatency: time.Duration(math.MaxInt64),
		maxLatency: 0,
	}
}

// AgentID_019 returns the agent ID this aggregator is tracking.
func (a *SiblingMetricsAggregator_019) AgentID_019() string {
	return a.agentID
}

// Add_019 incorporates a single metrics report into the aggregator.
// It updates all running counters and latency samples.
func (a *SiblingMetricsAggregator_019) Add_019(report SiblingMetricsReport_019) error {
	if report.AgentID != a.agentID {
		return fmt.Errorf("report agent ID %q does not match aggregator agent ID %q",
			report.AgentID, a.agentID)
	}
	if err := ValidateSiblingMetricsReport_019(&report); err != nil {
		return fmt.Errorf("invalid report: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Store raw report for later percentile recalculation.
	a.reports = append(a.reports, report)

	// Update completion sums.
	a.totalAttempts += report.Completion.TotalAttempts
	a.completedCount += report.Completion.CompletedCount
	a.failedCount += report.Completion.FailedCount
	a.partialCount += report.Completion.PartialCount

	// Update success counters.
	if report.Success.SuccessRatio >= 0.5 {
		// treat as successful report
		a.totalSuccesses++
		if a.lastWasSuccess {
			a.consecutiveSuccesses++
		} else {
			a.consecutiveSuccesses = 1
			a.consecutiveFailures = 0
		}
		a.lastWasSuccess = true
	} else {
		a.totalFailures++
		if !a.lastWasSuccess {
			a.consecutiveFailures++
		} else {
			a.consecutiveFailures = 1
			a.consecutiveSuccesses = 0
		}
		a.lastWasSuccess = false
	}

	// Update latency extremes and sums.
	if report.Latency.SampleCount > 0 {
		if report.Latency.MinLatency < a.minLatency {
			a.minLatency = report.Latency.MinLatency
		}
		if report.Latency.MaxLatency > a.maxLatency {
			a.maxLatency = report.Latency.MaxLatency
		}
		a.latencySum += report.Latency.AvgLatency * time.Duration(report.Latency.SampleCount)
		a.latencyN += report.Latency.SampleCount

		// Store raw latencies if provided in the report; we can simulate by
		// using the percentile values to populate some sample latencies.
		// For simplicity, we assume the report's latency fields represent
		// either raw data or we store the given percentiles as synthetic samples.
		a.appendLatencySamples(report.Latency)
	}

	return nil
}

// appendLatencySamples adds synthetic latency samples based on the report's
// percentile data to support later percentile recomputation.  In a real
// implementation the report might carry raw latency lists.  Here we create
// a representative set of latencies that preserve the reported percentiles.
func (a *SiblingMetricsAggregator_019) appendLatencySamples(metric SiblingLatencyMetric_019) {
	if metric.SampleCount == 0 {
		return
	}
	// We generate a small set of synthetic samples that reflect the percentiles.
	// For a proper aggregator, the report would include raw samples.
	// This is a placeholder for demonstration.
	syntheticSamples := make([]time.Duration, 0, 5)
	if metric.P50Latency > 0 {
		syntheticSamples = append(syntheticSamples, metric.P50Latency)
	}
	if metric.P90Latency > 0 {
		syntheticSamples = append(syntheticSamples, metric.P90Latency)
	}
	if metric.P99Latency > 0 {
		syntheticSamples = append(syntheticSamples, metric.P99Latency)
	}
	if metric.MinLatency > 0 {
		syntheticSamples = append(syntheticSamples, metric.MinLatency)
	}
	if metric.MaxLatency > 0 {
		syntheticSamples = append(syntheticSamples, metric.MaxLatency)
	}
	if len(syntheticSamples) == 0 {
		// fallback: use avg latency
		if metric.AvgLatency > 0 {
			syntheticSamples = append(syntheticSamples, metric.AvgLatency)
		} else {
			return
		}
	}
	a.rawLatencies = append(a.rawLatencies, syntheticSamples...)
}

// Merge_019 combines another aggregator's data into this aggregator.
func (a *SiblingMetricsAggregator_019) Merge_019(other *SiblingMetricsAggregator_019) error {
	if other.agentID != a.agentID {
		return fmt.Errorf("cannot merge aggregator for agent %q into aggregator for agent %q",
			other.agentID, a.agentID)
	}
	other.mu.Lock()
	defer other.mu.Unlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.reports = append(a.reports, other.reports...)
	a.totalAttempts += other.totalAttempts
	a.completedCount += other.completedCount
	a.failedCount += other.failedCount
	a.partialCount += other.partialCount
	a.totalSuccesses += other.totalSuccesses
	a.totalFailures += other.totalFailures
	// For consecutive counts, we take the last state from the other aggregator.
	// This is a simplification; a more robust merge would sequence reports.
	if other.lastWasSuccess {
		a.consecutiveSuccesses += other.consecutiveSuccesses
		a.lastWasSuccess = true
	} else {
		a.consecutiveFailures += other.consecutiveFailures
		a.lastWasSuccess = false
	}
	if other.minLatency < a.minLatency {
		a.minLatency = other.minLatency
	}
	if other.maxLatency > a.maxLatency {
		a.maxLatency = other.maxLatency
	}
	a.latencySum += other.latencySum
	a.latencyN += other.latencyN
	a.rawLatencies = append(a.rawLatencies, other.rawLatencies...)
	return nil
}

// Summary_019 computes and returns the aggregated metrics report.
func (a *SiblingMetricsAggregator_019) Summary_019() SiblingMetricsReport_019 {
	a.mu.Lock()
	defer a.mu.Unlock()

	report := SiblingMetricsReport_019{
		AgentID:   a.agentID,
		Timestamp: time.Now(),
	}

	// Completion
	report.Completion = SiblingCompletionMetric_019{
		TotalAttempts:  a.totalAttempts,
		CompletedCount: a.completedCount,
		FailedCount:    a.failedCount,
		PartialCount:   a.partialCount,
	}

	// Success
	var successRatio float64
	if a.totalAttempts > 0 {
		successRatio = float64(a.completedCount) / float64(a.totalAttempts)
	}
	report.Success = SiblingSuccessMetric_019{
		SuccessRatio:         successRatio,
		ConsecutiveSuccesses: a.consecutiveSuccesses,
		ConsecutiveFailures:  a.consecutiveFailures,
	}

	// Latency
	latency := SiblingLatencyMetric_019{
		SampleCount: a.latencyN,
	}
	if a.latencyN > 0 {
		latency.MinLatency = a.minLatency
		latency.MaxLatency = a.maxLatency
		latency.AvgLatency = time.Duration(int64(a.latencySum) / int64(a.latencyN))
	}
	if len(a.rawLatencies) > 0 {
		sorted := make([]time.Duration, len(a.rawLatencies))
		copy(sorted, a.rawLatencies)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		latency.P50Latency = percentileDuration(sorted, 50)
		latency.P90Latency = percentileDuration(sorted, 90)
		latency.P99Latency = percentileDuration(sorted, 99)
	}
	report.Latency = latency

	// Tags: merge all unique tags from all reports
	report.Tags = make(map[string]string)
	for _, r := range a.reports {
		for k, v := range r.Tags {
			report.Tags[k] = v
		}
	}

	return report
}

// percentileDuration returns the p-th percentile from a sorted slice of durations.
// p must be between 0 and 100.
func percentileDuration(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	index := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// ValidateSiblingMetricsReport_019 validates a metrics report and returns an
// error if any field is inconsistent or out of range.
func ValidateSiblingMetricsReport_019(report *SiblingMetricsReport_019) error {
	if report == nil {
		return errors.New("nil report")
	}
	if report.AgentID == "" {
		return errors.New("agent ID must not be empty")
	}
	if report.Timestamp.IsZero() {
		return errors.New("timestamp must not be zero")
	}
	// Validate completion fields
	if report.Completion.TotalAttempts == 0 &&
		report.Completion.CompletedCount == 0 &&
		report.Completion.FailedCount == 0 &&
		report.Completion.PartialCount == 0 {
		// zero values allowed (empty report)
	} else {
		if report.Completion.TotalAttempts < report.Completion.CompletedCount+report.Completion.FailedCount+report.Completion.PartialCount {
			return fmt.Errorf("total attempts %d less than sum of completed (%d), failed (%d), partial (%d)",
				report.Completion.TotalAttempts,
				report.Completion.CompletedCount,
				report.Completion.FailedCount,
				report.Completion.PartialCount)
		}
	}
	// Validate success ratio
	if report.Success.SuccessRatio < 0.0 || report.Success.SuccessRatio > 1.0 {
		return fmt.Errorf("success ratio %f out of range [0,1]", report.Success.SuccessRatio)
	}
	// Validate latency fields
	if report.Latency.SampleCount > 0 {
		if report.Latency.MinLatency < 0 {
			return fmt.Errorf("min latency %v is negative", report.Latency.MinLatency)
		}
		if report.Latency.MaxLatency < report.Latency.MinLatency {
			return fmt.Errorf("max latency %v less than min latency %v",
				report.Latency.MaxLatency, report.Latency.MinLatency)
		}
	}
	if report.Latency.P50Latency < 0 || report.Latency.P90Latency < 0 || report.Latency.P99Latency < 0 {
		return fmt.Errorf("latency percentiles must be non-negative")
	}
	return nil
}

// ValidateSiblingAggregator_019 checks internal consistency of an aggregator.
func ValidateSiblingAggregator_019(agg *SiblingMetricsAggregator_019) error {
	if agg == nil {
		return errors.New("nil aggregator")
	}
	if agg.agentID == "" {
		return errors.New("aggregator agent ID is empty")
	}
	return nil
}

// ResourceAgentSiblingMetricsDefaultTags_019 returns a default set of tags
// for sibling metrics reports.
func ResourceAgentSiblingMetricsDefaultTags_019() map[string]string {
	return map[string]string{
		"source":      "resource-agent",
		"metric-type": "sibling",
		"version":     "019",
	}
}

// Table-driven helper data for testing.
// SiblingMetricsTestData_019 returns a slice of sample reports for
// various scenarios (normal, edge, error).
func SiblingMetricsTestData_019() []SiblingMetricsReport_019 {
	return []SiblingMetricsReport_019{
		// Normal healthy sibling
		{
			AgentID:   "sibling-1",
			Timestamp: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
			Completion: SiblingCompletionMetric_019{
				TotalAttempts:  1000,
				CompletedCount: 950,
				FailedCount:    30,
				PartialCount:   20,
			},
			Success: SiblingSuccessMetric_019{
				SuccessRatio:         0.95,
				ConsecutiveSuccesses: 50,
				ConsecutiveFailures:  2,
			},
			Latency: SiblingLatencyMetric_019{
				MinLatency:  5 * time.Millisecond,
				MaxLatency:  200 * time.Millisecond,
				AvgLatency:  25 * time.Millisecond,
				P50Latency:  20 * time.Millisecond,
				P90Latency:  60 * time.Millisecond,
				P99Latency:  150 * time.Millisecond,
				SampleCount: 1000,
			},
			Tags: map[string]string{"region": "us-east"},
		},
		// Sibling with high failure rate
		{
			AgentID:   "sibling-2",
			Timestamp: time.Date(2024, 6, 1, 10, 5, 0, 0, time.UTC),
			Completion: SiblingCompletionMetric_019{
				TotalAttempts:  500,
				CompletedCount: 100,
				FailedCount:    400,
				PartialCount:   0,
			},
			Success: SiblingSuccessMetric_019{
				SuccessRatio:         0.20,
				ConsecutiveSuccesses: 0,
				ConsecutiveFailures:  15,
			},
			Latency: SiblingLatencyMetric_019{
				MinLatency:  10 * time.Millisecond,
				MaxLatency:  500 * time.Millisecond,
				AvgLatency:  120 * time.Millisecond,
				P50Latency:  100 * time.Millisecond,
				P90Latency:  300 * time.Millisecond,
				P99Latency:  450 * time.Millisecond,
				SampleCount: 500,
			},
			Tags: map[string]string{"region": "eu-west"},
		},
		// Sibling with no attempts
		{
			AgentID:   "sibling-3",
			Timestamp: time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC),
			Completion: SiblingCompletionMetric_019{
				TotalAttempts:  0,
				CompletedCount: 0,
				FailedCount:    0,
				PartialCount:   0,
			},
			Success: SiblingSuccessMetric_019{
				SuccessRatio:         0.0,
				ConsecutiveSuccesses: 0,
				ConsecutiveFailures:  0,
			},
			Latency: SiblingLatencyMetric_019{
				SampleCount: 0,
			},
			Tags: map[string]string{"region": "ap-southeast"},
		},
		// Sibling with extreme latency
		{
			AgentID:   "sibling-4",
			Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
			Completion: SiblingCompletionMetric_019{
				TotalAttempts:  10,
				CompletedCount: 9,
				FailedCount:    1,
				PartialCount:   0,
			},
			Success: SiblingSuccessMetric_019{
				SuccessRatio:         0.9,
				ConsecutiveSuccesses: 5,
				ConsecutiveFailures:  1,
			},
			Latency: SiblingLatencyMetric_019{
				MinLatency:  1 * time.Second,
				MaxLatency:  30 * time.Second,
				AvgLatency:  5 * time.Second,
				P50Latency:  3 * time.Second,
				P90Latency:  15 * time.Second,
				P99Latency:  25 * time.Second,
				SampleCount: 10,
			},
			Tags: map[string]string{"region": "us-west", "tier": "slow"},
		},
		// Sibling with partial completions only
		{
			AgentID:   "sibling-5",
			Timestamp: time.Date(2024, 6, 1, 13, 0, 0, 0, time.UTC),
			Completion: SiblingCompletionMetric_019{
				TotalAttempts:  100,
				CompletedCount: 0,
				FailedCount:    0,
				PartialCount:   100,
			},
			Success: SiblingSuccessMetric_019{
				SuccessRatio:         0.0,
				ConsecutiveSuccesses: 0,
				ConsecutiveFailures:  10,
			},
			Latency: SiblingLatencyMetric_019{
				MinLatency:  50 * time.Millisecond,
				MaxLatency:  500 * time.Millisecond,
				AvgLatency:  200 * time.Millisecond,
				P50Latency:  180 * time.Millisecond,
				P90Latency:  400 * time.Millisecond,
				P99Latency:  490 * time.Millisecond,
				SampleCount: 100,
			},
			Tags: map[string]string{"region": "sa-east", "status": "partial"},
		},
	}
}

// ResourceAgentSiblingLatencyBuckets_019 returns a table of predefined latency
// buckets for histogram aggregation (in nanoseconds).  Keys are bucket upper
// bounds (exclusive), values are bucket labels.
func ResourceAgentSiblingLatencyBuckets_019() map[time.Duration]string {
	return map[time.Duration]string{
		1 * time.Millisecond:       "<1ms",
		5 * time.Millisecond:       "1-5ms",
		10 * time.Millisecond:      "5-10ms",
		25 * time.Millisecond:      "10-25ms",
		50 * time.Millisecond:      "25-50ms",
		100 * time.Millisecond:     "50-100ms",
		250 * time.Millisecond:     "100-250ms",
		500 * time.Millisecond:     "250-500ms",
		1 * time.Second:            "500ms-1s",
		5 * time.Second:            "1-5s",
		time.Duration(math.MaxInt64): ">5s",
	}
}

// ResourceAgentSiblingAggregateSummary_019 produces a human-readable summary
// string from an aggregated report.
func ResourceAgentSiblingAggregateSummary_019(report SiblingMetricsReport_019) string {
	return fmt.Sprintf("Agent %s: total attempts=%d, completed=%d, failed=%d, partial=%d, "+
		"successRatio=%.2f, consecutiveSuccesses=%d, consecutiveFailures=%d, "+
		"latency: min=%v, avg=%v, max=%v, p50=%v, p90=%v, p99=%v, samples=%d",
		report.AgentID,
		report.Completion.TotalAttempts,
		report.Completion.CompletedCount,
		report.Completion.FailedCount,
		report.Completion.PartialCount,
		report.Success.SuccessRatio,
		report.Success.ConsecutiveSuccesses,
		report.Success.ConsecutiveFailures,
		report.Latency.MinLatency,
		report.Latency.AvgLatency,
		report.Latency.MaxLatency,
		report.Latency.P50Latency,
		report.Latency.P90Latency,
		report.Latency.P99Latency,
		report.Latency.SampleCount,
	)
}

// ResourceAgentSiblingReportFromRaw_019 constructs a SiblingMetricsReport_019
// from raw observations.
func ResourceAgentSiblingReportFromRaw_019(agentID string, completions []bool, latencies []time.Duration) SiblingMetricsReport_019 {
	var report SiblingMetricsReport_019
	report.AgentID = agentID
	report.Timestamp = time.Now()
	report.Tags = ResourceAgentSiblingMetricsDefaultTags_019()

	if len(completions) > 0 {
		var completed, failed, partial uint64
		for _, c := range completions {
			if c {
				completed++
			} else {
				failed++
			}
		}
		report.Completion = SiblingCompletionMetric_019{
			TotalAttempts:  uint64(len(completions)),
			CompletedCount: completed,
			FailedCount:    failed,
			PartialCount:   partial,
		}
		if len(completions) > 0 {
			report.Success.SuccessRatio = float64(completed) / float64(len(completions))
		}
		// Consecutive tracking
		consecSuccess := uint64(0)
		consecFail := uint64(0)
		maxConsecSuccess := uint64(0)
		maxConsecFail := uint64(0)
		for _, c := range completions {
			if c {
				consecSuccess++
				consecFail = 0
				if consecSuccess > maxConsecSuccess {
					maxConsecSuccess = consecSuccess
				}
			} else {
				consecFail++
				consecSuccess = 0
				if consecFail > maxConsecFail {
					maxConsecFail = consecFail
				}
			}
		}
		report.Success.ConsecutiveSuccesses = maxConsecSuccess
		report.Success.ConsecutiveFailures = maxConsecFail
	}

	if len(latencies) > 0 {
		sorted := make([]time.Duration, len(latencies))
		copy(sorted, latencies)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		report.Latency = SiblingLatencyMetric_019{
			MinLatency:  sorted[0],
			MaxLatency:  sorted[len(sorted)-1],
			AvgLatency:  time.Duration(int64(sum) / int64(len(latencies))),
			P50Latency:  percentileDuration(sorted, 50),
			P90Latency:  percentileDuration(sorted, 90),
			P99Latency:  percentileDuration(sorted, 99),
			SampleCount: uint64(len(latencies)),
		}
	}
	return report
}

// ResourceAgentSiblingNormalizeReport_019 ensures report fields are within
// expected ranges (e.g., non-negative counts, clamped ratio).  Returns a copy.
func ResourceAgentSiblingNormalizeReport_019(report SiblingMetricsReport_019) SiblingMetricsReport_019 {
	normalized := report

	// Clamp success ratio
	if normalized.Success.SuccessRatio < 0.0 {
		normalized.Success.SuccessRatio = 0.0
	}
	if normalized.Success.SuccessRatio > 1.0 {
		normalized.Success.SuccessRatio = 1.0
	}

	// Ensure counts are non-negative
	if normalized.Completion.TotalAttempts < 0 {
		normalized.Completion.TotalAttempts = 0
	}
	if normalized.Completion.CompletedCount < 0 {
		normalized.Completion.CompletedCount = 0
	}
	if normalized.Completion.FailedCount < 0 {
		normalized.Completion.FailedCount = 0
	}
	if normalized.Completion.PartialCount < 0 {
		normalized.Completion.PartialCount = 0
	}

	// Ensure totals make sense
	nonNegativeSum := normalized.Completion.CompletedCount +
		normalized.Completion.FailedCount +
		normalized.Completion.PartialCount
	if nonNegativeSum > normalized.Completion.TotalAttempts {
		// adjust total attempts upward if needed (conservative)
		normalized.Completion.TotalAttempts = nonNegativeSum
	}

	// Latency fields
	if normalized.Latency.MinLatency < 0 {
		normalized.Latency.MinLatency = 0
	}
	if normalized.Latency.MaxLatency < normalized.Latency.MinLatency {
		normalized.Latency.MaxLatency = normalized.Latency.MinLatency
	}
	if normalized.Latency.P50Latency < 0 {
		normalized.Latency.P50Latency = 0
	}
	if normalized.Latency.P90Latency < normalized.Latency.P50Latency {
		normalized.Latency.P90Latency = normalized.Latency.P50Latency
	}
	if normalized.Latency.P99Latency < normalized.Latency.P90Latency {
		normalized.Latency.P99Latency = normalized.Latency.P90Latency
	}
	if normalized.Latency.SampleCount < 0 {
		normalized.Latency.SampleCount = 0
	}

	return normalized
}
