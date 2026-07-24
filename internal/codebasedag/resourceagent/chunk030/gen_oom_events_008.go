package chunk030

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ResourceAgentMemoryEvents represents parsed memory.events cgroup file.
type ResourceAgentMemoryEvents struct {
	Oom      uint64 // Number of OOM events since cgroup creation
	OomKill  uint64 // Number of OOM kill events since cgroup creation
	OomGroup *uint64 // Optional: number of OOM group kills (if present)
	ParsedAt time.Time
	Path     string
}

// ResourceAgentOomEventRecord is an evidence record for a single OOM event.
type ResourceAgentOomEventRecord struct {
	ID          string // Unique identifier for the record
	Timestamp   time.Time
	CgroupPath  string
	EventType   string // "oom" or "oom_kill" or "oom_group"
	Count       uint64 // Cumulative count at time of event (not point-increment)
	Diagnostics string // Additional context (e.g., memory pressure info)
}

// OomEventParseResult holds the result of parsing memory.events.
type ResourceAgentOomEventParseResult struct {
	Events  ResourceAgentMemoryEvents
	Records []ResourceAgentOomEventRecord
}

// Supported event keys in memory.events
var supportedEventKeys_008 = []string{"oom", "oom_kill", "oom_group"}

// validateEventName_008 checks if a given key is a valid memory.events event name.
func validateEventName_008(key string) error {
	lower := strings.ToLower(key)
	for _, k := range supportedEventKeys_008 {
		if lower == k {
			return nil
		}
	}
	return fmt.Errorf("unknown memory event key: %q", key)
}

// ParseMemoryEventsFromReader_008 reads memory.events from an io.Reader and returns parsed events.
func ParseMemoryEventsFromReader_008(r io.Reader) (*ResourceAgentMemoryEvents, error) {
	if r == nil {
		return nil, errors.New("reader cannot be nil")
	}
	events := &ResourceAgentMemoryEvents{
		ParsedAt: time.Now().UTC(),
	}
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: expected '<key> <value>', got %q", lineNum, line)
		}
		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		if err := validateEventName_008(key); err != nil {
			return nil, fmt.Errorf("line %d: %v", lineNum, err)
		}
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid count %q: %v", lineNum, valueStr, err)
		}
		switch key {
		case "oom":
			events.Oom = value
		case "oom_kill":
			events.OomKill = value
		case "oom_group":
			if events.OomGroup == nil {
				events.OomGroup = new(uint64)
			}
			*events.OomGroup = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning memory.events: %v", err)
	}
	if lineNum == 0 {
		return nil, errors.New("empty memory.events file")
	}
	return events, nil
}

// ParseMemoryEventsFromFile_008 opens a memory.events file and parses it.
func ParseMemoryEventsFromFile_008(path string) (*ResourceAgentMemoryEvents, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open memory.events file %q: %v", path, err)
	}
	defer f.Close()
	events, err := ParseMemoryEventsFromReader_008(f)
	if err != nil {
		return nil, err
	}
	events.Path = path
	return events, nil
}

// ValidateMemoryEvents_008 checks that the parsed events are semantically valid.
// Returns an error if any count is unexpectedly high (e.g., > 2^32 for sanity)
// or if an oom_kill count is present but oom count is zero (unlikely but worth flagging).
func ValidateMemoryEvents_008(events *ResourceAgentMemoryEvents) error {
	if events == nil {
		return errors.New("events is nil")
	}
	if events.Oom < events.OomKill {
		return fmt.Errorf("oom count (%d) is less than oom_kill count (%d)", events.Oom, events.OomKill)
	}
	// Sanity check: counts should not exceed maximum reasonable value
	// (e.g., 4 billion, since memory.events is cumulative and rarely exceeds 2^32)
	const maxReasonableCount uint64 = 1 << 32
	if events.Oom > maxReasonableCount {
		return fmt.Errorf("oom count %d exceeds reasonable maximum %d", events.Oom, maxReasonableCount)
	}
	if events.OomKill > maxReasonableCount {
		return fmt.Errorf("oom_kill count %d exceeds reasonable maximum %d", events.OomKill, maxReasonableCount)
	}
	if events.OomGroup != nil && *events.OomGroup > maxReasonableCount {
		return fmt.Errorf("oom_group count %d exceeds reasonable maximum %d", *events.OomGroup, maxReasonableCount)
	}
	return nil
}

