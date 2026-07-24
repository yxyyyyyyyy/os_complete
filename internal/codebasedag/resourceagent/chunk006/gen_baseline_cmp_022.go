package chunk006

import (
	"errors"
	"fmt"
	"sort"
)

// ResourceAgentComparisonType022 represents the type of comparison being built.
type ResourceAgentComparisonType022 int

const (
	ComparisonBaseline022 ResourceAgentComparisonType022 = iota
	ComparisonIsolationOnly022
	ComparisonAortR022
)

// ResourceAgentIsolationScope022 defines the scope or level of isolation.
type ResourceAgentIsolationScope022 int

const (
	IsolationNone022 ResourceAgentIsolationScope022 = iota
	IsolationProcess022
	IsolationMemory022
	IsolationNetwork022
	IsolationFull022
)

// ResourceAgentComparisonConfig022 holds parameters for a single comparison scenario.
type ResourceAgentComparisonConfig022 struct {
	Type            ResourceAgentComparisonType022 `json:"type"`
	Name            string                         `json:"name,omitempty"`
	CPUAllocation   int                            `json:"cpu_allocation"`   // in millicores
	MemoryAllocMb   int                            `json:"memory_alloc_mb"`
	IsolationScope  ResourceAgentIsolationScope022 `json:"isolation_scope"`
	ContainerCount  int                            `json:"container_count"`
	EnablePolicies  bool                           `json:"enable_policies"`
	EnableProfiling bool                           `json:"enable_profiling"`
	ExtraLatencyMs  int                            `json:"extra_latency_ms"` // simulated overhead
}

// ResourceAgentComparisonResult022 stores the computed outcome of a comparison.
type ResourceAgentComparisonResult022 struct {
	TypeLabel    string  `json:"type_label"`
	Score        float64 `json:"score"`
	OverheadMs   int     `json:"overhead_ms"`
	MemoryUsage  int     `json:"memory_usage"`  // in MB
	CPUUsage     int     `json:"cpu_usage"`     // in millicores
	IsolationGap int     `json:"isolation_gap"` // 0-100, higher is better
}

// ResourceAgentComparisonScenario022 is an entry in the deterministic test table.
type ResourceAgentComparisonScenario022 struct {
	Config     ResourceAgentComparisonConfig022   `json:"config"`
	Expected   *ResourceAgentComparisonResult022  `json:"expected,omitempty"`
	Validation func(cfg ResourceAgentComparisonConfig022) error `json:"-"`
}

