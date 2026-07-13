package review

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaVersion          = "review/v1"
	MeasurementMeasured    = "measured"
	MeasurementDerived     = "derived"
	MeasurementUnsupported = "unsupported"
	csvHeader              = "mode,metric,count,success_count,failed_count,success_rate,mean,stddev,min,max,p50,p95,measurement_kind,unit"
)

type MetricValue struct {
	Value float64 `json:"value"`
	Kind  string  `json:"measurement_kind"`
	Unit  string  `json:"unit,omitempty"`
}

type Stats struct {
	Count        int     `json:"count"`
	SuccessCount int     `json:"success_count"`
	FailedCount  int     `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`
	Mean         float64 `json:"mean"`
	Stddev       float64 `json:"stddev"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	P50          float64 `json:"p50"`
	P95          float64 `json:"p95"`
}

type RunObservation struct {
	ScenarioID    string                 `json:"scenario_id"`
	RunID         string                 `json:"run_id"`
	Mode          string                 `json:"mode"`
	Timestamp     string                 `json:"timestamp"`
	Success       bool                   `json:"success"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Metrics       map[string]MetricValue `json:"metrics"`
	Events        []EventRecord          `json:"events,omitempty"`
	Artifact      string                 `json:"artifact,omitempty"`
}

type EventRecord struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type ComparisonRow struct {
	Mode             string  `json:"mode"`
	Metric           string  `json:"metric"`
	Stats            Stats   `json:"stats"`
	MeasurementKind  string  `json:"measurement_kind"`
	Unit             string  `json:"unit,omitempty"`
	BaselineMean     float64 `json:"baseline_mean,omitempty"`
	ImprovementPct   float64 `json:"improvement_pct,omitempty"`
	ImprovementValid bool    `json:"improvement_valid"`
}

type ScenarioResult struct {
	SchemaVersion string                      `json:"schema_version"`
	ScenarioID    string                      `json:"scenario_id"`
	RunID         string                      `json:"run_id"`
	Timestamp     string                      `json:"timestamp"`
	GitCommit     string                      `json:"git_commit"`
	GitDirty      bool                        `json:"git_dirty"`
	Environment   map[string]string           `json:"environment"`
	EvidenceMode  string                      `json:"evidence_mode"`
	Parameters    map[string]any              `json:"parameters"`
	Seed          int64                       `json:"seed"`
	Warmup        int                         `json:"warmup"`
	MeasuredRuns  int                         `json:"measured_runs"`
	PerRun        []RunObservation            `json:"per_run"`
	Summary       map[string]map[string]Stats `json:"summary"`
	Comparison    []ComparisonRow             `json:"comparison"`
	ArtifactPaths []string                    `json:"artifact_paths"`
	Limitations   []string                    `json:"limitations"`
}

// Aggregate computes population statistics for the supplied observations. A
// missing success entry is treated as a failed observation so dropped runs are
// visible in success_rate instead of silently disappearing.
func Aggregate(values []float64, success []bool) Stats {
	stats := Stats{Count: len(values)}
	if len(values) == 0 {
		return stats
	}
	clean := make([]float64, 0, len(values))
	for i, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		clean = append(clean, value)
		ok := i < len(success) && success[i]
		if ok {
			stats.SuccessCount++
		}
	}
	stats.FailedCount = stats.Count - stats.SuccessCount
	if stats.Count > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.Count)
	}
	if len(clean) == 0 {
		return stats
	}
	sort.Float64s(clean)
	sum := 0.0
	for _, value := range clean {
		sum += value
	}
	stats.Mean = sum / float64(len(clean))
	variance := 0.0
	for _, value := range clean {
		delta := value - stats.Mean
		variance += delta * delta
	}
	stats.Stddev = math.Sqrt(variance / float64(len(clean)))
	stats.Min = clean[0]
	stats.Max = clean[len(clean)-1]
	stats.P50 = Percentile(clean, 0.50)
	stats.P95 = Percentile(clean, 0.95)
	return stats
}

// Percentile uses linear interpolation and expects an already sorted slice.
func Percentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if percentile <= 0 {
		return sortedValues[0]
	}
	if percentile >= 1 {
		return sortedValues[len(sortedValues)-1]
	}
	position := percentile * float64(len(sortedValues)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sortedValues[lower]
	}
	weight := position - float64(lower)
	return sortedValues[lower] + (sortedValues[upper]-sortedValues[lower])*weight
}

func Improvement(baseline, candidate float64) (float64, bool) {
	if baseline == 0 || math.IsNaN(baseline) || math.IsNaN(candidate) || math.IsInf(baseline, 0) || math.IsInf(candidate, 0) {
		return 0, false
	}
	return (baseline - candidate) / baseline * 100, true
}

