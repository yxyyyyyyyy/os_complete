package chunk005

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MetricType represents the type of metric being compared.
type MetricType int

const (
	MetricCPUUsage MetricType = iota
	MetricMemoryUsage
	MetricLatency
	MetricThroughput
	MetricAllocations
	MetricCooldownTime
)

func (m MetricType) String() string {
	switch m {
	case MetricCPUUsage:
		return "cpu_usage"
	case MetricMemoryUsage:
		return "memory_usage"
	case MetricLatency:
		return "latency"
	case MetricThroughput:
		return "throughput"
	case MetricAllocations:
		return "allocations"
	case MetricCooldownTime:
		return "cooldown_time"
	default:
		return "unknown"
	}
}

// ResourceAgentBaselineData_010 holds the baseline performance data for a resource.
type ResourceAgentBaselineData_010 struct {
	ResourceID string
	SampleSize int
	Values     map[MetricType]float64
}

// ResourceAgentIsolationOnlyData_010 holds performance data under isolation-only scheduling.
type ResourceAgentIsolationOnlyData_010 struct {
	ResourceID string
	SampleSize int
	Values     map[MetricType]float64
}

// ResourceAgentAortRData_010 holds performance data under AORT-R scheduling.
type ResourceAgentAortRData_010 struct {
	ResourceID string
	SampleSize int
	Values     map[MetricType]float64
}

// ResourceAgentComparisonResult_010 stores the result of comparing two scheduling strategies.
type ResourceAgentComparisonResult_010 struct {
	ResourceID string
	Metric     MetricType
	Baseline   float64
	Isolation  float64
	AortR      float64
	// Difference: positive means improvement over baseline, negative means regression.
	IsolationImprovement float64
	AortRImprovement     float64
	// Relative improvement as percentage.
	IsolationImprovementPct float64
	AortRImprovementPct     float64
}

// ResourceAgentComparisonSummary_010 aggregates multiple comparison results.
type ResourceAgentComparisonSummary_010 struct {
	Results         []ResourceAgentComparisonResult_010
	TotalResources  int
	TotalComparisons int
	GeneratedAt     time.Time
	ValidationErrors []string
}

// ResourceAgentComparisonBuilder_010 builds comparison results between baseline, isolation-only, and AORT-R data.
type ResourceAgentComparisonBuilder_010 struct {
	baseline    []ResourceAgentBaselineData_010
	isolation   []ResourceAgentIsolationOnlyData_010
	aortR       []ResourceAgentAortRData_010
	metrics     []MetricType
	validateFlag bool
}

// NewResourceAgentComparisonBuilder_010 creates a new builder with default validation enabled.
func NewResourceAgentComparisonBuilder_010() *ResourceAgentComparisonBuilder_010 {
	return &ResourceAgentComparisonBuilder_010{
		metrics:      []MetricType{MetricCPUUsage, MetricMemoryUsage, MetricLatency, MetricThroughput},
		validateFlag: true,
	}
}

// WithBaseline sets the baseline data.
func (b *ResourceAgentComparisonBuilder_010) WithBaseline(data []ResourceAgentBaselineData_010) *ResourceAgentComparisonBuilder_010 {
	b.baseline = data
	return b
}

// WithIsolationOnly sets the isolation-only data.
func (b *ResourceAgentComparisonBuilder_010) WithIsolationOnly(data []ResourceAgentIsolationOnlyData_010) *ResourceAgentComparisonBuilder_010 {
	b.isolation = data
	return b
}

// WithAortR sets the AORT-R data.
func (b *ResourceAgentComparisonBuilder_010) WithAortR(data []ResourceAgentAortRData_010) *ResourceAgentComparisonBuilder_010 {
	b.aortR = data
	return b
}

// WithMetrics specifies which metrics to compare.
func (b *ResourceAgentComparisonBuilder_010) WithMetrics(metrics []MetricType) *ResourceAgentComparisonBuilder_010 {
	b.metrics = metrics
	return b
}

// WithValidation enables or disables validation.
func (b *ResourceAgentComparisonBuilder_010) WithValidation(flag bool) *ResourceAgentComparisonBuilder_010 {
	b.validateFlag = flag
	return b
}

