package chunk014

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CPUThrottleState041 represents the observed CPU throttle state for a given scenario.
type CPUThrottleState041 int

const (
	CPUThrottleUnknown041       CPUThrottleState041 = iota
	CPUThrottleNoThrottle041                         // No throttling occurred
	CPUThrottleLight041                              // Throttle < 10% of time
	CPUThrottleModerate041                           // Throttle 10-50% of time
	CPUThrottleSevere041                             // Throttle > 50% of time
)

// String returns human-readable representation of CPUThrottleState041.
func (c CPUThrottleState041) String() string {
	switch c {
	case CPUThrottleNoThrottle041:
		return "no-throttle"
	case CPUThrottleLight041:
		return "light-throttle"
	case CPUThrottleModerate041:
		return "moderate-throttle"
	case CPUThrottleSevere041:
		return "severe-throttle"
	default:
		return "unknown-throttle"
	}
}

// CPUSaturationScenario041 defines a CPU saturation scenario for fault-agent testing.
type CPUSaturationScenario041 struct {
	Name               string
	CPULoadPercent     int   // Target CPU load percentage (1-100)
	DurationSeconds    int   // Duration of the saturation in seconds (>=1)
	AllowedCores       int   // Number of cores allowed to use (0 means all)
	ExpectedThrottle   CPUThrottleState041
	Description        string
}

// Validate returns an error if the scenario has invalid fields.
func (s *CPUSaturationScenario041) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("scenario name cannot be empty")
	}
	if s.CPULoadPercent < 1 || s.CPULoadPercent > 100 {
		return fmt.Errorf("CPU load percent must be between 1 and 100, got %d", s.CPULoadPercent)
	}
	if s.DurationSeconds < 1 {
		return fmt.Errorf("duration seconds must be at least 1, got %d", s.DurationSeconds)
	}
	if s.AllowedCores < 0 {
		return fmt.Errorf("allowed cores cannot be negative, got %d", s.AllowedCores)
	}
	// Validate expected throttle value
	switch s.ExpectedThrottle {
	case CPUThrottleUnknown041, CPUThrottleNoThrottle041, CPUThrottleLight041, CPUThrottleModerate041, CPUThrottleSevere041:
		// valid
	default:
		return fmt.Errorf("invalid expected throttle state: %v", s.ExpectedThrottle)
	}
	return nil
}

// ValidateScenario041 is a standalone validation function that returns an error for invalid scenarios.
func ValidateScenario041(s CPUSaturationScenario041) error {
	return s.Validate()
}

// CPUThrottleObservation041 captures the observed throttle behavior during a fault injection.
type CPUThrottleObservation041 struct {
	ScenarioName       string
	StartTime          time.Time
	EndTime            time.Time
	ObservedThrottle   CPUThrottleState041
	ThrottlePercent    float64 // 0.0 - 100.0
	PeakCPULoad        float64 // highest observed CPU load percentage
	AverageCPULoad     float64
	Error              error // any error during observation
}

// NewCPUThrottleObservation041 creates a new observation with current time.
func NewCPUThrottleObservation041(scenarioName string) CPUThrottleObservation041 {
	return CPUThrottleObservation041{
		ScenarioName: scenarioName,
		StartTime:    time.Now(),
	}
}

// Finish sets the end time and computes throttle state based on throttle percent.
func (o *CPUThrottleObservation041) Finish(throttlePercent, peakLoad, avgLoad float64) {
	o.EndTime = time.Now()
	o.ThrottlePercent = throttlePercent
	o.PeakCPULoad = peakLoad
	o.AverageCPULoad = avgLoad
	o.ObservedThrottle = ThrottleStateFromPercent041(throttlePercent)
}

// ThrottleStateFromPercent041 returns a CPUThrottleState based on throttle percentage.
func ThrottleStateFromPercent041(pct float64) CPUThrottleState041 {
	switch {
	case pct < 0.0:
		return CPUThrottleUnknown041
	case pct < 1.0:
		return CPUThrottleNoThrottle041
	case pct < 10.0:
		return CPUThrottleLight041
	case pct <= 50.0:
		return CPUThrottleModerate041
	default:
		return CPUThrottleSevere041
	}
}

// ThrottleResult041 compares expected and observed throttle states.
type ThrottleResult041 int

const (
	ThrottleResultPass041 ThrottleResult041 = iota
	ThrottleResultFail041
	ThrottleResultInvalid041
)

