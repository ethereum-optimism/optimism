package engine

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// DecisionEngine produces a PipelineDecision covering every job category.
type DecisionEngine struct {
	scoping   model.ScopingConfig
	placement model.PlacementConfig
	flakeDB   *model.FlakeDB
	affected  *AffectedResult
	emitter   *events.Emitter
}

// NewDecisionEngine creates a new DecisionEngine.
func NewDecisionEngine(scoping model.ScopingConfig, placement model.PlacementConfig, flakeDB *model.FlakeDB, affected *AffectedResult, emitter *events.Emitter) *DecisionEngine {
	return &DecisionEngine{
		scoping:   scoping,
		placement: placement,
		flakeDB:   flakeDB,
		affected:  affected,
		emitter:   emitter,
	}
}

// Decide evaluates all job categories against the changed files and affected results.
func (de *DecisionEngine) Decide(changedFiles []string, branch string, isSchedule bool) *model.PipelineDecision {
	isDevelop := branch == "develop"
	stage := model.DetermineStage(branch, isSchedule)

	decision := &model.PipelineDecision{
		ForceAll:   de.affected.ForceAll,
		Branch:     branch,
		Stage:      stage,
		IsDevelop:  isDevelop,
		IsSchedule: isSchedule,
		Categories: make(map[string]*model.CategoryDecision),
	}

	for name, cat := range de.scoping.JobCategories {
		cd := de.evaluateCategory(name, cat, changedFiles, isDevelop, isSchedule)

		// Apply stage-based filtering: if the category is placed at a later
		// stage than the current run, skip it (unless it was already skipped
		// for another reason).
		if cd.Needed && !cd.Skipped {
			placedAt := de.placement.GetCategoryStage(name)
			cd.PlacedAt = placedAt
			if !model.ShouldRunAtStage(placedAt, stage) {
				cd.Needed = false
				cd.Skipped = true
				cd.StageSkipped = true
				cd.SkipWhy = fmt.Sprintf("deferred to %s stage (current: %s)", placedAt, stage)
			}
		}

		decision.Categories[name] = cd
	}

	if de.emitter != nil {
		de.emitter.Emit(model.EventPipelineDecision, decision)
	}

	return decision
}

func (de *DecisionEngine) evaluateCategory(
	name string,
	cat model.JobCategoryConfig,
	changedFiles []string,
	isDevelop, isSchedule bool,
) *model.CategoryDecision {
	cd := &model.CategoryDecision{}

	// Schedule-only jobs skip on non-schedule runs.
	if cat.ScheduleOnly && !isSchedule {
		cd.Skipped = true
		cd.SkipWhy = "schedule-only job, not a scheduled run"
		return cd
	}

	// Tag-only jobs skip on non-tag runs.
	if cat.TagOnly {
		cd.Skipped = true
		cd.SkipWhy = "tag-only job"
		return cd
	}

	// Develop-only jobs skip on non-develop branches.
	if cat.DevelopOnly && !isDevelop {
		cd.Skipped = true
		cd.SkipWhy = "develop-only job, running on PR branch"
		return cd
	}

	// Always-needed jobs.
	if cat.Always {
		cd.Needed = true
		cd.Reason = "always-run category"
		return cd
	}

	// Force-all means everything runs.
	if de.affected.ForceAll {
		cd.Needed = true
		cd.Reason = "force-all triggered"
		return cd
	}

	// Always-on-develop means this runs on develop regardless.
	if cat.AlwaysOnDevelop && isDevelop {
		cd.Needed = true
		cd.Reason = "always runs on develop"
		return cd
	}

	// Graph-based categories delegate to the affected result.
	if cat.UseGraph && cat.Language != "" {
		return de.evaluateGraphCategory(name, cat)
	}

	// Fuzz packages: check each package's trigger paths individually.
	if len(cat.FuzzPackages) > 0 {
		return de.evaluateFuzzCategory(cat, changedFiles)
	}

	// Path-based: check if any changed file matches trigger paths.
	if len(cat.TriggerPaths) > 0 {
		matched := matchPaths(changedFiles, cat.TriggerPaths)
		if matched {
			cd.Needed = true
			cd.Reason = fmt.Sprintf("changed files match trigger paths for %s", name)

			// For feature matrix categories, include all features.
			if len(cat.FeatureMatrix) > 0 {
				cd.Features = cat.FeatureMatrix
			}
			if len(cat.Configs) > 0 {
				cd.Configs = cat.Configs
			}
		} else {
			cd.Skipped = true
			cd.SkipWhy = "no changed files match trigger paths"
		}
		return cd
	}

	// No trigger configuration — skip by default.
	cd.Skipped = true
	cd.SkipWhy = "no trigger configuration"
	return cd
}

func (de *DecisionEngine) evaluateGraphCategory(name string, cat model.JobCategoryConfig) *model.CategoryDecision {
	cd := &model.CategoryDecision{}
	lr, ok := de.affected.ByLanguage[cat.Language]
	if !ok || lr.SelectedTargets == 0 {
		cd.Skipped = true
		cd.SkipWhy = fmt.Sprintf("no %s targets affected", cat.Language)
		return cd
	}

	cd.Needed = true
	cd.Reason = fmt.Sprintf("%d %s targets affected (%.0f%% skip rate)", lr.SelectedTargets, cat.Language, lr.SkipRate*100)

	targets := make([]string, len(lr.Targets))
	for i, t := range lr.Targets {
		targets[i] = t.ID
	}
	cd.Targets = targets

	// For sol, include feature matrix if configured.
	if len(cat.FeatureMatrix) > 0 {
		cd.Features = cat.FeatureMatrix
	}

	// Include configurations from the adapter.
	if len(lr.Configurations) > 0 {
		configs := make([]string, len(lr.Configurations))
		for i, c := range lr.Configurations {
			configs[i] = c.Name
		}
		cd.Configs = configs
	}

	return cd
}

func (de *DecisionEngine) evaluateFuzzCategory(cat model.JobCategoryConfig, changedFiles []string) *model.CategoryDecision {
	cd := &model.CategoryDecision{}
	var neededPackages []string

	for _, fp := range cat.FuzzPackages {
		if matchPaths(changedFiles, fp.TriggerPaths) {
			neededPackages = append(neededPackages, fp.Package)
		}
	}

	if len(neededPackages) == 0 {
		cd.Skipped = true
		cd.SkipWhy = "no fuzz packages triggered by changed files"
		return cd
	}

	cd.Needed = true
	cd.Packages = neededPackages
	cd.Reason = fmt.Sprintf("%d fuzz packages triggered: %s", len(neededPackages), strings.Join(neededPackages, ", "))
	return cd
}

// matchPaths checks if any changed file matches any of the trigger path prefixes.
func matchPaths(changedFiles, triggerPaths []string) bool {
	for _, f := range changedFiles {
		for _, tp := range triggerPaths {
			// Support glob-like "**/" prefix by stripping it.
			pattern := strings.TrimPrefix(tp, "**/")
			if strings.HasPrefix(f, pattern) || strings.HasSuffix(f, pattern) {
				return true
			}
		}
	}
	return false
}
