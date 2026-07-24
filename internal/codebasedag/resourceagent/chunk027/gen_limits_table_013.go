package chunk027

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Unit multipliers for memory limits (cgroup v2 uses bytes).
// Keys are suffixes (case-insensitive), values are multipliers to bytes.
var ResourceAgentMemoryUnitTable_013 = map[string]int64{
	"B":  1,
	"K":  1024,
	"KB": 1024,
	"KI": 1024, // kibibyte alias
	"KIB": 1024,
	"M":  1024 * 1024,
	"MB": 1024 * 1024,
	"MI": 1024 * 1024, // mebibyte
	"MIB": 1024 * 1024,
	"G":  1024 * 1024 * 1024,
	"GB": 1024 * 1024 * 1024,
	"GI": 1024 * 1024 * 1024,
	"GIB": 1024 * 1024 * 1024,
	"T":  1024 * 1024 * 1024 * 1024,
	"TB": 1024 * 1024 * 1024 * 1024,
	"TI": 1024 * 1024 * 1024 * 1024,
	"TIB": 1024 * 1024 * 1024 * 1024,
	"P":  1024 * 1024 * 1024 * 1024 * 1024,
	"PB": 1024 * 1024 * 1024 * 1024 * 1024,
	"PI": 1024 * 1024 * 1024 * 1024 * 1024,
	"PIB": 1024 * 1024 * 1024 * 1024 * 1024,
	"E":  1024 * 1024 * 1024 * 1024 * 1024 * 1024,
	"EB": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
	"EI": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
	"EIB": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
}

// ResourceAgentPidsMaxValidTable defines allowed values for pids.max.
// "max" means unlimited, otherwise a non-negative integer is accepted.
var ResourceAgentPidsMaxValidTable_013 = []struct {
	value   string
	isValid bool
}{
	{"max", true},
	{"0", true},
	{"1", true},
	{"1024", true},
	{"65535", true},
	{"-1", false},
	{"", false},
	{"abc", false},
}

// ResourceAgentCpuWeightRange defines the valid range for cpu.weight (cgroup v2).
const ResourceAgentCpuWeightMin_013 = 1
const ResourceAgentCpuWeightMax_013 = 10000

// ResourceAgentCpuMaxPattern defines the format for cpu.max: "$quota $period" or "max".
// quota and period are microseconds. Period cannot be zero.
var ResourceAgentCpuMaxPattern_013 = regexp.MustCompile(`^(max|\d+)\s+(\d+)$`)

// MemoryLimitInfo_013 holds parsed memory limit value and unit.
type MemoryLimitInfo_013 struct {
	Bytes int64
	Raw   string
}

// PidsLimitInfo_013 holds parsed pids.max value (or Max flag).
type PidsLimitInfo_013 struct {
	Value   int64
	IsMax   bool
}

// CpuLimitInfo_013 holds parsed CPU limit (quota/period).
type CpuLimitInfo_013 struct {
	Quota  int64 // microseconds, or -1 for max
	Period int64 // microseconds, or 0 if max
}

// CpuWeightInfo_013 holds parsed cpu.weight value.
type CpuWeightInfo_013 struct {
	Weight int64
}

// CpusetCpusInfo_013 holds parsed list of CPUs.
type CpusetCpusInfo_013 struct {
	CPUs []int
}

// ParseMemoryLimitUnit_013 parses a memory limit string like "1G", "500M", "1048576" or "max".
// Returns the value in bytes. For "max", returns MaxInt64 (representing unlimited).
func ParseMemoryLimitUnit_013(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty memory limit string")
	}
	trimmed := strings.TrimSpace(s)
	if strings.EqualFold(trimmed, "max") {
		return math.MaxInt64, nil
	}
	// Separate number and unit
	var numStr string
	var unit string
	for i, ch := range trimmed {
		if ch < '0' || ch > '9' {
			numStr = trimmed[:i]
			unit = trimmed[i:]
			break
		}
	}
	if numStr == "" {
		numStr = trimmed
		unit = ""
	}
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit number %q: %w", numStr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("negative memory limit %d", num)
	}
	if num > math.MaxInt64 {
		return 0, fmt.Errorf("memory limit %d overflows int64", num)
	}
	if unit == "" {
		return num, nil
	}
	unitUpper := strings.ToUpper(strings.TrimSpace(unit))
	multiplier, ok := ResourceAgentMemoryUnitTable_013[unitUpper]
	if !ok {
		return 0, fmt.Errorf("unknown memory unit %q", unit)
	}
	// Check overflow
	if num > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("memory limit %s would overflow int64", trimmed)
	}
	result := num * multiplier
	return result, nil
}

// ValidateMemoryLimit_013 validates a memory limit string.
func ValidateMemoryLimit_013(s string) error {
	_, err := ParseMemoryLimitUnit_013(s)
	return err
}

