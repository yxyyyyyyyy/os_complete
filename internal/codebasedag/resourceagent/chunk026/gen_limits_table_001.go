package chunk026

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ResourceAgentMemoryLimit_001 holds a parsed cgroup v2 memory limit in bytes.
type ResourceAgentMemoryLimit_001 struct {
	Bytes   uint64 // memory.max value in bytes
	Unit    string // original unit (e.g. "1G", "100M", "1T") – may be empty if no suffix
	Raw     string // raw input string
	Default bool   // true if the input was "max" (no limit)
}

// ResourceAgentPidsLimit_001 holds a parsed cgroup v2 pids limit.
type ResourceAgentPidsLimit_001 struct {
	Max     int32  // pids.max value (max is the number of pids)
	Raw     string // raw input string
	Default bool   // true if input was "max" (no limit)
}

// ResourceAgentCpuLimit_001 holds parsed cgroup v2 CPU limits from cpu.max and cpu.weight.
type ResourceAgentCpuLimit_001 struct {
	Period uint64 // in microseconds, from cpu.max
	Quota  uint64 // in microseconds, from cpu.max; 0 if "max"
	Weight uint64 // cpu.weight value (1-10000)
	Raw    string // raw input string for cpu.max
}

// memoryUnitsTable_001 is a deterministic table mapping known memory suffixes to bytes factor.
var memoryUnitsTable_001 = []struct {
	Suffix string
	Factor uint64
}{
	{"B", 1},
	{"", 1}, // no suffix means bytes
	{"K", 1024},
	{"KB", 1024},
	{"KiB", 1024},
	{"M", 1024 * 1024},
	{"MB", 1024 * 1024},
	{"MiB", 1024 * 1024},
	{"G", 1024 * 1024 * 1024},
	{"GB", 1024 * 1024 * 1024},
	{"GiB", 1024 * 1024 * 1024},
	{"T", 1024 * 1024 * 1024 * 1024},
	{"TB", 1024 * 1024 * 1024 * 1024},
	{"TiB", 1024 * 1024 * 1024 * 1024},
	{"P", 1024 * 1024 * 1024 * 1024 * 1024},
	{"PB", 1024 * 1024 * 1024 * 1024 * 1024},
	{"PiB", 1024 * 1024 * 1024 * 1024 * 1024},
}

// memoryValidation_001 provides constraints for memory limits.
// All values are in bytes. Use 0 for no limit (max).
var memoryValidation_001 = struct {
	MinBytes uint64
	MaxBytes uint64
}{
	MinBytes: 4096,             // 4 KB minimum
	MaxBytes: math.MaxUint64,  // no practical maximum
}

// pidsValidation_001 provides constraints for pids limits.
var pidsValidation_001 = struct {
	MinPids int32
	MaxPids int32
}{
	MinPids: 1,
	MaxPids: math.MaxInt32,
}

// cpuValidation_001 provides constraints for CPU limits.
var cpuValidation_001 = struct {
	MinPeriod uint64
	MaxPeriod uint64
	MinQuota  uint64
	MaxQuota  uint64
	MinWeight uint64
	MaxWeight uint64
}{
	MinPeriod: 1000,           // 1 ms minimum
	MaxPeriod: 1000000,        // 1 sec maximum (typical kernel limit)
	MinQuota:  0,              // 0 means "max"
	MaxQuota:  1000000,        // same as max period
	MinWeight: 1,
	MaxWeight: 10000,
}

// ParseMemoryLimit_001 parses a cgroup v2 memory limit string (e.g. "1G", "max", "536870912").
// Returns an error if the string is invalid or the value is out of bounds.
func ParseMemoryLimit_001(raw string) (ResourceAgentMemoryLimit_001, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ResourceAgentMemoryLimit_001{}, errors.New("memory limit string is empty")
	}
	// "max" means no limit
	if strings.EqualFold(raw, "max") {
		return ResourceAgentMemoryLimit_001{Default: true, Raw: raw}, nil
	}
	// Separate numeric part and optional suffix.
	// Suffix can be letters only; numeric part can include decimal point? cgroup v2 expects integer bytes.
	// We'll support simple integer with optional suffix (case-insensitive).
	suffix := extractSuffix_001(raw)
	numStr := raw[:len(raw)-len(suffix)]
	if numStr == "" {
		return ResourceAgentMemoryLimit_001{}, fmt.Errorf("memory limit %q: no numeric value", raw)
	}
	value, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return ResourceAgentMemoryLimit_001{}, fmt.Errorf("memory limit %q: invalid numeric part: %w", raw, err)
	}
	factor := uint64(1)
	if suffix != "" {
		found := false
		for _, entry := range memoryUnitsTable_001 {
			if strings.EqualFold(entry.Suffix, suffix) {
				factor = entry.Factor
				found = true
				break
			}
		}
		if !found {
			return ResourceAgentMemoryLimit_001{}, fmt.Errorf("memory limit %q: unknown suffix %q", raw, suffix)
		}
	}
	// Check overflow
	if value > math.MaxUint64/factor {
		return ResourceAgentMemoryLimit_001{}, fmt.Errorf("memory limit %q: value too large after scaling", raw)
	}
	bytes := value * factor
	if bytes < memoryValidation_001.MinBytes {
		return ResourceAgentMemoryLimit_001{}, fmt.Errorf("memory limit %q: value %d bytes is below minimum %d", raw, bytes, memoryValidation_001.MinBytes)
	}
	return ResourceAgentMemoryLimit_001{
		Bytes: bytes,
		Unit:  suffix,
		Raw:   raw,
	}, nil
}

