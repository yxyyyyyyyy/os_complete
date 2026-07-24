package chunk017

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ResourceAgentMemoryPressureZone_027 represents the severity level of memory pressure.
type ResourceAgentMemoryPressureZone_027 int

const (
	MemoryPressureZoneNormal_027  ResourceAgentMemoryPressureZone_027 = iota // No pressure
	MemoryPressureZoneWarning_027                                            // Slight pressure
	MemoryPressureZoneCritical_027                                           // High pressure
	MemoryPressureZoneOOM_027                                                // Out of memory risk
)

// String returns a human-readable name for the pressure zone.
func (z ResourceAgentMemoryPressureZone_027) String() string {
	switch z {
	case MemoryPressureZoneNormal_027:
		return "normal"
	case MemoryPressureZoneWarning_027:
		return "warning"
	case MemoryPressureZoneCritical_027:
		return "critical"
	case MemoryPressureZoneOOM_027:
		return "oom"
	default:
		return "unknown"
	}
}

// ResourceAgentMemoryPressureMetrics_027 holds raw memory metrics used for pressure detection.
type ResourceAgentMemoryPressureMetrics_027 struct {
	TotalBytes       uint64
	UsedBytes        uint64
	CachedBytes      uint64
	BuffersBytes     uint64
	SwapTotalBytes   uint64
	SwapUsedBytes    uint64
	AvailableBytes   uint64
	PageFaultsPerSec float64
	OOMScore         int
}

// ResourceAgentFaultMemoryScenario_027 defines a single fault injection scenario for memory pressure.
type ResourceAgentFaultMemoryScenario_027 struct {
	Name              string
	TargetUsagePercent float64            // 0.0 to 1.0
	Duration          time.Duration
	AllocationSize    uint64              // bytes per allocation
	AllocationCount   int                 // number of allocations
	InjectSwap        bool
	InjectPageFaults  bool
	RaiseOOMScore     bool
}

// ResourceAgentMemoryPressureHeuristic_027 defines thresholds for pressure zone classification.
type ResourceAgentMemoryPressureHeuristic_027 struct {
	Zone                ResourceAgentMemoryPressureZone_027
	MinAvailablePercent float64
	MaxSwapUsagePercent float64
	PageFaultThreshold  float64
	OOMScoreThreshold   int
}

// ResourceAgentMemoryPressureState_027 tracks the current pressure state and decision history.
type ResourceAgentMemoryPressureState_027 struct {
	mu          sync.Mutex
	currentZone ResourceAgentMemoryPressureZone_027
	lastMetrics ResourceAgentMemoryPressureMetrics_027
	lastUpdate  time.Time
	scenarios   []ResourceAgentFaultMemoryScenario_027
	history     []pressureRecord_027
}

type pressureRecord_027 struct {
	timestamp time.Time
	zone      ResourceAgentMemoryPressureZone_027
	metrics   ResourceAgentMemoryPressureMetrics_027
}

// NewResourceAgentMemoryPressureState_027 creates a new pressure state tracker.
func NewResourceAgentMemoryPressureState_027() *ResourceAgentMemoryPressureState_027 {
	return &ResourceAgentMemoryPressureState_027{
		currentZone: MemoryPressureZoneNormal_027,
		lastUpdate:  time.Now(),
	}
}

// ValidateResourceAgentFaultMemoryScenario_027 checks if a scenario is valid.
func ValidateResourceAgentFaultMemoryScenario_027(s *ResourceAgentFaultMemoryScenario_027) error {
	if s == nil {
		return errors.New("scenario cannot be nil")
	}
	if s.Name == "" {
		return errors.New("scenario name must not be empty")
	}
	if s.TargetUsagePercent < 0.0 || s.TargetUsagePercent > 1.0 {
		return fmt.Errorf("target usage percent %f out of range [0,1]", s.TargetUsagePercent)
	}
	if s.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	if s.AllocationSize == 0 {
		return errors.New("allocation size must be positive")
	}
	if s.AllocationCount <= 0 {
		return errors.New("allocation count must be positive")
	}
	return nil
}