// Validate_010 checks the supplied data for consistency and completeness.
// Returns an error if any issues are found.
func (b *ResourceAgentComparisonBuilder_010) Validate_010() error {
	if len(b.baseline) == 0 {
		return errors.New("baseline data is empty")
	}
	if len(b.isolation) == 0 {
		return errors.New("isolation-only data is empty")
	}
	if len(b.aortR) == 0 {
		return errors.New("aort-r data is empty")
	}
	if len(b.metrics) == 0 {
		return errors.New("no metrics specified")
	}

	baselineMap := make(map[string]ResourceAgentBaselineData_010, len(b.baseline))
	for _, d := range b.baseline {
		if d.ResourceID == "" {
			return errors.New("baseline entry missing ResourceID")
		}
		if d.SampleSize <= 0 {
			return fmt.Errorf("baseline resource %s has non-positive sample size %d", d.ResourceID, d.SampleSize)
		}
		if len(d.Values) == 0 {
			return fmt.Errorf("baseline resource %s has no metric values", d.ResourceID)
		}
		baselineMap[d.ResourceID] = d
	}

	isolationMap := make(map[string]ResourceAgentIsolationOnlyData_010, len(b.isolation))
	for _, d := range b.isolation {
		if d.ResourceID == "" {
			return errors.New("isolation-only entry missing ResourceID")
		}
		if d.SampleSize <= 0 {
			return fmt.Errorf("isolation-only resource %s has non-positive sample size %d", d.ResourceID, d.SampleSize)
		}
		if len(d.Values) == 0 {
			return fmt.Errorf("isolation-only resource %s has no metric values", d.ResourceID)
		}
		isolationMap[d.ResourceID] = d
	}

	aortRMap := make(map[string]ResourceAgentAortRData_010, len(b.aortR))
	for _, d := range b.aortR {
		if d.ResourceID == "" {
			return errors.New("aort-r entry missing ResourceID")
		}
		if d.SampleSize <= 0 {
			return fmt.Errorf("aort-r resource %s has non-positive sample size %d", d.ResourceID, d.SampleSize)
		}
		if len(d.Values) == 0 {
			return fmt.Errorf("aort-r resource %s has no metric values", d.ResourceID)
		}
		aortRMap[d.ResourceID] = d
	}

	for id := range baselineMap {
		if _, ok := isolationMap[id]; !ok {
			return fmt.Errorf("resource %s missing in isolation-only data", id)
		}
		if _, ok := aortRMap[id]; !ok {
			return fmt.Errorf("resource %s missing in aort-r data", id)
		}
	}

	for id := range isolationMap {
		if _, ok := baselineMap[id]; !ok {
			return fmt.Errorf("resource %s missing in baseline data", id)
		}
	}

	for id := range aortRMap {
		if _, ok := baselineMap[id]; !ok {
			return fmt.Errorf("resource %s missing in baseline data", id)
		}
	}

	// Check all required metrics exist in all entries.
	requiredMetrics := make(map[MetricType]bool, len(b.metrics))
	for _, m := range b.metrics {
		requiredMetrics[m] = true
	}
	for id := range baselineMap {
		for m := range requiredMetrics {
			if _, ok := baselineMap[id].Values[m]; !ok {
				return fmt.Errorf("baseline resource %s missing metric %s", id, m)
			}
			if _, ok := isolationMap[id].Values[m]; !ok {
				return fmt.Errorf("isolation-only resource %s missing metric %s", id, m)
			}
			if _, ok := aortRMap[id].Values[m]; !ok {
				return fmt.Errorf("aort-r resource %s missing metric %s", id, m)
			}
		}
	}
	return nil
}

