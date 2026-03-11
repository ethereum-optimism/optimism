package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	goAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/golang"
	rustAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/rust"
	solAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/sol"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/cache"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
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
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
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

	// Build adapter runner for language-native test execution.
	registry := buildRegistry(cfg)
	var store events.Store
	if *eventsDir != "" {
		store = events.NewLocalStore(*eventsDir)
	}
	emitter := events.NewEmitter(store, *pipelineID, 0, "", "")
	adapterRunner := executor.NewAdapterRunner(registry, emitter)

	exec := executor.New(
		executor.Config{
			Group:      *group,
			DryRun:     *dryRun,
			ResultsDir: *resultsDir,
		},
		&executor.ShellRunner{},
		adapterRunner,
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

	// Write combined test results from adapter-backed categories.
	var allTestResults []model.TestResult
	for _, r := range results {
		allTestResults = append(allTestResults, r.TestResults...)
	}
	if len(allTestResults) > 0 {
		trJSON, _ := json.MarshalIndent(allTestResults, "", "  ")
		trPath := fmt.Sprintf("%s/%s-test-results.json", *resultsDir, *group)
		os.WriteFile(trPath, trJSON, 0o644)
		fmt.Printf("Wrote %d test results to %s\n", len(allTestResults), trPath)
	}

	// Exit non-zero if any failures.
	for _, r := range results {
		if r.Status == "fail" {
			os.Exit(1)
		}
	}
}

func buildRegistry(cfg *model.Config) *adapters.Registry {
	registry := adapters.NewRegistry()

	if cfg.Adapters.Go != nil && cfg.Adapters.Go.Enabled {
		g := goAdapter.NewGraph(cfg.Adapters.Go.Root, cfg.Adapters.Go.SpecialPaths)
		r := goAdapter.NewRunner(cfg.Adapters.Go.Root)
		registry.Register("go", adapters.Adapter{Graph: g, Runner: r})
	}

	if cfg.Adapters.Sol != nil && cfg.Adapters.Sol.Enabled {
		g := solAdapter.NewGraph(
			cfg.Adapters.Sol.Root,
			cfg.Adapters.Sol.SourceDirs,
			cfg.Adapters.Sol.RemappingsFile,
			cfg.Adapters.Sol.SpecialPaths,
			cfg.Adapters.Sol.Features,
		)
		r := solAdapter.NewRunner(cfg.Adapters.Sol.Root)
		registry.Register("sol", adapters.Adapter{Graph: g, Runner: r})
	}

	if cfg.Adapters.Rust != nil && cfg.Adapters.Rust.Enabled {
		g := rustAdapter.NewGraph(cfg.Adapters.Rust.Root, cfg.Adapters.Rust.SpecialPaths)
		r := rustAdapter.NewRunner(cfg.Adapters.Rust.Root)
		registry.Register("rust", adapters.Adapter{Graph: g, Runner: r})
	}

	return registry
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
