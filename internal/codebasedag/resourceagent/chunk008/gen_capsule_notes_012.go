package chunk008

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ResourceAgentEvidenceMode represents the mode of evidence collection.
type ResourceAgentEvidenceMode int

const (
	EvidenceModeInvalid_012  ResourceAgentEvidenceMode = 0
	EvidenceModeLive_012     ResourceAgentEvidenceMode = 1
	EvidenceModeSnapshot_012 ResourceAgentEvidenceMode = 2
	EvidenceModeReplay_012   ResourceAgentEvidenceMode = 3
	EvidenceModeArchive_012  ResourceAgentEvidenceMode = 4
	EvidenceModeDegraded_012 ResourceAgentEvidenceMode = 5
)

// ResourceAgentDegradedRule defines a rule for degradation with condition and action.
type ResourceAgentDegradedRule struct {
	RuleID    string               `json:"rule_id"`
	Severity  int                  `json:"severity"`
	Condition func(interface{}) bool `json:"-"`
	Action    func(interface{}) error `json:"-"`
	Timeout   time.Duration        `json:"timeout"`
}

// TransitionEntry represents a valid evidence mode transition.
type TransitionEntry struct {
	From ResourceAgentEvidenceMode `json:"from"`
	To   ResourceAgentEvidenceMode `json:"to"`
}

// DegradedRuleEntry is a static descriptor for degraded rules.
type DegradedRuleEntry struct {
	RuleID      string `json:"rule_id"`
	Severity    int    `json:"severity"`
	Description string `json:"description"`
}

// ResourceAgentCapsulePathNaming defines the structure for capsule path naming.
type ResourceAgentCapsulePathNaming struct {
	Prefix    string `json:"prefix"`
	Separator string `json:"separator"`
	MaxLength int    `json:"max_length"`
	Regexp    *regexp.Regexp
}

// ResourceAgentCapsuleNote holds combined note information for capsule path,
// evidence mode transitions, and degraded rules.
type ResourceAgentCapsuleNote struct {
	PathNaming ResourceAgentCapsulePathNaming `json:"path_naming"`
	Evidence   ResourceAgentEvidenceMode      `json:"evidence_mode"`
	Degraded   []ResourceAgentDegradedRule    `json:"degraded_rules"`
}

// capsulePathNamingDefaults_012 provides default constraints for path naming.
var capsulePathNamingDefaults_012 = ResourceAgentCapsulePathNaming{
	Prefix:    "capsule",
	Separator: "_",
	MaxLength: 255,
}

// degradedRuleEntryTable_012 holds predefined degraded rule templates.
var degradedRuleEntryTable_012 = []DegradedRuleEntry{
	{
		RuleID:      "DR001",
		Severity:    1,
		Description: "Evidence source unavailable, degrade to fallback",
	},
	{
		RuleID:      "DR002",
		Severity:    2,
		Description: "Network latency exceeds threshold, enable degraded mode",
	},
	{
		RuleID:      "DR003",
		Severity:    3,
		Description: "Storage quota reached, switch to archive- degraded",
	},
	{
		RuleID:      "DR004",
		Severity:    4,
		Description: "Critical data integrity failure, force degraded",
	},
}

// evidenceModeTransitionTable_012 defines allowed transitions.
var evidenceModeTransitionTable_012 = []TransitionEntry{
	{From: EvidenceModeLive_012, To: EvidenceModeSnapshot_012},
	{From: EvidenceModeLive_012, To: EvidenceModeReplay_012},
	{From: EvidenceModeSnapshot_012, To: EvidenceModeArchive_012},
	{From: EvidenceModeReplay_012, To: EvidenceModeLive_012},
	{From: EvidenceModeReplay_012, To: EvidenceModeDegraded_012},
	{From: EvidenceModeArchive_012, To: EvidenceModeLive_012},
	{From: EvidenceModeDegraded_012, To: EvidenceModeLive_012},
}

// evidenceModeNames_012 maps modes to human-readable names.
var evidenceModeNames_012 = map[ResourceAgentEvidenceMode]string{
	EvidenceModeInvalid_012:  "invalid",
	EvidenceModeLive_012:     "live",
	EvidenceModeSnapshot_012: "snapshot",
	EvidenceModeReplay_012:   "replay",
	EvidenceModeArchive_012:  "archive",
	EvidenceModeDegraded_012: "degraded",
}

// GetCapsulePathNamingDefaults_012 returns default path naming settings.
func GetCapsulePathNamingDefaults_012() ResourceAgentCapsulePathNaming {
	return capsulePathNamingDefaults_012
}

