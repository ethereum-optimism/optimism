// check-runner is a parallel check execution tool for contracts-bedrock.
//
// It reads a checks.yaml configuration file that defines phases of checks to run.
// Each phase can optionally require a build step, and checks within a phase run
// in parallel by default (with dependency support for ordering).
//
// Key features:
//   - Parallel execution of independent checks within each phase
//   - Build caching based on source file hashes (SHA256 of all .sol files)
//   - Automatic artifact preservation and restoration
//   - Graceful shutdown on Ctrl+C (restores artifacts before exit)
//   - Dependency-based ordering within phases
//   - Pretty terminal output with spinners and colors
//
// Usage:
//
//	go run ./scripts/check-runner -config checks.yaml
//	go run ./scripts/check-runner -run lint,snapshots
//	go run ./scripts/check-runner -list
//	go run ./scripts/check-runner -no-build
//	go run ./scripts/check-runner -clean
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chelnak/ysmrr"
	"github.com/chelnak/ysmrr/pkg/colors"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// ANSI Color Codes
// =============================================================================

// Terminal color codes for pretty output.
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Cyan      = "\033[36m"
	BoldGreen = "\033[1;32m"
	BoldRed   = "\033[1;31m"
	BoldCyan  = "\033[1;36m"
)

// =============================================================================
// Cache Configuration
// =============================================================================

const (
	// MaxCacheCount is the maximum number of cached builds to keep per phase.
	// Older caches are evicted using LRU.
	MaxCacheCount = 5

	// Baseline times for the old check script (in seconds).
	// Used to calculate time saved.
	BaselineCacheHit   = 90.0  // 1.5 minutes when cache hit
	BaselineNoCacheHit = 180.0 // 3.0 minutes when no cache hit
)

// Paths are computed at runtime to use user's home directory.
var (
	// CacheDir is where build artifacts are cached between runs.
	// Keyed by phase name and source hash.
	CacheDir string

	// StatsFile is where cumulative time savings are stored.
	StatsFile string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	CacheDir = filepath.Join(home, ".cache", "check-runner", "builds")
	StatsFile = filepath.Join(home, ".cache", "check-runner", "stats.json")
}

// artifactDirs are the directories containing build artifacts.
// These are saved/restored for caching and preserved across check runs.
var artifactDirs = []string{"artifacts", "forge-artifacts", "cache"}

// =============================================================================
// Stats Tracking
// =============================================================================

// Stats tracks cumulative time savings across runs.
type Stats struct {
	TotalRuns      int     `json:"total_runs"`
	TotalTimeSaved float64 `json:"total_time_saved"` // in seconds
}

// loadStats reads the stats file from disk.
func loadStats() Stats {
	data, err := os.ReadFile(StatsFile)
	if err != nil {
		return Stats{}
	}
	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		return Stats{}
	}
	return stats
}

// saveStats writes the stats file to disk.
func saveStats(stats Stats) {
	data, err := json.Marshal(stats)
	if err != nil {
		return
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(StatsFile), 0755); err != nil {
		return
	}
	_ = os.WriteFile(StatsFile, data, 0644)
}

// formatDuration formats seconds into a human-readable string.
func formatDuration(seconds float64) string {
	secs := int(seconds + 0.5) // Round to nearest second
	if secs < 60 {
		return fmt.Sprintf("%d seconds", secs)
	}
	minutes := secs / 60
	secs = secs % 60
	if minutes < 60 {
		if secs == 0 {
			return fmt.Sprintf("%d minutes", minutes)
		}
		return fmt.Sprintf("%d minutes %d seconds", minutes, secs)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, mins)
}