// String returns text of throttle result.
func (t ThrottleResult041) String() string {
	switch t {
	case ThrottleResultPass041:
		return "PASS"
	case ThrottleResultFail041:
		return "FAIL"
	case ThrottleResultInvalid041:
		return "INVALID"
	default:
		return "UNKNOWN"
	}
}

// CompareThrottleResults041 compares expected vs observed throttle state.
// Returns Pass if they match, Fail if not, Invalid if either is Unknown.
func CompareThrottleResults041(expected, observed CPUThrottleState041) ThrottleResult041 {
	if expected == CPUThrottleUnknown041 || observed == CPUThrottleUnknown041 {
		return ThrottleResultInvalid041
	}
	if expected == observed {
		return ThrottleResultPass041
	}
	return ThrottleResultFail041
}

// CPUSaturationEvent041 represents a complete CPU saturation event with its scenario and observation.
type CPUSaturationEvent041 struct {
	Scenario    CPUSaturationScenario041
	Observation CPUThrottleObservation041
	Result      ThrottleResult041
}

// String returns a formatted summary of the event.
func (e *CPUSaturationEvent041) String() string {
	return fmt.Sprintf("Event[%s]: expected=%s, observed=%s, result=%s",
		e.Scenario.Name,
		e.Scenario.ExpectedThrottle.String(),
		e.Observation.ObservedThrottle.String(),
		e.Result.String(),
	)
}

// Validate returns an error if the event's scenario or observation is invalid.
func (e *CPUSaturationEvent041) Validate() error {
	if err := e.Scenario.Validate(); err != nil {
		return fmt.Errorf("invalid scenario: %w", err)
	}
	if e.Observation.ScenarioName != e.Scenario.Name {
		return fmt.Errorf("observation scenario name mismatch: %q vs %q", e.Observation.ScenarioName, e.Scenario.Name)
	}
	if e.Observation.StartTime.After(e.Observation.EndTime) {
		return errors.New("observation start time after end time")
	}
	return nil
}

// ValidateEvent041 validates a CPUSaturationEvent041 pointer.
func ValidateEvent041(e *CPUSaturationEvent041) error {
	if e == nil {
		return errors.New("event is nil")
	}
	return e.Validate()
}

// ScenarioTable041 is a deterministic table of predefined CPU saturation scenarios for testing.
var ScenarioTable041 = []CPUSaturationScenario041{
	{
		Name:             "idle-light-single-core",
		CPULoadPercent:   10,
		DurationSeconds:  30,
		AllowedCores:     1,
		ExpectedThrottle: CPUThrottleNoThrottle041,
		Description:      "Light load on a single core, no throttling expected.",
	},
	{
		Name:             "moderate-load-two-cores",
		CPULoadPercent:   40,
		DurationSeconds:  60,
		AllowedCores:     2,
		ExpectedThrottle: CPUThrottleLight041,
		Description:      "Moderate load on two cores, expected light throttling.",
	},
	{
		Name:             "high-load-all-cores",
		CPULoadPercent:   90,
		DurationSeconds:  120,
		AllowedCores:     0, // all cores
		ExpectedThrottle: CPUThrottleSevere041,
		Description:      "High load on all cores, severe throttling expected.",
	},
	{
		Name:             "burst-short-high",
		CPULoadPercent:   95,
		DurationSeconds:  5,
		AllowedCores:     4,
		ExpectedThrottle: CPUThrottleModerate041,
		Description:      "Short burst of very high load on 4 cores, moderate throttle due to short duration.",
	},
	{
		Name:             "low-load-long-duration",
		CPULoadPercent:   20,
		DurationSeconds:  300,
		AllowedCores:     0,
		ExpectedThrottle: CPUThrottleNoThrottle041,
		Description:      "Sustained low load over 5 minutes, no throttling.",
	},
	{
		Name:             "medium-load-asymmetric",
		CPULoadPercent:   70,
		DurationSeconds:  45,
		AllowedCores:     3,
		ExpectedThrottle: CPUThrottleModerate041,
		Description:      "Medium-high load on 3 cores, moderate throttle.",
	},
	{
		Name:             "extreme-load-single-core",
		CPULoadPercent:   100,
		DurationSeconds:  15,
		AllowedCores:     1,
		ExpectedThrottle: CPUThrottleSevere041,
		Description:      "100% load on single core short duration, severe throttle.",
	},
	{
		Name:             "minimal-load-test",
		CPULoadPercent:   1,
		DurationSeconds:  10,
		AllowedCores:     2,
		ExpectedThrottle: CPUThrottleNoThrottle041,
		Description:      "Minimal load (1%) on 2 cores, no throttle.",
	},
}

