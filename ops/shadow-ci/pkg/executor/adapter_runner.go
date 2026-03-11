package executor

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// AdapterRunner implements Runner by delegating to language-native test adapters
// via engine.TestExecutor. Produces per-test TestResult with classification.
type AdapterRunner struct {
	executor    *engine.TestExecutor
	lastResults []model.TestResult
}

// NewAdapterRunner creates an AdapterRunner from an adapter registry.
func NewAdapterRunner(registry *adapters.Registry, emitter *events.Emitter) *AdapterRunner {
	classifier := engine.NewClassifier()
	fingerprinter := engine.NewFingerprinter()
	return &AdapterRunner{
		executor: engine.NewTestExecutor(registry, classifier, fingerprinter, emitter),
	}
}

// Run executes tests via the language adapter, producing per-test results.
func (a *AdapterRunner) Run(ctx RunContext) error {
	job := buildPlannedJob(ctx)

	result, err := a.executor.Execute(job)
	if err != nil {
		a.lastResults = nil
		return err
	}

	a.lastResults = result.Results

	// Write summary to log file for compatibility with shell runner output.
	if ctx.LogPath != "" {
		writeSummaryLog(ctx.LogPath, result)
	}

	// Check for real failures.
	realFailures := countRealFailures(result.Results)
	if realFailures > 0 {
		return fmt.Errorf("%d real test failure(s)", realFailures)
	}
	return nil
}

// Results returns test results from the last Run call.
func (a *AdapterRunner) Results() []model.TestResult {
	return a.lastResults
}

// buildPlannedJob converts RunContext into a model.PlannedJob for the TestExecutor.
func buildPlannedJob(ctx RunContext) model.PlannedJob {
	job := model.PlannedJob{
		Name:     ctx.Category,
		Language: ctx.Cat.Language,
		Resources: model.Resources{
			Parallelism: runtime.NumCPU(),
			Runner:      "large",
			Timeout:     20 * time.Minute,
		},
		SelectionReason: "executor-dispatch",
	}

	if ctx.Cat.RunnerClass != "" {
		job.Resources.Runner = ctx.Cat.RunnerClass
	}

	// Build targets from decision.
	if ctx.Decision != nil && len(ctx.Decision.Targets) > 0 {
		for _, t := range ctx.Decision.Targets {
			job.Targets = append(job.Targets, model.Target{
				ID:       t,
				Language: ctx.Cat.Language,
				Scope:    model.ScopeAffected,
			})
		}
	} else {
		// No targets (force-all or non-graph): single catch-all target.
		job.Targets = []model.Target{{
			ID:       ctx.Category,
			Language: ctx.Cat.Language,
			Scope:    model.ScopeAlways,
		}}
	}

	// Build configurations from features (sol matrix) or default.
	if ctx.Decision != nil && len(ctx.Decision.Features) > 0 {
		for _, f := range ctx.Decision.Features {
			job.Configurations = append(job.Configurations, model.Configuration{
				Name: f,
				Env:  map[string]string{},
			})
		}
	} else {
		job.Configurations = []model.Configuration{{
			Name: "default",
			Env:  map[string]string{},
		}}
	}

	return job
}

// writeSummaryLog writes a human-readable summary to logPath for compatibility.
func writeSummaryLog(logPath string, result *engine.JobResult) {
	f, err := os.Create(logPath)
	if err != nil {
		return
	}
	defer f.Close()

	passed, failed, skipped, flakes := 0, 0, 0, 0
	for _, r := range result.Results {
		switch r.Status {
		case model.StatusPass:
			passed++
		case model.StatusFail:
			failed++
		case model.StatusSkip:
			skipped++
		}
		if r.Classification == model.Flake {
			flakes++
		}
	}

	fmt.Fprintf(f, "=== Test Results ===\n")
	fmt.Fprintf(f, "Total: %d, Passed: %d, Failed: %d, Skipped: %d, Flakes: %d\n\n",
		len(result.Results), passed, failed, skipped, flakes)

	// Print failure details.
	var failures []string
	for _, r := range result.Results {
		if r.Classification == model.RealFailure {
			failures = append(failures, fmt.Sprintf("FAIL: %s/%s (%s)\n%s",
				r.Test.Package, r.Test.Name, r.Duration, r.Output))
		}
	}
	if len(failures) > 0 {
		fmt.Fprintf(f, "=== Failures ===\n")
		fmt.Fprint(f, strings.Join(failures, "\n---\n"))
	}
}

// countRealFailures counts test results classified as real failures.
func countRealFailures(results []model.TestResult) int {
	n := 0
	for _, r := range results {
		if r.Classification == model.RealFailure {
			n++
		}
	}
	return n
}