// formatDurationHoursMinutes formats seconds into compact hours and minutes (e.g., "2h15m").
func formatDurationHoursMinutes(seconds float64) string {
	minutes := int(seconds) / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// cacheRestoreDirs are the directories restored from cache before builds.
// We restore all artifact directories for incremental builds, then clean stale artifacts.
var cacheRestoreDirs = []string{"artifacts", "forge-artifacts", "cache"}

// =============================================================================
// Configuration Types
// =============================================================================

// Check represents a single check to run.
type Check struct {
	Name        string   `yaml:"name"`        // Unique identifier for the check
	Description string   `yaml:"description"` // Human-readable description
	Command     string   `yaml:"command"`     // Shell command to execute
	Depends     []string `yaml:"depends"`     // Names of checks that must pass first
	RetryClean  bool     `yaml:"retry-clean"` // If true, retry with clean build on failure
}

// Phase represents a group of related checks.
// Checks within a phase run in parallel by default.
type Phase struct {
	Name     string  `yaml:"name"`     // Phase identifier (e.g., "setup", "source", "dev")
	Build    string  `yaml:"build"`    // Optional build command to run before checks
	Parallel *bool   `yaml:"parallel"` // Whether to run checks in parallel (default: true)
	Checks   []Check `yaml:"checks"`   // Checks to run in this phase
}

// Config is the top-level configuration loaded from checks.yaml.
type Config struct {
	Phases []Phase `yaml:"phases"`
}

// =============================================================================
// Execution State Types
// =============================================================================

// CheckResult holds the outcome of running a single check.
type CheckResult struct {
	Name     string
	Success  bool
	Output   string // Combined stdout/stderr
	Duration time.Duration
}

// checkState tracks the execution state of a check during parallel runs.
type checkState struct {
	status  string // "pending", "queued", "running", "pass", "fail", "skipped"
	spinner *ysmrr.Spinner
}

// Runner orchestrates the execution of all checks.
type Runner struct {
	config  *Config
	results map[string]*CheckResult
	states  map[string]*checkState
	mu      sync.Mutex

	// Configuration flags
	noBuild bool // Skip phases that require builds
	verbose bool // Show output for all checks, not just failures
	noCache bool // Disable build caching
	clean   bool // Clean artifacts before each build

	// Build state
	tempDir    string // Temp directory for preserving working artifacts
	sourceHash string // SHA256 hash of source files for cache key

	// Results tracking
	totalPassed  int
	totalFailed  int
	totalSkipped int
	failedChecks []string
	buildError   string // Build failure output to display at end

	// Retry-clean tracking
	retryCleanChecks []string // Checks that failed and have retry-clean enabled

	// Graceful shutdown support
	interrupted atomic.Bool  // Set to true on first Ctrl+C
	sigCount    atomic.Int32 // Number of times Ctrl+C pressed
	cancelFunc  context.CancelFunc

	// Timing and stats
	startTime   time.Time
	hadCacheHit bool // Whether any phase had a cache hit
	isFullRun   bool // Whether this is a full run (no -run filter)
}

// =============================================================================
// Entry Point
// =============================================================================

func main() {
	var (
		configPath string
		listChecks bool
		runChecks  string
		noBuild    bool
		verbose    bool
		noCache    bool
		clean      bool
	)

	flag.StringVar(&configPath, "config", "", "Path to checks.yaml config file")
	flag.BoolVar(&listChecks, "list", false, "List available checks")
	flag.StringVar(&runChecks, "run", "", "Run specific check(s), comma-separated")
	flag.BoolVar(&noBuild, "no-build", false, "Skip phases that have builds")
	flag.BoolVar(&verbose, "verbose", false, "Show output for all checks, not just failures")
	flag.BoolVar(&noCache, "no-cache", false, "Disable build caching")
	flag.BoolVar(&clean, "clean", false, "Clean build artifacts before each build (forces fresh compilation)")
	flag.Parse()

	// Find config file
	if configPath == "" {
		// Default to checks.yaml in current directory
		if _, err := os.Stat("checks.yaml"); err == nil {
			configPath = "checks.yaml"
		} else {
			fmt.Fprintf(os.Stderr, "Error: could not find checks.yaml (use -config to specify path)\n")
			os.Exit(1)
		}
	}

	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if listChecks {
		printCheckList(config)
		return
	}

	runner := &Runner{
		config:       config,
		results:      make(map[string]*CheckResult),
		states:       make(map[string]*checkState),
		noBuild:      noBuild,
		verbose:      verbose,
		noCache:      noCache,
		clean:        clean,
		failedChecks: []string{},
	}

	// Parse which checks to run (if specified)
	var selectedChecks map[string]bool
	if runChecks != "" {
		selectedChecks = make(map[string]bool)
		for _, name := range strings.Split(runChecks, ",") {
			name = strings.TrimSpace(name)
			if !runner.checkExists(name) {
				fmt.Fprintf(os.Stderr, "Error: unknown check '%s'\n", name)
				fmt.Fprintf(os.Stderr, "Run with -list to see available checks\n")
				os.Exit(1)
			}
			selectedChecks[name] = true
		}
	}

	success := runner.Run(selectedChecks)
	if !success {
		os.Exit(1)
	}
}

// =============================================================================
// Configuration Loading
// =============================================================================

// loadConfig reads and parses the checks.yaml configuration file.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// printCheckList displays all available checks grouped by phase.
func printCheckList(config *Config) {
	fmt.Println("Available checks:")
	fmt.Println()

	for _, phase := range config.Phases {
		fmt.Printf("%s%s%s", BoldCyan, phase.Name, Reset)
		if phase.Build != "" {
			fmt.Printf(" %s(builds)%s", Dim, Reset)
		}
		if phase.Parallel != nil && !*phase.Parallel {
			fmt.Printf(" %s(sequential)%s", Dim, Reset)
		}
		fmt.Println()

		maxNameLen := 0
		for _, c := range phase.Checks {
			if len(c.Name) > maxNameLen {
				maxNameLen = len(c.Name)
			}
		}

		for _, c := range phase.Checks {
			info := ""
			if len(c.Depends) > 0 {
				info = fmt.Sprintf(" %s→ %s%s", Dim, strings.Join(c.Depends, ", "), Reset)
			}
			fmt.Printf("  %-*s  %s%s%s%s\n", maxNameLen, c.Name, Dim, c.Description, Reset, info)
		}
		fmt.Println()
	}
}

// checkExists returns true if a check with the given name exists.
func (r *Runner) checkExists(name string) bool {
	for _, phase := range r.config.Phases {
		for _, c := range phase.Checks {
			if c.Name == name {
				return true
			}
		}
	}
	return false
}

// getCheck returns the Check with the given name, or nil if not found.
func (r *Runner) getCheck(name string) *Check {
	for _, phase := range r.config.Phases {
		for i := range phase.Checks {
			if phase.Checks[i].Name == name {
				return &phase.Checks[i]
			}
		}
	}
	return nil
}

// =============================================================================
// Build Caching
// =============================================================================

// computeSourceHash calculates a SHA256 hash of all Solidity source files
// and foundry.toml. This hash is used as the cache key for build artifacts.
func computeSourceHash() (string, error) {
	h := sha256.New()

	var files []string

	// Walk src/ and interfaces/ directories for .sol files
	sourceDirs := []string{"src", "interfaces"}
	for _, dir := range sourceDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil // Directory doesn't exist, skip
				}
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".sol") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	// Include foundry.toml as it affects compilation
	if _, err := os.Stat("foundry.toml"); err == nil {
		files = append(files, "foundry.toml")
	}

	// Sort for deterministic hashing
	sort.Strings(files)

	// Hash each file's path and contents
	for _, path := range files {
		h.Write([]byte(path))
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}

	// Return first 16 hex chars (64 bits) - enough for cache uniqueness
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// getCachePath returns the filesystem path for a cached build.
func getCachePath(phaseName, hash string) string {
	return filepath.Join(CacheDir, phaseName, hash)
}

