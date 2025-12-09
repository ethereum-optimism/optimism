package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chelnak/ysmrr"
	"github.com/chelnak/ysmrr/pkg/colors"
	"gopkg.in/yaml.v3"
)

// Build types
const (
	BuildNone   = "none"
	BuildSource = "source"
	BuildDev    = "dev"
)

// Cache settings
const (
	CacheDir      = "/tmp/check-runner-cache"
	MaxCacheCount = 5
)

// Artifact directories to cache/restore
var artifactDirs = []string{"artifacts", "forge-artifacts", "cache"}

// Check represents a single check configuration
type Check struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Command     string   `yaml:"command"`
	Build       string   `yaml:"build"`   // none, source, dev
	Depends     []string `yaml:"depends"` // checks that must complete first
}

// Config represents the check runner configuration
type Config struct {
	Checks []Check `yaml:"checks"`
}

// CheckResult holds the result of a check execution
type CheckResult struct {
	Name     string
	Success  bool
	Output   string
	Duration time.Duration
}

// checkState tracks state during execution
type checkState struct {
	status  string // "pending", "queued", "running", "pass", "fail", "skipped"
	spinner *ysmrr.Spinner
}

// Runner orchestrates check execution
type Runner struct {
	config     *Config
	results    map[string]*CheckResult
	states     map[string]*checkState
	mu         sync.Mutex
	noBuild    bool
	verbose    bool
	noCache    bool
	clean      bool
	tempDir    string
	sourceHash string
}

