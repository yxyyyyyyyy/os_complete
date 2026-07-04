package pressure

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ModePSI      = "psi"
	ModeDegraded = "degraded"
)

type Line struct {
	Kind   string  `json:"kind"`
	Avg10  float64 `json:"avg10"`
	Avg60  float64 `json:"avg60"`
	Avg300 float64 `json:"avg300"`
	Total  uint64  `json:"total"`
}

type Resource struct {
	Some Line `json:"some"`
	Full Line `json:"full"`
}

type Status struct {
	Mode           string   `json:"mode"`
	Degraded       bool     `json:"degraded"`
	Reason         string   `json:"reason,omitempty"`
	CPU            Resource `json:"cpu"`
	Memory         Resource `json:"memory"`
	IO             Resource `json:"io"`
	Throttle       bool     `json:"throttle"`
	ThrottleReason string   `json:"throttle_reason,omitempty"`
	SampledAt      int64    `json:"sampled_at"`
}

type Config struct {
	Root                 string
	CPUPath              string
	MemoryPath           string
	IOPath               string
	CPUAvg10Threshold    float64
	MemoryAvg10Threshold float64
	IOAvg10Threshold     float64
}

type Monitor struct {
	cfg Config
}

func NewMonitor(cfg Config) *Monitor {
	if cfg.Root == "" {
		cfg.Root = "/proc/pressure"
	}
	if cfg.CPUPath == "" {
		cfg.CPUPath = filepath.Join(cfg.Root, "cpu")
		if _, err := os.Stat(cfg.CPUPath); err != nil {
			cfg.CPUPath = filepath.Join(cfg.Root, "cpu.pressure")
		}
	}
	if cfg.MemoryPath == "" {
		cfg.MemoryPath = filepath.Join(cfg.Root, "memory")
		if _, err := os.Stat(cfg.MemoryPath); err != nil {
			cfg.MemoryPath = filepath.Join(cfg.Root, "memory.pressure")
		}
	}
	if cfg.IOPath == "" {
		cfg.IOPath = filepath.Join(cfg.Root, "io")
		if _, err := os.Stat(cfg.IOPath); err != nil {
			cfg.IOPath = filepath.Join(cfg.Root, "io.pressure")
		}
	}
	if cfg.CPUAvg10Threshold == 0 {
		cfg.CPUAvg10Threshold = 50
	}
	if cfg.MemoryAvg10Threshold == 0 {
		cfg.MemoryAvg10Threshold = 20
	}
	if cfg.IOAvg10Threshold == 0 {
		cfg.IOAvg10Threshold = 30
	}
	return &Monitor{cfg: cfg}
}

func (m *Monitor) Sample() Status {
	status := Status{Mode: ModePSI, SampledAt: time.Now().UnixMilli()}
	var reasons []string
	cpu, err := readResource(m.cfg.CPUPath)
	if err != nil {
		reasons = append(reasons, "cpu: "+err.Error())
	}
	memory, err := readResource(m.cfg.MemoryPath)
	if err != nil {
		reasons = append(reasons, "memory: "+err.Error())
	}
	ioPressure, err := readResource(m.cfg.IOPath)
	if err != nil {
		reasons = append(reasons, "io: "+err.Error())
	}
	if len(reasons) > 0 {
		status.Mode = ModeDegraded
		status.Degraded = true
		status.Reason = strings.Join(reasons, "; ")
		return status
	}
	status.CPU = cpu
	status.Memory = memory
	status.IO = ioPressure
	var throttle []string
	if status.CPU.Some.Avg10 >= m.cfg.CPUAvg10Threshold {
		throttle = append(throttle, fmt.Sprintf("cpu avg10 %.2f >= %.2f", status.CPU.Some.Avg10, m.cfg.CPUAvg10Threshold))
	}
	if status.Memory.Some.Avg10 >= m.cfg.MemoryAvg10Threshold || status.Memory.Full.Avg10 >= m.cfg.MemoryAvg10Threshold {
		throttle = append(throttle, fmt.Sprintf("memory avg10 some=%.2f full=%.2f threshold=%.2f", status.Memory.Some.Avg10, status.Memory.Full.Avg10, m.cfg.MemoryAvg10Threshold))
	}
	if status.IO.Some.Avg10 >= m.cfg.IOAvg10Threshold || status.IO.Full.Avg10 >= m.cfg.IOAvg10Threshold {
		throttle = append(throttle, fmt.Sprintf("io avg10 some=%.2f full=%.2f threshold=%.2f", status.IO.Some.Avg10, status.IO.Full.Avg10, m.cfg.IOAvg10Threshold))
	}
	if len(throttle) > 0 {
		status.Throttle = true
		status.ThrottleReason = strings.Join(throttle, "; ")
	}
	return status
}

func ParseLine(text string) (Line, error) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return Line{}, fmt.Errorf("empty PSI line")
	}
	line := Line{Kind: fields[0]}
	if line.Kind != "some" && line.Kind != "full" {
		return Line{}, fmt.Errorf("unsupported PSI kind %q", line.Kind)
	}
	for _, field := range fields[1:] {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return Line{}, fmt.Errorf("invalid PSI field %q", field)
		}
		switch key {
		case "avg10":
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Line{}, err
			}
			line.Avg10 = parsed
		case "avg60":
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Line{}, err
			}
			line.Avg60 = parsed
		case "avg300":
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Line{}, err
			}
			line.Avg300 = parsed
		case "total":
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return Line{}, err
			}
			line.Total = parsed
		}
	}
	return line, nil
}

func readResource(path string) (Resource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Resource{}, err
	}
	var resource Resource
	for _, raw := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		line, err := ParseLine(raw)
		if err != nil {
			return Resource{}, err
		}
		switch line.Kind {
		case "some":
			resource.Some = line
		case "full":
			resource.Full = line
		}
	}
	if resource.Some.Kind == "" {
		return Resource{}, fmt.Errorf("missing some PSI line")
	}
	return resource, nil
}
