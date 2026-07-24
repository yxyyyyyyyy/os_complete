package chunk038

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ProcessResult035 represents the outcome of a resource process step,
// containing the measured limits that will be bridged into sampler pressure.
type ProcessResult035 struct {
	CPULimit      float64        // vCPUs (fractional)
	MemoryLimit   float64        // MiB
	IOLimit       float64        // MB/s
	GPULimit      float64        // GPU compute units
	NetLimit      float64        // Gbps
	Duration      time.Duration  // time window the result covers
	Timestamp     time.Time      // when the result was recorded
	Overhead      float64        // overhead factor [0,1] from instrumentation
}

// SamplerPressure035 indicates the pressure exerted on the resource sampler
// after bridging from a ProcessResult035.
type SamplerPressure035 struct {
	CPU          float64 // pressure value [0,1] derived from CPULimit
	Memory       float64 // pressure value [0,1] derived from MemoryLimit
	IO           float64 // pressure value [0,1] derived from IOLimit
	GPU          float64 // pressure value [0,1] derived from GPULimit
	Net          float64 // pressure value [0,1] derived from NetLimit
	Aggregate    float64 // composite pressure across dimensions
	SampleWeight float64 // weight for the sampler update [0,1]
	LatencyHint  float64 // predicted sampler latency in seconds
}

// SamplerBridgeConfig035 holds tuning parameters for the bridging calculation.
type SamplerBridgeConfig035 struct {
	CPUBreakpoints    []float64 // CPU limit breakpoints (increasing)
	CPUPressures      []float64 // corresponding pressure values [0,1]
	MemoryBreakpoints []float64
	MemoryPressures   []float64
	IOBreakpoints     []float64
	IOPressures       []float64
	GPUBreakpoints    []float64
	GPUPressures      []float64
	NetBreakpoints    []float64
	NetPressures      []float64
	AggregateWeights  [5]float64 // weights for CPU, Mem, IO, GPU, Net sum = 1
	OverheadPenalty   float64    // additional penalty subtracted from pressure [0,1]
	LatencyAlpha      float64    // smoothing factor for latency hint
	LatencyBase       float64    // base latency in seconds
}

// defaultBridgeConfig035 returns a deterministic default configuration.
func defaultBridgeConfig035() SamplerBridgeConfig035 {
	return SamplerBridgeConfig035{
		CPUBreakpoints:    []float64{0.0, 0.5, 1.0, 2.0, 4.0, 8.0, 16.0},
		CPUPressures:      []float64{0.00, 0.10, 0.25, 0.45, 0.65, 0.85, 1.00},
		MemoryBreakpoints: []float64{0, 256, 512, 1024, 2048, 4096, 8192},
		MemoryPressures:   []float64{0.00, 0.05, 0.15, 0.30, 0.50, 0.75, 1.00},
		IOBreakpoints:     []float64{0, 50, 100, 200, 500, 1000, 2000},
		IOPressures:       []float64{0.00, 0.08, 0.18, 0.35, 0.55, 0.80, 1.00},
		GPUBreakpoints:    []float64{0, 1, 2, 4, 8, 16, 32},
		GPUPressures:      []float64{0.00, 0.12, 0.28, 0.48, 0.68, 0.88, 1.00},
		NetBreakpoints:    []float64{0, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
		NetPressures:      []float64{0.00, 0.07, 0.16, 0.30, 0.50, 0.72, 1.00},
		AggregateWeights:  [5]float64{0.30, 0.25, 0.15, 0.20, 0.10},
		OverheadPenalty:   0.05,
		LatencyAlpha:      0.7,
		LatencyBase:       0.010, // 10 ms
	}
}

// bridgeTableEntry is a deterministic entry used for table‑driven pressure lookup.
type bridgeTableEntry struct {
	Dimension   string
	Limit       float64
	Breakpoints []float64
	Pressures   []float64
}

// standardBridgeTable035 builds a sorted table of breakpoint entries for validation
// and reference. The table is sorted by dimension then by limit.
func standardBridgeTable035() []bridgeTableEntry {
	cfg := defaultBridgeConfig035()
	entries := []bridgeTableEntry{
		{"cpu", 0, cfg.CPUBreakpoints, cfg.CPUPressures},
		{"memory", 0, cfg.MemoryBreakpoints, cfg.MemoryPressures},
		{"io", 0, cfg.IOBreakpoints, cfg.IOPressures},
		{"gpu", 0, cfg.GPUBreakpoints, cfg.GPUPressures},
		{"net", 0, cfg.NetBreakpoints, cfg.NetPressures},
	}
	// Fill in a few example limits for table‑driven use.
	for i := range entries {
		e := &entries[i]
		switch e.Dimension {
		case "cpu":
			e.Limit = 2.0
		case "memory":
			e.Limit = 1024
		case "io":
			e.Limit = 200
		case "gpu":
			e.Limit = 4
		case "net":
			e.Limit = 1.0
		}
	}
	// Keep only one entry per dimension (the example above) – we sort anyway.
	// For completeness, let's add a few more deterministic rows.
	extra := []bridgeTableEntry{
		{"cpu", 4.0, cfg.CPUBreakpoints, cfg.CPUPressures},
		{"memory", 2048, cfg.MemoryBreakpoints, cfg.MemoryPressures},
		{"io", 500, cfg.IOBreakpoints, cfg.IOPressures},
		{"gpu", 8, cfg.GPUBreakpoints, cfg.GPUPressures},
		{"net", 2.5, cfg.NetBreakpoints, cfg.NetPressures},
	}
	entries = append(entries, extra...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dimension != entries[j].Dimension {
			return entries[i].Dimension < entries[j].Dimension
		}
		return entries[i].Limit < entries[j].Limit
	})
	return entries
}

