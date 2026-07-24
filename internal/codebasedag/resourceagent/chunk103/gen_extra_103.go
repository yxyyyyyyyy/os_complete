package chunk103

import (
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

// BlastConfig103 holds configuration parameters for blast radius estimation.
type BlastConfig103 struct {
	DecayExponent   float64 // Exponent for distance decay (e.g., 2.0 for inverse square)
	MaxDistance     float64 // Maximum radius beyond which impact is zero
	DefaultSeverity float64 // Severity used when not specified (0..1)
	MaxAgents       int     // Upper bound on number of agents
	TableResolution int     // Number of bins in precomputed tables
}

// DefaultBlastConfig103 returns a config with sensible defaults.
func DefaultBlastConfig103() BlastConfig103 {
	return BlastConfig103{
		DecayExponent:   2.0,
		MaxDistance:     1000.0,
		DefaultSeverity: 0.5,
		MaxAgents:       1000,
		TableResolution: 100,
	}
}

// ValidateBlastConfig103 checks config fields for sanity.
func ValidateBlastConfig103(cfg BlastConfig103) error {
	if cfg.DecayExponent <= 0 {
		return errors.New("BlastConfig103.DecayExponent must be positive")
	}
	if cfg.MaxDistance <= 0 {
		return errors.New("BlastConfig103.MaxDistance must be positive")
	}
	if cfg.DefaultSeverity < 0 || cfg.DefaultSeverity > 1 {
		return errors.New("BlastConfig103.DefaultSeverity must be in [0,1]")
	}
	if cfg.MaxAgents < 1 {
		return errors.New("BlastConfig103.MaxAgents must be at least 1")
	}
	if cfg.TableResolution < 1 {
		return errors.New("BlastConfig103.TableResolution must be at least 1")
	}
	return nil
}

// Agent103 represents a single participant in the multi-agent system.
type Agent103 struct {
	ID            int
	X, Y, Z       float64
	Susceptibility float64 // 0 = immune, 1 = fully susceptible
	Active        bool
}

// NewAgent103 creates a new agent with default susceptibility if not provided.
func NewAgent103(id int, x, y, z, susceptibility float64) Agent103 {
	if susceptibility < 0 {
		susceptibility = 0
	} else if susceptibility > 1 {
		susceptibility = 1
	}
	return Agent103{
		ID:             id,
		X:              x,
		Y:              y,
		Z:              z,
		Susceptibility: susceptibility,
		Active:         true,
	}
}

// ValidateAgent103 checks an Agent103 for required fields.
func ValidateAgent103(a Agent103) error {
	if a.ID < 0 {
		return errors.New("Agent103.ID must be non-negative")
	}
	if a.Susceptibility < 0 || a.Susceptibility > 1 {
		return errors.New("Agent103.Susceptibility must be in [0,1]")
	}
	return nil
}

// FaultEvent103 describes a fault occurring at an agent.
type FaultEvent103 struct {
	SourceID int
	Severity float64 // 0..1
	FaultType string
}

// ValidateFaultEvent103 checks fields of a fault event.
func ValidateFaultEvent103(ev FaultEvent103, maxAgents int) error {
	if ev.SourceID < 0 || ev.SourceID >= maxAgents {
		return fmt.Errorf("FaultEvent103.SourceID out of range [0,%d)", maxAgents)
	}
	if ev.Severity < 0 || ev.Severity > 1 {
		return errors.New("FaultEvent103.Severity must be in [0,1]")
	}
	return nil
}

// BlastResult103 holds the computed impact on each agent.
type BlastResult103 struct {
	ImpactMap map[int]float64 // Agent ID -> impact value (0..1)
	TotalImpact float64
	AffectedCount int
	RadiusEstimate float64 // Estimated blast radius
}

// NewBlastResult103 initializes an empty result.
func NewBlastResult103() *BlastResult103 {
	return &BlastResult103{
		ImpactMap: make(map[int]float64),
	}
}

// BlastRadiusEstimator103 is the main estimator for fault blast radius.
type BlastRadiusEstimator103 struct {
	config    BlastConfig103
	agents    []Agent103
	distTable [][]float64     // Precomputed distances between agents
	impactTable [][]float64   // impact[distanceBin][susceptibilityBin] precomputed impact
	distanceBins []float64    // bin edges
	susceptBins  []float64
	mu         sync.RWMutex
}

// NewBlastRadiusEstimator103 creates an estimator from config and agents.
func NewBlastRadiusEstimator103(cfg BlastConfig103, agents []Agent103) (*BlastRadiusEstimator103, error) {
	if err := ValidateBlastConfig103(cfg); err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, errors.New("agents list cannot be empty")
	}
	if len(agents) > cfg.MaxAgents {
		return nil, fmt.Errorf("number of agents %d exceeds MaxAgents %d", len(agents), cfg.MaxAgents)
	}
	for i, a := range agents {
		if err := ValidateAgent103(a); err != nil {
			return nil, fmt.Errorf("agent %d invalid: %w", i, err)
		}
	}

	est := &BlastRadiusEstimator103{
		config:    cfg,
		agents:    make([]Agent103, len(agents)),
		distTable: make([][]float64, len(agents)),
	}
	copy(est.agents, agents)

	// Precompute distance table
	for i := 0; i < len(agents); i++ {
		est.distTable[i] = make([]float64, len(agents))
		for j := 0; j < len(agents); j++ {
			est.distTable[i][j] = Distance_103(agents[i], agents[j])
		}
	}

	// Build distance bins
	est.distanceBins = make([]float64, cfg.TableResolution+1)
	for k := 0; k <= cfg.TableResolution; k++ {
		est.distanceBins[k] = cfg.MaxDistance * float64(k) / float64(cfg.TableResolution)
	}

	// Build susceptibility bins
	est.susceptBins = make([]float64, cfg.TableResolution+1)
	for k := 0; k <= cfg.TableResolution; k++ {
		est.susceptBins[k] = float64(k) / float64(cfg.TableResolution)
	}

	// Precompute impact table: impact[distanceBin][susceptibilityBin]
	est.impactTable = make([][]float64, cfg.TableResolution)
	for d := 0; d < cfg.TableResolution; d++ {
		est.impactTable[d] = make([]float64, cfg.TableResolution)
		for s := 0; s < cfg.TableResolution; s++ {
			distMid := (est.distanceBins[d] + est.distanceBins[d+1]) / 2.0
			suscMid := (est.susceptBins[s] + est.susceptBins[s+1]) / 2.0
			raw := DecayFunction_103(distMid, cfg.DecayExponent, cfg.MaxDistance)
			est.impactTable[d][s] = raw * suscMid
		}
	}
	return est, nil
}

