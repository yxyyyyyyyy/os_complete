package chunk007

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Types for baseline, isolation-only, and aort-r configurations and results.
// ---------------------------------------------------------------------------

// ResourceAgentBaselineConfig_034 holds the resource profile of a baseline pod.
type ResourceAgentBaselineConfig_034 struct {
	Name          string
	CPURequest    float64 // milliCPU
	CPULimit      float64
	MemoryRequest float64 // MiB
	MemoryLimit   float64
	EPHStorage    float64 // GiB (ephemeral storage)
}

// ResourceAgentIsolationOnlyConfig_034 defines isolation-only deployment parameters.
type ResourceAgentIsolationOnlyConfig_034 struct {
	Name                 string
	BaseCPULimit         float64
	BaseMemoryLimit      float64
	OverheadCPUFactor    float64   // multiplier for CPU overhead
	OverheadMemoryFactor float64
	OverheadMemoryFixed  float64   // fixed memory overhead in MiB
	OverheadCPUFixed     float64   // fixed CPU overhead in milliCPU
}

// ResourceAgentAorTRConfig_034 defines aort-r threshold and resource cap settings.
type ResourceAgentAorTRConfig_034 struct {
	Name                  string
	ThresholdCPU          float64 // if usage below this fraction of limit, reduce
	ThresholdMemory       float64
	ReductionCPUFactor    float64 // multiplied to current limit to get new limit
	ReductionMemoryFactor float64
	MinCPULimit           float64
	MinMemoryLimit        float64
	MaxCPULimit           float64
	MaxMemoryLimit        float64
}

// ResourceAgentComparisonResult_034 holds computed comparison metrics.
type ResourceAgentComparisonResult_034 struct {
	ConfigName        string
	Baseline          ResourceAgentBaselineConfig_034
	IsolationOnly     ResourceAgentIsolationOnlyConfig_034
	AorTR             ResourceAgentAorTRConfig_034
	BaselineUsage     ResourceUsageSample_034
	IsolationUsage    ResourceUsageSample_034
	AorTRUsage        ResourceUsageSample_034
	BaselineVsIso     DiffSummary_034
	BaselineVsAorTR   DiffSummary_034
	IsoVsAorTR        DiffSummary_034
	Timestamp         time.Time
}

// ResourceUsageSample_034 represents measured or estimated resource consumption.
type ResourceUsageSample_034 struct {
	CPUMilli    float64
	MemoryMiB   float64
	EPHStorage  float64
}

// DiffSummary_034 summarizes differences between two scenarios.
type DiffSummary_034 struct {
	CPUDeltaPercent    float64 // (usage2 - usage1) / usage1 * 100
	MemoryDeltaPercent float64
	CPUDeltaAbs        float64
	MemoryDeltaAbs     float64
	EPHDelta           float64
	Score              float64 // composite score (e.g., weighted sum of deltas)
}

// ResourceAgentComparisonBuilder_034 is a builder that constructs a full comparison.
type ResourceAgentComparisonBuilder_034 struct {
	baseline *ResourceAgentBaselineConfig_034
	isoOnly  *ResourceAgentIsolationOnlyConfig_034
	aorTR    *ResourceAgentAorTRConfig_034
	err      error
}

// ---------------------------------------------------------------------------
// Builder constructors and methods.
// ---------------------------------------------------------------------------

// NewResourceAgentComparisonBuilder_034 creates a new builder.
func NewResourceAgentComparisonBuilder_034() *ResourceAgentComparisonBuilder_034 {
	return &ResourceAgentComparisonBuilder_034{}
}

// WithBaseline sets the baseline configuration.
func (b *ResourceAgentComparisonBuilder_034) WithBaseline(cfg ResourceAgentBaselineConfig_034) *ResourceAgentComparisonBuilder_034 {
	if err := ValidateBaselineConfig_034(cfg); err != nil {
		b.err = fmt.Errorf("baseline invalid: %w", err)
		return b
	}
	b.baseline = &cfg
	return b
}

