// Package executor runs shadow CI categories in parallel with DAG-based
// dependency scheduling and content-addressed build caching.
//
// The orchestration logic is separated from command execution via the Runner
// interface, making the full DAG/cache/parallel flow locally testable with
// mock runners.
package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// RunContext provides execution context for a category run.
type RunContext struct {
	Category string
	Command  string                    // resolved command (ShellRunner uses this)
	LogPath  string
	Cat      model.JobCategoryConfig   // full category config
	Decision *model.CategoryDecision   // targets, features, configs
}

// Runner executes a category. Implementations control where output goes
// (file, buffer, stdout) and how execution happens (shell, adapter).
type Runner interface {
	Run(ctx RunContext) error
}

// CacheResolver handles content-addressed build cache operations.
type CacheResolver interface {
	ComputeKey(category string, cat model.JobCategoryConfig) (string, error)
	Resolve(category string, cat model.JobCategoryConfig) (*CacheResolution, error)
	Restore(category string, cat model.JobCategoryConfig) error
	Save(category string, cat model.JobCategoryConfig, key string) error
	Verify(cat model.JobCategoryConfig) error
}

// CacheResolution is the result of resolving a category's cache.
type CacheResolution struct {
	CacheKey string
	Hit      bool
}

// Result is the outcome of executing a single category.
type Result struct {
	Category    string             `json:"category"`
	Status      string             `json:"status"` // pass, fail, cached, no_command, dry_run, skipped
	Duration    time.Duration      `json:"duration_ms"`
	TestResults []model.TestResult `json:"test_results,omitempty"`
}

// Config controls executor behavior.
type Config struct {
	Group      string
	DryRun     bool
	ResultsDir string
}

// Executor runs categories for a group with DAG scheduling and caching.
type Executor struct {
	cfg           Config
	runner        Runner          // shell runner (always present)
	adapterRunner *AdapterRunner  // nil disables adapter dispatch
	cache         CacheResolver   // nil disables caching
	scoping       *model.ScopingConfig
	decision      *model.PipelineDecision
}

// New creates an executor. Pass nil for adapterRunner to disable adapter dispatch,
// nil for cache to disable caching.
func New(cfg Config, runner Runner, adapterRunner *AdapterRunner, cache CacheResolver, scoping *model.ScopingConfig, decision *model.PipelineDecision) *Executor {
	return &Executor{
		cfg:           cfg,
		runner:        runner,
		adapterRunner: adapterRunner,
		cache:         cache,
		scoping:       scoping,
		decision:      decision,
	}
}

// isFuzzCategory returns true if the category has fuzz packages, indicating
// fuzz execution semantics that differ from standard test adapters.
func isFuzzCategory(cat model.JobCategoryConfig) bool {
	return len(cat.FuzzPackages) > 0
}

// catEntry pairs a category name with its config and decision.
type catEntry struct {
	name string
	cat  model.JobCategoryConfig
	cd   *model.CategoryDecision
}

// Execute runs all needed categories for the configured group.
// Returns results sorted by category name.
func (e *Executor) Execute() ([]Result, error) {
	// Restore cross-group dependency artifacts from cache before running.
	if e.cache != nil {
		e.restoreCrossGroupDeps()
	}

	entries := e.collectEntries()
	if len(entries) == 0 {
		fmt.Printf("No categories to execute for group %q\n", e.cfg.Group)
		return nil, nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	fmt.Printf("=== Shadow CI Execute: group=%s, %d categories ===\n", e.cfg.Group, len(entries))
	for _, ent := range entries {
		fmt.Printf("  RUN   %-30s %s\n", ent.name, ent.cd.Reason)
	}
	fmt.Println()

	results := e.runDAG(entries)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})
	return results, nil
}

// restoreCrossGroupDeps restores build artifacts from cache for categories
// in other groups that the current group's categories depend on. This replaces
// the old workspace-based artifact passing with cache-based restoration.
func (e *Executor) restoreCrossGroupDeps() {
	// Find all cross-group dependencies.
	restored := make(map[string]bool)
	for _, cat := range e.scoping.JobCategories {
		if cat.Group != e.cfg.Group {
			continue
		}
		for _, dep := range cat.DependsOn {
			depCat, ok := e.scoping.JobCategories[dep]
			if !ok || depCat.Group == e.cfg.Group || depCat.Group == "" {
				continue
			}
			if restored[dep] || len(depCat.WorkspacePaths) == 0 {
				continue
			}
			// Restore this dependency's artifacts from cache.
			if err := e.cache.Restore(dep, depCat); err != nil {
				fmt.Printf("  WARN  failed to restore %s artifacts from cache: %v\n", dep, err)
			} else {
				fmt.Printf("  DEPS  restored %s artifacts from cache\n", dep)
			}
			restored[dep] = true
		}
	}
}