// GetScenarioByName041 returns a pointer to the scenario from ScenarioTable041 matching the name.
// Returns nil if not found.
func GetScenarioByName041(name string) *CPUSaturationScenario041 {
	for i := range ScenarioTable041 {
		if ScenarioTable041[i].Name == name {
			return &ScenarioTable041[i]
		}
	}
	return nil
}

// FilterScenariosByThrottle041 returns a slice of scenarios with the given expected throttle state.
func FilterScenariosByThrottle041(state CPUThrottleState041) []CPUSaturationScenario041 {
	var result []CPUSaturationScenario041
	for _, s := range ScenarioTable041 {
		if s.ExpectedThrottle == state {
			result = append(result, s)
		}
	}
	return result
}

// SortScenariosByLoad041 sorts the input slice by CPULoadPercent ascending.
func SortScenariosByLoad041(scenarios []CPUSaturationScenario041) {
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].CPULoadPercent < scenarios[j].CPULoadPercent
	})
}

// GenerateSaturationEvent041 creates a CPUSaturationEvent041 from a scenario, setting initial observation.
func GenerateSaturationEvent041(scenario CPUSaturationScenario041) (*CPUSaturationEvent041, error) {
	if err := scenario.Validate(); err != nil {
		return nil, err
	}
	obs := NewCPUThrottleObservation041(scenario.Name)
	return &CPUSaturationEvent041{
		Scenario:    scenario,
		Observation: obs,
		Result:      ThrottleResultInvalid041, // not yet compared
	}, nil
}

// EvaluateEvent041 sets the result field of the event by comparing expected and observed throttle.
func EvaluateEvent041(event *CPUSaturationEvent041) error {
	if event == nil {
		return errors.New("event is nil")
	}
	if event.Scenario.ExpectedThrottle == CPUThrottleUnknown041 || event.Observation.ObservedThrottle == CPUThrottleUnknown041 {
		event.Result = ThrottleResultInvalid041
	} else if event.Scenario.ExpectedThrottle == event.Observation.ObservedThrottle {
		event.Result = ThrottleResultPass041
	} else {
		event.Result = ThrottleResultFail041
	}
	return nil
}

// RunScenarioSuite041 runs all scenarios from ScenarioTable041 and returns a slice of events with results.
func RunScenarioSuite041() []CPUSaturationEvent041 {
	var events []CPUSaturationEvent041
	for _, scenario := range ScenarioTable041 {
		event, err := GenerateSaturationEvent041(scenario)
		if err != nil {
			// In case of validation failure, create a failed event with error.
			event = &CPUSaturationEvent041{
				Scenario: scenario,
				Observation: CPUThrottleObservation041{
					ScenarioName: scenario.Name,
					StartTime:    time.Now(),
					EndTime:      time.Now(),
					Error:        err,
				},
				Result: ThrottleResultFail041,
			}
		}
		_ = EvaluateEvent041(event)
		events = append(events, *event)
	}
	return events
}

// SummarizeResults041 provides a summary of passed, failed, and invalid events.
func SummarizeResults041(events []CPUSaturationEvent041) (passed, failed, invalid int) {
	for _, e := range events {
		switch e.Result {
		case ThrottleResultPass041:
			passed++
		case ThrottleResultFail041:
			failed++
		default:
			invalid++
		}
	}
	return
}