// Distance_103 computes Euclidean distance between two agents.
func Distance_103(a, b Agent103) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// DecayFunction_103 returns a unit impact (0..1) based on distance.
// Uses inverse power law with cutoff.
func DecayFunction_103(dist, exponent, maxDist float64) float64 {
	if dist >= maxDist {
		return 0.0
	}
	if dist <= 0 {
		return 1.0
	}
	return 1.0 / (1.0 + math.Pow(dist/maxDist, exponent))
}

// ValidateAgents103 validates a slice of agents for consistency.
func ValidateAgents103(agents []Agent103) error {
	if len(agents) == 0 {
		return errors.New("agent list empty")
	}
	seen := make(map[int]bool)
	for i, a := range agents {
		if err := ValidateAgent103(a); err != nil {
			return fmt.Errorf("agent %d invalid: %w", i, err)
		}
		if !a.Active {
			return fmt.Errorf("agent %d is inactive", i)
		}
		if seen[a.ID] {
			return fmt.Errorf("duplicate agent ID %d", a.ID)
		}
		seen[a.ID] = true
	}
	return nil
}

// EstimateBlastRadius_103 computes the blast radius for a given fault event.
// It returns a BlastResult103 with impacts per agent.
func (e *BlastRadiusEstimator103) EstimateBlastRadius_103(ev FaultEvent103) (*BlastResult103, error) {
	if err := ValidateFaultEvent103(ev, len(e.agents)); err != nil {
		return nil, err
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := NewBlastResult103()
	source := e.agents[ev.SourceID]
	if !source.Active {
		return nil, fmt.Errorf("source agent %d is inactive", ev.SourceID)
	}
	severity := ev.Severity
	if severity == 0 {
		severity = e.config.DefaultSeverity
	}

	var total float64
	var count int
	var maxDistAffected float64

	for i, target := range e.agents {
		if i == ev.SourceID {
			result.ImpactMap[ev.SourceID] = severity
			total += severity
			count++
			continue
		}
		if !target.Active {
			result.ImpactMap[i] = 0.0
			continue
		}
		dist := e.distTable[ev.SourceID][i]
		if dist >= e.config.MaxDistance {
			result.ImpactMap[i] = 0.0
			continue
		}
		// Lookup impact from precomputed table
		dBin := int(dist / e.config.MaxDistance * float64(e.config.TableResolution))
		if dBin >= e.config.TableResolution {
			dBin = e.config.TableResolution - 1
		}
		sBin := int(target.Susceptibility * float64(e.config.TableResolution))
		if sBin >= e.config.TableResolution {
			sBin = e.config.TableResolution - 1
		}
		impact := e.impactTable[dBin][sBin] * severity
		result.ImpactMap[i] = impact
		if impact > 0 {
			total += impact
			count++
			if dist > maxDistAffected {
				maxDistAffected = dist
			}
		}
	}
	result.TotalImpact = total
	result.AffectedCount = count
	result.RadiusEstimate = maxDistAffected
	return result, nil
}

// EstimateRadiusWithWorkers_103 uses concurrent workers to estimate blast radius
// for multiple fault events. Returns channel of results.
func (e *BlastRadiusEstimator103) EstimateRadiusWithWorkers_103(events []FaultEvent103, workers int) (<-chan *BlastResult103, <-chan error) {
	resultCh := make(chan *BlastResult103, len(events))
	errCh := make(chan error, len(events))
	if workers <= 0 {
		workers = 1
	}
	var wg sync.WaitGroup
	workCh := make(chan FaultEvent103, len(events))
	for _, ev := range events {
		workCh <- ev
	}
	close(workCh)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ev := range workCh {
				res, err := e.EstimateBlastRadius_103(ev)
				if err != nil {
					errCh <- err
					continue
				}
				resultCh <- res
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()
	return resultCh, errCh
}

// GenerateAgents103 creates a list of random agents for testing.
func GenerateAgents103(n int, xRange, yRange, zRange float64) []Agent103 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	agents := make([]Agent103, n)
	for i := 0; i < n; i++ {
		x := rng.Float64() * xRange
		y := rng.Float64() * yRange
		z := rng.Float64() * zRange
		s := rng.Float64()
		agents[i] = NewAgent103(i, x, y, z, s)
	}
	return agents
}

// GenerateFaultEvents103 creates a slice of random fault events.
func GenerateFaultEvents103(numEvents, numAgents int, severityRange [2]float64) []FaultEvent103 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	events := make([]FaultEvent103, numEvents)
	for i := 0; i < numEvents; i++ {
		sev := severityRange[0] + rng.Float64()*(severityRange[1]-severityRange[0])
		if sev < 0 {
			sev = 0
		} else if sev > 1 {
			sev = 1
		}
		events[i] = FaultEvent103{
			SourceID: rng.Intn(numAgents),
			Severity: sev,
			FaultType: fmt.Sprintf("TYPE_%d", rng.Intn(5)),
		}
	}
	return events
}

