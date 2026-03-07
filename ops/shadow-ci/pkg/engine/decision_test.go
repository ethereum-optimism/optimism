package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

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

func TestDecisionEngine_PathBasedWithFeatureMatrix(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"sol_tests": {
				TriggerPaths:  []string{"packages/contracts-bedrock/"},
				FeatureMatrix: []string{"main", "CUSTOM_GAS_TOKEN", "OPCM_V2"},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	decision := de.Decide([]string{"packages/contracts-bedrock/src/L1/OptimismPortal.sol"}, "feat/x", false)
	cat := decision.Categories["sol_tests"]
	assert.True(t, cat.Needed)
	assert.Equal(t, []string{"main", "CUSTOM_GAS_TOKEN", "OPCM_V2"}, cat.Features)
}

func TestDecisionEngine_PathBasedWithConfigs(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"sol_checks": {
				TriggerPaths: []string{"packages/contracts-bedrock/"},
				Configs:      []string{"lint", "size-check", "snapshots"},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	decision := de.Decide([]string{"packages/contracts-bedrock/src/L1/SystemConfig.sol"}, "feat/x", false)
	cat := decision.Categories["sol_checks"]
	assert.True(t, cat.Needed)
	assert.Equal(t, []string{"lint", "size-check", "snapshots"}, cat.Configs)
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

	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "feat/x", false)

	// go_tests: has affected targets.
	assert.True(t, decision.Categories["go_tests"].Needed)
	assert.Len(t, decision.Categories["go_tests"].Targets, 2)

	// sol_tests: no sol targets.
	assert.True(t, decision.Categories["sol_tests"].Skipped)
}

func TestDecisionEngine_GraphBasedWithFeatureMatrix(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"sol_tests": {
				UseGraph:      true,
				Language:      "sol",
				FeatureMatrix: []string{"main", "CUSTOM_GAS_TOKEN"},
			},
		},
	}

	affected := &AffectedResult{
		ByLanguage: map[string]*LanguageResult{
			"sol": {
				Targets:         []model.Target{{ID: "test/L1/OptimismPortal.t.sol"}},
				SelectedTargets: 1,
				TotalTargets:    50,
				SkipRate:        0.98,
			},
		},
	}

	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "feat/x", false)

	cat := decision.Categories["sol_tests"]
	assert.True(t, cat.Needed)
	assert.Equal(t, []string{"main", "CUSTOM_GAS_TOKEN"}, cat.Features)
	assert.Len(t, cat.Targets, 1)
}

func TestDecisionEngine_GraphBasedWithConfigurations(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests": {
				UseGraph: true,
				Language: "go",
			},
		},
	}

	affected := &AffectedResult{
		ByLanguage: map[string]*LanguageResult{
			"go": {
				Targets:         []model.Target{{ID: "pkg/foo"}},
				Configurations:  []model.Configuration{{Name: "short"}, {Name: "full"}},
				SelectedTargets: 1,
				TotalTargets:    100,
				SkipRate:        0.99,
			},
		},
	}

	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "feat/x", false)

	cat := decision.Categories["go_tests"]
	assert.True(t, cat.Needed)
	assert.Equal(t, []string{"short", "full"}, cat.Configs)
}

func TestDecisionEngine_GraphBasedNoTargets(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests": {
				UseGraph: true,
				Language: "go",
			},
		},
	}

	affected := &AffectedResult{
		ByLanguage: map[string]*LanguageResult{
			"go": {
				Targets:         nil,
				SelectedTargets: 0,
				TotalTargets:    100,
				SkipRate:        1.0,
			},
		},
	}

	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "feat/x", false)

	cat := decision.Categories["go_tests"]
	assert.True(t, cat.Skipped)
	assert.Contains(t, cat.SkipWhy, "no go targets affected")
}

