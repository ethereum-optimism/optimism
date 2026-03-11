package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func main() {
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci/events", "Events directory")
	lookbackDays := flag.Int("lookback-days", 30, "How many days of history to analyze")
	dryRun := flag.Bool("dry-run", false, "Print recommendations without applying")
	output := flag.String("output", "", "Output updated placement.yaml path (default: overwrite config)")
	flag.Parse()

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	store := events.NewLocalStore(*eventsDir)

	// Compute stats and correlations from the event store.
	start := time.Now().Add(-time.Duration(*lookbackDays) * 24 * time.Hour)
	end := time.Now()

	aggregator := engine.NewStatsAggregator(store)
	testStats, err := aggregator.ComputeTestStats(start, end)
	if err != nil {
		fatal("computing test stats: %v", err)
	}
	fmt.Printf("Computed stats for %d tests\n", len(testStats))

	categoryStats, err := aggregator.ComputeCategoryStats(testStats, cfg.Scoping)
	if err != nil {
		fatal("computing category stats: %v", err)
	}
	fmt.Printf("Computed stats for %d categories\n", len(categoryStats))

	corrEngine := engine.NewCorrelationEngine(store)
	corrMatrix, err := corrEngine.Compute(start, end, engine.DefaultCorrelationConfig())
	if err != nil {
		fatal("computing correlations: %v", err)
	}
	fmt.Printf("Computed %d correlations from %d pipelines\n", len(corrMatrix.Correlations), corrMatrix.PipelinesAnalyzed)

	config := engine.DefaultPlacementOptimizerConfig()
	optimizer := engine.NewPlacementOptimizer(store, config)
	recs := optimizer.Optimize(cfg.Placement, corrMatrix, categoryStats)

	fmt.Printf("Placement optimizer: %d recommendations\n", len(recs))
	for _, rec := range recs {
		fmt.Printf("  %s: %s -> %s (%s)\n", rec.Category, rec.CurrentStage, rec.ProposedStage, rec.Reason)
	}

	if *dryRun {
		data, _ := json.MarshalIndent(recs, "", "  ")
		fmt.Println(string(data))
		return
	}

	emitter := events.NewEmitter(store, "optimizer", 0, "", "")
	optimizer.ApplyRecommendations(&cfg.Placement, recs, emitter)

	outputPath := *output
	if outputPath == "" {
		outputPath = *configDir + "/placement.yaml"
	}

	// Write as YAML.
	yamlData, err := marshalPlacementYAML(cfg.Placement)
	if err != nil {
		fatal("marshaling placement: %v", err)
	}
	if err := os.WriteFile(outputPath, yamlData, 0o644); err != nil {
		fatal("writing placement: %v", err)
	}
	fmt.Printf("Updated placement written to %s\n", outputPath)
}

func marshalPlacementYAML(p model.PlacementConfig) ([]byte, error) {
	return yaml.Marshal(p)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
