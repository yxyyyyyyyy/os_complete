package chunk039

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// SiblingOutcome represents the outcome of a sibling agent operation.
type SiblingOutcome int

const (
	SiblingOutcomeCompleted SiblingOutcome = iota
	SiblingOutcomeFailed
	SiblingOutcomeTimeout
)

// String returns a human-readable representation of the outcome.
func (o SiblingOutcome) String() string {
	switch o {
	case SiblingOutcomeCompleted:
		return "completed"
	case SiblingOutcomeFailed:
		return "failed"
	case SiblingOutcomeTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// SiblingMetricRecord007 holds a single metric entry for a sibling agent.
type SiblingMetricRecord007 struct {
	SiblingID string
	Outcome   SiblingOutcome
	Latency   time.Duration
	Timestamp time.Time
}

// SiblingAggregatorConfig007 configures the aggregator's behaviour.
type SiblingAggregatorConfig007 struct {
	WindowDuration    time.Duration
	SuccessThreshold  float64 // e.g. 0.95 for 95% success
	LatencyPercentiles []float64 // e.g. []float64{0.5, 0.9, 0.99}
	MaxRecords        int
}

// DefaultSiblingConfigs007 provides a deterministic table of default configurations.
var DefaultSiblingConfigs007 = []SiblingAggregatorConfig007{
	{WindowDuration: 5 * time.Minute, SuccessThreshold: 0.99, LatencyPercentiles: []float64{0.5, 0.95, 0.99}, MaxRecords: 10000},
	{WindowDuration: 15 * time.Minute, SuccessThreshold: 0.95, LatencyPercentiles: []float64{0.5, 0.9, 0.99}, MaxRecords: 50000},
	{WindowDuration: 1 * time.Hour, SuccessThreshold: 0.9, LatencyPercentiles: []float64{0.5, 0.8, 0.95}, MaxRecords: 200000},
}

// ValidateSiblingConfig007 validates an aggregator configuration.
func ValidateSiblingConfig007(cfg SiblingAggregatorConfig007) error {
	if cfg.WindowDuration <= 0 {
		return errors.New("WindowDuration must be positive")
	}
	if cfg.SuccessThreshold < 0 || cfg.SuccessThreshold > 1 {
		return errors.New("SuccessThreshold must be in [0,1]")
	}
	if len(cfg.LatencyPercentiles) == 0 {
		return errors.New("LatencyPercentiles must not be empty")
	}
	for i, p := range cfg.LatencyPercentiles {
		if p < 0 || p > 1 {
			return fmt.Errorf("LatencyPercentiles[%d]=%f out of [0,1]", i, p)
		}
	}
	// Check sortedness
	if !sort.Float64sAreSorted(cfg.LatencyPercentiles) {
		return errors.New("LatencyPercentiles must be sorted in ascending order")
	}
	if cfg.MaxRecords <= 0 {
		return errors.New("MaxRecords must be positive")
	}
	return nil
}

// SiblingAggregator007 aggregates completion, success, and latency metrics for sibling agents.
type SiblingAggregator007 struct {
	mu          sync.RWMutex
	config      SiblingAggregatorConfig007
	records     []SiblingMetricRecord007
	counters    map[string]map[SiblingOutcome]int
	latencySum  map[string]time.Duration
	latencyN    map[string]int
	lastReset   time.Time
}

// NewSiblingAggregator007 creates a new aggregator with the given configuration.
func NewSiblingAggregator007(config SiblingAggregatorConfig007) (*SiblingAggregator007, error) {
	if err := ValidateSiblingConfig007(config); err != nil {
		return nil, err
	}
	return &SiblingAggregator007{
		config:     config,
		records:    make([]SiblingMetricRecord007, 0, config.MaxRecords),
		counters:   make(map[string]map[SiblingOutcome]int),
		latencySum: make(map[string]time.Duration),
		latencyN:   make(map[string]int),
		lastReset:  time.Now(),
	}, nil
}

// Record007 adds a new metric record to the aggregator.
func (a *SiblingAggregator007) Record007(rec SiblingMetricRecord007) error {
	if rec.SiblingID == "" {
		return errors.New("SiblingID must not be empty")
	}
	if rec.Outcome < SiblingOutcomeCompleted || rec.Outcome > SiblingOutcomeTimeout {
		return errors.New("invalid Outcome")
	}
	if rec.Latency < 0 {
		return errors.New("Latency must be non-negative")
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.evictOldRecords007()
	if len(a.records) >= a.config.MaxRecords {
		a.records = a.records[1:]
	}
	a.records = append(a.records, rec)
	outMap, ok := a.counters[rec.SiblingID]
	if !ok {
		outMap = make(map[SiblingOutcome]int)
		a.counters[rec.SiblingID] = outMap
	}
	outMap[rec.Outcome]++
	if rec.Outcome == SiblingOutcomeCompleted {
		a.latencySum[rec.SiblingID] += rec.Latency
		a.latencyN[rec.SiblingID]++
	}
	return nil
}

// evictOldRecords007 removes records outside the configured window and rebuilds counters.
func (a *SiblingAggregator007) evictOldRecords007() {
	cutoff := time.Now().Add(-a.config.WindowDuration)
	var keep []SiblingMetricRecord007
	for _, r := range a.records {
		if !r.Timestamp.Before(cutoff) {
			keep = append(keep, r)
		}
	}
	a.records = keep
	a.rebuildCounters007()
}

// rebuildCounters007 recalculates per-sibling counters and latency sums from current records.
func (a *SiblingAggregator007) rebuildCounters007() {
	a.counters = make(map[string]map[SiblingOutcome]int)
	a.latencySum = make(map[string]time.Duration)
	a.latencyN = make(map[string]int)
	for _, r := range a.records {
		m, ok := a.counters[r.SiblingID]
		if !ok {
			m = make(map[SiblingOutcome]int)
			a.counters[r.SiblingID] = m
		}
		m[r.Outcome]++
		if r.Outcome == SiblingOutcomeCompleted {
			a.latencySum[r.SiblingID] += r.Latency
			a.latencyN[r.SiblingID]++
		}
	}
}

// SiblingStats007 holds computed statistics for a single sibling.
type SiblingStats007 struct {
	SiblingID          string
	TotalAttempts      int
	Successes          int
	Failures           int
	Timeouts           int
	SuccessRate        float64
	AvgLatency         time.Duration
	LatencyPercentiles []time.Duration
}

// ComputeStats007 computes statistics for a given sibling ID.
func (a *SiblingAggregator007) ComputeStats007(siblingID string) (SiblingStats007, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m, ok := a.counters[siblingID]
	if !ok {
		return SiblingStats007{}, fmt.Errorf("no data for sibling %q", siblingID)
	}
	completed := m[SiblingOutcomeCompleted]
	failed := m[SiblingOutcomeFailed]
	timeout := m[SiblingOutcomeTimeout]
	total := completed + failed + timeout
	successRate := 0.0
	if total > 0 {
		successRate = float64(completed) / float64(total)
	}
	avgLatency := time.Duration(0)
	if n := a.latencyN[siblingID]; n > 0 {
		avgLatency = a.latencySum[siblingID] / time.Duration(n)
	}
	latencies := a.getSiblingLatencies007(siblingID)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	pVals := make([]time.Duration, len(a.config.LatencyPercentiles))
	for i, p := range a.config.LatencyPercentiles {
		if len(latencies) == 0 {
			pVals[i] = 0
		} else {
			idx := int(math.Ceil(p*float64(len(latencies))) - 1)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(latencies) {
				idx = len(latencies) - 1
			}
			pVals[i] = latencies[idx]
		}
	}
	return SiblingStats007{
		SiblingID:          siblingID,
		TotalAttempts:      total,
		Successes:          completed,
		Failures:           failed,
		Timeouts:           timeout,
		SuccessRate:        successRate,
		AvgLatency:         avgLatency,
		LatencyPercentiles: pVals,
	}, nil
}

// getSiblingLatencies007 returns the latencies of completed operations for a sibling.
// Caller must hold at least a read lock.
func (a *SiblingAggregator007) getSiblingLatencies007(siblingID string) []time.Duration {
	var lats []time.Duration
	for _, r := range a.records {
		if r.SiblingID == siblingID && r.Outcome == SiblingOutcomeCompleted {
			lats = append(lats, r.Latency)
		}
	}
	return lats
}

// AllSiblingIDs007 returns a sorted list of all tracked sibling IDs.
func (a *SiblingAggregator007) AllSiblingIDs007() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ids := make([]string, 0, len(a.counters))
	for id := range a.counters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Reset007 clears all aggregated data and resets the last reset timestamp.
func (a *SiblingAggregator007) Reset007() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = make([]SiblingMetricRecord007, 0, a.config.MaxRecords)
	a.counters = make(map[string]map[SiblingOutcome]int)
	a.latencySum = make(map[string]time.Duration)
	a.latencyN = make(map[string]int)
	a.lastReset = time.Now()
}

// LastReset007 returns the time when the aggregator was last reset.
func (a *SiblingAggregator007) LastReset007() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastReset
}

// Configuration007 returns a copy of the current configuration.
func (a *SiblingAggregator007) Configuration007() SiblingAggregatorConfig007 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// RecordCount007 returns the number of metric records currently stored.
func (a *SiblingAggregator007) RecordCount007() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.records)
}

