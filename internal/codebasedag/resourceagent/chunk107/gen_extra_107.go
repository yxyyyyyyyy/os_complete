package chunk107

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// DefaultCPUStatPath107 is a sensible default path for cpu.stat (cgroup v2).
const DefaultCPUStatPath107 = "/sys/fs/cgroup/cpu.stat"

// Field names expected in cpu.stat files.
const (
	FieldUsageUsec107       = "usage_usec"
	FieldUserUsec107        = "user_usec"
	FieldSystemUsec107      = "system_usec"
	FieldNrPeriods107       = "nr_periods"
	FieldNrThrottled107     = "nr_throttled"
	FieldThrottledUsec107   = "throttled_usec"
	FieldNrBurstPeriods107  = "nr_burst_periods"
	FieldNrBurstThrottled107 = "nr_burst_throttled"
	FieldBurstUsec107       = "burst_usec"
)

// Default thresholds (as ratios) for warnings and errors.
const (
	ThrottleRatioWarn107  = 0.05  // 5%
	ThrottleRatioErr107   = 0.20  // 20%
	ThrottleRatioCrit107  = 0.50  // 50%
)

// MaxHistorySize107 defines the default number of samples kept in a throttle history.
const MaxHistorySize107 = 100

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// CPUStat107 represents the parsed fields of a cgroup v2 cpu.stat file.
type CPUStat107 struct {
	UsageUsec       uint64 `json:"usage_usec"`
	UserUsec        uint64 `json:"user_usec"`
	SystemUsec      uint64 `json:"system_usec"`
	NrPeriods       uint64 `json:"nr_periods"`
	NrThrottled     uint64 `json:"nr_throttled"`
	ThrottledUsec   uint64 `json:"throttled_usec"`
	NrBurstPeriods  uint64 `json:"nr_burst_periods"`
	NrBurstThrottled uint64 `json:"nr_burst_throttled"`
	BurstUsec       uint64 `json:"burst_usec"`
	RawFields       map[string]uint64 `json:"raw_fields"` // any extra fields
}

// ThrottleRatioResult107 holds the computed throttle ratios and derived values.
type ThrottleRatioResult107 struct {
	PeriodRatio      float64 `json:"period_ratio"`      // nr_throttled / nr_periods
	TimeRatio        float64 `json:"time_ratio"`        // throttled_usec / usage_usec
	TimePercent      float64 `json:"time_percent"`
	PeriodPercent    float64 `json:"period_percent"`
	BurstPeriodRatio float64 `json:"burst_period_ratio"`
	BurstTimeRatio   float64 `json:"burst_time_ratio"`
	Sample           CPUStat107 `json:"sample"`
	Timestamp        time.Time `json:"timestamp"`
}

// ThrottleRatioValidator107 validates a throttle ratio against thresholds.
type ThrottleRatioValidator107 struct {
	WarnThreshold  float64
	ErrorThreshold float64
	CritThreshold  float64
}

// ThrottleHistory107 maintains a sliding window of recent throttle ratio samples.
type ThrottleHistory107 struct {
	mu       sync.RWMutex
	samples  []ThrottleRatioResult107
	maxSize  int
}

// ThrottleTableEntry107 is a single row in a throttle ratio summary table.
type ThrottleTableEntry107 struct {
	Index        int
	Timestamp    string
	PeriodRatio  string
	TimeRatio    string
	PeriodPct    string
	TimePct      string
	NrPeriods    uint64
	NrThrottled  uint64
	ThrottledUsec uint64
	UsageUsec    uint64
}

// ThrottleTable107 holds a formatted table of throttle entries.
type ThrottleTable107 struct {
	Headers []string
	Rows    []ThrottleTableEntry107
}

// ---------------------------------------------------------------------------
// Parsing functions
// ---------------------------------------------------------------------------

// ParseCPUStat107 reads a cpu.stat file and returns a CPUStat107 struct.
func ParseCPUStat107(path string) (CPUStat107, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return CPUStat107{}, fmt.Errorf("read file %s: %w", path, err)
	}
	return ParseCPUStatFromString107(string(data))
}

