package engine

import (
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// ShadowDeferralReport summarizes what would have been deferred in a pipeline.
type ShadowDeferralReport struct {
	PipelineID   string                    `json:"pipeline_id"`
	Stage        model.Stage               `json:"stage"`
	Categories   []CategoryDeferralReport  `json:"categories"`
	TotalTests   int                       `json:"total_tests"`
	WouldDefer   int                       `json:"would_defer"`
	EstTimeSaved time.Duration             `json:"est_time_saved"`
	ActualMisses int                       `json:"actual_misses"` // deferred tests that actually failed
	MissRate     float64                   `json:"miss_rate"`     // actual_misses / would_defer
}

// CategoryDeferralReport is the deferral summary for a single category.
type CategoryDeferralReport struct {
	Category      string         `json:"category"`
	Tests         int            `json:"tests"`
	Deferred      int            `json:"deferred"`
	DeferredTests []DeferredTest `json:"deferred_tests"`
	Misses        int            `json:"misses"`
}

// DeferredTest describes a single test that would have been deferred.
type DeferredTest struct {
	TestKey         string           `json:"test_key"`
	Duration        time.Duration    `json:"duration"`
	DeferTo         model.Stage      `json:"defer_to"`
	ActualResult    model.TestStatus `json:"actual_result"`
	WouldHaveMissed bool             `json:"would_have_missed"`
}

// ShadowDeferralAnalyzer produces deferral reports from test results.
type ShadowDeferralAnalyzer struct {
	emitter *events.Emitter
}

// NewShadowDeferralAnalyzer creates a new ShadowDeferralAnalyzer.
func NewShadowDeferralAnalyzer(emitter *events.Emitter) *ShadowDeferralAnalyzer {
	return &ShadowDeferralAnalyzer{emitter: emitter}
}

// Analyze takes test results (with shadow deferral annotations) and produces a report.
func (sda *ShadowDeferralAnalyzer) Analyze(pipelineID string, stage model.Stage, categoryResults map[string][]model.TestResult) *ShadowDeferralReport {
	report := &ShadowDeferralReport{
		PipelineID: pipelineID,
		Stage:      stage,
	}

	for catName, results := range categoryResults {
		catReport := CategoryDeferralReport{
			Category: catName,
			Tests:    len(results),
		}

		for _, r := range results {
			report.TotalTests++

			if !r.WouldDefer {
				continue
			}

			dt := DeferredTest{
				TestKey:      r.Test.Key(),
				Duration:     r.Duration,
				DeferTo:      model.Stage(r.DeferTo),
				ActualResult: r.Status,
			}

			// A miss is when a deferred test actually failed with a real failure.
			if r.Status == model.StatusFail && r.Classification == model.RealFailure {
				dt.WouldHaveMissed = true
				catReport.Misses++
				report.ActualMisses++
			}

			catReport.Deferred++
			catReport.DeferredTests = append(catReport.DeferredTests, dt)
			report.WouldDefer++
			report.EstTimeSaved += r.Duration
		}

		if catReport.Deferred > 0 {
			report.Categories = append(report.Categories, catReport)
		}
	}

	if report.WouldDefer > 0 {
		report.MissRate = float64(report.ActualMisses) / float64(report.WouldDefer)
	}

	if sda.emitter != nil {
		sda.emitter.Emit(model.EventShadowDeferralReport, report)
	}

	return report
}
