package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestDecisionEngine_PathBased(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint": {
				TriggerPaths: []string{"op-node/", "op-batcher/"},
			},
			"sol_tests": {
				TriggerPaths: []string{"packages/contracts-bedrock/"},
				FeatureMatrix: []string{"main", "CUSTOM_GAS_TOKEN"},
			},
			"nightly_fuzz": {
				ScheduleOnly: true,
			},
			"fault_proofs": {
				DevelopOnly: true,
				TriggerPaths: []string{"op-e2e/"},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, affected, nil)

	changedFiles := []string{"op-node/rollup/derive/batch.go", "op-node/README.md"}
	decision := de.Decide(changedFiles, "feat/my-feature", false)

	// go_lint: triggered by op-node/ change.
	assert.True(t, decision.Categories["go_lint"].Needed)

	// sol_tests: not triggered (no sol changes).
	assert.True(t, decision.Categories["sol_tests"].Skipped)

	// nightly_fuzz: schedule-only, not a scheduled run.
	assert.True(t, decision.Categories["nightly_fuzz"].Skipped)
	assert.Contains(t, decision.Categories["nightly_fuzz"].SkipWhy, "schedule-only")

	// fault_proofs: develop-only, we're on a feature branch.
	assert.True(t, decision.Categories["fault_proofs"].Skipped)
	assert.Contains(t, decision.Categories["fault_proofs"].SkipWhy, "develop-only")
}

func TestDecisionEngine_GraphBased(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests": {
				UseGraph: true,
				Language: "go",
			},
			"sol_tests": {
				UseGraph: true,
				Language: "sol",
			},
		},
	}

	affected := &AffectedResult{
		ByLanguage: map[string]*LanguageResult{
			"go": {
				Targets:         []model.Target{{ID: "pkg/foo"}, {ID: "pkg/bar"}},
				SelectedTargets: 2,
				TotalTargets:    100,
				SkipRate:        0.98,
			},
		},
	}

	de := NewDecisionEngine(scoping, affected, nil)
	decision := de.Decide(nil, "feat/x", false)

	// go_tests: has affected targets.
	assert.True(t, decision.Categories["go_tests"].Needed)
	assert.Len(t, decision.Categories["go_tests"].Targets, 2)

	// sol_tests: no sol targets.
	assert.True(t, decision.Categories["sol_tests"].Skipped)
}

func TestDecisionEngine_ForceAll(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint":  {TriggerPaths: []string{"op-node/"}},
			"sol_lint": {TriggerPaths: []string{"packages/contracts-bedrock/"}},
		},
	}

	affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, affected, nil)
	decision := de.Decide(nil, "develop", false)

	// Everything runs when force-all is triggered.
	assert.True(t, decision.Categories["go_lint"].Needed)
	assert.True(t, decision.Categories["sol_lint"].Needed)
}

func TestDecisionEngine_FuzzPackages(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_fuzz": {
				FuzzPackages: []model.FuzzPackage{
					{Package: "op-node", TriggerPaths: []string{"op-node/"}},
					{Package: "op-service", TriggerPaths: []string{"op-service/"}},
					{Package: "cannon", TriggerPaths: []string{"cannon/"}},
				},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, affected, nil)

	// Only op-node changed.
	changedFiles := []string{"op-node/rollup/derive/batch.go"}
	decision := de.Decide(changedFiles, "feat/x", false)

	assert.True(t, decision.Categories["go_fuzz"].Needed)
	assert.Equal(t, []string{"op-node"}, decision.Categories["go_fuzz"].Packages)
}

func TestDecisionEngine_AlwaysOnDevelop(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"acceptance": {
				TriggerPaths:    []string{"op-node/"},
				AlwaysOnDevelop: true,
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, affected, nil)

	// On develop: always runs regardless of changed files.
	d1 := de.Decide([]string{"README.md"}, "develop", false)
	assert.True(t, d1.Categories["acceptance"].Needed)

	// On feature branch: only runs if trigger paths match.
	d2 := de.Decide([]string{"README.md"}, "feat/x", false)
	assert.True(t, d2.Categories["acceptance"].Skipped)
}