// ParseCPUStatFromString107 parses the content of a cpu.stat file given as a string.
func ParseCPUStatFromString107(content string) (CPUStat107, error) {
	stat := CPUStat107{RawFields: make(map[string]uint64)}
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue // skip malformed lines
		}
		key := parts[0]
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return stat, fmt.Errorf("parse field %s: %w", key, err)
		}
		switch key {
		case FieldUsageUsec107:
			stat.UsageUsec = val
		case FieldUserUsec107:
			stat.UserUsec = val
		case FieldSystemUsec107:
			stat.SystemUsec = val
		case FieldNrPeriods107:
			stat.NrPeriods = val
		case FieldNrThrottled107:
			stat.NrThrottled = val
		case FieldThrottledUsec107:
			stat.ThrottledUsec = val
		case FieldNrBurstPeriods107:
			stat.NrBurstPeriods = val
		case FieldNrBurstThrottled107:
			stat.NrBurstThrottled = val
		case FieldBurstUsec107:
			stat.BurstUsec = val
		default:
			stat.RawFields[key] = val
		}
	}
	if err := scanner.Err(); err != nil {
		return stat, fmt.Errorf("scan: %w", err)
	}
	return stat, nil
}

// ParseCPUStatFromReader107 parses cpu.stat from an io.Reader.
func ParseCPUStatFromReader107(r io.Reader) (CPUStat107, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return CPUStat107{}, err
	}
	return ParseCPUStatFromString107(string(data))
}

// ---------------------------------------------------------------------------
// Ratio calculation functions
// ---------------------------------------------------------------------------

// ComputeThrottleRatios107 computes all throttle ratios from a CPUStat107 sample.
// Returns zero ratios when denominators are zero.
func ComputeThrottleRatios107(s CPUStat107) ThrottleRatioResult107 {
	res := ThrottleRatioResult107{
		Sample:    s,
		Timestamp: time.Now(),
	}
	if s.UsageUsec > 0 {
		res.TimeRatio = float64(s.ThrottledUsec) / float64(s.UsageUsec)
	}
	if s.NrPeriods > 0 {
		res.PeriodRatio = float64(s.NrThrottled) / float64(s.NrPeriods)
	}
	if s.NrBurstPeriods > 0 {
		res.BurstPeriodRatio = float64(s.NrBurstThrottled) / float64(s.NrBurstPeriods)
	}
	if s.NrBurstPeriods > 0 && s.BurstUsec > 0 {
		// burst time ratio: burst_usec / (burst_periods * expected slice?) – let's approximate
		res.BurstTimeRatio = float64(s.BurstUsec) / float64(s.NrBurstPeriods*100000) // roughly
	}
	res.TimePercent = res.TimeRatio * 100
	res.PeriodPercent = res.PeriodRatio * 100
	return res
}

// ThrottleRatio107 returns the time-based throttle ratio (throttled_usec / usage_usec).
func ThrottleRatio107(s CPUStat107) float64 {
	if s.UsageUsec == 0 {
		return 0
	}
	return float64(s.ThrottledUsec) / float64(s.UsageUsec)
}

// ThrottlePercent107 returns the throttle percentage (throttled_usec / usage_usec * 100).
func ThrottlePercent107(s CPUStat107) float64 {
	return ThrottleRatio107(s) * 100
}

// ThrottlePeriodRatio107 returns nr_throttled / nr_periods.
func ThrottlePeriodRatio107(s CPUStat107) float64 {
	if s.NrPeriods == 0 {
		return 0
	}
	return float64(s.NrThrottled) / float64(s.NrPeriods)
}

// ThrottlePeriodPercent107 returns the period-based throttle percentage.
func ThrottlePeriodPercent107(s CPUStat107) float64 {
	return ThrottlePeriodRatio107(s) * 100
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

// ValidateRatio107 checks that a ratio is between 0 and 1 (or NaN). Returns error if outside.
func ValidateRatio107(ratio float64) error {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return errors.New("ratio is NaN or Inf")
	}
	if ratio < 0 || ratio > 1 {
		return fmt.Errorf("ratio %f out of [0,1] range", ratio)
	}
	return nil
}

