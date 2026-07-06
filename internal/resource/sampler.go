package resource

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aort-r/internal/avp"
	"aort-r/internal/evidence"
	"aort-r/internal/scheduler"
)

type ResourceSampler interface {
	Sample(agent avp.AVP) (scheduler.ResourcePressure, error)
}

type CgroupSampler struct {
	pressureRoot string
}

func NewCgroupSampler(pressureRoot string) *CgroupSampler {
	if pressureRoot == "" {
		pressureRoot = "/proc/pressure"
	}
	return &CgroupSampler{pressureRoot: pressureRoot}
}

func (s *CgroupSampler) Sample(agent avp.AVP) (scheduler.ResourcePressure, error) {
	_, pressure, err := s.Enrich(agent)
	return pressure, err
}

func (s *CgroupSampler) Enrich(agent avp.AVP) (avp.AVP, scheduler.ResourcePressure, error) {
	if agent.CgroupPath == "" {
		return agent, degradedPressure("agent cgroup path is empty"), fmt.Errorf("agent %q cgroup path is empty", agent.AgentID)
	}
	memoryCurrent, err := readInt64(filepath.Join(agent.CgroupPath, "memory.current"))
	if err != nil {
		return agent, degradedPressure("read memory.current: " + err.Error()), err
	}
	memoryMax, err := readMax(filepath.Join(agent.CgroupPath, "memory.max"))
	if err != nil {
		return agent, degradedPressure("read memory.max: " + err.Error()), err
	}
	pidsCurrent, err := readInt64(filepath.Join(agent.CgroupPath, "pids.current"))
	if err != nil {
		return agent, degradedPressure("read pids.current: " + err.Error()), err
	}
	pidsMax, err := readMax(filepath.Join(agent.CgroupPath, "pids.max"))
	if err != nil {
		return agent, degradedPressure("read pids.max: " + err.Error()), err
	}
	cpuStat, err := readKV(filepath.Join(agent.CgroupPath, "cpu.stat"))
	if err != nil {
		return agent, degradedPressure("read cpu.stat: " + err.Error()), err
	}
	agent.MemoryCurrent = memoryCurrent
	agent.PidsCurrent = pidsCurrent
	agent.CPUStat = cpuStat
	psi, psiErr := s.psiPressure()
	pressure := scheduler.ResourcePressure{
		MemoryPressure:      ratio(memoryCurrent, memoryMax),
		PidsPressure:        ratio(pidsCurrent, pidsMax),
		CPUThrottlePressure: cpuThrottlePressure(cpuStat),
		PSIPressure:         psi,
		EvidenceMode:        evidence.ModeRealCgroupV2,
	}
	if psiErr != nil {
		pressure.EvidenceMode = evidence.ModeDegraded
		pressure.FallbackReason = "read PSI pressure: " + psiErr.Error()
		return agent, pressure, psiErr
	}
	return agent, pressure, nil
}

func degradedPressure(reason string) scheduler.ResourcePressure {
	return scheduler.ResourcePressure{
		EvidenceMode:   evidence.ModeDegraded,
		FallbackReason: reason,
	}
}

func readInt64(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

func readMax(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	text := strings.TrimSpace(string(data))
	if text == "max" {
		return 0, nil
	}
	return strconv.ParseInt(text, 10, 64)
}

func readKV(path string) (map[string]uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]uint64{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		out[fields[0]] = value
	}
	return out, nil
}

func (s *CgroupSampler) psiPressure() (float64, error) {
	maxAvg := 0.0
	for _, name := range []string{"cpu", "memory", "io"} {
		avg, err := readPSIAvg10(filepath.Join(s.pressureRoot, name))
		if err != nil {
			return 0, err
		}
		if avg > maxAvg {
			maxAvg = avg
		}
	}
	return round3(maxAvg / 100), nil
}

func readPSIAvg10(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	for _, field := range strings.Fields(string(data)) {
		if !strings.HasPrefix(field, "avg10=") {
			continue
		}
		return strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
	}
	return 0, fmt.Errorf("avg10 missing in %s", path)
}

func ratio(current, max int64) float64 {
	if current <= 0 || max <= 0 {
		return 0
	}
	return round3(float64(current) / float64(max))
}

func cpuThrottlePressure(stat map[string]uint64) float64 {
	if len(stat) == 0 {
		return 0
	}
	byCount := float64(stat["nr_throttled"]) / 100
	byUsec := float64(stat["throttled_usec"]) / 10_000_000
	return round3(math.Max(clamp(byCount), clamp(byUsec)))
}

func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
