package chunk031

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Constants and zero values for memory event keys
// ---------------------------------------------------------------------------
const (
	MemoryEventKeyOom     = "oom"
	MemoryEventKeyOomKill = "oom_kill"

	MemoryEventFile = "memory.events"
)

// ---------------------------------------------------------------------------
// MemoryEventData holds parsed counts from a memory.events file
// ---------------------------------------------------------------------------
type MemoryEventData struct {
	Oom     uint64 `json:"oom"`
	OomKill uint64 `json:"oom_kill"`
	// additional common keys might be present, store as Extra
	Extra map[string]uint64 `json:"extra,omitempty"`
}

// MemoryEventData020 is alias for backward compatibility / file index.
type MemoryEventData020 = MemoryEventData

// ResourceAgentMemoryEventData is exported with prefix.
type ResourceAgentMemoryEventData struct {
	MemoryEventData
	Source     string    `json:"source,omitempty"`
	ParsedAt   time.Time `json:"parsed_at,omitempty"`
	Err        error     `json:"-"`
}

// ---------------------------------------------------------------------------
// OomEvent represents a single OOM kill event (e.g. from dmesg or cgroupv2).
// ---------------------------------------------------------------------------
type OomEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	PID         int       `json:"pid"`
	ProcessName string    `json:"process_name"`
	MemoryCGroup string   `json:"memory_cgroup,omitempty"`
	OOMScore     int       `json:"oom_score,omitempty"`
	Killed       bool      `json:"killed"`
	Slab         uint64    `json:"slab,omitempty"`
	RSS          uint64    `json:"rss,omitempty"`
	Swap         uint64    `json:"swap,omitempty"`
}

// OomEvent020 is alias.
type OomEvent020 = OomEvent

// ResourceAgentOomEvent is exported.
type ResourceAgentOomEvent struct {
	OomEvent
	Source string `json:"source,omitempty"`
}

// ---------------------------------------------------------------------------
// EvidenceRecord is a generic record for an observed resource event.
// Here we re-export with prefix for consistency.
// ---------------------------------------------------------------------------
type EvidenceRecord struct {
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Details   interface{} `json:"details,omitempty"`
	Tags      []string    `json:"tags,omitempty"`
}

// ResourceAgentEvidenceRecord is the exported version.
type ResourceAgentEvidenceRecord struct {
	EvidenceRecord
	CollectedBy string `json:"collected_by,omitempty"`
}

// ---------------------------------------------------------------------------
// Validate* functions
// ---------------------------------------------------------------------------

// ValidateMemoryEventsData_020 checks counts and extra fields.
func ValidateMemoryEventsData_020(data *MemoryEventData) error {
	if data == nil {
		return errors.New("memory event data is nil")
	}
	if data.Oom > math.MaxUint64 {
		return errors.New("oom count overflow")
	}
	if data.OomKill > math.MaxUint64 {
		return errors.New("oom_kill count overflow")
	}
	if data.OomKill > data.Oom {
		return fmt.Errorf("oom_kill count (%d) > oom count (%d)", data.OomKill, data.Oom)
	}
	for key, val := range data.Extra {
		if key == "" {
			return errors.New("extra key cannot be empty")
		}
		if val > math.MaxUint64 {
			return fmt.Errorf("extra field %s overflows", key)
		}
	}
	return nil
}

// ValidateOomEvent_020 validates a single OOM event.
func ValidateOomEvent_020(event *OomEvent) error {
	if event == nil {
		return errors.New("oom event is nil")
	}
	if event.Timestamp.IsZero() {
		return errors.New("oom event timestamp is zero")
	}
	if event.PID <= 0 {
		return fmt.Errorf("invalid PID %d", event.PID)
	}
	if event.ProcessName == "" {
		return errors.New("oom event process name is empty")
	}
	if event.Killed && event.OOMScore == 0 {
		// not necessarily an error, but warn? just allow.
	}
	return nil
}

// ---------------------------------------------------------------------------
// Parsing functions
// ---------------------------------------------------------------------------

// ParseMemoryEventsFromLines_020 parses lines from a memory.events file.
func ParseMemoryEventsFromLines_020(lines []string) (*MemoryEventData, error) {
	if len(lines) == 0 {
		return nil, errors.New("empty lines")
	}
	data := &MemoryEventData{
		Extra: make(map[string]uint64),
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid memory.events line: %q", line)
		}
		key := parts[0]
		valStr := strings.TrimSpace(parts[1])
		val, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse value for key %q: %w", key, err)
		}
		switch key {
		case MemoryEventKeyOom:
			data.Oom = val
		case MemoryEventKeyOomKill:
			data.OomKill = val
		default:
			data.Extra[key] = val
		}
	}
	// If oom_kill missing but we have oom>0, may set default.
	if _, hasOomKill := data.Extra[MemoryEventKeyOomKill]; !hasOomKill && data.Oom > 0 {
		// Not an error, just note.
	}
	return data, nil
}