// ResourceAgentComparisonScenariosTable022 is a deterministic table of test scenarios.
var ResourceAgentComparisonScenariosTable022 = []ResourceAgentComparisonScenario022{
	{
		Config: ResourceAgentComparisonConfig022{
			Type:            ComparisonBaseline022,
			Name:            "Baseline minimal",
			CPUAllocation:   100,
			MemoryAllocMb:   64,
			IsolationScope:  IsolationNone022,
			ContainerCount:  1,
			EnablePolicies:  false,
			EnableProfiling: false,
			ExtraLatencyMs:  0,
		},
		Expected: &ResourceAgentComparisonResult022{
			TypeLabel:    "baseline",
			Score:        100.0,
			OverheadMs:   0,
			MemoryUsage:  64,
			CPUUsage:     100,
			IsolationGap: 0,
		},
		Validation: func(cfg ResourceAgentComparisonConfig022) error {
			if cfg.CPUAllocation < 50 {
				return errors.New("cpu allocation too low for baseline")
			}
			return nil
		},
	},
	{
		Config: ResourceAgentComparisonConfig022{
			Type:            ComparisonIsolationOnly022,
			Name:            "Isolation process only",
			CPUAllocation:   200,
			MemoryAllocMb:   128,
			IsolationScope:  IsolationProcess022,
			ContainerCount:  2,
			EnablePolicies:  false,
			EnableProfiling: false,
			ExtraLatencyMs:  5,
		},
		Expected: &ResourceAgentComparisonResult022{
			TypeLabel:    "isolation-only",
			Score:        85.0,
			OverheadMs:   5,
			MemoryUsage:  140,
			CPUUsage:     210,
			IsolationGap: 30,
		},
		Validation: func(cfg ResourceAgentComparisonConfig022) error {
			if cfg.IsolationScope == IsolationNone022 {
				return errors.New("isolation-only requires non-none isolation")
			}
			return nil
		},
	},
	{
		Config: ResourceAgentComparisonConfig022{
			Type:            ComparisonAortR022,
			Name:            "AORT-R full",
			CPUAllocation:   250,
			MemoryAllocMb:   256,
			IsolationScope:  IsolationFull022,
			ContainerCount:  4,
			EnablePolicies:  true,
			EnableProfiling: true,
			ExtraLatencyMs:  2,
		},
		Expected: &ResourceAgentComparisonResult022{
			TypeLabel:    "aort-r",
			Score:        95.0,
			OverheadMs:   2,
			MemoryUsage:  270,
			CPUUsage:     260,
			IsolationGap: 95,
		},
		Validation: func(cfg ResourceAgentComparisonConfig022) error {
			if !cfg.EnablePolicies {
				return errors.New("AORT-R requires policies enabled")
			}
			if !cfg.EnableProfiling {
				return errors.New("AORT-R requires profiling enabled")
			}
			return nil
		},
	},
	{
		Config: ResourceAgentComparisonConfig022{
			Type:            ComparisonIsolationOnly022,
			Name:            "Isolation high overhead",
			CPUAllocation:   1000,
			MemoryAllocMb:   512,
			IsolationScope:  IsolationFull022,
			ContainerCount:  8,
			EnablePolicies:  false,
			EnableProfiling: false,
			ExtraLatencyMs:  100,
		},
		Expected: &ResourceAgentComparisonResult022{
			TypeLabel:    "isolation-only",
			Score:        40.0,
			OverheadMs:   100,
			MemoryUsage:  600,
			CPUUsage:     1100,
			IsolationGap: 70,
		},
		Validation: func(cfg ResourceAgentComparisonConfig022) error {
			if cfg.CPUAllocation > 2000 {
				return fmt.Errorf("cpu allocation %d unrealistic", cfg.CPUAllocation)
			}
			return nil
		},
	},
	{
		Config: ResourceAgentComparisonConfig022{
			Type:            ComparisonBaseline022,
			Name:            "Baseline oversized",
			CPUAllocation:   500,
			MemoryAllocMb:   1024,
			IsolationScope:  IsolationNone022,
			ContainerCount:  1,
			EnablePolicies:  false,
			EnableProfiling: false,
			ExtraLatencyMs:  0,
		},
		Expected: &ResourceAgentComparisonResult022{
			TypeLabel:    "baseline",
			Score:        60.0,
			OverheadMs:   0,
			MemoryUsage:  1024,
			CPUUsage:     500,
			IsolationGap: 0,
		},
		Validation: func(cfg ResourceAgentComparisonConfig022) error {
			if cfg.MemoryAllocMb > 512 {
				return errors.New("baseline memory too high, consider isolation")
			}
			return nil
		},
	},
}

// ResourceAgentNewBaselineComparison_022 builds a baseline comparison config with sensible defaults.
func ResourceAgentNewBaselineComparison_022(cpu int, mem int) *ResourceAgentComparisonConfig022 {
	return &ResourceAgentComparisonConfig022{
		Type:            ComparisonBaseline022,
		Name:            "baseline-auto",
		CPUAllocation:   cpu,
		MemoryAllocMb:   mem,
		IsolationScope:  IsolationNone022,
		ContainerCount:  1,
		EnablePolicies:  false,
		EnableProfiling: false,
		ExtraLatencyMs:  0,
	}
}

// ResourceAgentNewIsolationOnlyComparison_022 builds an isolation-only comparison config with sensible defaults.
func ResourceAgentNewIsolationOnlyComparison_022(cpu int, mem int, scope ResourceAgentIsolationScope022) *ResourceAgentComparisonConfig022 {
	return &ResourceAgentComparisonConfig022{
		Type:            ComparisonIsolationOnly022,
		Name:            "isolation-only-auto",
		CPUAllocation:   cpu,
		MemoryAllocMb:   mem,
		IsolationScope:  scope,
		ContainerCount:  2,
		EnablePolicies:  false,
		EnableProfiling: false,
		ExtraLatencyMs:  10,
	}
}