// cacheExists checks if a cached build exists for the given phase and hash.
func cacheExists(phaseName, hash string) bool {
	path := getCachePath(phaseName, hash)
	_, err := os.Stat(path)
	return err == nil
}

// getLatestCachePath returns the path to the "latest" symlink for a phase.
func getLatestCachePath(phaseName string) string {
	return filepath.Join(CacheDir, phaseName, "latest")
}

// getLatestCache returns the hash of the most recent cached build for a phase.
func getLatestCache(phaseName string) string {
	latestPath := getLatestCachePath(phaseName)
	target, err := os.Readlink(latestPath)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

// restoreFromCache restores build artifacts from a cached build.
func restoreFromCache(phaseName, hash string) error {
	cachePath := getCachePath(phaseName, hash)
	if _, err := os.Stat(cachePath); err != nil {
		return fmt.Errorf("cache not found: %s", cachePath)
	}

	for _, dir := range cacheRestoreDirs {
		src := filepath.Join(cachePath, dir)
		if _, err := os.Stat(src); err == nil {
			os.RemoveAll(dir)
			if err := copyDir(src, dir); err != nil {
				return fmt.Errorf("failed to restore %s: %w", dir, err)
			}
		}
	}

	return nil
}

// saveToCache saves current build artifacts to the cache.
func saveToCache(phaseName, hash string) error {
	cachePath := getCachePath(phaseName, hash)

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return err
	}

	for _, dir := range artifactDirs {
		if _, err := os.Stat(dir); err == nil {
			dst := filepath.Join(cachePath, dir)
			// Remove existing cache directory first to ensure clean overwrite
			os.RemoveAll(dst)
			if err := copyDir(dir, dst); err != nil {
				return fmt.Errorf("failed to cache %s: %w", dir, err)
			}
		}
	}

	// Update "latest" symlink to point to this cache
	latestPath := getLatestCachePath(phaseName)
	os.Remove(latestPath)
	if err := os.Symlink(hash, latestPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update latest symlink: %v\n", err)
	}

	// Clean up old caches to stay under MaxCacheCount
	evictOldCaches(phaseName)

	return nil
}