// ValidateThrottleRatio107 validates a throttle ratio against the validator's thresholds.
// Returns (level, message) where level is 0=OK, 1=WARN, 2=ERROR, 3=CRITICAL.
func (v ThrottleRatioValidator107) ValidateThrottleRatio107(ratio float64) (int, string) {
	if err := ValidateRatio107(ratio); err != nil {
		return 3, fmt.Sprintf("invalid ratio: %v", err)
	}
	switch {
	case ratio >= v.CritThreshold:
		return 3, fmt.Sprintf("CRITICAL: ratio %.4f >= %.4f", ratio, v.CritThreshold)
	case ratio >= v.ErrorThreshold:
		return 2, fmt.Sprintf("ERROR: ratio %.4f >= %.4f", ratio, v.ErrorThreshold)
	case ratio >= v.WarnThreshold:
		return 1, fmt.Sprintf("WARNING: ratio %.4f >= %.4f", ratio, v.WarnThreshold)
	default:
		return 0, "OK"
	}
}

// CheckThreshold107 is a convenience function that uses default thresholds.
func CheckThreshold107(ratio float64) (int, string) {
	v := ThrottleRatioValidator107{
		WarnThreshold:  ThrottleRatioWarn107,
		ErrorThreshold: ThrottleRatioErr107,
		CritThreshold:  ThrottleRatioCrit107,
	}
	return v.ValidateThrottleRatio107(ratio)
}

// ValidateCPUStat107 checks that the basic fields are non-negative and have valid relations.
func ValidateCPUStat107(s CPUStat107) error {
	if s.UsageUsec == 0 && s.ThrottledUsec > 0 {
		return errors.New("throttled_usec > 0 but usage_usec == 0")
	}
	if s.NrPeriods == 0 && s.NrThrottled > 0 {
		return errors.New("nr_throttled > 0 but nr_periods == 0")
	}
	if s.NrThrottled > s.NrPeriods {
		return fmt.Errorf("nr_throttled (%d) > nr_periods (%d)", s.NrThrottled, s.NrPeriods)
	}
	if s.ThrottledUsec > s.UsageUsec && s.UsageUsec > 0 {
		return fmt.Errorf("throttled_usec (%d) > usage_usec (%d)", s.ThrottledUsec, s.UsageUsec)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Table generation
// ---------------------------------------------------------------------------

// FormatTable107 returns a string representation of the throttle table.
func (t ThrottleTable107) FormatTable107() string {
	var b strings.Builder
	// Write headers
	b.WriteString(strings.Join(t.Headers, "\t"))
	b.WriteString("\n")
	for _, row := range t.Rows {
		b.WriteString(fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
			row.Index,
			row.Timestamp,
			row.PeriodRatio,
			row.TimeRatio,
			row.PeriodPct,
			row.TimePct,
			row.NrPeriods,
			row.NrThrottled,
			row.ThrottledUsec,
			row.UsageUsec,
		))
	}
	return b.String()
}

// GenerateThrottleReport107 creates a table from a slice of throttle ratio results.
func GenerateThrottleReport107(results []ThrottleRatioResult107) ThrottleTable107 {
	headers := []string{
		"Index", "Timestamp", "PeriodRatio", "TimeRatio",
		"PeriodPct", "TimePct", "NrPeriods", "NrThrottled",
		"ThrottledUsec", "UsageUsec",
	}
	rows := make([]ThrottleTableEntry107, len(results))
	for i, res := range results {
		rows[i] = ThrottleTableEntry107{
			Index:         i,
			Timestamp:     res.Timestamp.Format(time.RFC3339),
			PeriodRatio:   fmt.Sprintf("%.6f", res.PeriodRatio),
			TimeRatio:     fmt.Sprintf("%.6f", res.TimeRatio),
			PeriodPct:     fmt.Sprintf("%.2f%%", res.PeriodPercent),
			TimePct:       fmt.Sprintf("%.2f%%", res.TimePercent),
			NrPeriods:     res.Sample.NrPeriods,
			NrThrottled:   res.Sample.NrThrottled,
			ThrottledUsec: res.Sample.ThrottledUsec,
			UsageUsec:     res.Sample.UsageUsec,
		}
	}
	return ThrottleTable107{Headers: headers, Rows: rows}
}

// ThrottleSummary107 returns a one-line summary string for a single result.
func ThrottleSummary107(res ThrottleRatioResult107) string {
	level, msg := CheckThreshold107(res.TimeRatio)
	return fmt.Sprintf("[%s] %s: time=%.2f%% period=%.2f%% %s",
		res.Timestamp.Format(time.Stamp), levelName107(level),
		res.TimePercent, res.PeriodPercent, msg)
}

func levelName107(level int) string {
	switch level {
	case 0:
		return "OK"
	case 1:
		return "WARN"
	case 2:
		return "ERROR"
	case 3:
		return "CRIT"
	default:
		return "UNKN"
	}
}

// ---------------------------------------------------------------------------
// History / Aggregator
// ---------------------------------------------------------------------------

// NewThrottleHistory107 creates a new ThrottleHistory107 with given maximum size.
func NewThrottleHistory107(maxSize int) *ThrottleHistory107 {
	if maxSize <= 0 {
		maxSize = MaxHistorySize107
	}
	return &ThrottleHistory107{
		samples: make([]ThrottleRatioResult107, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add107 appends a sample to the history, removing oldest if at capacity.
func (h *ThrottleHistory107) Add107(sample ThrottleRatioResult107) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.samples) >= h.maxSize {
		h.samples = h.samples[1:]
	}
	h.samples = append(h.samples, sample)
}

// All107 returns a copy of all samples (newest last).
func (h *ThrottleHistory107) All107() []ThrottleRatioResult107 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]ThrottleRatioResult107, len(h.samples))
	copy(out, h.samples)
	return out
}