// NewCapsulePathNaming_012 creates a new path naming config after validation.
func NewCapsulePathNaming_012(prefix, separator string, maxLength int) (*ResourceAgentCapsulePathNaming, error) {
	naming := &ResourceAgentCapsulePathNaming{
		Prefix:    prefix,
		Separator: separator,
		MaxLength: maxLength,
	}
	if err := ValidateCapsulePathNaming_012(naming); err != nil {
		return nil, fmt.Errorf("invalid path naming: %w", err)
	}
	// Build regexp based on prefix and separator
	pattern := fmt.Sprintf("^%s[%s][a-zA-Z0-9_]+$", regexp.QuoteMeta(prefix), regexp.QuoteMeta(separator))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}
	naming.Regexp = re
	return naming, nil
}

// ValidateCapsulePathNaming_012 validates the path naming configuration.
func ValidateCapsulePathNaming_012(naming *ResourceAgentCapsulePathNaming) error {
	if naming == nil {
		return errors.New("naming cannot be nil")
	}
	if naming.Prefix == "" {
		return errors.New("prefix cannot be empty")
	}
	if len(naming.Prefix) > 64 {
		return errors.New("prefix too long (max 64)")
	}
	if naming.Separator == "" {
		return errors.New("separator cannot be empty")
	}
	if len(naming.Separator) > 4 {
		return errors.New("separator too long (max 4)")
	}
	if naming.MaxLength < 10 || naming.MaxLength > 1024 {
		return fmt.Errorf("maxLength must be between 10 and 1024, got %d", naming.MaxLength)
	}
	return nil
}

// ValidateCapsuleNote_012 validates the entire capsule note.
func ValidateCapsuleNote_012(note *ResourceAgentCapsuleNote) error {
	if note == nil {
		return errors.New("note cannot be nil")
	}
	if err := ValidateCapsulePathNaming_012(&note.PathNaming); err != nil {
		return fmt.Errorf("path naming: %w", err)
	}
	if err := ValidateEvidenceMode_012(note.Evidence); err != nil {
		return fmt.Errorf("evidence mode: %w", err)
	}
	for _, rule := range note.Degraded {
		if err := ValidateDegradedRule_012(&rule); err != nil {
			return fmt.Errorf("degraded rule %s: %w", rule.RuleID, err)
		}
	}
	return nil
}

// ValidateEvidenceMode_012 checks if the evidence mode is valid.
func ValidateEvidenceMode_012(mode ResourceAgentEvidenceMode) error {
	if mode <= EvidenceModeInvalid_012 || mode > EvidenceModeDegraded_012 {
		return fmt.Errorf("invalid evidence mode: %d", mode)
	}
	return nil
}

// ValidateDegradedRule_012 validates a degraded rule.
func ValidateDegradedRule_012(rule *ResourceAgentDegradedRule) error {
	if rule == nil {
		return errors.New("degraded rule cannot be nil")
	}
	if rule.RuleID == "" {
		return errors.New("rule ID cannot be empty")
	}
	if rule.Severity < 1 || rule.Severity > 5 {
		return fmt.Errorf("severity must be 1-5, got %d", rule.Severity)
	}
	if rule.Condition == nil {
		return errors.New("condition function cannot be nil")
	}
	if rule.Action == nil {
		return errors.New("action function cannot be nil")
	}
	if rule.Timeout < 0 {
		return errors.New("timeout must be non-negative")
	}
	return nil
}

// IsValidEvidenceTransition_012 checks if a transition is permitted.
func IsValidEvidenceTransition_012(from, to ResourceAgentEvidenceMode) bool {
	for _, entry := range evidenceModeTransitionTable_012 {
		if entry.From == from && entry.To == to {
			return true
		}
	}
	return false
}

// GetEvidenceTransitionTable_012 returns a copy of the transition table.
func GetEvidenceTransitionTable_012() []TransitionEntry {
	table := make([]TransitionEntry, len(evidenceModeTransitionTable_012))
	copy(table, evidenceModeTransitionTable_012)
	return table
}

// GetDegradedRuleEntryTable_012 returns a copy of the degraded rule entry table.
func GetDegradedRuleEntryTable_012() []DegradedRuleEntry {
	table := make([]DegradedRuleEntry, len(degradedRuleEntryTable_012))
	copy(table, degradedRuleEntryTable_012)
	return table
}

// EvidenceModeName_012 returns a human-readable name for an evidence mode.
func EvidenceModeName_012(mode ResourceAgentEvidenceMode) (string, error) {
	name, ok := evidenceModeNames_012[mode]
	if !ok {
		return "", fmt.Errorf("unknown evidence mode %d", mode)
	}
	return name, nil
}