// collectEntries filters categories for the configured group and decision.
func (e *Executor) collectEntries() []catEntry {
	var entries []catEntry
	for name, cat := range e.scoping.JobCategories {
		if cat.Group != e.cfg.Group {
			continue
		}
		cd, ok := e.decision.Categories[name]
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
	return entries
}

// runDAG executes entries respecting dependency order. Independent categories
// run concurrently. Each category signals completion (even on failure) so
// dependents don't hang.
func (e *Executor) runDAG(entries []catEntry) []Result {
	done := make(map[string]chan struct{})
	for _, ent := range entries {
		done[ent.name] = make(chan struct{})
	}

	var (
		mu       sync.Mutex
		results  []Result
		failures int
	)

	var wg sync.WaitGroup
	for _, ent := range entries {
		wg.Add(1)
		go func(ent catEntry) {
			defer wg.Done()

			// Wait for in-group dependencies.
			for _, dep := range ent.cat.DependsOn {
				if ch, ok := done[dep]; ok {
					<-ch
				}
			}

			r := e.executeCategory(ent)

			mu.Lock()
			results = append(results, r)
			if r.Status == "fail" {
				failures++
			}
			mu.Unlock()

			close(done[ent.name])
		}(ent)
	}

	wg.Wait()

	fmt.Printf("\n=== Shadow CI Execute Summary (group=%s) ===\n", e.cfg.Group)
	// Sort for printing.
	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Category < sorted[j].Category
	})
	for _, r := range sorted {
		emoji := "✓"
		switch r.Status {
		case "fail":
			emoji = "✗"
		case "no_command", "dry_run":
			emoji = "-"
		case "cached":
			emoji = "⚡"
		}
		fmt.Printf("  %s %-30s %s (%s)\n", emoji, r.Category, r.Status, r.Duration.Round(time.Millisecond))
	}

	if failures > 0 {
		fmt.Printf("\nFAIL: %d/%d categories failed\n", failures, len(results))
	} else {
		fmt.Printf("\nPASS: all %d categories succeeded\n", len(results))
	}

	return results
}

// executeCategory runs a single category through the full lifecycle:
// resolve command → cache check → execute → save cache.
func (e *Executor) executeCategory(ent catEntry) Result {
	command := ResolveCommand(ent.cat, ent.cd)
	if command == "" {
		fmt.Printf("--- %s: no command configured, skipping execution ---\n", ent.name)
		return Result{Category: ent.name, Status: "no_command"}
	}

	// Cache resolution for categories with workspace outputs.
	if e.cache != nil && len(ent.cat.WorkspacePaths) > 0 {
		if r, done := e.tryCacheRestore(ent); done {
			return r
		}
	}

	isTargeted := strings.TrimSpace(ent.cat.TargetCommand) != "" && len(ent.cd.Targets) > 0
	if isTargeted {
		fmt.Printf("--- %s: executing (targeted, %d targets) ---\n", ent.name, len(ent.cd.Targets))
	} else {
		fmt.Printf("--- %s: executing ---\n", ent.name)
	}
	fmt.Printf("Command: %s\n", command)

	if e.cfg.DryRun {
		fmt.Printf("DRY RUN: would execute %q\n\n", command)
		return Result{Category: ent.name, Status: "dry_run"}
	}

	logPath := filepath.Join(e.cfg.ResultsDir, ent.name+".log")

	// Dispatch to adapter runner for language-backed test categories.
	useAdapter := ent.cat.Language != "" && e.adapterRunner != nil && !isFuzzCategory(ent.cat)

	start := time.Now()
	var runErr error
	if useAdapter {
		runErr = e.adapterRunner.Run(RunContext{
			Category: ent.name,
			LogPath:  logPath,
			Cat:      ent.cat,
			Decision: ent.cd,
		})
	} else {
		runErr = e.runner.Run(RunContext{
			Category: ent.name,
			Command:  command,
			LogPath:  logPath,
			Cat:      ent.cat,
			Decision: ent.cd,
		})
	}
	duration := time.Since(start)

	if runErr != nil {
		fmt.Printf("--- %s: FAILED in %s: %v ---\n", ent.name, duration.Round(time.Second), runErr)
		printLogTail(logPath, 30)
		fmt.Println()
		return Result{Category: ent.name, Status: "fail", Duration: duration}
	}

	fmt.Printf("--- %s: PASSED in %s ---\n\n", ent.name, duration.Round(time.Second))

	// Collect test results from adapter runner.
	r := Result{Category: ent.name, Status: "pass", Duration: duration}
	if useAdapter {
		testResults := e.adapterRunner.Results()
		if len(testResults) > 0 {
			r.TestResults = testResults
			testJSON, _ := json.MarshalIndent(testResults, "", "  ")
			os.WriteFile(filepath.Join(e.cfg.ResultsDir, ent.name+"-tests.json"), testJSON, 0o644)
			fmt.Printf("    %d test results written\n", len(testResults))
		}
	}

	// Save to cache after successful build.
	if e.cache != nil && len(ent.cat.WorkspacePaths) > 0 {
		if key, keyErr := e.cache.ComputeKey(ent.name, ent.cat); keyErr == nil {
			if saveErr := e.cache.Save(ent.name, ent.cat, key); saveErr != nil {
				fmt.Printf("    warning: cache save failed: %v\n", saveErr)
			} else {
				fmt.Printf("    cached artifacts for key=%s\n", key)
			}
		}
	}

	return r
}

