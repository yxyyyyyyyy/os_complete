// gen_limits_table_025.go
package chunk028

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Memory unit constants.
const (
	MemUnitBytes  = "B"
	MemUnitKiB    = "KiB"
	MemUnitMiB    = "MiB"
	MemUnitGiB    = "GiB"
	MemUnitTiB    = "TiB"
)

// MemUnitMultipliers maps memory unit strings to byte multipliers.
var MemUnitMultipliers_025 = map[string]uint64{
	MemUnitBytes: 1,
	MemUnitKiB:   1024,
	MemUnitMiB:   1024 * 1024,
	MemUnitGiB:   1024 * 1024 * 1024,
	MemUnitTiB:   1024 * 1024 * 1024 * 1024,
}

// DefaultCpuPeriod is the default cgroup cpu period in microseconds (100ms).
const DefaultCpuPeriod_025 uint64 = 100000

// ResourceAgentMemoryLimit represents a cgroup v2 memory limit.
type ResourceAgentMemoryLimit struct {
	Bytes uint64 // limit in bytes
}

// ResourceAgentPidsLimit represents a cgroup v2 pids limit.
type ResourceAgentPidsLimit struct {
	Max uint64 // maximum number of pids; 0 means unlimited
}

// ResourceAgentCpuLimit represents a cgroup v2 cpu limit.
type ResourceAgentCpuLimit struct {
	Quota  uint64 // quota in microseconds
	Period uint64 // period in microseconds
}

// ResourceAgentLimits aggregates all resource limits.
type ResourceAgentLimits struct {
	Memory *ResourceAgentMemoryLimit
	Pids   *ResourceAgentPidsLimit
	CPU    *ResourceAgentCpuLimit
}

// Validation table entries for deterministic tests.
type memValidationEntry_025 struct {
	input      string
	expected   uint64
	shouldFail bool
}

// ValidMemLimits_025 is a table of valid/invalid memory limit strings.
var ValidMemLimits_025 = []memValidationEntry_025{
	{"128MiB", 128 * 1024 * 1024, false},
	{"1GiB", 1 * 1024 * 1024 * 1024, false},
	{"512KiB", 512 * 1024, false},
	{"2TiB", 2 * 1024 * 1024 * 1024 * 1024, false},
	{"0B", 0, true},
	{"-1MiB", 0, true},
	{"abc", 0, true},
	{"", 0, true},
	{"10XB", 0, true},
}

// ValidPidsLimits_025 is a table of valid/invalid pids limit strings.
var ValidPidsLimits_025 = []struct {
	input      string
	expected   uint64
	shouldFail bool
}{
	{"100", 100, false},
	{"max", math.MaxUint64, false},
	{"0", 0, true},
	{"-5", 0, true},
	{"abc", 0, true},
}

// ValidCpuLimits_025 is a table of valid/invalid CPU limit strings.
var ValidCpuLimits_025 = []struct {
	input          string
	expectedQuota  uint64
	expectedPeriod uint64
	shouldFail     bool
}{
	{"2", 2 * DefaultCpuPeriod_025, DefaultCpuPeriod_025, false},
	{"0.5", DefaultCpuPeriod_025 / 2, DefaultCpuPeriod_025, false},
	{"1.5", 3 * DefaultCpuPeriod_025 / 2, DefaultCpuPeriod_025, false},
	{"0", 0, DefaultCpuPeriod_025, true},
	{"-1", 0, 0, true},
	{"abc", 0, 0, true},
	{"", 0, 0, true},
}

// minMemoryBytes is the minimum allowed memory limit (4 MB).
const minMemoryBytes_025 uint64 = 4 * 1024 * 1024

// maxMemoryBytes is the maximum allowed memory limit (1 TB).
const maxMemoryBytes_025 uint64 = 1 * 1024 * 1024 * 1024 * 1024

// minPids is the minimum allowed pids limit.
const minPids_025 uint64 = 1

// maxPids is the maximum allowed pids limit.
const maxPids_025 uint64 = 65536

