package codebasedag

import (
	"strings"
	"testing"
)

func TestValidateOSProbeAcceptsRealOpenEulerStack(t *testing.T) {
	result := OSProbeResult{
		OS: OSRelease{ID: "openEuler", VersionID: "24.03", PrettyName: "openEuler 24.03 LTS"},
		Cgroup: CgroupProbe{FilesystemType: "cgroup2fs", Writable: true, NestedCreate: true, EvidenceMode: "real-cgroup-v2"},
		Overlay: OverlayProbe{Available: true, MountSucceeded: true, EvidenceMode: "real-overlayfs"},
		Memfd: MemfdProbe{Available: true, MmapSucceeded: true, FDPassingSucceeded: true, EvidenceMode: "real-memfd"},
	}
	if err := result.ValidateOpenWorld(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOSProbeRejectsDegradedModes(t *testing.T) {
	result := OSProbeResult{
		OS: OSRelease{ID: "ubuntu", VersionID: "22.04"},
		Cgroup: CgroupProbe{FilesystemType: "tmpfs", Writable: false, EvidenceMode: "degraded"},
		Overlay: OverlayProbe{Available: true, MountSucceeded: false, EvidenceMode: "degraded"},
		Memfd: MemfdProbe{Available: false, EvidenceMode: "unsupported"},
	}
	err := result.ValidateOpenWorld()
	if err == nil {
		t.Fatal("degraded probe should fail")
	}
	for _, want := range []string{"openEuler", "cgroup2fs", "overlay", "memfd"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestParseOSRelease(t *testing.T) {
	release := ParseOSRelease([]byte("ID=openEuler\nVERSION_ID=\"24.03\"\nPRETTY_NAME=\"openEuler 24.03 LTS\"\n"))
	if release.ID != "openEuler" || release.VersionID != "24.03" || release.PrettyName != "openEuler 24.03 LTS" {
		t.Fatalf("release = %#v", release)
	}
}

func TestProbeCollectorUsesFakeSources(t *testing.T) {
	collector := ProbeCollector{
		ReadOSRelease: func() ([]byte, error) {
			return []byte("ID=openEuler\nVERSION_ID=24.03\n"), nil
		},
		Cgroup: func() CgroupProbe {
			return CgroupProbe{FilesystemType: "cgroup2fs", Writable: true, NestedCreate: true, EvidenceMode: "real-cgroup-v2"}
		},
		Overlay: func() OverlayProbe {
			return OverlayProbe{Available: true, MountSucceeded: true, EvidenceMode: "real-overlayfs"}
		},
		Memfd: func() MemfdProbe {
			return MemfdProbe{Available: true, MmapSucceeded: true, FDPassingSucceeded: true, EvidenceMode: "real-memfd"}
		},
	}
	result, err := collector.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if err := result.ValidateOpenWorld(); err != nil {
		t.Fatal(err)
	}
}