// evictOldCaches removes old cached builds, keeping only the MaxCacheCount most recent.
func evictOldCaches(phaseName string) {
	cacheTypeDir := filepath.Join(CacheDir, phaseName)
	entries, err := os.ReadDir(cacheTypeDir)
	if err != nil {
		return
	}

	type cacheEntry struct {
		name    string
		modTime time.Time
	}
	var caches []cacheEntry

	for _, entry := range entries {
		if entry.Name() == "latest" {
			continue // Skip the symlink
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		caches = append(caches, cacheEntry{
			name:    entry.Name(),
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time, newest first
	sort.Slice(caches, func(i, j int) bool {
		return caches[i].modTime.After(caches[j].modTime)
	})

	// Remove caches beyond the limit
	for i := MaxCacheCount; i < len(caches); i++ {
		path := filepath.Join(cacheTypeDir, caches[i].name)
		os.RemoveAll(path)
	}
}

// =============================================================================
// Main Execution
// =============================================================================

// Run executes all configured checks, optionally filtering to selectedChecks.
// Returns true if all checks passed, false otherwise.
func (r *Runner) Run(selectedChecks map[string]bool) bool {
	r.startTime = time.Now()
	r.isFullRun = selectedChecks == nil // Full run if no filter specified

	// Check if any phase has a build step
	hasBuilds := false
	for _, phase := range r.config.Phases {
		if phase.Build != "" {
			hasBuilds = true
			break
		}
	}

	// Save current working artifacts so we can restore them after checks complete.
	// This ensures the user's build state is preserved even if checks modify artifacts.
	if hasBuilds && !r.noBuild {
		if err := r.saveWorkingArtifacts(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save artifacts: %v\n", err)
		}
		defer r.restoreWorkingArtifacts()
	}

	// Set up signal handling for graceful shutdown.
	// First Ctrl+C sets interrupted flag and restores artifacts.
	// Third Ctrl+C force exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for sig := range sigChan {
			count := r.sigCount.Add(1)
			if count == 1 {
				fmt.Printf("\n%s⚠ Interrupted%s - restoring artifacts...%s\n", Yellow, Dim, Reset)
				r.interrupted.Store(true)
				cancel()
			} else if count >= 3 {
				fmt.Printf("\n%s✗ Force exit%s\n", BoldRed, Reset)
				os.Exit(130) // Standard exit code for SIGINT
			} else {
				fmt.Printf("\n%sPress Ctrl+C %d more time(s) to force exit%s\n", Dim, 3-count, Reset)
			}
			_ = sig // acknowledge
		}
	}()
	defer signal.Stop(sigChan)

	hashComputed := false
	_ = ctx // Used by signal handler via cancel()

	// Execute each phase in order
	for _, phase := range r.config.Phases {
		// Check for interruption at start of each phase
		if r.interrupted.Load() {
			break
		}

		// Filter checks for this phase if specific checks were requested
		var checksToRun []Check
		for _, c := range phase.Checks {
			if selectedChecks == nil || selectedChecks[c.Name] {
				checksToRun = append(checksToRun, c)
			}
		}

		if len(checksToRun) == 0 {
			continue
		}

		// Skip phases with builds if -no-build flag was passed
		if phase.Build != "" && r.noBuild {
			fmt.Printf("%s⊘ %s%s %s(skipped - no build)%s\n", Yellow, phase.Name, Reset, Dim, Reset)
			continue
		}

		// Print phase header
		fmt.Printf("\n%s→ %s%s\n", BoldCyan, phase.Name, Reset)

		// Compute source hash before first build phase (for cache key)
		if phase.Build != "" && !hashComputed && !r.noCache {
			hash, err := computeSourceHash()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not compute source hash: %v\n", err)
			} else {
				r.sourceHash = hash
			}
			hashComputed = true
		}

		// Run build step if this phase requires it
		if phase.Build != "" {
			if err := r.doBuildWithCache(phase.Name, phase.Build); err != nil {
				if r.interrupted.Load() {
					break // Don't print build failed if interrupted
				}
				fmt.Fprintf(os.Stderr, "%s✗ Build failed%s\n", BoldRed, Reset)
				r.printFinalSummary()
				return false
			}
		}

		// Check for interruption after build
		if r.interrupted.Load() {
			break
		}

		// Run the checks (parallel by default)
		parallel := phase.Parallel == nil || *phase.Parallel
		r.runPhaseChecks(checksToRun, parallel)
	}

	// Check if any failed checks have retry-clean enabled
	if !r.interrupted.Load() && len(r.retryCleanChecks) > 0 {
		r.runRetryClean()
	}

	r.printFinalSummary()
	return r.totalFailed == 0 && !r.interrupted.Load()
}

// runRetryClean re-runs failed checks that have retry-clean enabled after a clean build.
func (r *Runner) runRetryClean() {
	// Group retry checks by their phase's build command
	checksByBuild := make(map[string][]string)
	for _, checkName := range r.retryCleanChecks {
		check := r.getCheck(checkName)
		if check == nil {
			continue
		}
		// Find the phase this check belongs to
		for _, phase := range r.config.Phases {
			for _, c := range phase.Checks {
				if c.Name == checkName {
					checksByBuild[phase.Build] = append(checksByBuild[phase.Build], checkName)
					break
				}
			}
		}
	}

	fmt.Printf("\n%s→ retry-clean%s\n", BoldCyan, Reset)
	fmt.Printf("%s⟳ Retrying %d check(s) with clean build...%s\n", Dim, len(r.retryCleanChecks), Reset)

	// Clean all artifacts
	for _, dir := range artifactDirs {
		os.RemoveAll(dir)
	}

	// Re-run builds and checks for each build type that has retry checks
	for buildCmd, checkNames := range checksByBuild {
		if r.interrupted.Load() {
			return
		}

		// Find phase name for this build
		phaseName := ""
		for _, phase := range r.config.Phases {
			if phase.Build == buildCmd {
				phaseName = phase.Name
				break
			}
		}

		// Run the build (clean, no cache)
		if buildCmd != "" {
			if err := r.doBuildClean(phaseName, buildCmd); err != nil {
				fmt.Fprintf(os.Stderr, "%s✗ Clean build failed%s\n", BoldRed, Reset)
				return
			}
		}

		// Re-run the failed checks
		for _, checkName := range checkNames {
			if r.interrupted.Load() {
				return
			}
			r.runRetryCheck(checkName)
		}
	}
}

// doBuildClean runs a build without using cache (for retry-clean).
func (r *Runner) doBuildClean(phaseName, buildCmd string) error {
	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)
	spinner := sm.AddSpinner(fmt.Sprintf("Clean building (%s)", buildCmd))
	sm.Start()

	startTime := time.Now()
	cmd := exec.Command("sh", "-c", buildCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		spinner.UpdateMessage(fmt.Sprintf("%sClean build failed%s (%s) %s%.1fs%s", Red, Reset, buildCmd, Dim, duration.Seconds(), Reset))
		spinner.Error()
		sm.Stop()
		output := stdout.String() + stderr.String()
		if output != "" {
			r.buildError = output
		}
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("%sClean built%s (%s) %s%.1fs%s", Green, Reset, buildCmd, Dim, duration.Seconds(), Reset))
	spinner.Complete()
	sm.Stop()

	// Save clean build to cache
	if r.sourceHash != "" && !r.noCache {
		fmt.Printf("%s⟳ Saving clean build to cache %s...%s\n", Dim, r.sourceHash[:8], Reset)
		if err := saveToCache(phaseName, r.sourceHash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save to cache: %v\n", err)
		}
	}

	return nil
}