// convertMemoryToBytes_025 parses a memory string like "128MiB" into bytes.
func convertMemoryToBytes_025(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("memory string too short: %q", s)
	}
	// Determine unit suffix.
	var unit string
	switch {
	case strings.HasSuffix(s, MemUnitTiB):
		unit = MemUnitTiB
	case strings.HasSuffix(s, MemUnitGiB):
		unit = MemUnitGiB
	case strings.HasSuffix(s, MemUnitMiB):
		unit = MemUnitMiB
	case strings.HasSuffix(s, MemUnitKiB):
		unit = MemUnitKiB
	case strings.HasSuffix(s, MemUnitBytes):
		unit = MemUnitBytes
	default:
		return 0, fmt.Errorf("unknown memory unit in %q", s)
	}
	// Extract numeric part.
	numStr := strings.TrimSuffix(s, unit)
	numStr = strings.TrimSpace(numStr)
	value, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value %q: %w", numStr, err)
	}
	multiplier, ok := MemUnitMultipliers_025[unit]
	if !ok {
		return 0, fmt.Errorf("no multiplier for unit %q", unit)
	}
	// Check for overflow.
	if value > math.MaxUint64/multiplier {
		return 0, errors.New("memory value overflow")
	}
	bytes := value * multiplier
	return bytes, nil
}

// ParseMemoryLimit_025 parses a memory limit string and returns a MemoryLimit.
func ParseMemoryLimit_025(s string) (*ResourceAgentMemoryLimit, error) {
	bytes, err := convertMemoryToBytes_025(s)
	if err != nil {
		return nil, err
	}
	return &ResourceAgentMemoryLimit{Bytes: bytes}, nil
}

// ParsePidsLimit_025 parses a pids limit string. "max" means unlimited.
func ParsePidsLimit_025(s string) (*ResourceAgentPidsLimit, error) {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "max") {
		return &ResourceAgentPidsLimit{Max: math.MaxUint64}, nil
	}
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid pids limit %q: %w", s, err)
	}
	return &ResourceAgentPidsLimit{Max: value}, nil
}

// ParseCpuLimit_025 parses a CPU limit string (e.g., "2", "0.5", "1.5").
// Returns quota and period in microseconds.
func ParseCpuLimit_025(s string) (*ResourceAgentCpuLimit, error) {
	s = strings.TrimSpace(s)
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU limit %q: %w", s, err)
	}
	if value <= 0 {
		return nil, fmt.Errorf("CPU limit must be positive, got %v", value)
	}
	period := DefaultCpuPeriod_025
	// quota = value * period
	quotaFloat := value * float64(period)
	if quotaFloat > float64(math.MaxUint64) {
		return nil, errors.New("CPU quota too large")
	}
	quota := uint64(quotaFloat)
	// Ensure quota at least 1 if value > 0
	if quota == 0 {
		quota = 1
	}
	return &ResourceAgentCpuLimit{Quota: quota, Period: period}, nil
}

// ValidateMemoryLimit_025 validates a memory limit.
func ValidateMemoryLimit_025(limit *ResourceAgentMemoryLimit) error {
	if limit == nil {
		return errors.New("memory limit is nil")
	}
	if limit.Bytes < minMemoryBytes_025 {
		return fmt.Errorf("memory limit %d bytes is below minimum %d", limit.Bytes, minMemoryBytes_025)
	}
	if limit.Bytes > maxMemoryBytes_025 {
		return fmt.Errorf("memory limit %d bytes exceeds maximum %d", limit.Bytes, maxMemoryBytes_025)
	}
	// Memory limit should be page-aligned (4KB).
	pageSize := uint64(4096)
	if limit.Bytes%pageSize != 0 {
		return fmt.Errorf("memory limit %d is not page-aligned (4KB)", limit.Bytes)
	}
	return nil
}

