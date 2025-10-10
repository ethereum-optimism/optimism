// flake-shake-aggregator aggregates multiple flake-shake reports from parallel workers
// into a single comprehensive report.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	html_pkg "html"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	logger := log.New(os.Stdout, "[flake-shake-aggregator] ", log.LstdFlags)
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
		logger.Printf("Found %d report files to aggregate:", len(reportFiles))
		for _, f := range reportFiles {
			logger.Printf("  - %s", f)
		}
	}

	// Aggregate all reports
	aggregated := make(map[string]*AggregatedTestStats)
	var gate string
	var runID string
	totalIterations := 0

	for _, reportFile := range reportFiles {
		if verbose {
			logger.Printf("Processing %s...", reportFile)
		}

		data, err := os.ReadFile(reportFile)
		if err != nil {
			logger.Printf("Warning: failed to read %s: %v", reportFile, err)
			continue
		}

		var report FlakeShakeReport
		if err := json.Unmarshal(data, &report); err != nil {
			logger.Printf("Warning: failed to parse %s: %v", reportFile, err)
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

				// Merge failure logs (keep first 50)
				stats.FailureLogs = append(stats.FailureLogs, test.FailureLogs...)
				if len(stats.FailureLogs) > 50 {
					stats.FailureLogs = stats.FailureLogs[:50]
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

	logger.Printf("✅ Aggregation complete!")
	logger.Printf("   - Processed %d worker reports", len(reportFiles))
	logger.Printf("   - Aggregated %d unique tests", len(finalTests))
	logger.Printf("   - Total iterations: %d", totalIterations)
	logger.Printf("   - Reports saved to:")
	logger.Printf("     • %s", jsonFile)
	logger.Printf("     • %s", htmlFile)

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

	logger.Printf("\n📊 Test Stability Summary:")
	if len(finalTests) > 0 {
		logger.Printf("   - STABLE: %d tests (%.1f%%)", stableCount,
			float64(stableCount)/float64(len(finalTests))*100)
		logger.Printf("   - UNSTABLE: %d tests (%.1f%%)", unstableCount,
			float64(unstableCount)/float64(len(finalTests))*100)
	} else {
		logger.Printf("   - No tests found")
	}

	// List unstable tests if any
	if unstableCount > 0 && verbose {
		logger.Printf("\n⚠️  Unstable tests:")
		for _, test := range finalTests {
			if test.Recommendation == "UNSTABLE" {
				logger.Printf("   - %s (%.1f%% pass rate)",
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
`)

	// Append grouped failure details
	html.WriteString(`
        <h2>Failure Details</h2>
`)

	normalizer := regexp.MustCompile(`(?m)^\s*\[?\d{4}-\d{2}-\d{2}.*$|\bt=\d{4}-\d{2}-\d{2}.*$|\b(duration|elapsed|took)[:=].*$`)
	ansiStrip := regexp.MustCompile("\x1b\\[[0-9;]*m")
	classify := func(s string) string {
		ls := strings.ToLower(s)
		switch {
		case strings.Contains(ls, "context deadline exceeded"):
			return "context deadline"
		case strings.Contains(ls, "deadline exceeded"):
			return "deadline exceeded"
		case strings.Contains(ls, "timeout"):
			return "timeout"
		case strings.Contains(ls, "connection refused"):
			return "connection refused"
		case strings.Contains(ls, "connection reset"):
			return "connection reset"
		case strings.Contains(ls, "rpc error") || strings.Contains(ls, "rpc call failed"):
			return "rpc error"
		case strings.Contains(ls, "assert") || strings.Contains(ls, "require"):
			return "assertion"
		default:
			return "unknown"
		}
	}
	for _, test := range report.Tests {
		if len(test.FailureLogs) == 0 {
			continue
		}
		html.WriteString(fmt.Sprintf(`<details><summary>%s — %s (failures: %d)</summary>`,
			html_pkg.EscapeString(test.TestName), html_pkg.EscapeString(test.Package), test.Failures))

		groups := map[string]struct {
			Count  int
			Sample string
			Type   string
		}{}
		typeSummary := map[string]int{}
		for _, raw := range test.FailureLogs {
			// Extract human-readable content from Go test JSON events by keeping only non-empty Output fields.
			processed := strings.TrimSpace(raw)
			if strings.HasPrefix(processed, "{") {
				var ev struct {
					Output string `json:"Output"`
				}
				if json.Unmarshal([]byte(processed), &ev) == nil {
					processed = strings.TrimSpace(ev.Output)
				}
			}
			if processed == "" {
				continue
			}
			// Strip ANSI color codes for readability
			processed = ansiStrip.ReplaceAllString(processed, "")
			// Normalize noisy timestamps/durations and trim
			norm := normalizer.ReplaceAllString(processed, "")
			norm = strings.TrimSpace(norm)
			if norm == "" {
				continue
			}
			sum := sha256.Sum256([]byte(norm))
			key := fmt.Sprintf("%x", sum[:])
			g := groups[key]
			if g.Count == 0 {
				g.Sample = norm
				g.Type = classify(norm)
			}
			g.Count++
			groups[key] = g
		}
		// Build type summary
		for _, g := range groups {
			typeSummary[g.Type] += g.Count
		}
		// Render summary
		html.WriteString(`<div class="failure-group">`)
		html.WriteString(`<strong>Summary:</strong><ul>`)
		for t, c := range typeSummary {
			html.WriteString(fmt.Sprintf(`<li>%s: %d</li>`, html_pkg.EscapeString(t), c))
		}
		html.WriteString(`</ul></div>`)
		// Render groups
		for _, g := range groups {
			html.WriteString(`<div class="failure-group">`)
			html.WriteString(fmt.Sprintf(`<div><strong>Type:</strong> %s</div>`, html_pkg.EscapeString(g.Type)))
			html.WriteString(fmt.Sprintf(`<div><strong>Occurrences:</strong> %d</div>`, g.Count))
			html.WriteString(`<pre>` + html_pkg.EscapeString(g.Sample) + `</pre>`)
			html.WriteString(`</div>`)
		}
		html.WriteString(`</details>`)
	}

	html.WriteString(`
    </div>
</body>
</html>`)

	return html.String()
}