// ParsePidsLimit_013 parses a pids.max string ("max" or integer).
func ParsePidsLimit_013(s string) (PidsLimitInfo_013, error) {
	trimmed := strings.TrimSpace(s)
	if strings.EqualFold(trimmed, "max") {
		return PidsLimitInfo_013{IsMax: true}, nil
	}
	val, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return PidsLimitInfo_013{}, fmt.Errorf("invalid pids.max value %q: %w", trimmed, err)
	}
	if val < 0 {
		return PidsLimitInfo_013{}, fmt.Errorf("negative pids.max value %d", val)
	}
	return PidsLimitInfo_013{Value: val}, nil
}

// ValidatePidsLimit_013 validates a pids.max string.
func ValidatePidsLimit_013(s string) error {
	_, err := ParsePidsLimit_013(s)
	return err
}

// ParseCpuMax_013 parses a cpu.max string ("<quota> <period>" or "max").
func ParseCpuMax_013(s string) (CpuLimitInfo_013, error) {
	trimmed := strings.TrimSpace(s)
	if strings.EqualFold(trimmed, "max") {
		return CpuLimitInfo_013{Quota: -1, Period: 0}, nil
	}
	matches := ResourceAgentCpuMaxPattern_013.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return CpuLimitInfo_013{}, fmt.Errorf("invalid cpu.max format %q, expected \"<quota> <period>\" or \"max\"", trimmed)
	}
	quotaStr, periodStr := matches[1], matches[2]
	quota, err := strconv.ParseInt(quotaStr, 10, 64)
	if err != nil {
		return CpuLimitInfo_013{}, fmt.Errorf("invalid cpu.max quota %q: %w", quotaStr, err)
	}
	period, err := strconv.ParseInt(periodStr, 10, 64)
	if err != nil {
		return CpuLimitInfo_013{}, fmt.Errorf("invalid cpu.max period %q: %w", periodStr, err)
	}
	if period == 0 {
		return CpuLimitInfo_013{}, fmt.Errorf("cpu.max period cannot be zero")
	}
	if quota < 0 {
		return CpuLimitInfo_013{}, fmt.Errorf("cpu.max quota cannot be negative")
	}
	return CpuLimitInfo_013{Quota: quota, Period: period}, nil
}

// ValidateCpuMax_013 validates a cpu.max string.
func ValidateCpuMax_013(s string) error {
	_, err := ParseCpuMax_013(s)
	return err
}

// ParseCpuWeight_013 parses a cpu.weight string and validates range.
func ParseCpuWeight_013(s string) (CpuWeightInfo_013, error) {
	trimmed := strings.TrimSpace(s)
	val, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return CpuWeightInfo_013{}, fmt.Errorf("invalid cpu.weight %q: %w", trimmed, err)
	}
	if val < ResourceAgentCpuWeightMin_013 || val > ResourceAgentCpuWeightMax_013 {
		return CpuWeightInfo_013{}, fmt.Errorf("cpu.weight %d out of range [%d, %d]", val, ResourceAgentCpuWeightMin_013, ResourceAgentCpuWeightMax_013)
	}
	return CpuWeightInfo_013{Weight: val}, nil
}

// ValidateCpuWeight_013 validates a cpu.weight string.
func ValidateCpuWeight_013(s string) error {
	_, err := ParseCpuWeight_013(s)
	return err
}

// ParseCpusetCpus_013 parses a cpuset.cpus string like "0-3,5,8-11".
func ParseCpusetCpus_013(s string) (CpusetCpusInfo_013, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return CpusetCpusInfo_013{}, errors.New("empty cpuset string")
	}
	parts := strings.Split(trimmed, ",")
	var cpus []int
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			// Range
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return CpusetCpusInfo_013{}, fmt.Errorf("invalid cpu range %q", part)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				return CpusetCpusInfo_013{}, fmt.Errorf("invalid cpu range numbers in %q", part)
			}
			if start < 0 || end < 0 || end < start {
				return CpusetCpusInfo_013{}, fmt.Errorf("invalid cpu range %q", part)
			}
			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			// Single CPU
			cpu, err := strconv.Atoi(part)
			if err != nil {
				return CpusetCpusInfo_013{}, fmt.Errorf("invalid cpu number %q: %w", part, err)
			}
			if cpu < 0 {
				return CpusetCpusInfo_013{}, fmt.Errorf("negative cpu number %d", cpu)
			}
			cpus = append(cpus, cpu)
		}
	}
	// Remove duplicates and sort
	cpus = removeDuplicatesAndSort_013(cpus)
	return CpusetCpusInfo_013{CPUs: cpus}, nil
}