// Last107 returns the most recent sample, or zero value if empty.
func (h *ThrottleHistory107) Last107() ThrottleRatioResult107 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.samples) == 0 {
		return ThrottleRatioResult107{}
	}
	return h.samples[len(h.samples)-1]
}

// Average107 computes the average time ratio and period ratio over the history.
func (h *ThrottleHistory107) Average107() (float64, float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.samples) == 0 {
		return 0, 0
	}
	var sumTime, sumPeriod float64
	for _, s := range h.samples {
		sumTime += s.TimeRatio
		sumPeriod += s.PeriodRatio
	}
	n := float64(len(h.samples))
	return sumTime / n, sumPeriod / n
}

// MaxRatio107 returns the maximum time ratio and period ratio seen.
func (h *ThrottleHistory107) MaxRatio107() (float64, float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.samples) == 0 {
		return 0, 0
	}
	maxTime, maxPeriod := 0.0, 0.0
	for _, s := range h.samples {
		if s.TimeRatio > maxTime {
			maxTime = s.TimeRatio
		}
		if s.PeriodRatio > maxPeriod {
			maxPeriod = s.PeriodRatio
		}
	}
	return maxTime, maxPeriod
}

// Reset107 clears all samples.
func (h *ThrottleHistory107) Reset107() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.samples = h.samples[:0]
}

// ---------------------------------------------------------------------------
// Monitor / Polling
// ---------------------------------------------------------------------------

// ThrottleMonitor107 polls cpu.stat at regular intervals and calls a callback.
type ThrottleMonitor107 struct {
	Path     string
	Interval time.Duration
	History  *ThrottleHistory107
	stopCh   chan struct{}
	callback func(ThrottleRatioResult107)
}