// runRetryCheck re-runs a single check and updates the results.
func (r *Runner) runRetryCheck(name string) {
	check := r.getCheck(name)
	if check == nil {
		return
	}

	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)
	spinner := sm.AddSpinner(fmt.Sprintf("%s (retry)", name))
	sm.Start()

	startTime := time.Now()
	cmd := exec.Command("sh", "-c", check.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	timeStr := fmt.Sprintf("%s%.1fs%s", Dim, duration.Seconds(), Reset)

	if err == nil {
		// Retry succeeded - update results
		spinner.UpdateMessage(fmt.Sprintf("%s (retry) %s", name, timeStr))
		spinner.Complete()
		sm.Stop()

		// Update totals: was failed, now passed
		r.totalFailed--
		r.totalPassed++

		// Remove from failed checks list
		newFailed := []string{}
		for _, n := range r.failedChecks {
			if n != name {
				newFailed = append(newFailed, n)
			}
		}
		r.failedChecks = newFailed

		// Update result
		r.results[name] = &CheckResult{
			Name:     name,
			Success:  true,
			Output:   output,
			Duration: duration,
		}
	} else {
		// Retry also failed
		spinner.UpdateMessage(fmt.Sprintf("%s (retry) %s", name, timeStr))
		spinner.Error()
		sm.Stop()

		// Update result with new output
		r.results[name] = &CheckResult{
			Name:     name,
			Success:  false,
			Output:   output,
			Duration: duration,
		}
	}
}

