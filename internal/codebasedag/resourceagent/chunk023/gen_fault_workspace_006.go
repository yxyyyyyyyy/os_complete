package chunk023

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DestructFaultType defines the category of workspace destruction fault.
type DestructFaultType int

const (
	DestructFaultTypeNone              DestructFaultType = iota
	DestructFaultTypeMissingTempRoot   DestructFaultType = iota
	DestructFaultTypePermissionDenied  DestructFaultType = iota
	DestructFaultTypeSymlinkChain      DestructFaultType = iota
	DestructFaultTypeNonEmptyRoot      DestructFaultType = iota
	DestructFaultTypeCorruptedMarker   DestructFaultType = iota
	DestructFaultTypeGuardStale        DestructFaultType = iota
	DestructFaultTypeInnerFileLock     DestructFaultType = iota
	DestructFaultTypeOverlayMount      DestructFaultType = iota
	DestructFaultTypeResourceBusy      DestructFaultType = iota
)

// destructFaultTypeNames maps fault types to human-readable names.
var destructFaultTypeNames = map[DestructFaultType]string{
	DestructFaultTypeNone:             "none",
	DestructFaultTypeMissingTempRoot:  "missing_temp_root",
	DestructFaultTypePermissionDenied: "permission_denied",
	DestructFaultTypeSymlinkChain:     "symlink_chain",
	DestructFaultTypeNonEmptyRoot:     "non_empty_root",
	DestructFaultTypeCorruptedMarker:  "corrupted_marker",
	DestructFaultTypeGuardStale:       "guard_stale",
	DestructFaultTypeInnerFileLock:    "inner_file_lock",
	DestructFaultTypeOverlayMount:     "overlay_mount",
	DestructFaultTypeResourceBusy:     "resource_busy",
}

// String implements fmt.Stringer for DestructFaultType.
func (d DestructFaultType) String() string {
	if name, ok := destructFaultTypeNames[d]; ok {
		return name
	}
	return fmt.Sprintf("DestructFaultType(%d)", d)
}

// TempRootGuard_006 specifies constraints on the temporary root directory.
type TempRootGuard_006 struct {
	// Path expected absolute path to the temp root.
	Path string
	// EnsureEmpty is true if the temp root must be empty or non‑existent.
	EnsureEmpty bool
	// EnsureWritable is true if the temp root must be writable.
	EnsureWritable bool
	// MountProtection checks that no unwanted mounts exist inside.
	MountProtection bool
	// SymlinkProtection checks for dangerous symlinks leading outside.
	SymlinkProtection bool
	// MarkedAsTempRoot requires a specific marker file to verify identity.
	MarkedAsTempRoot bool
}

// ValidateGuard_006 validates a TempRootGuard_006 for consistency.
// Returns an error if the guard is invalid (e.g. empty path).
func ValidateGuard_006(g *TempRootGuard_006) error {
	if g == nil {
		return errors.New("temp root guard is nil")
	}
	cleaned := filepath.Clean(g.Path)
	if !filepath.IsAbs(cleaned) {
		return fmt.Errorf("temp root path %q is not absolute", g.Path)
	}
	if len(cleaned) < 2 {
		return errors.New("temp root path too short after cleaning")
	}
	return nil
}

// WorkspaceDestructFaultConfig describes a complete fault injection for workspace destruction.
type WorkspaceDestructFaultConfig struct {
	// Type specifies the fault scenario.
	Type DestructFaultType
	// TempRootGuard_006 holds the guard that will be tested during destruction.
	TempRootGuard_006 TempRootGuard_006
	// SimulateFailure indicates whether the fault should actually trigger an error.
	SimulateFailure bool
	// ExpectedError is the error message that should be produced if SimulateFailure is true.
	ExpectedError string
}

// ValidateDestructFaultConfig_006 validates a WorkspaceDestructFaultConfig.
// It checks that the fault type is known and that the temp root guard is valid.
func ValidateDestructFaultConfig_006(cfg *WorkspaceDestructFaultConfig) error {
	if cfg == nil {
		return errors.New("destruct fault config is nil")
	}
	if _, ok := destructFaultTypeNames[cfg.Type]; !ok {
		return fmt.Errorf("unknown destruct fault type: %v", cfg.Type)
	}
	if err := ValidateGuard_006(&cfg.TempRootGuard_006); err != nil {
		return fmt.Errorf("invalid temp root guard: %w", err)
	}
	if cfg.SimulateFailure && cfg.ExpectedError == "" {
		return errors.New("simulate failure enabled but expected error is empty")
	}
	return nil
}

// DestructScenario captures a deterministic test case for workspace destruction faults.
type DestructScenario struct {
	// Name describes the scenario.
	Name string
	// Config is the fault configuration to apply.
	Config WorkspaceDestructFaultConfig
	// ShouldFail is true if the scenario is expected to produce an error.
	ShouldFail bool
	// ExpectedErrorSubstr is a substring that the error message should contain (if ShouldFail).
	ExpectedErrorSubstr string
}