// ParseMemoryEventsFromReader_020 reads from an io.Reader.
func ParseMemoryEventsFromReader_020(r *bufio.Reader) (*MemoryEventData, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan memory.events: %w", err)
	}
	return ParseMemoryEventsFromLines_020(lines)
}

// ParseOomEventLine_020 parses a single line from dmesg or journalctl.
// Expected format: "oom-kill: ..." or similar. This is simplistic.
func ParseOomEventLine_020(line string) (*OomEvent, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, errors.New("empty line")
	}
	event := &OomEvent{
		Timestamp: time.Now(), // fallback
		Killed:    true,
	}
	// Very basic parsing: look for "pid=<num>" and "comm=<name>".
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "oom-kill") && !strings.Contains(lower, "oom_kill") && !strings.Contains(lower, MemoryEventKeyOomKill) {
		return nil, errors.New("not an OOM kill line")
	}
	// Attempt to extract pid and comm
	pidPos := strings.Index(lower, "pid=")
	if pidPos >= 0 {
		rest := line[pidPos+4:]
		end := strings.IndexAny(rest, ", ")
		if end > 0 {
			pidStr := rest[:end]
			if pid, err := strconv.Atoi(pidStr); err == nil {
				event.PID = pid
			}
		}
	}
	commPos := strings.Index(lower, "comm=")
	if commPos >= 0 {
		rest := line[commPos+5:]
		// comm may be quoted
		if len(rest) > 0 && rest[0] == '"' {
			rest = rest[1:]
			if idx := strings.IndexByte(rest, '"'); idx >= 0 {
				event.ProcessName = rest[:idx]
			}
		} else {
			end := strings.IndexAny(rest, ", ")
			if end > 0 {
				event.ProcessName = rest[:end]
			} else {
				event.ProcessName = rest
			}
		}
	}
	// Try timestamp from beginning (kernel format)
	// e.g. "May 15 10:30:45 kernel: ..."
	if len(line) > 20 {
		// crude: assume first 15 chars are timestamp
		short := strings.Fields(line)
		if len(short) >= 3 {
			dateStr := strings.Join(short[:3], " ")
			if t, err := time.Parse("Jan 2 15:04:05", dateStr); err == nil {
				event.Timestamp = t
			} else if t, err = time.Parse("2006-01-02T15:04:05", short[0]); err == nil {
				event.Timestamp = t
			}
		}
	}
	return event, nil
}

// ---------------------------------------------------------------------------
// Evidence record creation
// ---------------------------------------------------------------------------

// NewOomEvidenceRecord_020 creates an evidence record from an OomEvent.
func NewOomEvidenceRecord_020(event *OomEvent, collectedBy string, tags ...string) *ResourceAgentEvidenceRecord {
	if event == nil {
		return nil
	}
	rec := &ResourceAgentEvidenceRecord{
		EvidenceRecord: EvidenceRecord{
			EventType: "oom_kill",
			Timestamp: event.Timestamp,
			Details:   event,
			Tags:      append([]string(nil), tags...),
		},
		CollectedBy: collectedBy,
	}
	if rec.Tags == nil {
		rec.Tags = []string{}
	}
	return rec
}

// NewMemoryEventEvidenceRecord_020 creates an evidence record from MemoryEventData.
func NewMemoryEventEvidenceRecord_020(data *MemoryEventData, collectedBy string, tags ...string) *ResourceAgentEvidenceRecord {
	if data == nil {
		return nil
	}
	rec := &ResourceAgentEvidenceRecord{
		EvidenceRecord: EvidenceRecord{
			EventType: "memory_event_counts",
			Timestamp: time.Now(),
			Details:   data,
			Tags:      append([]string(nil), tags...),
		},
		CollectedBy: collectedBy,
	}
	if rec.Tags == nil {
		rec.Tags = []string{}
	}
	return rec
}

// ---------------------------------------------------------------------------
// Table-driven helpers (deterministic)
// ---------------------------------------------------------------------------