// ValidateBridgeConfig035 validates a SamplerBridgeConfig035 and returns an error
// if any breakpoint arrays are inconsistent or the aggregate weights do not sum to 1.
func ValidateBridgeConfig035(cfg *SamplerBridgeConfig035) error {
	if cfg == nil {
		return errors.New("bridge config is nil")
	}
	pairs := []struct {
		name        string
		breakpoints []float64
		pressures   []float64
	}{
		{"cpu", cfg.CPUBreakpoints, cfg.CPUPressures},
		{"memory", cfg.MemoryBreakpoints, cfg.MemoryPressures},
		{"io", cfg.IOBreakpoints, cfg.IOPressures},
		{"gpu", cfg.GPUBreakpoints, cfg.GPUPressures},
		{"net", cfg.NetBreakpoints, cfg.NetPressures},
	}
	for _, p := range pairs {
		if len(p.breakpoints) != len(p.pressures) {
			return fmt.Errorf("breakpoint/pressure length mismatch for %s: %d vs %d",
				p.name, len(p.breakpoints), len(p.pressures))
		}
		if len(p.breakpoints) < 2 {
			return fmt.Errorf("at least 2 breakpoints required for %s", p.name)
		}
		for i := 1; i < len(p.breakpoints); i++ {
			if p.breakpoints[i] <= p.breakpoints[i-1] {
				return fmt.Errorf("breakpoints not increasing for %s at index %d", p.name, i)
			}
		}
		for i := range p.pressures {
			if p.pressures[i] < 0 || p.pressures[i] > 1 {
				return fmt.Errorf("pressure out of [0,1] for %s at index %d", p.name, i)
			}
		}
	}
	// Validate aggregate weights sum to 1 (within epsilon)
	var sum float64
	for _, w := range cfg.AggregateWeights {
		sum += w
	}
	if math.Abs(sum-1.0) > 1e-9 {
		return fmt.Errorf("aggregate weights sum to %f, expected 1.0", sum)
	}
	if cfg.OverheadPenalty < 0 || cfg.OverheadPenalty > 1 {
		return errors.New("overhead penalty out of [0,1]")
	}
	if cfg.LatencyAlpha <= 0 || cfg.LatencyAlpha > 1 {
		return errors.New("latency alpha out of (0,1]")
	}
	if cfg.LatencyBase <= 0 {
		return errors.New("latency base must be positive")
	}
	return nil
}

// interpolatePressure035 returns the pressure for a given limit using a piecewise linear
// interpolation on the breakpoint arrays.
func interpolatePressure035(limit float64, breakpoints, pressures []float64) float64 {
	if limit <= breakpoints[0] {
		return pressures[0]
	}
	if limit >= breakpoints[len(breakpoints)-1] {
		return pressures[len(pressures)-1]
	}
	// binary search for the segment
	idx := sort.Search(len(breakpoints), func(i int) bool {
		return breakpoints[i] > limit
	})
	if idx == 0 {
		return pressures[0]
	}
	if idx >= len(breakpoints) {
		return pressures[len(pressures)-1]
	}
	x0, x1 := breakpoints[idx-1], breakpoints[idx]
	y0, y1 := pressures[idx-1], pressures[idx]
	frac := (limit - x0) / (x1 - x0)
	return y0 + frac*(y1-y0)
}

