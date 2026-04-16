package executor

import (
	"fmt"
	"os/exec"
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

// CheckResult records the outcome of a single execution item.
type CheckResult struct {
	ItemID   string
	Command  string
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

// Executor runs execution items.
type Executor struct {
	rootDir string
	dryRun  bool
}

// New creates an Executor.
func New(rootDir string, dryRun bool) *Executor {
	return &Executor{rootDir: rootDir, dryRun: dryRun}
}

// Run executes items using the parallel schedule.
func (e *Executor) Run(items []selector.ExecutionItem, cat *catalog.Catalog) *RunResult {
	result := &RunResult{}
	schedule := selector.ComputeSchedule(items, 0)

	failed := make(map[string]bool)
	var mu sync.Mutex
	wallStart := time.Now()

	// Build prerequisite lookup
	prereqsOf := make(map[string][]string)
	for _, item := range items {
		prereqsOf[item.ID] = item.Prerequisites
	}

	// Build item lookup
	itemByID := make(map[string]*selector.ExecutionItem)
	for i := range items {
		itemByID[items[i].ID] = &items[i]
	}

	for _, layer := range schedule.Layers {
		var wg sync.WaitGroup
		layerResults := make([]CheckResult, len(layer.ItemIDs))

		for i, itemID := range layer.ItemIDs {
			item := itemByID[itemID]
			if item == nil {
				continue
			}

			// Check if any prerequisite failed
			mu.Lock()
			skip := false
			for _, prereq := range prereqsOf[itemID] {
				if failed[prereq] {
					skip = true
					break
				}
			}
			mu.Unlock()

			if skip {
				layerResults[i] = CheckResult{ItemID: itemID, Status: StatusSkipped}
				continue
			}

			// Resolve command
			ct := cat.ByID(item.CheckTypeID)
			if ct == nil {
				layerResults[i] = CheckResult{
					ItemID: itemID,
					Status: StatusError,
					Output: fmt.Sprintf("check type %q not found in catalog", item.CheckTypeID),
				}
				mu.Lock()
				failed[itemID] = true
				mu.Unlock()
				continue
			}

			command := item.ResolvedCommandWithCatalog(ct, cat)

			if e.dryRun {
				layerResults[i] = CheckResult{
					ItemID:  itemID,
					Command: command,
					Status:  StatusPassed,
					Output:  fmt.Sprintf("[dry-run] %s", command),
				}
				continue
			}

			wg.Add(1)
			go func(idx int, id, cmd string) {
				defer wg.Done()
				cr := e.execute(id, cmd)
				layerResults[idx] = cr
				if cr.Status == StatusFailed || cr.Status == StatusError {
					mu.Lock()
					failed[id] = true
					mu.Unlock()
				}
			}(i, itemID, command)
		}

		wg.Wait()

		for _, cr := range layerResults {
			if cr.ItemID == "" {
				continue
			}
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

func (e *Executor) execute(itemID, command string) CheckResult {
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
		ItemID:   itemID,
		Command:  command,
		Status:   status,
		Duration: duration,
		Output:   output,
	}
}