// ResourceAgentNewAortRComparison_022 builds an AORT-R comparison config with sensible defaults.
func ResourceAgentNewAortRComparison_022(cpu int, mem int, scope ResourceAgentIsolationScope022) *ResourceAgentComparisonConfig022 {
	return &ResourceAgentComparisonConfig022{
		Type:            ComparisonAortR022,
		Name:            "aortr-auto",
		CPUAllocation:   cpu,
		MemoryAllocMb:   mem,
		IsolationScope:  scope,
		ContainerCount:  4,
		EnablePolicies:  true,
		EnableProfiling: true,
		ExtraLatencyMs:  2,
	}
}

// ValidateResourceAgentComparisonConfig_022 validates the entire comparison configuration.
// It returns an error if any field is out of acceptable range or inconsistent with the type.
func ValidateResourceAgentComparisonConfig_022(cfg *ResourceAgentComparisonConfig022) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	if cfg.CPUAllocation <= 0 {
		return fmt.Errorf("invalid CPU allocation: %d", cfg.CPUAllocation)
	}
	if cfg.MemoryAllocMb <= 0 {
		return fmt.Errorf("invalid memory allocation: %d", cfg.MemoryAllocMb)
	}
	if cfg.IsolationScope < IsolationNone022 || cfg.IsolationScope > IsolationFull022 {
		return fmt.Errorf("invalid isolation scope: %d", cfg.IsolationScope)
	}
	if cfg.ContainerCount <= 0 {
		return fmt.Errorf("container count must be positive: %d", cfg.ContainerCount)
	}
	if cfg.ExtraLatencyMs < 0 {
		return fmt.Errorf("extra latency cannot be negative: %d", cfg.ExtraLatencyMs)
	}
	switch cfg.Type {
	case ComparisonBaseline022:
		if cfg.IsolationScope != IsolationNone022 {
			return errors.New("baseline type must have isolation set to none")
		}
		if cfg.EnablePolicies || cfg.EnableProfiling {
			return errors.New("baseline type cannot have policies or profiling enabled")
		}
	case ComparisonIsolationOnly022:
		if cfg.IsolationScope == IsolationNone022 {
			return errors.New("isolation-only type must have a non-none isolation scope")
		}
		if cfg.EnablePolicies || cfg.EnableProfiling {
			return errors.New("isolation-only type cannot have policies or profiling enabled")
		}
	case ComparisonAortR022:
		if !cfg.EnablePolicies {
			return errors.New("AORT-R type must have policies enabled")
		}
		if !cfg.EnableProfiling {
			return errors.New("AORT-R type must have profiling enabled")
		}
	default:
		return fmt.Errorf("unknown comparison type: %d", cfg.Type)
	}
	return nil
}

// ResourceAgentComputeBaselineVsIsolationOnly_022 computes the differences between baseline and isolation-only results.
func ResourceAgentComputeBaselineVsIsolationOnly_022(base, iso *ResourceAgentComparisonResult022) (*ResourceAgentComparisonResult022, error) {
	if base == nil || iso == nil {
		return nil, errors.New("nil result")
	}
	return &ResourceAgentComparisonResult022{
		TypeLabel:    "baseline_vs_isolation",
		Score:        base.Score - iso.Score,
		OverheadMs:   iso.OverheadMs - base.OverheadMs,
		MemoryUsage:  iso.MemoryUsage - base.MemoryUsage,
		CPUUsage:     iso.CPUUsage - base.CPUUsage,
		IsolationGap: iso.IsolationGap - base.IsolationGap,
	}, nil
}

// ResourceAgentComputeBaselineVsAortR_022 computes the differences between baseline and AORT-R results.
func ResourceAgentComputeBaselineVsAortR_022(base, aort *ResourceAgentComparisonResult022) (*ResourceAgentComparisonResult022, error) {
	if base == nil || aort == nil {
		return nil, errors.New("nil result")
	}
	return &ResourceAgentComparisonResult022{
		TypeLabel:    "baseline_vs_aortr",
		Score:        base.Score - aort.Score,
		OverheadMs:   aort.OverheadMs - base.OverheadMs,
		MemoryUsage:  aort.MemoryUsage - base.MemoryUsage,
		CPUUsage:     aort.CPUUsage - base.CPUUsage,
		IsolationGap: aort.IsolationGap - base.IsolationGap,
	}, nil
}

