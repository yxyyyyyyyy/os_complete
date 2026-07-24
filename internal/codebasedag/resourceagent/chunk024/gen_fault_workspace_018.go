package chunk024

import (
	"errors"
	"fmt"
)

// ResourceAgentWorkspaceDestructionFault018v2 defines a fault scenario for workspace destruction
// involving temporary root filesystem guards.
type ResourceAgentWorkspaceDestructionFault018v2 struct {
	WorkspaceID string
	GuardType   ResourceAgentTempRootGuardType018v2
	Action      ResourceAgentGuardAction018v2
	Severity    ResourceAgentFaultSeverity018v2
	Extra       map[string]string
}

// ResourceAgentTempRootGuardType018v2 categorizes the kind of guard applied to a temp-root.
type ResourceAgentTempRootGuardType018v2 int

const (
	GuardMustBeEmpty ResourceAgentTempRootGuardType018v2 = iota
	GuardNoProcesses
	GuardNotMounted
	GuardOwnerMatch
	GuardPermissionSet
	GuardNoSymlinks
	GuardMaxSize
	GuardInodeQuota
)

// ResourceAgentGuardAction018v2 specifies what action the guard should take if a condition fails.
type ResourceAgentGuardAction018v2 int

const (
	ActionBlock   ResourceAgentGuardAction018v2 = iota
	ActionWarn
	ActionForce
	ActionCleanup
)

// ResourceAgentFaultSeverity018v2 indicates the severity of a workspace destruction fault.
type ResourceAgentFaultSeverity018v2 int

const (
	SeverityLow018v2      ResourceAgentFaultSeverity018v2 = iota
	SeverityMedium018v2
	SeverityHigh018v2
	SeverityCritical018v2
)

// ResourceAgentTempRootGuard018v2 represents a guard policy for a temp root directory.
type ResourceAgentTempRootGuard018v2 struct {
	GuardType    ResourceAgentTempRootGuardType018v2
	Action       ResourceAgentGuardAction018v2
	Threshold    int64
	Owner        string
	Permissions  int
	ExtraOptions map[string]string
}

// ResourceAgentDestructionFaultConfig018v2 holds configuration parameters for fault evaluation.
type ResourceAgentDestructionFaultConfig018v2 struct {
	WorkspaceID        string
	RequiredGuards     []ResourceAgentTempRootGuard018v2
	AllowForceOverride bool
	DefaultAction      ResourceAgentGuardAction018v2
}

// PredefinedScenario018v2 stores a named test scenario and its expected guard outcome.
type PredefinedScenario018v2 struct {
	Name           string
	Fault          *ResourceAgentWorkspaceDestructionFault018v2
	ExpectedResult string // "pass", "warn", "block", "cleanup"
	ExpectedError  bool
}

var errValidation = errors.New("validation error")

// ValidateWorkspaceDestructionFault018v2 checks consistency of a destruction fault.
func ValidateWorkspaceDestructionFault018v2(fault *ResourceAgentWorkspaceDestructionFault018v2) error {
	if fault == nil {
		return fmt.Errorf("%w: fault must not be nil", errValidation)
	}
	if fault.WorkspaceID == "" {
		return fmt.Errorf("%w: WorkspaceID must not be empty", errValidation)
	}
	if fault.GuardType < GuardMustBeEmpty || fault.GuardType > GuardInodeQuota {
		return fmt.Errorf("%w: invalid GuardType value %d", errValidation, fault.GuardType)
	}
	if fault.Action < ActionBlock || fault.Action > ActionCleanup {
		return fmt.Errorf("%w: invalid Action value %d", errValidation, fault.Action)
	}
	if fault.Severity < SeverityLow018v2 || fault.Severity > SeverityCritical018v2 {
		return fmt.Errorf("%w: invalid Severity value %d", errValidation, fault.Severity)
	}
	if fault.Extra == nil {
		fault.Extra = make(map[string]string)
	}
	return nil
}

