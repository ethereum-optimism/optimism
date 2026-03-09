package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/cache"
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

	// Create cache resolver if caching is enabled.
	var resolver *cache.Resolver
	if *cacheDir != "" {
		resolver = cache.NewResolver(".", *cacheDir)
	}

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

	if len(entries) == 0 {
		fmt.Printf("No categories to execute for group %q\n", *group)
		os.Exit(0)
	}

	// Sort for deterministic display.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	fmt.Printf("=== Shadow CI Execute: group=%s, %d categories ===\n", *group, len(entries))
	for _, e := range entries {
		fmt.Printf("  RUN   %-30s %s\n", e.name, e.cd.Reason)
	}
	fmt.Println()

	type result struct {
		Category string        `json:"category"`
		Status   string        `json:"status"`
		Duration time.Duration `json:"duration_ms"`
		Output   string        `json:"output,omitempty"`
	}

	// Build the set of categories in this group for dep resolution.
	inGroup := make(map[string]bool)
	for _, e := range entries {
		inGroup[e.name] = true
	}

	// Build dependency graph: for each category, track which in-group deps
	// must complete before it can start. Out-of-group deps (e.g. build deps
	// from another group) are assumed already satisfied.
	done := make(map[string]chan struct{})
	for _, e := range entries {
		done[e.name] = make(chan struct{})
	}

	var (
		mu       sync.Mutex
		results  []result
		failures int
	)

	var wg sync.WaitGroup
	for _, e := range entries {
		wg.Add(1)
		go func(e catEntry) {
			defer wg.Done()

			// Wait for in-group dependencies.
			for _, dep := range e.cat.DependsOn {
				if ch, ok := done[dep]; ok {
					<-ch
				}
			}

			command := strings.TrimSpace(e.cat.Command)
			if command == "" {
				fmt.Printf("--- %s: no command configured, skipping execution ---\n", e.name)
				mu.Lock()
				results = append(results, result{Category: e.name, Status: "no_command"})
				mu.Unlock()
				close(done[e.name])
				return
			}

			// Cache resolution for categories with workspace outputs.
			if resolver != nil && len(e.cat.WorkspacePaths) > 0 {
				if *dryRun {
					if key, keyErr := resolver.ComputeKey(e.name, e.cat); keyErr == nil {
						fmt.Printf("--- %s: DRY RUN cache key=%s ---\n", e.name, key)
					}
				} else {
					res, resolveErr := resolver.Resolve(e.name, e.cat)
					if resolveErr != nil {
						fmt.Printf("--- %s: cache key error: %v, proceeding with full build ---\n", e.name, resolveErr)
					} else if res.Hit && res.Verified {
						// Cache hit — restore artifacts and skip build.
						if restoreErr := resolver.Restore(e.name, e.cat); restoreErr != nil {
							fmt.Printf("--- %s: cache restore failed: %v, proceeding with full build ---\n", e.name, restoreErr)
						} else {
							fmt.Printf("--- %s: CACHED (key=%s, verified in %s) ---\n\n", e.name, res.CacheKey, res.Duration.Round(time.Millisecond))
							mu.Lock()
							results = append(results, result{Category: e.name, Status: "cached", Duration: res.Duration})
							mu.Unlock()
							close(done[e.name])
							return
						}
					} else if res.Hit && !res.Verified {
						fmt.Printf("--- %s: CACHE STALE (key=%s matched but verify failed in %s) ---\n", e.name, res.CacheKey, res.Duration.Round(time.Millisecond))
						fmt.Printf("    WARNING: cache key was insufficient — rebuilding\n")
						// Write warning to results dir for observability.
						warnPath := filepath.Join(*resultsDir, e.name+"-cache-verify-failed.json")
						warnJSON, _ := json.Marshal(map[string]string{
							"category": e.name, "cache_key": res.CacheKey,
							"event": "cache.verify_failed",
						})
						os.WriteFile(warnPath, warnJSON, 0o644)
					} else {
						fmt.Printf("--- %s: cache miss (key=%s) ---\n", e.name, res.CacheKey)
					}
				}
			}

			fmt.Printf("--- %s: executing ---\n", e.name)
			fmt.Printf("Command: %s\n", command)

			if *dryRun {
				fmt.Printf("DRY RUN: would execute %q\n\n", command)
				mu.Lock()
				results = append(results, result{Category: e.name, Status: "dry_run"})
				mu.Unlock()
				close(done[e.name])
				return
			}

			// Each category captures output to its own log file to avoid
			// interleaved parallel output. On completion, the tail of the
			// log is printed for visibility.
			logPath := filepath.Join(*resultsDir, e.name+".log")
			logFile, err := os.Create(logPath)
			if err != nil {
				fatal("creating log file for %s: %v", e.name, err)
			}

			start := time.Now()
			cmd := exec.Command("bash", "-c", command)
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			cmd.Env = os.Environ()

			runErr := cmd.Run()
			duration := time.Since(start)
			logFile.Close()

			status := "pass"
			if runErr != nil {
				status = "fail"
				mu.Lock()
				failures++
				mu.Unlock()
				fmt.Printf("--- %s: FAILED in %s: %v ---\n", e.name, duration.Round(time.Second), runErr)
				// Print tail of log for debugging.
				printLogTail(logPath, 30)
				fmt.Println()
			} else {
				fmt.Printf("--- %s: PASSED in %s ---\n\n", e.name, duration.Round(time.Second))
				// Save to cache after successful build.
				if resolver != nil && len(e.cat.WorkspacePaths) > 0 {
					if key, keyErr := resolver.ComputeKey(e.name, e.cat); keyErr == nil {
						if saveErr := resolver.Save(e.name, e.cat, key); saveErr != nil {
							fmt.Printf("    warning: cache save failed: %v\n", saveErr)
						} else {
							fmt.Printf("    cached artifacts for key=%s\n", key)
						}
					}
				}
			}

			mu.Lock()
			results = append(results, result{Category: e.name, Status: status, Duration: duration})
			mu.Unlock()

			// Signal completion so dependents can start, even on failure
			// (let them fail too rather than hang).
			close(done[e.name])
		}(e)
	}

	wg.Wait()

	// Sort results for deterministic output.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})

	fmt.Printf("\n=== Shadow CI Execute Summary (group=%s) ===\n", *group)
	for _, r := range results {
		emoji := "✓"
		if r.Status == "fail" {
			emoji = "✗"
		} else if r.Status == "no_command" || r.Status == "dry_run" {
			emoji = "-"
		} else if r.Status == "cached" {
			emoji = "⚡"
		}
		fmt.Printf("  %s %-30s %s (%s)\n", emoji, r.Category, r.Status, r.Duration.Round(time.Millisecond))
	}

	resultsJSON, _ := json.MarshalIndent(results, "", "  ")
	resultPath := fmt.Sprintf("%s/%s-results.json", *resultsDir, *group)
	os.WriteFile(resultPath, resultsJSON, 0o644)
	fmt.Printf("\nWrote results to %s\n", resultPath)

	if failures > 0 {
		fmt.Printf("\nFAIL: %d/%d categories failed\n", failures, len(results))
		os.Exit(1)
	}
	fmt.Printf("\nPASS: all %d categories succeeded\n", len(results))
}

func printLogTail(path string, lines int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	all := strings.Split(string(data), "\n")
	start := 0
	if len(all) > lines {
		start = len(all) - lines
		fmt.Printf("  ... (%d lines omitted, see %s)\n", start, path)
	}
	for _, line := range all[start:] {
		fmt.Printf("  | %s\n", line)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