// ResourceAgentMemoryPressureZoneFromPercent_027 converts a usage fraction to a pressure zone.
func ResourceAgentMemoryPressureZoneFromPercent_027(usedPercent, swapUsedPercent, pageFaults float64, oomScore int) ResourceAgentMemoryPressureZone_027 {
	heuristics := ResourceAgentDefaultHeuristicsTable_027()
	// Heuristics sorted by severity (ascending). Evaluate from low to high.
	for _, h := range heuristics {
		if usedPercent >= h.MinAvailablePercent ||
			swapUsedPercent >= h.MaxSwapUsagePercent ||
			pageFaults >= h.PageFaultThreshold ||
			oomScore >= h.OOMScoreThreshold {
			return h.Zone
		}
	}
	return MemoryPressureZoneNormal_027
}

// ResourceAgentDefaultHeuristicsTable_027 returns a deterministic table of pressure heuristics.
func ResourceAgentDefaultHeuristicsTable_027() []ResourceAgentMemoryPressureHeuristic_027 {
	// Table entries ordered by increasing severity.
	return []ResourceAgentMemoryPressureHeuristic_027{
		{Zone: MemoryPressureZoneNormal_027, MinAvailablePercent: 0.0, MaxSwapUsagePercent: 0.2, PageFaultThreshold: 100, OOMScoreThreshold: 0},
		{Zone: MemoryPressureZoneWarning_027, MinAvailablePercent: 0.2, MaxSwapUsagePercent: 0.5, PageFaultThreshold: 500, OOMScoreThreshold: 200},
		{Zone: MemoryPressureZoneCritical_027, MinAvailablePercent: 0.5, MaxSwapUsagePercent: 0.8, PageFaultThreshold: 1000, OOMScoreThreshold: 500},
		{Zone: MemoryPressureZoneOOM_027, MinAvailablePercent: 0.8, MaxSwapUsagePercent: 1.0, PageFaultThreshold: 2000, OOMScoreThreshold: 800},
	}
}

// ResourceAgentMemoryUsagePercent_027 computes the percentage of memory used.
func ResourceAgentMemoryUsagePercent_027(metrics ResourceAgentMemoryPressureMetrics_027) float64 {
	if metrics.TotalBytes == 0 {
		return 0.0
	}
	used := float64(metrics.UsedBytes - metrics.CachedBytes - metrics.BuffersBytes)
	if used < 0 {
		used = 0
	}
	return used / float64(metrics.TotalBytes)
}

// ResourceAgentSwapUsagePercent_027 computes swap usage fraction.
func ResourceAgentSwapUsagePercent_027(metrics ResourceAgentMemoryPressureMetrics_027) float64 {
	if metrics.SwapTotalBytes == 0 {
		return 0.0
	}
	return float64(metrics.SwapUsedBytes) / float64(metrics.SwapTotalBytes)
}

// ResourceAgentMemoryPressureDetector_027 performs ongoing pressure detection using heuristics.
type ResourceAgentMemoryPressureDetector_027 struct {
	state     *ResourceAgentMemoryPressureState_027
	heuristics []ResourceAgentMemoryPressureHeuristic_027
}

// NewResourceAgentMemoryPressureDetector_027 creates a detector with default heuristics.
func NewResourceAgentMemoryPressureDetector_027(state *ResourceAgentMemoryPressureState_027) *ResourceAgentMemoryPressureDetector_027 {
	return &ResourceAgentMemoryPressureDetector_027{
		state:      state,
		heuristics: ResourceAgentDefaultHeuristicsTable_027(),
	}
}

