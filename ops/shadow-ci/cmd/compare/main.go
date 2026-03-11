package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform/circleci"
)

func main() {
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
	planID := flag.String("plan-id", "", "Plan ID")
	resultsDir := flag.String("results-dir", "/tmp/shadow-ci-results", "Shadow CI results directory")
	outputDir := flag.String("output-dir", "/tmp/shadow-ci-comparison", "Comparison output directory")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	configDir := flag.String("config", "ops/shadow-ci/config", "Config directory")
	flag.Parse()

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	// Load shadow CI results.
	shadowResults, err := loadShadowResults(*resultsDir)
	if err != nil {
		fatal("loading shadow results: %v", err)
	}
	fmt.Printf("Shadow results: %d tests\n", len(shadowResults))

	// Fetch main CI results.
	adapter := circleci.NewAdapter(cfg.Platform.CircleCI)
	mainResults, err := adapter.FetchResults(*pipelineID)
	if err != nil {
		fmt.Printf("Warning: could not fetch main CI results: %v\n", err)
		fmt.Println("Comparison will be shadow-only (no catch rate computation)")
		mainResults = nil
	}
	fmt.Printf("Main CI results: %d tests\n", len(mainResults))

	// Compare.
	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, *pipelineID, 0, "", "")

	compEngine := engine.NewComparisonEngine(emitter)
	comparison := compEngine.Compare(shadowResults, mainResults)

	// Write comparison.
	os.MkdirAll(*outputDir, 0o755)
	data, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		fatal("marshaling comparison: %v", err)
	}

	outputPath := filepath.Join(*outputDir, "comparison.json")
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fatal("writing comparison: %v", err)
	}

	// Print report.
	fmt.Println("\n=== Shadow CI Comparison Report ===")
	fmt.Printf("Plan: %s\n", *planID)
	fmt.Printf("Pipeline: %s\n", *pipelineID)
	fmt.Println()
	fmt.Printf("Catch Rate:        %.1f%% (%d/%d)\n", comparison.CatchRate*100, comparison.ShadowCICaught, comparison.MainCIRealFailures)
	fmt.Printf("False Negatives:   %d\n", comparison.FalseNegatives)
	fmt.Printf("Speedup:           %.1fx\n", comparison.Speedup)
	fmt.Printf("Compute Reduction: %.1f%%\n", comparison.ComputeReduction*100)
	fmt.Printf("Flakes Detected:   %d (unique fingerprints: %d)\n", comparison.ShadowCIFlakes, len(comparison.UniqueFingerprints))
	fmt.Println()

	if len(comparison.FalseNegativeDetails) > 0 {
		fmt.Println("FALSE NEGATIVES (CRITICAL):")
		for _, fn := range comparison.FalseNegativeDetails {
			fmt.Printf("  - %s/%s (%s): %s\n", fn.Test.Package, fn.Test.Name, fn.Language, fn.MissedBecause)
		}
	} else {
		fmt.Println("No false negatives detected.")
	}

	fmt.Printf("\nWrote comparison to %s\n", outputPath)

	// Exit with failure if there are false negatives.
	if comparison.FalseNegatives > 0 {
		os.Exit(1)
	}
}

func loadShadowResults(dir string) ([]model.TestResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var allResults []model.TestResult
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		// Try engine.JobResult format first (legacy).
		var jobResult engine.JobResult
		if err := json.Unmarshal(data, &jobResult); err == nil && len(jobResult.Results) > 0 {
			allResults = append(allResults, jobResult.Results...)
			continue
		}

		// Try []model.TestResult format (new per-test JSON from adapter runner).
		var testResults []model.TestResult
		if err := json.Unmarshal(data, &testResults); err == nil && len(testResults) > 0 {
			allResults = append(allResults, testResults...)
			continue
		}
		// Skip unrecognized files (group-results.json, etc.)
	}

	return allResults, nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
