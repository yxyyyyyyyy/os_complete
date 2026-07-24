package chunk029

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ResourceAgentCgroupV2LimitType_037 enumerates the resource types validated.
type ResourceAgentCgroupV2LimitType_037 int

const (
	ResourceAgentLimitMemory_037 ResourceAgentCgroupV2LimitType_037 = iota
	ResourceAgentLimitPIDs_037
	ResourceAgentLimitCPU_037
)

func (t ResourceAgentCgroupV2LimitType_037) String() string {
	switch t {
	case ResourceAgentLimitMemory_037:
		return "memory"
	case ResourceAgentLimitPIDs_037:
		return "pids"
	case ResourceAgentLimitCPU_037:
		return "cpu"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// ResourceAgentCgroupV2Limit_037 holds a parsed cgroup v2 limit value.
type ResourceAgentCgroupV2Limit_037 struct {
	Type ResourceAgentCgroupV2LimitType_037
	// For memory: value in bytes, -1 means max.
	// For pids: maximum number of pids, -1 means max.
	// For cpu: maximum bandwidth as time.Duration per period, 0 means unlimited.
	Value   int64
	Raw     string
	Parsed  bool
	Err     error
}

// ResourceAgentValidateMemoryLimit_037 parses and validates a memory.max value (e.g., "max", "1G", "1048576").
// Returns an error if the string is not a valid cgroup v2 memory limit.
func ResourceAgentValidateMemoryLimit_037(raw string) (ResourceAgentCgroupV2Limit_037, error) {
	lim := ResourceAgentCgroupV2Limit_037{
		Type: ResourceAgentLimitMemory_037,
		Raw:  raw,
	}
	if raw == "max" {
		lim.Value = -1
		lim.Parsed = true
		return lim, nil
	}
	// Accept suffixes: K, M, G, T, P, E (case-insensitive)
	suffixes := map[string]int64{
		"K": 1 << 10,
		"M": 1 << 20,
		"G": 1 << 30,
		"T": 1 << 40,
		"P": 1 << 50,
		"E": 1 << 60,
	}
	valStr := raw
	multiplier := int64(1)
	for suffix, factor := range suffixes {
		if strings.HasSuffix(strings.ToUpper(raw), suffix) {
			valStr = strings.TrimSuffix(raw, suffix)
			valStr = strings.TrimSuffix(valStr, strings.ToLower(suffix))
			multiplier = factor
			break
		}
	}
	n, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return lim, fmt.Errorf("memory limit %q: invalid integer: %w", raw, err)
	}
	if n < 0 {
		return lim, fmt.Errorf("memory limit %q: negative value not allowed", raw)
	}
	if multiplier > 1 {
		// check for overflow
		if n > math.MaxInt64/multiplier {
			return lim, fmt.Errorf("memory limit %q: overflow after multiplier", raw)
		}
		n *= multiplier
	}
	lim.Value = n
	lim.Parsed = true
	return lim, nil
}

// ResourceAgentValidatePIDsLimit_037 parses a pids.max string ("max" or a number).
func ResourceAgentValidatePIDsLimit_037(raw string) (ResourceAgentCgroupV2Limit_037, error) {
	lim := ResourceAgentCgroupV2Limit_037{
		Type: ResourceAgentLimitPIDs_037,
		Raw:  raw,
	}
	if raw == "max" {
		lim.Value = -1
		lim.Parsed = true
		return lim, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return lim, fmt.Errorf("pids limit %q: invalid integer: %w", raw, err)
	}
	if n < 0 {
		return lim, fmt.Errorf("pids limit %q: negative not allowed", raw)
	}
	lim.Value = n
	lim.Parsed = true
	return lim, nil
}

// ResourceAgentValidateCPULimit_037 parses a CPU limit from cgroup v2 cpu.max (e.g., "100000 100000").
// The format is "quota period" or "max" for unlimited.
func ResourceAgentValidateCPULimit_037(raw string) (ResourceAgentCgroupV2Limit_037, error) {
	lim := ResourceAgentCgroupV2Limit_037{
		Type: ResourceAgentLimitCPU_037,
		Raw:  raw,
	}
	if raw == "max" {
		lim.Value = 0 // unlimited
		lim.Parsed = true
		return lim, nil
	}
	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return lim, fmt.Errorf("cpu.max %q: expected 2 fields (quota period)", raw)
	}
	quotaStr, periodStr := parts[0], parts[1]
	period, err := strconv.ParseInt(periodStr, 10, 64)
	if err != nil {
		return lim, fmt.Errorf("cpu.max period %q: %w", periodStr, err)
	}
	if period <= 0 {
		return lim, fmt.Errorf("cpu.max period %q: must be positive", periodStr)
	}
	if quotaStr == "max" {
		// unlimited
		lim.Value = 0
		lim.Parsed = true
		return lim, nil
	}
	quota, err := strconv.ParseInt(quotaStr, 10, 64)
	if err != nil {
		return lim, fmt.Errorf("cpu.max quota %q: %w", quotaStr, err)
	}
	if quota < 0 {
		return lim, fmt.Errorf("cpu.max quota %q: negative not allowed", quotaStr)
	}
	// Represent as duration in nanoseconds (period is microseconds)
	lim.Value = int64(time.Duration(quota) * time.Microsecond)
	lim.Parsed = true
	return lim, nil
}

