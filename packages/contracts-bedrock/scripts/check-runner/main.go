package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// ANSI color codes
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
	Depends     []string `yaml:"depends"`
}

// Phase represents a phase of checks
type Phase struct {
	Name     string  `yaml:"name"`
	Build    string  `yaml:"build"`    // optional build command
	Parallel *bool   `yaml:"parallel"` // default true
	Checks   []Check `yaml:"checks"`
}

// Config represents the check runner configuration
type Config struct {
	Phases []Phase `yaml:"phases"`
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
	config       *Config
	results      map[string]*CheckResult
	states       map[string]*checkState
	mu           sync.Mutex
	noBuild      bool
	verbose      bool
	noCache      bool
	clean        bool
	tempDir      string
	sourceHash   string
	totalPassed  int
	totalFailed  int
	totalSkipped int
	failedChecks []string
	buildError   string // Build failure output to show at end

	// Signal handling
	interrupted atomic.Bool
	sigCount    atomic.Int32
	cancelFunc  context.CancelFunc
}

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

// ============================================================================
// Build Cache Functions
// ============================================================================

func computeSourceHash() (string, error) {
	h := sha256.New()

	var files []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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

	if _, err := os.Stat("foundry.toml"); err == nil {
		files = append(files, "foundry.toml")
	}

	sort.Strings(files)

	for _, path := range files {
		h.Write([]byte(path))
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func getCachePath(phaseName, hash string) string {
	return filepath.Join(CacheDir, phaseName, hash)
}

func cacheExists(phaseName, hash string) bool {
	path := getCachePath(phaseName, hash)
	_, err := os.Stat(path)
	return err == nil
}

func getLatestCachePath(phaseName string) string {
	return filepath.Join(CacheDir, phaseName, "latest")
}

func getLatestCache(phaseName string) string {
	latestPath := getLatestCachePath(phaseName)
	target, err := os.Readlink(latestPath)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

func restoreFromCache(phaseName, hash string) error {
	cachePath := getCachePath(phaseName, hash)
	if _, err := os.Stat(cachePath); err != nil {
		return fmt.Errorf("cache not found: %s", cachePath)
	}

	for _, dir := range artifactDirs {
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

func saveToCache(phaseName, hash string) error {
	cachePath := getCachePath(phaseName, hash)

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return err
	}

	for _, dir := range artifactDirs {
		if _, err := os.Stat(dir); err == nil {
			dst := filepath.Join(cachePath, dir)
			if err := copyDir(dir, dst); err != nil {
				return fmt.Errorf("failed to cache %s: %w", dir, err)
			}
		}
	}

	latestPath := getLatestCachePath(phaseName)
	os.Remove(latestPath)
	if err := os.Symlink(hash, latestPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update latest symlink: %v\n", err)
	}

	evictOldCaches(phaseName)

	return nil
}

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

	sort.Slice(caches, func(i, j int) bool {
		return caches[i].modTime.After(caches[j].modTime)
	})

	for i := MaxCacheCount; i < len(caches); i++ {
		path := filepath.Join(cacheTypeDir, caches[i].name)
		os.RemoveAll(path)
	}
}

// ============================================================================
// Main Run Logic
// ============================================================================

// Run executes the configured checks, optionally filtering to selectedChecks.
func (r *Runner) Run(selectedChecks map[string]bool) bool {
	// Check if any phase has a build
	hasBuilds := false
	for _, phase := range r.config.Phases {
		if phase.Build != "" {
			hasBuilds = true
			break
		}
	}

	// Save current artifacts (to restore after all checks)
	if hasBuilds && !r.noBuild {
		if err := r.saveWorkingArtifacts(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save artifacts: %v\n", err)
		}
		defer r.restoreWorkingArtifacts()
	}

	// Set up signal handling for graceful shutdown
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
	_ = ctx // Used by signal handler

	for _, phase := range r.config.Phases {
		// Check for interruption at start of each phase
		if r.interrupted.Load() {
			break
		}

		// Filter checks for this phase if specific checks requested
		var checksToRun []Check
		for _, c := range phase.Checks {
			if selectedChecks == nil || selectedChecks[c.Name] {
				checksToRun = append(checksToRun, c)
			}
		}

		if len(checksToRun) == 0 {
			continue
		}

		// Skip phases with builds if -no-build
		if phase.Build != "" && r.noBuild {
			fmt.Printf("%s⊘ %s%s %s(skipped - no build)%s\n", Yellow, phase.Name, Reset, Dim, Reset)
			continue
		}

		// Print phase header
		fmt.Printf("\n%s→ %s%s\n", BoldCyan, phase.Name, Reset)

		// Compute source hash before first build phase
		if phase.Build != "" && !hashComputed && !r.noCache {
			hash, err := computeSourceHash()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not compute source hash: %v\n", err)
			} else {
				r.sourceHash = hash
			}
			hashComputed = true
		}

		// Run build if specified
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

		// Run checks
		parallel := phase.Parallel == nil || *phase.Parallel
		r.runPhaseChecks(checksToRun, parallel)
	}

	r.printFinalSummary()
	return r.totalFailed == 0 && !r.interrupted.Load()
}

func (r *Runner) printFinalSummary() {
	fmt.Println()

	// If interrupted, just show that we're done restoring
	if r.interrupted.Load() {
		fmt.Printf("%s✓ Artifacts restored%s\n", Green, Reset)
		return
	}

	// Print build error with box drawing
	if r.buildError != "" {
		fmt.Printf("%s┌─ build%s\n", Red, Reset)
		lines := strings.Split(strings.TrimSpace(r.buildError), "\n")
		for _, line := range lines {
			fmt.Printf("%s│%s %s\n", Red, Reset, line)
		}
		fmt.Printf("%s└%s\n", Red, Reset)
		fmt.Println()
	}

	// Print failed check details with box drawing
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

	// Print final status
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
}

func (r *Runner) doBuildWithCache(phaseName, buildCmd string) error {
	hash := r.sourceHash
	cacheHit := false

	if r.clean {
		fmt.Printf("%s⟳ Cleaning artifacts...%s\n", Dim, Reset)
		for _, dir := range artifactDirs {
			os.RemoveAll(dir)
		}
	} else {
		if hash != "" && !r.noCache && cacheExists(phaseName, hash) {
			fmt.Printf("%s⟳ Restoring from cache %s...%s\n", Dim, hash[:8], Reset)
			if err := restoreFromCache(phaseName, hash); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cache restore failed: %v\n", err)
			} else {
				cacheHit = true
			}
		}

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

	// Run build with spinner
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
		// Store build output for display at end
		output := stdout.String() + stderr.String()
		if output != "" {
			r.buildError = output
		}
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("%sBuilt%s (%s) %s%.1fs%s", Green, Reset, buildCmd, Dim, duration.Seconds(), Reset))
	spinner.Complete()
	sm.Stop()

	if !cacheHit && hash != "" && !r.noCache {
		fmt.Printf("%s⟳ Saving to cache %s...%s\n", Dim, hash[:8], Reset)
		if err := saveToCache(phaseName, hash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save to cache: %v\n", err)
		}
	}

	return nil
}

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

func (r *Runner) runChecksSequential(checks []Check) {
	for _, check := range checks {
		// Check for interruption
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
		} else {
			fmt.Printf("%s✗%s %s %s%.1fs%s\n", Red, Reset, check.Name, Dim, duration.Seconds(), Reset)
			r.totalFailed++
			r.failedChecks = append(r.failedChecks, check.Name)
		}
	}
}