func TestDecisionEngine_ForceAll(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint":  {TriggerPaths: []string{"op-node/"}},
			"sol_lint": {TriggerPaths: []string{"packages/contracts-bedrock/"}},
		},
	}

	affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "develop", false)

	// Everything runs when force-all is triggered.
	assert.True(t, decision.Categories["go_lint"].Needed)
	assert.True(t, decision.Categories["sol_lint"].Needed)
	assert.True(t, decision.ForceAll)
}

func TestDecisionEngine_ForceAllSkipsScheduleOnly(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint":      {TriggerPaths: []string{"op-node/"}},
			"nightly_fuzz": {ScheduleOnly: true},
		},
	}

	affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
	decision := de.Decide(nil, "develop", false)

	// Force-all runs normal jobs.
	assert.True(t, decision.Categories["go_lint"].Needed)
	// But schedule-only is checked before force-all.
	assert.True(t, decision.Categories["nightly_fuzz"].Skipped)
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
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// Only op-node changed.
	changedFiles := []string{"op-node/rollup/derive/batch.go"}
	decision := de.Decide(changedFiles, "feat/x", false)

	assert.True(t, decision.Categories["go_fuzz"].Needed)
	assert.Equal(t, []string{"op-node"}, decision.Categories["go_fuzz"].Packages)
}

func TestDecisionEngine_FuzzPackagesMultiple(t *testing.T) {
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
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	changedFiles := []string{"op-node/rollup/derive/batch.go", "cannon/mipsevm/exec.go"}
	decision := de.Decide(changedFiles, "feat/x", false)

	cat := decision.Categories["go_fuzz"]
	assert.True(t, cat.Needed)
	assert.Len(t, cat.Packages, 2)
	assert.Contains(t, cat.Packages, "op-node")
	assert.Contains(t, cat.Packages, "cannon")
}

func TestDecisionEngine_FuzzPackagesNoneTriggered(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_fuzz": {
				FuzzPackages: []model.FuzzPackage{
					{Package: "op-node", TriggerPaths: []string{"op-node/"}},
					{Package: "cannon", TriggerPaths: []string{"cannon/"}},
				},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	changedFiles := []string{"README.md"}
	decision := de.Decide(changedFiles, "feat/x", false)

	cat := decision.Categories["go_fuzz"]
	assert.True(t, cat.Skipped)
	assert.Contains(t, cat.SkipWhy, "no fuzz packages triggered")
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
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// On develop: always runs regardless of changed files.
	d1 := de.Decide([]string{"README.md"}, "develop", false)
	assert.True(t, d1.Categories["acceptance"].Needed)
	assert.Contains(t, d1.Categories["acceptance"].Reason, "always runs on develop")

	// On feature branch: only runs if trigger paths match.
	d2 := de.Decide([]string{"README.md"}, "feat/x", false)
	assert.True(t, d2.Categories["acceptance"].Skipped)
}