// ResourceAgentValidateLimits_037 validates a set of cgroup v2 limits given as a map.
// Returns a map of errors per field name, or nil if all valid.
func ResourceAgentValidateLimits_037(limits map[ResourceAgentCgroupV2LimitType_037]string) map[ResourceAgentCgroupV2LimitType_037]error {
	errors := make(map[ResourceAgentCgroupV2LimitType_037]error)
	for typ, raw := range limits {
		var err error
		switch typ {
		case ResourceAgentLimitMemory_037:
			_, err = ResourceAgentValidateMemoryLimit_037(raw)
		case ResourceAgentLimitPIDs_037:
			_, err = ResourceAgentValidatePIDsLimit_037(raw)
		case ResourceAgentLimitCPU_037:
			_, err = ResourceAgentValidateCPULimit_037(raw)
		default:
			err = fmt.Errorf("unknown limit type %v", typ)
		}
		if err != nil {
			errors[typ] = err
		}
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}

// ResourceAgentCgroupV2LimitTableEntry_037 is a single row in a deterministic validation table.
type ResourceAgentCgroupV2LimitTableEntry_037 struct {
	Type          ResourceAgentCgroupV2LimitType_037
	Raw           string
	ExpectedValue int64
	ExpectError   bool
	ErrorContains string // substring expected if error
}

// ResourceAgentCgroupV2LimitTable_037 returns deterministic test data for validation.
// The table is fixed and ordered; used for both unit tests and documentation.
func ResourceAgentCgroupV2LimitTable_037() []ResourceAgentCgroupV2LimitTableEntry_037 {
	return []ResourceAgentCgroupV2LimitTableEntry_037{
		{Type: ResourceAgentLimitMemory_037, Raw: "max", ExpectedValue: -1, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "0", ExpectedValue: 0, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1", ExpectedValue: 1, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1024", ExpectedValue: 1024, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1K", ExpectedValue: 1024, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1k", ExpectedValue: 1024, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1M", ExpectedValue: 1 << 20, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1G", ExpectedValue: 1 << 30, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1T", ExpectedValue: 1 << 40, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1P", ExpectedValue: 1 << 50, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "1E", ExpectedValue: 1 << 60, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "2G", ExpectedValue: 2 << 30, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "500M", ExpectedValue: 500 << 20, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "2147483648", ExpectedValue: 2147483648, ExpectError: false},
		{Type: ResourceAgentLimitMemory_037, Raw: "999999999999999999", ExpectedValue: 999999999999999999, ExpectError: false},

		// Error cases for memory
		{Type: ResourceAgentLimitMemory_037, Raw: "", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitMemory_037, Raw: "-1", ExpectError: true, ErrorContains: "negative"},
		{Type: ResourceAgentLimitMemory_037, Raw: "abc", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitMemory_037, Raw: "1.5G", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitMemory_037, Raw: "1KB", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitMemory_037, Raw: "999999999999999999999", ExpectError: true, ErrorContains: "overflow"},

		// PIDs
		{Type: ResourceAgentLimitPIDs_037, Raw: "max", ExpectedValue: -1, ExpectError: false},
		{Type: ResourceAgentLimitPIDs_037, Raw: "0", ExpectedValue: 0, ExpectError: false},
		{Type: ResourceAgentLimitPIDs_037, Raw: "100", ExpectedValue: 100, ExpectError: false},
		{Type: ResourceAgentLimitPIDs_037, Raw: "65535", ExpectedValue: 65535, ExpectError: false},
		{Type: ResourceAgentLimitPIDs_037, Raw: "1234567890123", ExpectedValue: 1234567890123, ExpectError: false},
		{Type: ResourceAgentLimitPIDs_037, Raw: "", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitPIDs_037, Raw: "-10", ExpectError: true, ErrorContains: "negative"},
		{Type: ResourceAgentLimitPIDs_037, Raw: "abc", ExpectError: true, ErrorContains: "invalid"},

		// CPU
		{Type: ResourceAgentLimitCPU_037, Raw: "max", ExpectedValue: 0, ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000 100000", ExpectedValue: int64(100 * time.Millisecond), ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "50000 100000", ExpectedValue: int64(50 * time.Millisecond), ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "max 100000", ExpectedValue: 0, ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "200000 100000", ExpectedValue: int64(200 * time.Millisecond), ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000 100000", ExpectedValue: int64(time.Duration(100000) * time.Microsecond), ExpectError: false},
		{Type: ResourceAgentLimitCPU_037, Raw: "", ExpectError: true, ErrorContains: "expected 2 fields"},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000", ExpectError: true, ErrorContains: "expected 2 fields"},
		{Type: ResourceAgentLimitCPU_037, Raw: "abc 100000", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000 abc", ExpectError: true, ErrorContains: "invalid"},
		{Type: ResourceAgentLimitCPU_037, Raw: "-100000 100000", ExpectError: true, ErrorContains: "negative"},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000 0", ExpectError: true, ErrorContains: "must be positive"},
		{Type: ResourceAgentLimitCPU_037, Raw: "100000 -100000", ExpectError: true, ErrorContains: "invalid"},
	}
}

// ResourceAgentParseMemoryLimitToBytes_037 is a helper to convert a string to bytes, not exported from Validate function.
func ResourceAgentParseMemoryLimitToBytes_037(raw string) (int64, error) {
	lim, err := ResourceAgentValidateMemoryLimit_037(raw)
	if err != nil {
		return 0, err
	}
	return lim.Value, nil
}

// ResourceAgentParsePIDsLimitToInt64_037 parses pids.max to int64.
func ResourceAgentParsePIDsLimitToInt64_037(raw string) (int64, error) {
	lim, err := ResourceAgentValidatePIDsLimit_037(raw)
	if err != nil {
		return 0, err
	}
	return lim.Value, nil
}

// ResourceAgentParseCPULimitToDuration_037 returns the quota as time.Duration (0 means unlimited).
// Note: period is not returned; caller should parse separately if needed.
func ResourceAgentParseCPULimitToDuration_037(raw string) (time.Duration, error) {
	lim, err := ResourceAgentValidateCPULimit_037(raw)
	if err != nil {
		return 0, err
	}
	// For unlimited we store 0, otherwise quota in microseconds.
	if lim.Value == 0 {
		return 0, nil
	}
	return time.Duration(lim.Value), nil
}

// ResourceAgentFormatMemoryLimitFromBytes_037 formats an int64 byte value into cgroup v2 format.
// -1 is formatted as "max". Values that are multiples of large units are shortened.
func ResourceAgentFormatMemoryLimitFromBytes_037(bytes int64) string {
	if bytes < 0 {
		return "max"
	}
	// Try largest suffix first
	type suffix struct {
		unit int64
		str  string
	}
	suffixes := []suffix{
		{1 << 60, "E"},
		{1 << 50, "P"},
		{1 << 40, "T"},
		{1 << 30, "G"},
		{1 << 20, "M"},
		{1 << 10, "K"},
	}
	for _, s := range suffixes {
		if bytes >= s.unit && bytes%s.unit == 0 {
			return fmt.Sprintf("%d%s", bytes/s.unit, s.str)
		}
	}
	return fmt.Sprintf("%d", bytes)
}

// ResourceAgentFormatPIDsLimit_037 formats a pids limit. -1 => "max".
func ResourceAgentFormatPIDsLimit_037(limit int64) string {
	if limit < 0 {
		return "max"
	}
	return fmt.Sprintf("%d", limit)
}
