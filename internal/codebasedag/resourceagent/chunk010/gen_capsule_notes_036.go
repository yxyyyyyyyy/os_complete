package chunk010

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ResourceAgentCapsulePathError defines errors for capsule path operations.
type ResourceAgentCapsulePathError struct {
	Op      string
	Path    string
	Message string
}

func (e *ResourceAgentCapsulePathError) Error() string {
	return fmt.Sprintf("capsule path error [op=%s path=%s]: %s", e.Op, e.Path, e.Message)
}

// ResourceAgentCapsulePathComponents holds decomposed parts of a capsule path.
// Format: /capsules/{namespace}/{name}/{version}/{evidence_mode}
type ResourceAgentCapsulePathComponents struct {
	Namespace    string
	Name         string
	Version      string
	EvidenceMode ResourceAgentEvidenceMode_036
}

// ResourceAgentEvidenceMode_036 represents the mode of evidence collection for a capsule.
type ResourceAgentEvidenceMode_036 int

const (
	EvidenceModeNone_036    ResourceAgentEvidenceMode_036 = iota // No evidence tracked
	EvidenceModePartial_036                                      // Partial evidence collected
	EvidenceModeFull_036                                         // Full evidence collected
)

func (m ResourceAgentEvidenceMode_036) String() string {
	switch m {
	case EvidenceModeNone_036:
		return "none"
	case EvidenceModePartial_036:
		return "partial"
	case EvidenceModeFull_036:
		return "full"
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

// ParseCapsulePath_036 parses a capsule path string into its components.
// Expected format: /capsules/{namespace}/{name}/{version}/{evidence_mode}
// Each segment must be non-empty and conform to naming rules.
func ParseCapsulePath_036(path string) (*ResourceAgentCapsulePathComponents, error) {
	if path == "" {
		return nil, &ResourceAgentCapsulePathError{
			Op:      "parse",
			Path:    path,
			Message: "path is empty",
		}
	}

	// Normalize leading slash
	cleaned := strings.TrimPrefix(path, "/")
	parts := strings.Split(cleaned, "/")
	if len(parts) != 5 {
		return nil, &ResourceAgentCapsulePathError{
			Op:      "parse",
			Path:    path,
			Message: fmt.Sprintf("expected 5 segments, got %d", len(parts)),
		}
	}

	// Check that the first segment is "capsules"
	if parts[0] != "capsules" {
		return nil, &ResourceAgentCapsulePathError{
			Op:      "parse",
			Path:    path,
			Message: fmt.Sprintf("first segment must be 'capsules', got %q", parts[0]),
		}
	}

	namespace := parts[1]
	name := parts[2]
	version := parts[3]
	modeStr := parts[4]

	if namespace == "" || name == "" || version == "" || modeStr == "" {
		return nil, &ResourceAgentCapsulePathError{
			Op:      "parse",
			Path:    path,
			Message: "all segments must be non-empty",
		}
	}

	var mode ResourceAgentEvidenceMode_036
	switch modeStr {
	case "none":
		mode = EvidenceModeNone_036
	case "partial":
		mode = EvidenceModePartial_036
	case "full":
		mode = EvidenceModeFull_036
	default:
		return nil, &ResourceAgentCapsulePathError{
			Op:      "parse",
			Path:    path,
			Message: fmt.Sprintf("invalid evidence mode %q", modeStr),
		}
	}

	return &ResourceAgentCapsulePathComponents{
		Namespace:    namespace,
		Name:         name,
		Version:      version,
		EvidenceMode: mode,
	}, nil
}

// ValidateCapsulePath_036 validates the components of a capsule path.
// It checks naming conventions for namespace, name, version.
// Additionally, it ensures evidence mode is valid.
func ValidateCapsulePath_036(components *ResourceAgentCapsulePathComponents) error {
	if components == nil {
		return errors.New("capsule path components are nil")
	}

	// Validate namespace: lowercase alphanumeric with hyphens, 3-63 characters
	if err := validateSegment_036("namespace", components.Namespace, namePattern_036); err != nil {
		return &ResourceAgentCapsulePathError{
			Op:      "validate",
			Path:    fmt.Sprintf("capsules/%s/%s/%s/%s", components.Namespace, components.Name, components.Version, components.EvidenceMode),
			Message: err.Error(),
		}
	}

	// Validate name: same pattern, but allow underscores too
	if err := validateSegment_036("name", components.Name, namePatternWithUnderscore_036); err != nil {
		return &ResourceAgentCapsulePathError{
			Op:      "validate",
			Path:    fmt.Sprintf("capsules/%s/%s/%s/%s", components.Namespace, components.Name, components.Version, components.EvidenceMode),
			Message: err.Error(),
		}
	}

	// Validate version: must start with 'v' followed by digits and dots (e.g., v1.2.3)
	if !versionPattern_036.MatchString(components.Version) {
		return &ResourceAgentCapsulePathError{
			Op:      "validate",
			Path:    fmt.Sprintf("capsules/%s/%s/%s/%s", components.Namespace, components.Name, components.Version, components.EvidenceMode),
			Message: fmt.Sprintf("invalid version format %q; must match pattern %s", components.Version, versionPattern_036.String()),
		}
	}

	// Evidence mode is already validated during parse, but double-check valid range
	if components.EvidenceMode < EvidenceModeNone_036 || components.EvidenceMode > EvidenceModeFull_036 {
		return &ResourceAgentCapsulePathError{
			Op:      "validate",
			Path:    fmt.Sprintf("capsules/%s/%s/%s/%s", components.Namespace, components.Name, components.Version, components.EvidenceMode),
			Message: fmt.Sprintf("evidence mode value %d out of range", int(components.EvidenceMode)),
		}
	}

	return nil
}

var namePattern_036 = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
var namePatternWithUnderscore_036 = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]{0,61}[a-z0-9])?$`)
var versionPattern_036 = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)*$`)