// BridgeProcessResultToPressure035 converts a ProcessResult035 into a SamplerPressure035
// using the given configuration. If cfg is nil, the default configuration is used.
func BridgeProcessResultToPressure035(result *ProcessResult035, cfg *SamplerBridgeConfig035) (*SamplerPressure035, error) {
	if result == nil {
		return nil, errors.New("process result is nil")
	}
	if cfg == nil {
		defaultCfg := defaultBridgeConfig035()
		cfg = &defaultCfg
	}
	if err := ValidateBridgeConfig035(cfg); err != nil {
		return nil, fmt.Errorf("invalid bridge config: %w", err)
	}

	pressure := &SamplerPressure035{}

	// Map each dimension limit to its pressure via interpolation.
	pressure.CPU = interpolatePressure035(result.CPULimit, cfg.CPUBreakpoints, cfg.CPUPressures)
	pressure.Memory = interpolatePressure035(result.MemoryLimit, cfg.MemoryBreakpoints, cfg.MemoryPressures)
	pressure.IO = interpolatePressure035(result.IOLimit, cfg.IOBreakpoints, cfg.IOPressures)
	pressure.GPU = interpolatePressure035(result.GPULimit, cfg.GPUBreakpoints, cfg.GPUPressures)
	pressure.Net = interpolatePressure035(result.NetLimit, cfg.NetBreakpoints, cfg.NetPressures)

	// Apply overhead penalty (subtract, clamp to 0)
	pressure.CPU = math.Max(0, pressure.CPU-cfg.OverheadPenalty)
	pressure.Memory = math.Max(0, pressure.Memory-cfg.OverheadPenalty)
	pressure.IO = math.Max(0, pressure.IO-cfg.OverheadPenalty)
	pressure.GPU = math.Max(0, pressure.GPU-cfg.OverheadPenalty)
	pressure.Net = math.Max(0, pressure.Net-cfg.OverheadPenalty)

	// Compute aggregate pressure as weighted sum.
	pressure.Aggregate = cfg.AggregateWeights[0]*pressure.CPU +
		cfg.AggregateWeights[1]*pressure.Memory +
		cfg.AggregateWeights[2]*pressure.IO +
		cfg.AggregateWeights[3]*pressure.GPU +
		cfg.AggregateWeights[4]*pressure.Net

	// Determine sample weight inversely related to aggregate pressure.
	// High pressure -> lower weight to avoid overloading sampler.
	pressure.SampleWeight = 1.0 - pressure.Aggregate
	if pressure.SampleWeight < 0 {
		pressure.SampleWeight = 0
	}

	// Latency hint: exponential smoothing based on aggregate pressure.
	// Use a simple model: latency = base + (maxExtra * pressure).
	const maxExtra = 0.050 // 50 ms extra at full pressure
	rawLatency := cfg.LatencyBase + maxExtra*pressure.Aggregate
	// Apply smoothing with a historical value; since we don't have history here,
	// we treat it as the current estimate.
	pressure.LatencyHint = (1-cfg.LatencyAlpha)*rawLatency + cfg.LatencyAlpha*rawLatency
	// This effectively just returns rawLatency, but demonstrates the formula.

	return pressure, nil
}