// SiblingMetricKey007 builds a standard metric key for a sibling and metric name.
func SiblingMetricKey007(siblingID, metricName string) string {
	return fmt.Sprintf("sibling.%s.%s", siblingID, metricName)
}

// ValidateSiblingMetricName007 checks a metric name for allowed characters (alphanumeric and underscore).
func ValidateSiblingMetricName007(name string) error {
	if name == "" {
		return errors.New("metric name must not be empty")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("invalid character %q in metric name", c)
		}
	}
	return nil
}

// SiblingMetricDescriptor007 describes a single metric.
type SiblingMetricDescriptor007 struct {
	Name        string
	Description string
	Unit        string
}

// SiblingMetricDescriptors007 is a deterministic table of standard sibling metrics.
var SiblingMetricDescriptors007 = []SiblingMetricDescriptor007{
	{Name: "completion_count", Description: "Number of completed sibling operations", Unit: "count"},
	{Name: "failure_count", Description: "Number of failed sibling operations", Unit: "count"},
	{Name: "timeout_count", Description: "Number of timed-out sibling operations", Unit: "count"},
	{Name: "success_rate", Description: "Fraction of successful sibling operations", Unit: "ratio"},
	{Name: "avg_latency", Description: "Average latency of completed sibling operations", Unit: "ns"},
	{Name: "p50_latency", Description: "50th percentile latency of completed sibling operations", Unit: "ns"},
	{Name: "p95_latency", Description: "95th percentile latency of completed sibling operations", Unit: "ns"},
	{Name: "p99_latency", Description: "99th percentile latency of completed sibling operations", Unit: "ns"},
}