// ValidatePidsLimit_025 validates a pids limit.
func ValidatePidsLimit_025(limit *ResourceAgentPidsLimit) error {
	if limit == nil {
		return errors.New("pids limit is nil")
	}
	// max == 0 means unlimited, which is valid.
	if limit.Max == 0 {
		return nil
	}
	if limit.Max < minPids_025 {
		return fmt.Errorf("pids limit %d below minimum %d", limit.Max, minPids_025)
	}
	if limit.Max > maxPids_025 {
		return fmt.Errorf("pids limit %d exceeds maximum %d", limit.Max, maxPids_025)
	}
	return nil
}

// ValidateCpuLimit_025 validates a CPU limit.
func ValidateCpuLimit_025(limit *ResourceAgentCpuLimit) error {
	if limit == nil {
		return errors.New("cpu limit is nil")
	}
	if limit.Period == 0 {
		return errors.New("cpu period cannot be zero")
	}
	if limit.Quota == 0 {
		return errors.New("cpu quota cannot be zero")
	}
	// Quota must be <= period (no overcommit? Actually cgroup allows, but we restrict to <= period for simplicity)
	if limit.Quota > limit.Period {
		return fmt.Errorf("cpu quota %d exceeds period %d", limit.Quota, limit.Period)
	}
	// Period should be reasonable: between 1000us and 1s.
	if limit.Period < 1000 || limit.Period > 1000000 {
		return fmt.Errorf("cpu period %d outside valid range [1000, 1000000] us", limit.Period)
	}
	return nil
}

// ValidateResourceLimits_025 validates all resource limits collectively.
func ValidateResourceLimits_025(limits *ResourceAgentLimits) error {
	if limits == nil {
		return errors.New("resource limits are nil")
	}
	if err := ValidateMemoryLimit_025(limits.Memory); err != nil {
		return fmt.Errorf("memory limit: %w", err)
	}
	if err := ValidatePidsLimit_025(limits.Pids); err != nil {
		return fmt.Errorf("pids limit: %w", err)
	}
	if err := ValidateCpuLimit_025(limits.CPU); err != nil {
		return fmt.Errorf("cpu limit: %w", err)
	}
	return nil
}

// ValidateMemoryLimitFromString_025 is a convenience wrapper to parse and validate.
func ValidateMemoryLimitFromString_025(s string) error {
	limit, err := ParseMemoryLimit_025(s)
	if err != nil {
		return err
	}
	return ValidateMemoryLimit_025(limit)
}

// ValidatePidsLimitFromString_025 is a convenience wrapper.
func ValidatePidsLimitFromString_025(s string) error {
	limit, err := ParsePidsLimit_025(s)
	if err != nil {
		return err
	}
	return ValidatePidsLimit_025(limit)
}

// ValidateCpuLimitFromString_025 is a convenience wrapper.
func ValidateCpuLimitFromString_025(s string) error {
	limit, err := ParseCpuLimit_025(s)
	if err != nil {
		return err
	}
	return ValidateCpuLimit_025(limit)
}

// ResourceAgentGenerateDefaultLimits_025 returns a set of default resource limits.
func ResourceAgentGenerateDefaultLimits_025() *ResourceAgentLimits {
	return &ResourceAgentLimits{
		Memory: &ResourceAgentMemoryLimit{Bytes: 512 * 1024 * 1024}, // 512 MiB
		Pids:   &ResourceAgentPidsLimit{Max: 1024},
		CPU:    &ResourceAgentCpuLimit{Quota: DefaultCpuPeriod_025, Period: DefaultCpuPeriod_025}, // 1 CPU core
	}
}

// ResourceAgentLimitsEqual_025 checks equality of two limits (nil-safe).
func ResourceAgentLimitsEqual_025(a, b *ResourceAgentLimits) bool {
	if a == nil || b == nil {
		return a == b
	}
	return equalMemory_025(a.Memory, b.Memory) &&
		equalPids_025(a.Pids, b.Pids) &&
		equalCPU_025(a.CPU, b.CPU)
}

func equalMemory_025(a, b *ResourceAgentMemoryLimit) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Bytes == b.Bytes
}

func equalPids_025(a, b *ResourceAgentPidsLimit) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Max == b.Max
}

func equalCPU_025(a, b *ResourceAgentCpuLimit) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Quota == b.Quota && a.Period == b.Period
}
