package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"aort-r/internal/experiment"
	"aort-r/internal/workspace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl experiment|demo|workspace|observer|ipc|cvm|replay ...")
	}
	switch args[0] {
	case "_hog":
		return runHog(args[1:])
	case "experiment":
		return runExperiment(args[1:])
	case "evidence":
		return runEvidence(args[1:])
	case "demo":
		return runDemo(args[1:])
	case "workspace":
		return runWorkspace(args[1:])
	case "observer":
		return runObserver(args[1:])
	case "ipc":
		return runIPC(args[1:])
	case "cvm":
		return runCVM(args[1:])
	case "replay":
		return runReplay(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runExperiment(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl experiment all|e1|e1-pressure|e2|e2-pressure-fault|real-cgroup-smoke|real-pressure-smoke|deepseek-real-smoke|real-all")
	}
	switch args[0] {
	case "all":
		fs := flag.NewFlagSet("experiment all", flag.ContinueOnError)
		runs := fs.Int("runs", 1, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results", "all"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunAllExperiments(experiment.AllExperimentsConfig{
			Runs:   *runs,
			OutDir: *out,
		})
		return err
	case "e1":
		fs := flag.NewFlagSet("experiment e1", flag.ContinueOnError)
		policy := fs.String("policy", "resource-aware", "scheduler policy: resource-aware or all")
		runs := fs.Int("runs", 5, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results", "e1"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *policy != "resource-aware" && *policy != "all" {
			return fmt.Errorf("unsupported e1 policy %q", *policy)
		}
		_, err := experiment.RunE1ResourceAwareBenchmark(*runs, *out)
		return err
	case "e1-pressure":
		fs := flag.NewFlagSet("experiment e1-pressure", flag.ContinueOnError)
		runs := fs.Int("runs", 5, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results", "e1_pressure"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunE1PressureBenchmark(*runs, *out)
		return err
	case "e2":
		fs := flag.NewFlagSet("experiment e2", flag.ContinueOnError)
		runs := fs.Int("runs", 5, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		results := experiment.RunE2RealFaultIsolation(*runs)
		if err := experiment.WriteJSON(filepath.Join(*out, "e2-real-fault.json"), results); err != nil {
			return err
		}
		return experiment.WriteCSV(filepath.Join(*out, "e2-real-fault.csv"), experiment.E2RealCSV(results))
	case "e2-pressure-fault":
		fs := flag.NewFlagSet("experiment e2-pressure-fault", flag.ContinueOnError)
		runs := fs.Int("runs", 5, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results", "e2_pressure_fault"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunE2PressureFault(*runs, *out)
		return err
	case "real-cgroup-smoke":
		fs := flag.NewFlagSet("experiment real-cgroup-smoke", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "real_cgroup_smoke"), "output directory")
		cgroupRoot := fs.String("cgroup-root", "", "cgroup v2 root for AORT test capsules")
		memoryMax := fs.String("memory-max", "67108864", "memory.max for worker cgroup in bytes")
		pidsMax := fs.String("pids-max", "8", "pids.max for worker cgroup")
		cpuMax := fs.String("cpu-max", "100000 100000", "cpu.max for worker cgroup")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunRealCgroupSmoke(experiment.RealCgroupSmokeConfig{
			CgroupRoot: *cgroupRoot,
			OutDir:     *out,
			MemoryMax:  *memoryMax,
			PidsMax:    *pidsMax,
			CPUMax:     *cpuMax,
		})
		return err
	case "real-pressure-smoke":
		fs := flag.NewFlagSet("experiment real-pressure-smoke", flag.ContinueOnError)
		runs := fs.Int("runs", 3, "number of scheduler selections")
		out := fs.String("out", filepath.Join("experiments", "results", "real_pressure_smoke"), "output directory")
		requireReal := fs.Bool("require-real", false, "fail instead of writing degraded pressure evidence")
		cgroupRoot := fs.String("cgroup-root", "", "cgroup v2 root for AORT pressure test capsules")
		memoryMB := fs.Int("memory-mb", 64, "bounded memory hog size in MiB")
		pids := fs.Int("pids", 8, "bounded pids hog child count")
		cpuMS := fs.Int("cpu-ms", 2000, "bounded CPU hog duration in milliseconds")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunRealPressureSmoke(experiment.RealPressureSmokeConfig{
			Runs:           *runs,
			OutDir:         *out,
			RequireReal:    *requireReal,
			CgroupRoot:     *cgroupRoot,
			MemoryHogBytes: int64(*memoryMB) * 1024 * 1024,
			PidsHogCount:   *pids,
			CPUHogDuration: time.Duration(*cpuMS) * time.Millisecond,
		})
		return err
	case "deepseek-real-smoke":
		fs := flag.NewFlagSet("experiment deepseek-real-smoke", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "deepseek_real"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunDeepSeekRealSmoke(experiment.DeepSeekRealSmokeConfigFromEnv(*out))
		return err
	case "real-all":
		fs := flag.NewFlagSet("experiment real-all", flag.ContinueOnError)
		runs := fs.Int("runs", 3, "number of runs")
		out := fs.String("out", filepath.Join("experiments", "results", "real_all"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunRealAll(*runs, *out)
		return err
	default:
		return fmt.Errorf("unknown experiment %q", args[0])
	}
}

func runObserver(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl observer ebpf-smoke")
	}
	switch args[0] {
	case "ebpf-smoke":
		fs := flag.NewFlagSet("observer ebpf-smoke", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "ebpf_smoke"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunEBPFSmoke(*out)
		return err
	default:
		return fmt.Errorf("unknown observer command %q", args[0])
	}
}

func runIPC(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl ipc shm-smoke")
	}
	switch args[0] {
	case "shm-smoke":
		fs := flag.NewFlagSet("ipc shm-smoke", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "ipc_shm"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunIPCShmSmoke(*out)
		return err
	default:
		return fmt.Errorf("unknown ipc command %q", args[0])
	}
}

func runCVM(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl cvm memory-smoke")
	}
	switch args[0] {
	case "memory-smoke":
		fs := flag.NewFlagSet("cvm memory-smoke", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "cvm_memory"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunCVMMemorySmoke(*out)
		return err
	default:
		return fmt.Errorf("unknown cvm command %q", args[0])
	}
}

func runReplay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	tracePath := fs.String("trace", filepath.Join("experiments", "results", "software_real_demo", "trace.json"), "trace JSON path")
	out := fs.String("out", filepath.Join("experiments", "results", "replay"), "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tracePath == "" {
		return fmt.Errorf("trace path is required")
	}
	_, err := experiment.RunReplay(*tracePath, *out)
	return err
}

func runEvidence(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl evidence final")
	}
	switch args[0] {
	case "final":
		fs := flag.NewFlagSet("evidence final", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "final"), "output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.WriteFinalEvidence(*out)
		return err
	default:
		return fmt.Errorf("unknown evidence command %q", args[0])
	}
}

func runWorkspace(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl workspace probe")
	}
	switch args[0] {
	case "probe":
		fs := flag.NewFlagSet("workspace probe", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results", "workspace_probe.json"), "output JSON path")
		root := fs.String("root", "", "runtime root for the probe workspace")
		requireReal := fs.Bool("require-real", false, "fail unless overlayfs is mounted and selected")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		result := workspace.ProbeOverlay(*root)
		if *requireReal && (result.EvidenceMode != "real-overlayfs" || !result.MountTestSuccess || !result.MergedIsMountpoint) {
			result.Success = false
			result.Error = "real-overlayfs required: " + result.FallbackReason
			if result.FallbackReason == "" {
				result.FallbackReason = "workspace probe did not mount overlayfs"
				result.Error = "real-overlayfs required: " + result.FallbackReason
			}
			if err := experiment.WriteJSON(*out, result); err != nil {
				return err
			}
			return fmt.Errorf("%s", result.Error)
		}
		return experiment.WriteJSON(*out, result)
	default:
		return fmt.Errorf("unknown workspace command %q", args[0])
	}
}

func runDemo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl demo software-real|fault")
	}
	switch args[0] {
	case "software-real":
		fs := flag.NewFlagSet("demo software-real", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results"), "output directory")
		runs := fs.Int("runs", 5, "number of benchmark runs")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunSoftwareRealDemo(*runs, *out)
		return err
	case "fault":
		if len(args) < 2 {
			return fmt.Errorf("usage: aortctl demo fault workspace-rmrf")
		}
		switch args[1] {
		case "workspace-rmrf":
			fs := flag.NewFlagSet("demo fault workspace-rmrf", flag.ContinueOnError)
			out := fs.String("out", filepath.Join("experiments", "results"), "output directory")
			root := fs.String("root", "", "runtime root for workspaces")
			requireReal := fs.Bool("require-real-overlayfs", false, "fail unless the demo uses real overlayfs")
			forceDegraded := fs.Bool("force-degraded", false, "force degraded-copy mode for tests")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			workspaceRoot := *root
			if workspaceRoot == "" {
				workspaceRoot = filepath.Join(*out, "workspace_rmrf_runtime")
			}
			result, err := workspace.RunRMFaultDemo(workspace.Config{Root: workspaceRoot, ForceDegraded: *forceDegraded})
			if err != nil {
				return err
			}
			outPath := filepath.Join(*out, "workspace_isolation_evidence.json")
			if *requireReal && (result.EvidenceMode != "real-overlayfs" || result.Mode != workspace.ModeOverlayFS || !result.MergedIsMountpoint) {
				result.Success = false
				if result.FallbackReason == "" {
					result.FallbackReason = "workspace-rmrf did not use mounted overlayfs"
				}
				result.Error = "real-overlayfs required: " + result.FallbackReason
				if err := experiment.WriteJSON(outPath, result); err != nil {
					return err
				}
				return fmt.Errorf("%s", result.Error)
			}
			return experiment.WriteJSON(outPath, result)
		default:
			return fmt.Errorf("unknown fault demo %q", args[1])
		}
	case "tool-workspace":
		fs := flag.NewFlagSet("demo tool-workspace", flag.ContinueOnError)
		out := fs.String("out", filepath.Join("experiments", "results"), "output directory")
		root := fs.String("root", "", "runtime root for workspaces")
		requireReal := fs.Bool("require-real-overlayfs", false, "fail unless tool.exec uses real overlayfs workspace")
		forceDegraded := fs.Bool("force-degraded", false, "force degraded-copy mode for tests")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		_, err := experiment.RunToolWorkspaceDemo(workspace.Config{Root: *root, ForceDegraded: *forceDegraded}, *out, *requireReal)
		return err
	default:
		return fmt.Errorf("unknown demo %q", args[0])
	}
}

func runHog(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl _hog pressure|sleep")
	}
	switch args[0] {
	case "sleep":
		fs := flag.NewFlagSet("_hog sleep", flag.ContinueOnError)
		durationMS := fs.Int("duration-ms", 3000, "sleep duration")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		time.Sleep(time.Duration(*durationMS) * time.Millisecond)
		return nil
	case "pressure":
		fs := flag.NewFlagSet("_hog pressure", flag.ContinueOnError)
		memoryBytes := fs.Int64("memory-bytes", 64*1024*1024, "bounded memory allocation")
		pids := fs.Int("pids", 8, "bounded child process count")
		durationMS := fs.Int("duration-ms", 2000, "CPU and hold duration")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *memoryBytes < 0 {
			return fmt.Errorf("memory-bytes must be non-negative")
		}
		if *pids < 0 || *pids > 32 {
			return fmt.Errorf("pids must be between 0 and 32")
		}
		if *durationMS <= 0 || *durationMS > 10000 {
			return fmt.Errorf("duration-ms must be between 1 and 10000")
		}
		return runBoundedPressureHog(*memoryBytes, *pids, time.Duration(*durationMS)*time.Millisecond)
	default:
		return fmt.Errorf("unknown hog %q", args[0])
	}
}

func runBoundedPressureHog(memoryBytes int64, pids int, duration time.Duration) error {
	memory := make([]byte, memoryBytes)
	for i := 0; i < len(memory); i += 4096 {
		memory[i] = byte(i)
	}
	children := make([]*exec.Cmd, 0, pids)
	sleepSeconds := strconv.FormatFloat(duration.Seconds()+1, 'f', 3, 64)
	for i := 0; i < pids; i++ {
		child := exec.Command("sleep", sleepSeconds)
		if err := child.Start(); err != nil {
			for _, started := range children {
				_ = started.Process.Kill()
				_ = started.Wait()
			}
			return err
		}
		children = append(children, child)
	}
	done := make(chan struct{})
	workers := runtime.NumCPU() * 2
	if workers < 2 {
		workers = 2
	}
	for i := 0; i < workers; i++ {
		go func(seed int) {
			value := uint64(seed + 1)
			for {
				select {
				case <-done:
					return
				default:
					value = value*1664525 + 1013904223
				}
			}
		}(i)
	}
	time.Sleep(duration)
	close(done)
	for _, child := range children {
		_ = child.Process.Kill()
		_ = child.Wait()
	}
	runtime.KeepAlive(memory)
	return nil
}
