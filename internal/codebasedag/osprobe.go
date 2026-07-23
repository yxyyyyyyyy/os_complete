package codebasedag

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type OSProbeResult struct {
	OS      OSRelease    `json:"os"`
	Cgroup  CgroupProbe  `json:"cgroup"`
	Overlay OverlayProbe `json:"overlay"`
	Memfd   MemfdProbe   `json:"memfd"`
}

type OSRelease struct {
	ID         string `json:"id"`
	VersionID  string `json:"version_id"`
	PrettyName string `json:"pretty_name"`
}

type CgroupProbe struct {
	FilesystemType string `json:"filesystem_type"`
	Writable       bool   `json:"writable"`
	NestedCreate   bool   `json:"nested_create"`
	EvidenceMode   string `json:"evidence_mode"`
	Error          string `json:"error,omitempty"`
}

type OverlayProbe struct {
	Available      bool   `json:"available"`
	MountSucceeded bool   `json:"mount_succeeded"`
	EvidenceMode   string `json:"evidence_mode"`
	Error          string `json:"error,omitempty"`
}

type MemfdProbe struct {
	Available          bool   `json:"available"`
	MmapSucceeded      bool   `json:"mmap_succeeded"`
	FDPassingSucceeded bool   `json:"fd_passing_succeeded"`
	EvidenceMode       string `json:"evidence_mode"`
	Error              string `json:"error,omitempty"`
}

func (r OSProbeResult) ValidateOpenWorld() error {
	var failures []string
	if !strings.EqualFold(r.OS.ID, "openEuler") || !strings.HasPrefix(r.OS.VersionID, "24.03") {
		failures = append(failures, "openEuler 24.03 required")
	}
	if r.Cgroup.FilesystemType != "cgroup2fs" || !r.Cgroup.Writable || !r.Cgroup.NestedCreate || r.Cgroup.EvidenceMode != "real-cgroup-v2" {
		failures = append(failures, "cgroup2fs real-cgroup-v2 writable nested cgroups required")
	}
	if !r.Overlay.Available || !r.Overlay.MountSucceeded || r.Overlay.EvidenceMode != "real-overlayfs" {
		failures = append(failures, "overlay real-overlayfs mount required")
	}
	if !r.Memfd.Available || !r.Memfd.MmapSucceeded || !r.Memfd.FDPassingSucceeded || r.Memfd.EvidenceMode != "real-memfd" {
		failures = append(failures, "memfd mmap FD-passing real evidence required")
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf(strings.Join(failures, "; "))
	}
	return nil
}

func ParseOSRelease(data []byte) OSRelease {
	values := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return OSRelease{ID: values["ID"], VersionID: values["VERSION_ID"], PrettyName: values["PRETTY_NAME"]}
}

type ProbeCollector struct {
	ReadOSRelease func() ([]byte, error)
	Cgroup        func() CgroupProbe
	Overlay       func() OverlayProbe
	Memfd         func() MemfdProbe
}

func (c ProbeCollector) Collect() (OSProbeResult, error) {
	readOS := c.ReadOSRelease
	if readOS == nil {
		readOS = func() ([]byte, error) { return os.ReadFile("/etc/os-release") }
	}
	raw, err := readOS()
	if err != nil {
		return OSProbeResult{}, err
	}
	cgroup := CgroupProbe{EvidenceMode: "unsupported", Error: "cgroup probe not configured"}
	if c.Cgroup != nil {
		cgroup = c.Cgroup()
	}
	overlay := OverlayProbe{EvidenceMode: "unsupported", Error: "overlay probe not configured"}
	if c.Overlay != nil {
		overlay = c.Overlay()
	}
	memfd := MemfdProbe{EvidenceMode: "unsupported", Error: "memfd probe not configured"}
	if c.Memfd != nil {
		memfd = c.Memfd()
	}
	return OSProbeResult{OS: ParseOSRelease(raw), Cgroup: cgroup, Overlay: overlay, Memfd: memfd}, nil
}