// UpdateMetrics evaluates new metrics and updates the state.
func (d *ResourceAgentMemoryPressureDetector_027) UpdateMetrics_027(metrics ResourceAgentMemoryPressureMetrics_027) {
	usage := ResourceAgentMemoryUsagePercent_027(metrics)
	swapUsage := ResourceAgentSwapUsagePercent_027(metrics)
	zone := ResourceAgentMemoryPressureZoneFromPercent_027(usage, swapUsage, metrics.PageFaultsPerSec, metrics.OOMScore)

	d.state.mu.Lock()
	defer d.state.mu.Unlock()

	d.state.currentZone = zone
	d.state.lastMetrics = metrics
	d.state.lastUpdate = time.Now()
	rec := pressureRecord_027{
		timestamp: d.state.lastUpdate,
		zone:      zone,
		metrics:   metrics,
	}
	d.state.history = append(d.state.history, rec)
	// Keep last 100 records
	if len(d.state.history) > 100 {
		d.state.history = d.state.history[len(d.state.history)-100:]
	}
}

// CurrentZone returns the current pressure zone.
func (d *ResourceAgentMemoryPressureDetector_027) CurrentZone_027() ResourceAgentMemoryPressureZone_027 {
	d.state.mu.Lock()
	defer d.state.mu.Unlock()
	return d.state.currentZone
}

// IsPressureActive returns true if zone is Warning or above.
func (d *ResourceAgentMemoryPressureDetector_027) IsPressureActive_027() bool {
	return d.CurrentZone_027() >= MemoryPressureZoneWarning_027
}

// AddScenario registers a fault memory scenario for later execution.
func (d *ResourceAgentMemoryPressureDetector_027) AddScenario_027(s ResourceAgentFaultMemoryScenario_027) error {
	if err := ValidateResourceAgentFaultMemoryScenario_027(&s); err != nil {
		return err
	}
	d.state.mu.Lock()
	defer d.state.mu.Unlock()
	d.state.scenarios = append(d.state.scenarios, s)
	return nil
}

// ExecuteScenario runs a specific scenario by name. Returns error if not found or invalid.
func (d *ResourceAgentMemoryPressureDetector_027) ExecuteScenario_027(name string) error {
	d.state.mu.Lock()
	scenarios := d.state.scenarios
	d.state.mu.Unlock()

	var scenario *ResourceAgentFaultMemoryScenario_027
	for i := range scenarios {
		if scenarios[i].Name == name {
			scenario = &scenarios[i]
			break
		}
	}
	if scenario == nil {
		return fmt.Errorf("scenario %q not found", name)
	}
	// Simulate execution (in real system would allocate memory)
	_ = scenario.AllocationSize * uint64(scenario.AllocationCount)
	_ = scenario.Duration
	// For demonstration, we just record a metrics spike.
	spike := ResourceAgentMemoryPressureMetrics_027{
		TotalBytes:       1 << 30, // 1GB
		UsedBytes:        uint64(float64(1<<30) * scenario.TargetUsagePercent),
		CachedBytes:      0,
		BuffersBytes:     0,
		SwapTotalBytes:   1 << 30,
		SwapUsedBytes:    0,
		AvailableBytes:   1 << 30,
		PageFaultsPerSec: 0,
		OOMScore:         0,
	}
	if scenario.InjectSwap {
		spike.SwapUsedBytes = spike.SwapTotalBytes / 2
	}
	if scenario.InjectPageFaults {
		spike.PageFaultsPerSec = 5000
	}
	if scenario.RaiseOOMScore {
		spike.OOMScore = 900
	}
	d.UpdateMetrics_027(spike)
	return nil
}

// History returns the last N records.
func (d *ResourceAgentMemoryPressureDetector_027) History_027(n int) []pressureRecord_027 {
	d.state.mu.Lock()
	defer d.state.mu.Unlock()
	if n <= 0 || n > len(d.state.history) {
		n = len(d.state.history)
	}
	cp := make([]pressureRecord_027, n)
	copy(cp, d.state.history[len(d.state.history)-n:])
	return cp
}

// ResourceAgentMemoryPressureSummary_027 provides a quick summary of current pressure.
type ResourceAgentMemoryPressureSummary_027 struct {
	Zone       ResourceAgentMemoryPressureZone_027
	UsagePct   float64
	SwapPct    float64
	PageFaults float64
	OOMScore   int
	RecordedAt time.Time
}

