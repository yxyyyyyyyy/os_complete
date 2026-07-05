package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		return fmt.Errorf("usage: aortctl experiment|demo ...")
	}
	switch args[0] {
	case "experiment":
		return runExperiment(args[1:])
	case "demo":
		return runDemo(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runExperiment(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aortctl experiment e1|e2")
	}
	switch args[0] {
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
	default:
		return fmt.Errorf("unknown experiment %q", args[0])
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
		result := experiment.RunE5EndToEndBenchmark(*runs)
		return experiment.WriteJSON(filepath.Join(*out, "software_real_demo", "result.json"), result)
	case "fault":
		if len(args) < 2 {
			return fmt.Errorf("usage: aortctl demo fault workspace-rmrf")
		}
		switch args[1] {
		case "workspace-rmrf":
			fs := flag.NewFlagSet("demo fault workspace-rmrf", flag.ContinueOnError)
			out := fs.String("out", filepath.Join("experiments", "results"), "output directory")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			result, err := workspace.RunRMFaultDemo(workspace.Config{})
			if err != nil {
				return err
			}
			return experiment.WriteJSON(filepath.Join(*out, "workspace_isolation_evidence.json"), result)
		default:
			return fmt.Errorf("unknown fault demo %q", args[1])
		}
	default:
		return fmt.Errorf("unknown demo %q", args[0])
	}
}