// ParsePidsLimit_001 parses a cgroup v2 pids limit string (e.g. "100", "max").
func ParsePidsLimit_001(raw string) (ResourceAgentPidsLimit_001, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ResourceAgentPidsLimit_001{}, errors.New("pids limit string is empty")
	}
	if strings.EqualFold(raw, "max") {
		return ResourceAgentPidsLimit_001{Default: true, Raw: raw}, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return ResourceAgentPidsLimit_001{}, fmt.Errorf("pids limit %q: invalid number: %w", raw, err)
	}
	if value < int64(pidsValidation_001.MinPids) || value > int64(pidsValidation_001.MaxPids) {
		return ResourceAgentPidsLimit_001{}, fmt.Errorf("pids limit %q: out of range [%d, %d]", raw, pidsValidation_001.MinPids, pidsValidation_001.MaxPids)
	}
	return ResourceAgentPidsLimit_001{
		Max: int32(value),
		Raw: raw,
	}, nil
}

// ParseCpuLimit_001 parses a cgroup v2 cpu.max string (e.g. "50000 100000" or "max 100000").
// Also accepts a separate weight string. The raw parameter is the cpu.max value.
// Returns the parsed limits. Weight handling is separate.
func ParseCpuLimit_001(raw string, weightRaw string) (ResourceAgentCpuLimit_001, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ResourceAgentCpuLimit_001{}, errors.New("cpu.max string is empty")
	}
	// Split by space: first quota, second period
	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.max %q: expected two fields (quota period)", raw)
	}
	quotaStr := parts[0]
	periodStr := parts[1]
	// Parse period
	period, err := strconv.ParseUint(periodStr, 10, 64)
	if err != nil {
		return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.max %q: invalid period: %w", raw, err)
	}
	if period < cpuValidation_001.MinPeriod || period > cpuValidation_001.MaxPeriod {
		return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.max %q: period %d out of range [%d, %d]", raw, period, cpuValidation_001.MinPeriod, cpuValidation_001.MaxPeriod)
	}
	// Parse quota – could be "max"
	var quota uint64
	if strings.EqualFold(quotaStr, "max") {
		quota = 0
	} else {
		quota, err = strconv.ParseUint(quotaStr, 10, 64)
		if err != nil {
			return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.max %q: invalid quota: %w", raw, err)
		}
		if quota > cpuValidation_001.MaxQuota {
			return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.max %q: quota %d exceeds max %d", raw, quota, cpuValidation_001.MaxQuota)
		}
	}
	// Parse weight (cpu.weight, optional, default 100)
	weight := uint64(100)
	weightRaw = strings.TrimSpace(weightRaw)
	if weightRaw != "" {
		w, err := strconv.ParseUint(weightRaw, 10, 64)
		if err != nil {
			return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.weight %q: invalid: %w", weightRaw, err)
		}
		if w < cpuValidation_001.MinWeight || w > cpuValidation_001.MaxWeight {
			return ResourceAgentCpuLimit_001{}, fmt.Errorf("cpu.weight %q: out of range [%d, %d]", weightRaw, cpuValidation_001.MinWeight, cpuValidation_001.MaxWeight)
		}
		weight = w
	}
	return ResourceAgentCpuLimit_001{
		Period: period,
		Quota:  quota,
		Weight: weight,
		Raw:    raw,
	}, nil
}

// ValidateMemoryLimit_001 validates a memory limit string and returns an error if it doesn't meet constraints.
func ValidateMemoryLimit_001(raw string) error {
	_, err := ParseMemoryLimit_001(raw)
	return err
}

// ValidatePidsLimit_001 validates a pids limit string.
func ValidatePidsLimit_001(raw string) error {
	_, err := ParsePidsLimit_001(raw)
	return err
}

