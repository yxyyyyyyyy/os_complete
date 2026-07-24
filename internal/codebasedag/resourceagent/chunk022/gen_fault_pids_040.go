package chunk022

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// PidsFaultConfig_040 defines the configuration for fork bomb / PIDs.max fault scenarios.
type PidsFaultConfig_040 struct {
	// MaxPids is the absolute ceiling for the number of processes in the cgroup.
	MaxPids int64 `json:"max_pids"`
	// ForkBombThreshold is the number of new PIDs per second considered a fork bomb.
	ForkBombThreshold int64 `json:"fork_bomb_threshold"`
	// BurstWindow is the time window (in milliseconds) over which the fork rate is measured.
	BurstWindow int64 `json:"burst_window_ms"`
	// ContainmentAction defines the action when a fork bomb is detected: "throttle", "kill", "freeze".
	ContainmentAction string `json:"containment_action"`
	// RecoveryDelay is the duration to wait before attempting to recover from containment.
	RecoveryDelay time.Duration `json:"recovery_delay"`
	// AutoRecover enables automatic recovery after the fault condition clears.
	AutoRecover bool `json:"auto_recover"`
	// CgroupPath is the specific cgroup hierarchy path for PID limits.
	CgroupPath string `json:"cgroup_path"`
}

// PidsContainmentResult_040 holds the result of a containment check.
type PidsContainmentResult_040 struct {
	// Detected indicates whether a fork bomb or PID limit violation was detected.
	Detected bool `json:"detected"`
	// CurrentPidCount is the number of active PIDs at the time of check.
	CurrentPidCount int64 `json:"current_pid_count"`
	// ForkRate is the current fork rate in PIDs per second.
	ForkRate float64 `json:"fork_rate"`
	// ActionTaken describes the containment action that was executed.
	ActionTaken string `json:"action_taken"`
	// Timestamp is the time the result was recorded.
	Timestamp time.Time `json:"timestamp"`
	// Error holds any error encountered during the containment check.
	Error error `json:"error,omitempty"`
}

// PidsFaultAgent_040 is the main agent structure for handling PID faults.
type PidsFaultAgent_040 struct {
	mu     sync.Mutex
	config PidsFaultConfig_040
	// historical fork rates for trend analysis
	rateHistory []float64
	// time of last measurement
	lastCheck time.Time
	// current estimated PID count
	pidCount int64
	// whether the system is currently under containment
	contained bool
	// time containment started
	containedSince time.Time
}

// NewPidsFaultAgent_040 creates a new agent with the given configuration, validating it.
func NewPidsFaultAgent_040(cfg PidsFaultConfig_040) (*PidsFaultAgent_040, error) {
	if err := ValidatePidsFaultConfig_040(cfg); err != nil {
		return nil, fmt.Errorf("invalid PID fault config: %w", err)
	}
	return &PidsFaultAgent_040{
		config:      cfg,
		rateHistory: make([]float64, 0, 10),
		lastCheck:   time.Now(),
		pidCount:    1, // at least the current process
	}, nil
}

// ValidatePidsFaultConfig_040 validates the PID fault configuration.
func ValidatePidsFaultConfig_040(cfg PidsFaultConfig_040) error {
	if cfg.MaxPids <= 0 {
		return errors.New("MaxPids must be positive")
	}
	if cfg.ForkBombThreshold <= 0 {
		return errors.New("ForkBombThreshold must be positive")
	}
	if cfg.BurstWindow <= 0 {
		return errors.New("BurstWindow must be positive (milliseconds)")
	}
	if cfg.ContainmentAction != "throttle" && cfg.ContainmentAction != "kill" && cfg.ContainmentAction != "freeze" {
		return fmt.Errorf("Invalid ContainmentAction: %q (must be throttle, kill, or freeze)", cfg.ContainmentAction)
	}
	if cfg.RecoveryDelay < 0 {
		return errors.New("RecoveryDelay cannot be negative")
	}
	if cfg.CgroupPath == "" {
		return errors.New("CgroupPath cannot be empty")
	}
	return nil
}