// printFinalSummary displays the final results including any failures.
func (r *Runner) printFinalSummary() {
	fmt.Println()

	// If interrupted, just confirm artifacts were restored
	if r.interrupted.Load() {
		fmt.Printf("%s✓ Artifacts restored%s\n", Green, Reset)
		return
	}

	// Print build error with box drawing characters
	if r.buildError != "" {
		fmt.Printf("%s┌─ build%s\n", Red, Reset)
		lines := strings.Split(strings.TrimSpace(r.buildError), "\n")
		for _, line := range lines {
			fmt.Printf("%s│%s %s\n", Red, Reset, line)
		}
		fmt.Printf("%s└%s\n", Red, Reset)
		fmt.Println()
	}

	// Print failed check details with box drawing characters
	if len(r.failedChecks) > 0 {
		for _, name := range r.failedChecks {
			result := r.results[name]
			if result != nil && result.Output != "" {
				fmt.Printf("%s┌─ %s%s\n", Red, name, Reset)
				lines := strings.Split(strings.TrimSpace(result.Output), "\n")
				for _, line := range lines {
					fmt.Printf("%s│%s %s\n", Red, Reset, line)
				}
				fmt.Printf("%s└%s\n", Red, Reset)
				fmt.Println()
			}
		}
	}

	// Print final status line
	total := r.totalPassed + r.totalFailed + r.totalSkipped
	if r.buildError != "" {
		fmt.Printf("%s✗ Build failed%s\n", BoldRed, Reset)
	} else if r.totalFailed == 0 {
		fmt.Printf("%s✓ All checks passed%s", BoldGreen, Reset)
		fmt.Printf(" %s(%d/%d)%s\n", Dim, r.totalPassed, total, Reset)
	} else {
		fmt.Printf("%s✗ %d check(s) failed%s", BoldRed, r.totalFailed, Reset)
		fmt.Printf(" %s(%d passed, %d failed)%s\n", Dim, r.totalPassed, r.totalFailed, Reset)
	}

	// Print timing stats
	r.printTimingStats()
}

// printTimingStats displays execution time and cumulative time saved.
func (r *Runner) printTimingStats() {
	duration := time.Since(r.startTime).Seconds()

	// Print this run's time
	fmt.Printf("\n%sThis run took %s%s\n", Dim, formatDuration(duration), Reset)

	// Only track stats for full runs
	if !r.isFullRun {
		return
	}

	// Select baseline based on cache hit
	baseline := BaselineNoCacheHit
	if r.hadCacheHit {
		baseline = BaselineCacheHit
	}

	// Calculate time saved (baseline - actual)
	timeSaved := baseline - duration
	if timeSaved < 0 {
		timeSaved = 0
	}

	// Load and update stats
	stats := loadStats()
	stats.TotalRuns++
	stats.TotalTimeSaved += timeSaved
	saveStats(stats)

	// Print cumulative time saved (only if at least 1 minute saved)
	if stats.TotalTimeSaved >= 60 {
		fmt.Printf("%sYou've saved ~%s using check-fast%s\n", Green, formatDurationHoursMinutes(stats.TotalTimeSaved), Reset)
	}
}

// =============================================================================
// Build Execution
// =============================================================================

// doBuildWithCache runs a build command, using caching when possible.
// It handles cache restoration, incremental builds, and cache saving.
func (r *Runner) doBuildWithCache(phaseName, buildCmd string) error {
	hash := r.sourceHash
	cacheHit := false

	if r.clean {
		// Clean build: remove all artifacts first
		fmt.Printf("%s⟳ Cleaning artifacts...%s\n", Dim, Reset)
		for _, dir := range artifactDirs {
			os.RemoveAll(dir)
		}
	} else {
		// Try to restore from exact cache match
		if hash != "" && !r.noCache && cacheExists(phaseName, hash) {
			fmt.Printf("%s⟳ Restoring from cache %s...%s\n", Dim, hash[:8], Reset)
			if err := restoreFromCache(phaseName, hash); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cache restore failed: %v\n", err)
			} else {
				cacheHit = true
				r.hadCacheHit = true
			}
		}

		// If no exact match, try to restore latest cache for incremental build
		if !cacheHit && hash != "" && !r.noCache {
			latest := getLatestCache(phaseName)
			if latest != "" && latest != hash {
				fmt.Printf("%s⟳ Restoring latest cache for incremental build...%s\n", Dim, Reset)
				if err := restoreFromCache(phaseName, latest); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not restore latest cache: %v\n", err)
				}
			}
		}
	}

	// Run the build with a spinner
	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)
	spinner := sm.AddSpinner(fmt.Sprintf("Building (%s)", buildCmd))
	sm.Start()

	startTime := time.Now()
	cmd := exec.Command("sh", "-c", buildCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		spinner.UpdateMessage(fmt.Sprintf("%sBuild failed%s (%s) %s%.1fs%s", Red, Reset, buildCmd, Dim, duration.Seconds(), Reset))
		spinner.Error()
		sm.Stop()
		// Store build output for display in final summary
		output := stdout.String() + stderr.String()
		if output != "" {
			r.buildError = output
		}
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("%sBuilt%s (%s) %s%.1fs%s", Green, Reset, buildCmd, Dim, duration.Seconds(), Reset))
	spinner.Complete()
	sm.Stop()

	// Always save to cache after successful build.
	// Even on cache hit, tests/scripts may have changed and we want the latest artifacts cached.
	if hash != "" && !r.noCache {
		fmt.Printf("%s⟳ Saving to cache %s...%s\n", Dim, hash[:8], Reset)
		if err := saveToCache(phaseName, hash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save to cache: %v\n", err)
		}
	}

	return nil
}