// ResourceAgentComputeIsolationOnlyVsAortR_022 computes the differences between isolation-only and AORT-R results.
func ResourceAgentComputeIsolationOnlyVsAortR_022(iso, aort *ResourceAgentComparisonResult022) (*ResourceAgentComparisonResult022, error) {
	if iso == nil || aort == nil {
		return nil, errors.New("nil result")
	}
	return &ResourceAgentComparisonResult022{
		TypeLabel:    "isolation_vs_aortr",
		Score:        iso.Score - aort.Score,
		OverheadMs:   aort.OverheadMs - iso.OverheadMs,
		MemoryUsage:  aort.MemoryUsage - iso.MemoryUsage,
		CPUUsage:     aort.CPUUsage - iso.CPUUsage,
		IsolationGap: aort.IsolationGap - iso.IsolationGap,
	}, nil
}

// ResourceAgentComparisonResultFromConfig_022 derives a result from a configuration using internal models.
func ResourceAgentComparisonResultFromConfig_022(cfg *ResourceAgentComparisonConfig022) *ResourceAgentComparisonResult022 {
	if cfg == nil {
		return nil
	}
	res := &ResourceAgentComparisonResult022{
		TypeLabel:   typeLabelFromType022(cfg.Type),
		MemoryUsage: cfg.MemoryAllocMb + cfg.ContainerCount*8,
		CPUUsage:    cfg.CPUAllocation + cfg.ContainerCount*5,
	}
	// Base score before overhead
	baseScore := 100.0
	if cfg.CPUAllocation > 400 {
		baseScore -= float64(cfg.CPUAllocation-400) * 0.02
	}
	if cfg.MemoryAllocMb > 256 {
		baseScore -= float64(cfg.MemoryAllocMb-256) * 0.005
	}
	// Overhead computation
	overhead := cfg.ExtraLatencyMs
	overhead += cfg.ContainerCount * 2
	if cfg.EnablePolicies {
		overhead += 5
	}
	if cfg.EnableProfiling {
		overhead += 3
	}
	res.OverheadMs = overhead
	// Isolation gap
	gap := 0
	switch cfg.IsolationScope {
	case IsolationNone022:
		gap = 0
	case IsolationProcess022:
		gap = 25
	case IsolationMemory022:
		gap = 50
	case IsolationNetwork022:
		gap = 75
	case IsolationFull022:
		gap = 100
	}
	// Adjust for policies and profiling (AORT-R)
	if cfg.EnablePolicies {
		gap += 20
		if gap > 100 {
			gap = 100
		}
	}
	if cfg.EnableProfiling {
		gap += 10
		if gap > 100 {
			gap = 100
		}
	}
	res.IsolationGap = gap
	// Final score
	penalty := float64(overhead) * 0.5
	if gap > 0 {
		penalty -= float64(gap) * 0.2 // benefit from isolation
	}
	finalScore := baseScore - penalty
	if finalScore < 0 {
		finalScore = 0
	}
	if finalScore > 100 {
		finalScore = 100
	}
	res.Score = finalScore
	return res
}

// typeLabelFromType022 maps type to human-readable label.
func typeLabelFromType022(t ResourceAgentComparisonType022) string {
	switch t {
	case ComparisonBaseline022:
		return "baseline"
	case ComparisonIsolationOnly022:
		return "isolation-only"
	case ComparisonAortR022:
		return "aort-r"
	default:
		return "unknown"
	}
}

// ResourceAgentSortComparisonsByScore_022 sorts the given slice of results in descending order by Score.
func ResourceAgentSortComparisonsByScore_022(results []*ResourceAgentComparisonResult022) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// ResourceAgentFilterForType_022 returns a filtered slice of scenarios matching the given comparison type.
func ResourceAgentFilterForType_022(ts []ResourceAgentComparisonScenario022, ct ResourceAgentComparisonType022) []ResourceAgentComparisonScenario022 {
	var out []ResourceAgentComparisonScenario022
	for _, s := range ts {
		if s.Config.Type == ct {
			out = append(out, s)
		}
	}
	return out
}

// ResourceAgentValidateAllScenarios_022 runs validators on every scenario in the table.
// It returns a combined error containing all validation failures.
func ResourceAgentValidateAllScenarios_022() error {
	var errs []error
	for i, s := range ResourceAgentComparisonScenariosTable022 {
		if s.Validation != nil {
			if err := s.Validation(s.Config); err != nil {
				errs = append(errs, fmt.Errorf("scenario %d (%s): %w", i, s.Config.Name, err))
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("validation errors: %v", errs)
}