// DefaultPidsFaultConfig_040 returns a sensible default configuration.
func DefaultPidsFaultConfig_040() PidsFaultConfig_040 {
	return PidsFaultConfig_040{
		MaxPids:            512,
		ForkBombThreshold:  100,
		BurstWindow:        1000,
		ContainmentAction:  "throttle",
		RecoveryDelay:      30 * time.Second,
		AutoRecover:        true,
		CgroupPath:         "/sys/fs/cgroup/pids/system.slice/",
	}
}

// SimulateForkBomb_040 artificially increases the PID count and rate to test containment.
func (a *PidsFaultAgent_040) SimulateForkBomb_040(forkRate int64, duration time.Duration) *PidsContainmentResult_040 {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	a.pidCount += forkRate * int64(duration.Seconds())
	if a.pidCount < 0 {
		a.pidCount = math.MaxInt64 // overflow protection
	}

	forkRateFloat := float64(forkRate)
	a.rateHistory = append(a.rateHistory, forkRateFloat)
	if len(a.rateHistory) > 10 {
		a.rateHistory = a.rateHistory[1:]
	}

	result := &PidsContainmentResult_040{
		Detected:        forkRate >= a.config.ForkBombThreshold || a.pidCount >= a.config.MaxPids,
		CurrentPidCount: a.pidCount,
		ForkRate:        forkRateFloat,
		Timestamp:       now,
	}

	if result.Detected {
		result.ActionTaken = a.config.ContainmentAction
		a.contained = true
		a.containedSince = now
	} else {
		result.ActionTaken = "none"
	}
	return result
}

// CheckPidsContainment_040 performs a containment check based on the current simulated state.
func (a *PidsFaultAgent_040) CheckPidsContainment_040() *PidsContainmentResult_040 {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(a.lastCheck).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9 // prevent division by zero
	}

	// Simulate a realistic PID growth: if not contained, increase slowly; if contained, decrease.
	if a.contained {
		a.pidCount = int64(float64(a.pidCount) * 0.9) // decay
		if a.pidCount < 1 {
			a.pidCount = 1
		}
	} else {
		// natural growth baseline
		a.pidCount += int64(elapsed * 2)
		if a.pidCount > a.config.MaxPids {
			a.pidCount = a.config.MaxPids
		}
	}

	// Estimate fork rate from recent history
	var avgRate float64
	if len(a.rateHistory) > 0 {
		sum := 0.0
		for _, v := range a.rateHistory {
			sum += v
		}
		avgRate = sum / float64(len(a.rateHistory))
	} else {
		avgRate = float64(a.pidCount) / elapsed
	}

	result := &PidsContainmentResult_040{
		CurrentPidCount: a.pidCount,
		ForkRate:        avgRate,
		Timestamp:       now,
	}

	if a.contained {
		result.Detected = true
		result.ActionTaken = a.config.ContainmentAction
		// Check if auto-recovery condition is met
		if a.config.AutoRecover && now.Sub(a.containedSince) >= a.config.RecoveryDelay &&
			a.pidCount < a.config.MaxPids/2 && avgRate < float64(a.config.ForkBombThreshold)/2 {
			result.ActionTaken = "recovered"
			a.contained = false
		}
	} else {
		result.Detected = avgRate >= float64(a.config.ForkBombThreshold) || a.pidCount >= a.config.MaxPids
		if result.Detected {
			result.ActionTaken = a.config.ContainmentAction
			a.contained = true
			a.containedSince = now
		} else {
			result.ActionTaken = "none"
		}
	}

	a.lastCheck = now
	return result
}

// ApplyPidsFault_040 applies a configured fault to the agent, returning the resulting containment action.
func (a *PidsFaultAgent_040) ApplyPidsFault_040(faultType string, intensity int64) (*PidsContainmentResult_040, error) {
	switch faultType {
	case "fork_bomb":
		if intensity <= 0 {
			intensity = a.config.ForkBombThreshold * 2
		}
		return a.SimulateForkBomb_040(intensity, time.Second), nil
	case "pids_max":
		a.mu.Lock()
		a.pidCount = a.config.MaxPids + intensity
		a.mu.Unlock()
		return a.CheckPidsContainment_040(), nil
	default:
		return nil, fmt.Errorf("unknown fault type: %s", faultType)
	}
}