func WriteScenarioArtifacts(outDir string, result *ScenarioResult) error {
	if result == nil {
		return fmt.Errorf("scenario result is required")
	}
	if outDir == "" {
		return fmt.Errorf("scenario output directory is required")
	}
	if result.SchemaVersion == "" {
		result.SchemaVersion = SchemaVersion
	}
	if err := os.MkdirAll(filepath.Join(outDir, "raw"), 0o755); err != nil {
		return err
	}
	for i := range result.PerRun {
		run := &result.PerRun[i]
		if run.RunID == "" {
			run.RunID = fmt.Sprintf("run-%03d", i+1)
		}
		run.Artifact = filepath.ToSlash(filepath.Join("raw", run.RunID+".json"))
		data, err := json.MarshalIndent(run, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outDir, filepath.FromSlash(run.Artifact)), append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	result.Summary = summarizeRuns(result.PerRun)
	result.Comparison = comparisonRows(result.Summary)
	result.ArtifactPaths = []string{"raw/", "summary.json", "comparison.csv", "report.md"}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "summary.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := writeComparisonCSV(filepath.Join(outDir, "comparison.csv"), result.Comparison); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "report.md"), []byte(renderReport(*result)), 0o644)
}

func summarizeRuns(runs []RunObservation) map[string]map[string]Stats {
	values := make(map[string]map[string][]float64)
	statuses := make(map[string]map[string][]bool)
	for _, run := range runs {
		if values[run.Mode] == nil {
			values[run.Mode] = make(map[string][]float64)
			statuses[run.Mode] = make(map[string][]bool)
		}
		for name, metric := range run.Metrics {
			values[run.Mode][name] = append(values[run.Mode][name], metric.Value)
			statuses[run.Mode][name] = append(statuses[run.Mode][name], run.Success)
		}
	}
	out := make(map[string]map[string]Stats)
	for mode, metrics := range values {
		out[mode] = make(map[string]Stats)
		for name, series := range metrics {
			out[mode][name] = Aggregate(series, statuses[mode][name])
		}
	}
	return out
}

func comparisonRows(summary map[string]map[string]Stats) []ComparisonRow {
	baseline := summary["baseline"]
	modes := make([]string, 0, len(summary))
	for mode := range summary {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	rows := make([]ComparisonRow, 0)
	for _, mode := range modes {
		metrics := summary[mode]
		names := make([]string, 0, len(metrics))
		for name := range metrics {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			row := ComparisonRow{Mode: mode, Metric: name, Stats: metrics[name], MeasurementKind: MeasurementMeasured}
			if base, ok := baseline[name]; ok {
				row.BaselineMean = base.Mean
				row.ImprovementPct, row.ImprovementValid = Improvement(base.Mean, metrics[name].Mean)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func writeComparisonCSV(path string, rows []ComparisonRow) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.Write(strings.Split(csvHeader, ",")); err != nil {
		return err
	}
	for _, row := range rows {
		stats := row.Stats
		if err := writer.Write([]string{
			row.Mode, row.Metric, strconv.Itoa(stats.Count), strconv.Itoa(stats.SuccessCount), strconv.Itoa(stats.FailedCount),
			formatFloat(stats.SuccessRate), formatFloat(stats.Mean), formatFloat(stats.Stddev), formatFloat(stats.Min),
			formatFloat(stats.Max), formatFloat(stats.P50), formatFloat(stats.P95), row.MeasurementKind, row.Unit,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func renderReport(result ScenarioResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", result.ScenarioID)
	fmt.Fprintf(&b, "- schema_version: `%s`\n- run_id: `%s`\n- evidence_mode: `%s`\n- seed: `%d`\n- warmup: %d\n- measured_runs: %d\n\n", result.SchemaVersion, result.RunID, result.EvidenceMode, result.Seed, result.Warmup, result.MeasuredRuns)
	b.WriteString("| mode | metric | mean | stddev | p50 | p95 | success_rate | kind |\n|---|---|---:|---:|---:|---:|---:|---|\n")
	for _, row := range result.Comparison {
		fmt.Fprintf(&b, "| %s | %s | %.3f | %.3f | %.3f | %.3f | %.3f | %s |\n", row.Mode, row.Metric, row.Stats.Mean, row.Stats.Stddev, row.Stats.P50, row.Stats.P95, row.Stats.SuccessRate, row.MeasurementKind)
	}
	if len(result.Limitations) > 0 {
		b.WriteString("\n## Limitations\n\n")
		for _, limitation := range result.Limitations {
			fmt.Fprintf(&b, "- %s\n", limitation)
		}
	}
	return b.String()
}

func newScenarioResult(scenarioID, mode string, seed int64, warmup, measured int, parameters map[string]any) ScenarioResult {
	return ScenarioResult{
		SchemaVersion: SchemaVersion,
		ScenarioID:    scenarioID,
		RunID:         fmt.Sprintf("%s-%d", scenarioID, time.Now().UnixNano()),
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		GitCommit:     gitCommit(),
		GitDirty:      gitDirty(),
		Environment: map[string]string{
			"go_version": runtime.Version(),
			"goos":       runtime.GOOS,
			"goarch":     runtime.GOARCH,
		},
		EvidenceMode: "real-runtime",
		Parameters:   parameters,
		Seed:         seed,
		Warmup:       warmup,
		MeasuredRuns: measured,
		PerRun:       []RunObservation{},
		Summary:      map[string]map[string]Stats{},
		Comparison:   []ComparisonRow{},
		Limitations:  []string{},
		// mode is kept in parameters so the common schema remains stable.
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func gitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func gitDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	return err != nil || len(strings.TrimSpace(string(output))) > 0
}