// =============================================================================
// Artifact Preservation
// =============================================================================

// saveWorkingArtifacts copies current build artifacts to a temp directory.
// This preserves the user's working state before checks potentially modify artifacts.
func (r *Runner) saveWorkingArtifacts() error {
	tempDir, err := os.MkdirTemp("", "check-runner-working-")
	if err != nil {
		return err
	}
	r.tempDir = tempDir

	for _, dir := range artifactDirs {
		if _, err := os.Stat(dir); err == nil {
			dst := filepath.Join(tempDir, dir)
			if err := copyDir(dir, dst); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save %s: %v\n", dir, err)
			}
		}
	}
	return nil
}

// restoreWorkingArtifacts restores the user's original build artifacts.
// Called after checks complete or on interrupt.
func (r *Runner) restoreWorkingArtifacts() {
	if r.tempDir == "" {
		return
	}
	defer os.RemoveAll(r.tempDir)

	for _, dir := range artifactDirs {
		src := filepath.Join(r.tempDir, dir)
		if _, err := os.Stat(src); err == nil {
			os.RemoveAll(dir)
			if err := copyDir(src, dir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not restore %s: %v\n", dir, err)
			}
		}
	}
}

// copyDir copies a directory recursively using cp -r.
func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-r", src, dst)
	return cmd.Run()
}

// =============================================================================
// Check Execution
// =============================================================================

// runPhaseChecks runs all checks for a phase, either in parallel or sequentially.
func (r *Runner) runPhaseChecks(checks []Check, parallel bool) {
	if len(checks) == 0 {
		return
	}

	if parallel {
		r.runChecksParallel(checks)
	} else {
		r.runChecksSequential(checks)
	}
}

// runChecksSequential runs checks one at a time in order.
func (r *Runner) runChecksSequential(checks []Check) {
	for _, check := range checks {
		// Check for interruption before each check
		if r.interrupted.Load() {
			return
		}

		startTime := time.Now()
		cmd := exec.Command("sh", "-c", check.Command)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		duration := time.Since(startTime)

		// Check for interruption after command completes
		if r.interrupted.Load() {
			return
		}

		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		result := &CheckResult{
			Name:     check.Name,
			Success:  err == nil,
			Output:   output,
			Duration: duration,
		}
		r.results[check.Name] = result

		if result.Success {
			fmt.Printf("%s✓%s %s %s%.1fs%s\n", Green, Reset, check.Name, Dim, duration.Seconds(), Reset)
			r.totalPassed++
		} else if check.RetryClean {
			// Show retry indicator for checks that will retry with clean build
			fmt.Printf("%s↻%s %s %s%.1fs%s %s(will retry with clean build)%s\n", Yellow, Reset, check.Name, Dim, duration.Seconds(), Reset, Yellow, Reset)
			r.totalFailed++
			r.failedChecks = append(r.failedChecks, check.Name)
			r.retryCleanChecks = append(r.retryCleanChecks, check.Name)
		} else {
			fmt.Printf("%s✗%s %s %s%.1fs%s\n", Red, Reset, check.Name, Dim, duration.Seconds(), Reset)
			r.totalFailed++
			r.failedChecks = append(r.failedChecks, check.Name)
		}
	}
}