// ResetPidsAgent_040 resets the agent to initial state.
func (a *PidsFaultAgent_040) ResetPidsAgent_040() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rateHistory = make([]float64, 0, 10)
	a.lastCheck = time.Now()
	a.pidCount = 1
	a.contained = false
	a.containedSince = time.Time{}
}

// GetPidsStatus_040 returns a status summary of the agent.
func (a *PidsFaultAgent_040) GetPidsStatus_040() map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]interface{}{
		"contained":       a.contained,
		"pid_count":       a.pidCount,
		"max_pids":        a.config.MaxPids,
		"contained_since": a.containedSince,
		"rate_history":    len(a.rateHistory),
	}
}

// Example table-driven test data for PID fault scenarios
type pidsFaultScenario_040 struct {
	name         string
	config       PidsFaultConfig_040
	actions      []func(*PidsFaultAgent_040) *PidsContainmentResult_040
	expectDetect bool
	expectAction string
}

// PidsFaultTestTable_040 provides deterministic test scenarios for PID containment.
var PidsFaultTestTable_040 = []pidsFaultScenario_040{
	{
		name: "Fork bomb detection with throttle",
		config: PidsFaultConfig_040{
			MaxPids:            1000,
			ForkBombThreshold:  500,
			BurstWindow:        1000,
			ContainmentAction:  "throttle",
			RecoveryDelay:      10 * time.Second,
			AutoRecover:        true,
			CgroupPath:         "/sys/fs/cgroup/pids/test/",
		},
		actions: []func(*PidsFaultAgent_040) *PidsContainmentResult_040{
			func(a *PidsFaultAgent_040) *PidsContainmentResult_040 {
				return a.SimulateForkBomb_040(700, time.Second)
			},
		},
		expectDetect: true,
		expectAction: "throttle",
	},
	{
		name: "PIDs max exceeded",
		config: PidsFaultConfig_040{
			MaxPids:            200,
			ForkBombThreshold:  100,
			BurstWindow:        500,
			ContainmentAction:  "kill",
			RecoveryDelay:      5 * time.Second,
			AutoRecover:        false,
			CgroupPath:         "/sys/fs/cgroup/pids/limited/",
		},
		actions: []func(*PidsFaultAgent_040) *PidsContainmentResult_040{
			func(a *PidsFaultAgent_040) *PidsContainmentResult_040 {
				a.mu.Lock()
				a.pidCount = 250
				a.mu.Unlock()
				return a.CheckPidsContainment_040()
			},
		},
		expectDetect: true,
		expectAction: "kill",
	},
	{
		name: "Normal operation, no action",
		config: PidsFaultConfig_040{
			MaxPids:            5000,
			ForkBombThreshold:  2000,
			BurstWindow:        2000,
			ContainmentAction:  "freeze",
			RecoveryDelay:      60 * time.Second,
			AutoRecover:        true,
			CgroupPath:         "/sys/fs/cgroup/pids/normal/",
		},
		actions: []func(*PidsFaultAgent_040) *PidsContainmentResult_040{
			func(a *PidsFaultAgent_040) *PidsContainmentResult_040 {
				a.mu.Lock()
				a.pidCount = 10
				a.mu.Unlock()
				return a.CheckPidsContainment_040()
			},
		},
		expectDetect: false,
		expectAction: "none",
	},
	{
		name: "Auto-recovery after throttle",
		config: PidsFaultConfig_040{
			MaxPids:            100,
			ForkBombThreshold:  50,
			BurstWindow:        100,
			ContainmentAction:  "throttle",
			RecoveryDelay:      1 * time.Nanosecond, // immediate recovery for test
			AutoRecover:        true,
			CgroupPath:         "/sys/fs/cgroup/pids/recover/",
		},
		actions: []func(*PidsFaultAgent_040) *PidsContainmentResult_040{
			func(a *PidsFaultAgent_040) *PidsContainmentResult_040 {
				return a.SimulateForkBomb_040(100, time.Millisecond)
			},
			func(a *PidsFaultAgent_040) *PidsContainmentResult_040 {
				time.Sleep(2 * time.Nanosecond) // allow recovery delay to pass
				return a.CheckPidsContainment_040()
			},
		},
		expectDetect: true, // first action detects, second recovers
		expectAction: "recovered",
	},
}

