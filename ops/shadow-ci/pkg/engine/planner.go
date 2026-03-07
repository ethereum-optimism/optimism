package engine

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Planner creates TestPlans from AffectedResults.
type Planner struct{}

// NewPlanner creates a new Planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Plan produces a TestPlan from a trigger and affected result.
func (p *Planner) Plan(trigger model.Trigger, changedFiles []string, affected *AffectedResult, emitter *events.Emitter) *model.TestPlan {
	plan := &model.TestPlan{
		ID:           generatePlanID(),
		CreatedAt:    time.Now().UTC(),
		Trigger:      trigger,
		ChangedFiles: changedFiles,
		Summary: model.PlanSummary{
			ByLanguage: make(map[string]model.LanguageSummary),
		},
	}

	for lang, lr := range affected.ByLanguage {
		if lr.SelectedTargets == 0 {
			plan.Summary.ByLanguage[lang] = model.LanguageSummary{
				TotalTargets: lr.TotalTargets,
				SkipRate:     1.0,
			}
			continue
		}

		// One job per configuration.
		for _, config := range lr.Configurations {
			job := model.PlannedJob{
				Name:            fmt.Sprintf("%s-%s", lang, config.Name),
				Language:        lang,
				Targets:         lr.Targets,
				Configurations:  []model.Configuration{config},
				Resources:       computeResources(lr),
				SelectionReason: selectionReason(lr, affected.ForceAll),
			}
			plan.Jobs = append(plan.Jobs, job)
		}

		estTime := estimatedTime(lr.Targets)
		plan.Summary.ByLanguage[lang] = model.LanguageSummary{
			SelectedTargets: lr.SelectedTargets,
			TotalTargets:    lr.TotalTargets,
			SkipRate:        lr.SkipRate,
			AlwaysRunCount:  lr.AlwaysRunCount,
			Configurations:  len(lr.Configurations),
			EstimatedTime:   estTime,
		}
	}

	// Comparison job depends on all test jobs.
	if len(plan.Jobs) > 0 {
		jobNames := make([]string, len(plan.Jobs))
		for i, j := range plan.Jobs {
			jobNames[i] = j.Name
		}
		plan.Dependencies = map[string][]string{
			"comparison": jobNames,
		}
	}

	if emitter != nil {
		emitter.Emit(model.EventPlanCreated, plan)
	}

	return plan
}

func computeResources(lr *LanguageResult) model.Resources {
	totalDuration := estimatedTime(lr.Targets)

	switch {
	case totalDuration < 2*time.Minute:
		return model.Resources{Parallelism: 1, Runner: "medium", Timeout: 5 * time.Minute}
	case totalDuration < 10*time.Minute:
		return model.Resources{Parallelism: 2, Runner: "large", Timeout: 15 * time.Minute}
	case totalDuration < 30*time.Minute:
		return model.Resources{Parallelism: 4, Runner: "large", Timeout: 20 * time.Minute}
	default:
		return model.Resources{Parallelism: 8, Runner: "xlarge", Timeout: 30 * time.Minute}
	}
}

func selectionReason(lr *LanguageResult, forceAll bool) string {
	if forceAll {
		return "full_suite"
	}
	if lr.AlwaysRunCount == lr.SelectedTargets {
		return "always_run"
	}
	return "affected"
}

func estimatedTime(targets []model.Target) time.Duration {
	var total time.Duration
	for _, t := range targets {
		if t.EstimatedDuration > 0 {
			total += t.EstimatedDuration
		} else {
			total += 10 * time.Second // default estimate
		}
	}
	return total
}

func generatePlanID() string {
	return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}
