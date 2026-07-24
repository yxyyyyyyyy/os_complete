package chunk032

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Supported OOM event types as defined in cgroup v2 memory.events.
// ---------------------------------------------------------------------------

type OomEventType int

const (
	OomEventOom      OomEventType = iota // "oom"
	OomEventOomKill                      // "oom_kill"
	OomEventOomGroup                     // "oom_group"
)

var oomEventTypeNames = map[OomEventType]string{
	OomEventOom:      "oom",
	OomEventOomKill:  "oom_kill",
	OomEventOomGroup: "oom_group",
}

var oomEventTypeFromName = map[string]OomEventType{
	"oom":       OomEventOom,
	"oom_kill":  OomEventOomKill,
	"oom_group": OomEventOomGroup,
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrOomEventLineFormat = errors.New("malformed memory.events line: expected '<name> <value>'")
	ErrOomEventType       = errors.New("unknown OOM event type")
	ErrOomEventValue      = errors.New("invalid OOM event count (must be non‑negative integer)")
	ErrOomEventsEmpty     = errors.New("memory.events input is empty")
	ErrOomEventsDuplicate = errors.New("duplicate OOM event type in memory.events")
	ErrOomEventsMissing   = errors.New("missing required OOM event type")
)

// ---------------------------------------------------------------------------
// Core data structures
// ---------------------------------------------------------------------------

// MemoryEventLine holds a single parsed line from memory.events.
type MemoryEventLine struct {
	Event OomEventType
	Count uint64
}

// MemoryEvents holds all parsed event counters from a memory.events file.
type MemoryEvents struct {
	Entries map[OomEventType]uint64 // always contains exactly the three known keys
	RawText string                  // original input (optional, for debugging)
}

// OomEvidence records a snapshot of OOM-related events, typically used for
// blame assignment after an OOM kill.
type OomEvidence struct {
	OomCount      uint64 // number of times cgroup's memory was over limit
	OomKillCount  uint64 // number of processes OOM-killed within the cgroup
	OomGroupCount uint64 // number of OOM kills from group-level events
	HasOomKill    bool   // true if OomKillCount > 0
	RawEvents     *MemoryEvents
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// parseOomEventLine_032 parses one line of memory.events.
// Format: "<event_name> <count>" (whitespace separated).
func parseOomEventLine_032(line string) (MemoryEventLine, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return MemoryEventLine{}, ErrOomEventLineFormat
	}

	parts := strings.Fields(line)
	if len(parts) != 2 {
		return MemoryEventLine{}, ErrOomEventLineFormat
	}

	evt, ok := oomEventTypeFromName[parts[0]]
	if !ok {
		return MemoryEventLine{}, fmt.Errorf("%w: %q", ErrOomEventType, parts[0])
	}

	val, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return MemoryEventLine{}, fmt.Errorf("%w: %v", ErrOomEventValue, err)
	}

	return MemoryEventLine{Event: evt, Count: val}, nil
}

// ResourceAgentParseMemoryEvents_032 parses the full content of a memory.events
// file (as a string). It validates that all three known event types are present
// and that no duplicates exist.
func ResourceAgentParseMemoryEvents_032(raw string) (*MemoryEvents, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrOomEventsEmpty
	}

	entries := make(map[OomEventType]uint64, 3)
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parsed, err := parseOomEventLine_032(line)
		if err != nil {
			return nil, fmt.Errorf("line %q: %w", line, err)
		}

		if _, exists := entries[parsed.Event]; exists {
			return nil, fmt.Errorf("duplicate event %q: %w", oomEventTypeNames[parsed.Event], ErrOomEventsDuplicate)
		}
		entries[parsed.Event] = parsed.Count
	}

	// Ensure all known types are present.
	for _, et := range []OomEventType{OomEventOom, OomEventOomKill, OomEventOomGroup} {
		if _, ok := entries[et]; !ok {
			return nil, fmt.Errorf("missing event %q: %w", oomEventTypeNames[et], ErrOomEventsMissing)
		}
	}

	return &MemoryEvents{
		Entries: entries,
		RawText: raw,
	}, nil
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// ValidateMemoryEvents_032 checks that a MemoryEvents record is internally
// consistent. It verifies that all expected keys exist and that counts are
// non‑negative (always true for uint64). For added safety it also ensures
// that oom_kill count does not exceed oom count (a common sanity rule).
func ValidateMemoryEvents_032(me *MemoryEvents) error {
	if me == nil {
		return errors.New("memory events pointer is nil")
	}
	if len(me.Entries) == 0 {
		return ErrOomEventsEmpty
	}

	required := []OomEventType{OomEventOom, OomEventOomKill, OomEventOomGroup}
	for _, et := range required {
		if _, ok := me.Entries[et]; !ok {
			return fmt.Errorf("missing event type %q: %w", oomEventTypeNames[et], ErrOomEventsMissing)
		}
	}

	// Sanity: an OOM kill cannot happen without an OOM event.
	if me.Entries[OomEventOomKill] > me.Entries[OomEventOom] {
		return fmt.Errorf("oom_kill count (%d) exceeds oom count (%d): invalid state",
			me.Entries[OomEventOomKill], me.Entries[OomEventOom])
	}
	return nil
}