// ConvertMemoryEventsToRecords_008 produces a slice of evidence records from parsed events.
// It creates records for each type of OOM event present (oom, oom_kill, oom_group).
// If a count is zero, no record is created for that type (unless forced).
func ConvertMemoryEventsToRecords_008(events *ResourceAgentMemoryEvents) []ResourceAgentOomEventRecord {
	if events == nil {
		return nil
	}
	now := time.Now().UTC()
	var records []ResourceAgentOomEventRecord
	// Record for OOM events
	if events.Oom > 0 {
		records = append(records, ResourceAgentOomEventRecord{
			ID:         fmt.Sprintf("oom-%d-%s", events.Oom, now.Format("20060102150405.000000000")),
			Timestamp:  now,
			CgroupPath: events.Path,
			EventType:  "oom",
			Count:      events.Oom,
			Diagnostics: fmt.Sprintf("Cumulative OOM events at %s", events.ParsedAt.Format(time.RFC3339)),
		})
	}
	// Record for OOM kill events
	if events.OomKill > 0 {
		records = append(records, ResourceAgentOomEventRecord{
			ID:         fmt.Sprintf("oom_kill-%d-%s", events.OomKill, now.Format("20060102150405.000000000")),
			Timestamp:  now,
			CgroupPath: events.Path,
			EventType:  "oom_kill",
			Count:      events.OomKill,
			Diagnostics: fmt.Sprintf("Cumulative OOM kill events at %s", events.ParsedAt.Format(time.RFC3339)),
		})
	}
	// Record for OOM group events if present
	if events.OomGroup != nil && *events.OomGroup > 0 {
		records = append(records, ResourceAgentOomEventRecord{
			ID:         fmt.Sprintf("oom_group-%d-%s", *events.OomGroup, now.Format("20060102150405.000000000")),
			Timestamp:  now,
			CgroupPath: events.Path,
			EventType:  "oom_group",
			Count:      *events.OomGroup,
			Diagnostics: fmt.Sprintf("Cumulative OOM group kill events at %s", events.ParsedAt.Format(time.RFC3339)),
		})
	}
	return records
}

// ParseAndGenerateRecords_008 combines reading, parsing, validation, and record generation.
func ParseAndGenerateRecords_008(path string) (*ResourceAgentOomEventParseResult, error) {
	events, err := ParseMemoryEventsFromFile_008(path)
	if err != nil {
		return nil, err
	}
	if err := ValidateMemoryEvents_008(events); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}
	records := ConvertMemoryEventsToRecords_008(events)
	return &ResourceAgentOomEventParseResult{Events: *events, Records: records}, nil
}

// ExampleEventTable_008 provides known-good memory.events contents and their expected outcomes.
// This table can be used in tests or for deterministic reference.
var ExampleEventTable_008 = []struct {
	Input      string
	ExpectOom  uint64
	ExpectKill uint64
	ExpectGrp  *uint64
	WantErr    bool
}{
	{
		Input:      "oom 0\noom_kill 0\n",
		ExpectOom:  0,
		ExpectKill: 0,
		ExpectGrp:  nil,
		WantErr:    false,
	},
	{
		Input:      "oom 5\noom_kill 3\n",
		ExpectOom:  5,
		ExpectKill: 3,
		ExpectGrp:  nil,
		WantErr:    false,
	},
	{
		Input:      "oom 100\noom_kill 100\noom_group 12\n",
		ExpectOom:  100,
		ExpectKill: 100,
		ExpectGrp:  func() *uint64 { v := uint64(12); return &v }(),
		WantErr:    false,
	},
	{
		Input:      "oom_kill 10\n", // missing oom line, but valid
		ExpectOom:  0,
		ExpectKill: 10,
		ExpectGrp:  nil,
		WantErr:    false,
	},
	{
		Input:      "unknown 1\n",
		ExpectOom:  0,
		ExpectKill: 0,
		ExpectGrp:  nil,
		WantErr:    true,
	},
}

// ValidateEventCountsFromTable_008 validates the EventTable entries for consistency.
func ValidateEventCountsFromTable_008() error {
	for i, tc := range ExampleEventTable_008 {
		if tc.WantErr {
			continue
		}
		if tc.ExpectOom > 1<<32 || tc.ExpectKill > 1<<32 || (tc.ExpectGrp != nil && *tc.ExpectGrp > 1<<32) {
			return fmt.Errorf("table entry %d has count exceeding reasonable maximum", i)
		}
		if tc.ExpectOom < tc.ExpectKill {
			return fmt.Errorf("table entry %d: oom %d < oom_kill %d", i, tc.ExpectOom, tc.ExpectKill)
		}
	}
	return nil
}

// mockMemoryEventsFile_008 creates a temporary file with the given content and returns the path.
func mockMemoryEventsFile_008(content string) (string, func(), error) {
	f, err := os.CreateTemp("", "memory-events-*.events")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create temp file: %v", err)
	}
	path := f.Name()
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, fmt.Errorf("cannot write temp file: %v", err)
	}
	f.Close()
	cleanup := func() { os.Remove(path) }
	return path, cleanup, nil
}

