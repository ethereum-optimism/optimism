package model

import "testing"

func TestGetCategoryStage_Default(t *testing.T) {
	pc := PlacementConfig{DefaultStage: StagePR}
	got := pc.GetCategoryStage("unknown_category")
	if got != StagePR {
		t.Errorf("GetCategoryStage(unknown) = %q, want %q", got, StagePR)
	}
}

func TestGetCategoryStage_EmptyDefault(t *testing.T) {
	pc := PlacementConfig{}
	got := pc.GetCategoryStage("unknown_category")
	if got != StagePR {
		t.Errorf("GetCategoryStage with empty default = %q, want %q", got, StagePR)
	}
}

func TestGetCategoryStage_ExplicitAssignment(t *testing.T) {
	pc := PlacementConfig{
		DefaultStage: StagePR,
		Assignments: map[string]CategoryPlacement{
			"sol_coverage": {Stage: StagePostMerge, Source: "static"},
		},
	}
	got := pc.GetCategoryStage("sol_coverage")
	if got != StagePostMerge {
		t.Errorf("GetCategoryStage(sol_coverage) = %q, want %q", got, StagePostMerge)
	}
}

func TestGetCategoryStage_PinnedOverridesAssignment(t *testing.T) {
	pc := PlacementConfig{
		Constraints: []PlacementConstraint{
			{Category: "go_lint", PinnedStage: StagePR, Reason: "lint must run early"},
		},
		Assignments: map[string]CategoryPlacement{
			"go_lint": {Stage: StageNightly, Source: "optimizer"},
		},
	}
	got := pc.GetCategoryStage("go_lint")
	if got != StagePR {
		t.Errorf("pinned should override assignment: got %q, want %q", got, StagePR)
	}
}

func TestGetCategoryStage_NonMatchingConstraint(t *testing.T) {
	pc := PlacementConfig{
		DefaultStage: StagePR,
		Constraints: []PlacementConstraint{
			{Category: "go_lint", PinnedStage: StagePR},
		},
		Assignments: map[string]CategoryPlacement{
			"sol_tests": {Stage: StagePostMerge, Source: "optimizer"},
		},
	}
	// Constraint is for go_lint, not sol_tests — assignment should win.
	got := pc.GetCategoryStage("sol_tests")
	if got != StagePostMerge {
		t.Errorf("got %q, want %q", got, StagePostMerge)
	}
}