// ResourceAgentValidateOomEvents_032 is a top‑level validation function that
// accepts a raw memory.events string, parses it, and validates the result.
// Returns an error if the input cannot be parsed or fails validation rules.
func ResourceAgentValidateOomEvents_032(raw string) error {
	me, err := ResourceAgentParseMemoryEvents_032(raw)
	if err != nil {
		return err
	}
	return ValidateMemoryEvents_032(me)
}

// ---------------------------------------------------------------------------
// Evidence construction
// ---------------------------------------------------------------------------

// OomEvidenceFromEvents_032 builds an OomEvidence record from parsed events.
func OomEvidenceFromEvents_032(me *MemoryEvents) *OomEvidence {
	if me == nil {
		return nil
	}
	return &OomEvidence{
		OomCount:      me.Entries[OomEventOom],
		OomKillCount:  me.Entries[OomEventOomKill],
		OomGroupCount: me.Entries[OomEventOomGroup],
		HasOomKill:    me.Entries[OomEventOomKill] > 0,
		RawEvents:     me,
	}
}

// ResourceAgentExtractOomEvidence_032 is a convenience function that takes a
// raw memory.events string and returns an OomEvidence if parsing succeeds.
func ResourceAgentExtractOomEvidence_032(raw string) (*OomEvidence, error) {
	me, err := ResourceAgentParseMemoryEvents_032(raw)
	if err != nil {
		return nil, err
	}
	return OomEvidenceFromEvents_032(me), nil
}

// ---------------------------------------------------------------------------
// String / representation
// ---------------------------------------------------------------------------

// String returns a human‑readable summary of OomEvidence.
func (oe *OomEvidence) String() string {
	if oe == nil {
		return "<nil>"
	}
	return fmt.Sprintf("OomEvidence{oom=%d, oom_kill=%d, oom_group=%d, has_kill=%t}",
		oe.OomCount, oe.OomKillCount, oe.OomGroupCount, oe.HasOomKill)
}

