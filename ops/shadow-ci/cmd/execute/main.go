package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// execute reads a pipeline decision and scoping config, then runs all needed
// categories for a given group. This is the bridge between shadow CI's decision
// engine and actual test execution — one binary that dispatches dynamically
// based on config, so shadow-ci.yml never needs to change when categories are
// added or modified.
func main() {
	group := flag.String("group", "", "Execution group to run (build, go, sol, rust, misc)")
	decisionPath := flag.String("decision", "/tmp/shadow-ci-workspace/decision.json", "Path to decision JSON")
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	resultsDir := flag.String("results-dir", "/tmp/shadow-ci-test-results", "Results output directory")
	dryRun := flag.Bool("dry-run", false, "Print what would run without executing")
	flag.Parse()

	if *group == "" {
		fatal("--group is required (build, go, sol, rust, misc)")
	}

	// Load config and decision.
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

	// Collect categories for this group.
	type catEntry struct {
		name string
		cat  model.JobCategoryConfig
		cd   *model.CategoryDecision
	}
	var entries []catEntry
	for name, cat := range cfg.Scoping.JobCategories {
		if cat.Group != *group {
			continue
		}
		cd, ok := decision.Categories[name]
		if !ok {
			fmt.Printf("  SKIP  %-30s not in decision\n", name)
			continue
		}
		if !cd.Needed || cd.Skipped {
			fmt.Printf("  SKIP  %-30s %s\n", name, cd.SkipWhy)
			continue
		}
		entries = append(entries, catEntry{name: name, cat: cat, cd: cd})
	}

	// Sort by dependency order — categories with no deps first.
	sort.Slice(entries, func(i, j int) bool {
		// Categories that are depended upon should run first.
		for _, dep := range entries[j].cat.DependsOn {
			if dep == entries[i].name {
				return true
			}
		}
		for _, dep := range entries[i].cat.DependsOn {
			if dep == entries[j].name {
				return false
			}
		}
		return entries[i].name < entries[j].name
	})

	if len(entries) == 0 {
		fmt.Printf("No categories to execute for group %q\n", *group)
		os.Exit(0)
	}

	fmt.Printf("=== Shadow CI Execute: group=%s, %d categories ===\n", *group, len(entries))
	for _, e := range entries {
		fmt.Printf("  RUN   %-30s %s\n", e.name, e.cd.Reason)
	}
	fmt.Println()

	// Execute each category.
	type result struct {
		Category string        `json:"category"`
		Status   string        `json:"status"`
		Duration time.Duration `json:"duration_ms"`
		Output   string        `json:"output,omitempty"`
	}
	var results []result
	failures := 0

	for _, e := range entries {
		command := strings.TrimSpace(e.cat.Command)
		if command == "" {
			fmt.Printf("--- %s: no command configured, skipping execution ---\n", e.name)
			results = append(results, result{
				Category: e.name,
				Status:   "no_command",
			})
			continue
		}

		fmt.Printf("--- %s: executing ---\n", e.name)
		fmt.Printf("Command: %s\n", command)

		if *dryRun {
			fmt.Printf("DRY RUN: would execute %q\n\n", command)
			results = append(results, result{
				Category: e.name,
				Status:   "dry_run",
			})
			continue
		}

		start := time.Now()
		cmd := exec.Command("bash", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()

		err := cmd.Run()
		duration := time.Since(start)

		status := "pass"
		if err != nil {
			status = "fail"
			failures++
			fmt.Printf("--- %s: FAILED in %s: %v ---\n\n", e.name, duration.Round(time.Second), err)
		} else {
			fmt.Printf("--- %s: PASSED in %s ---\n\n", e.name, duration.Round(time.Second))
		}

		results = append(results, result{
			Category: e.name,
			Status:   status,
			Duration: duration,
		})
	}

	// Write results summary.
	fmt.Printf("\n=== Shadow CI Execute Summary (group=%s) ===\n", *group)
	for _, r := range results {
		emoji := "✓"
		if r.Status == "fail" {
			emoji = "✗"
		} else if r.Status == "no_command" || r.Status == "dry_run" {
			emoji = "-"
		}
		fmt.Printf("  %s %-30s %s (%s)\n", emoji, r.Category, r.Status, r.Duration.Round(time.Millisecond))
	}

	resultsJSON, _ := json.MarshalIndent(results, "", "  ")
	resultPath := fmt.Sprintf("%s/%s-results.json", *resultsDir, *group)
	os.WriteFile(resultPath, resultsJSON, 0o644)
	fmt.Printf("\nWrote results to %s\n", resultPath)

	if failures > 0 {
		fmt.Printf("\nFAIL: %d/%d categories failed\n", failures, len(entries))
		os.Exit(1)
	}
	fmt.Printf("\nPASS: all %d categories succeeded\n", len(entries))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