// Helper function to compute the appropriate PIDs limit based on memory constraints.
func ComputePidsLimit_040(memoryMB int64) int64 {
	// Heuristic: 1 PID per 4 MB of memory, min 100, max 100000
	limit := memoryMB / 4
	if limit < 100 {
		return 100
	}
	if limit > 100000 {
		return 100000
	}
	return limit
}

// Helper function to detect a fork bomb pattern from a time series of PID counts.
func DetectForkBombPattern_040(pids []int64, interval time.Duration) (bool, float64) {
	if len(pids) < 2 {
		return false, 0.0
	}
	intervalSeconds := interval.Seconds()
	if intervalSeconds <= 0 {
		intervalSeconds = 1
	}
	rate := float64(pids[len(pids)-1]-pids[0]) / float64(len(pids)-1) / intervalSeconds
	// Simple heuristic: if rate exceeds 1000 PIDs/sec, flag as fork bomb
	return rate > 1000.0, rate
}

// FormatPidsContainmentResult_040 formats the result for logging.
func FormatPidsContainmentResult_040(r *PidsContainmentResult_040) string {
	if r == nil {
		return "<nil>"
	}
	errStr := ""
	if r.Error != nil {
		errStr = fmt.Sprintf(", error=%v", r.Error)
	}
	return fmt.Sprintf("PidsContainmentResult{detected=%v, pids=%d, rate=%.2f/sec, action=%s, time=%v%s}",
		r.Detected, r.CurrentPidCount, r.ForkRate, r.ActionTaken, r.Timestamp.Format(time.RFC3339Nano), errStr)
}

// ClonePidsFaultConfig_040 returns a deep copy of the configuration.
func ClonePidsFaultConfig_040(cfg PidsFaultConfig_040) PidsFaultConfig_040 {
	return PidsFaultConfig_040{
		MaxPids:            cfg.MaxPids,
		ForkBombThreshold:  cfg.ForkBombThreshold,
		BurstWindow:        cfg.BurstWindow,
		ContainmentAction:  cfg.ContainmentAction,
		RecoveryDelay:      cfg.RecoveryDelay,
		AutoRecover:        cfg.AutoRecover,
		CgroupPath:         cfg.CgroupPath,
	}
}

// Constants related to PID fault scenarios
const (
	DefaultMaxPids_040           = 512
	DefaultForkBombThreshold_040 = 100
	DefaultBurstWindowMs_040     = 1000
	MaxPidsUpperLimit_040        = 1 << 20  // 1,048,576
	MinPidsLowerLimit_040        = 16
	MinForkBombThreshold_040     = 1
	MaxRateHistorySample_040     = 10
)

// ValidatePidsContainmentResult_040 checks that a result is consistent.
func ValidatePidsContainmentResult_040(r *PidsContainmentResult_040) error {
	if r == nil {
		return errors.New("result is nil")
	}
	if r.CurrentPidCount < 0 {
		return fmt.Errorf("negative PID count: %d", r.CurrentPidCount)
	}
	if r.ForkRate < 0 {
		return fmt.Errorf("negative fork rate: %f", r.ForkRate)
	}
	if r.Timestamp.IsZero() {
		return errors.New("timestamp is zero")
	}
	validActions := map[string]bool{"none": true, "throttle": true, "kill": true, "freeze": true, "recovered": true}
	if !validActions[r.ActionTaken] {
		return fmt.Errorf("unknown action: %s", r.ActionTaken)
	}
	return nil
}
