package executor

import (
	"bytes"
	"fmt"
	"os"
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
	Output   string // full combined stdout+stderr; no truncation
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

// Run executes items using the parallel schedule. Each layer runs its
// items concurrently; layers are strict barriers. A previous failure
// propagates via item.Prerequisites: any downstream item whose prereq
// failed is marked StatusSkipped and not executed.
//
// Per-item progress is printed to stderr as checks start and finish —
// passing checks collapse to a single result line; failing checks get
// their full output reproduced by the caller (see cmd/checks/run.go).
func (e *Executor) Run(items []selector.ExecutionItem, cat *catalog.Catalog) *RunResult {
	result := &RunResult{}
	schedule := selector.ComputeSchedule(items, 0)

	failed := make(map[string]bool)
	var mu sync.Mutex
	wallStart := time.Now()

	prereqsOf := make(map[string][]string, len(items))
	itemByID := make(map[string]*selector.ExecutionItem, len(items))
	for i := range items {
		it := &items[i]
		prereqsOf[it.ID] = it.Prerequisites
		itemByID[it.ID] = it
	}

	printMu := sync.Mutex{}
	println := func(format string, a ...any) {
		printMu.Lock()
		defer printMu.Unlock()
		fmt.Fprintf(os.Stderr, format, a...)
	}

	for layerIdx, layer := range schedule.Layers {
		var wg sync.WaitGroup
		layerResults := make([]CheckResult, len(layer.ItemIDs))

		for i, itemID := range layer.ItemIDs {
			item := itemByID[itemID]
			if item == nil {
				continue
			}

			// Skip if any prerequisite failed.
			mu.Lock()
			skipReason := ""
			for _, prereq := range prereqsOf[itemID] {
				if failed[prereq] {
					skipReason = prereq
					break
				}
			}
			mu.Unlock()
			if skipReason != "" {
				layerResults[i] = CheckResult{
					ItemID: itemID,
					Status: StatusSkipped,
					Output: fmt.Sprintf("skipped: prerequisite %q failed", skipReason),
				}
				println("  → %s  (skipped: prereq %s failed)\n", itemID, skipReason)
				continue
			}

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
				println("  • %s  [dry-run] %s\n", itemID, truncateRight(command, 100))
				continue
			}

			println("  ▶ %s  (layer %d, starting)\n", itemID, layerIdx+1)

			wg.Add(1)
			go func(idx int, id, cmd string) {
				defer wg.Done()
				cr := e.execute(id, cmd)
				layerResults[idx] = cr
				if cr.Status == StatusFailed || cr.Status == StatusError {
					mu.Lock()
					failed[id] = true
					mu.Unlock()
					println("  ✗ %s  (%s) FAILED\n", id, cr.Duration.Round(100*time.Millisecond))
				} else {
					println("  ✓ %s  (%s)\n", id, cr.Duration.Round(100*time.Millisecond))
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
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	duration := time.Since(start)

	status := StatusPassed
	if err != nil {
		status = StatusFailed
	}
	return CheckResult{
		ItemID:   itemID,
		Command:  command,
		Status:   status,
		Duration: duration,
		Output:   buf.String(),
	}
}

func truncateRight(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
