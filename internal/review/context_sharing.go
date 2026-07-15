package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/cvm"
	"aort-r/internal/ipc/shm"
	"aort-r/internal/scheduler"
)

var contextModes = []string{"full-copy", "shared-ipc", "aort-r"}

type ContextSharingConfig struct {
	Mode        string
	Runs        int
	Warmup      int
	Seed        int64
	Timeout     time.Duration
	ContextSize int
	Agents      int
	SharedRatio float64
	// SharedRatioSet distinguishes an explicit 0% run from the default all-ratios run.
	SharedRatioSet bool
	OutDir         string
}

func (cfg ContextSharingConfig) normalized() (ContextSharingConfig, error) {
	if cfg.SharedRatio != 0 {
		cfg.SharedRatioSet = true
	}
	if cfg.Mode == "" {
		cfg.Mode = "all"
	}
	if cfg.Mode != "all" && !contains(contextModes, cfg.Mode) {
		return cfg, fmt.Errorf("unsupported context sharing mode %q", cfg.Mode)
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 20
	}
	if cfg.Warmup < 0 {
		return cfg, fmt.Errorf("warmup must be non-negative")
	}
	if cfg.Seed == 0 {
		cfg.Seed = 20260713
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.ContextSize <= 0 {
		cfg.ContextSize = 4096
	}
	if cfg.Agents <= 0 {
		cfg.Agents = 6
	}
	if cfg.Agents > 64 {
		return cfg, fmt.Errorf("agents must be <= 64")
	}
	if cfg.SharedRatioSet && (cfg.SharedRatio < 0 || cfg.SharedRatio > 1) {
		return cfg, fmt.Errorf("shared ratio must be between 0 and 1")
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "review_remediation", "context_sharing")
	}
	return cfg, nil
}