// DefaultDestructScenarios_006 returns a slice of deterministic test scenarios
// for workspace destruction fault handling.
func DefaultDestructScenarios_006() []DestructScenario {
	return []DestructScenario{
		{
			Name: "NoFault_ValidGuard",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeNone,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      true,
					EnsureWritable:   true,
					MountProtection:  true,
					SymlinkProtection: true,
					MarkedAsTempRoot: true,
				},
				SimulateFailure: false,
				ExpectedError:   "",
			},
			ShouldFail:          false,
			ExpectedErrorSubstr: "",
		},
		{
			Name: "MissingTempRoot",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeMissingTempRoot,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/nonexistent/path",
					EnsureEmpty:      true,
					EnsureWritable:   true,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "temp root does not exist",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "temp root does not exist",
		},
		{
			Name: "PermissionDenied_OnWritableCheck",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypePermissionDenied,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/readonly-root",
					EnsureEmpty:      true,
					EnsureWritable:   true,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "permission denied: cannot write to temp root",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "permission denied",
		},
		{
			Name: "SymlinkChain_EscapesRoot",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeSymlinkChain,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: true,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "symlink chain detected: resolves outside temp root",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "symlink chain",
		},
		{
			Name: "NonEmptyRoot_NotAllowed",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeNonEmptyRoot,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      true,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "temp root is not empty",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "not empty",
		},
		{
			Name: "CorruptedMarkerFile",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeCorruptedMarker,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      true,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: true,
				},
				SimulateFailure: true,
				ExpectedError:   "marker file '.workspace' is corrupted or missing",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "marker file",
		},
		{
			Name: "GuardStale_ProcessStillRunning",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeGuardStale,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: true,
				},
				SimulateFailure: true,
				ExpectedError:   "temp root guard is stale: workspace still in use",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "stale",
		},
		{
			Name: "InnerFileLock_Conflict",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeInnerFileLock,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "file lock held inside workspace root",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "file lock",
		},
		{
			Name: "OverlayMountPresent",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeOverlayMount,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  true,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "overlay mount detected inside temp root",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "overlay mount",
		},
		{
			Name: "ResourceBusy_ByAnotherProcess",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeResourceBusy,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "/tmp/workspace-root",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: true,
				ExpectedError:   "resource busy: workspace cannot be destroyed",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "resource busy",
		},
		{
			Name: "ValidWithRelativePath_ShouldFail",
			Config: WorkspaceDestructFaultConfig{
				Type: DestructFaultTypeNone,
				TempRootGuard_006: TempRootGuard_006{
					Path:             "relative/path",
					EnsureEmpty:      false,
					EnsureWritable:   false,
					MountProtection:  false,
					SymlinkProtection: false,
					MarkedAsTempRoot: false,
				},
				SimulateFailure: false,
				ExpectedError:   "",
			},
			ShouldFail:          true,
			ExpectedErrorSubstr: "not absolute",
		},
	}
}

// SimulateDestructFault_006 runs a fault simulation based on the config.
// It returns an error if the fault would cause destruction to fail.
func SimulateDestructFault_006(cfg *WorkspaceDestructFaultConfig) error {
	if err := ValidateDestructFaultConfig_006(cfg); err != nil {
		return err
	}
	if !cfg.SimulateFailure {
		return nil
	}
	// Simulate logical checks based on fault type.
	// In a real implementation this would do actual filesystem checks.
	switch cfg.Type {
	case DestructFaultTypeMissingTempRoot:
		// Check if path exists.
		if _, err := os.Stat(cfg.TempRootGuard_006.Path); os.IsNotExist(err) {
			return errors.New(cfg.ExpectedError)
		}
		return nil
	case DestructFaultTypePermissionDenied:
		// Simulate permission check. (In real code, would use access(2))
		info, err := os.Stat(cfg.TempRootGuard_006.Path)
		if err != nil {
			return err
		}
		if err := checkWritable_006(info, cfg.TempRootGuard_006.Path); err != nil {
			return errors.New(cfg.ExpectedError)
		}
		return nil
	case DestructFaultTypeSymlinkChain:
		// Simulate symlink resolution. (In real code, would use EvalSymlinks)
		resolved, err := filepath.EvalSymlinks(cfg.TempRootGuard_006.Path)
		if err != nil {
			return err
		}
		cleanRoot := filepath.Clean(cfg.TempRootGuard_006.Path)
		if !strings.HasPrefix(resolved, cleanRoot) {
			return errors.New(cfg.ExpectedError)
		}
		return nil
	case DestructFaultTypeNonEmptyRoot:
		// Simulate empty check.
		entries, err := os.ReadDir(cfg.TempRootGuard_006.Path)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return errors.New(cfg.ExpectedError)
		}
		return nil
	case DestructFaultTypeCorruptedMarker:
		// Simulate marker file check.
		markerPath := filepath.Join(cfg.TempRootGuard_006.Path, ".workspace")
		if _, err := os.Stat(markerPath); os.IsNotExist(err) {
			return errors.New(cfg.ExpectedError)
		}
		return nil
	case DestructFaultTypeGuardStale:
		// Simulate guard staleness check (always fails if configured).
		return errors.New(cfg.ExpectedError)
	case DestructFaultTypeInnerFileLock:
		// Simulate file lock detection.
		return errors.New(cfg.ExpectedError)
	case DestructFaultTypeOverlayMount:
		// Simulate mount detection.
		return errors.New(cfg.ExpectedError)
	case DestructFaultTypeResourceBusy:
		// Simulate resource busy.
		return errors.New(cfg.ExpectedError)
	default:
		return nil
	}
}

// checkWritable_006 is a helper that checks whether the file info indicates writability.
// In a real implementation it would check permissions.
func checkWritable_006(info os.FileInfo, path string) error {
	// Placeholder: always succeed for valid paths.
	return nil
}