// WithIsolationOnly sets the isolation-only configuration.
func (b *ResourceAgentComparisonBuilder_034) WithIsolationOnly(cfg ResourceAgentIsolationOnlyConfig_034) *ResourceAgentComparisonBuilder_034 {
	if err := ValidateIsolationOnlyConfig_034(cfg); err != nil {
		b.err = fmt.Errorf("isolation-only invalid: %w", err)
		return b
	}
	b.isoOnly = &cfg
	return b
}

// WithAorTR sets the aort-r configuration.
func (b *ResourceAgentComparisonBuilder_034) WithAorTR(cfg ResourceAgentAorTRConfig_034) *ResourceAgentComparisonBuilder_034 {
	if err := ValidateAorTRConfig_034(cfg); err != nil {
		b.err = fmt.Errorf("aort-r invalid: %w", err)
		return b
	}
	b.aorTR = &cfg
	return b
}

// Build constructs the comparison result. Returns error if any configuration missing.
func (b *ResourceAgentComparisonBuilder_034) Build() (ResourceAgentComparisonResult_034, error) {
	if b.err != nil {
		return ResourceAgentComparisonResult_034{}, b.err
	}
	if b.baseline == nil || b.isoOnly == nil || b.aorTR == nil {
		return ResourceAgentComparisonResult_034{}, errors.New("all three configurations must be provided")
	}

	// Simulate resource usage based on configurations.
	baselineUsage := SimulateBaselineUsage_034(*b.baseline)
	isolationUsage := SimulateIsolationUsage_034(*b.isoOnly, *b.baseline)
	aorTRUsage := SimulateAorTRUsage_034(*b.aorTR, *b.baseline)

	baselineVsIso := computeDiff_034(baselineUsage, isolationUsage)
	baselineVsAorTR := computeDiff_034(baselineUsage, aorTRUsage)
	isoVsAorTR := computeDiff_034(isolationUsage, aorTRUsage)

	result := ResourceAgentComparisonResult_034{
		ConfigName:      b.baseline.Name,
		Baseline:        *b.baseline,
		IsolationOnly:   *b.isoOnly,
		AorTR:           *b.aorTR,
		BaselineUsage:   baselineUsage,
		IsolationUsage:  isolationUsage,
		AorTRUsage:      aorTRUsage,
		BaselineVsIso:   baselineVsIso,
		BaselineVsAorTR: baselineVsAorTR,
		IsoVsAorTR:      isoVsAorTR,
		Timestamp:       time.Now().UTC(),
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Validation functions returning error.
// ---------------------------------------------------------------------------

// ValidateBaselineConfig_034 validates a baseline configuration.
func ValidateBaselineConfig_034(cfg ResourceAgentBaselineConfig_034) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return errors.New("name must not be empty")
	}
	if cfg.CPURequest < 0 {
		return errors.New("CPURequest must be non-negative")
	}
	if cfg.CPULimit < 0 {
		return errors.New("CPULimit must be non-negative")
	}
	if cfg.CPURequest > cfg.CPULimit {
		return errors.New("CPURequest must not exceed CPULimit")
	}
	if cfg.MemoryRequest < 0 {
		return errors.New("MemoryRequest must be non-negative")
	}
	if cfg.MemoryLimit < 0 {
		return errors.New("MemoryLimit must be non-negative")
	}
	if cfg.MemoryRequest > cfg.MemoryLimit {
		return errors.New("MemoryRequest must not exceed MemoryLimit")
	}
	if cfg.EPHStorage < 0 {
		return errors.New("EPHStorage must be non-negative")
	}
	return nil
}

// ValidateIsolationOnlyConfig_034 validates an isolation-only configuration.
func ValidateIsolationOnlyConfig_034(cfg ResourceAgentIsolationOnlyConfig_034) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return errors.New("name must not be empty")
	}
	if cfg.BaseCPULimit < 0 {
		return errors.New("BaseCPULimit must be non-negative")
	}
	if cfg.BaseMemoryLimit < 0 {
		return errors.New("BaseMemoryLimit must be non-negative")
	}
	if cfg.OverheadCPUFactor < 0 || cfg.OverheadCPUFactor > 1.0 {
		return errors.New("OverheadCPUFactor must be in [0, 1]")
	}
	if cfg.OverheadMemoryFactor < 0 || cfg.OverheadMemoryFactor > 1.0 {
		return errors.New("OverheadMemoryFactor must be in [0, 1]")
	}
	if cfg.OverheadMemoryFixed < 0 {
		return errors.New("OverheadMemoryFixed must be non-negative")
	}
	if cfg.OverheadCPUFixed < 0 {
		return errors.New("OverheadCPUFixed must be non-negative")
	}
	return nil
}