func TestDecisionEngine_Always(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"setup": {
				Always: true,
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// Runs on feature branch with no changes.
	decision := de.Decide(nil, "feat/x", false)
	assert.True(t, decision.Categories["setup"].Needed)
	assert.Contains(t, decision.Categories["setup"].Reason, "always-run")
}

func TestDecisionEngine_TagOnly(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"release": {
				TagOnly: true,
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	decision := de.Decide(nil, "develop", false)
	cat := decision.Categories["release"]
	assert.True(t, cat.Skipped)
	assert.Contains(t, cat.SkipWhy, "tag-only")
}

func TestDecisionEngine_ScheduleOnlyOnSchedule(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"nightly": {
				ScheduleOnly: true,
				TriggerPaths: []string{"op-node/"},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// Schedule-only skips on non-schedule.
	d1 := de.Decide([]string{"op-node/foo.go"}, "develop", false)
	assert.True(t, d1.Categories["nightly"].Skipped)

	// Schedule-only runs on schedule with matching paths.
	d2 := de.Decide([]string{"op-node/foo.go"}, "develop", true)
	assert.True(t, d2.Categories["nightly"].Needed)
}

func TestDecisionEngine_DevelopOnlyOnDevelop(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"fault_proofs": {
				DevelopOnly:  true,
				TriggerPaths: []string{"op-e2e/"},
			},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// On develop with matching paths: runs.
	d1 := de.Decide([]string{"op-e2e/faultproof_test.go"}, "develop", false)
	assert.True(t, d1.Categories["fault_proofs"].Needed)

	// On feature branch: skipped.
	d2 := de.Decide([]string{"op-e2e/faultproof_test.go"}, "feat/x", false)
	assert.True(t, d2.Categories["fault_proofs"].Skipped)
}

func TestDecisionEngine_NoTriggerConfig(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"unknown": {},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	decision := de.Decide([]string{"anything.go"}, "feat/x", false)
	cat := decision.Categories["unknown"]
	assert.True(t, cat.Skipped)
	assert.Contains(t, cat.SkipWhy, "no trigger configuration")
}

func TestDecisionEngine_DecisionMetadata(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{},
	}

	affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	decision := de.Decide(nil, "develop", true)
	assert.Equal(t, "develop", decision.Branch)
	assert.True(t, decision.IsDevelop)
	assert.True(t, decision.IsSchedule)
	assert.True(t, decision.ForceAll)
}

func TestDecisionEngine_PriorityOrder(t *testing.T) {
	// Tests that the evaluation priority is correct:
	// schedule-only > tag-only > develop-only > always > force-all > always-on-develop > graph > fuzz > path > default
	t.Run("schedule-only beats force-all", func(t *testing.T) {
		scoping := model.ScopingConfig{
			JobCategories: map[string]model.JobCategoryConfig{
				"nightly": {ScheduleOnly: true},
			},
		}
		affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
		de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
		d := de.Decide(nil, "develop", false)
		assert.True(t, d.Categories["nightly"].Skipped)
	})

	t.Run("tag-only beats force-all", func(t *testing.T) {
		scoping := model.ScopingConfig{
			JobCategories: map[string]model.JobCategoryConfig{
				"release": {TagOnly: true},
			},
		}
		affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
		de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
		d := de.Decide(nil, "develop", false)
		assert.True(t, d.Categories["release"].Skipped)
	})

	t.Run("develop-only beats force-all on PR", func(t *testing.T) {
		scoping := model.ScopingConfig{
			JobCategories: map[string]model.JobCategoryConfig{
				"heavy": {DevelopOnly: true, TriggerPaths: []string{"op-node/"}},
			},
		}
		affected := &AffectedResult{ForceAll: true, ByLanguage: map[string]*LanguageResult{}}
		de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
		d := de.Decide([]string{"op-node/foo.go"}, "feat/x", false)
		assert.True(t, d.Categories["heavy"].Skipped)
	})

	t.Run("always beats everything else", func(t *testing.T) {
		scoping := model.ScopingConfig{
			JobCategories: map[string]model.JobCategoryConfig{
				"setup": {Always: true},
			},
		}
		affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
		de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)
		d := de.Decide(nil, "feat/x", false)
		assert.True(t, d.Categories["setup"].Needed)
	})
}

// matchPaths tests

func TestMatchPaths_PrefixMatch(t *testing.T) {
	assert.True(t, matchPaths(
		[]string{"op-node/rollup/derive/batch.go"},
		[]string{"op-node/"},
	))
}

func TestMatchPaths_SuffixMatch(t *testing.T) {
	assert.True(t, matchPaths(
		[]string{"packages/contracts-bedrock/src/L1/Portal.sol"},
		[]string{".sol"},
	))
}

func TestMatchPaths_GlobPrefix(t *testing.T) {
	// **/ is stripped, leaving "test/" — matches as suffix since file contains "/test/".
	assert.True(t, matchPaths(
		[]string{"test/L1/Portal.t.sol"},
		[]string{"**/test/"},
	))
	// Mid-path: "test/" is in the path but matchPaths uses HasPrefix/HasSuffix,
	// so it only matches at the start or end of the path.
	assert.False(t, matchPaths(
		[]string{"packages/contracts-bedrock/test/L1/Portal.t.sol"},
		[]string{"**/test/"},
	))
}

func TestMatchPaths_NoMatch(t *testing.T) {
	assert.False(t, matchPaths(
		[]string{"README.md"},
		[]string{"op-node/", "cannon/"},
	))
}

func TestMatchPaths_EmptyChangedFiles(t *testing.T) {
	assert.False(t, matchPaths(nil, []string{"op-node/"}))
}

func TestMatchPaths_EmptyTriggerPaths(t *testing.T) {
	assert.False(t, matchPaths([]string{"op-node/foo.go"}, nil))
}

func TestMatchPaths_MultipleFilesOneMatch(t *testing.T) {
	assert.True(t, matchPaths(
		[]string{"README.md", "op-node/foo.go", "docs/intro.md"},
		[]string{"op-node/"},
	))
}

func TestMatchPaths_MultipleTriggerPathsOneMatch(t *testing.T) {
	assert.True(t, matchPaths(
		[]string{"cannon/mipsevm/exec.go"},
		[]string{"op-node/", "cannon/", "op-e2e/"},
	))
}

// mergeTargets tests

func TestMergeTargets_NoDuplicates(t *testing.T) {
	a := []model.Target{{ID: "pkg/foo"}, {ID: "pkg/bar"}}
	b := []model.Target{{ID: "pkg/baz"}}
	merged := mergeTargets(a, b)
	require.Len(t, merged, 3)
	assert.Equal(t, "pkg/foo", merged[0].ID)
	assert.Equal(t, "pkg/bar", merged[1].ID)
	assert.Equal(t, "pkg/baz", merged[2].ID)
}

func TestMergeTargets_WithDuplicates(t *testing.T) {
	a := []model.Target{{ID: "pkg/foo"}, {ID: "pkg/bar"}}
	b := []model.Target{{ID: "pkg/bar"}, {ID: "pkg/baz"}}
	merged := mergeTargets(a, b)
	require.Len(t, merged, 3)
}

func TestMergeTargets_EmptyFirst(t *testing.T) {
	b := []model.Target{{ID: "pkg/bar"}}
	merged := mergeTargets(nil, b)
	require.Len(t, merged, 1)
	assert.Equal(t, "pkg/bar", merged[0].ID)
}

func TestMergeTargets_EmptySecond(t *testing.T) {
	a := []model.Target{{ID: "pkg/foo"}}
	merged := mergeTargets(a, nil)
	require.Len(t, merged, 1)
}

func TestMergeTargets_BothEmpty(t *testing.T) {
	merged := mergeTargets(nil, nil)
	assert.Empty(t, merged)
}

// resolveAlwaysRun tests

type mockGraph struct {
	targets []model.Target
}

func (m *mockGraph) AllTargets() []model.Target { return m.targets }

func TestResolveAlwaysRun_ExactMatch(t *testing.T) {
	g := &mockGraph{targets: []model.Target{
		{ID: "pkg/foo"},
		{ID: "pkg/bar"},
		{ID: "pkg/baz"},
	}}
	result := resolveAlwaysRun(g, []string{"pkg/bar"}, "go")
	require.Len(t, result, 1)
	assert.Equal(t, "pkg/bar", result[0].ID)
	assert.Equal(t, model.ScopeAlways, result[0].Scope)
}

func TestResolveAlwaysRun_PrefixMatch(t *testing.T) {
	g := &mockGraph{targets: []model.Target{
		{ID: "pkg/foo/a"},
		{ID: "pkg/foo/b"},
		{ID: "pkg/bar"},
	}}
	result := resolveAlwaysRun(g, []string{"pkg/foo"}, "go")
	require.Len(t, result, 2)
}

func TestResolveAlwaysRun_NoMatch(t *testing.T) {
	g := &mockGraph{targets: []model.Target{{ID: "pkg/bar"}}}
	result := resolveAlwaysRun(g, []string{"pkg/nonexistent"}, "go")
	assert.Empty(t, result)
}

func TestResolveAlwaysRun_EmptyIDs(t *testing.T) {
	g := &mockGraph{targets: []model.Target{{ID: "pkg/bar"}}}
	result := resolveAlwaysRun(g, nil, "go")
	assert.Nil(t, result)
}

// applyConfidence tests

func TestApplyConfidence_BelowThreshold(t *testing.T) {
	ac := &AffectedComputer{scoping: model.ScopingConfig{ConfidenceThreshold: 0.8}}
	targets := []model.Target{
		{ID: "pkg/low", Confidence: 0.5},
		{ID: "pkg/high", Confidence: 0.9},
	}
	result := ac.applyConfidence(targets)
	assert.Equal(t, model.ScopeAlways, result[0].Scope)
	assert.NotEqual(t, model.ScopeAlways, result[1].Scope)
}

func TestApplyConfidence_ZeroThreshold(t *testing.T) {
	ac := &AffectedComputer{scoping: model.ScopingConfig{ConfidenceThreshold: 0}}
	targets := []model.Target{{ID: "pkg/low", Confidence: 0.1}}
	result := ac.applyConfidence(targets)
	// Zero threshold means no promotion.
	assert.NotEqual(t, model.ScopeAlways, result[0].Scope)
}

func TestApplyConfidence_AlreadyAlways(t *testing.T) {
	ac := &AffectedComputer{scoping: model.ScopingConfig{ConfidenceThreshold: 0.8}}
	targets := []model.Target{
		{ID: "pkg/x", Confidence: 0.3, Scope: model.ScopeAlways},
	}
	result := ac.applyConfidence(targets)
	// Already always — stays always but doesn't error.
	assert.Equal(t, model.ScopeAlways, result[0].Scope)
}

// checkForceAllPaths tests

func TestCheckForceAllPaths(t *testing.T) {
	ac := &AffectedComputer{scoping: model.ScopingConfig{
		ForceAllPaths: []string{".circleci/", "Makefile"},
	}}

	assert.True(t, ac.checkForceAllPaths([]string{".circleci/config.yml"}))
	assert.True(t, ac.checkForceAllPaths([]string{"Makefile"}))
	assert.False(t, ac.checkForceAllPaths([]string{"op-node/foo.go"}))
	assert.False(t, ac.checkForceAllPaths(nil))
}

// Full integration-style test: realistic pipeline decision

func TestDecisionEngine_RealisticPipeline(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests": {
				UseGraph: true,
				Language: "go",
			},
			"go_fuzz": {
				FuzzPackages: []model.FuzzPackage{
					{Package: "op-node", TriggerPaths: []string{"op-node/"}},
					{Package: "op-service", TriggerPaths: []string{"op-service/"}},
					{Package: "cannon", TriggerPaths: []string{"cannon/"}},
					{Package: "op-e2e", TriggerPaths: []string{"op-e2e/", "packages/contracts-bedrock/src/"}},
				},
			},
			"go_lint": {
				TriggerPaths: []string{"op-node/", "op-batcher/", "op-proposer/", "op-service/", "op-e2e/", "cannon/"},
			},
			"cannon_tests": {
				TriggerPaths: []string{"cannon/", "packages/contracts-bedrock/src/cannon/"},
			},
			"sol_tests": {
				UseGraph:      true,
				Language:      "sol",
				FeatureMatrix: []string{"main", "CUSTOM_GAS_TOKEN", "OPCM_V2"},
			},
			"sol_checks": {
				TriggerPaths: []string{"packages/contracts-bedrock/"},
				Configs:      []string{"lint", "size-check"},
			},
			"acceptance_tests": {
				TriggerPaths:    []string{"op-node/", "op-batcher/", "op-e2e/", "op-acceptance-tests/"},
				AlwaysOnDevelop: true,
			},
			"nightly_heavy_fuzz": {
				ScheduleOnly: true,
			},
			"fault_proof_full": {
				DevelopOnly:  true,
				TriggerPaths: []string{"op-e2e/"},
			},
		},
	}

	affected := &AffectedResult{
		ByLanguage: map[string]*LanguageResult{
			"go": {
				Targets:         []model.Target{{ID: "op-node/..."}, {ID: "op-service/..."}},
				SelectedTargets: 2,
				TotalTargets:    200,
				SkipRate:        0.99,
			},
		},
	}

	de := NewDecisionEngine(scoping, model.PlacementConfig{DefaultStage: model.StagePR}, nil, affected, nil)

	// Simulate a PR that changes op-node and op-service.
	changedFiles := []string{
		"op-node/rollup/derive/batch.go",
		"op-service/txmgr/txmgr.go",
	}

	decision := de.Decide(changedFiles, "feat/txmgr-refactor", false)

	// Go tests: graph-based, has affected targets.
	assert.True(t, decision.Categories["go_tests"].Needed)

	// Go fuzz: op-node and op-service triggered (not cannon or op-e2e).
	fuzz := decision.Categories["go_fuzz"]
	assert.True(t, fuzz.Needed)
	assert.Contains(t, fuzz.Packages, "op-node")
	assert.Contains(t, fuzz.Packages, "op-service")
	assert.NotContains(t, fuzz.Packages, "cannon")

	// Go lint: triggered by op-node/ and op-service/.
	assert.True(t, decision.Categories["go_lint"].Needed)

	// Cannon tests: not triggered (no cannon/ changes).
	assert.True(t, decision.Categories["cannon_tests"].Skipped)

	// Sol tests: graph-based, no sol targets.
	assert.True(t, decision.Categories["sol_tests"].Skipped)

	// Sol checks: not triggered (no sol changes).
	assert.True(t, decision.Categories["sol_checks"].Skipped)

	// Acceptance: triggered by op-node/.
	assert.True(t, decision.Categories["acceptance_tests"].Needed)

	// Nightly heavy fuzz: schedule-only, skipped.
	assert.True(t, decision.Categories["nightly_heavy_fuzz"].Skipped)

	// Fault proof full: develop-only, skipped on PR.
	assert.True(t, decision.Categories["fault_proof_full"].Skipped)
}

