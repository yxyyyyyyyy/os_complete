package chunk037

import (
	"errors"
	"fmt"
	"math"
)

// PressureLevel_023 represents a level of resource pressure.
type PressureLevel_023 int

const (
	PressureLevelLow_023      PressureLevel_023 = iota // 0
	PressureLevelMedium_023                            // 1
	PressureLevelHigh_023                              // 2
	PressureLevelCritical_023                          // 3
)

// String returns a human-readable name for the pressure level.
func (p PressureLevel_023) String() string {
	switch p {
	case PressureLevelLow_023:
		return "low"
	case PressureLevelMedium_023:
		return "medium"
	case PressureLevelHigh_023:
		return "high"
	case PressureLevelCritical_023:
		return "critical"
	default:
		return fmt.Sprintf("PressureLevel_023(%d)", int(p))
	}
}

// ResourceAgentProcessResultLimits_023 contains limit values obtained from ProcessResult.
type ResourceAgentProcessResultLimits_023 struct {
	CPULimit       float64 // CPU limit in cores (e.g., 2.5)
	MemoryLimit    float64 // memory limit in megabytes
	DiskIOLimit    float64 // disk I/O limit in MB/s
	NetworkLimit   float64 // network bandwidth limit in Mbps
	WallTimeLimit  float64 // wall time limit in seconds
}

// ResourceAgentSamplerPressure_023 holds pressure levels for each resource dimension.
type ResourceAgentSamplerPressure_023 struct {
	CPUPressure     PressureLevel_023
	MemoryPressure  PressureLevel_023
	DiskIOPressure  PressureLevel_023
	NetworkPressure PressureLevel_023
}

// ResourceAgentSamplerBridgeThresholds_023 defines the threshold boundaries between pressure levels.
type ResourceAgentSamplerBridgeThresholds_023 struct {
	CPULowHigh         float64 // CPU limit below this → low, above → medium/high
	CPUMediumHigh      float64 // CPU limit below this → medium, above → high
	CPUCritical        float64 // CPU limit above this → critical

	MemoryLowHigh      float64
	MemoryMediumHigh   float64
	MemoryCritical     float64

	DiskIOLowHigh      float64
	DiskIOMediumHigh   float64
	DiskIOCritical     float64

	NetworkLowHigh     float64
	NetworkMediumHigh  float64
	NetworkCritical    float64

	WallTimeLowHigh    float64
	WallTimeMediumHigh float64
	WallTimeCritical   float64
}

// ResourceAgentSamplerBridgeConfig_023 holds configuration for converting limits to pressure.
type ResourceAgentSamplerBridgeConfig_023 struct {
	Thresholds       ResourceAgentSamplerBridgeThresholds_023
	EnableWallTime   bool // if true, incorporate wall time limit
	EnableDiskIO     bool
	EnableNetwork    bool
	CustomLabel      string // optional label for logging
}

// Predefined default thresholds. These are deterministic and based on typical container limits.
var DefaultThresholds_023 = ResourceAgentSamplerBridgeThresholds_023{
	CPULowHigh:       1.0,   // 1 core
	CPUMediumHigh:    4.0,   // 4 cores
	CPUCritical:      8.0,   // 8 cores

	MemoryLowHigh:     512.0,   // 512 MB
	MemoryMediumHigh:  2048.0,  // 2 GB
	MemoryCritical:    8192.0,  // 8 GB

	DiskIOLowHigh:     50.0,    // 50 MB/s
	DiskIOMediumHigh:  200.0,   // 200 MB/s
	DiskIOCritical:    1000.0,  // 1 GB/s

	NetworkLowHigh:     100.0,   // 100 Mbps
	NetworkMediumHigh:  500.0,   // 500 Mbps
	NetworkCritical:    2000.0,  // 2 Gbps

	WallTimeLowHigh:    300.0,   // 5 min
	WallTimeMediumHigh: 1800.0,  // 30 min
	WallTimeCritical:   7200.0,  // 2 hours
}