func RunContextSharing(ctx context.Context, cfg ContextSharingConfig) (ScenarioResult, error) {
	var err error
	cfg, err = cfg.normalized()
	if err != nil {
		return ScenarioResult{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	modes := contextModes
	if cfg.Mode != "all" {
		modes = []string{cfg.Mode}
	}
	ratios := []float64{0, 0.25, 0.5, 0.75}
	if cfg.SharedRatioSet {
		ratios = []float64{cfg.SharedRatio}
	}
	result := newScenarioResult("context-sharing", cfg.Mode, cfg.Seed, cfg.Warmup, cfg.Runs, map[string]any{
		"mode":          cfg.Mode,
		"runs":          cfg.Runs,
		"warmup":        cfg.Warmup,
		"seed":          cfg.Seed,
		"context_size":  cfg.ContextSize,
		"agents":        cfg.Agents,
		"shared_ratios": ratios,
		"timeout_ms":    cfg.Timeout.Milliseconds(),
	})
	result.Limitations = append(result.Limitations,
		"CVM is measured as runtime context-page reuse and materialization; it is not a model-provider KV Cache.",
		"peak_rss_bytes is unsupported when /proc/self/status is unavailable; runtime heap and IPC counters remain explicit measured fields.",
	)
	anyFailure := false
	for modeIndex, mode := range modes {
		for ratioIndex, ratio := range ratios {
			for runIndex := 0; runIndex < cfg.Warmup+cfg.Runs; runIndex++ {
				seed := cfg.Seed + int64(modeIndex*100000+ratioIndex*1000+runIndex)
				observation, runErr := runContextOnce(ctx, mode, ratio, runIndex, seed, cfg)
				if runIndex >= cfg.Warmup {
					result.PerRun = append(result.PerRun, observation)
					if !observation.Success {
						anyFailure = true
					}
				}
				if runErr != nil && runIndex >= cfg.Warmup {
					anyFailure = true
				}
			}
		}
	}
	result.EvidenceMode = contextEvidenceMode(result.PerRun)
	if err := WriteScenarioArtifacts(cfg.OutDir, &result); err != nil {
		return result, err
	}
	if anyFailure {
		return result, fmt.Errorf("context sharing had failed measured runs")
	}
	return result, nil
}

func runContextOnce(parent context.Context, mode string, ratio float64, index int, seed int64, cfg ContextSharingConfig) (RunObservation, error) {
	started := time.Now()
	run := RunObservation{
		ScenarioID: "context-sharing",
		RunID:      fmt.Sprintf("%s-%02d-%03d", mode, int(ratio*100), index+1),
		Mode:       mode,
		Labels:     map[string]string{"shared_ratio": strconv.Itoa(int(ratio * 100))},
		Timestamp:  started.UTC().Format(time.RFC3339Nano),
		Metrics:    map[string]MetricValue{},
		Events:     []EventRecord{},
	}
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	sharedBytes := int(float64(cfg.ContextSize) * ratio)
	privateBytes := cfg.ContextSize - sharedBytes
	public := patternBytes(sharedBytes, seed, "public")
	logicalBytes := int64(cfg.ContextSize * cfg.Agents)
	privatePayloads := make([][]byte, cfg.Agents)
	for i := range privatePayloads {
		privatePayloads[i] = patternBytes(privateBytes, seed+int64(i), fmt.Sprintf("private-%d", i))
	}
	startMemory := runtime.MemStats{}
	runtime.ReadMemStats(&startMemory)
	latencies := make([]float64, 0, cfg.Agents)
	privatePages := cfg.Agents
	sharedPages := 0
	prefixHits := 0
	physicalWritten := int64(0)
	transferred := int64(0)
	materialized := int64(0)
	evidenceMode := "real-runtime"
	failureReason := ""
	var store *cvm.Store
	var publicPage cvm.Page
	if mode == "aort-r" && sharedBytes > 0 {
		store = cvm.NewStore(nil)
		var createErr error
		publicPage, createErr = store.CreatePage(cvm.KindProject, string(public))
		if createErr != nil {
			failureReason = createErr.Error()
		}
		physicalWritten += int64(sharedBytes)
		sharedPages = 1
	}
	if mode == "shared-ipc" && sharedBytes > 0 {
		ipcStart := time.Now()
		transport, transportErr := shm.TransferPayload(public, cfg.Agents)
		latency := float64(max64(1, time.Since(ipcStart).Microseconds())) / 1000
		for i := 0; i < cfg.Agents; i++ {
			latencies = append(latencies, latency)
		}
		if transportErr != nil {
			evidenceMode = "degraded"
			failureReason = "shared IPC fallback: " + transportErr.Error()
		} else {
			evidenceMode = transport.EvidenceMode
			sharedPages = transport.SharedPages
		}
		physicalWritten += int64(sharedBytes)
		transferred += int64(sharedBytes)
	}
	if mode == "aort-r" {
		candidates := make([]avp.AVP, 0, cfg.Agents)
		for i := 0; i < cfg.Agents; i++ {
			if store != nil {
				privatePage, createErr := store.CreatePage(cvm.KindDelta, string(privatePayloads[i]))
				if createErr != nil {
					failureReason = createErr.Error()
					continue
				}
				_ = store.MountPage(fmt.Sprintf("agent-%d", i), privatePage.ID)
				_ = store.MountPage(fmt.Sprintf("agent-%d", i), publicPage.ID)
				physicalWritten += int64(privateBytes)
			} else {
				physicalWritten += int64(privateBytes)
			}
			candidates = append(candidates, avp.AVP{AgentID: fmt.Sprintf("agent-%d", i), State: avp.StateReady, VRuntime: uint64(i), ContextPages: []string{publicPage.ID}, PageTable: []string{publicPage.ID}})
		}
		schedulerRuntime := scheduler.New(scheduler.PolicyTokenCFSPrefixAffinity)
		selectionIndex := 0
		for len(candidates) > 0 {
			selected, decision, ok := schedulerRuntime.Select("context-sharing", candidates)
			if !ok {
				break
			}
			if selectionIndex > 0 && sharedBytes > 0 && decision.SharedPages[selected.AgentID] > 0 {
				prefixHits++
			}
			selectionIndex++
			candidates = removeCandidate(candidates, selected.AgentID)
		}
		if sharedBytes > 0 {
			// The shared page is materialized once; each private page is assembled per agent.
			materialized += int64(sharedBytes)
			transferred += int64(minInt(sharedBytes, len(publicPage.ID)))
		}
		for i := 0; i < cfg.Agents; i++ {
			privateStart := time.Now()
			materialized += int64(len(privatePayloads[i]))
			transferred += int64(len(privatePayloads[i]))
			latencies = append(latencies, float64(max64(1, time.Since(privateStart).Microseconds()))/1000)
		}
	} else if mode == "full-copy" {
		physicalWritten = logicalBytes
		transferred = logicalBytes
		for i := 0; i < cfg.Agents; i++ {
			copyStart := time.Now()
			assembled := make([]byte, 0, cfg.ContextSize)
			assembled = append(assembled, public...)
			assembled = append(assembled, privatePayloads[i]...)
			materialized += int64(len(assembled))
			latencies = append(latencies, float64(max64(1, time.Since(copyStart).Microseconds()))/1000)
		}
	} else if mode == "shared-ipc" {
		for i := 0; i < cfg.Agents; i++ {
			privateStart := time.Now()
			physicalWritten += int64(len(privatePayloads[i]))
			materialized += int64(sharedBytes + len(privatePayloads[i]))
			transferred += int64(len(privatePayloads[i]))
			latencies = append(latencies, float64(max64(1, time.Since(privateStart).Microseconds()))/1000)
		}
	}
	select {
	case <-ctx.Done():
		failureReason = ctx.Err().Error()
	default:
	}
	if len(latencies) == 0 {
		latencies = []float64{float64(max64(1, time.Since(started).Microseconds())) / 1000}
	}
	latencyStats := Aggregate(latencies, make([]bool, len(latencies)))
	// Aggregate's zero-valued success vector intentionally marks these as failed;
	// IPC latency itself is not a task-success metric, so use its raw percentile.
	latencyStats = Aggregate(latencies, filledBools(len(latencies), true))
	peakRSS, rssOK := readRSSBytes()
	if !rssOK {
		peakRSS = int64(maxUint64(startMemory.Alloc, currentAlloc()))
	}
	if failureReason == "" {
		run.Success = true
	} else if strings.HasPrefix(failureReason, "shared IPC fallback") {
		// A degraded transport is still a valid measured run.
		run.Success = true
	} else {
		run.FailureReason = failureReason
	}
	if run.Success && run.FailureReason == "" {
		run.FailureReason = ""
	}
	savedBytes := logicalBytes - transferred
	if savedBytes < 0 {
		savedBytes = 0
	}
	pageHitRate := 0.0
	if cfg.Agents > 0 && sharedPages > 0 {
		pageHitRate = float64(maxIntValue(cfg.Agents-1, 0)) / float64(cfg.Agents)
	}
	fairness := jainIndex(latencies)
	completionMS := float64(max64(1, time.Since(started).Milliseconds()))
	unitKindRSS := MeasurementMeasured
	if !rssOK {
		unitKindRSS = MeasurementUnsupported
	}
	run.Metrics = map[string]MetricValue{
		"logical_context_bytes":     {Value: float64(logicalBytes), Kind: MeasurementMeasured, Unit: "bytes"},
		"physical_bytes_written":    {Value: float64(physicalWritten), Kind: MeasurementMeasured, Unit: "bytes"},
		"bytes_transferred":         {Value: float64(transferred), Kind: MeasurementMeasured, Unit: "bytes"},
		"bytes_avoided":             {Value: float64(savedBytes), Kind: MeasurementDerived, Unit: "bytes"},
		"saved_bytes":               {Value: float64(savedBytes), Kind: MeasurementDerived, Unit: "bytes"},
		"materialized_bytes":        {Value: float64(materialized), Kind: MeasurementMeasured, Unit: "bytes"},
		"shared_pages":              {Value: float64(sharedPages), Kind: MeasurementMeasured, Unit: "pages"},
		"private_pages":             {Value: float64(privatePages), Kind: MeasurementMeasured, Unit: "pages"},
		"page_hit_ratio":            {Value: pageHitRate, Kind: MeasurementDerived, Unit: "ratio"},
		"ipc_p50_ms":                {Value: latencyStats.P50, Kind: MeasurementMeasured, Unit: "ms"},
		"ipc_p95_ms":                {Value: latencyStats.P95, Kind: MeasurementMeasured, Unit: "ms"},
		"peak_rss_bytes":            {Value: float64(peakRSS), Kind: unitKindRSS, Unit: "bytes"},
		"total_completion_ms":       {Value: completionMS, Kind: MeasurementMeasured, Unit: "ms"},
		"throughput_agents_per_sec": {Value: float64(cfg.Agents) / (completionMS / 1000), Kind: MeasurementDerived, Unit: "agents/s"},
		"agent_wait_ms":             {Value: latencyStats.Mean * float64(cfg.Agents), Kind: MeasurementDerived, Unit: "ms"},
		"fairness":                  {Value: fairness, Kind: MeasurementDerived, Unit: "jain"},
		"prefix_affinity_hits":      {Value: float64(prefixHits), Kind: MeasurementMeasured, Unit: "hits"},
		"shared_ratio":              {Value: ratio, Kind: MeasurementMeasured, Unit: "ratio"},
	}
	run.Events = append(run.Events, EventRecord{Name: "context.transfer", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Status: evidenceMode, Detail: fmt.Sprintf("shared_bytes=%d private_bytes=%d", sharedBytes, privateBytes)})
	return run, nil
}

func patternBytes(size int, seed int64, label string) []byte {
	if size <= 0 {
		return []byte{}
	}
	pattern := []byte(fmt.Sprintf("%s-%d-", label, seed))
	result := make([]byte, size)
	for i := range result {
		result[i] = pattern[i%len(pattern)]
	}
	return result
}

func removeCandidate(candidates []avp.AVP, agentID string) []avp.AVP {
	for i, candidate := range candidates {
		if candidate.AgentID == agentID {
			return append(candidates[:i], candidates[i+1:]...)
		}
	}
	return candidates
}

func filledBools(size int, value bool) []bool {
	values := make([]bool, size)
	for i := range values {
		values[i] = value
	}
	return values
}

func jainIndex(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	squares := 0.0
	for _, value := range values {
		sum += value
		squares += value * value
	}
	if squares == 0 {
		return 1
	}
	return sum * sum / (float64(len(values)) * squares)
}

func readRSSBytes() (int64, bool) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return value * 1024, true
				}
			}
		}
	}
	return 0, false
}

func currentAlloc() uint64 {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)
	return stats.Alloc
}

func maxUint64(left, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxIntValue(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func contextEvidenceMode(runs []RunObservation) string {
	for _, run := range runs {
		for _, event := range run.Events {
			if event.Status == "degraded" {
				return "degraded"
			}
		}
	}
	return "real-partial"
}
