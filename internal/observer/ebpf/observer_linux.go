//go:build linux

package ebpf

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	bpfMapCreate       = 0
	bpfMapLookupElem   = 1
	bpfProgLoad        = 5
	bpfMapTypeHash     = 1
	bpfProgTracepoint  = 2
	bpfAny             = 0
	perfTypeTracepoint = 2
	perfFlagFDCloexec  = 8

	perfEventIOCEnable = 0x2400
	perfEventIOCSetBPF = 0x40042408

	sysBPF = 321
)

type bpfInsn struct {
	Code   uint8
	DstSrc uint8
	Off    int16
	Imm    int32
}

type bpfMapCreateAttr struct {
	MapType    uint32
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
	MapFlags   uint32
	InnerMapFD uint32
	NumaNode   uint32
	MapName    [16]byte
	MapIfIndex uint32
}

type bpfProgLoadAttr struct {
	ProgType           uint32
	InsnCnt            uint32
	Insns              uint64
	License            uint64
	LogLevel           uint32
	LogSize            uint32
	LogBuf             uint64
	KernVersion        uint32
	ProgFlags          uint32
	ProgName           [16]byte
	ProgIfIndex        uint32
	ExpectedAttachType uint32
}

type bpfMapElemAttr struct {
	MapFD uint32
	Pad   uint32
	Key   uint64
	Value uint64
	Flags uint64
}

type perfEventAttr struct {
	Type         uint32
	Size         uint32
	Config       uint64
	SamplePeriod uint64
	SampleType   uint64
	ReadFormat   uint64
	Bits         uint64
	WakeupEvents uint32
	BPType       uint32
	BPAddr       uint64
	BPLen        uint64
}

func platformSmoke() SmokeResult {
	traceRoot, traceOK := findTraceFS()
	capable := rootOrCapable()
	if !traceOK {
		return degraded(true, capable, false, "tracepoint unavailable: tracefs/debugfs sched_process_exit id not found")
	}
	if !capable {
		return degraded(true, false, true, "no permission: requires root or CAP_BPF/CAP_PERFMON/CAP_SYS_ADMIN")
	}

	mapFD, err := createPIDMap()
	if err != nil {
		return degraded(true, true, true, "map create failed: "+err.Error())
	}
	cleanupOK := true
	defer func() {
		if err := syscall.Close(mapFD); err != nil {
			cleanupOK = false
		}
	}()

	progFD, verifierLog, err := loadTracepointProgram(mapFD)
	if err != nil {
		result := degraded(true, true, true, "verifier failed: "+err.Error())
		result.VerifierLog = verifierLog
		return result
	}
	defer func() {
		if err := syscall.Close(progFD); err != nil {
			cleanupOK = false
		}
	}()

	tpID, err := tracepointID(traceRoot, "sched", "sched_process_exit")
	if err != nil {
		result := degraded(true, true, true, "tracepoint unavailable: "+err.Error())
		result.ProgramLoaded = true
		return result
	}
	perfFDs, err := attachTracepointAllCPUs(tpID, progFD)
	if err != nil {
		result := degraded(true, true, true, "attach failed: "+err.Error())
		result.ProgramLoaded = true
		return result
	}
	defer func() {
		for _, fd := range perfFDs {
			if err := syscall.Close(fd); err != nil {
				cleanupOK = false
			}
		}
	}()

	worker := exec.Command("sh", "-c", "sleep 0.05")
	if err := worker.Start(); err != nil {
		result := degraded(true, true, true, "worker start failed: "+err.Error())
		result.ProgramLoaded = true
		result.AttachedTracepoints = []string{"sched_process_exit"}
		return result
	}
	workerPID := worker.Process.Pid
	_ = worker.Wait()

	observed := lookupPID(mapFD, uint32(workerPID))
	events := 0
	if observed {
		events = 1
	}
	mode := "degraded"
	reason := "worker pid was not observed by sched_process_exit eBPF program"
	if observed {
		mode = "real-ebpf"
		reason = ""
	}
	return SmokeResult{
		Observer:            "ebpf",
		EvidenceMode:        mode,
		Linux:               true,
		RootOrCapable:       true,
		ProgramLoaded:       true,
		AttachedTracepoints: []string{"sched_process_exit"},
		EventsCollected:     events,
		WorkerPIDObserved:   observed,
		WorkerPID:           workerPID,
		CleanupSuccess:      cleanupOK,
		FallbackReason:      reason,
		TraceFSAvailable:    true,
		VerifierLog:         verifierLog,
	}
}