// ValidateCpusetCpus_013 validates a cpuset.cpus string.
func ValidateCpusetCpus_013(s string) error {
	_, err := ParseCpusetCpus_013(s)
	return err
}

// removeDuplicatesAndSort_013 removes duplicate integers and sorts ascending.
func removeDuplicatesAndSort_013(nums []int) []int {
	if len(nums) == 0 {
		return nums
	}
	seen := make(map[int]bool)
	result := make([]int, 0, len(nums))
	for _, v := range nums {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	// Bubble sort (small list, deterministic)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// ResourceAgentMemoryLimitSuffixTable is a deterministic list of valid suffixes and examples.
var ResourceAgentMemoryLimitSuffixTable_013 = []struct {
	Suffix   string
	Example  string
	Bytes    int64
}{
	{"B", "1024B", 1024},
	{"K", "1K", 1024},
	{"KB", "1KB", 1024},
	{"Ki", "1Ki", 1024},
	{"KiB", "1KiB", 1024},
	{"M", "1M", 1024 * 1024},
	{"MB", "1MB", 1024 * 1024},
	{"Mi", "1Mi", 1024 * 1024},
	{"MiB", "1MiB", 1024 * 1024},
	{"G", "1G", 1024 * 1024 * 1024},
	{"GB", "1GB", 1024 * 1024 * 1024},
	{"Gi", "1Gi", 1024 * 1024 * 1024},
	{"GiB", "1GiB", 1024 * 1024 * 1024},
}

// ResourceAgentCpusetValidPatterns_013 lists valid cpuset patterns for testing.
var ResourceAgentCpusetValidPatterns_013 = []string{
	"0",
	"0-3",
	"0-3,5",
	"0-3,5,8-11",
	"0-2,4-6",
	"7",
	"0-15",
}

// ResourceAgentCpusetInvalidPatterns_013 lists invalid cpuset patterns.
var ResourceAgentCpusetInvalidPatterns_013 = []string{
	"",
	"-1",
	"0-",
	"-3",
	"0-3-4",
	"a",
	"0,,",
}

// ResourceAgentMemoryInvalidExamples_013 lists invalid memory limit strings.
var ResourceAgentMemoryInvalidExamples_013 = []string{
	"",
	"abc",
	"1ZB",      // unknown unit
	"1.5G",     // decimal not allowed
	"-1K",
	"9223372036854775808B", // overflow
}

// ResourceAgentPidsLimitTable_013 provides test cases for pids.max validation.
var ResourceAgentPidsLimitTable_013 = []struct {
	Input    string
	Expected interface{} // nil for error, else PidsLimitInfo_013
}{
	{"max", PidsLimitInfo_013{IsMax: true}},
	{"0", PidsLimitInfo_013{Value: 0}},
	{"100", PidsLimitInfo_013{Value: 100}},
	{"-1", nil},
	{"", nil},
	{"abc", nil},
	{"999999999999999999999", nil}, // overflow
}

// ResourceAgentCpuWeightTable_013 provides valid and invalid cpu.weight examples.
var ResourceAgentCpuWeightTable_013 = []struct {
	Input   string
	IsValid bool
}{
	{"1", true},
	{"500", true},
	{"10000", true},
	{"0", false},
	{"10001", false},
	{"-5", false},
	{"abc", false},
}

// ResourceAgentCpuMaxValidExamples_013 lists valid cpu.max strings.
var ResourceAgentCpuMaxValidExamples_013 = []string{
	"max",
	"100000 100000",
	"50000 100000",
	"200000 500000",
}

// ResourceAgentCpuMaxInvalidExamples_013 lists invalid cpu.max strings.
var ResourceAgentCpuMaxInvalidExamples_013 = []string{
	"",
	"max 100000",      // quota "max" not allowed with period
	"100000",          // missing period
	"100000 0",        // period zero
	"-1 100000",       // negative quota
	"abc 100000",      // invalid quota
	"100000 abc",      // invalid period
	"100000 100000 1", // extra field
}

// ValidateAllLimits_013 is a top-level validation function that accepts a map of limit names to values.
// It returns a map of errors for invalid limits.
func ValidateAllLimits_013(limits map[string]string) map[string]error {
	errs := make(map[string]error)
	for name, value := range limits {
		var err error
		switch strings.ToLower(name) {
		case "memory.max", "memory.high", "memory.low", "memory.min":
			err = ValidateMemoryLimit_013(value)
		case "pids.max":
			err = ValidatePidsLimit_013(value)
		case "cpu.max":
			err = ValidateCpuMax_013(value)
		case "cpu.weight":
			err = ValidateCpuWeight_013(value)
		case "cpuset.cpus":
			err = ValidateCpusetCpus_013(value)
		default:
			err = fmt.Errorf("unknown limit: %s", name)
		}
		if err != nil {
			errs[name] = err
		}
	}
	return errs
}
