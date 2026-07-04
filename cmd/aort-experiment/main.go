package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"aort-r/internal/experiment"
)

func main() {
	name := flag.String("name", "all", "experiment name: e1-scheduler, e2-fault, e3-context, or all")
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
		return run("e3-context", runs, outDir)
	case "e1-scheduler":
		results := experiment.RunE1Scheduler(runs)
		if err := experiment.WriteJSON(filepath.Join(outDir, "e1-scheduler.json"), results); err != nil {
			return err
		}
		if err := experiment.WriteCSV(filepath.Join(outDir, "e1-scheduler.csv"), experiment.E1CSV(results)); err != nil {
			return err
		}
	case "e2-fault":
		results := experiment.RunE2FaultIsolation(runs)
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
	default:
		return fmt.Errorf("unknown experiment %q", name)
	}
	return nil
}
