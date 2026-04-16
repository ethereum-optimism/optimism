package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/coverage"
)

func cmdCoverage(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: checks coverage <subcommand>\n\nSubcommands:\n  collect   Collect coverage for a single test\n  plan      Print the full (test × profile) collection plan\n  batch     Run the full collection plan, skipping reports that already exist\n  ingest    Ingest coverage reports into graph")
	}

	switch args[0] {
	case "collect":
		return cmdCoverageCollect(args[1:])
	case "plan":
		return cmdCoveragePlan(args[1:])
	case "batch":
		return cmdCoverageBatch(args[1:])
	case "ingest":
		return cmdCoverageIngest(args[1:])
	default:
		return fmt.Errorf("unknown coverage subcommand: %s", args[0])
	}
}

func cmdCoverageCollect(args []string) error {
	fs := flag.NewFlagSet("coverage collect", flag.ExitOnError)
	lang := fs.String("lang", "", "language (solidity, go, rust)")
	test := fs.String("test", "", "test path (file, package, or crate)")
	root := fs.String("root", ".", "repository root")
	output := fs.String("output", "", "output path for coverage report (default: stdout)")
	profileName := fs.String("profile", "", "profile name from catalog (applies env vars during collection)")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to catalog (needed if --profile is set)")
	fs.Parse(args)

	if *lang == "" || *test == "" {
		return fmt.Errorf("--lang and --test are required")
	}

	var collector coverage.Collector
	switch *lang {
	case "solidity":
		collector = coverage.NewSolidityCollector()
	case "go":
		collector = coverage.NewGoCollector()
	case "rust":
		collector = coverage.NewRustCollector()
	default:
		return fmt.Errorf("unknown language: %s (valid: solidity, go, rust)", *lang)
	}

	profile := coverage.Profile{}
	if *profileName != "" {
		cat, err := catalog.Load(*catalogPath)
		if err != nil {
			return fmt.Errorf("loading catalog for profile lookup: %w", err)
		}
		p := cat.ProfileByName(*profileName)
		if p == nil {
			return fmt.Errorf("profile %q not found in catalog", *profileName)
		}
		profile = coverage.Profile{Name: p.Name, Env: p.Env}
	}

	report, err := collector.Collect(*root, *test, profile)
	if err != nil {
		return fmt.Errorf("collecting coverage: %w", err)
	}

	if *output != "" {
		if err := coverage.SaveReport(report, *output); err != nil {
			return fmt.Errorf("saving report: %w", err)
		}
		fmt.Printf("Coverage report written to %s\n", *output)
	} else {
		// Print to stdout
		if err := coverage.SaveReport(report, "/dev/stdout"); err != nil {
			return err
		}
	}

	return nil
}

func cmdCoveragePlan(args []string) error {
	fs := flag.NewFlagSet("coverage plan", flag.ExitOnError)
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to catalog")
	contractsDir := fs.String("contracts-dir", "packages/contracts-bedrock", "contracts dir (relative to repo root)")
	root := fs.String("root", ".", "repository root")
	output := fs.String("output", "", "write plan to file (default: stdout)")
	fs.Parse(args)

	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	absContractsDir := *contractsDir
	if !filepath.IsAbs(absContractsDir) {
		absContractsDir = filepath.Join(absRoot, absContractsDir)
	}

	jobs, err := coverage.ComputeSolidityJobs(cat, absContractsDir)
	if err != nil {
		return fmt.Errorf("computing jobs: %w", err)
	}

	var w io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	for _, j := range jobs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", j.Language, j.Profile, j.Test)
	}

	byProfile := make(map[string]int)
	for _, j := range jobs {
		byProfile[j.Profile]++
	}
	fmt.Fprintf(os.Stderr, "Total jobs: %d\n", len(jobs))
	for _, p := range cat.Profiles {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", p.Name, byProfile[p.Name])
	}
	return nil
}

func cmdCoverageBatch(args []string) error {
	fs := flag.NewFlagSet("coverage batch", flag.ExitOnError)
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to catalog")
	contractsDir := fs.String("contracts-dir", "packages/contracts-bedrock", "contracts dir (relative to repo root)")
	root := fs.String("root", ".", "repository root")
	outputDir := fs.String("output-dir", "ops/checks/coverage-data", "output directory for reports")
	dryRun := fs.Bool("dry-run", false, "print plan summary without running")
	fs.Parse(args)

	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	absContractsDir := *contractsDir
	if !filepath.IsAbs(absContractsDir) {
		absContractsDir = filepath.Join(absRoot, absContractsDir)
	}
	absOutputDir := *outputDir
	if !filepath.IsAbs(absOutputDir) {
		absOutputDir = filepath.Join(absRoot, absOutputDir)
	}

	jobs, err := coverage.ComputeSolidityJobs(cat, absContractsDir)
	if err != nil {
		return fmt.Errorf("computing jobs: %w", err)
	}

	if *dryRun {
		byProfile := make(map[string]int)
		existing := 0
		for _, j := range jobs {
			byProfile[j.Profile]++
			if _, err := os.Stat(filepath.Join(absOutputDir, j.OutputName())); err == nil {
				existing++
			}
		}
		fmt.Printf("Plan: %d jobs (%d existing, %d to run)\n", len(jobs), existing, len(jobs)-existing)
		for _, p := range cat.Profiles {
			fmt.Printf("  %s: %d\n", p.Name, byProfile[p.Name])
		}
		return nil
	}

	res, err := coverage.RunBatch(absRoot, jobs, absOutputDir, cat)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Done ===\n")
	fmt.Printf("Total: %d  Completed: %d  Skipped: %d  Failed: %d\n",
		res.Total, res.Completed, res.Skipped, res.Failed)
	fmt.Printf("Elapsed: %s\n", res.Elapsed.Round(time.Second))
	return nil
}

func cmdCoverageIngest(args []string) error {
	fs := flag.NewFlagSet("coverage ingest", flag.ExitOnError)
	graphPath := fs.String("graph", "ops/checks/graph.json", "graph file to update")
	coverageDir := fs.String("dir", "ops/checks/coverage-data", "directory of coverage reports")
	fs.Parse(args)

	// Load graph
	g, err := loadGraph(*graphPath)
	if err != nil {
		return err
	}

	// Load reports
	reports, err := coverage.LoadReports(*coverageDir)
	if err != nil {
		return fmt.Errorf("loading reports: %w", err)
	}

	// Ingest into graph
	if err := coverage.IngestReports(g, reports); err != nil {
		return fmt.Errorf("ingesting reports: %w", err)
	}

	// Save updated graph
	if err := saveGraph(g, *graphPath); err != nil {
		return err
	}

	fmt.Printf("Ingested %d coverage reports into graph\n", len(reports))
	return nil
}
