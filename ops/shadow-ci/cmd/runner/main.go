package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	goAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/golang"
	rustAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/rust"
	solAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/sol"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func main() {
	language := flag.String("language", "", "Language adapter to use (go, sol, rust)")
	targets := flag.String("targets", "", "Comma-separated target IDs")
	config := flag.String("config", "default", "Configuration name")
	reason := flag.String("reason", "affected", "Selection reason")
	configDir := flag.String("config-dir", "ops/shadow-ci/config", "Config directory")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
	outputDir := flag.String("output-dir", "/tmp/shadow-ci-results", "Results output directory")
	flag.Parse()

	if *language == "" {
		fatal("--language is required")
	}

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	// Build the target list.
	targetList := make([]model.Target, 0)
	for _, id := range strings.Split(*targets, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			targetList = append(targetList, model.Target{
				ID:       id,
				Language: *language,
			})
		}
	}

	// Build the configuration.
	jobConfig := model.Configuration{Name: *config, Env: map[string]string{}}
	// Pull env from config if available.
	if cfg.Adapters.Sol != nil && *language == "sol" {
		for _, f := range cfg.Adapters.Sol.Features {
			if f.Name == *config {
				jobConfig.Env = f.Env
				break
			}
		}
	}

	job := model.PlannedJob{
		Name:            fmt.Sprintf("%s-%s", *language, *config),
		Language:        *language,
		Targets:         targetList,
		Configurations:  []model.Configuration{jobConfig},
		Resources:       model.Resources{Parallelism: 1, Runner: "large", Timeout: 20 * 60},
		SelectionReason: *reason,
	}

	// Build adapter registry.
	registry := buildRegistry(cfg)

	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, *pipelineID, 0, "", "")

	classifier := engine.NewClassifier()
	fingerprinter := engine.NewFingerprinter()
	executor := engine.NewTestExecutor(registry, classifier, fingerprinter, emitter)

	result, err := executor.Execute(job)
	if err != nil {
		fatal("executing: %v", err)
	}

	// Write results.
	os.MkdirAll(*outputDir, 0o755)
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatal("marshaling results: %v", err)
	}

	resultPath := fmt.Sprintf("%s/%s-%s-results.json", *outputDir, *language, *config)
	if err := os.WriteFile(resultPath, resultData, 0o644); err != nil {
		fatal("writing results: %v", err)
	}

	// Print summary.
	passed, failed, flakes, infra := 0, 0, 0, 0
	for _, r := range result.Results {
		switch r.Status {
		case model.StatusPass:
			passed++
		case model.StatusFail:
			failed++
		}
		switch r.Classification {
		case model.Flake:
			flakes++
		case model.Infrastructure:
			infra++
		}
	}

	fmt.Printf("Results: %d total, %d passed, %d failed, %d flakes, %d infra\n",
		len(result.Results), passed, failed, flakes, infra)
	fmt.Printf("Wrote results to %s\n", resultPath)

	// Exit with failure if there are real failures.
	for _, r := range result.Results {
		if r.Classification == model.RealFailure {
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
