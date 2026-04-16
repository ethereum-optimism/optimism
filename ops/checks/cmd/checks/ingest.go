package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/cihistory"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func cmdIngest(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: checks ingest <subcommand>\n\nSubcommands:\n  ci-history   Fetch CI event history, write observed-correlation edges and learned priors")
	}
	switch args[0] {
	case "ci-history":
		return cmdIngestCIHistory(args[1:])
	default:
		return fmt.Errorf("unknown ingest subcommand: %s", args[0])
	}
}

func cmdIngestCIHistory(args []string) error {
	fs := flag.NewFlagSet("ingest ci-history", flag.ExitOnError)
	eventsPath := fs.String("events", "", "path to events JSON (FileFetcher source) — required")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file to update")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	learnedPath := fs.String("out-learned", "ops/checks/policy/learned.yaml", "path to write learned policy overrides")
	root := fs.String("root", ".", "repository root (for stamping source_sha on correlation edges)")
	windowDays := fs.Int("window-days", 0, "only consider events merged within the last N days (0 = all)")
	minObs := fs.Int("min-observations", 3, "minimum per-(file,check) observations to emit a correlation edge")
	minPrec := fs.Float64("min-precision", 0.1, "minimum precision to emit a correlation edge")
	minPriorObs := fs.Int("min-prior-observations", 20, "minimum samples before learning a per-kind/per-check prior")
	fs.Parse(args)

	if *eventsPath == "" {
		return fmt.Errorf("--events is required")
	}

	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	fetcher := cihistory.NewFileFetcher(*eventsPath)
	var since time.Time
	if *windowDays > 0 {
		since = time.Now().UTC().Add(-time.Duration(*windowDays) * 24 * time.Hour)
	}
	events, err := fetcher.Fetch(since)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	analysis := cihistory.Analyze(events, cat, cihistory.Options{
		MinObservations:         *minObs,
		MinPrecision:            *minPrec,
		MinObservationsForPrior: *minPriorObs,
	})

	fmt.Printf("Events: %d; window: %s → %s\n",
		len(events),
		analysis.WindowStart.Format("2006-01-02"),
		analysis.WindowEnd.Format("2006-01-02"))
	fmt.Printf("Correlations: %d (MinObs=%d, MinPrec=%.2f)\n", len(analysis.Correlations), *minObs, *minPrec)
	fmt.Printf("Learned priors: %d kinds, %d checks\n", len(analysis.PriorsByKind), len(analysis.PriorsByCheck))

	// Load and update the graph with correlation edges.
	g, err := graph.Load(*graphPath)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}
	absRoot, _ := filepath.Abs(*root)
	added, err := cihistory.WriteEdges(g, analysis, absRoot)
	if err != nil {
		return fmt.Errorf("writing edges: %w", err)
	}
	if err := graph.Save(g, *graphPath); err != nil {
		return fmt.Errorf("saving graph: %w", err)
	}
	fmt.Printf("Added %d correlation edges to %s\n", added, *graphPath)

	// Write learned policy.
	if err := cihistory.WriteLearnedPolicy(*learnedPath, analysis); err != nil {
		return fmt.Errorf("writing learned policy: %w", err)
	}
	fmt.Printf("Wrote learned priors to %s\n", *learnedPath)

	return nil
}