// ValidateThrottleObservation041 validates the observation fields.
func ValidateThrottleObservation041(obs CPUThrottleObservation041) error {
	if strings.TrimSpace(obs.ScenarioName) == "" {
		return errors.New("observation scenario name cannot be empty")
	}
	if obs.StartTime.IsZero() || obs.EndTime.IsZero() {
		return errors.New("observation times must be set")
	}
	if obs.EndTime.Before(obs.StartTime) {
		return errors.New("end time before start time")
	}
	if obs.ThrottlePercent < 0 || obs.ThrottlePercent > 100 {
		return fmt.Errorf("throttle percent must be 0-100, got %f", obs.ThrottlePercent)
	}
	if obs.PeakCPULoad < 0 || obs.PeakCPULoad > 100 {
		return fmt.Errorf("peak CPU load must be 0-100, got %f", obs.PeakCPULoad)
	}
	if obs.AverageCPULoad < 0 || obs.AverageCPULoad > 100 {
		return fmt.Errorf("average CPU load must be 0-100, got %f", obs.AverageCPULoad)
	}
	switch obs.ObservedThrottle {
	case CPUThrottleUnknown041, CPUThrottleNoThrottle041, CPUThrottleLight041, CPUThrottleModerate041, CPUThrottleSevere041:
		// valid
	default:
		return fmt.Errorf("invalid observed throttle state: %v", obs.ObservedThrottle)
	}
	return nil
}

// IsSaturationSevere041 returns true if the throttle percentage indicates severe saturation.
func IsSaturationSevere041(throttlePercent float64) bool {
	return throttlePercent > 50.0
}

// IsSaturationModerate041 returns true if the throttle percentage is between 10 and 50 inclusive.
func IsSaturationModerate041(throttlePercent float64) bool {
	return throttlePercent >= 10.0 && throttlePercent <= 50.0
}

// IsSaturationLight041 returns true if the throttle percentage is between 1 and 10 exclusive.
func IsSaturationLight041(throttlePercent float64) bool {
	return throttlePercent >= 1.0 && throttlePercent < 10.0
}

// IsSaturationNone041 returns true if the throttle percentage is less than 1.
func IsSaturationNone041(throttlePercent float64) bool {
	return throttlePercent < 1.0
}

// ThrottleCategoryFromPercent041 returns a category string for the given throttle percent.
func ThrottleCategoryFromPercent041(throttlePercent float64) string {
	switch {
	case IsSaturationNone041(throttlePercent):
		return "none"
	case IsSaturationLight041(throttlePercent):
		return "light"
	case IsSaturationModerate041(throttlePercent):
		return "moderate"
	case IsSaturationSevere041(throttlePercent):
		return "severe"
	default:
		return "unknown"
	}
}

// CPUSaturationReport041 aggregates multiple events into a report.
type CPUSaturationReport041 struct {
	Events      []CPUSaturationEvent041
	TotalEvents int
	Passed      int
	Failed      int
	Invalid     int
	GeneratedAt time.Time
}

// NewCPUSaturationReport041 creates a report from a slice of events.
func NewCPUSaturationReport041(events []CPUSaturationEvent041) *CPUSaturationReport041 {
	passed, failed, invalid := SummarizeResults041(events)
	return &CPUSaturationReport041{
		Events:      events,
		TotalEvents: len(events),
		Passed:      passed,
		Failed:      failed,
		Invalid:     invalid,
		GeneratedAt: time.Now(),
	}
}

// String returns a short summary of the report.
func (r *CPUSaturationReport041) String() string {
	return fmt.Sprintf("CPUSaturationReport041: total=%d passed=%d failed=%d invalid=%d",
		r.TotalEvents, r.Passed, r.Failed, r.Invalid)
}

// FormatReport041 returns a multi-line detailed report string.
func FormatReport041(report *CPUSaturationReport041) string {
	if report == nil {
		return "<nil report>"
	}
	var b strings.Builder
	b.WriteString("=== CPU Saturation Report ===\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Total events: %d\n", report.TotalEvents))
	b.WriteString(fmt.Sprintf("Passed: %d\n", report.Passed))
	b.WriteString(fmt.Sprintf("Failed: %d\n", report.Failed))
	b.WriteString(fmt.Sprintf("Invalid: %d\n", report.Invalid))
	b.WriteString("--- Event Details ---\n")
	for i, e := range report.Events {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, e.String()))
	}
	b.WriteString("=== End Report ===")
	return b.String()
}

// ThrottleStateMapping041 is a deterministic mapping from load range to expected throttle state.
type ThrottleStateMapping041 struct {
	MinLoad int
	MaxLoad int
	State   CPUThrottleState041
}

// ThrottleStateTable041 provides deterministic mapping of load ranges to states.
var ThrottleStateTable041 = []ThrottleStateMapping041{
	{MinLoad: 0, MaxLoad: 20, State: CPUThrottleNoThrottle041},
	{MinLoad: 21, MaxLoad: 40, State: CPUThrottleLight041},
	{MinLoad: 41, MaxLoad: 70, State: CPUThrottleModerate041},
	{MinLoad: 71, MaxLoad: 100, State: CPUThrottleSevere041},
}