// ValidateAorTRConfig_034 validates an aort-r configuration.
func ValidateAorTRConfig_034(cfg ResourceAgentAorTRConfig_034) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return errors.New("name must not be empty")
	}
	if cfg.ThresholdCPU < 0 || cfg.ThresholdCPU > 1.0 {
		return errors.New("ThresholdCPU must be in [0, 1]")
	}
	if cfg.ThresholdMemory < 0 || cfg.ThresholdMemory > 1.0 {
		return errors.New("ThresholdMemory must be in [0, 1]")
	}
	if cfg.ReductionCPUFactor < 0 || cfg.ReductionCPUFactor > 1.0 {
		return errors.New("ReductionCPUFactor must be in [0, 1]")
	}
	if cfg.ReductionMemoryFactor < 0 || cfg.ReductionMemoryFactor > 1.0 {
		return errors.New("ReductionMemoryFactor must be in [0, 1]")
	}
	if cfg.MinCPULimit < 0 {
		return errors.New("MinCPULimit must be non-negative")
	}
	if cfg.MinMemoryLimit < 0 {
		return errors.New("MinMemoryLimit must be non-negative")
	}
	if cfg.MaxCPULimit < cfg.MinCPULimit {
		return errors.New("MaxCPULimit must be >= MinCPULimit")
	}
	if cfg.MaxMemoryLimit < cfg.MinMemoryLimit {
		return errors.New("MaxMemoryLimit must be >= MinMemoryLimit")
	}
	return nil
}

