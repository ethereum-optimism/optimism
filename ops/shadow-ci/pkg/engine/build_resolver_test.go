package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestBuildResolver_DirectDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests":        {Language: "go", DependsOn: []string{"go_build"}},
			"go_build":        {Group: "build"},
			"contracts_build": {Group: "build"},
		},
	}

	resolver := NewBuildResolver(scoping)
	categories := map[string]*model.CategoryDecision{
		"go_tests": {Needed: true},
	}

	builds := resolver.Resolve(categories)
	if len(builds) != 1 || builds[0] != "go_build" {
		t.Errorf("expected [go_build], got %v", builds)
	}
}

func TestBuildResolver_TransitiveDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests":  {Language: "go", DependsOn: []string{"go_build"}},
			"go_build":  {Group: "build", DependsOn: []string{"base_build"}},
			"base_build": {Group: "build"},
		},
	}

	resolver := NewBuildResolver(scoping)
	categories := map[string]*model.CategoryDecision{
		"go_tests": {Needed: true},
	}

	builds := resolver.Resolve(categories)
	if len(builds) != 2 {
		t.Fatalf("expected 2 builds, got %v", builds)
	}
	if !contains(builds, "go_build") || !contains(builds, "base_build") {
		t.Errorf("expected go_build and base_build, got %v", builds)
	}
}

func TestBuildResolver_NoDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint": {TriggerPaths: []string{"op-node/"}},
		},
	}

	resolver := NewBuildResolver(scoping)
	categories := map[string]*model.CategoryDecision{
		"go_lint": {Needed: true},
	}

	builds := resolver.Resolve(categories)
	if len(builds) != 0 {
		t.Errorf("expected no builds, got %v", builds)
	}
}

func TestBuildResolver_SkippedCategoriesIgnored(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"sol_tests":       {Language: "sol", DependsOn: []string{"contracts_build"}},
			"contracts_build": {Group: "build"},
		},
	}

	resolver := NewBuildResolver(scoping)
	categories := map[string]*model.CategoryDecision{
		"sol_tests": {Needed: false, Skipped: true, SkipWhy: "no sol targets"},
	}

	builds := resolver.Resolve(categories)
	if len(builds) != 0 {
		t.Errorf("expected no builds (category skipped), got %v", builds)
	}
}

func TestBuildResolver_CircularDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"a": {DependsOn: []string{"b"}},
			"b": {Group: "build", DependsOn: []string{"a"}},
		},
	}

	resolver := NewBuildResolver(scoping)
	categories := map[string]*model.CategoryDecision{
		"a": {Needed: true},
	}

	// Should not hang due to circular deps.
	builds := resolver.Resolve(categories)
	if len(builds) != 1 || builds[0] != "b" {
		t.Errorf("expected [b], got %v", builds)
	}
}