// tryCacheRestore attempts cache restore and verification. Returns (result, true)
// if the cache was used (hit+verified) or if dry-run printed the key.
// Returns (zero, false) if the caller should proceed with a full build.
func (e *Executor) tryCacheRestore(ent catEntry) (Result, bool) {
	if e.cfg.DryRun {
		if key, keyErr := e.cache.ComputeKey(ent.name, ent.cat); keyErr == nil {
			fmt.Printf("--- %s: DRY RUN cache key=%s ---\n", ent.name, key)
		}
		return Result{}, false
	}

	res, resolveErr := e.cache.Resolve(ent.name, ent.cat)
	if resolveErr != nil {
		fmt.Printf("--- %s: cache key error: %v, proceeding with full build ---\n", ent.name, resolveErr)
		return Result{}, false
	}

	if !res.Hit {
		fmt.Printf("--- %s: cache miss (key=%s) ---\n", ent.name, res.CacheKey)
		return Result{}, false
	}

	// Key matched — restore artifacts, then verify.
	if restoreErr := e.cache.Restore(ent.name, ent.cat); restoreErr != nil {
		fmt.Printf("--- %s: cache restore failed: %v, proceeding with full build ---\n", ent.name, restoreErr)
		return Result{}, false
	}

	start := time.Now()
	verifyErr := e.cache.Verify(ent.cat)
	verifyDur := time.Since(start)

	if verifyErr == nil {
		fmt.Printf("--- %s: CACHED (key=%s, verified in %s) ---\n\n", ent.name, res.CacheKey, verifyDur.Round(time.Millisecond))
		return Result{Category: ent.name, Status: "cached", Duration: verifyDur}, true
	}

	fmt.Printf("--- %s: CACHE STALE (key=%s matched, restored, but verify failed in %s) ---\n", ent.name, res.CacheKey, verifyDur.Round(time.Millisecond))
	fmt.Printf("    verify error: %v\n", verifyErr)
	fmt.Printf("    WARNING: cache key was insufficient — rebuilding\n")

	if e.cfg.ResultsDir != "" {
		warnPath := filepath.Join(e.cfg.ResultsDir, ent.name+"-cache-verify-failed.json")
		warnJSON, _ := json.Marshal(map[string]string{
			"category": ent.name, "cache_key": res.CacheKey,
			"event": "cache.verify_failed",
		})
		os.WriteFile(warnPath, warnJSON, 0o644)
	}

	return Result{}, false
}

// ResolveCommand picks the right command for a category:
// - If target_command is set AND the decision has targets, substitute and use it.
// - Otherwise, fall back to the static command.
//
// Placeholders in target_command:
//
//	{{targets}}       — space-separated target list
//	{{targets_csv}}   — comma-separated target list
//	{{targets_glob}}  — brace glob: {a,b,c} (single target returned bare)
func ResolveCommand(cat model.JobCategoryConfig, cd *model.CategoryDecision) string {
	targetCmd := strings.TrimSpace(cat.TargetCommand)
	if targetCmd != "" && len(cd.Targets) > 0 {
		space := strings.Join(cd.Targets, " ")
		csv := strings.Join(cd.Targets, ",")
		var glob string
		if len(cd.Targets) == 1 {
			glob = cd.Targets[0]
		} else {
			glob = "{" + csv + "}"
		}

		resolved := targetCmd
		resolved = strings.ReplaceAll(resolved, "{{targets}}", space)
		resolved = strings.ReplaceAll(resolved, "{{targets_csv}}", csv)
		resolved = strings.ReplaceAll(resolved, "{{targets_glob}}", glob)
		return resolved
	}
	return strings.TrimSpace(cat.Command)
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