// runChecksParallel runs checks concurrently with dependency ordering.
// Uses a worker pool and respects check dependencies.
func (r *Runner) runChecksParallel(checks []Check) {
	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)

	// Initialize state for each check with a spinner
	checkNames := make([]string, len(checks))
	for i, c := range checks {
		checkNames[i] = c.Name
		spinner := sm.AddSpinner(c.Name)
		r.states[c.Name] = &checkState{status: "pending", spinner: spinner}
	}

	sm.Start()

	// Channel for checks ready to run (dependencies satisfied)
	ready := make(chan string, len(checks))
	var wg sync.WaitGroup

	// Queue checks with no dependencies
	r.mu.Lock()
	for _, name := range checkNames {
		if r.depsReady(name, checkNames) {
			r.states[name].status = "queued"
			ready <- name
		}
	}
	r.mu.Unlock()

	// Start worker pool
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range ready {
				// Check for interruption before starting each check
				if r.interrupted.Load() {
					return
				}
				r.runCheckParallel(name)
			}
		}()
	}

	// Monitor loop: queue checks as their dependencies complete
	go func() {
		for {
			// Check for interruption
			if r.interrupted.Load() {
				close(ready)
				return
			}

			r.mu.Lock()
			allDone := true
			for _, name := range checkNames {
				state := r.states[name]
				if state.status == "pending" {
					// Skip checks whose dependencies failed
					if r.depsFailed(name, checkNames) {
						state.status = "skipped"
						state.spinner.UpdateMessage(fmt.Sprintf("  %s %s(skipped)%s", name, Dim, Reset))
						state.spinner.Error()
						r.totalSkipped++
						continue
					}
					// Queue checks whose dependencies passed
					if r.depsReady(name, checkNames) {
						state.status = "queued"
						ready <- name
					} else {
						allDone = false
					}
				} else if state.status == "running" || state.status == "queued" {
					allDone = false
				}
			}
			r.mu.Unlock()

			if allDone {
				close(ready)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	wg.Wait()
	sm.Stop()

	// Tally results
	r.mu.Lock()
	for _, name := range checkNames {
		state := r.states[name]
		if state == nil {
			continue
		}
		switch state.status {
		case "pass":
			r.totalPassed++
		case "fail":
			r.totalFailed++
			r.failedChecks = append(r.failedChecks, name)
			// Track checks with retry-clean enabled
			check := r.getCheck(name)
			if check != nil && check.RetryClean {
				r.retryCleanChecks = append(r.retryCleanChecks, name)
			}
		}
	}
	r.mu.Unlock()
}

// depsReady returns true if all dependencies of a check have passed.
func (r *Runner) depsReady(name string, checkNames []string) bool {
	check := r.getCheck(name)
	if check == nil || len(check.Depends) == 0 {
		return true
	}

	// Build set of checks in this phase
	inPhase := make(map[string]bool)
	for _, n := range checkNames {
		inPhase[n] = true
	}

	// Check each dependency
	for _, dep := range check.Depends {
		if !inPhase[dep] {
			continue // Dependency not in this phase, ignore
		}
		state := r.states[dep]
		if state == nil || state.status != "pass" {
			return false
		}
	}
	return true
}

// depsFailed returns true if any dependency of a check has failed.
func (r *Runner) depsFailed(name string, checkNames []string) bool {
	check := r.getCheck(name)
	if check == nil || len(check.Depends) == 0 {
		return false
	}

	// Build set of checks in this phase
	inPhase := make(map[string]bool)
	for _, n := range checkNames {
		inPhase[n] = true
	}

	// Check each dependency
	for _, dep := range check.Depends {
		if !inPhase[dep] {
			continue // Dependency not in this phase, ignore
		}
		state := r.states[dep]
		if state != nil && state.status == "fail" {
			return true
		}
	}
	return false
}

// runCheckParallel executes a single check and updates its spinner.
func (r *Runner) runCheckParallel(name string) {
	check := r.getCheck(name)
	if check == nil {
		return
	}

	r.mu.Lock()
	state := r.states[name]
	state.status = "running"
	spinner := state.spinner
	r.mu.Unlock()

	startTime := time.Now()
	cmd := exec.Command("sh", "-c", check.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	result := &CheckResult{
		Name:     name,
		Success:  err == nil,
		Output:   output,
		Duration: duration,
	}

	r.mu.Lock()
	r.results[name] = result
	if result.Success {
		state.status = "pass"
	} else {
		state.status = "fail"
	}
	r.mu.Unlock()

	// Update spinner with result
	// Note: spinner.Complete() and spinner.Error() add ✓/✗ prefix automatically
	timeStr := fmt.Sprintf("%s%.1fs%s", Dim, duration.Seconds(), Reset)
	if result.Success {
		spinner.UpdateMessage(fmt.Sprintf("%s %s", name, timeStr))
		spinner.Complete()
	} else if check.RetryClean {
		// Show retry indicator for checks that will retry with clean build
		spinner.CompleteCharacter(fmt.Sprintf("%s↻%s", Yellow, Reset))
		spinner.UpdateMessage(fmt.Sprintf("%s %s %s(will retry with clean build)%s", name, timeStr, Yellow, Reset))
		spinner.Complete()
	} else {
		spinner.UpdateMessage(fmt.Sprintf("%s %s", name, timeStr))
		spinner.Error()
	}
}

// Ensure io package is used (for potential future extensions)
var _ io.Reader
