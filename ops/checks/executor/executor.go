package executor

import (
	"fmt"
	"os/exec"
	"strings"
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
	TotalTime time.Duration
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

// Run executes the selected checks in prerequisite order.
func (e *Executor) Run(selections []selector.Selection, cat *catalog.Catalog) *RunResult {
	result := &RunResult{}

	// Build execution order: prerequisites first, then checks
	// Use topological sort based on prerequisites
	ordered := topologicalSort(selections)

	failed := make(map[string]bool)

	for _, sel := range ordered {
		// Skip if any prerequisite failed
		skip := false
		for _, prereq := range sel.Prerequisites {
			if failed[prereq] {
				skip = true
				break
			}
		}

		if skip {
			result.Results = append(result.Results, CheckResult{
				CheckID: sel.CheckID,
				Status:  StatusSkipped,
			})
			result.Skipped++
			continue
		}

		// Look up the command
		checkID := strings.TrimPrefix(sel.CheckID, "check:")
		ch := cat.ByID(checkID)
		if ch == nil {
			result.Results = append(result.Results, CheckResult{
				CheckID: sel.CheckID,
				Status:  StatusError,
				Output:  fmt.Sprintf("check %q not found in catalog", checkID),
			})
			result.Failed++
			failed[sel.CheckID] = true
			continue
		}

		if e.dryRun {
			result.Results = append(result.Results, CheckResult{
				CheckID: sel.CheckID,
				Status:  StatusPassed,
				Output:  fmt.Sprintf("[dry-run] would execute: %s", ch.Command),
			})
			result.Passed++
			continue
		}

		// Execute
		cr := e.execute(sel.CheckID, ch.Command)
		result.Results = append(result.Results, cr)
		result.TotalTime += cr.Duration

		switch cr.Status {
		case StatusPassed:
			result.Passed++
		case StatusFailed:
			result.Failed++
			failed[sel.CheckID] = true
		case StatusError:
			result.Failed++
			failed[sel.CheckID] = true
		}
	}

	return result
}

func (e *Executor) execute(checkID, command string) CheckResult {
	start := time.Now()

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = e.rootDir

	out, err := cmd.CombinedOutput()
	duration := time.Since(start)

	output := string(out)
	// Truncate long output
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

// topologicalSort orders selections so prerequisites come before dependents.
func topologicalSort(selections []selector.Selection) []selector.Selection {
	// Build dependency map
	byID := make(map[string]*selector.Selection)
	for i := range selections {
		byID[selections[i].CheckID] = &selections[i]
	}

	visited := make(map[string]bool)
	var ordered []selector.Selection

	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		sel, ok := byID[id]
		if !ok {
			return
		}

		for _, prereq := range sel.Prerequisites {
			visit(prereq)
		}
		ordered = append(ordered, *sel)
	}

	for _, sel := range selections {
		visit(sel.CheckID)
	}

	return ordered
}
