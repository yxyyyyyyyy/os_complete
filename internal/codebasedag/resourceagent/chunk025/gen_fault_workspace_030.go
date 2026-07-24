package chunk025

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ResourceAgentFaultSeverity030 represents the severity level of a fault.
type ResourceAgentFaultSeverity030 int

const (
	SeverityLow     ResourceAgentFaultSeverity030 = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// ============================================================================
// Types
// ============================================================================

// DestructionFaultType enumerates possible workspace destruction fault categories.
type DestructionFaultType int

const (
	FaultTypeUnknown DestructionFaultType = iota
	FaultTypeRemovalFailure
	FaultTypeGuardViolation
	FaultTypeResourceLeak
	FaultTypePartialDestruction
	FaultTypeTempRootCorruption
	FaultTypeConcurrencyRace
	FaultTypePermissionDenied
	FaultTypeDiskFull
	FaultTypeTimeoutExpired
)

func (d DestructionFaultType) String() string {
	switch d {
	case FaultTypeRemovalFailure:
		return "removal_failure"
	case FaultTypeGuardViolation:
		return "guard_violation"
	case FaultTypeResourceLeak:
		return "resource_leak"
	case FaultTypePartialDestruction:
		return "partial_destruction"
	case FaultTypeTempRootCorruption:
		return "temp_root_corruption"
	case FaultTypeConcurrencyRace:
		return "concurrency_race"
	case FaultTypePermissionDenied:
		return "permission_denied"
	case FaultTypeDiskFull:
		return "disk_full"
	case FaultTypeTimeoutExpired:
		return "timeout_expired"
	default:
		return "unknown"
	}
}

// FaultSeverity is an alias for the package-level severity type to avoid redeclaration.
type FaultSeverity = ResourceAgentFaultSeverity030

// TempRootGuard represents a guard that prevents untimely removal of a temporary root.
type TempRootGuard struct {
	GuardID   string    `json:"guard_id"`
	Workspace string    `json:"workspace"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Owner     string    `json:"owner"`
	Reason    string    `json:"reason"`
}

// WorkspaceDestructionConfig holds parameters for workspace destruction operations.
type WorkspaceDestructionConfig struct {
	TempRootPath     string        `json:"temp_root_path"`
	ForceRemoval     bool          `json:"force_removal"`
	RetryCount       int           `json:"retry_count"`
	RetryDelay       time.Duration `json:"retry_delay"`
	GuardTimeout     time.Duration `json:"guard_timeout"`
	CleanupSubdirs   bool          `json:"cleanup_subdirs"`
	KeepPartialLogs  bool          `json:"keep_partial_logs"`
	MaxOpenFiles     int           `json:"max_open_files"`
	AllowPartialFail bool          `json:"allow_partial_fail"`
}

// DestructionFault describes a fault that occurred during workspace removal.
type DestructionFault struct {
	Type         DestructionFaultType `json:"fault_type"`
	Severity     FaultSeverity        `json:"severity"`
	AffectedPath string               `json:"affected_path"`
	Message      string               `json:"message"`
	Timestamp    time.Time            `json:"timestamp"`
	Recoverable  bool                 `json:"recoverable"`
}

// DestructionScenarioDescriptor is used for table‑driven testing and documentation.
type DestructionScenarioDescriptor struct {
	Name              string
	FaultType         DestructionFaultType
	ExpectedSeverity  FaultSeverity
	GuardActive       bool
	ForceRemoval      bool
	AffectedSubDir    string
}

// ============================================================================
// Constants and table‑driven data
// ============================================================================

// DefaultRetryDelay is the default pause between destruction retry attempts.
const DefaultRetryDelay_030 = 500 * time.Millisecond

// DefaultGuardTimeout is the default maximum lifetime of a temp root guard.
const DefaultGuardTimeout_030 = 30 * time.Minute

// DestructionScenarioTable_030 provides a deterministic set of fault scenarios.
var DestructionScenarioTable_030 = []DestructionScenarioDescriptor{
	{
		Name:              "guard_active_no_force",
		FaultType:         FaultTypeGuardViolation,
		ExpectedSeverity:  SeverityHigh,
		GuardActive:       true,
		ForceRemoval:      false,
		AffectedSubDir:    "guard_active_dir",
	},
	{
		Name:              "guard_active_with_force",
		FaultType:         FaultTypeRemovalFailure,
		ExpectedSeverity:  SeverityMedium,
		GuardActive:       true,
		ForceRemoval:      true,
		AffectedSubDir:    "force_removal_dir",
	},
	{
		Name:              "permission_denied_scenario",
		FaultType:         FaultTypePermissionDenied,
		ExpectedSeverity:  SeverityHigh,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "no_permissions_dir",
	},
	{
		Name:              "disk_full_during_removal",
		FaultType:         FaultTypeDiskFull,
		ExpectedSeverity:  SeverityCritical,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "disk_full_dir",
	},
	{
		Name:              "partial_removal_race",
		FaultType:         FaultTypeConcurrencyRace,
		ExpectedSeverity:  SeverityMedium,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "race_dir",
	},
	{
		Name:              "temp_root_corrupted",
		FaultType:         FaultTypeTempRootCorruption,
		ExpectedSeverity:  SeverityCritical,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "corrupted_root",
	},
	{
		Name:              "timeout_while_removing",
		FaultType:         FaultTypeTimeoutExpired,
		ExpectedSeverity:  SeverityHigh,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "large_dir_timeout",
	},
	{
		Name:              "resource_leak_after_failure",
		FaultType:         FaultTypeResourceLeak,
		ExpectedSeverity:  SeverityMedium,
		GuardActive:       false,
		ForceRemoval:      false,
		AffectedSubDir:    "leak_dir",
	},
}

// TempRootGuardTestTable_030 holds sample guards for validation and simulation.
var TempRootGuardTestTable_030 = []TempRootGuard{
	{
		GuardID:   "guard-001",
		Workspace: "ws-alpha",
		Active:    true,
		CreatedAt: time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC),
		Owner:     "infra-bot",
		Reason:    "scheduled maintenance",
	},
	{
		GuardID:   "guard-002",
		Workspace: "ws-beta",
		Active:    false,
		CreatedAt: time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 3, 1, 12, 30, 0, 0, time.UTC),
		Owner:     "sre-team",
		Reason:    "manual hold",
	},
	{
		GuardID:   "guard-003",
		Workspace: "ws-gamma",
		Active:    true,
		CreatedAt: time.Date(2025, 3, 1, 14, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 3, 1, 14, 45, 0, 0, time.UTC),
		Owner:     "ci-pipeline",
		Reason:    "deployment in progress",
	},
	{
		GuardID:   "guard-004",
		Workspace: "ws-delta",
		Active:    true,
		CreatedAt: time.Date(2025, 3, 1, 16, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 3, 1, 16, 20, 0, 0, time.UTC),
		Owner:     "test-runner",
		Reason:    "integration test",
	},
	{
		GuardID:   "guard-005",
		Workspace: "ws-epsilon",
		Active:    false,
		CreatedAt: time.Date(2025, 3, 2, 8, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 3, 2, 8, 15, 0, 0, time.UTC),
		Owner:     "backup-agent",
		Reason:    "snapshot operation",
	},
}

// FaultSeverityMapping_030 maps DestructionFaultType to its default severity.
var FaultSeverityMapping_030 = map[DestructionFaultType]FaultSeverity{
	FaultTypeRemovalFailure:    SeverityHigh,
	FaultTypeGuardViolation:    SeverityHigh,
	FaultTypeResourceLeak:      SeverityMedium,
	FaultTypePartialDestruction: SeverityMedium,
	FaultTypeTempRootCorruption: SeverityCritical,
	FaultTypeConcurrencyRace:   SeverityMedium,
	FaultTypePermissionDenied:  SeverityHigh,
	FaultTypeDiskFull:          SeverityCritical,
	FaultTypeTimeoutExpired:    SeverityHigh,
}

// ============================================================================
// Constructors
// ============================================================================

// NewWorkspaceDestructionConfig_030 creates a new WorkspaceDestructionConfig with
// default values, then overrides with provided parameters. It validates the result.
func NewWorkspaceDestructionConfig_030(tempRootPath string, forceRemoval bool, retries int) (*WorkspaceDestructionConfig, error) {
	cfg := &WorkspaceDestructionConfig{
		TempRootPath:     tempRootPath,
		ForceRemoval:     forceRemoval,
		RetryCount:       retries,
		RetryDelay:       DefaultRetryDelay_030,
		GuardTimeout:     DefaultGuardTimeout_030,
		CleanupSubdirs:   true,
		KeepPartialLogs:  false,
		MaxOpenFiles:     256,
		AllowPartialFail: false,
	}
	if err := ValidateWorkspaceDestructionConfig_030(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DefaultWorkspaceDestructionConfig_030 returns a config with safe built‑in defaults.
func DefaultWorkspaceDestructionConfig_030() *WorkspaceDestructionConfig {
	return &WorkspaceDestructionConfig{
		TempRootPath:     "/tmp/aort-workspace",
		ForceRemoval:     false,
		RetryCount:       3,
		RetryDelay:       DefaultRetryDelay_030,
		GuardTimeout:     DefaultGuardTimeout_030,
		CleanupSubdirs:   true,
		KeepPartialLogs:  false,
		MaxOpenFiles:     256,
		AllowPartialFail: false,
	}
}

// ============================================================================
// Validation
// ============================================================================

// ValidateWorkspaceDestructionConfig_030 validates a WorkspaceDestructionConfig.
// It returns an error if any field is invalid.
func ValidateWorkspaceDestructionConfig_030(cfg *WorkspaceDestructionConfig) error {
	if cfg == nil {
		return errors.New("workspace destruction config is nil")
	}
	if strings.TrimSpace(cfg.TempRootPath) == "" {
		return errors.New("temp root path must not be empty")
	}
	if !strings.HasPrefix(cfg.TempRootPath, "/") {
		return fmt.Errorf("temp root path %q must be absolute", cfg.TempRootPath)
	}
	if cfg.RetryCount < 0 {
		return fmt.Errorf("retry count must be non-negative, got %d", cfg.RetryCount)
	}
	if cfg.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be non-negative, got %v", cfg.RetryDelay)
	}
	if cfg.GuardTimeout <= 0 {
		return fmt.Errorf("guard timeout must be positive, got %v", cfg.GuardTimeout)
	}
	if cfg.MaxOpenFiles < 1 {
		return fmt.Errorf("max open files must be at least 1, got %d", cfg.MaxOpenFiles)
	}
	return nil
}

// ValidateTempRootGuards_030 validates a list of TempRootGuard entries.
// It checks for duplicate GuardIDs, empty fields, and expired active guards.
func ValidateTempRootGuards_030(guards []TempRootGuard) error {
	if len(guards) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(guards))
	now := time.Now()
	for _, g := range guards {
		if strings.TrimSpace(g.GuardID) == "" {
			return errors.New("guard ID must not be empty")
		}
		if strings.TrimSpace(g.Workspace) == "" {
			return fmt.Errorf("guard %q has empty workspace", g.GuardID)
		}
		if strings.TrimSpace(g.Owner) == "" {
			return fmt.Errorf("guard %q has empty owner", g.GuardID)
		}
		if strings.TrimSpace(g.Reason) == "" {
			return fmt.Errorf("guard %q has empty reason", g.GuardID)
		}
		if seen[g.GuardID] {
			return fmt.Errorf("duplicate guard ID: %q", g.GuardID)
		}
		seen[g.GuardID] = true
		if g.Active && g.ExpiresAt.Before(now) {
			return fmt.Errorf("active guard %q is already expired", g.GuardID)
		}
		if g.CreatedAt.After(g.ExpiresAt) {
			return fmt.Errorf("guard %q created after it expires", g.GuardID)
		}
	}
	return nil
}

// ============================================================================
// Fault simulation and analysis
// ============================================================================

// SimulateDestructionFault_030 creates a DestructionFault based on a scenario descriptor.
// If descriptor is nil, it returns a default unknown fault.
func SimulateDestructionFault_030(desc *DestructionScenarioDescriptor) *DestructionFault {
	if desc == nil {
		return &DestructionFault{
			Type:         FaultTypeUnknown,
			Severity:     SeverityMedium,
			AffectedPath: "",
			Message:      "unknown fault scenario",
			Timestamp:    time.Now(),
			Recoverable:  false,
		}
	}
	fault := &DestructionFault{
		Type:         desc.FaultType,
		Severity:     desc.ExpectedSeverity,
		AffectedPath: desc.AffectedSubDir,
		Message:      fmt.Sprintf("simulated fault: %s", desc.Name),
		Timestamp:    time.Now(),
		Recoverable:  desc.ForceRemoval,
	}
	if desc.GuardActive && !desc.ForceRemoval {
		fault.Recoverable = false
	}
	return fault
}
