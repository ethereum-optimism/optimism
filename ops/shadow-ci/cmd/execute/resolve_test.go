package main

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestResolveCommand_NoTargetCommand(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command: "make test",
	}
	cd := &model.CategoryDecision{Targets: []string{"pkg/a", "pkg/b"}}
	got := resolveCommand(cat, cd)
	if got != "make test" {
		t.Errorf("expected fallback to Command, got %q", got)
	}
}

func TestResolveCommand_NoTargets(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "make test",
		TargetCommand: "gotestsum --packages={{targets}}",
	}
	cd := &model.CategoryDecision{} // no targets
	got := resolveCommand(cat, cd)
	if got != "make test" {
		t.Errorf("expected fallback to Command, got %q", got)
	}
}

func TestResolveCommand_SpaceSeparated(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "make test",
		TargetCommand: "gotestsum --packages=\"{{targets}}\"",
	}
	cd := &model.CategoryDecision{
		Targets: []string{"github.com/foo/bar/op-node", "github.com/foo/bar/op-batcher"},
	}
	got := resolveCommand(cat, cd)
	want := "gotestsum --packages=\"github.com/foo/bar/op-node github.com/foo/bar/op-batcher\""
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCommand_BraceGlob(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "forge test",
		TargetCommand: "forge test --match-path \"{{targets_glob}}\"",
	}
	cd := &model.CategoryDecision{
		Targets: []string{"test/L1/Portal.t.sol", "test/L2/Bridge.t.sol"},
	}
	got := resolveCommand(cat, cd)
	want := "forge test --match-path \"{test/L1/Portal.t.sol,test/L2/Bridge.t.sol}\""
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCommand_BraceGlobSingle(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "forge test",
		TargetCommand: "forge test --match-path \"{{targets_glob}}\"",
	}
	cd := &model.CategoryDecision{
		Targets: []string{"test/L1/Portal.t.sol"},
	}
	got := resolveCommand(cat, cd)
	want := "forge test --match-path \"test/L1/Portal.t.sol\""
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCommand_CSV(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "run all",
		TargetCommand: "run --targets={{targets_csv}}",
	}
	cd := &model.CategoryDecision{
		Targets: []string{"a", "b", "c"},
	}
	got := resolveCommand(cat, cd)
	want := "run --targets=a,b,c"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCommand_MultiplePlaceholders(t *testing.T) {
	cat := model.JobCategoryConfig{
		Command:       "run all",
		TargetCommand: "echo {{targets}} && run --glob={{targets_glob}}",
	}
	cd := &model.CategoryDecision{
		Targets: []string{"x", "y"},
	}
	got := resolveCommand(cat, cd)
	want := "echo x y && run --glob={x,y}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