// Summary returns a snapshot of current pressure state.
func (d *ResourceAgentMemoryPressureDetector_027) Summary_027() ResourceAgentMemoryPressureSummary_027 {
	d.state.mu.Lock()
	defer d.state.mu.Unlock()
	m := d.state.lastMetrics
	return ResourceAgentMemoryPressureSummary_027{
		Zone:       d.state.currentZone,
		UsagePct:   ResourceAgentMemoryUsagePercent_027(m),
		SwapPct:    ResourceAgentSwapUsagePercent_027(m),
		PageFaults: m.PageFaultsPerSec,
		OOMScore:   m.OOMScore,
		RecordedAt: d.state.lastUpdate,
	}
}

// ResourceAgentMemoryFaultHeuristicsTable_027 returns a table of heuristic entries for testing.
func ResourceAgentMemoryFaultHeuristicsTable_027() []struct {
	Zone      ResourceAgentMemoryPressureZone_027
	Threshold float64
} {
	return []struct {
		Zone      ResourceAgentMemoryPressureZone_027
		Threshold float64
	}{
		{Zone: MemoryPressureZoneNormal_027, Threshold: 0.1},
		{Zone: MemoryPressureZoneWarning_027, Threshold: 0.4},
		{Zone: MemoryPressureZoneCritical_027, Threshold: 0.7},
		{Zone: MemoryPressureZoneOOM_027, Threshold: 0.9},
	}
}

// ResourceAgentComputePressureScore_027 computes a composite pressure score [0, 100].
func ResourceAgentComputePressureScore_027(metrics ResourceAgentMemoryPressureMetrics_027) float64 {
	usage := ResourceAgentMemoryUsagePercent_027(metrics)
	swapUsage := ResourceAgentSwapUsagePercent_027(metrics)
	score := usage*50 + swapUsage*30 + math.Min(metrics.PageFaultsPerSec/100, 20)
	if score > 100 {
		score = 100
	}
	return score
}

// ResourceAgentSortScenariosBySeverity_027 sorts scenarios by their target usage descending.
func ResourceAgentSortScenariosBySeverity_027(scenarios []ResourceAgentFaultMemoryScenario_027) {
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].TargetUsagePercent > scenarios[j].TargetUsagePercent
	})
}

// ResourceAgentMemoryPressureEvent_027 represents a detected pressure event.
type ResourceAgentMemoryPressureEvent_027 struct {
	Zone      ResourceAgentMemoryPressureZone_027
	Timestamp time.Time
	Metrics   ResourceAgentMemoryPressureMetrics_027
}

// ResourceAgentDetectPressureEvents_027 scans history and returns events where zone changed.
func ResourceAgentDetectPressureEvents_027(history []pressureRecord_027) []ResourceAgentMemoryPressureEvent_027 {
	if len(history) == 0 {
		return nil
	}
	var events []ResourceAgentMemoryPressureEvent_027
	prev := history[0].zone
	for _, rec := range history[1:] {
		if rec.zone != prev {
			events = append(events, ResourceAgentMemoryPressureEvent_027{
				Zone:      rec.zone,
				Timestamp: rec.timestamp,
				Metrics:   rec.metrics,
			})
			prev = rec.zone
		}
	}
	return events
}

// ResourceAgentCheckMemoryAvailability_027 returns true if available memory for allocation exists.
func ResourceAgentCheckMemoryAvailability_027(metrics ResourceAgentMemoryPressureMetrics_027, neededBytes uint64) bool {
	avail := metrics.AvailableBytes
	return avail >= neededBytes
}

