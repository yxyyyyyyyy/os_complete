//go:build !linux

package ebpf

func platformSmoke() SmokeResult {
	return degraded(false, false, false, "not linux: real eBPF observer requires Linux tracepoints")
}
