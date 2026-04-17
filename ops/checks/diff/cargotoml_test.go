package diff

import (
	"reflect"
	"testing"
)

// TestAnalyzeCargoTomlChange_DepVersionBump — a single dep bump in
// the [dependencies] block produces exactly that dep in AffectedDeps
// with no blast.
func TestAnalyzeCargoTomlChange_DepVersionBump(t *testing.T) {
	d := FileDiff{
		Path: "rust/crates/kona-derive/Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[dependencies]"},
			Removed: []string{`serde = "1.0.0"`},
			Added:   []string{`serde = "1.0.1"`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	want := []string{"serde"}
	if !reflect.DeepEqual(got.AffectedDeps, want) {
		t.Errorf("AffectedDeps = %v, want %v", got.AffectedDeps, want)
	}
	if got.ForceBlast {
		t.Error("version bump should not force blast")
	}
}

// TestAnalyzeCargoTomlChange_AddAndRemove — adds and removes both
// register their crate name.
func TestAnalyzeCargoTomlChange_AddAndRemove(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[dependencies]"},
			Removed: []string{`old-crate = "0.1"`},
			Added:   []string{`new-crate = { version = "2.0" }`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	want := map[string]bool{"old-crate": true, "new-crate": true}
	for _, d := range got.AffectedDeps {
		if !want[d] {
			t.Errorf("unexpected dep %q", d)
		}
		delete(want, d)
	}
	if len(want) > 0 {
		t.Errorf("missing deps: %v", want)
	}
}

// TestAnalyzeCargoTomlChange_DevAndBuildDeps — changes in
// [dev-dependencies] and [build-dependencies] are captured too.
func TestAnalyzeCargoTomlChange_DevAndBuildDeps(t *testing.T) {
	for _, section := range []string{"dev-dependencies", "build-dependencies"} {
		d := FileDiff{
			Path: "Cargo.toml",
			Hunks: []Hunk{{
				Context: []string{"[" + section + "]"},
				Added:   []string{`proptest = "1"`},
			}},
		}
		got := AnalyzeCargoTomlChange(d)
		if len(got.AffectedDeps) != 1 || got.AffectedDeps[0] != "proptest" {
			t.Errorf("section %s: got %v, want [proptest]", section, got.AffectedDeps)
		}
	}
}

// TestAnalyzeCargoTomlChange_WorkspaceDependencies — workspace.* dep
// tables share the rule.
func TestAnalyzeCargoTomlChange_WorkspaceDependencies(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[workspace.dependencies]"},
			Added:   []string{`alloy = "0.8"`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if len(got.AffectedDeps) != 1 || got.AffectedDeps[0] != "alloy" {
		t.Errorf("workspace.dependencies: got %v, want [alloy]", got.AffectedDeps)
	}
	if got.ForceBlast {
		t.Error("workspace.dependencies dep add should not force blast")
	}
}

// TestAnalyzeCargoTomlChange_ForceBlastOnPackageFields — touching
// [package] identity fields (name, edition, resolver) forces blast.
func TestAnalyzeCargoTomlChange_ForceBlastOnPackageFields(t *testing.T) {
	for _, field := range []string{"name", "edition", "rust-version", "resolver"} {
		d := FileDiff{
			Path: "Cargo.toml",
			Hunks: []Hunk{{
				Context: []string{"[package]"},
				Added:   []string{field + ` = "x"`},
			}},
		}
		got := AnalyzeCargoTomlChange(d)
		if !got.ForceBlast {
			t.Errorf("change to [package].%s should force blast", field)
		}
	}
}

// TestAnalyzeCargoTomlChange_ForceBlastOnFeatures — [features]
// changes flip compiled code paths across every consumer; can't
// model as a dep change.
func TestAnalyzeCargoTomlChange_ForceBlastOnFeatures(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[features]"},
			Added:   []string{`default = ["foo"]`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if !got.ForceBlast {
		t.Error("features change should force blast")
	}
}

// TestAnalyzeCargoTomlChange_ForceBlastOnWorkspaceSection — the
// [workspace] section itself (members list, etc.) is structural.
func TestAnalyzeCargoTomlChange_ForceBlastOnWorkspaceSection(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[workspace]"},
			Added:   []string{`members = ["foo", "bar"]`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if !got.ForceBlast {
		t.Error("[workspace].members change should force blast")
	}
}

// TestAnalyzeCargoTomlChange_AmbiguousHunkFallsBackToShape — no
// section header in the hunk; a dep-shaped line still registers.
func TestAnalyzeCargoTomlChange_AmbiguousHunkFallsBackToShape(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Removed: []string{`reqwest = "0.11"`},
			Added:   []string{`reqwest = "0.12"`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if len(got.AffectedDeps) != 1 || got.AffectedDeps[0] != "reqwest" {
		t.Errorf("ambiguous section: got %v, want [reqwest]", got.AffectedDeps)
	}
}

// TestAnalyzeCargoTomlChange_CommentedOutIgnored — a line that's
// purely a comment doesn't contribute a dep.
func TestAnalyzeCargoTomlChange_CommentedOutIgnored(t *testing.T) {
	d := FileDiff{
		Path: "Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[dependencies]"},
			Added:   []string{`# serde = "1.0" — temporarily disabled`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if len(got.AffectedDeps) != 0 {
		t.Errorf("comment-only line: got %v, want []", got.AffectedDeps)
	}
}

// TestAnalyzeCargoTomlChange_NonCargoToml — called on a non-
// Cargo.toml, returns zero.
func TestAnalyzeCargoTomlChange_NonCargoToml(t *testing.T) {
	d := FileDiff{
		Path: "src/main.rs",
		Hunks: []Hunk{{
			Added: []string{`serde = "1"`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if len(got.AffectedDeps) != 0 || got.ForceBlast {
		t.Errorf("non-Cargo.toml: got %+v, want empty", got)
	}
}

// TestAnalyzeCargoTomlChange_NestedCargoToml — works on deeply
// nested Cargo.toml paths.
func TestAnalyzeCargoTomlChange_NestedCargoToml(t *testing.T) {
	d := FileDiff{
		Path: "rust/kona/crates/derive/Cargo.toml",
		Hunks: []Hunk{{
			Context: []string{"[dependencies]"},
			Added:   []string{`alloy-primitives = "0.8"`},
		}},
	}
	got := AnalyzeCargoTomlChange(d)
	if len(got.AffectedDeps) != 1 {
		t.Errorf("nested path: got %v, want one dep", got.AffectedDeps)
	}
}