// BlastRadiusTable103 represents a precomputed lookup table for impact factors.
type BlastRadiusTable103 struct {
	DistanceBins []float64
	SusceptBins  []float64
	Factors      [][]float64 // stored as CSV string
}

// NewBlastRadiusTable103 builds a table from config parameters.
func NewBlastRadiusTable103(cfg BlastConfig103) *BlastRadiusTable103 {
	distBins := make([]float64, cfg.TableResolution+1)
	for k := 0; k <= cfg.TableResolution; k++ {
		distBins[k] = cfg.MaxDistance * float64(k) / float64(cfg.TableResolution)
	}
	suscBins := make([]float64, cfg.TableResolution+1)
	for k := 0; k <= cfg.TableResolution; k++ {
		suscBins[k] = float64(k) / float64(cfg.TableResolution)
	}
	factors := make([][]float64, cfg.TableResolution)
	for d := 0; d < cfg.TableResolution; d++ {
		factors[d] = make([]float64, cfg.TableResolution)
		for s := 0; s < cfg.TableResolution; s++ {
			distMid := (distBins[d] + distBins[d+1]) / 2.0
			suscMid := (suscBins[s] + suscBins[s+1]) / 2.0
			factors[d][s] = DecayFunction_103(distMid, cfg.DecayExponent, cfg.MaxDistance) * suscMid
		}
	}
	return &BlastRadiusTable103{
		DistanceBins: distBins,
		SusceptBins:  suscBins,
		Factors:      factors,
	}
}