// Stage-aware placement tests

func TestDecisionEngine_StageFiltering(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests":     {TriggerPaths: []string{"op-node/"}},
			"sol_coverage": {TriggerPaths: []string{"packages/contracts-bedrock/"}},
			"fault_proofs": {TriggerPaths: []string{"op-e2e/"}},
		},
	}

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"sol_coverage": {Stage: model.StagePostMerge, Source: "static"},
			"fault_proofs": {Stage: model.StagePostMerge, Source: "static"},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}

	t.Run("PR stage skips post_merge categories", func(t *testing.T) {
		de := NewDecisionEngine(scoping, placement, nil, affected, nil)
		d := de.Decide([]string{"op-node/foo.go", "packages/contracts-bedrock/src/X.sol", "op-e2e/test.go"}, "feat/x", false)

		// go_tests: placed at PR (default), runs at PR stage.
		assert.True(t, d.Categories["go_tests"].Needed)
		assert.False(t, d.Categories["go_tests"].StageSkipped)

		// sol_coverage: placed at post_merge, deferred at PR stage.
		assert.True(t, d.Categories["sol_coverage"].Skipped)
		assert.True(t, d.Categories["sol_coverage"].StageSkipped)
		assert.Equal(t, model.StagePostMerge, d.Categories["sol_coverage"].PlacedAt)
		assert.Contains(t, d.Categories["sol_coverage"].SkipWhy, "deferred to post_merge")

		// fault_proofs: placed at post_merge, deferred at PR stage.
		assert.True(t, d.Categories["fault_proofs"].Skipped)
		assert.True(t, d.Categories["fault_proofs"].StageSkipped)
	})

	t.Run("post_merge stage runs everything", func(t *testing.T) {
		de := NewDecisionEngine(scoping, placement, nil, affected, nil)
		// develop branch → post_merge stage.
		d := de.Decide([]string{"op-node/foo.go", "packages/contracts-bedrock/src/X.sol", "op-e2e/test.go"}, "develop", false)

		assert.Equal(t, model.StagePostMerge, d.Stage)
		assert.True(t, d.Categories["go_tests"].Needed)
		assert.True(t, d.Categories["sol_coverage"].Needed)
		assert.False(t, d.Categories["sol_coverage"].StageSkipped)
		assert.True(t, d.Categories["fault_proofs"].Needed)
		assert.False(t, d.Categories["fault_proofs"].StageSkipped)
	})
}