// NewThrottleMonitor107 creates a monitor for cpu.stat polling.
func NewThrottleMonitor107(path string, interval time.Duration, callback func(ThrottleRatioResult107)) *ThrottleMonitor107 {
	return &ThrottleMonitor107{
		Path:     path,
		Interval: interval,
		History:  NewThrottleHistory107(MaxHistorySize107),
		stopCh:   make(chan struct{}),
		callback: callback,
	}
}

// Start107 begins polling in a goroutine. Returns immediately.
func (m *ThrottleMonitor107) Start107() {
	go func() {
		ticker := time.NewTicker(m.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s, err := ParseCPUStat107(m.Path)
				if err != nil {
					// silently ignore for now
					continue
				}
				res := ComputeThrottleRatios107(s)
				m.History.Add107(res)
				if m.callback != nil {
					m.callback(res)
				}
			case <-m.stopCh:
				return
			}
		}
	}()
}

// Stop107 signals the monitor goroutine to stop.
func (m *ThrottleMonitor107) Stop107() {
	close(m.stopCh)
}

// ---------------------------------------------------------------------------
// Utility / Helper functions
// ---------------------------------------------------------------------------

// ParseDuration107 parses a duration string from cpu.stat (microseconds suffix "us").
// Not currently used in cpu.stat, but provided for completeness.
func ParseDuration107(s string) (time.Duration, error) {
	if !strings.HasSuffix(s, "us") {
		return 0, fmt.Errorf("missing 'us' suffix: %s", s)
	}
	val := strings.TrimSuffix(s, "us")
	us, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(us) * time.Microsecond, nil
}

// MicrosecondsToString107 formats a uint64 microsecond value as human-readable string.
func MicrosecondsToString107(us uint64) string {
	d := time.Duration(us) * time.Microsecond
	switch {
	case d >= time.Hour:
		return d.Round(time.Second).String()
	case d >= time.Minute:
		return d.Round(time.Second).String()
	default:
		return d.Round(time.Millisecond).String()
	}
}

// FormatThrottleRatio107 returns a percentage string with two decimal places.
func FormatThrottleRatio107(ratio float64) string {
	return fmt.Sprintf("%.2f%%", ratio*100)
}

// diffCPUStat107 computes the delta between two CPUStat107 samples.
// Returns a new CPUStat107 where each field is the difference (b - a).
// Useful for computing intervals.
func diffCPUStat107(a, b CPUStat107) CPUStat107 {
	return CPUStat107{
		UsageUsec:       b.UsageUsec - a.UsageUsec,
		UserUsec:        b.UserUsec - a.UserUsec,
		SystemUsec:      b.SystemUsec - a.SystemUsec,
		NrPeriods:       b.NrPeriods - a.NrPeriods,
		NrThrottled:     b.NrThrottled - a.NrThrottled,
		ThrottledUsec:   b.ThrottledUsec - a.ThrottledUsec,
		NrBurstPeriods:  b.NrBurstPeriods - a.NrBurstPeriods,
		NrBurstThrottled: b.NrBurstThrottled - a.NrBurstThrottled,
		BurstUsec:       b.BurstUsec - a.BurstUsec,
	}
}

// IsHighThrottle107 returns true if the time ratio exceeds the given threshold.
func IsHighThrottle107(result ThrottleRatioResult107, threshold float64) bool {
	return result.TimeRatio > threshold
}

// SortByTimeRatio107 sorts a slice of results by time ratio ascending.
func SortByTimeRatio107(results []ThrottleRatioResult107) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].TimeRatio < results[j].TimeRatio
	})
}

// SortByTimestamp107 sorts results by timestamp ascending.
func SortByTimestamp107(results []ThrottleRatioResult107) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
}

