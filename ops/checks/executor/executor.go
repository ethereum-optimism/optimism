package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

// Status represents the outcome of a check execution.
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
	StatusError   Status = "error"
)

// CheckResult records the outcome of a single check.
type CheckResult struct {
	CheckID  string
	Status   Status
	Duration time.Duration
	Output   string
}

// RunResult is the aggregate outcome.
type RunResult struct {
	Results   []CheckResult
	WallClock time.Duration
	Passed    int
	Failed    int
	Skipped   int
}

// Executor runs checks.
type Executor struct {
	rootDir string
	dryRun  bool
}

// New creates an Executor.
func New(rootDir string, dryRun bool) *Executor {
	return &Executor{rootDir: rootDir, dryRun: dryRun}
}

// Run executes selected checks using the parallel schedule.
// Each layer runs its checks in parallel; layers run sequentially.
// If any check in a layer fails, dependents in later layers are skipped.
func (e *Executor) Run(selections []selector.Selection, cat *catalog.Catalog) *RunResult {
	result := &RunResult{}
	schedule := selector.ComputeSchedule(selections, 0)

	failed := make(map[string]bool)
	var mu sync.Mutex
	wallStart := time.Now()

	// Build prerequisite lookup
	prereqsOf := make(map[string][]string)
	for _, sel := range selections {
		prereqsOf[sel.CheckID] = sel.Prerequisites
	}

	for _, layer := range schedule.Layers {
		var wg sync.WaitGroup
		layerResults := make([]CheckResult, len(layer.Checks))

		for i, checkID := range layer.Checks {
			// Check if any prerequisite failed
			mu.Lock()
			skip := false
			for _, prereq := range prereqsOf[checkID] {
				if failed[prereq] {
					skip = true
					break
				}
			}
			mu.Unlock()

			if skip {
				layerResults[i] = CheckResult{
					CheckID: checkID,
					Status:  StatusSkipped,
				}
				continue
			}

			// Look up the command
			rawID := strings.TrimPrefix(checkID, "check:")
			ch := cat.ByID(rawID)
			if ch == nil {
				layerResults[i] = CheckResult{
					CheckID: checkID,
					Status:  StatusError,
					Output:  fmt.Sprintf("check %q not found in catalog", rawID),
				}
				mu.Lock()
				failed[checkID] = true
				mu.Unlock()
				continue
			}

			if e.dryRun {
				layerResults[i] = CheckResult{
					CheckID: checkID,
					Status:  StatusPassed,
					Output:  fmt.Sprintf("[dry-run] would execute: %s", ch.Command),
				}
				continue
			}

			// Execute in parallel
			wg.Add(1)
			go func(idx int, cID, command string) {
				defer wg.Done()
				cr := e.execute(cID, command)
				layerResults[idx] = cr
				if cr.Status == StatusFailed || cr.Status == StatusError {
					mu.Lock()
					failed[cID] = true
					mu.Unlock()
				}
			}(i, checkID, ch.Command)
		}

		wg.Wait()

		// Collect results
		for _, cr := range layerResults {
			result.Results = append(result.Results, cr)
			switch cr.Status {
			case StatusPassed:
				result.Passed++
			case StatusFailed, StatusError:
				result.Failed++
			case StatusSkipped:
				result.Skipped++
			}
		}
	}

	result.WallClock = time.Since(wallStart)
	return result
}

func (e *Executor) execute(checkID, command string) CheckResult {
	start := time.Now()

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = e.rootDir

	out, err := cmd.CombinedOutput()
	duration := time.Since(start)

	output := string(out)
	const maxOutput = 4096
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... (truncated)"
	}

	status := StatusPassed
	if err != nil {
		status = StatusFailed
	}

	return CheckResult{
		CheckID:  checkID,
		Status:   status,
		Duration: duration,
		Output:   output,
	}
}