// Predefined CPU ↔ pressure level mapping table (deterministic).
type cpuPressureEntry_023 struct {
	MinLimit float64           // inclusive lower bound
	MaxLimit float64           // exclusive upper bound (max is inclusive for last entry)
	Level    PressureLevel_023
}

var cpuPressureTable_023 = []cpuPressureEntry_023{
	{MinLimit: 0.0, MaxLimit: 1.0, Level: PressureLevelLow_023},
	{MinLimit: 1.0, MaxLimit: 4.0, Level: PressureLevelMedium_023},
	{MinLimit: 4.0, MaxLimit: 8.0, Level: PressureLevelHigh_023},
	{MinLimit: 8.0, MaxLimit: math.Inf(1), Level: PressureLevelCritical_023},
}

// Disk I/O pressure table.
var diskIOPressureTable_023 = []struct {
	MinLimit float64
	MaxLimit float64
	Level    PressureLevel_023
}{
	{MinLimit: 0.0, MaxLimit: 50.0, Level: PressureLevelLow_023},
	{MinLimit: 50.0, MaxLimit: 200.0, Level: PressureLevelMedium_023},
	{MinLimit: 200.0, MaxLimit: 1000.0, Level: PressureLevelHigh_023},
	{MinLimit: 1000.0, MaxLimit: math.Inf(1), Level: PressureLevelCritical_023},
}

// Network pressure table.
var networkPressureTable_023 = []struct {
	MinLimit float64
	MaxLimit float64
	Level    PressureLevel_023
}{
	{MinLimit: 0.0, MaxLimit: 100.0, Level: PressureLevelLow_023},
	{MinLimit: 100.0, MaxLimit: 500.0, Level: PressureLevelMedium_023},
	{MinLimit: 500.0, MaxLimit: 2000.0, Level: PressureLevelHigh_023},
	{MinLimit: 2000.0, MaxLimit: math.Inf(1), Level: PressureLevelCritical_023},
}

// Wall time pressure table.
var wallTimePressureTable_023 = []struct {
	MinLimit float64
	MaxLimit float64
	Level    PressureLevel_023
}{
	{MinLimit: 0.0, MaxLimit: 300.0, Level: PressureLevelLow_023},
	{MinLimit: 300.0, MaxLimit: 1800.0, Level: PressureLevelMedium_023},
	{MinLimit: 1800.0, MaxLimit: 7200.0, Level: PressureLevelHigh_023},
	{MinLimit: 7200.0, MaxLimit: math.Inf(1), Level: PressureLevelCritical_023},
}

// lookupPressureFromTable_023 finds the pressure level for a given value using a sorted table.
// Table must be sorted ascending by MinLimit. Last entry's MaxLimit is considered inclusive.
func lookupPressureFromTable_023(value float64, table []cpuPressureEntry_023) PressureLevel_023 {
	for i, entry := range table {
		if value >= entry.MinLimit {
			if i == len(table)-1 {
				return entry.Level // last entry includes anything >= MinLimit
			}
			if value < entry.MaxLimit {
				return entry.Level
			}
		}
	}
	return PressureLevelLow_023 // fallback
}

// lookupPressureFromDiskIOTable_023 looks up disk I/O pressure.
func lookupPressureFromDiskIOTable_023(value float64) PressureLevel_023 {
	for i, entry := range diskIOPressureTable_023 {
		if value >= entry.MinLimit {
			if i == len(diskIOPressureTable_023)-1 {
				return entry.Level
			}
			if value < entry.MaxLimit {
				return entry.Level
			}
		}
	}
	return PressureLevelLow_023
}

// lookupPressureFromNetworkTable_023 looks up network pressure.
func lookupPressureFromNetworkTable_023(value float64) PressureLevel_023 {
	for i, entry := range networkPressureTable_023 {
		if value >= entry.MinLimit {
			if i == len(networkPressureTable_023)-1 {
				return entry.Level
			}
			if value < entry.MaxLimit {
				return entry.Level
			}
		}
	}
	return PressureLevelLow_023
}