// MemoryEventsString_032 formats the parsed events as a table for debugging.
func MemoryEventsString_032(me *MemoryEvents) string {
	if me == nil || me.Entries == nil {
		return "<nil>"
	}
	var b strings.Builder
	for _, et := range []OomEventType{OomEventOom, OomEventOomKill, OomEventOomGroup} {
		if count, ok := me.Entries[et]; ok {
			fmt.Fprintf(&b, "%s %d\n", oomEventTypeNames[et], count)
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Deterministic table-driven helper data for testing / documentation
// ---------------------------------------------------------------------------

// MemoryEventTestData_032 is a struct representing a single test case for
// parsing and validation logic. Exported for use in unit tests.
type MemoryEventTestData_032 struct {
	Label      string
	Input      string
	WantEvents map[string]uint64 // event name -> count; nil means fail
	WantErr    error
}

// MemoryEventsTestCases_032 is a deterministic table of test cases.
// It covers happy paths, edge cases, and invalid inputs.
var MemoryEventsTestCases_032 = []MemoryEventTestData_032{
	{
		Label: "valid minimal",
		Input: "oom 0\noom_kill 0\noom_group 0\n",
		WantEvents: map[string]uint64{
			"oom":       0,
			"oom_kill":  0,
			"oom_group": 0,
		},
		WantErr: nil,
	},
	{
		Label: "valid non-zero counts",
		Input: "oom 42\noom_kill 7\noom_group 1\n",
		WantEvents: map[string]uint64{
			"oom":       42,
			"oom_kill":  7,
			"oom_group": 1,
		},
		WantErr: nil,
	},
	{
		Label: "valid with large numbers",
		Input: "oom 18446744073709551615\noom_kill 0\noom_group 0\n",
		WantEvents: map[string]uint64{
			"oom":       18446744073709551615, // max uint64
			"oom_kill":  0,
			"oom_group": 0,
		},
		WantErr: nil,
	},
	{
		Label:      "empty input",
		Input:      "",
		WantEvents: nil,
		WantErr:    ErrOomEventsEmpty,
	},
	{
		Label:      "whitespace only",
		Input:      "   \n  \n",
		WantEvents: nil,
		WantErr:    ErrOomEventsEmpty,
	},
	{
		Label: "missing oom_kill",
		Input: "oom 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventsMissing,
	},
	{
		Label: "duplicate oom",
		Input: "oom 0\noom 1\noom_kill 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventsDuplicate,
	},
	{
		Label: "unknown event type",
		Input: "oom 0\nunknown 1\noom_kill 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventType,
	},
	{
		Label: "invalid count (negative string)", // ParseUint fails
		Input: "oom -1\noom_kill 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventValue,
	},
	{
		Label: "non-integer count",
		Input: "oom abc\noom_kill 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventValue,
	},
	{
		Label: "extra field",
		Input: "oom 0 extra\noom_kill 0\noom_group 0\n",
		WantEvents: nil,
		WantErr:    ErrOomEventLineFormat,
	},
	{
		Label: "trailing whitespace",
		Input: "oom 987  \n  oom_kill 3\noom_group 0\n",
		WantEvents: map[string]uint64{
			"oom":       987,
			"oom_kill":  3,
			"oom_group": 0,
		},
		WantErr: nil,
	},
}

// ---------------------------------------------------------------------------
// Additional helpers
// ---------------------------------------------------------------------------

// ResourceAgentCountOomKills_032 returns the OOM kill count from a raw
// memory.events string. Returns 0 and an error if parsing fails.
func ResourceAgentCountOomKills_032(raw string) (uint64, error) {
	me, err := ResourceAgentParseMemoryEvents_032(raw)
	if err != nil {
		return 0, err
	}
	return me.Entries[OomEventOomKill], nil
}

// ResourceAgentHasRecentOomKill_032 returns true if the input reports at least
// one OOM kill. This is a quick check used in decision logic.
func ResourceAgentHasRecentOomKill_032(raw string) (bool, error) {
	count, err := ResourceAgentCountOomKills_032(raw)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---------------------------------------------------------------------------
// Merge / compare helpers (optional but realistic)
// ---------------------------------------------------------------------------

// MemoryEventsEqual_032 compares two MemoryEvents for equality.
func MemoryEventsEqual_032(a, b *MemoryEvents) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.Entries) != len(b.Entries) {
		return false
	}
	for k, va := range a.Entries {
		vb, ok := b.Entries[k]
		if !ok || va != vb {
			return false
		}
	}
	return true
}

// ResourceAgentMergeMemoryEvents_032 combines two MemoryEvents by summing
// their counts. If the same event type appears in both, the counts are added.
// The raw text of the first input is preserved.
func ResourceAgentMergeMemoryEvents_032(a, b *MemoryEvents) (*MemoryEvents, error) {
	if a == nil || b == nil {
		return nil, errors.New("cannot merge nil MemoryEvents")
	}
	merged := &MemoryEvents{
		Entries: make(map[OomEventType]uint64, 3),
		RawText: a.RawText,
	}
	for _, et := range []OomEventType{OomEventOom, OomEventOomKill, OomEventOomGroup} {
		va, aok := a.Entries[et]
		vb, bok := b.Entries[et]
		if !aok && !bok {
			return nil, fmt.Errorf("event type %q missing from both inputs", oomEventTypeNames[et])
		}
		var sum uint64
		if aok {
			sum += va
		}
		if bok {
			sum += vb
		}
		merged.Entries[et] = sum
	}
	return merged, nil
}

// ---------------------------------------------------------------------------
// End of file
// ---------------------------------------------------------------------------