// Build performs the comparison and returns a summary.
func (b *ResourceAgentComparisonBuilder_010) Build() (*ResourceAgentComparisonSummary_010, error) {
	if b.validateFlag {
		if err := b.Validate_010(); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Build maps for fast lookup.
	baselineMap := make(map[string]ResourceAgentBaselineData_010, len(b.baseline))
	for _, d := range b.baseline {
		baselineMap[d.ResourceID] = d
	}
	isolationMap := make(map[string]ResourceAgentIsolationOnlyData_010, len(b.isolation))
	for _, d := range b.isolation {
		isolationMap[d.ResourceID] = d
	}
	aortRMap := make(map[string]ResourceAgentAortRData_010, len(b.aortR))
	for _, d := range b.aortR {
		aortRMap[d.ResourceID] = d
	}

	var results []ResourceAgentComparisonResult_010
	for _, bl := range b.baseline {
		rID := bl.ResourceID
		iso := isolationMap[rID]
		ar := aortRMap[rID]
		for _, m := range b.metrics {
			baselineVal := bl.Values[m]
			isolationVal := iso.Values[m]
			aortRVal := ar.Values[m]
			isolationImprovement := baselineVal - isolationVal
			aortRImprovement := baselineVal - aortRVal
			var isolationPct, aortRPct float64
			if baselineVal != 0 {
				isolationPct = (isolationImprovement / baselineVal) * 100
				aortRPct = (aortRImprovement / baselineVal) * 100
			}
			results = append(results, ResourceAgentComparisonResult_010{
				ResourceID:              rID,
				Metric:                  m,
				Baseline:                baselineVal,
				Isolation:               isolationVal,
				AortR:                   aortRVal,
				IsolationImprovement:    isolationImprovement,
				AortRImprovement:        aortRImprovement,
				IsolationImprovementPct: isolationPct,
				AortRImprovementPct:     aortRPct,
			})
		}
	}

	// Sort results by resource ID then metric.
	sort.Slice(results, func(i, j int) bool {
		if results[i].ResourceID != results[j].ResourceID {
			return results[i].ResourceID < results[j].ResourceID
		}
		return results[i].Metric < results[j].Metric
	})

	summary := &ResourceAgentComparisonSummary_010{
		Results:         results,
		TotalResources:  len(b.baseline),
		TotalComparisons: len(results),
		GeneratedAt:     time.Now(),
	}
	return summary, nil
}

// ResourceAgentPredefinedScenarios_010 returns a table of deterministic comparison scenarios for testing or demo.
func ResourceAgentPredefinedScenarios_010() []struct {
	Name      string
	Baseline  []ResourceAgentBaselineData_010
	Isolation []ResourceAgentIsolationOnlyData_010
	AortR     []ResourceAgentAortRData_010
	Metrics   []MetricType
} {
	return []struct {
		Name      string
		Baseline  []ResourceAgentBaselineData_010
		Isolation []ResourceAgentIsolationOnlyData_010
		AortR     []ResourceAgentAortRData_010
		Metrics   []MetricType
	}{
		{
			Name: "StandardWorkload",
			Baseline: []ResourceAgentBaselineData_010{
				{ResourceID: "res-001", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 50.0, MetricMemoryUsage: 2048.0}},
				{ResourceID: "res-002", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 70.0, MetricMemoryUsage: 4096.0}},
			},
			Isolation: []ResourceAgentIsolationOnlyData_010{
				{ResourceID: "res-001", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 45.0, MetricMemoryUsage: 1900.0}},
				{ResourceID: "res-002", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 65.0, MetricMemoryUsage: 3800.0}},
			},
			AortR: []ResourceAgentAortRData_010{
				{ResourceID: "res-001", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 42.0, MetricMemoryUsage: 1800.0}},
				{ResourceID: "res-002", SampleSize: 100, Values: map[MetricType]float64{MetricCPUUsage: 60.0, MetricMemoryUsage: 3600.0}},
			},
			Metrics: []MetricType{MetricCPUUsage, MetricMemoryUsage},
		},
		{
			Name: "LatencySensitive",
			Baseline: []ResourceAgentBaselineData_010{
				{ResourceID: "res-003", SampleSize: 200, Values: map[MetricType]float64{MetricLatency: 120.0, MetricThroughput: 500.0}},
			},
			Isolation: []ResourceAgentIsolationOnlyData_010{
				{ResourceID: "res-003", SampleSize: 200, Values: map[MetricType]float64{MetricLatency: 110.0, MetricThroughput: 520.0}},
			},
			AortR: []ResourceAgentAortRData_010{
				{ResourceID: "res-003", SampleSize: 200, Values: map[MetricType]float64{MetricLatency: 95.0, MetricThroughput: 550.0}},
			},
			Metrics: []MetricType{MetricLatency, MetricThroughput},
		},
		{
			Name: "MixedMetrics",
			Baseline: []ResourceAgentBaselineData_010{
				{ResourceID: "res-004", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 30.0, MetricMemoryUsage: 1024.0, MetricLatency: 80.0}},
				{ResourceID: "res-005", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 90.0, MetricMemoryUsage: 8192.0, MetricLatency: 200.0}},
			},
			Isolation: []ResourceAgentIsolationOnlyData_010{
				{ResourceID: "res-004", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 28.0, MetricMemoryUsage: 1000.0, MetricLatency: 78.0}},
				{ResourceID: "res-005", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 85.0, MetricMemoryUsage: 7800.0, MetricLatency: 190.0}},
			},
			AortR: []ResourceAgentAortRData_010{
				{ResourceID: "res-004", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 25.0, MetricMemoryUsage: 950.0, MetricLatency: 70.0}},
				{ResourceID: "res-005", SampleSize: 150, Values: map[MetricType]float64{MetricCPUUsage: 80.0, MetricMemoryUsage: 7500.0, MetricLatency: 180.0}},
			},
			Metrics: []MetricType{MetricCPUUsage, MetricMemoryUsage, MetricLatency},
		},
	}
}