// ResourceAgentInjectMemoryPressure_027 simulates a pressure injection by modifying metrics.
func ResourceAgentInjectMemoryPressure_027(metrics *ResourceAgentMemoryPressureMetrics_027, increaseFraction float64) {
	if increaseFraction < 0 || increaseFraction > 1 {
		increaseFraction = 0.5
	}
	usedIncrease := uint64(float64(metrics.TotalBytes) * increaseFraction)
	metrics.UsedBytes += usedIncrease
	if metrics.UsedBytes > metrics.TotalBytes {
		metrics.UsedBytes = metrics.TotalBytes
	}
	metrics.AvailableBytes = metrics.TotalBytes - metrics.UsedBytes
}

// ResourceAgentResetMemoryPressureState_027 resets the state to normal.
func ResourceAgentResetMemoryPressureState_027(state *ResourceAgentMemoryPressureState_027) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.currentZone = MemoryPressureZoneNormal_027
	state.lastMetrics = ResourceAgentMemoryPressureMetrics_027{}
	state.lastUpdate = time.Now()
	state.scenarios = nil
	state.history = nil
}

// ResourceAgentNewMemoryPressureMetrics_027 creates a metrics instance with given usage.
func ResourceAgentNewMemoryPressureMetrics_027(total uint64, usedPercent float64) ResourceAgentMemoryPressureMetrics_027 {
	used := uint64(float64(total) * usedPercent)
	return ResourceAgentMemoryPressureMetrics_027{
		TotalBytes:       total,
		UsedBytes:        used,
		CachedBytes:      0,
		BuffersBytes:     0,
		SwapTotalBytes:   0,
		SwapUsedBytes:    0,
		AvailableBytes:   total - used,
		PageFaultsPerSec: 0,
		OOMScore:         0,
	}
}

// ResourceAgentMemoryPressureName_027 returns the name associated with a zone.
func ResourceAgentMemoryPressureName_027(zone ResourceAgentMemoryPressureZone_027) string {
	return zone.String()
}

// ResourceAgentParseMemoryPressureZone_027 parses a string to zone.
func ResourceAgentParseMemoryPressureZone_027(s string) (ResourceAgentMemoryPressureZone_027, error) {
	switch s {
	case "normal":
		return MemoryPressureZoneNormal_027, nil
	case "warning":
		return MemoryPressureZoneWarning_027, nil
	case "critical":
		return MemoryPressureZoneCritical_027, nil
	case "oom":
		return MemoryPressureZoneOOM_027, nil
	default:
		return MemoryPressureZoneNormal_027, fmt.Errorf("unknown memory pressure zone: %s", s)
	}
}

// ResourceAgentMemoryPressureRecordCount_027 returns the number of history records.
func ResourceAgentMemoryPressureRecordCount_027(state *ResourceAgentMemoryPressureState_027) int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return len(state.history)
}

// ResourceAgentMemoryPressureLastUpdate_027 returns the last update time.
func ResourceAgentMemoryPressureLastUpdate_027(state *ResourceAgentMemoryPressureState_027) time.Time {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.lastUpdate
}

// ResourceAgentMemoryPressureScenarioNames_027 returns sorted list of registered scenario names.
func ResourceAgentMemoryPressureScenarioNames_027(state *ResourceAgentMemoryPressureState_027) []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	names := make([]string, len(state.scenarios))
	for i, s := range state.scenarios {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names
}

// ResourceAgentMemoryPressureDefaultScenarios_027 returns a set of default scenarios.
func ResourceAgentMemoryPressureDefaultScenarios_027() []ResourceAgentFaultMemoryScenario_027 {
	return []ResourceAgentFaultMemoryScenario_027{
		{
			Name:               "light_pressure",
			TargetUsagePercent: 0.3,
			Duration:           10 * time.Second,
			AllocationSize:     1024,
			AllocationCount:    100,
		},
		{
			Name:               "heavy_pressure",
			TargetUsagePercent: 0.8,
			Duration:           30 * time.Second,
			AllocationSize:     4096,
			AllocationCount:    500,
		},
		{
			Name:               "oom_simulation",
			TargetUsagePercent: 0.95,
			Duration:           5 * time.Second,
			AllocationSize:     65536,
			AllocationCount:    1000,
			RaiseOOMScore:      true,
		},
	}
}