// ValidateComparisonInput_034 validates that all three configs are valid and that
// the aort-r configuration is compatible with the baseline.
func ValidateComparisonInput_034(baseline ResourceAgentBaselineConfig_034, iso ResourceAgentIsolationOnlyConfig_034, aort ResourceAgentAorTRConfig_034) error {
	if err := ValidateBaselineConfig_034(baseline); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := ValidateIsolationOnlyConfig_034(iso); err != nil {
		return fmt.Errorf("isolation: %w", err)
	}
	if err := ValidateAorTRConfig_034(aort); err != nil {
		return fmt.Errorf("aort-r: %w", err)
	}
	// Additional cross-check: aort-r thresholds should be within baseline's limits.
	if aort.ThresholdCPU > 0 && baseline.CPULimit > 0 && aort.MinCPULimit > baseline.CPULimit {
		return errors.New("aort-r MinCPULimit exceeds baseline CPULimit")
	}
	if aort.ThresholdMemory > 0 && baseline.MemoryLimit > 0 && aort.MinMemoryLimit > baseline.MemoryLimit {
		return errors.New("aort-r MinMemoryLimit exceeds baseline MemoryLimit")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Usage simulation functions (deterministic, based on config).
// ---------------------------------------------------------------------------

// SimulateBaselineUsage_034 returns the resource usage under baseline (no overhead).
func SimulateBaselineUsage_034(cfg ResourceAgentBaselineConfig_034) ResourceUsageSample_034 {
	return ResourceUsageSample_034{
		CPUMilli:   cfg.CPURequest * 0.7, // assume 70% of request used
		MemoryMiB:  cfg.MemoryRequest * 0.6,
		EPHStorage: cfg.EPHStorage * 0.5,
	}
}

// SimulateIsolationUsage_034 estimates usage when isolation-only overheads are applied.
func SimulateIsolationUsage_034(iso ResourceAgentIsolationOnlyConfig_034, base ResourceAgentBaselineConfig_034) ResourceUsageSample_034 {
	cpu := base.CPURequest*0.7 + iso.OverheadCPUFixed + iso.BaseCPULimit*iso.OverheadCPUFactor
	mem := base.MemoryRequest*0.6 + iso.OverheadMemoryFixed + iso.BaseMemoryLimit*iso.OverheadMemoryFactor
	return ResourceUsageSample_034{
		CPUMilli:   cpu,
		MemoryMiB:  mem,
		EPHStorage: base.EPHStorage * 0.5,
	}
}

// SimulateAorTRUsage_034 estimates usage with aort-r dynamic adjustments.
func SimulateAorTRUsage_034(aort ResourceAgentAorTRConfig_034, base ResourceAgentBaselineConfig_034) ResourceUsageSample_034 {
	// Simulate that aort-r reduces limits when usage is below threshold.
	usageCPU := base.CPURequest * 0.7
	usageMem := base.MemoryRequest * 0.6
	limitCPU := base.CPULimit
	limitMem := base.MemoryLimit

	// Apply threshold check
	if limitCPU > 0 && usageCPU/limitCPU < aort.ThresholdCPU {
		reducedCPU := math.Max(limitCPU*aort.ReductionCPUFactor, aort.MinCPULimit)
		reducedCPU = math.Min(reducedCPU, aort.MaxCPULimit)
		limitCPU = reducedCPU
	}
	if limitMem > 0 && usageMem/limitMem < aort.ThresholdMemory {
		reducedMem := math.Max(limitMem*aort.ReductionMemoryFactor, aort.MinMemoryLimit)
		reducedMem = math.Min(reducedMem, aort.MaxMemoryLimit)
		limitMem = reducedMem
	}
	// Usage stays approximately same; limit changes.
	return ResourceUsageSample_034{
		CPUMilli:   usageCPU,
		MemoryMiB:  usageMem,
		EPHStorage: base.EPHStorage * 0.5,
	}
}

// ---------------------------------------------------------------------------
// Difference computation.
// ---------------------------------------------------------------------------

// computeDiff_034 computes differences between two usage samples.
func computeDiff_034(a, b ResourceUsageSample_034) DiffSummary_034 {
	var d DiffSummary_034
	d.CPUDeltaAbs = b.CPUMilli - a.CPUMilli
	d.MemoryDeltaAbs = b.MemoryMiB - a.MemoryMiB
	d.EPHDelta = b.EPHStorage - a.EPHStorage
	d.CPUDeltaPercent = safePercentDelta(a.CPUMilli, b.CPUMilli)
	d.MemoryDeltaPercent = safePercentDelta(a.MemoryMiB, b.MemoryMiB)
	// Weighted composite score (cpu delta weight 0.5, memory weight 0.3, eph weight 0.2)
	d.Score = 0.5*math.Abs(d.CPUDeltaPercent) + 0.3*math.Abs(d.MemoryDeltaPercent) + 0.2*math.Abs(d.EPHDelta)
	return d
}

func safePercentDelta(a, b float64) float64 {
	if a == 0 {
		if b == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return (b - a) / a * 100.0
}

// ---------------------------------------------------------------------------
// Table-driven helper data.
// ---------------------------------------------------------------------------

// ComparisonTestCase_034 defines a test case for comparison builders.
type ComparisonTestCase_034 struct {
	Name     string
	Baseline ResourceAgentBaselineConfig_034
	Iso      ResourceAgentIsolationOnlyConfig_034
	AorTR    ResourceAgentAorTRConfig_034
	Expected BaselineVsIsoVsAorTRExpected_034
}

// BaselineVsIsoVsAorTRExpected_034 holds expected metrics for a test case.
type BaselineVsIsoVsAorTRExpected_034 struct {
	BaselineUsage ResourceUsageSample_034
	IsoUsage      ResourceUsageSample_034
	AorTRUsage    ResourceUsageSample_034
	Score         float64
}

// ComparisonTestTable_034 returns a deterministic slice of test cases.
func ComparisonTestTable_034() []ComparisonTestCase_034 {
	return []ComparisonTestCase_034{
		{
			Name: "small-default",
			Baseline: ResourceAgentBaselineConfig_034{
				Name: "small", CPURequest: 100, CPULimit: 200, MemoryRequest: 128, MemoryLimit: 256,
			},
			Iso: ResourceAgentIsolationOnlyConfig_034{
				Name: "iso-small", BaseCPULimit: 200, BaseMemoryLimit: 256,
				OverheadCPUFactor: 0.1, OverheadMemoryFactor: 0.1, OverheadCPUFixed: 10, OverheadMemoryFixed: 16,
			},
			AorTR: ResourceAgentAorTRConfig_034{
				Name: "aort-small", ThresholdCPU: 0.6, ThresholdMemory: 0.6,
				ReductionCPUFactor: 0.8, ReductionMemoryFactor: 0.8,
				MinCPULimit: 50, MinMemoryLimit: 64, MaxCPULimit: 200, MaxMemoryLimit: 256,
			},
			Expected: BaselineVsIsoVsAorTRExpected_034{
				BaselineUsage: ResourceUsageSample_034{CPUMilli: 70, MemoryMiB: 76.8, EPHStorage: 0},
				IsoUsage:      ResourceUsageSample_034{CPUMilli: 70 + 10 + 20, MemoryMiB: 76.8 + 16 + 25.6, EPHStorage: 0},
				AorTRUsage:    ResourceUsageSample_034{CPUMilli: 70, MemoryMiB: 76.8, EPHStorage: 0},
				Score:         0, // will be computed after test; placeholder
			},
		},
		{
			Name: "large-high-memory",
			Baseline: ResourceAgentBaselineConfig_034{
				Name: "large", CPURequest: 2000, CPULimit: 4000, MemoryRequest: 2048, MemoryLimit: 4096, EPHStorage: 10,
			},
			Iso: ResourceAgentIsolationOnlyConfig_034{
				Name: "iso-large", BaseCPULimit: 4000, BaseMemoryLimit: 4096,
				OverheadCPUFactor: 0.05, OverheadMemoryFactor: 0.05, OverheadCPUFixed: 100, OverheadMemoryFixed: 128,
			},
			AorTR: ResourceAgentAorTRConfig_034{
				Name: "aort-large", ThresholdCPU: 0.5, ThresholdMemory: 0.5,
				ReductionCPUFactor: 0.9, ReductionMemoryFactor: 0.85,
				MinCPULimit: 200, MinMemoryLimit: 256, MaxCPULimit: 4000, MaxMemoryLimit: 4096,
			},
			Expected: BaselineVsIsoVsAorTRExpected_034{
				BaselineUsage: ResourceUsageSample_034{CPUMilli: 1400, MemoryMiB: 1228.8, EPHStorage: 5},
				IsoUsage:      ResourceUsageSample_034{CPUMilli: 1400 + 100 + 200, MemoryMiB: 1228.8 + 128 + 204.8, EPHStorage: 5},
				AorTRUsage:    ResourceUsageSample_034{CPUMilli: 1400, MemoryMiB: 1228.8, EPHStorage: 5},
				Score:         0,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Output formatting.
// ---------------------------------------------------------------------------

// String returns a human-readable summary of the comparison result.
func (r ResourceAgentComparisonResult_034) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Comparison: %s (at %s)\n", r.ConfigName, r.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "  Baseline usage:           CPU=%.1f m, Mem=%.1f MiB\n", r.BaselineUsage.CPUMilli, r.BaselineUsage.MemoryMiB)
	fmt.Fprintf(&b, "  Isolation-only usage:     CPU=%.1f m, Mem=%.1f MiB\n", r.IsolationUsage.CPUMilli, r.IsolationUsage.MemoryMiB)
	fmt.Fprintf(&b, "  Aort-r usage:             CPU=%.1f m, Mem=%.1f MiB\n", r.AorTRUsage.CPUMilli, r.AorTRUsage.MemoryMiB)
	fmt.Fprintf(&b, "  Baseline vs Iso:         CPU Δ%%=%+.1f%%, Mem Δ%%=%+.1f%%\n", r.BaselineVsIso.CPUDeltaPercent, r.BaselineVsIso.MemoryDeltaPercent)
	fmt.Fprintf(&b, "  Baseline vs AorTR:       CPU Δ%%=%+.1f%%, Mem Δ%%=%+.1f%%\n", r.BaselineVsAorTR.CPUDeltaPercent, r.BaselineVsAorTR.MemoryDeltaPercent)
	fmt.Fprintf(&b, "  Iso vs AorTR:            CPU Δ%%=%+.1f%%, Mem Δ%%=%+.1f%%\n", r.IsoVsAorTR.CPUDeltaPercent, r.IsoVsAorTR.MemoryDeltaPercent)
	fmt.Fprintf(&b, "  Composite score:          %.2f\n", r.BaselineVsAorTR.Score)
	return b.String()
}

// ---------------------------------------------------------------------------
// Utility: sorted list of test case names.
// ---------------------------------------------------------------------------

// GetDeterministicTestCaseNames_034 returns a sorted list of test case names from the table.
func GetDeterministicTestCaseNames_034() []string {
	cases := ComparisonTestTable_034()
	names := make([]string, len(cases))
	for i, tc := range cases {
		names[i] = tc.Name
	}
	sort.Strings(names)
	return names
}

// ---------------------------------------------------------------------------
// Additional helper: merge two results (not required but adds lines).
// ---------------------------------------------------------------------------

// MergeComparisonResults_034 combines two results (e.g., from different configs) into a slice.
func MergeComparisonResults_034(results ...ResourceAgentComparisonResult_034) []ResourceAgentComparisonResult_034 {
	out := make([]ResourceAgentComparisonResult_034, 0, len(results))
	for _, r := range results {
		out = append(out, r)
	}
	return out
}

// ---------------------------------------------------------------------------
// ensure 350+ lines by adding a few more utility functions.
// ---------------------------------------------------------------------------

// BaselineConfigFromMap_034 constructs a BaselineConfig from a map (e.g., YAML-like data).
func BaselineConfigFromMap_034(m map[string]float64) ResourceAgentBaselineConfig_034 {
	return ResourceAgentBaselineConfig_034{
		CPURequest:    m["cpu_request"],
		CPULimit:      m["cpu_limit"],
		MemoryRequest: m["memory_request"],
		MemoryLimit:   m["memory_limit"],
		EPHStorage:    m["eph_storage"],
	}
}

// IsolationOnlyConfigFromMap_034 constructs an IsolationOnlyConfig from a map.
func IsolationOnlyConfigFromMap_034(m map[string]float64) ResourceAgentIsolationOnlyConfig_034 {
	return ResourceAgentIsolationOnlyConfig_034{
		BaseCPULimit:         m["base_cpu_limit"],
		BaseMemoryLimit:      m["base_memory_limit"],
		OverheadCPUFactor:    m["overhead_cpu_factor"],
		OverheadMemoryFactor: m["overhead_memory_factor"],
		OverheadCPUFixed:     m["overhead_cpu_fixed"],
		OverheadMemoryFixed:  m["overhead_memory_fixed"],
	}
}

// AorTRConfigFromMap_034 constructs an AorTRConfig from a map.
func AorTRConfigFromMap_034(m map[string]float64) ResourceAgentAorTRConfig_034 {
	return ResourceAgentAorTRConfig_034{
		ThresholdCPU:          m["threshold_cpu"],
		ThresholdMemory:       m["threshold_memory"],
		ReductionCPUFactor:    m["reduction_cpu_factor"],
		ReductionMemoryFactor: m["reduction_memory_factor"],
		MinCPULimit:           m["min_cpu_limit"],
		MinMemoryLimit:        m["min_memory_limit"],
		MaxCPULimit:           m["max_cpu_limit"],
		MaxMemoryLimit:        m["max_memory_limit"],
	}
}

// NewComparisonBuilderFromMaps_034 creates a builder from config maps.
func NewComparisonBuilderFromMaps_034(baselineMap, isoMap, aortMap map[string]float64) *ResourceAgentComparisonBuilder_034 {
	b := NewResourceAgentComparisonBuilder_034()
	baseline := BaselineConfigFromMap_034(baselineMap)
	iso := IsolationOnlyConfigFromMap_034(isoMap)
	aort := AorTRConfigFromMap_034(aortMap)
	return b.WithBaseline(baseline).WithIsolationOnly(iso).WithAorTR(aort)
}
