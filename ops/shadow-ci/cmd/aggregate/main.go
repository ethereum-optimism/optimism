package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/aggregator"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func main() {
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	outputDir := flag.String("output-dir", "/tmp/shadow-ci-reports", "Reports output directory")
	mode := flag.String("mode", "weekly", "Report mode: weekly, dashboard")
	lookbackDays := flag.Int("lookback", 30, "Dashboard lookback days")
	flag.Parse()

	store := events.NewLocalStore(*eventsDir)
	agg := aggregator.NewAggregator(store)

	os.MkdirAll(*outputDir, 0o755)

	switch *mode {
	case "weekly":
		now := time.Now().UTC()
		weekStart := now.AddDate(0, 0, -int(now.Weekday()))
		weekEnd := weekStart.AddDate(0, 0, 7)

		report, err := agg.GenerateWeeklyReport(weekStart, weekEnd)
		if err != nil {
			fatal("generating weekly report: %v", err)
		}

		data, _ := json.MarshalIndent(report, "", "  ")
		path := fmt.Sprintf("%s/weekly-%s.json", *outputDir, report.Week)
		os.WriteFile(path, data, 0o644)

		// Emit the report as an event for agents.
		emitter := events.NewEmitter(store, "", 0, "", "")
		emitter.Emit(model.EventWeeklyReport, report)

		printWeeklyReport(report)
		fmt.Printf("\nWrote weekly report to %s\n", path)

	case "dashboard":
		dash, err := agg.GenerateDashboard(*lookbackDays)
		if err != nil {
			fatal("generating dashboard: %v", err)
		}

		data, _ := json.MarshalIndent(dash, "", "  ")
		path := fmt.Sprintf("%s/dashboard.json", *outputDir)
		os.WriteFile(path, data, 0o644)

		fmt.Printf("Dashboard: %d pipelines, catch_rate=%.1f%%, speedup=%.1fx, active_flakes=%d\n",
			dash.TotalPipelines, dash.OverallCatchRate*100, dash.OverallSpeedup, dash.ActiveFlakes)
		fmt.Printf("Wrote dashboard to %s\n", path)

	default:
		fatal("unknown mode: %s", *mode)
	}
}

func printWeeklyReport(report *aggregator.WeeklyReport) {
	fmt.Printf("\n=== Shadow CI Weekly Report: %s ===\n", report.Week)
	fmt.Printf("Pipelines:        %d\n", report.TotalPipelines)
	fmt.Printf("Catch Rate:       %.1f%%\n", report.CatchRate*100)
	fmt.Printf("False Negatives:  %d\n", report.FalseNegatives)
	fmt.Printf("Median Speedup:   %.1fx\n", report.MedianSpeedup)
	fmt.Printf("Flakes Detected:  %d (%d unique patterns)\n", report.FlakesDetected, report.UniqueFlakePatterns)
	fmt.Printf("Graph Gaps:       %d\n", report.GraphGaps)

	if len(report.TopFlakes) > 0 {
		fmt.Println("\nTop Flakes:")
		for i, f := range report.TopFlakes {
			fmt.Printf("  %d. %s (count=%d, test=%s)\n", i+1, f.Fingerprint, f.Count, f.Test)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