// TestParseMemoryEventsFromTable_008 runs the example table and reports errors.
func TestParseMemoryEventsFromTable_008() []error {
	var errs []error
	for i, tc := range ExampleEventTable_008 {
		path, cleanup, err := mockMemoryEventsFile_008(tc.Input)
		if err != nil {
			errs = append(errs, fmt.Errorf("entry %d: setup failed: %v", i, err))
			continue
		}
		events, parseErr := ParseMemoryEventsFromFile_008(path)
		cleanup()
		if tc.WantErr {
			if parseErr == nil {
				errs = append(errs, fmt.Errorf("entry %d: expected error but got none", i))
			}
			continue
		}
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("entry %d: unexpected parse error: %v", i, parseErr))
			continue
		}
		if events.Oom != tc.ExpectOom {
			errs = append(errs, fmt.Errorf("entry %d: oom expected %d, got %d", i, tc.ExpectOom, events.Oom))
		}
		if events.OomKill != tc.ExpectKill {
			errs = append(errs, fmt.Errorf("entry %d: oom_kill expected %d, got %d", i, tc.ExpectKill, events.OomKill))
		}
		if tc.ExpectGrp == nil && events.OomGroup != nil {
			errs = append(errs, fmt.Errorf("entry %d: oom_group expected nil, got %d", i, *events.OomGroup))
		}
		if tc.ExpectGrp != nil && events.OomGroup == nil {
			errs = append(errs, fmt.Errorf("entry %d: oom_group expected %d, got nil", i, *tc.ExpectGrp))
		}
		if tc.ExpectGrp != nil && events.OomGroup != nil && *events.OomGroup != *tc.ExpectGrp {
			errs = append(errs, fmt.Errorf("entry %d: oom_group expected %d, got %d", i, *tc.ExpectGrp, *events.OomGroup))
		}
	}
	return errs
}

// extractEventCounts_008 is a helper to quickly get counts from a parsed events struct.
func extractEventCounts_008(e *ResourceAgentMemoryEvents) (oom, oomKill uint64, oomGroup *uint64) {
	if e == nil {
		return 0, 0, nil
	}
	return e.Oom, e.OomKill, e.OomGroup
}

// RecordIDFromEvent_008 generates a deterministic record ID for an event type and count.
func RecordIDFromEvent_008(eventType string, count uint64, timestamp time.Time) string {
	ts := timestamp.Format("20060102150405.000000000")
	return fmt.Sprintf("%s-%d-%s", eventType, count, ts)
}

// ParseMemoryEventsFromString_008 parses memory.events from a string (useful for testing).
func ParseMemoryEventsFromString_008(content string) (*ResourceAgentMemoryEvents, error) {
	return ParseMemoryEventsFromReader_008(strings.NewReader(content))
}

// AreEventsIdentical_008 compares two ResourceAgentMemoryEvents for equality.
func AreEventsIdentical_008(a, b *ResourceAgentMemoryEvents) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Oom != b.Oom || a.OomKill != b.OomKill {
		return false
	}
	if a.OomGroup == nil && b.OomGroup == nil {
		return true
	}
	if a.OomGroup == nil || b.OomGroup == nil {
		return false
	}
	return *a.OomGroup == *b.OomGroup
}

// SanityCheckEvents_008 performs a quick non-validating check to see if counts make sense in context.
func SanityCheckEvents_008(events *ResourceAgentMemoryEvents) bool {
	if events == nil {
		return false
	}
	if events.Oom > 1<<32 || events.OomKill > 1<<32 {
		return false
	}
	if events.OomGroup != nil && *events.OomGroup > 1<<32 {
		return false
	}
	return true
}

// FormatEventsForDisplay_008 returns a multi-line string representation of the events.
func FormatEventsForDisplay_008(events *ResourceAgentMemoryEvents) string {
	if events == nil {
		return "<nil>"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("oom: %d\n", events.Oom))
	b.WriteString(fmt.Sprintf("oom_kill: %d\n", events.OomKill))
	if events.OomGroup != nil {
		b.WriteString(fmt.Sprintf("oom_group: %d\n", *events.OomGroup))
	}
	b.WriteString(fmt.Sprintf("parsed_at: %s\n", events.ParsedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("path: %s\n", events.Path))
	return b.String()
}

// minEventsLength_008 is the minimum acceptable length for a memory.events content string.
const minEventsLength_008 = 10

// EnsureEventFileContentValid_008 checks the raw content for obvious issues before parsing.
func EnsureEventFileContentValid_008(content string) error {
	if len(content) < minEventsLength_008 {
		return fmt.Errorf("content too short (%d bytes)", len(content))
	}
	if !strings.Contains(content, "oom") {
		return errors.New("content must contain at least 'oom' key")
	}
	return nil
}

// recordToEventType_008 maps a record event type to its key counterpart.
func recordToEventType_008(eventType string) string {
	switch eventType {
	case "oom":
		return "oom"
	case "oom_kill":
		return "oom_kill"
	case "oom_group":
		return "oom_group"
	default:
		return ""
	}
}

// eventTypeToRecordName_008 maps memory.events key to a human-readable record name.
func eventTypeToRecordName_008(key string) string {
	switch key {
	case "oom":
		return "OOM event"
	case "oom_kill":
		return "OOM kill event"
	case "oom_group":
		return "OOM group kill event"
	default:
		return ""
	}
}

// accumulateEventCounts_008 returns the sum of all event counts.
func accumulateEventCounts_008(events *ResourceAgentMemoryEvents) uint64 {
	if events == nil {
		return 0
	}
	total := events.Oom + events.OomKill
	if events.OomGroup != nil {
		total += *events.OomGroup
	}
	return total
}

// ensureNonZero_008 is a self-test: at least one validation function must exist.
func ensureNonZero_008() {
	_ = ValidateMemoryEvents_008
}