func main() {
	var (
		configPath string
		listChecks bool
		runCheck   string
		noBuild    bool
		verbose    bool
		noCache    bool
		clean      bool
	)

	flag.StringVar(&configPath, "config", "", "Path to checks.yaml config file")
	flag.BoolVar(&listChecks, "list", false, "List available checks")
	flag.StringVar(&runCheck, "run", "", "Run specific check(s), comma-separated")
	flag.BoolVar(&noBuild, "no-build", false, "Skip builds and checks that require them")
	flag.BoolVar(&verbose, "verbose", false, "Show output for all checks, not just failures")
	flag.BoolVar(&noCache, "no-cache", false, "Disable build caching")
	flag.BoolVar(&clean, "clean", false, "Clean build artifacts before each build (forces fresh compilation)")
	flag.Parse()

	// Find config file
	if configPath == "" {
		candidates := []string{
			"scripts/check-runner/checks.yaml",
			"checks.yaml",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				configPath = c
				break
			}
		}
		if configPath == "" {
			fmt.Fprintf(os.Stderr, "Error: could not find checks.yaml config file\n")
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
		config:  config,
		results: make(map[string]*CheckResult),
		states:  make(map[string]*checkState),
		noBuild: noBuild,
		verbose: verbose,
		noCache: noCache,
		clean:   clean,
	}

	// Determine which checks to run
	var checksToRun []string
	if runCheck != "" {
		checksToRun = strings.Split(runCheck, ",")
		for i, name := range checksToRun {
			checksToRun[i] = strings.TrimSpace(name)
		}
		// Validate check names
		for _, name := range checksToRun {
			if !runner.checkExists(name) {
				fmt.Fprintf(os.Stderr, "Error: unknown check '%s'\n", name)
				fmt.Fprintf(os.Stderr, "Run with -list to see available checks\n")
				os.Exit(1)
			}
		}
	} else {
		for _, c := range config.Checks {
			checksToRun = append(checksToRun, c.Name)
		}
	}

	success := runner.Run(checksToRun)
	if !success {
		os.Exit(1)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Default build type to "none"
	for i := range config.Checks {
		if config.Checks[i].Build == "" {
			config.Checks[i].Build = BuildNone
		}
	}

	return &config, nil
}

func printCheckList(config *Config) {
	fmt.Println("Available checks:")
	fmt.Println()
	maxNameLen := 0
	for _, c := range config.Checks {
		if len(c.Name) > maxNameLen {
			maxNameLen = len(c.Name)
		}
	}
	for _, c := range config.Checks {
		info := ""
		if c.Build != "" && c.Build != BuildNone {
			info += fmt.Sprintf(" [build: %s]", c.Build)
		}
		if len(c.Depends) > 0 {
			info += fmt.Sprintf(" [depends: %s]", strings.Join(c.Depends, ", "))
		}
		fmt.Printf("  %-*s  %s%s\n", maxNameLen, c.Name, c.Description, info)
	}
}

func (r *Runner) checkExists(name string) bool {
	for _, c := range r.config.Checks {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (r *Runner) getCheck(name string) *Check {
	for i := range r.config.Checks {
		if r.config.Checks[i].Name == name {
			return &r.config.Checks[i]
		}
	}
	return nil
}

// groupByBuild groups checks by their build requirement
func (r *Runner) groupByBuild(checkNames []string) map[string][]string {
	groups := make(map[string][]string)
	for _, name := range checkNames {
		check := r.getCheck(name)
		if check == nil {
			continue
		}
		build := check.Build
		if build == "" {
			build = BuildNone
		}
		groups[build] = append(groups[build], name)
	}
	return groups
}

// ============================================================================
// Build Cache Functions
// ============================================================================

// computeSourceHash computes SHA256 hash of all .sol files and foundry.toml
func computeSourceHash() (string, error) {
	h := sha256.New()

	// Collect all files to hash
	var files []string

	// Add all .sol files
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden dirs and common non-source dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "forge-artifacts" || base == "artifacts" || base == "cache" || base == "out" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".sol") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	// Add foundry.toml
	if _, err := os.Stat("foundry.toml"); err == nil {
		files = append(files, "foundry.toml")
	}

	// Sort for determinism
	sort.Strings(files)

	// Hash each file
	for _, path := range files {
		// Include path in hash
		h.Write([]byte(path))

		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nil // Use first 16 chars
}

// getCachePath returns the cache directory for a build type and hash
func getCachePath(buildType, hash string) string {
	return filepath.Join(CacheDir, buildType, hash)
}

// cacheExists checks if a cache entry exists
func cacheExists(buildType, hash string) bool {
	path := getCachePath(buildType, hash)
	_, err := os.Stat(path)
	return err == nil
}

// getLatestCachePath returns the path to the "latest" cache for a build type
func getLatestCachePath(buildType string) string {
	return filepath.Join(CacheDir, buildType, "latest")
}

// getLatestCache resolves the "latest" symlink and returns the hash
func getLatestCache(buildType string) string {
	latestPath := getLatestCachePath(buildType)
	target, err := os.Readlink(latestPath)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

// restoreFromCache restores artifacts from cache to working directory
func restoreFromCache(buildType, hash string) error {
	cachePath := getCachePath(buildType, hash)
	if _, err := os.Stat(cachePath); err != nil {
		return fmt.Errorf("cache not found: %s", cachePath)
	}

	for _, dir := range artifactDirs {
		src := filepath.Join(cachePath, dir)
		if _, err := os.Stat(src); err == nil {
			// Remove existing and copy from cache
			os.RemoveAll(dir)
			if err := copyDir(src, dir); err != nil {
				return fmt.Errorf("failed to restore %s: %w", dir, err)
			}
		}
	}
	return nil
}

// saveToCache saves current artifacts to cache
func saveToCache(buildType, hash string) error {
	cachePath := getCachePath(buildType, hash)

	// Create cache directory
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return err
	}

	// Copy each artifact directory
	for _, dir := range artifactDirs {
		if _, err := os.Stat(dir); err == nil {
			dst := filepath.Join(cachePath, dir)
			if err := copyDir(dir, dst); err != nil {
				return fmt.Errorf("failed to cache %s: %w", dir, err)
			}
		}
	}

	// Update "latest" symlink
	latestPath := getLatestCachePath(buildType)
	os.Remove(latestPath) // Remove old symlink
	if err := os.Symlink(hash, latestPath); err != nil {
		// Non-fatal
		fmt.Fprintf(os.Stderr, "Warning: could not update latest symlink: %v\n", err)
	}

	// Evict old caches
	evictOldCaches(buildType)

	return nil
}

// evictOldCaches keeps only the most recent MaxCacheCount caches
func evictOldCaches(buildType string) {
	cacheTypeDir := filepath.Join(CacheDir, buildType)
	entries, err := os.ReadDir(cacheTypeDir)
	if err != nil {
		return
	}

	// Collect cache entries (excluding "latest" symlink)
	type cacheEntry struct {
		name    string
		modTime time.Time
	}
	var caches []cacheEntry

	for _, entry := range entries {
		if entry.Name() == "latest" {
			continue
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

	// Sort by mod time (newest first)
	sort.Slice(caches, func(i, j int) bool {
		return caches[i].modTime.After(caches[j].modTime)
	})

	// Remove oldest entries beyond MaxCacheCount
	for i := MaxCacheCount; i < len(caches); i++ {
		path := filepath.Join(cacheTypeDir, caches[i].name)
		os.RemoveAll(path)
	}
}

// ============================================================================
// Main Run Logic
// ============================================================================

func (r *Runner) Run(checkNames []string) bool {
	// Group checks by build type
	groups := r.groupByBuild(checkNames)

	// Determine build order
	buildOrder := []string{BuildNone, BuildSource, BuildDev}

	// Check if we need to do any builds
	needsBuild := false
	for _, build := range []string{BuildSource, BuildDev} {
		if len(groups[build]) > 0 && !r.noBuild {
			needsBuild = true
			break
		}
	}

	// Save current artifacts (to restore after all checks)
	if needsBuild {
		if err := r.saveWorkingArtifacts(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save artifacts: %v\n", err)
		}
		defer r.restoreWorkingArtifacts()
	}

	// Run checks in phases
	allSuccess := true
	hashComputed := false
	for _, build := range buildOrder {
		checks := groups[build]
		if len(checks) == 0 {
			continue
		}

		// Skip build-requiring checks if -no-build
		if r.noBuild && build != BuildNone {
			fmt.Printf("Skipping %d checks requiring build-%s (--no-build)\n", len(checks), build)
			continue
		}

		// Compute source hash after "none" phase (after lint-fix may have changed files)
		// but before the first build
		if build != BuildNone && !hashComputed && !r.noCache {
			hash, err := computeSourceHash()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not compute source hash: %v\n", err)
			} else {
				r.sourceHash = hash
			}
			hashComputed = true
		}

		// Do the build if needed (with caching)
		if build != BuildNone {
			if err := r.doBuildWithCache(build); err != nil {
				fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
				return false
			}
		}

		// Run checks for this phase
		if build == BuildNone {
			fmt.Println("▶ Running checks (no build required)...")
		} else {
			fmt.Printf("▶ Running checks (build: %s)...\n", build)
		}

		if !r.runChecks(checks) {
			allSuccess = false
		}
	}

	return allSuccess
}

func (r *Runner) doBuildWithCache(buildType string) error {
	hash := r.sourceHash
	cacheHit := false

	// If --clean flag is set, remove all artifact directories before building
	if r.clean {
		fmt.Printf("▶ Cleaning artifacts for fresh %s build...\n", buildType)
		for _, dir := range artifactDirs {
			os.RemoveAll(dir)
		}
	} else {
		// Check for exact cache hit (only if not doing clean build)
		if hash != "" && !r.noCache && cacheExists(buildType, hash) {
			fmt.Printf("▶ Restoring %s build from cache (%s)...\n", buildType, hash)
			if err := restoreFromCache(buildType, hash); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cache restore failed: %v\n", err)
			} else {
				cacheHit = true
			}
		}

		// No exact match - try to restore "latest" for incremental build
		if !cacheHit && hash != "" && !r.noCache {
			latest := getLatestCache(buildType)
			if latest != "" && latest != hash {
				fmt.Printf("▶ Restoring latest %s cache for incremental build...\n", buildType)
				if err := restoreFromCache(buildType, latest); err != nil {
					// Non-fatal, will just do full build
					fmt.Fprintf(os.Stderr, "Warning: could not restore latest cache: %v\n", err)
				}
			}
		}
	}

	// Always run the build command after restoring cache (or cleaning).
	// If cache is accurate, forge will report "No files changed" and return quickly.
	// If cache is stale or cleaned, this ensures we get a correct build.
	fmt.Printf("▶ Building (%s)...\n", buildType)
	var cmd *exec.Cmd
	switch buildType {
	case BuildSource:
		cmd = exec.Command("just", "build-source")
	case BuildDev:
		cmd = exec.Command("just", "forge-build-dev")
	default:
		return fmt.Errorf("unknown build type: %s", buildType)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Save to cache (if we didn't have an exact cache hit)
	if !cacheHit && hash != "" && !r.noCache {
		fmt.Printf("▶ Saving %s build to cache (%s)...\n", buildType, hash)
		if err := saveToCache(buildType, hash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save to cache: %v\n", err)
		}
	}

	return nil
}

// saveWorkingArtifacts saves current artifacts to temp dir
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

// restoreWorkingArtifacts restores artifacts from temp dir
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

func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-r", src, dst)
	return cmd.Run()
}

// ============================================================================
// Check Execution
// ============================================================================

// depsReady returns true if all dependencies of a check are satisfied (completed successfully)
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

	for _, dep := range check.Depends {
		// Only check deps that are in this phase
		if !inPhase[dep] {
			continue
		}
		state := r.states[dep]
		if state == nil || state.status != "pass" {
			return false
		}
	}
	return true
}

// depsFailed returns true if any dependency failed (so this check should be skipped)
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

	for _, dep := range check.Depends {
		if !inPhase[dep] {
			continue
		}
		state := r.states[dep]
		if state != nil && state.status == "fail" {
			return true
		}
	}
	return false
}

func (r *Runner) runChecks(checkNames []string) bool {
	// Create spinner manager
	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)

	// Initialize spinners for each check
	for _, name := range checkNames {
		spinner := sm.AddSpinner(name)
		r.states[name] = &checkState{status: "pending", spinner: spinner}
	}

	// Start spinner manager
	sm.Start()

	// Channel for ready checks
	ready := make(chan string, len(checkNames))
	var wg sync.WaitGroup

	// Queue checks with no dependencies initially
	r.mu.Lock()
	for _, name := range checkNames {
		if r.depsReady(name, checkNames) {
			r.states[name].status = "queued"
			ready <- name
		}
	}
	r.mu.Unlock()

	// Worker pool
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range ready {
				r.runCheck(name)
			}
		}()
	}

	// Scheduler: watch for completed checks and queue newly-ready ones
	go func() {
		for {
			r.mu.Lock()
			allDone := true
			for _, name := range checkNames {
				state := r.states[name]
				if state.status == "pending" {
					// Check if deps failed -> skip this check
					if r.depsFailed(name, checkNames) {
						state.status = "skipped"
						state.spinner.UpdateMessage(name + " (skipped - dependency failed)")
						state.spinner.Error()
						continue
					}
					// Check if deps are now ready
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

	return r.printResults(checkNames)
}

func (r *Runner) runCheck(name string) {
	check := r.getCheck(name)
	if check == nil {
		return
	}

	r.mu.Lock()
	state := r.states[name]
	state.status = "running"
	spinner := state.spinner
	r.mu.Unlock()

	spinner.UpdateMessage(name + " (running)")

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

	if result.Success {
		spinner.UpdateMessage(fmt.Sprintf("%s (%.1fs)", name, duration.Seconds()))
		spinner.Complete()
	} else {
		spinner.UpdateMessage(fmt.Sprintf("%s (%.1fs)", name, duration.Seconds()))
		spinner.Error()
	}
}

func (r *Runner) printResults(checkNames []string) bool {
	fmt.Println()

	hasFailures := false
	passCount := 0
	failCount := 0

	r.mu.Lock()
	for _, name := range checkNames {
		state := r.states[name]
		if state == nil {
			continue
		}
		switch state.status {
		case "pass":
			passCount++
		case "fail":
			failCount++
			hasFailures = true
		}
	}

	// Print verbose or failure output
	for _, name := range checkNames {
		result := r.results[name]
		if result == nil {
			continue
		}

		if r.verbose || !result.Success {
			if result.Output != "" {
				fmt.Printf("\033[1m=== %s ===\033[0m\n", name)
				fmt.Println(strings.TrimSpace(result.Output))
				fmt.Println()
			}
		}
	}
	r.mu.Unlock()

	fmt.Printf("\033[1mSummary:\033[0m %d passed", passCount)
	if failCount > 0 {
		fmt.Printf(", \033[31m%d failed\033[0m", failCount)
	}
	fmt.Println()

	return !hasFailures
}

// Unused but keeping for interface compatibility
var _ io.Reader // silence unused import warning if needed