// ValidateCpuLimit_001 validates cpu.max and cpu.weight strings together.
func ValidateCpuLimit_001(cpuMaxRaw, cpuWeightRaw string) error {
	_, err := ParseCpuLimit_001(cpuMaxRaw, cpuWeightRaw)
	return err
}

// ValidateResourceLimits_001 validates a map of cgroup v2 limit files to their raw string values.
// Supported keys: "memory.max", "pids.max", "cpu.max", "cpu.weight".
// Returns a combined error if any fail.
func ValidateResourceLimits_001(limits map[string]string) error {
	var errs []string
	for key, value := range limits {
		switch key {
		case "memory.max":
			if e := ValidateMemoryLimit_001(value); e != nil {
				errs = append(errs, fmt.Sprintf("memory.max: %v", e))
			}
		case "pids.max":
			if e := ValidatePidsLimit_001(value); e != nil {
				errs = append(errs, fmt.Sprintf("pids.max: %v", e))
			}
		case "cpu.max":
			// weight may come from separate key; default to empty
			weight := limits["cpu.weight"]
			if e := ValidateCpuLimit_001(value, weight); e != nil {
				errs = append(errs, fmt.Sprintf("cpu.max: %v", e))
			}
		case "cpu.weight":
			// independently validated later when cpu.max is processed
		default:
			errs = append(errs, fmt.Sprintf("unknown limit key %q", key))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// extractSuffix_001 extracts the alphabetic suffix from a memory limit string.
// Returns either the suffix (if any) or empty string.
func extractSuffix_001(s string) string {
	// Find the first non-digit character (or decimal point if we allowed floats). We allow only digits and decimal? For simplicity, we look for trailing letters.
	runes := []rune(s)
	i := len(runes) - 1
	for i >= 0 && isLetter_001(runes[i]) {
		i--
	}
	if i == len(runes)-1 {
		return ""
	}
	return string(runes[i+1:])
}

// isLetter_001 returns true if the rune is a letter (a-z, A-Z).
func isLetter_001(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// FormatMemoryLimit_001 formats a memory limit value in bytes to a string.
// Uses the specified suffix (e.g., "M", "G"). If suffix is empty, returns raw bytes.
func FormatMemoryLimit_001(bytes uint64, suffix string) string {
	factor := uint64(1)
	if suffix != "" {
		for _, entry := range memoryUnitsTable_001 {
			if strings.EqualFold(entry.Suffix, suffix) {
				factor = entry.Factor
				break
			}
		}
	}
	if factor == 1 {
		return strconv.FormatUint(bytes, 10)
	}
	val := bytes / factor
	if bytes%factor != 0 {
		// Return as-is with decimal? For simplicity, we truncate.
	}
	return strconv.FormatUint(val, 10) + strings.ToUpper(suffix)
}

// FormatPidsLimit_001 formats a pids limit to string.
func FormatPidsLimit_001(pids ResourceAgentPidsLimit_001) string {
	if pids.Default {
		return "max"
	}
	return strconv.FormatInt(int64(pids.Max), 10)
}

// FormatCpuLimit_001 formats cpu limits as a string (period and quota), with optional weight.
func FormatCpuLimit_001(limit ResourceAgentCpuLimit_001) string {
	quotaStr := strconv.FormatUint(limit.Quota, 10)
	if limit.Quota == 0 {
		quotaStr = "max"
	}
	return quotaStr + " " + strconv.FormatUint(limit.Period, 10)
}

// NormalizeMemoryLimit_001 normalizes a memory limit string to a canonical form.
// Returns the normalized string or error.
func NormalizeMemoryLimit_001(raw string) (string, error) {
	parsed, err := ParseMemoryLimit_001(raw)
	if err != nil {
		return "", err
	}
	if parsed.Default {
		return "max", nil
	}
	// Choose a sensible unit: if bytes < 1024, use B; else if < 1024*1024 use K; else if < 1024*1024*1024 use M; else use G.
	var suffix string
	switch {
	case parsed.Bytes < 1024:
		suffix = "B"
	case parsed.Bytes < 1024*1024:
		suffix = "K"
	case parsed.Bytes < 1024*1024*1024:
		suffix = "M"
	default:
		suffix = "G"
	}
	return FormatMemoryLimit_001(parsed.Bytes, suffix), nil
}

// NormalizePidsLimit_001 normalizes a pids limit string.
func NormalizePidsLimit_001(raw string) (string, error) {
	parsed, err := ParsePidsLimit_001(raw)
	if err != nil {
		return "", err
	}
	return FormatPidsLimit_001(parsed), nil
}

// NormalizeCpuLimit_001 normalizes cpu.max and weight strings.
func NormalizeCpuLimit_001(cpuMaxRaw, cpuWeightRaw string) (string, error) {
	parsed, err := ParseCpuLimit_001(cpuMaxRaw, cpuWeightRaw)
	if err != nil {
		return "", err
	}
	return FormatCpuLimit_001(parsed), nil
}

// ResourceAgentLimitsTable_001 provides a deterministic table for testing and documentation
// of valid and invalid cgroup v2 limits.
type ResourceAgentLimitsTableEntry_001 struct {
	Label      string
	MemoryMax  string
	PidsMax    string
	CPUMax     string
	CPUWeight  string
	Valid      bool
	ErrorMsg   string // if invalid, expected error substring
}

// ResourceAgentLimitsTable_001 is a table of test cases for limit validation.
var ResourceAgentLimitsTable_001 = []ResourceAgentLimitsTableEntry_001{
	{
		Label:     "valid memory max",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     true,
	},
	{
		Label:     "valid memory no limit",
		MemoryMax: "max",
		PidsMax:   "max",
		CPUMax:    "max 100000",
		CPUWeight: "",
		Valid:     true,
	},
	{
		Label:     "valid minimal memory",
		MemoryMax: "4096",
		PidsMax:   "1",
		CPUMax:    "1000 1000000",
		CPUWeight: "1",
		Valid:     true,
	},
	{
		Label:     "valid memory with suffix KB",
		MemoryMax: "100KB",
		PidsMax:   "500",
		CPUMax:    "25000 50000",
		CPUWeight: "500",
		Valid:     true,
	},
	{
		Label:     "valid memory with decimal? not supported: no decimal allowed",
		MemoryMax: "1.5G", // will be invalid because we use ParseUint
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "invalid numeric part",
	},
	{
		Label:     "invalid memory suffix",
		MemoryMax: "100X",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "unknown suffix",
	},
	{
		Label:     "invalid memory below min",
		MemoryMax: "100",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "below minimum",
	},
	{
		Label:     "invalid pids negative",
		MemoryMax: "1G",
		PidsMax:   "-5",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "invalid number",
	},
	{
		Label:     "invalid pids zero",
		MemoryMax: "1G",
		PidsMax:   "0",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "out of range",
	},
	{
		Label:     "invalid cpu.max one field",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "expected two fields",
	},
	{
		Label:     "invalid cpu.max period too low",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000 999",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "period 999 out of range",
	},
	{
		Label:     "invalid cpu.max quota too high",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "2000000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "quota 2000000 exceeds max",
	},
	{
		Label:     "invalid cpu.weight above 10000",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "10001",
		Valid:     false,
		ErrorMsg:  "out of range",
	},
	{
		Label:     "valid cpu.weight zero? not allowed",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "0",
		Valid:     false,
		ErrorMsg:  "out of range",
	},
	{
		Label:     "valid empty weight defaults to 100",
		MemoryMax: "1G",
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "",
		Valid:     true,
	},
	{
		Label:     "valid memory with TiB suffix",
		MemoryMax: "1TiB",
		PidsMax:   "1000",
		CPUMax:    "100000 1000000",
		CPUWeight: "100",
		Valid:     true,
	},
	{
		Label:     "invalid memory overflow",
		MemoryMax: "18446744073709551616", // > max uint64
		PidsMax:   "100",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     false,
		ErrorMsg:  "value too large",
	},
	{
		Label:     "valid memory with no suffix",
		MemoryMax: "536870912",
		PidsMax:   "200",
		CPUMax:    "50000 100000",
		CPUWeight: "100",
		Valid:     true,
	},
}

// RunLimitsTableValidation_001 runs through the deterministic table and returns an error if any entry
// does not match expected validity. This is used for testing and validation.
func RunLimitsTableValidation_001() error {
	for _, entry := range ResourceAgentLimitsTable_001 {
		err := ValidateResourceLimits_001(map[string]string{
			"memory.max": entry.MemoryMax,
			"pids.max":   entry.PidsMax,
			"cpu.max":    entry.CPUMax,
			"cpu.weight": entry.CPUWeight,
		})
		if entry.Valid && err != nil {
			return fmt.Errorf("entry %q expected valid, got error: %w", entry.Label, err)
		}
		if !entry.Valid && err == nil {
			return fmt.Errorf("entry %q expected invalid (error contains %q), but got no error", entry.Label, entry.ErrorMsg)
		}
		if !entry.Valid && err != nil && !strings.Contains(err.Error(), entry.ErrorMsg) {
			return fmt.Errorf("entry %q expected error containing %q, got %v", entry.Label, entry.ErrorMsg, err)
		}
	}
	return nil
}