// NewResourceAgentSamplerBridge035 creates a new bridge with the default configuration.
// It returns the bridge (currently as a config wrapper) and any validation error.
func NewResourceAgentSamplerBridge035() (*SamplerBridgeConfig035, error) {
	cfg := defaultBridgeConfig035()
	if err := ValidateBridgeConfig035(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MustNewResourceAgentSamplerBridge035 is like NewResourceAgentSamplerBridge035 but panics on error.
func MustNewResourceAgentSamplerBridge035() *SamplerBridgeConfig035 {
	cfg, err := NewResourceAgentSamplerBridge035()
	if err != nil {
		panic(fmt.Sprintf("NewResourceAgentSamplerBridge035: %v", err))
	}
	return cfg
}

// ValidateProcessResult035 validates that a ProcessResult035 contains sensible values.
func ValidateProcessResult035(result *ProcessResult035) error {
	if result == nil {
		return errors.New("process result is nil")
	}
	if result.CPULimit < 0 {
		return errors.New("cpu limit must be non-negative")
	}
	if result.MemoryLimit < 0 {
		return errors.New("memory limit must be non-negative")
	}
	if result.IOLimit < 0 {
		return errors.New("io limit must be non-negative")
	}
	if result.GPULimit < 0 {
		return errors.New("gpu limit must be non-negative")
	}
	if result.NetLimit < 0 {
		return errors.New("net limit must be non-negative")
	}
	if result.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	if result.Timestamp.IsZero() {
		return errors.New("timestamp must be set")
	}
	if result.Overhead < 0 || result.Overhead > 1 {
		return errors.New("overhead must be in [0,1]")
	}
	return nil
}

// bridgeDimensionTable035 returns a slice of test cases for deterministic table‑driven testing.
// Each entry provides an input limit and expected pressure dimension output.
func bridgeDimensionTable035() []struct {
	Dimension string
	Limit     float64
	Expected  float64
} {
	// Use the default config to compute expected values.
	cfg := defaultBridgeConfig035()
	table := []struct {
		Dimension string
		Limit     float64
		Expected  float64
	}{
		{"cpu", 0.0, 0.00},
		{"cpu", 1.0, 0.25},
		{"cpu", 2.0, 0.45},
		{"cpu", 16.0, 1.00},
		{"memory", 0, 0.00},
		{"memory", 256, 0.05},
		{"memory", 512, 0.15},
		{"memory", 8192, 1.00},
		{"io", 0, 0.00},
		{"io", 100, 0.18},
		{"io", 2000, 1.00},
		{"gpu", 0, 0.00},
		{"gpu", 4, 0.48},
		{"gpu", 32, 1.00},
		{"net", 0, 0.00},
		{"net", 0.5, 0.16},
		{"net", 10.0, 1.00},
	}
	for i := range table {
		var bp, pr []float64
		switch table[i].Dimension {
		case "cpu":
			bp, pr = cfg.CPUBreakpoints, cfg.CPUPressures
		case "memory":
			bp, pr = cfg.MemoryBreakpoints, cfg.MemoryPressures
		case "io":
			bp, pr = cfg.IOBreakpoints, cfg.IOPressures
		case "gpu":
			bp, pr = cfg.GPUBreakpoints, cfg.GPUPressures
		case "net":
			bp, pr = cfg.NetBreakpoints, cfg.NetPressures
		}
		table[i].Expected = interpolatePressure035(table[i].Limit, bp, pr)
	}
	return table
}

// DumpBridgeTable035 returns a human-readable string of the standard bridge table.
func DumpBridgeTable035() string {
	entries := standardBridgeTable035()
	var out string
	for _, e := range entries {
		out += fmt.Sprintf("dim=%s limit=%.2f breakpoints=%v pressures=%v\n",
			e.Dimension, e.Limit, e.Breakpoints, e.Pressures)
	}
	return out
}

// PressureSummary035 returns a concise summary string from a SamplerPressure035.
func PressureSummary035(p *SamplerPressure035) string {
	if p == nil {
		return "pressure is nil"
	}
	return fmt.Sprintf("CPU=%.3f Mem=%.3f IO=%.3f GPU=%.3f Net=%.3f Agg=%.3f Weight=%.3f Latency=%.6fs",
		p.CPU, p.Memory, p.IO, p.GPU, p.Net, p.Aggregate, p.SampleWeight, p.LatencyHint)
}

// CloneBridgeConfig035 returns a deep copy of the configuration.
func CloneBridgeConfig035(cfg *SamplerBridgeConfig035) *SamplerBridgeConfig035 {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.CPUBreakpoints = append([]float64(nil), cfg.CPUBreakpoints...)
	clone.CPUPressures = append([]float64(nil), cfg.CPUPressures...)
	clone.MemoryBreakpoints = append([]float64(nil), cfg.MemoryBreakpoints...)
	clone.MemoryPressures = append([]float64(nil), cfg.MemoryPressures...)
	clone.IOBreakpoints = append([]float64(nil), cfg.IOBreakpoints...)
	clone.IOPressures = append([]float64(nil), cfg.IOPressures...)
	clone.GPUBreakpoints = append([]float64(nil), cfg.GPUBreakpoints...)
	clone.GPUPressures = append([]float64(nil), cfg.GPUPressures...)
	clone.NetBreakpoints = append([]float64(nil), cfg.NetBreakpoints...)
	clone.NetPressures = append([]float64(nil), cfg.NetPressures...)
	return &clone
}

// UpdateBridgeLatencyHint035 updates the LatencyHint in a SamplerPressure035 using a previous hint.
// This is a helper for iterative smoothing.
func UpdateBridgeLatencyHint035(alpha float64, previous, current float64) float64 {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.7
	}
	return alpha*current + (1-alpha)*previous
}

// ZeroPressure035 returns a SamplerPressure035 with all fields set to zero.
func ZeroPressure035() SamplerPressure035 {
	return SamplerPressure035{}
}

// MaxPressure035 returns a SamplerPressure035 representing maximum pressure (1.0 in all dimensions).
func MaxPressure035() SamplerPressure035 {
	return SamplerPressure035{
		CPU:          1.0,
		Memory:       1.0,
		IO:           1.0,
		GPU:          1.0,
		Net:          1.0,
		Aggregate:    1.0,
		SampleWeight: 0.0,
		LatencyHint:  0.060, // base + maxExtra
	}
}
