package review

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAggregateComputesPopulationStatsAndPercentiles(t *testing.T) {
	stats := Aggregate([]float64{1, 2, 3, 4}, []bool{true, true, false, true})
	if stats.Count != 4 || stats.SuccessCount != 3 || stats.FailedCount != 1 {
		t.Fatalf("counts = %+v", stats)
	}
	if stats.Mean != 2.5 || stats.Stddev != 1.118033988749895 {
		t.Fatalf("stats = %+v", stats)
	}
	if math.Abs(stats.P50-2.5) > 1e-9 || math.Abs(stats.P95-3.85) > 1e-9 {
		t.Fatalf("percentiles = %+v", stats)
	}
	if stats.SuccessRate != 0.75 {
		t.Fatalf("success rate = %v", stats.SuccessRate)
	}
}

func TestAggregateEmptyAndImprovementHandleMissingValues(t *testing.T) {
	if got := Aggregate(nil, nil); got.Count != 0 || got.SuccessRate != 0 {
		t.Fatalf("empty stats = %+v", got)
	}
	if _, ok := Improvement(0, 1); ok {
		t.Fatal("zero baseline must be unsupported")
	}
	if got, ok := Improvement(100, 75); !ok || got != 25 {
		t.Fatalf("improvement = %v, %v", got, ok)
	}
}

func TestWriteScenarioArtifactsPreservesRawRunsAndStableCSV(t *testing.T) {
	out := t.TempDir()
	result := ScenarioResult{
		SchemaVersion: "review/v1",
		ScenarioID:    "test",
		RunID:         "bundle-1",
		EvidenceMode:  "degraded",
		PerRun: []RunObservation{
			{RunID: "run-1", Mode: "baseline", Success: true, Metrics: map[string]MetricValue{"latency_ms": {Value: 10, Kind: MeasurementMeasured, Unit: "ms"}, "saved_bytes": {Value: 0, Kind: MeasurementDerived, Unit: "bytes"}}},
			{RunID: "run-2", Mode: "baseline", Success: false, FailureReason: "timeout", Metrics: map[string]MetricValue{"latency_ms": {Value: 20, Kind: MeasurementMeasured, Unit: "ms"}, "saved_bytes": {Value: 5, Kind: MeasurementDerived, Unit: "bytes"}}},
		},
	}
	if err := WriteScenarioArtifacts(out, &result); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"summary.json", "comparison.csv", "report.md", filepath.Join("raw", "run-1.json"), filepath.Join("raw", "run-2.json")} {
		if _, err := os.Stat(filepath.Join(out, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(out, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded ScenarioResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.PerRun) != 2 || decoded.Summary["baseline"]["latency_ms"].Count != 2 {
		t.Fatalf("decoded = %+v", decoded)
	}
	file, err := os.Open(filepath.Join(out, "comparison.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || strings.Join(rows[0], ",") != csvHeader {
		t.Fatalf("csv rows = %#v", rows)
	}
	foundDerived := false
	for _, row := range rows[1:] {
		if row[1] == "saved_bytes" && row[12] == MeasurementDerived && row[13] == "bytes" {
			foundDerived = true
		}
	}
	if !foundDerived {
		t.Fatalf("derived metric label was not preserved: %#v", rows)
	}
}