// ParseEvidenceModeName_012 converts a string back to an evidence mode.
func ParseEvidenceModeName_012(name string) (ResourceAgentEvidenceMode, error) {
	for mode, n := range evidenceModeNames_012 {
		if strings.EqualFold(n, name) {
			return mode, nil
		}
	}
	return EvidenceModeInvalid_012, fmt.Errorf("unknown evidence mode name: %s", name)
}

// ParseCapsulePath_012 splits a capsule path into components using the naming rules.
func ParseCapsulePath_012(path string) (prefix string, name string, err error) {
	if path == "" {
		return "", "", errors.New("path cannot be empty")
	}
	defaults := capsulePathNamingDefaults_012
	sep := defaults.Separator
	maxLen := defaults.MaxLength
	if len(path) > maxLen {
		return "", "", fmt.Errorf("path exceeds max length %d", maxLen)
	}
	parts := strings.SplitN(path, sep, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("path must contain separator '%s'", sep)
	}
	prefix = parts[0]
	name = parts[1]
	if len(prefix) == 0 || len(name) == 0 {
		return "", "", errors.New("prefix and name must be non-empty")
	}
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(prefix) {
		return "", "", errors.New("invalid prefix characters")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name) {
		return "", "", errors.New("invalid name characters")
	}
	return prefix, name, nil
}

// ApplyDegradedRule_012 executes a degraded rule's condition and action.
func ApplyDegradedRule_012(rule ResourceAgentDegradedRule, state interface{}) error {
	if rule.Condition(state) {
		return rule.Action(state)
	}
	return nil
}

// BuildCapsulePath_012 constructs a capsule path from components.
func BuildCapsulePath_012(prefix, name string) (string, error) {
	if prefix == "" || name == "" {
		return "", errors.New("prefix and name must be non-empty")
	}
	defaults := capsulePathNamingDefaults_012
	path := prefix + defaults.Separator + name
	if len(path) > defaults.MaxLength {
		return "", fmt.Errorf("constructed path length %d exceeds max %d", len(path), defaults.MaxLength)
	}
	return path, nil
}

// ValidateCapsulePath_012 validates a fully constructed capsule path.
func ValidateCapsulePath_012(path string) error {
	defaults := capsulePathNamingDefaults_012
	if len(path) < 3 {
		return errors.New("path too short")
	}
	if len(path) > defaults.MaxLength {
		return fmt.Errorf("path length %d exceeds max %d", len(path), defaults.MaxLength)
	}
	re := regexp.MustCompile(fmt.Sprintf("^%s[%s].+$",
		regexp.QuoteMeta(defaults.Prefix), regexp.QuoteMeta(defaults.Separator)))
	if !re.MatchString(path) {
		return fmt.Errorf("path must start with '%s%s'", defaults.Prefix, defaults.Separator)
	}
	// Check no pathological patterns
	badPatterns := []string{"..", "//", "~", "\\"}
	for _, bp := range badPatterns {
		if strings.Contains(path, bp) {
			return fmt.Errorf("path contains forbidden pattern: %s", bp)
		}
	}
	return nil
}

// ListAllEvidenceModes_012 returns all defined evidence modes.
func ListAllEvidenceModes_012() []ResourceAgentEvidenceMode {
	return []ResourceAgentEvidenceMode{
		EvidenceModeLive_012,
		EvidenceModeSnapshot_012,
		EvidenceModeReplay_012,
		EvidenceModeArchive_012,
		EvidenceModeDegraded_012,
	}
}

// TransitionTableAsMap_012 returns the transition table as a set for fast lookup.
func TransitionTableAsMap_012() map[ResourceAgentEvidenceMode]map[ResourceAgentEvidenceMode]struct{} {
	m := make(map[ResourceAgentEvidenceMode]map[ResourceAgentEvidenceMode]struct{})
	for _, t := range evidenceModeTransitionTable_012 {
		if m[t.From] == nil {
			m[t.From] = make(map[ResourceAgentEvidenceMode]struct{})
		}
		m[t.From][t.To] = struct{}{}
	}
	return m
}

// CheckCapsuleDegradation_012 evaluates a set of degraded rules against a state.
func CheckCapsuleDegradation_012(rules []ResourceAgentDegradedRule, state interface{}) []ResourceAgentDegradedRule {
	var triggered []ResourceAgentDegradedRule
	for _, rule := range rules {
		if rule.Condition(state) {
			triggered = append(triggered, rule)
		}
	}
	return triggered
}