// ExpectedThrottleForLoad041 returns the expected throttle state for a given load percentage.
func ExpectedThrottleForLoad041(loadPercent int) CPUThrottleState041 {
	for _, m := range ThrottleStateTable041 {
		if loadPercent >= m.MinLoad && loadPercent <= m.MaxLoad {
			return m.State
		}
	}
	return CPUThrottleUnknown041
}

// LoadToString041 converts a load percent to a descriptive string.
func LoadToString041(load int) string {
	switch {
	case load <= 20:
		return "low"
	case load <= 40:
		return "light"
	case load <= 70:
		return "moderate"
	default:
		return "high"
	}
}

// ScenarioCountByThrottle041 returns a map with counts of scenarios per expected throttle state.
func ScenarioCountByThrottle041() map[CPUThrottleState041]int {
	counts := make(map[CPUThrottleState041]int)
	for _, s := range ScenarioTable041 {
		counts[s.ExpectedThrottle]++
	}
	return counts
}

// ValidateScenarioName041 checks if a scenario name exists in the table.
func ValidateScenarioName041(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("scenario name is empty")
	}
	if GetScenarioByName041(name) == nil {
		return fmt.Errorf("scenario %q not found in ScenarioTable041", name)
	}
	return nil
}

// EnsureCPULoadBoundary041 returns load within 1-100, clamping if necessary.
func EnsureCPULoadBoundary041(load int) int {
	if load < 1 {
		return 1
	}
	if load > 100 {
		return 100
	}
	return load
}

// EnsureDurationBoundary041 ensures duration is at least 1 second.
func EnsureDurationBoundary041(seconds int) int {
	if seconds < 1 {
		return 1
	}
	return seconds
}

// AllowedCoresOptions041 returns predefined allowed cores options for test generation.
var AllowedCoresOptions041 = []int{0, 1, 2, 4, 8}

// ThrottleDurationFactors041 maps throttle state to expected duration factors.
var ThrottleDurationFactors041 = map[CPUThrottleState041]float64{
	CPUThrottleNoThrottle041: 0.0,
	CPUThrottleLight041:      0.1,
	CPUThrottleModerate041:   0.3,
	CPUThrottleSevere041:     0.7,
}

// ExpectedThrottleDuration041 returns expected duration of throttle in seconds based on state and total duration.
func ExpectedThrottleDuration041(state CPUThrottleState041, totalSeconds int) float64 {
	factor, ok := ThrottleDurationFactors041[state]
	if !ok {
		return 0.0
	}
	return factor * float64(totalSeconds)
}

// ValidateThresholds041 validates that the thresholds in ThrottleStateTable041 are consistent.
func ValidateThresholds041() error {
	prevMax := -1
	for i, m := range ThrottleStateTable041 {
		if m.MinLoad < 0 || m.MaxLoad > 100 || m.MinLoad > m.MaxLoad {
			return fmt.Errorf("table entry %d: invalid load range [%d, %d]", i, m.MinLoad, m.MaxLoad)
		}
		if m.MinLoad <= prevMax {
			return fmt.Errorf("table entry %d: overlapping or non-increasing min load %d <= previous max %d", i, m.MinLoad, prevMax)
		}
		prevMax = m.MaxLoad
	}
	// Check that the union covers exactly 0-100
	if len(ThrottleStateTable041) > 0 {
		first := ThrottleStateTable041[0]
		last := ThrottleStateTable041[len(ThrottleStateTable041)-1]
		if first.MinLoad != 0 {
			return errors.New("first entry min load must be 0")
		}
		if last.MaxLoad != 100 {
			return errors.New("last entry max load must be 100")
		}
	}
	return nil
}

// CPUThrottleStateFromString041 converts a string to CPUThrottleState041.
func CPUThrottleStateFromString041(s string) (CPUThrottleState041, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "no-throttle", "none":
		return CPUThrottleNoThrottle041, nil
	case "light", "light-throttle":
		return CPUThrottleLight041, nil
	case "moderate", "moderate-throttle":
		return CPUThrottleModerate041, nil
	case "severe", "severe-throttle":
		return CPUThrottleSevere041, nil
	default:
		return CPUThrottleUnknown041, fmt.Errorf("unknown throttle state string: %q", s)
	}
}