// SummarizeResults107 returns a multi-line summary of multiple throttle results.
func SummarizeResults107(results []ThrottleRatioResult107) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total samples: %d\n", len(results)))
	if len(results) == 0 {
		return b.String()
	}
	var sumTime, sumPeriod float64
	maxTime, maxPeriod := 0.0, 0.0
	for _, r := range results {
		sumTime += r.TimeRatio
		sumPeriod += r.PeriodRatio
		if r.TimeRatio > maxTime {
			maxTime = r.TimeRatio
		}
		if r.PeriodRatio > maxPeriod {
			maxPeriod = r.PeriodRatio
		}
	}
	avgTime := sumTime / float64(len(results))
	avgPeriod := sumPeriod / float64(len(results))
	b.WriteString(fmt.Sprintf("Avg time ratio: %.4f (%.2f%%)\n", avgTime, avgTime*100))
	b.WriteString(fmt.Sprintf("Avg period ratio: %.4f (%.2f%%)\n", avgPeriod, avgPeriod*100))
	b.WriteString(fmt.Sprintf("Max time ratio: %.4f (%.2f%%)\n", maxTime, maxTime*100))
	b.WriteString(fmt.Sprintf("Max period ratio: %.4f (%.2f%%)\n", maxPeriod, maxPeriod*100))
	return b.String()
}

// ThrottleSeverity107 returns a severity string based on the ratio.
func ThrottleSeverity107(ratio float64) string {
	level, _ := CheckThreshold107(ratio)
	return levelName107(level)
}

// ---------------------------------------------------------------------------
// File helper
// ---------------------------------------------------------------------------

// WriteCPUStat107 writes a CPUStat107 struct as cpu.stat formatted text to a writer.
func WriteCPUStat107(w io.Writer, s CPUStat107) error {
	fields := []string{
		fmt.Sprintf("usage_usec %d", s.UsageUsec),
		fmt.Sprintf("user_usec %d", s.UserUsec),
		fmt.Sprintf("system_usec %d", s.SystemUsec),
		fmt.Sprintf("nr_periods %d", s.NrPeriods),
		fmt.Sprintf("nr_throttled %d", s.NrThrottled),
		fmt.Sprintf("throttled_usec %d", s.ThrottledUsec),
		fmt.Sprintf("nr_burst_periods %d", s.NrBurstPeriods),
		fmt.Sprintf("nr_burst_throttled %d", s.NrBurstThrottled),
		fmt.Sprintf("burst_usec %d", s.BurstUsec),
	}
	for k, v := range s.RawFields {
		fields = append(fields, fmt.Sprintf("%s %d", k, v))
	}
	_, err := fmt.Fprintln(w, strings.Join(fields, "\n"))
	return err
}

// ReadCPUStat107 wraps ParseCPUStat107 with an optional existence check.
func ReadCPUStat107(path string) (CPUStat107, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CPUStat107{}, fmt.Errorf("file not found: %s", path)
	}
	return ParseCPUStat107(path)
}

// ---------------------------------------------------------------------------
// Sample generator for testing (not exported – but we can export if needed)
// ---------------------------------------------------------------------------

// GenerateSampleCPUStat107 creates a dummy CPUStat107 for testing.
func GenerateSampleCPUStat107(usage, throttled uint64) CPUStat107 {
	periods := usage / 100000 // assume 100ms period
	if periods == 0 {
		periods = 1
	}
	throttledPeriods := uint64(float64(periods) * (float64(throttled) / float64(usage)))
	if throttledPeriods > periods {
		throttledPeriods = periods
	}
	return CPUStat107{
		UsageUsec:     usage,
		UserUsec:      usage * 7 / 10,
		SystemUsec:    usage * 3 / 10,
		NrPeriods:     periods,
		NrThrottled:   throttledPeriods,
		ThrottledUsec: throttled,
	}
}

// GenerateThrottleTrend107 creates a list of results simulating a trend over time.
func GenerateThrottleTrend107(base, step uint64, count int) []ThrottleRatioResult107 {
	results := make([]ThrottleRatioResult107, count)
	usage := uint64(10000000) // 10s
	throttled := base
	for i := 0; i < count; i++ {
		s := GenerateSampleCPUStat107(usage, throttled)
		results[i] = ComputeThrottleRatios107(s)
		throttled += step
	}
	return results
}

// Ensure all exported names have _107 suffix.