// ValidateSiblingMetricDescriptor007 validates a metric descriptor.
func ValidateSiblingMetricDescriptor007(d SiblingMetricDescriptor007) error {
	if d.Name == "" {
		return errors.New("Name must not be empty")
	}
	if d.Unit == "" {
		return errors.New("Unit must not be empty")
	}
	return ValidateSiblingMetricName007(d.Name)
}

// ResourceAgentSiblingSummary007 computes aggregate stats for all tracked siblings.
func ResourceAgentSiblingSummary007(a *SiblingAggregator007) map[string]SiblingStats007 {
	ids := a.AllSiblingIDs007()
	summary := make(map[string]SiblingStats007, len(ids))
	for _, id := range ids {
		stats, err := a.ComputeStats007(id)
		if err == nil {
			summary[id] = stats
		}
	}
	return summary
}

// ResourceAgentSiblingHealth007 computes a weighted health score across all siblings based on success rates.
func ResourceAgentSiblingHealth007(stats map[string]SiblingStats007) float64 {
	if len(stats) == 0 {
		return 1.0
	}
	totalWeight := 0.0
	weightedScore := 0.0
	for _, s := range stats {
		w := float64(s.TotalAttempts)
		if w == 0 {
			continue
		}
		weightedScore += s.SuccessRate * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 1.0
	}
	return weightedScore / totalWeight
}

// ResourceAgentAggregateLatency