func createPIDMap() (int, error) {
	attr := bpfMapCreateAttr{MapType: bpfMapTypeHash, KeySize: 4, ValueSize: 8, MaxEntries: 256}
	copy(attr.MapName[:], "aort_pids")
	fd, _, errno := syscall.Syscall(sysBPF, bpfMapCreate, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr))
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func loadTracepointProgram(mapFD int) (int, string, error) {
	insns := []bpfInsn{
		insn(0x85, 0, 0, 0, 14),
		insn(0xbf, reg(2, 0), 0, 0, 0),
		insn(0x77, reg(2, 0), 0, 0, 32),
		insn(0x63, reg(10, 2), -4, 0, 0),
		insn(0x7a, reg(10, 0), -16, 0, 1),
		insn(0x18, reg(1, 1), 0, 0, int32(mapFD)),
		insn(0x00, 0, 0, 0, 0),
		insn(0xbf, reg(2, 10), 0, 0, 0),
		insn(0x07, reg(2, 0), 0, 0, -4),
		insn(0xbf, reg(3, 10), 0, 0, 0),
		insn(0x07, reg(3, 0), 0, 0, -16),
		insn(0xb7, reg(4, 0), 0, 0, bpfAny),
		insn(0x85, 0, 0, 0, 2),
		insn(0xb7, reg(0, 0), 0, 0, 0),
		insn(0x95, 0, 0, 0, 0),
	}
	license := append([]byte("GPL"), 0)
	logBuf := bytes.Repeat([]byte{0}, 64*1024)
	attr := bpfProgLoadAttr{
		ProgType:    bpfProgTracepoint,
		InsnCnt:     uint32(len(insns)),
		Insns:       uint64(uintptr(unsafe.Pointer(&insns[0]))),
		License:     uint64(uintptr(unsafe.Pointer(&license[0]))),
		LogLevel:    1,
		LogSize:     uint32(len(logBuf)),
		LogBuf:      uint64(uintptr(unsafe.Pointer(&logBuf[0]))),
		KernVersion: 0,
	}
	copy(attr.ProgName[:], "aort_exit")
	fd, _, errno := syscall.Syscall(sysBPF, bpfProgLoad, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr))
	logText := strings.TrimRight(string(logBuf), "\x00")
	if errno != 0 {
		return -1, logText, errno
	}
	return int(fd), logText, nil
}

func attachTracepointAllCPUs(tracepointID, progFD int) ([]int, error) {
	cpus := onlineCPUs()
	fds := make([]int, 0, len(cpus))
	for _, cpu := range cpus {
		fd, err := perfEventOpen(tracepointID, cpu)
		if err != nil {
			closeAll(fds)
			return nil, err
		}
		if err := ioctl(fd, perfEventIOCSetBPF, progFD); err != nil {
			_ = syscall.Close(fd)
			closeAll(fds)
			return nil, err
		}
		if err := ioctl(fd, perfEventIOCEnable, 0); err != nil {
			_ = syscall.Close(fd)
			closeAll(fds)
			return nil, err
		}
		fds = append(fds, fd)
	}
	return fds, nil
}

func perfEventOpen(tracepointID, cpu int) (int, error) {
	attr := perfEventAttr{Type: perfTypeTracepoint, Size: uint32(unsafe.Sizeof(perfEventAttr{})), Config: uint64(tracepointID)}
	fd, _, errno := syscall.Syscall6(syscall.SYS_PERF_EVENT_OPEN, uintptr(unsafe.Pointer(&attr)), ^uintptr(0), uintptr(cpu), ^uintptr(0), perfFlagFDCloexec, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func lookupPID(mapFD int, pid uint32) bool {
	var value uint64
	attr := bpfMapElemAttr{
		MapFD: uint32(mapFD),
		Key:   uint64(uintptr(unsafe.Pointer(&pid))),
		Value: uint64(uintptr(unsafe.Pointer(&value))),
	}
	_, _, errno := syscall.Syscall(sysBPF, bpfMapLookupElem, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr))
	return errno == 0 && value > 0
}

func tracepointID(root, category, name string) (int, error) {
	raw, err := os.ReadFile(filepath.Join(root, "events", category, name, "id"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(raw)))
}

func findTraceFS() (string, bool) {
	for _, root := range []string{"/sys/kernel/tracing", "/sys/kernel/debug/tracing"} {
		if _, err := os.Stat(filepath.Join(root, "events", "sched", "sched_process_exit", "id")); err == nil {
			return root, true
		}
	}
	return "", false
}

func rootOrCapable() bool {
	if os.Geteuid() == 0 {
		return true
	}
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "CapEff:") {
			continue
		}
		hexValue := strings.TrimSpace(strings.TrimPrefix(line, "CapEff:"))
		value, err := strconv.ParseUint(hexValue, 16, 64)
		if err != nil {
			return false
		}
		return hasCap(value, 21) || hasCap(value, 38) || hasCap(value, 39)
	}
	return false
}

func hasCap(value uint64, cap int) bool {
	return value&(uint64(1)<<cap) != 0
}

func onlineCPUs() []int {
	raw, err := os.ReadFile("/sys/devices/system/cpu/online")
	if err != nil {
		return []int{0}
	}
	cpus := []int{}
	for _, part := range strings.Split(strings.TrimSpace(string(raw)), ",") {
		if start, end, ok := strings.Cut(part, "-"); ok {
			left, err1 := strconv.Atoi(start)
			right, err2 := strconv.Atoi(end)
			if err1 != nil || err2 != nil || right < left {
				continue
			}
			for cpu := left; cpu <= right; cpu++ {
				cpus = append(cpus, cpu)
			}
			continue
		}
		cpu, err := strconv.Atoi(part)
		if err == nil {
			cpus = append(cpus, cpu)
		}
	}
	if len(cpus) == 0 {
		return []int{0}
	}
	return cpus
}

func insn(code uint8, dstSrc uint8, off int16, _ int32, imm int32) bpfInsn {
	return bpfInsn{Code: code, DstSrc: dstSrc, Off: off, Imm: imm}
}

func reg(dst, src int) uint8 {
	return uint8(dst&0x0f) | uint8((src&0x0f)<<4)
}

func ioctl(fd int, request uintptr, arg int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func closeAll(fds []int) {
	for _, fd := range fds {
		_ = syscall.Close(fd)
	}
}
