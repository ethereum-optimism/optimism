package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/cihistory"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// eventsWindowStart returns the earliest merged_at across events,
// or zero time if none have a timestamp.
func eventsWindowStart(events []cihistory.Event) time.Time {
	var earliest time.Time
	for _, e := range events {
		if e.MergedAt.IsZero() {
			continue
		}
		if earliest.IsZero() || e.MergedAt.Before(earliest) {
			earliest = e.MergedAt
		}
	}
	return earliest
}

func eventsWindowEnd(events []cihistory.Event) time.Time {
	var latest time.Time
	for _, e := range events {
		if e.MergedAt.IsZero() {
			continue
		}
		if e.MergedAt.After(latest) {
			latest = e.MergedAt
		}
	}
	return latest
}

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
	source := fs.String("source", "file", "event source: 'file' (local JSON) or 'circleci'")
	eventsPath := fs.String("events", "", "path to events JSON (for --source=file)")
	dumpEvents := fs.String("dump-events", "", "if set, write fetched events to this JSON path and exit (skip analysis + ingest). Intended for `checks replay`.")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file to update")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	learnedPath := fs.String("out-learned", "ops/checks/policy/learned.yaml", "path to write learned policy overrides")
	root := fs.String("root", ".", "repository root (for stamping source_sha on correlation edges and resolving CircleCI file lists)")
	windowDays := fs.Int("window-days", 0, "only consider events merged within the last N days (0 = all)")
	minObs := fs.Int("min-observations", 3, "minimum per-(file,check) observations to emit a correlation edge")
	minPrec := fs.Float64("min-precision", 0.1, "minimum precision to emit a correlation edge")
	minPriorObs := fs.Int("min-prior-observations", 20, "minimum samples before learning a per-kind/per-check prior")

	// CircleCI-specific flags
	ccOrg := fs.String("circleci-org", "ethereum-optimism", "CircleCI VCS org (for --source=circleci)")
	ccRepo := fs.String("circleci-repo", "optimism", "CircleCI VCS repo (for --source=circleci)")
	ccBranch := fs.String("circleci-branch", "develop", "branch to scan (for --source=circleci). Pass '*' or empty to scan all branches — required for pre-merge failure data.")
	ccExcludeBranches := fs.String("circleci-exclude-branches", "develop,main,master,release/",
		"comma-separated branch prefixes to skip when --circleci-branch is '*' (default excludes post-merge trunks)")
	ccMaxPages := fs.Int("circleci-max-pages", 10, "pipeline pagination cap (for --source=circleci)")
	ccTimeout := fs.Duration("circleci-timeout", 60*time.Second, "HTTP timeout per request (for --source=circleci)")

	fs.Parse(args)

	resolvedCatalog := resolveFromRoot(*catalogPath)
	resolvedGraph := resolveFromRoot(*graphPath)
	resolvedLearned := resolveFromRoot(*learnedPath)
	resolvedRoot := *root
	if resolvedRoot == "." || resolvedRoot == "" {
		resolvedRoot = findRepoRoot()
	}

	cat, err := catalog.Load(resolvedCatalog)
	if err != nil {
		return fmt.Errorf("loading catalog %s: %w", resolvedCatalog, err)
	}

	var since time.Time
	if *windowDays > 0 {
		since = time.Now().UTC().Add(-time.Duration(*windowDays) * 24 * time.Hour)
	}

	branchFilter := *ccBranch
	if branchFilter == "*" {
		branchFilter = "" // empty = all branches in fetcher
	}
	var excludeBranches []string
	if *ccExcludeBranches != "" {
		for _, p := range strings.Split(*ccExcludeBranches, ",") {
			if p = strings.TrimSpace(p); p != "" {
				excludeBranches = append(excludeBranches, p)
			}
		}
	}

	fetcher, err := buildFetcher(*source, *eventsPath, cat, ciOpts{
		org: *ccOrg, repo: *ccRepo, branch: branchFilter, excludeBranches: excludeBranches,
		maxPages: *ccMaxPages, timeout: *ccTimeout,
		repoRoot: resolvedRoot,
	})
	if err != nil {
		return err
	}

	events, err := fetcher.Fetch(since)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	// Dump-only mode: write events to disk and stop. Used to produce
	// the input for `checks replay` without mutating the graph or
	// learned.yaml. The format is exactly what cihistory.LoadEvents
	// reads, so the round-trip is lossless.
	if *dumpEvents != "" {
		data, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling events: %w", err)
		}
		if err := os.WriteFile(*dumpEvents, data, 0o644); err != nil {
			return fmt.Errorf("writing events file: %w", err)
		}
		fmt.Printf("Wrote %d events to %s (window %s → %s)\n",
			len(events), *dumpEvents,
			eventsWindowStart(events).Format("2006-01-02"),
			eventsWindowEnd(events).Format("2006-01-02"))
		return nil
	}

	analysis := cihistory.Analyze(events, cat, cihistory.Options{
		MinObservations:         *minObs,
		MinPrecision:            *minPrec,
		MinObservationsForPrior: *minPriorObs,
	})

	fmt.Printf("Source: %s; events: %d; window: %s → %s\n",
		*source, len(events),
		analysis.WindowStart.Format("2006-01-02"),
		analysis.WindowEnd.Format("2006-01-02"))
	fmt.Printf("Correlations: %d (MinObs=%d, MinPrec=%.2f)\n", len(analysis.Correlations), *minObs, *minPrec)
	fmt.Printf("Learned priors: %d kinds, %d checks\n", len(analysis.PriorsByKind), len(analysis.PriorsByCheck))

	g, err := graph.Load(resolvedGraph)
	if err != nil {
		return missingGraphError(resolvedGraph, err)
	}
	absRoot, _ := filepath.Abs(resolvedRoot)
	added, err := cihistory.WriteEdges(g, analysis, absRoot)
	if err != nil {
		return fmt.Errorf("writing edges: %w", err)
	}
	if err := graph.Save(g, resolvedGraph); err != nil {
		return fmt.Errorf("saving graph: %w", err)
	}
	fmt.Printf("Added %d correlation edges to %s\n", added, resolvedGraph)

	if err := cihistory.WriteLearnedPolicy(resolvedLearned, analysis); err != nil {
		return fmt.Errorf("writing learned policy: %w", err)
	}
	fmt.Printf("Wrote learned priors to %s\n", resolvedLearned)

	return nil
}