// ImpactFromTable103 lookup impact for given distance and susceptibility.
func (t *BlastRadiusTable103) ImpactFromTable103(dist, susc float64) float64 {
	dBin := sort.SearchFloat64s(t.DistanceBins, dist)
	if dBin >= len(t.DistanceBins)-1 {
		dBin = len(t.DistanceBins) - 2
	}
	sBin := sort.SearchFloat64s(t.SusceptBins, susc)
	if sBin >= len(t.SusceptBins)-1 {
		sBin = len(t.SusceptBins) - 2
	}
	return t.Factors[dBin][sBin]
}

// CSVString103 returns the table as a CSV-formatted string (no error handling).
func (t *BlastRadiusTable103) CSVString103() string {
	var sb strings.Builder
	// Write header
	sb.WriteString("distance_bin_start,distance_bin_end,suscept_bin_start,suscept_bin_end,impact\n")
	for d := 0; d < len(t.Factors); d++ {
		for s := 0; s < len(t.Factors[d]); s++ {
			sb.WriteString(fmt.Sprintf("%.6f,%.6f,%.6f,%.6f,%.6f\n",
				t.DistanceBins[d], t.DistanceBins[d+1],
				t.SusceptBins[s], t.SusceptBins[s+1],
				t.Factors[d][s]))
		}
	}
	return sb.String()
}