// MemoryEventsTestCases_020 returns test cases for parsing and validation.
// Each entry has input lines, expected data, expected error (nil if ok).
func MemoryEventsTestCases_020() []struct {
	Name     string
	Lines    []string
	Expected *MemoryEventData
	ExpErr   error
} {
	return []struct {
		Name     string
		Lines    []string
		Expected *MemoryEventData
		ExpErr   error
	}{
		{
			Name: "happy_path",
			Lines: []string{
				"oom 100",
				"oom_kill 2",
			},
			Expected: &MemoryEventData{
				Oom:     100,
				OomKill: 2,
				Extra:   map[string]uint64{},
			},
			ExpErr: nil,
		},
		{
			Name: "extra_fields",
			Lines: []string{
				"oom 0",
				"oom_kill 0",
				"pgfault 12345",
				"pgmajfault 0",
			},
			Expected: &MemoryEventData{
				Oom:     0,
				OomKill: 0,
				Extra: map[string]uint64{
					"pgfault":   12345,
					"pgmajfault": 0,
				},
			},
			ExpErr: nil,
		},
		{
			Name: "invalid_line_no_space",
			Lines: []string{
				"oom 42",
				"invalid",
			},
			Expected: nil,
			ExpErr:   fmt.Errorf("invalid memory.events line: %q", "invalid"),
		},
		{
			Name: "non_numeric_value",
			Lines: []string{
				"oom abc",
			},
			Expected: nil,
			ExpErr:   fmt.Errorf("parse value for key \"oom\": strconv.ParseUint: parsing \"abc\": invalid syntax"),
		},
		{
			Name:   "empty_lines",
			Lines:  []string{},
			Expected: nil,
			ExpErr:   errors.New("empty lines"),
		},
		{
			Name: "oom_kill_gt_oom",
			Lines: []string{
				"oom 1",
				"oom_kill 5",
			},
			Expected: &MemoryEventData{
				Oom:     1,
				OomKill: 5,
				Extra:   map[string]uint64{},
			},
			ExpErr: nil, // parsing succeeds, validation later
		},
	}
}

// OomEventTestCases_020 returns deterministic test cases for OomEvent parsing.
func OomEventTestCases_020() []struct {
	Name       string
	InputLine  string
	Expected   *OomEvent
	ExpErr     error
} {
	return []struct {
		Name       string
		InputLine  string
		Expected   *OomEvent
		ExpErr     error
	}{
		{
			Name:      "basic_oom_kill_line",
			InputLine: "May 15 10:30:45 kernel: oom-kill: gfp_mask=0x... pid=12345 comm=\"myapp\"",
			Expected: &OomEvent{
				Timestamp:   time.Date(0, time.May, 15, 10, 30, 45, 0, time.UTC),
				PID:         12345,
				ProcessName: "myapp",
				Killed:      true,
			},
			ExpErr: nil,
		},
		{
			Name:      "not_oom_line",
			InputLine: "some random log",
			Expected:   nil,
			ExpErr:     errors.New("not an OOM kill line"),
		},
		{
			Name:      "empty_line",
			InputLine: "",
			Expected:   nil,
			ExpErr:     errors.New("empty line"),
		},
		{
			Name:      "realistic_journal_line",
			InputLine: "2025-01-15T14:32:10 node1 kernel: oom-kill: process pid=999 comm=\"java\"",
			Expected: &OomEvent{
				Timestamp:   time.Date(2025, time.January, 15, 14, 32, 10, 0, time.UTC),
				PID:         999,
				ProcessName: "java",
				Killed:      true,
			},
			ExpErr: nil,
		},
	}
}

// KnownOomEventsRegistry_020 returns a sorted slice of well-known OOM event
// patterns (substrings) that can be used for matching.
func KnownOomEventsRegistry_020() []string {
	// Deterministic list.
	patterns := []string{
		"oom-kill",
		"oom_kill",
		MemoryEventKeyOomKill,
		"Out of memory",
		"Killed process",
		"invoked oom",
	}
	sort.Strings(patterns)
	return patterns
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

// FormatOomEvent_020 returns a human-readable string.
func FormatOomEvent_020(e *OomEvent) string {
	if e == nil {
		return "<nil>"
	}
	ts := e.Timestamp.Format(time.RFC3339)
	return fmt.Sprintf("[%s] OOM kill: PID %d (%s) killed=%v score=%d",
		ts, e.PID, e.ProcessName, e.Killed, e.OOMScore)
}

// FormatMemoryEventData_020 returns a simple string.
func FormatMemoryEventData_020(d *MemoryEventData) string {
	if d == nil {
		return "<nil>"
	}
	return fmt.Sprintf("oom=%d oom_kill=%d extra=%v", d.Oom, d.OomKill, len(d.Extra))
}

// ---------------------------------------------------------------------------
// Example usage / init (no-op)
// ---------------------------------------------------------------------------

func init() {
	// Ensure the OomEvent020 type is used somewhere to avoid deadcode.
	_ = OomEvent020{}
	_ = MemoryEventData020{}
}