type ciOpts struct {
	org, repo, branch, repoRoot string
	excludeBranches             []string
	maxPages                    int
	timeout                     time.Duration
}

func buildFetcher(source, eventsPath string, cat *catalog.Catalog, ci ciOpts) (cihistory.Fetcher, error) {
	switch source {
	case "file":
		if eventsPath == "" {
			return nil, fmt.Errorf("--source=file requires --events")
		}
		return cihistory.NewFileFetcher(eventsPath), nil

	case "circleci":
		jobMap := cihistory.JobMapFromCatalog(cat)
		if len(jobMap) == 0 {
			return nil, fmt.Errorf("no ci_job_names defined in catalog — nothing to ingest from CircleCI")
		}
		return cihistory.NewCircleCIFetcher(cihistory.CircleCIConfig{
			Org:                   ci.org,
			Repo:                  ci.repo,
			Branch:                ci.branch,
			ExcludeBranchPrefixes: ci.excludeBranches,
			Token:                 resolveCircleCIToken(),
			HTTPClient:            &http.Client{Timeout: ci.timeout},
			MaxPages:              ci.maxPages,
			RepoRoot:              ci.repoRoot,
		}, jobMap), nil

	default:
		return nil, fmt.Errorf("unknown --source: %s (valid: file, circleci)", source)
	}
}

// resolveCircleCIToken follows the CLAUDE.md CI-credentials convention:
// public CircleCI projects don't require a token, so an empty result
// is acceptable. Only $CITOKEN is consulted — we deliberately do not
// auto-source ~/.bash_aliases.
func resolveCircleCIToken() string {
	return os.Getenv("CITOKEN")
}