// WriteTableToCSV103 writes the precomputed table to a file-like writer.
// For demonstration: writes to a string and returns it.
func (t *BlastRadiusTable103) WriteTableToCSV103() (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)
	// header
	header := []string{"distance_bin_start", "distance_bin_end", "suscept_bin_start", "suscept_bin_end", "impact"}
	if err := writer.Write(header); err != nil {
		return "", err
	}
	for d := 0; d < len(t.Factors); d++ {
		for s := 0; s < len(t.Factors[d]); s++ {
			row := []string{
				fmt.Sprintf("%.6f", t.DistanceBins[d]),
				fmt.Sprintf("%.6f", t.DistanceBins[d+1]),
				fmt.Sprintf("%.6f", t.SusceptBins[s]),
				fmt.Sprintf("%.6f", t.SusceptBins[s+1]),
				fmt.Sprintf("%.6f", t.Factors[d][s]),
			}
			if err := writer.Write(row); err != nil {
				return "", err
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// NormalizeImpact103 scales impact values to [0,1] based on a reference max.
func NormalizeImpact103(impact map[int]float64, maxRef float64) map[int]float64 {
	out := make(map[int]float64, len(impact))
	if maxRef <= 0 {
		for k := range impact {
			out[k] = 0
		}
		return out
	}
	for k, v := range impact {
		if v < 0 {
			out[k] = 0
		} else {
			out[k] = v / maxRef
			if out[k] > 1 {
				out[k] = 1
			}
		}
	}
	return out
}

// AggregateImpact103 sums impact values and returns total, count, average.
func AggregateImpact103(impact map[int]float64) (total float64, count int, avg float64) {
	for _, v := range impact {
		if v > 0 {
			total += v
			count++
		}
	}
	if count > 0 {
		avg = total / float64(count)
	}
	return
}

// ImpactThresholdCount103 counts agents with impact above a threshold.
func ImpactThresholdCount103(impact map[int]float64, threshold float64) int {
	count := 0
	for _, v := range impact {
		if v >= threshold {
			count++
		}
	}
	return count
}

// RadiusAtThreshold103 estimates the radius at which impact drops below threshold.
// Uses distance bins from estimator's precomputed table.
func (e *BlastRadiusEstimator103) RadiusAtThreshold103(ev FaultEvent103, threshold float64) (float64, error) {
	result, err := e.EstimateBlastRadius_103(ev)
	if err != nil {
		return 0, err
	}
	// Find the farthest agent with impact >= threshold
	maxDist := 0.0
	source := ev.SourceID
	for id, impact := range result.ImpactMap {
		if id == source {
			continue
		}
		if impact >= threshold {
			dist := e.distTable[source][id]
			if dist > maxDist {
				maxDist = dist
			}
		}
	}
	return maxDist, nil
}

// MonteCarloRadius103 runs multiple random fault events and returns average radius.
func (e *BlastRadiusEstimator103) MonteCarloRadius103(numTrials int, severityRange [2]float64) (avgRadius float64, stdDev float64, err error) {
	if numTrials <= 0 {
		return 0, 0, errors.New("numTrials must be positive")
	}
	radii := make([]float64, numTrials)
	for i := 0; i < numTrials; i++ {
		sev := severityRange[0] + (severityRange[1]-severityRange[0])*rand.Float64()
		ev := FaultEvent103{
			SourceID: rand.Intn(len(e.agents)),
			Severity: sev,
		}
		res, err := e.EstimateBlastRadius_103(ev)
		if err != nil {
			return 0, 0, err
		}
		radii[i] = res.RadiusEstimate
	}
	// compute mean and std
	var sum float64
	for _, r := range radii {
		sum += r
	}
	avgRadius = sum / float64(numTrials)
	var sqSum float64
	for _, r := range radii {
		d := r - avgRadius
		sqSum += d * d
	}
	stdDev = math.Sqrt(sqSum / float64(numTrials))
	return
}

// PrintBlastResult103 prints a summary of a blast result.
func PrintBlastResult103(res *BlastResult103) string {
	return fmt.Sprintf("BlastResult: totalImpact=%.4f, affected=%d, radiusEst=%.4f",
		res.TotalImpact, res.AffectedCount, res.RadiusEstimate)
}

// FormatBlastResultMap103 returns a sorted string representation of impact map.
func FormatBlastResultMap103(impact map[int]float64) string {
	ids := make([]int, 0, len(impact))
	for id := range impact {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var sb strings.Builder
	sb.WriteString("{")
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%d: %.4f", id, impact[id])
	}
	sb.WriteString("}")
	return sb.String()
}

// IsFaultContained103 returns true if total impact is below a containment threshold.
func IsFaultContained103(result *BlastResult103, containmentThreshold float64) bool {
	return result.TotalImpact <= containmentThreshold
}

// PrioritySort103 sorts agents by estimated impact (descending).
func PrioritySort103(impact map[int]float64) []int {
	ids := make([]int, 0, len(impact))
	for id := range impact {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return impact[ids[i]] > impact[ids[j]]
	})
	return ids
}

// ValidateFaultType103 checks that fault type is one of allowed types.
func ValidateFaultType103(faultType string, allowedTypes []string) bool {
	for _, t := range allowedTypes {
		if faultType == t {
			return true
		}
	}
	return false
}

// TableDistance103 returns a precomputed distance for a pair of agent indices.
func (e *BlastRadiusEstimator103) TableDistance103(i, j int) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if i < 0 || i >= len(e.distTable) || j < 0 || j >= len(e.distTable[i]) {
		return math.Inf(1)
	}
	return e.distTable[i][j]
}

// CopyAgents103 returns a deep copy of the agent slice.
func CopyAgents103(agents []Agent103) []Agent103 {
	out := make([]Agent103, len(agents))
	copy(out, agents)
	return out
}

// SusceptibilityDistribution103 returns a histogram of susceptibilities.
func SusceptibilityDistribution103(agents []Agent103, bins int) map[int]int {
	if bins <= 0 {
		bins = 10
	}
	hist := make(map[int]int)
	for _, a := range agents {
		bin := int(a.Susceptibility * float64(bins))
		if bin >= bins {
			bin = bins - 1
		}
		hist[bin]++
	}
	return hist
}

// DistanceDistribution103 returns a histogram of distances from a source index.
func (e *BlastRadiusEstimator103) DistanceDistribution103(sourceIdx int, bins int) map[int]int {
	if bins <= 0 {
		bins = 10
	}
	hist := make(map[int]int)
	e.mu.RLock()
	defer e.mu.RUnlock()
	for j := 0; j < len(e.distTable[sourceIdx]); j++ {
		if j == sourceIdx {
			continue
		}
		d := e.distTable[sourceIdx][j]
		if d >= e.config.MaxDistance {
			continue
		}
		bin := int(d / e.config.MaxDistance * float64(bins))
		if bin >= bins {
			bin = bins - 1
		}
		hist[bin]++
	}
	return hist
}

// EstimateImpactForAllSources103 computes blast radius for each possible source.
func (e *BlastRadiusEstimator103) EstimateImpactForAllSources103(severity float64) []*BlastResult103 {
	results := make([]*BlastResult103, len(e.agents))
	var wg sync.WaitGroup
	for i := 0; i < len(e.agents); i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ev := FaultEvent103{
				SourceID: idx,
				Severity: severity,
			}
			res, err := e.EstimateBlastRadius_103(ev)
			if err != nil {
				res = &BlastResult103{ImpactMap: map[int]float64{idx: severity}}
			}
			results[idx] = res
		}(i)
	}
	wg.Wait()
	return results
}