// ResourceAgentFormatComparisonSummary_010 returns a formatted string of the summary.
func ResourceAgentFormatComparisonSummary_010(s *ResourceAgentComparisonSummary_010) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Comparison Summary (generated %s)\n", s.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Total Resources: %d\n", s.TotalResources))
	b.WriteString(fmt.Sprintf("Total Comparisons: %d\n", s.TotalComparisons))
	if len(s.ValidationErrors) > 0 {
		b.WriteString("Validation Errors:\n")
		for _, err := range s.ValidationErrors {
			b.WriteString(fmt.Sprintf("  - %s\n", err))
		}
	}
	b.WriteString("\nResults:\n")
	for _, r := range s.Results {
		b.WriteString(fmt.Sprintf("  Resource: %s, Metric: %s\n", r.ResourceID, r.Metric))
		b.WriteString(fmt.Sprintf("    Baseline: %.2f, Isolation: %.2f, AORT-R: %.2f\n", r.Baseline, r.Isolation, r.AortR))
		b.WriteString(fmt.Sprintf("    Isolation Improvement: %.2f (%.2f%%), AORT-R Improvement: %.2f (%.2f%%)\n", r.IsolationImprovement, r.IsolationImprovementPct, r.AortRImprovement, r.AortRImprovementPct))
	}
	return b.String()
}

// ResourceAgentFindBestStrategy_010 returns which strategy (isolation or aortr) performs better on average across all metrics for a given resource.
// Returns "isolation", "aortr", or "tie".
func ResourceAgentFindBestStrategy_010(summary *ResourceAgentComparisonSummary_010, resourceID string) string {
	var isolationSum, aortRSum float64
	var count int
	for _, r := range summary.Results {
		if r.ResourceID == resourceID {
			isolationSum += r.IsolationImprovement
			aortRSum += r.AortRImprovement
			count++
		}
	}
	if count == 0 {
		return "unknown resource"
	}
	avgIsolation := isolationSum / float64(count)
	avgAortR := aortRSum / float64(count)
	if avgIsolation > avgAortR {
		return "isolation"
	} else if avgAortR > avgIsolation {
		return "aortr"
	}
	return "tie"
}

// ResourceAgentComputeOverallImprovement_010 computes the average improvement across all resources and metrics.
func ResourceAgentComputeOverallImprovement_010(summary *ResourceAgentComparisonSummary_010) (isolationAvg, aortRAvg float64) {
	if len(summary.Results) == 0 {
		return 0, 0
	}
	var isoSum, arSum float64
	for _, r := range summary.Results {
		isoSum += r.IsolationImprovementPct
		arSum += r.AortRImprovementPct
	}
	n := float64(len(summary.Results))
	return isoSum / n, arSum / n
}

// ResourceAgentComparisonPreference_010 defines which comparison aspect to prioritize.
type ResourceAgentComparisonPreference_010 int

const (
	PrefAortRWhenBeneficial ResourceAgentComparisonPreference_010 = iota
	PrefIsolationWhenBeneficial
	PrefMinOverhead
)

// ResourceAgentSuggestStrategy_010 recommends a scheduling strategy based on comparison results.
func ResourceAgentSuggestStrategy_010(summary *ResourceAgentComparisonSummary_010, pref ResourceAgentComparisonPreference_010) string {
	var resourcesBetterIsolation, resourcesBetterAortR, tieCount int
	resourceSet := make(map[string]bool)
	for _, r := range summary.Results {
		if !resourceSet[r.ResourceID] {
			resourceSet[r.ResourceID] = true
			winner := ResourceAgentFindBestStrategy_010(summary, r.ResourceID)
			switch winner {
			case "isolation":
				resourcesBetterIsolation++
			case "aortr":
				resourcesBetterAortR++
			default:
				tieCount++
			}
		}
	}

	switch pref {
	case PrefAortRWhenBeneficial:
		if float64(resourcesBetterAortR) >= float64(resourcesBetterIsolation) {
			return "recommend AORT-R"
		}
		return "recommend isolation-only"
	case PrefIsolationWhenBeneficial:
		if float64(resourcesBetterIsolation) >= float64(resourcesBetterAortR) {
			return "recommend isolation-only"
		}
		return "recommend AORT-R"
	case PrefMinOverhead:
		// Assume isolation has lower overhead.
		if resourcesBetterIsolation >= resourcesBetterAortR {
			return "recommend isolation-only (lower overhead)"
		}
		return "recommend AORT-R"
	}
	return "no clear recommendation"
}
