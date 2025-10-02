// flake-shake-aggregator aggregates multiple flake-shake reports from parallel workers
// into a single comprehensive report.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	html_pkg "html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FlakeShakeResult represents a single test's flake-shake analysis
type FlakeShakeResult struct {
	TestName       string        `json:"test_name"`
	Package        string        `json:"package"`
	TotalRuns      int           `json:"total_runs"`
	Passes         int           `json:"passes"`
	Failures       int           `json:"failures"`
	Skipped        int           `json:"skipped"`
	PassRate       float64       `json:"pass_rate"`
	AvgDuration    time.Duration `json:"avg_duration"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	FailureLogs    []string      `json:"failure_logs,omitempty"`
	LastFailure    *time.Time    `json:"last_failure,omitempty"`
	Recommendation string        `json:"recommendation"`
}

// FlakeShakeReport contains the complete flake-shake analysis
type FlakeShakeReport struct {
	Date        string             `json:"date"`
	Gate        string             `json:"gate"`
	TotalRuns   int                `json:"total_runs"`
	Iterations  int                `json:"iterations"`
	Tests       []FlakeShakeResult `json:"tests"`
	GeneratedAt time.Time          `json:"generated_at"`
	RunID       string             `json:"run_id"`
}

// AggregatedTestStats for accumulating results
type AggregatedTestStats struct {
	TestName      string
	Package       string
	TotalRuns     int
	Passes        int
	Failures      int
	Skipped       int
	MinDuration   time.Duration
	MaxDuration   time.Duration
	FailureLogs   []string
	LastFailure   *time.Time
	durationSum   time.Duration
	durationCount int
}

func main() {
	var (
		inputPattern string
		outputDir    string
		verbose      bool
	)

	flag.StringVar(&inputPattern, "input-pattern", "flake-shake-results-worker-*/flake-shake-report.json",
		"Glob pattern to find worker report files")
	flag.StringVar(&outputDir, "output-dir", "final-report",
		"Directory to write the aggregated report")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()

	if err := run(inputPattern, outputDir, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(inputPattern, outputDir string, verbose bool) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all report files
	reportFiles, err := filepath.Glob(inputPattern)
	if err != nil {
		return fmt.Errorf("failed to glob input files: %w", err)
	}

	if len(reportFiles) == 0 {
		// Try alternative patterns
		alternatives := []string{
			"flake-shake-results-worker-*/flake-shake-report.json",
			"*/flake-shake-report.json",
			"flake-shake-report-*.json",
		}
		for _, alt := range alternatives {
			reportFiles, err = filepath.Glob(alt)
			if err == nil && len(reportFiles) > 0 {
				break
			}
		}

		if len(reportFiles) == 0 {
			return fmt.Errorf("no report files found matching pattern: %s", inputPattern)
		}
	}

	if verbose {
		fmt.Printf("Found %d report files to aggregate:\n", len(reportFiles))
		for _, f := range reportFiles {
			fmt.Printf("  - %s\n", f)
		}
	}

	// Aggregate all reports
	aggregated := make(map[string]*AggregatedTestStats)
	var gate string
	var runID string
	totalIterations := 0

	for _, reportFile := range reportFiles {
		if verbose {
			fmt.Printf("Processing %s...\n", reportFile)
		}

		data, err := os.ReadFile(reportFile)
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", reportFile, err)
			continue
		}

		var report FlakeShakeReport
		if err := json.Unmarshal(data, &report); err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", reportFile, err)
			continue
		}

		// Use first report's metadata
		if gate == "" {
			gate = report.Gate
		}
		if runID == "" && report.RunID != "" {
			runID = report.RunID
		}
		totalIterations += report.Iterations

		// Aggregate test results
		for _, test := range report.Tests {
			key := fmt.Sprintf("%s::%s", test.Package, test.TestName)

			if stats, exists := aggregated[key]; exists {
				// Merge with existing stats
				stats.TotalRuns += test.TotalRuns
				stats.Passes += test.Passes
				stats.Failures += test.Failures
				stats.Skipped += test.Skipped

				// Update durations
				if test.MinDuration < stats.MinDuration || stats.MinDuration == 0 {
					stats.MinDuration = test.MinDuration
				}
				if test.MaxDuration > stats.MaxDuration {
					stats.MaxDuration = test.MaxDuration
				}
				stats.durationSum += time.Duration(test.AvgDuration) * time.Duration(test.TotalRuns)
				stats.durationCount += test.TotalRuns

				// Merge failure logs (keep first 10)
				stats.FailureLogs = append(stats.FailureLogs, test.FailureLogs...)
				if len(stats.FailureLogs) > 10 {
					stats.FailureLogs = stats.FailureLogs[:10]
				}

				// Update last failure time
				if test.LastFailure != nil && (stats.LastFailure == nil || test.LastFailure.After(*stats.LastFailure)) {
					stats.LastFailure = test.LastFailure
				}
			} else {
				// First occurrence of this test
				aggregated[key] = &AggregatedTestStats{
					TestName:      test.TestName,
					Package:       test.Package,
					TotalRuns:     test.TotalRuns,
					Passes:        test.Passes,
					Failures:      test.Failures,
					Skipped:       test.Skipped,
					MinDuration:   test.MinDuration,
					MaxDuration:   test.MaxDuration,
					durationSum:   time.Duration(test.AvgDuration) * time.Duration(test.TotalRuns),
					durationCount: test.TotalRuns,
					FailureLogs:   test.FailureLogs,
					LastFailure:   test.LastFailure,
				}
			}
		}
	}

	// Calculate final statistics
	var finalTests []FlakeShakeResult
	totalTestRuns := 0
	for _, stats := range aggregated {
		// Calculate pass rate
		passRate := 0.0
		if stats.TotalRuns > 0 {
			passRate = float64(stats.Passes) / float64(stats.TotalRuns) * 100
		}

		// Calculate average duration
		avgDuration := time.Duration(0)
		if stats.durationCount > 0 {
			avgDuration = stats.durationSum / time.Duration(stats.durationCount)
		}

		// Determine recommendation
		recommendation := "UNSTABLE"
		if passRate == 100 {
			recommendation = "STABLE"
		}

		// Convert to final format
		totalTestRuns += stats.TotalRuns
		finalTests = append(finalTests, FlakeShakeResult{
			TestName:       stats.TestName,
			Package:        stats.Package,
			TotalRuns:      stats.TotalRuns,
			Passes:         stats.Passes,
			Failures:       stats.Failures,
			Skipped:        stats.Skipped,
			PassRate:       passRate,
			AvgDuration:    avgDuration,
			MinDuration:    stats.MinDuration,
			MaxDuration:    stats.MaxDuration,
			FailureLogs:    stats.FailureLogs,
			LastFailure:    stats.LastFailure,
			Recommendation: recommendation,
		})
	}

	// Create final aggregated report
	finalReport := FlakeShakeReport{
		Date:        time.Now().Format("2006-01-02"),
		Gate:        gate,
		TotalRuns:   totalTestRuns,
		Iterations:  totalIterations,
		Tests:       finalTests,
		GeneratedAt: time.Now(),
		RunID:       runID,
	}

	// Save JSON report
	jsonFile := filepath.Join(outputDir, "flake-shake-report.json")
	jsonData, err := json.MarshalIndent(finalReport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	// Generate and save HTML report
	htmlFile := filepath.Join(outputDir, "flake-shake-report.html")
	htmlContent := generateHTMLReport(&finalReport)
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	fmt.Printf("✅ Aggregation complete!\n")
	fmt.Printf("   - Processed %d worker reports\n", len(reportFiles))
	fmt.Printf("   - Aggregated %d unique tests\n", len(finalTests))
	fmt.Printf("   - Total iterations: %d\n", totalIterations)
	fmt.Printf("   - Reports saved to:\n")
	fmt.Printf("     • %s\n", jsonFile)
	fmt.Printf("     • %s\n", htmlFile)

	// Print summary statistics
	stableCount := 0
	unstableCount := 0
	for _, test := range finalTests {
		if test.Recommendation == "STABLE" {
			stableCount++
		} else {
			unstableCount++
		}
	}

	fmt.Printf("\n📊 Test Stability Summary:\n")
	if len(finalTests) > 0 {
		fmt.Printf("   - STABLE: %d tests (%.1f%%)\n", stableCount,
			float64(stableCount)/float64(len(finalTests))*100)
		fmt.Printf("   - UNSTABLE: %d tests (%.1f%%)\n", unstableCount,
			float64(unstableCount)/float64(len(finalTests))*100)
	} else {
		fmt.Printf("   - No tests found\n")
	}

	// List unstable tests if any
	if unstableCount > 0 && verbose {
		fmt.Printf("\n⚠️  Unstable tests:\n")
		for _, test := range finalTests {
			if test.Recommendation == "UNSTABLE" {
				fmt.Printf("   - %s (%.1f%% pass rate)\n",
					strings.TrimPrefix(test.TestName, test.Package+"::"),
					test.PassRate)
			}
		}
	}

	return nil
}

func generateHTMLReport(report *FlakeShakeReport) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>Flake-Shake Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #007acc; padding-bottom: 10px; }
        .summary { display: flex; gap: 20px; margin: 20px 0; }
        .summary-card { flex: 1; background: #f8f9fa; padding: 15px; border-radius: 4px; }
        .summary-card h3 { margin-top: 0; color: #666; font-size: 14px; text-transform: uppercase; }
        .summary-card .value { font-size: 24px; font-weight: bold; color: #333; }
        .stable { color: #28a745; }
        .unstable { color: #dc3545; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th { background: #007acc; color: white; text-align: left; padding: 10px; }
        td { padding: 8px; border-bottom: 1px solid #ddd; }
        tr:hover { background: #f5f5f5; }
        .pass-rate-100 { background: #d4edda; }
        .pass-rate-low { background: #f8d7da; }
        .recommendation { padding: 2px 6px; border-radius: 3px; font-size: 12px; font-weight: bold; }
        .recommendation.stable { background: #28a745; color: white; }
        .recommendation.unstable { background: #dc3545; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Flake-Shake Report - ` + html_pkg.EscapeString(report.Gate) + `</h1>
        <p>Generated: ` + report.GeneratedAt.Format("2006-01-02 15:04:05") + `</p>

        <div class="summary">
            <div class="summary-card">
                <h3>Total Tests</h3>
                <div class="value">` + fmt.Sprintf("%d", len(report.Tests)) + `</div>
            </div>
            <div class="summary-card">
                <h3>Iterations</h3>
                <div class="value">` + fmt.Sprintf("%d", report.Iterations) + `</div>
            </div>
            <div class="summary-card">
                <h3>Stable Tests</h3>
                <div class="value stable">`)

	stableCount := 0
	for _, test := range report.Tests {
		if test.Recommendation == "STABLE" {
			stableCount++
		}
	}
	html.WriteString(fmt.Sprintf("%d", stableCount))

	html.WriteString(`</div>
            </div>
            <div class="summary-card">
                <h3>Unstable Tests</h3>
                <div class="value unstable">`)

	html.WriteString(fmt.Sprintf("%d", len(report.Tests)-stableCount))

	html.WriteString(`</div>
            </div>
    </div>

        <h2>Stable Tests</h2>`)

	if stableCount > 0 {
		html.WriteString(`
        <ul>`)
		for _, test := range report.Tests {
			if test.Recommendation == "STABLE" {
				html.WriteString(fmt.Sprintf(`
            <li>%s <small>(%s)</small></li>`,
					html_pkg.EscapeString(test.TestName),
					html_pkg.EscapeString(test.Package),
				))
			}
		}
		html.WriteString(`
        </ul>`)
	} else {
		html.WriteString(`
        <p>No stable tests in this run.</p>`)
	}

	html.WriteString(`
        <table>
            <thead>
                <tr>
                    <th>Test Name</th>
                    <th>Package</th>
                    <th>Pass Rate</th>
                    <th>Runs</th>
                    <th>Passed</th>
                    <th>Failed</th>
                    <th>Avg Duration</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>`)

	for _, test := range report.Tests {
		rowClass := ""
		if test.PassRate == 100 {
			rowClass = "pass-rate-100"
		} else if test.PassRate < 95 {
			rowClass = "pass-rate-low"
		}

		html.WriteString(fmt.Sprintf(`
                <tr class="%s">
                    <td>%s</td>
                    <td>%s</td>
                    <td>%.1f%%</td>
                    <td>%d</td>
                    <td>%d</td>
                    <td>%d</td>
                    <td>%s</td>
                    <td><span class="recommendation %s">%s</span></td>
                </tr>`,
			rowClass,
			html_pkg.EscapeString(test.TestName),
			html_pkg.EscapeString(test.Package),
			test.PassRate,
			test.TotalRuns,
			test.Passes,
			test.Failures,
			test.AvgDuration.Round(time.Millisecond),
			strings.ToLower(test.Recommendation),
			test.Recommendation,
		))
	}

	html.WriteString(`
            </tbody>
        </table>
    </div>
</body>
</html>`)

	return html.String()
}