// AverageImpactMatrix103 returns a matrix of average impacts (source x target) from a set of events.
func AverageImpactMatrix103(estimator *BlastRadiusEstimator103, events []FaultEvent103, numAgents int) [][]float64 {
	matrix := make([][]float64, numAgents)
	for i := range matrix {
		matrix[i] = make([]float64, numAgents)
	}
	for _, ev := range events {
		res, err := estimator.EstimateBlastRadius_103(ev)
		if err != nil {
			continue
		}
		for target, impact := range res.ImpactMap {
			matrix[ev.SourceID][target] += impact
		}
	}
	if len(events) > 0 {
		n := float64(len(events))
		for i := range matrix {
			for j := range matrix[i] {
				matrix[i][j] /= n
			}
		}
	}
	return matrix
}

// BlastValidator103 provides validation utilities for blast estimation inputs.
type BlastValidator103 struct{}

// ValidateAgentSlice103 validates all agents in a slice.
func (v BlastValidator103) ValidateAgentSlice103(agents []Agent103) error {
	return ValidateAgents103(agents)
}

// ValidateEventSequence103 checks a sequence of events for consistency.
func (v BlastValidator103) ValidateEventSequence103(events []FaultEvent103, numAgents int) error {
	for i, ev := range events {
		if err := ValidateFaultEvent103(ev, numAgents); err != nil {
			return fmt.Errorf("event %d invalid: %w", i, err)
		}
	}
	return nil
}