func TestDecisionEngine_PinnedConstraintOverridesStageFilter(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint": {TriggerPaths: []string{"op-node/"}},
		},
	}

	placement := model.PlacementConfig{
		DefaultStage: model.StagePostMerge,
		Constraints: []model.PlacementConstraint{
			{Category: "go_lint", PinnedStage: model.StagePR, Reason: "must run pre-merge"},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, placement, nil, affected, nil)
	d := de.Decide([]string{"op-node/foo.go"}, "feat/x", false)

	// go_lint is pinned to PR, so it runs even though default is post_merge.
	assert.True(t, d.Categories["go_lint"].Needed)
	assert.Equal(t, model.StagePR, d.Categories["go_lint"].PlacedAt)
}

func TestDecisionEngine_MergeQueueStage(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests":        {TriggerPaths: []string{"op-node/"}},
			"acceptance_tests": {TriggerPaths: []string{"op-node/"}},
		},
	}

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"acceptance_tests": {Stage: model.StageMergeQueue, Source: "optimizer"},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}

	t.Run("PR stage skips merge_queue categories", func(t *testing.T) {
		de := NewDecisionEngine(scoping, placement, nil, affected, nil)
		d := de.Decide([]string{"op-node/foo.go"}, "feat/x", false)

		assert.Equal(t, model.StagePR, d.Stage)
		assert.True(t, d.Categories["go_tests"].Needed)
		assert.True(t, d.Categories["acceptance_tests"].Skipped)
		assert.True(t, d.Categories["acceptance_tests"].StageSkipped)
	})

	t.Run("merge queue stage runs merge_queue categories", func(t *testing.T) {
		de := NewDecisionEngine(scoping, placement, nil, affected, nil)
		d := de.Decide([]string{"op-node/foo.go"}, "gh-readonly-queue/develop/pr-123-abc", false)

		assert.Equal(t, model.StageMergeQueue, d.Stage)
		assert.True(t, d.Categories["go_tests"].Needed)
		assert.True(t, d.Categories["acceptance_tests"].Needed)
		assert.False(t, d.Categories["acceptance_tests"].StageSkipped)
	})
}

