package chunk009

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// File index 24: capsule path naming, evidence_mode transitions, degraded rules.
// All exported symbols follow the pattern ResourceAgent<Name>_024 to avoid
// collisions with prior definitions in this package.

// ---------------------------------------------------------------------------
//  Capsule path naming
// ---------------------------------------------------------------------------

// baseCapsulePath is the prefix for all capsule paths.
const baseCapsulePath = "/nodes"

// capsulePathPattern matches a valid capsule path:
//   /nodes/{nodeID}/capsules/{capsuleID}/versions/{version}
var capsulePathPattern = regexp.MustCompile(
	`^/nodes/([a-zA-Z0-9_\-]+)/capsules/([a-zA-Z0-9_\-]+)/versions/([a-zA-Z0-9_\-\.]+)$`,
)

// ResourceAgentCapsulePathSpec_024 holds the components needed to build or
// validate a capsule path.
type ResourceAgentCapsulePathSpec_024 struct {
	NodeID    string
	CapsuleID string
	Version   string
}

// ResourceAgentNewCapsulePathSpec_024 creates a CapsulePathSpec after
// basic sanity checks on each component.
func ResourceAgentNewCapsulePathSpec_024(nodeID, capsuleID, version string) (ResourceAgentCapsulePathSpec_024, error) {
	if err := validatePathComponent(nodeID); err != nil {
		return ResourceAgentCapsulePathSpec_024{}, fmt.Errorf("invalid nodeID: %w", err)
	}
	if err := validatePathComponent(capsuleID); err != nil {
		return ResourceAgentCapsulePathSpec_024{}, fmt.Errorf("invalid capsuleID: %w", err)
	}
	if err := validateVersion(version); err != nil {
		return ResourceAgentCapsulePathSpec_024{}, fmt.Errorf("invalid version: %w", err)
	}
	return ResourceAgentCapsulePathSpec_024{
		NodeID:    nodeID,
		CapsuleID: capsuleID,
		Version:   version,
	}, nil
}

func validatePathComponent(s string) error {
	if len(s) == 0 {
		return errors.New("must not be empty")
	}
	// Allow only a subset of characters that are safe in file paths.
	if !regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`).MatchString(s) {
		return fmt.Errorf("component %q contains disallowed characters", s)
	}
	if len(s) > 128 {
		return fmt.Errorf("component %q is too long (max 128)", s)
	}
	return nil
}

func validateVersion(v string) error {
	if len(v) == 0 {
		return errors.New("version must not be empty")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`).MatchString(v) {
		return fmt.Errorf("version %q contains disallowed characters", v)
	}
	if len(v) > 64 {
		return fmt.Errorf("version %q is too long (max 64)", v)
	}
	return nil
}

// ResourceAgentGenerateCapsulePath_024 builds a capsule path from its
// components. The spec must have been obtained from
// ResourceAgentNewCapsulePathSpec_024 or validated independently.
func ResourceAgentGenerateCapsulePath_024(spec ResourceAgentCapsulePathSpec_024) string {
	return path.Join(baseCapsulePath,
		spec.NodeID, "capsules", spec.CapsuleID, "versions", spec.Version)
}

// ResourceAgentParseCapsulePath_024 attempts to parse a capsule path and
// returns the components if successful.
func ResourceAgentParseCapsulePath_024(capsulePath string) (ResourceAgentCapsulePathSpec_024, error) {
	matches := capsulePathPattern.FindStringSubmatch(capsulePath)
	if matches == nil {
		return ResourceAgentCapsulePathSpec_024{}, fmt.Errorf("path %q does not match expected pattern", capsulePath)
	}
	return ResourceAgentCapsulePathSpec_024{
		NodeID:    matches[1],
		CapsuleID: matches[2],
		Version:   matches[3],
	}, nil
}

// ResourceAgentValidateCapsulePath_024 validates a capsule path string
// against the expected pattern and component constraints.
func ResourceAgentValidateCapsulePath_024(capsulePath string) error {
	_, err := ResourceAgentParseCapsulePath_024(capsulePath)
	return err
}

// capsulePathValidationTable provides deterministic test data for capsule
// path validation. Each entry is a (path, shouldSucceed) pair.
var capsulePathValidationTable = []struct {
	Path          string
	ShouldSucceed bool
}{
	{"/nodes/node1/capsules/cap1/versions/1.0", true},
	{"/nodes/a/capsules/b/versions/c", true},
	{"/nodes/A-1_x/capsules/_012/versions/v2.3.4", true},
	{"/nodes/", false},
	{"/nodes//capsules//versions/", false},
	{"/nodes/.../capsules/../versions/..", false},
	{"/nodes/node%/capsules/cap/versions/v1", false},
	{"/nodes/node1/capsules/cap/versions/", false},
	{"/nodes/node1/capsules/cap/versions/v1/extra", false},
	{"/nodes/node1/capsules/cap/versions/%20", false},
	{"", false},
}