// ValidateBlastConfigEx102 is an alias for ValidateBlastConfig103 (backward compat).
func (v BlastValidator103) ValidateBlastConfigEx102(cfg BlastConfig103) error {
	return ValidateBlastConfig103(cfg)
}

// Helper103 holds miscellaneous helper functions.
type Helper103 struct{}

// Clamp103 clamps a value between min and max.
func (h Helper103) Clamp103(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Lerp103 linear interpolation between a and b by t.
func (h Helper103) Lerp103(a, b, t float64) float64 {
	return a + (b-a)*t
}

// InverseLerp103 returns t such that lerp(a,b,t) = val.
func (h Helper103) InverseLerp103(a, b, val float64) float64 {
	if a == b {
		return 0
	}
	return (val - a) / (b - a)
}

// MapRange103 maps val from input range to output range.
func (h Helper103) MapRange103(val, inMin, inMax, outMin, outMax float64) float64 {
	t := h.InverseLerp103(inMin, inMax, val)
	return h.Lerp103(outMin, outMax, t)
}

// ExponentialDecay103 returns decay factor for given distance, exponent, scale.
func (h Helper103) ExponentialDecay103(dist, exponent, scale float64) float64 {
	if dist <= 0 {
		return 1
	}
	return math.Exp(-exponent * dist / scale)
}

// GaussianDecay103 returns a Gaussian-like decay.
func (h Helper103) GaussianDecay103(dist, sigma float64) float64 {
	if sigma <= 0 {
		return 0
	}
	return math.Exp(-dist*dist / (2 * sigma*sigma))
}

// SmoothStep103 applies smoothstep interpolation.
func (h Helper103) SmoothStep103(edge0, edge1, x float64) float64 {
	t := h.Clamp103((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

// ImpactToSeverity103 converts impact to a severity level string.
func (h Helper103) ImpactToSeverity103(impact float64) string {
	switch {
	case impact >= 0.8:
		return "critical"
	case impact >= 0.5:
		return "high"
	case impact >= 0.2:
		return "medium"
	case impact > 0:
		return "low"
	default:
		return "none"
	}
}

// FormatTableHeader103 returns a string for a table row header.
func (h Helper103) FormatTableHeader103(columns []string) string {
	return strings.Join(columns, "\t")
}

// FormatInts103 formats a slice of ints.
func (h Helper103) FormatInts103(ints []int) string {
	parts := make([]string, len(ints))
	for i, v := range ints {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

// FormatFloats103 formats a slice of floats.
func (h Helper103) FormatFloats103(floats []float64, precision int) string {
	parts := make([]string, len(floats))
	fmtStr := fmt.Sprintf("%%.%df", precision)
	for i, v := range floats {
		parts[i] = fmt.Sprintf(fmtStr, v)
	}
	return strings.Join(parts, ",")
}

// TruncateDistance103 trims distance to maxDistance.
func (h Helper103) TruncateDistance103(dist, maxDist float64) float64 {
	if dist > maxDist {
		return maxDist
	}
	return dist
}

// AgentsToCSV103 converts agent list to CSV string.
func AgentsToCSV103(agents []Agent103) string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)
	writer.Write([]string{"id", "x", "y", "z", "susceptibility", "active"})
	for _, a := range agents {
		row := []string{
			fmt.Sprintf("%d", a.ID),
			fmt.Sprintf("%f", a.X),
			fmt.Sprintf("%f", a.Y),
			fmt.Sprintf("%f", a.Z),
			fmt.Sprintf("%f", a.Susceptibility),
			fmt.Sprintf("%t", a.Active),
		}
		writer.Write(row)
	}
	writer.Flush()
	return sb.String()
}