// NewWorkspaceDestructionFault018v2 creates a validated fault with defaults.
func NewWorkspaceDestructionFault018v2(workspaceID string, guardType ResourceAgentTempRootGuardType018v2,
	action ResourceAgentGuardAction018v2, severity ResourceAgentFaultSeverity018v2) (*ResourceAgentWorkspaceDestructionFault018v2, error) {
	fault := &ResourceAgentWorkspaceDestructionFault018v2{
		WorkspaceID: workspaceID,
		GuardType:   guardType,
		Action:      action,
		Severity:    severity,
		Extra:       make(map[string]string),
	}
	if err := ValidateWorkspaceDestructionFault018v2(fault); err != nil {
		return nil, err
	}
	return fault, nil
}

// ApplyGuard evaluates the fault against a guard configuration and returns the action to take.
// Check returns true if the guard condition is violated. On violation, the guard's own action is used;
// otherwise the config's default action is returned.
func (fault *ResourceAgentWorkspaceDestructionFault018v2) ApplyGuard(guard *ResourceAgentTempRootGuard018v2, config *ResourceAgentDestructionFaultConfig018v2) (ResourceAgentGuardAction018v2, error) {
	if guard == nil || config == nil {
		return ActionBlock, errors.New("guard or config is nil")
	}
	if fault.GuardType != guard.GuardType {
		return ActionWarn, fmt.Errorf("guard type mismatch: fault %d vs guard %d", fault.GuardType, guard.GuardType)
	}

	violation, err := guard.Check(fault)
	if err != nil {
		return ActionBlock, fmt.Errorf("guard check failed: %w", err)
	}

	if violation {
		// Apply the guard's action
		switch guard.Action {
		case ActionBlock:
			return ActionBlock, nil
		case ActionWarn:
			return ActionWarn, nil
		case ActionForce:
			return ActionForce, nil
		case ActionCleanup:
			_ = cleanupTempRoot(fault.WorkspaceID, guard)
			return ActionCleanup, nil
		default:
			return config.DefaultAction, nil
		}
	}
	// No violation -> allow with default action
	return config.DefaultAction, nil
}

