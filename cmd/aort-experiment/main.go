package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"aort-r/internal/experiment"
)

func main() {
	name := flag.String("name", "all", "experiment name: e1-scheduler, e2-fault, e3-context, e1-real-scheduler, e2-real-fault, e3-real-context, e4-real-ipc, e5-end-to-end, real-all, or all")
	runs := flag.Int("runs", 5, "number of repeated runs")
	outDir := flag.String("out", "experiments/results", "output directory")
	flag.Parse()

	if err := run(*name, *runs, *outDir); err != nil {
		log.Fatal(err)
	}
}

func run(name string, runs int, outDir string) error {
	switch name {
	case "all":
		if err := run("e1-scheduler", runs, outDir); err != nil {
			return err
		}
		if err := run("e2-fault", runs, outDir); err != nil {
			return err
		}
		if err := run("e3-context", runs, outDir); err != nil {
			return err
		}
		return run("real-all", runs, outDir)
	case "real-all":
		_, err := experiment.RunRealExperimentSuite(runs, outDir)
		return err
	case "e1-scheduler":
		results := experiment.RunLegacyE1Scheduler(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e1-scheduler.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e1-scheduler.csv"), experiment.E1CSV(results)); err != nil {
			return err
		}
	case "e2-fault":
		results := experiment.RunLegacyE2FaultIsolation(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e2-fault.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e2-fault.csv"), experiment.E2CSV(results)); err != nil {
			return err
		}
	case "e3-context":
		result := experiment.RunE3ContextSharing(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e3-context.json"), result); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e3-context.csv"), experiment.E3CSV(result)); err != nil {
			return err
		}
	case "e1-real-scheduler":
		results := experiment.RunE1RealSchedulerBenchmark(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e1-real-scheduler.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e1-real-scheduler.csv"), experiment.E1RealCSV(results)); err != nil {
			return err
		}
	case "e2-real-fault":
		results := experiment.RunE2RealFaultIsolation(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e2-real-fault.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e2-real-fault.csv"), experiment.E2RealCSV(results)); err != nil {
			return err
		}
	case "e3-real-context":
		results := experiment.RunE3RealContextReuse(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e3-real-context.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e3-real-context.csv"), experiment.E3RealCSV(results)); err != nil {
			return err
		}
	case "e4-real-ipc":
		results := experiment.RunE4RealIPCBenchmark(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e4-real-ipc.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e4-real-ipc.csv"), experiment.E4RealCSV(results)); err != nil {
			return err
		}
	case "e5-end-to-end":
		result := experiment.RunE5EndToEndBenchmark(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e5-end-to-end.json"), result); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e5-end-to-end.csv"), experiment.E5RealCSV(result)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown experiment %q", name)
	}
	return nil
}