func (r *Runner) runChecksParallel(checks []Check) {
	sm := ysmrr.NewSpinnerManager(
		ysmrr.WithSpinnerColor(colors.FgHiBlue),
	)

	checkNames := make([]string, len(checks))
	for i, c := range checks {
		checkNames[i] = c.Name
		spinner := sm.AddSpinner(c.Name)
		r.states[c.Name] = &checkState{status: "pending", spinner: spinner}
	}

	sm.Start()

	ready := make(chan string, len(checks))
	var wg sync.WaitGroup

	r.mu.Lock()
	for _, name := range checkNames {
		if r.depsReady(name, checkNames) {
			r.states[name].status = "queued"
			ready <- name
		}
	}
	r.mu.Unlock()

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
					if r.depsFailed(name, checkNames) {
						state.status = "skipped"
						state.spinner.UpdateMessage(fmt.Sprintf("  %s %s(skipped)%s", name, Dim, Reset))
						state.spinner.Error()
						r.totalSkipped++
						continue
					}
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

	// Count results
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
		}
	}
	r.mu.Unlock()
}

func (r *Runner) depsReady(name string, checkNames []string) bool {
	check := r.getCheck(name)
	if check == nil || len(check.Depends) == 0 {
		return true
	}

	inPhase := make(map[string]bool)
	for _, n := range checkNames {
		inPhase[n] = true
	}

	for _, dep := range check.Depends {
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

func (r *Runner) depsFailed(name string, checkNames []string) bool {
	check := r.getCheck(name)
	if check == nil || len(check.Depends) == 0 {
		return false
	}

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

	// Format: "✓ name 0.5s" or "✗ name 0.5s"
	// Note: spinner.Complete() and spinner.Error() add ✓/✗ prefix automatically
	timeStr := fmt.Sprintf("%s%.1fs%s", Dim, duration.Seconds(), Reset)
	if result.Success {
		spinner.UpdateMessage(fmt.Sprintf("%s %s", name, timeStr))
		spinner.Complete()
	} else {
		spinner.UpdateMessage(fmt.Sprintf("%s %s", name, timeStr))
		spinner.Error()
	}
}

// Unused but keeping for interface compatibility
var _ io.Reader // silence unused import warning if needed