func TestDecisionEngine_NightlyRunsEverything(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"heavy": {TriggerPaths: []string{"op-node/"}},
		},
	}

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"heavy": {Stage: model.StageNightly, Source: "optimizer"},
		},
	}

	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, placement, nil, affected, nil)
	d := de.Decide([]string{"op-node/foo.go"}, "develop", true)

	assert.Equal(t, model.StageNightly, d.Stage)
	assert.True(t, d.Categories["heavy"].Needed)
	assert.False(t, d.Categories["heavy"].StageSkipped)
}

func TestDecisionEngine_StageSetOnDecision(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{},
	}
	placement := model.PlacementConfig{DefaultStage: model.StagePR}
	affected := &AffectedResult{ByLanguage: map[string]*LanguageResult{}}
	de := NewDecisionEngine(scoping, placement, nil, affected, nil)

	tests := []struct {
		branch     string
		isSchedule bool
		want       model.Stage
	}{
		{"feat/x", false, model.StagePR},
		{"gh-readonly-queue/develop/pr-1-abc", false, model.StageMergeQueue},
		{"develop", false, model.StagePostMerge},
		{"develop", true, model.StageNightly},
	}

	for _, tt := range tests {
		d := de.Decide(nil, tt.branch, tt.isSchedule)
		assert.Equal(t, tt.want, d.Stage, "branch=%s schedule=%v", tt.branch, tt.isSchedule)
	}
}