// ---------------------------------------------------------------------------
//  Evidence_mode transitions (suffixed _024 to avoid redeclaration)
// ---------------------------------------------------------------------------

// ResourceAgentEvidenceMode_024 represents the operational mode of a
// capsule’s evidence collection.
type ResourceAgentEvidenceMode_024 int

const (
	// ResourceAgentEvidenceModeActive_024 means evidence is being collected and published.
	ResourceAgentEvidenceModeActive_024 ResourceAgentEvidenceMode_024 = iota
	// ResourceAgentEvidenceModePassive_024 means evidence is collected but not actively
	// pushed; it is stored locally for later retrieval.
	ResourceAgentEvidenceModePassive_024
	// ResourceAgentEvidenceModeDegraded_024 means collection is impaired (e.g., missing
	// sensors) but still operating.
	ResourceAgentEvidenceModeDegraded_024
	// ResourceAgentEvidenceModeDisabled_024 means collection is explicitly turned off.
	ResourceAgentEvidenceModeDisabled_024
	// ResourceAgentEvidenceModeRetired_024 means the capsule’s evidence is no longer
	// accepted; only historical data may exist.
	ResourceAgentEvidenceModeRetired_024
)

// evidenceModeNames_024 maps each mode to its canonical string representation.
var evidenceModeNames_024 = map[ResourceAgentEvidenceMode_024]string{
	ResourceAgentEvidenceModeActive_024:   "active",
	ResourceAgentEvidenceModePassive_024:  "passive",
	ResourceAgentEvidenceModeDegraded_024: "degraded",
	ResourceAgentEvidenceModeDisabled_024: "disabled",
	ResourceAgentEvidenceModeRetired_024:  "retired",
}

// ResourceAgentEvidenceModeToString_024 returns the canonical name for an
// evidence mode.
func ResourceAgentEvidenceModeToString_024(m ResourceAgentEvidenceMode_024) (string, error) {
	if name, ok := evidenceModeNames_024[m]; ok {
		return name, nil
	}
	return "", fmt.Errorf("unknown ResourceAgentEvidenceMode_024 %d", m)
}

// ResourceAgentStringToEvidenceMode_024 parses a string into an EvidenceMode.
func ResourceAgentStringToEvidenceMode_024(s string) (ResourceAgentEvidenceMode_024, error) {
	for m, name := range evidenceModeNames_024 {
		if strings.EqualFold(s, name) {
			return m, nil
		}
	}
	return 0, fmt.Errorf("unknown evidence mode string %q", s)
}

// evidenceModeTransitions_024 defines allowed mode transitions. The key is the
// current mode, and the values are a slice of modes that can be entered.
// This table is deterministic and used by validation functions.
//
// Rules:
//   - Active can go to any other mode.
//   - Passive can go to Active, Degraded, or Disabled.
//   - Degraded can go to Active or Passive (once issue is resolved) or Disabled.
//   - Disabled can only go to Retired (to discard data) or back to Active
//     (if re‑enabled).
//   - Retired is terminal; no transitions out.
var evidenceModeTransitions_024 = map[ResourceAgentEvidenceMode_024][]ResourceAgentEvidenceMode_024{
	ResourceAgentEvidenceModeActive_024: {
		ResourceAgentEvidenceModePassive_024,
		ResourceAgentEvidenceModeDegraded_024,
		ResourceAgentEvidenceModeDisabled_024,
		ResourceAgentEvidenceModeRetired_024,
	},
	ResourceAgentEvidenceModePassive_024: {
		ResourceAgentEvidenceModeActive_024,
		ResourceAgentEvidenceModeDegraded_024,
		ResourceAgentEvidenceModeDisabled_024,
	},
	ResourceAgentEvidenceModeDegraded_024: {
		ResourceAgentEvidenceModeActive_024,
		ResourceAgentEvidenceModePassive_024,
		ResourceAgentEvidenceModeDisabled_024,
	},
	ResourceAgentEvidenceModeDisabled_024: {
		ResourceAgentEvidenceModeActive_024,
		ResourceAgentEvidenceModeRetired_024,
	},
	ResourceAgentEvidenceModeRetired_024: {}, // no outgoing transitions
}

