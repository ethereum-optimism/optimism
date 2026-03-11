package engine

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// JobResult holds the results of executing a PlannedJob.
type JobResult struct {
	Job     model.PlannedJob `json:"job"`
	Results []model.TestResult `json:"results"`
}

// TestExecutor runs tests from a PlannedJob using the appropriate language adapter.
type TestExecutor struct {
	registry      *adapters.Registry
	classifier    *Classifier
	fingerprinter *Fingerprinter
	emitter       *events.Emitter
}

// NewTestExecutor creates a new TestExecutor.
func NewTestExecutor(registry *adapters.Registry, classifier *Classifier, fingerprinter *Fingerprinter, emitter *events.Emitter) *TestExecutor {
	return &TestExecutor{
		registry:      registry,
		classifier:    classifier,
		fingerprinter: fingerprinter,
		emitter:       emitter,
	}
}

// Execute runs a single PlannedJob and returns classified results.
func (e *TestExecutor) Execute(job model.PlannedJob) (*JobResult, error) {
	adapter, ok := e.registry.Get(job.Language)
	if !ok {
		return nil, fmt.Errorf("no adapter for language %q", job.Language)
	}

	if e.emitter != nil {
		e.emitter.Emit(model.EventJobStarted, map[string]any{
			"job":      job.Name,
			"language": job.Language,
			"targets":  len(job.Targets),
			"configs":  len(job.Configurations),
		})
	}

	var allResults []model.TestResult

	for _, config := range job.Configurations {
		results, err := adapter.Runner.Run(job.Targets, config, adapters.RunOptions{
			Timeout:     int(job.Resources.Timeout.Seconds()),
			Parallelism: job.Resources.Parallelism,
			TestFilter:  job.TestRunFilter,
		})
		if err != nil {
			return nil, fmt.Errorf("running %s tests: %w", job.Language, err)
		}

		for i := range results {
			results[i].Language = job.Language
			results[i].Config = config.Name
		}

		allResults = append(allResults, results...)
	}

	// Classify and retry failures.
	for i := range allResults {
		result := &allResults[i]

		if result.Status == model.StatusFail {
			retryResult, err := adapter.Runner.RunOne(result.Test, job.Configurations[0], adapters.RunOptions{
				Timeout: 60,
			})
			if err != nil {
				result.Classification = model.Infrastructure
				if e.emitter != nil {
					e.emitter.Emit(model.EventInfraFailure, map[string]any{
						"test":   result.Test,
						"error":  err.Error(),
						"config": result.Config,
					})
				}
				continue
			}

			retryResult.Language = job.Language
			retryResult.RetryOf = result
			result.RetriedBy = &retryResult

			if e.emitter != nil {
				e.emitter.Emit(model.EventTestRetried, model.RetryPayload{
					Original: *result,
					Retry:    retryResult,
				})
			}

			result.Classification = e.classifier.Classify(*result, retryResult)

			if result.Classification == model.Flake {
				result.Fingerprint = e.fingerprinter.Fingerprint(*result)
				if e.emitter != nil {
					e.emitter.Emit(model.EventFlakeDetected, model.FlakePayload{
						Result:      *result,
						Fingerprint: result.Fingerprint,
					})
				}
			} else if result.Classification == model.RealFailure {
				if e.emitter != nil {
					e.emitter.Emit(model.EventRealFailure, map[string]any{
						"test":        result.Test,
						"config":      result.Config,
						"output":      result.Output,
						"fingerprint": e.fingerprinter.Fingerprint(*result),
					})
				}
			}
		} else if result.Status == model.StatusPass {
			if e.emitter != nil {
				e.emitter.Emit(model.EventTestPassed, map[string]any{
					"test":     result.Test,
					"config":   result.Config,
					"duration": result.Duration.Seconds(),
				})
			}
		}
	}

	if e.emitter != nil {
		e.emitter.Emit(model.EventJobCompleted, map[string]any{
			"job":      job.Name,
			"language": job.Language,
			"total":    len(allResults),
			"passed":   countByStatus(allResults, model.StatusPass),
			"failed":   countByStatus(allResults, model.StatusFail),
			"skipped":  countByStatus(allResults, model.StatusSkip),
			"flakes":   countByClassification(allResults, model.Flake),
			"real":     countByClassification(allResults, model.RealFailure),
			"infra":    countByClassification(allResults, model.Infrastructure),
		})
	}

	return &JobResult{
		Job:     job,
		Results: allResults,
	}, nil
}

func countByStatus(results []model.TestResult, status model.TestStatus) int {
	n := 0
	for _, r := range results {
		if r.Status == status {
			n++
		}
	}
	return n
}

func countByClassification(results []model.TestResult, c model.Classification) int {
	n := 0
	for _, r := range results {
		if r.Classification == c {
			n++
		}
	}
	return n
}