// lookupPressureFromWallTimeTable_023 looks up wall time pressure.
func lookupPressureFromWallTimeTable_023(value float64) PressureLevel_023 {
	for i, entry := range wallTimePressureTable_023 {
		if value >= entry.MinLimit {
			if i == len(wallTimePressureTable_023)-1 {
				return entry.Level
			}
			if value < entry.MaxLimit {
				return entry.Level
			}
		}
	}
	return PressureLevelLow_023
}

// Common validation errors.
var (
	ErrNegativeLimit_023        = errors.New("limit value must be non-negative")
	ErrInvalidPressureLevel_023 = errors.New("pressure level out of valid range")
	ErrInvalidThresholdOrder_023 = errors.New("thresholds must be strictly increasing: LowHigh < MediumHigh < Critical")
)

// ValidateResourceAgentSamplerBridgeConfig_023 validates the bridge configuration.
// It checks that thresholds are positive and strictly increasing.
func ValidateResourceAgentSamplerBridgeConfig_023(config ResourceAgentSamplerBridgeConfig_023) error {
	t := config.Thresholds
	// CPU thresholds
	if t.CPULowHigh < 0 || t.CPUMediumHigh < 0 || t.CPUCritical < 0 {
		return fmt.Errorf("%w: CPU thresholds must be non-negative", ErrNegativeLimit_023)
	}
	if t.CPULowHigh >= t.CPUMediumHigh || t.CPUMediumHigh >= t.CPUCritical {
		return fmt.Errorf("%w: CPU", ErrInvalidThresholdOrder_023)
	}
	// Memory
	if t.MemoryLowHigh < 0 || t.MemoryMediumHigh < 0 || t.MemoryCritical < 0 {
		return fmt.Errorf("%w: memory thresholds must be non-negative", ErrNegativeLimit_023)
	}
	if t.MemoryLowHigh >= t.MemoryMediumHigh || t.MemoryMediumHigh >= t.MemoryCritical {
		return fmt.Errorf("%w: memory", ErrInvalidThresholdOrder_023)
	}
	if config.EnableDiskIO {
		if t.DiskIOLowHigh < 0 || t.DiskIOMediumHigh < 0 || t.DiskIOCritical < 0 {
			return fmt.Errorf("%w: disk I/O thresholds must be non-negative", ErrNegativeLimit_023)
		}
		if t.DiskIOLowHigh >= t.DiskIOMediumHigh || t.DiskIOMediumHigh >= t.DiskIOCritical {
			return fmt.Errorf("%w: disk I/O", ErrInvalidThresholdOrder_023)
		}
	}
	if config.EnableNetwork {
		if t.NetworkLowHigh < 0 || t.NetworkMediumHigh < 0 || t.NetworkCritical < 0 {
			return fmt.Errorf("%w: network thresholds must be non-negative", ErrNegativeLimit_023)
		}
		if t.NetworkLowHigh >= t.NetworkMediumHigh || t.NetworkMediumHigh >= t.NetworkCritical {
			return fmt.Errorf("%w: network", ErrInvalidThresholdOrder_023)
		}
	}
	if config.EnableWallTime {
		if t.WallTimeLowHigh < 0 || t.WallTimeMediumHigh < 0 || t.WallTimeCritical < 0 {
			return fmt.Errorf("%w: wall time thresholds must be non-negative", ErrNegativeLimit_023)
		}
		if t.WallTimeLowHigh >= t.WallTimeMediumHigh || t.WallTimeMediumHigh >= t.WallTimeCritical {
			return fmt.Errorf("%w: wall time", ErrInvalidThresholdOrder_023)
		}
	}
	return nil
}

// ValidateResourceAgentProcessResultLimits_023 checks that all limits in the result are valid.
func ValidateResourceAgentProcessResultLimits_023(limits ResourceAgentProcessResultLimits_023) error {
	if limits.CPULimit < 0 {
		return fmt.Errorf("CPU limit %w: %f", ErrNegativeLimit_023, limits.CPULimit)
	}
	if limits.MemoryLimit < 0 {
		return fmt.Errorf("memory limit %w: %f", ErrNegativeLimit_023, limits.MemoryLimit)
	}
	if limits.DiskIOLimit < 0 {
		return fmt.Errorf("disk I/O limit %w: %f", ErrNegativeLimit_023, limits.DiskIOLimit)
	}
	if limits.NetworkLimit < 0 {
		return fmt.Errorf("network limit %w: %f", ErrNegativeLimit_023, limits.NetworkLimit)
	}
	if limits.WallTimeLimit < 0 {
		return fmt.Errorf("wall time limit %w: %f", ErrNegativeLimit_023, limits.WallTimeLimit)
	}
	return nil
}