// ResourceAgentIsTransitionAllowed_024 checks whether moving from mode 'from'
// to mode 'to' is valid according to the transition table.
func ResourceAgentIsTransitionAllowed_024(from, to ResourceAgentEvidenceMode_024) bool {
	allowed, ok := evidenceModeTransitions_024[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// ResourceAgentValidateEvidenceModeTransition_024 returns an error if the
// transition is not allowed. It also checks that both modes are valid.
func ResourceAgentValidateEvidenceModeTransition_024(from, to ResourceAgentEvidenceMode_024) error {
	if _, ok := evidenceModeNames_024[from]; !ok {
		return fmt.Errorf("invalid source ResourceAgentEvidenceMode_024 %d", from)
	}
	if _, ok := evidenceModeNames_024[to]; !ok {
		return fmt.Errorf("invalid target ResourceAgentEvidenceMode_024 %d", to)
	}
	if !ResourceAgentIsTransitionAllowed_024(from, to) {
		return fmt.Errorf("transition from %s to %s is not allowed",
			evidenceModeNames_024[from], evidenceModeNames_024[to])
	}
	return nil
}

// evidenceModeTransitionTable_024 provides deterministic test data for
// transition validation. Each entry specifies source, target, and whether
// the transition is allowed.
var evidenceModeTransitionTable_024 = []struct {
	From      ResourceAgentEvidenceMode_024
	To        ResourceAgentEvidenceMode_024
	Allowed   bool
	FromName  string
	ToName    string
}{
	{ResourceAgentEvidenceModeActive_024, ResourceAgentEvidenceModePassive_024, true, "active", "passive"},
	{ResourceAgentEvidenceModeActive_024, ResourceAgentEvidenceModeDegraded_024, true, "active", "degraded"},
	{ResourceAgentEvidenceModeActive_024, ResourceAgentEvidenceModeDisabled_024, true, "active", "disabled"},
	{ResourceAgentEvidenceModeActive_024, ResourceAgentEvidenceModeRetired_024, true, "active", "retired"},
	{ResourceAgentEvidenceModePassive_024, ResourceAgentEvidenceModeActive_024, true, "passive", "active"},
	{ResourceAgentEvidenceModePassive_024, ResourceAgentEvidenceModeDegraded_024, true, "passive", "degraded"},
	{ResourceAgentEvidenceModePassive_024, ResourceAgentEvidenceModeDisabled_024, true, "passive", "disabled"},
	{ResourceAgentEvidenceModePassive_024, ResourceAgentEvidenceModeRetired_024, false, "passive", "retired"},
	{ResourceAgentEvidenceModeDegraded_024, ResourceAgentEvidenceModeActive_024, true, "degraded", "active"},
	{ResourceAgentEvidenceModeDegraded_024, ResourceAgentEvidenceModePassive_024, true, "degraded", "passive"},
	{ResourceAgentEvidenceModeDegraded_024, ResourceAgentEvidenceModeDisabled_024, true, "degraded", "disabled"},
	{ResourceAgentEvidenceModeDegraded_024, ResourceAgentEvidenceModeRetired_024, false, "degraded", "retired"},
	{ResourceAgentEvidenceModeDisabled_024, ResourceAgentEvidenceModeActive_024, true, "disabled", "active"},
	{ResourceAgentEvidenceModeDisabled_024, ResourceAgentEvidenceModeRetired_024, true, "disabled", "retired"},
	{ResourceAgentEvidenceModeDisabled_024, ResourceAgentEvidenceModePassive_024, false, "disabled", "passive"},
	{ResourceAgentEvidenceModeDisabled_024, ResourceAgentEvidenceModeDegraded_024, false, "disabled", "degraded"},
	{ResourceAgentEvidenceModeRetired_024, ResourceAgentEvidenceModeActive_024, false, "retired", "active"},
	{ResourceAgentEvidenceModeRetired_024, ResourceAgentEvidenceModePassive_024, false, "retired", "passive"},
	{ResourceAgentEvidenceModeRetired_024, ResourceAgentEvidenceModeDegraded_024, false, "retired", "degraded"},
	{ResourceAgentEvidenceModeRetired_024, ResourceAgentEvidenceModeDisabled_024, false, "retired", "disabled"},
	{ResourceAgentEvidenceModeRetired_024, ResourceAgentEvidenceModeRetired_024, false, "retired", "retired"},
}

// ---------------------------------------------------------------------------
//  Degraded rules
// ---------------------------------------------------------