// Check evaluates a single guard condition against the fault scenario.
// Returns true if the condition is violated (protection needed), false if satisfied.
func (guard *ResourceAgentTempRootGuard018v2) Check(fault *ResourceAgentWorkspaceDestructionFault018v2) (bool, error) {
	switch guard.GuardType {
	case GuardMustBeEmpty:
		if val, ok := fault.Extra["contains"]; ok && val != "" {
			return true, nil
		}
		return false, nil
	case GuardNoProcesses:
		if _, ok := fault.Extra["pids"]; ok {
			return true, nil
		}
		return false, nil
	case GuardNotMounted:
		if val, ok := fault.Extra["mounted"]; ok && val == "true" {
			return true, nil
		}
		return false, nil
	case GuardOwnerMatch:
		expected := guard.Owner
		actual, ok := fault.Extra["owner"]
		if !ok || actual != expected {
			return true, nil
		}
		return false, nil
	case GuardPermissionSet:
		expectedPerms := guard.Permissions
		actualPermsStr, ok := fault.Extra["permissions"]
		if !ok || actualPermsStr == "" {
			return true, nil
		}
		actualPerms := 0
		_, _ = fmt.Sscanf(actualPermsStr, "%o", &actualPerms)
		if actualPerms != expectedPerms {
			return true, nil
		}
		return false, nil
	case GuardNoSymlinks:
		if _, ok := fault.Extra["symlinks"]; ok {
			return true, nil
		}
		return false, nil
	case GuardMaxSize:
		threshold := guard.Threshold
		actualSizeStr, ok := fault.Extra["size"]
		if !ok {
			return false, nil
		}
		actualSize := int64(0)
		_, _ = fmt.Sscanf(actualSizeStr, "%d", &actualSize)
		if actualSize > threshold {
			return true, nil
		}
		return false, nil
	case GuardInodeQuota:
		threshold := guard.Threshold
		actualInodesStr, ok := fault.Extra["inodes"]
		if !ok {
			return false, nil
		}
		actualInodes := int64(0)
		_, _ = fmt.Sscanf(actualInodesStr, "%d", &actualInodes)
		if actualInodes > threshold {
			return true, nil
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown guard type %d", guard.GuardType)
	}
}

func cleanupTempRoot(workspaceID string, guard *ResourceAgentTempRootGuard018v2) error {
	_ = workspaceID
	_ = guard
	return nil
}

// ClassifyFault018v2 returns a severity label for a destruction fault.
func ClassifyFault018v2(fault *ResourceAgentWorkspaceDestructionFault018v2) string {
	switch fault.Severity {
	case SeverityLow018v2:
		return "informational"
	case SeverityMedium018v2:
		return "medium"
	case SeverityHigh018v2:
		return "high"
	case SeverityCritical018v2:
		return "critical"
	default:
		return "unknown"
	}
}

// IsTempRootGuarded018v2 reports whether a guard type is in the required list.
func IsTempRootGuarded018v2(config *ResourceAgentDestructionFaultConfig018v2, guardType ResourceAgentTempRootGuardType018v2) bool {
	for _, g := range config.RequiredGuards {
		if g.GuardType == guardType {
			return true
		}
	}
	return false
}

// GetPredefinedScenarios018v2 returns a table of deterministic test scenarios.
func GetPredefinedScenarios018v2() []PredefinedScenario018v2 {
	return []PredefinedScenario018v2{
		{
			Name: "empty_temp_root_block",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_001",
				GuardType:   GuardMustBeEmpty,
				Action:      ActionBlock,
				Severity:    SeverityHigh018v2,
				Extra:       map[string]string{},
			},
			ExpectedResult: "pass",
			ExpectedError:  false,
		},
		{
			Name: "empty_temp_root_contains_data",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_002",
				GuardType:   GuardMustBeEmpty,
				Action:      ActionBlock,
				Severity:    SeverityCritical018v2,
				Extra:       map[string]string{"contains": "data"},
			},
			ExpectedResult: "block",
			ExpectedError:  false,
		},
		{
			Name: "no_processes_fail",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_003",
				GuardType:   GuardNoProcesses,
				Action:      ActionWarn,
				Severity:    SeverityMedium018v2,
				Extra:       map[string]string{"pids": "123,456"},
			},
			ExpectedResult: "warn",
			ExpectedError:  false,
		},
		{
			Name: "not_mounted_ok",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_004",
				GuardType:   GuardNotMounted,
				Action:      ActionBlock,
				Severity:    SeverityLow018v2,
				Extra:       map[string]string{},
			},
			ExpectedResult: "pass",
			ExpectedError:  false,
		},
		{
			Name: "owner_mismatch",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_005",
				GuardType:   GuardOwnerMatch,
				Action:      ActionBlock,
				Severity:    SeverityHigh018v2,
				Extra:       map[string]string{},
			},
			ExpectedResult: "block",
			ExpectedError:  false,
		},
		{
			Name: "permission_wrong",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_006",
				GuardType:   GuardPermissionSet,
				Action:      ActionCleanup,
				Severity:    SeverityMedium018v2,
				Extra:       map[string]string{"permissions": "0755"},
			},
			ExpectedResult: "cleanup",
			ExpectedError:  false,
		},
		{
			Name: "symlink_present",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_007",
				GuardType:   GuardNoSymlinks,
				Action:      ActionForce,
				Severity:    SeverityCritical018v2,
				Extra:       map[string]string{"symlinks": "/foo"},
			},
			ExpectedResult: "force",
			ExpectedError:  false,
		},
		{
			Name: "size_exceeded",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_008",
				GuardType:   GuardMaxSize,
				Action:      ActionBlock,
				Severity:    SeverityHigh018v2,
				Extra:       map[string]string{"size": "1048577"},
			},
			ExpectedResult: "block",
			ExpectedError:  false,
		},
		{
			Name: "inode_quota_ok",
			Fault: &ResourceAgentWorkspaceDestructionFault018v2{
				WorkspaceID: "ws_009",
				GuardType:   GuardInodeQuota,
				Action:      ActionWarn,
				Severity:    SeverityLow018v2,
				Extra:       map[string]string{"inodes": "500"},
			},
			ExpectedResult: "pass",
			ExpectedError:  false,
		},
	}
}
