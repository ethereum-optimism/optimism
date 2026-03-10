package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/cache"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/executor"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// execute reads a pipeline decision and scoping config, then runs all needed
// categories for a given group. Categories with satisfied dependencies run in
// parallel — the executor builds a DAG from depends_on and launches each
// category as soon as its deps complete.
func main() {
	group := flag.String("group", "", "Execution group to run (build, go, sol, rust, misc)")
	decisionPath := flag.String("decision", "/tmp/shadow-ci-workspace/decision.json", "Path to decision JSON")
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	resultsDir := flag.String("results-dir", "/tmp/shadow-ci-test-results", "Results output directory")
	dryRun := flag.Bool("dry-run", false, "Print what would run without executing")
	cacheDir := flag.String("cache-dir", "", "Build cache directory (empty disables caching)")
	flag.Parse()

	if *group == "" {
		fatal("--group is required (build, go, sol, rust, misc)")
	}

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	data, err := os.ReadFile(*decisionPath)
	if err != nil {
		fatal("reading decision: %v", err)
	}
	var decision model.PipelineDecision
	if err := json.Unmarshal(data, &decision); err != nil {
		fatal("parsing decision: %v", err)
	}

	os.MkdirAll(*resultsDir, 0o755)

	// Build the executor with real implementations.
	var cacheResolver executor.CacheResolver
	if *cacheDir != "" {
		resolver := cache.NewResolver(".", *cacheDir)
		cacheResolver = executor.NewCacheAdapter(resolver, ".")
	}

	exec := executor.New(
		executor.Config{
			Group:      *group,
			DryRun:     *dryRun,
			ResultsDir: *resultsDir,
		},
		&executor.ShellRunner{},
		cacheResolver,
		&cfg.Scoping,
		&decision,
	)

	results, err := exec.Execute()
	if err != nil {
		fatal("execute: %v", err)
	}

	// Write results JSON.
	resultsJSON, _ := json.MarshalIndent(results, "", "  ")
	resultPath := fmt.Sprintf("%s/%s-results.json", *resultsDir, *group)
	os.WriteFile(resultPath, resultsJSON, 0o644)
	fmt.Printf("\nWrote results to %s\n", resultPath)

	// Exit non-zero if any failures.
	for _, r := range results {
		if r.Status == "fail" {
			os.Exit(1)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