// convertLimitToPressure_023 uses configured thresholds to determine pressure level.
func convertLimitToPressure_023(limit float64, lowHigh, mediumHigh, critical float64) PressureLevel_023 {
	if limit > critical {
		return PressureLevelCritical_023
	}
	if limit > mediumHigh {
		return PressureLevelHigh_023
	}
	if limit > lowHigh {
		return PressureLevelMedium_023
	}
	return PressureLevelLow_023
}

// ConvertLimitsToPressure_023 takes a ProcessResult's limits and a bridge configuration,
// and returns a SamplerPressure indicating the implied resource pressure.
func ConvertLimitsToPressure_023(limits ResourceAgentProcessResultLimits_023, config ResourceAgentSamplerBridgeConfig_023) (ResourceAgentSamplerPressure_023, error) {
	if err := ValidateResourceAgentProcessResultLimits_023(limits); err != nil {
		return ResourceAgentSamplerPressure_023{}, fmt.Errorf("invalid limits: %w", err)
	}
	if err := ValidateResourceAgentSamplerBridgeConfig_023(config); err != nil {
		return ResourceAgentSamplerPressure_023{}, fmt.Errorf("invalid config: %w", err)
	}

	pressure := ResourceAgentSamplerPressure_023{}
	t := config.Thresholds

	// CPU pressure
	pressure.CPUPressure = convertLimitToPressure_023(limits.CPULimit, t.CPULowHigh, t.CPUMediumHigh, t.CPUCritical)

	// Memory pressure
	pressure.MemoryPressure = convertLimitToPressure_023(limits.MemoryLimit, t.MemoryLowHigh, t.MemoryMediumHigh, t.MemoryCritical)

	// Disk I/O (if enabled)
	if config.EnableDiskIO {
		pressure.DiskIOPressure = convertLimitToPressure_023(limits.DiskIOLimit, t.DiskIOLowHigh, t.DiskIOMediumHigh, t.DiskIOCritical)
	} else {
		pressure.DiskIOPressure = PressureLevelLow_023 // not considered
	}

	// Network (if enabled)
	if config.EnableNetwork {
		pressure.NetworkPressure = convertLimitToPressure_023(limits.NetworkLimit, t.NetworkLowHigh, t.NetworkMediumHigh, t.NetworkCritical)
	} else {
		pressure.NetworkPressure = PressureLevelLow_023
	}

	return pressure, nil
}

// ConvertLimitsToPressureTableDriven_023 is an alternative conversion using the predefined tables.
// It directly maps each limit using the table lookup functions.
func ConvertLimitsToPressureTableDriven_023(limits ResourceAgentProcessResultLimits_023, enableDiskIO, enableNetwork bool) ResourceAgentSamplerPressure_023 {
	pressure := ResourceAgentSamplerPressure_023{}
	pressure.CPUPressure = lookupPressureFromTable_023(limits.CPULimit, cpuPressureTable_023)
	pressure.MemoryPressure = convertLimitToPressure_023(limits.MemoryLimit, DefaultThresholds_023.MemoryLowHigh, DefaultThresholds_023.MemoryMediumHigh, DefaultThresholds_023.MemoryCritical)
	if enableDiskIO {
		pressure.DiskIOPressure = lookupPressureFromDiskIOTable_023(limits.DiskIOLimit)
	} else {
		pressure.DiskIOPressure = PressureLevelLow_023
	}
	if enableNetwork {
		pressure.NetworkPressure = lookupPressureFromNetworkTable_023(limits.NetworkLimit)
	} else {
		pressure.NetworkPressure = PressureLevelLow_023
	}
	return pressure
}