func validateSegment_036(segmentName, segment string, pattern *regexp.Regexp) error {
	if segment == "" {
		return fmt.Errorf("%s segment is empty", segmentName)
	}
	if !pattern.MatchString(segment) {
		return fmt.Errorf("%s %q does not match required pattern %s", segmentName, segment, pattern.String())
	}
	return nil
}

// ResourceAgentEvidenceModeTransition defines an allowed transition between evidence modes.
type ResourceAgentEvidenceModeTransition struct {
	From ResourceAgentEvidenceMode_036
	To   ResourceAgentEvidenceMode_036
}

// ResourceAgentEvidenceModeTransitionTable_036 is a deterministic table of allowed transitions.
// It follows the rule: none -> partial -> full (progressively) and reverse is not allowed.
var ResourceAgentEvidenceModeTransitionTable_036 = []ResourceAgentEvidenceModeTransition{
	{From: EvidenceModeNone_036, To: EvidenceModePartial_036},
	{From: EvidenceModePartial_036, To: EvidenceModeFull_036},
	// Additionally, staying in the same mode is valid (handled elsewhere).
}

// IsValidEvidenceModeTransition_036 checks if a transition from one mode to another is allowed.
// Staying in the same mode is considered valid.
func IsValidEvidenceModeTransition_036(from, to ResourceAgentEvidenceMode_036) bool {
	if from == to {
		return true
	}
	for _, t := range ResourceAgentEvidenceModeTransitionTable_036 {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

// ValidateEvidenceModeTransition_036 returns an error if the transition is invalid.
func ValidateEvidenceModeTransition_036(from, to ResourceAgentEvidenceMode_036) error {
	if !IsValidEvidenceModeTransition_036(from, to) {
		return fmt.Errorf("evidence mode transition from %s to %s is not allowed", from, to)
	}
	return nil
}

// ResourceAgentDegradedRule_036 defines a rule that may cause a resource to be marked degraded.
type ResourceAgentDegradedRule_036 struct {
	// Unique identifier for the rule.
	RuleID string
	// Description of the rule.
	Description string
	// Condition is a function that takes capsule path components and returns true if degraded.
	Condition func(components *ResourceAgentCapsulePathComponents) bool
	// Severity indicates severity when rule triggers (0..10, higher = more severe).
	Severity int
}

// ValidateDegradedRule_036 validates a degraded rule.
func ValidateDegradedRule_036(rule ResourceAgentDegradedRule_036) error {
	if strings.TrimSpace(rule.RuleID) == "" {
		return errors.New("degraded rule ID must not be empty")
	}
	if rule.Condition == nil {
		return fmt.Errorf("degraded rule %q has nil condition", rule.RuleID)
	}
	if rule.Severity < 0 || rule.Severity > 10 {
		return fmt.Errorf("degraded rule %q severity %d out of range 0..10", rule.RuleID, rule.Severity)
	}
	return nil
}

// ResourceAgentDegradedResult holds the outcome of evaluating a rule.
type ResourceAgentDegradedResult struct {
	RuleID   string
	Degraded bool
	Message  string
}

// ResourceAgentDegradedRuleSet_036 is a collection of degraded rules.
type ResourceAgentDegradedRuleSet_036 struct {
	rules []ResourceAgentDegradedRule_036
}

// NewDegradedRuleSet_036 creates an empty rule set.
func NewDegradedRuleSet_036() *ResourceAgentDegradedRuleSet_036 {
	return &ResourceAgentDegradedRuleSet_036{
		rules: make([]ResourceAgentDegradedRule_036, 0),
	}
}

// AddRule adds a validated rule to the set.
func (rs *ResourceAgentDegradedRuleSet_036) AddRule(rule ResourceAgentDegradedRule_036) error {
	if err := ValidateDegradedRule_036(rule); err != nil {
		return fmt.Errorf("cannot add invalid rule: %w", err)
	}
	// Check for duplicate RuleID
	for _, existing := range rs.rules {
		if existing.RuleID == rule.RuleID {
			return fmt.Errorf("rule %q already exists in set", rule.RuleID)
		}
	}
	rs.rules = append(rs.rules, rule)
	return nil
}

// EvaluateAll evaluates all rules against the given capsule components.
func (rs *ResourceAgentDegradedRuleSet_036) EvaluateAll(components *ResourceAgentCapsulePathComponents) []ResourceAgentDegradedResult {
	results := make([]ResourceAgentDegradedResult, 0, len(rs.rules))
	for _, rule := range rs.rules {
		degraded := rule.Condition(components)
		msg := ""
		if degraded {
			msg = fmt.Sprintf("rule %q triggered: %s", rule.RuleID, rule.Description)
		}
		results = append(results, ResourceAgentDegradedResult{
			RuleID:   rule.RuleID,
			Degraded: degraded,
			Message:  msg,
		})
	}
	return results
}

// ResourceAgentDefaultDegradedRules_036 provides a set of default degraded rules.
// These are deterministic and table-driven.
var ResourceAgentDefaultDegradedRules_036 = []ResourceAgentDegradedRule_036{
	{
		RuleID:      "DEGRADED_CAPSULE_MODE_NONE",
		Description: "Capsule has no evidence tracked; considered degraded",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return comp.EvidenceMode == EvidenceModeNone_036
		},
		Severity: 8,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_PARTIAL",
		Description: "Capsule has only partial evidence; may indicate incomplete data",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return comp.EvidenceMode == EvidenceModePartial_036
		},
		Severity: 4,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_OLD_VERSION",
		Description: "Capsule version uses outdated naming (e.g., no minor version)",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return strings.Count(comp.Version, ".") == 0
		},
		Severity: 2,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_LONG_NAME",
		Description: "Capsule name exceeds 50 characters",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return len(comp.Name) > 50
		},
		Severity: 3,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_LONG_NAMESPACE",
		Description: "Capsule namespace exceeds 50 characters",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return len(comp.Namespace) > 50
		},
		Severity: 3,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_EMPTY_VERSION",
		Description: "Capsule version is empty (should never happen after validation, but safety check)",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return comp.Version == ""
		},
		Severity: 10,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_VERSION_NEGATIVE",
		Description: "Capsule version contains negative component (e.g., v-1.2)",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return strings.Contains(comp.Version, "-")
		},
		Severity: 5,
	},
	{
		RuleID:      "DEGRADED_CAPSULE_VERSION_TRAILING_DOT",
		Description: "Capsule version ends with a dot (e.g., v1.2.)",
		Condition: func(comp *ResourceAgentCapsulePathComponents) bool {
			return strings.HasSuffix(comp.Version, ".")
		},
		Severity: 5,
	},
}
