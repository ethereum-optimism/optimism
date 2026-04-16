package diff

import (
	"reflect"
	"testing"
)

// TestAnalyzeGoModChange_VersionBump — single-line require version bump
// produces one affected module, no blast.
func TestAnalyzeGoModChange_VersionBump(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Removed: []string{"	github.com/ethereum/go-ethereum v1.14.7"},
			Added:   []string{"	github.com/ethereum/go-ethereum v1.14.8"},
		}},
	}
	got := AnalyzeGoModChange(d)
	want := []string{"github.com/ethereum/go-ethereum"}
	if !reflect.DeepEqual(got.AffectedModules, want) {
		t.Errorf("AffectedModules = %v, want %v", got.AffectedModules, want)
	}
	if got.ForceBlast {
		t.Error("version bump should not force blast")
	}
}

// TestAnalyzeGoModChange_AddAndRemove — adding and removing modules
// both register.
func TestAnalyzeGoModChange_AddAndRemove(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Removed: []string{"	github.com/old/dep v1.0.0"},
			Added:   []string{"	github.com/new/dep v2.0.0 // indirect"},
		}},
	}
	got := AnalyzeGoModChange(d)
	wantSet := map[string]bool{
		"github.com/old/dep": true,
		"github.com/new/dep": true,
	}
	for _, m := range got.AffectedModules {
		if !wantSet[m] {
			t.Errorf("unexpected module %q", m)
		}
		delete(wantSet, m)
	}
	if len(wantSet) > 0 {
		t.Errorf("missing modules: %v", wantSet)
	}
}

// TestAnalyzeGoModChange_IgnoresIndirectComment — the "// indirect"
// trailer shouldn't swallow the module path.
func TestAnalyzeGoModChange_IgnoresIndirectComment(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Added: []string{"	github.com/foo/bar v1.0.0 // indirect"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if len(got.AffectedModules) != 1 || got.AffectedModules[0] != "github.com/foo/bar" {
		t.Errorf("got %v, want [github.com/foo/bar]", got.AffectedModules)
	}
}

// TestAnalyzeGoModChange_SkipsBlockSyntax — `(`, `)`, blank, and
// keyword-only lines don't produce modules.
func TestAnalyzeGoModChange_SkipsBlockSyntax(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Added: []string{
				"require (",
				"",
				")",
				"	github.com/real/mod v1.0.0",
			},
		}},
	}
	got := AnalyzeGoModChange(d)
	if len(got.AffectedModules) != 1 || got.AffectedModules[0] != "github.com/real/mod" {
		t.Errorf("got %v, want only the real module", got.AffectedModules)
	}
	if got.ForceBlast {
		t.Error("should not force blast for pure require-block changes")
	}
}

// TestAnalyzeGoModChange_ForceBlastOnGoVersion — go version bumps
// force blast-radius.
func TestAnalyzeGoModChange_ForceBlastOnGoVersion(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Removed: []string{"go 1.23.0"},
			Added:   []string{"go 1.24.0"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if !got.ForceBlast {
		t.Error("go version change should force blast")
	}
}

// TestAnalyzeGoModChange_ForceBlastOnToolchain — toolchain directive
// changes force blast.
func TestAnalyzeGoModChange_ForceBlastOnToolchain(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Added: []string{"toolchain go1.24.1"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if !got.ForceBlast {
		t.Error("toolchain change should force blast")
	}
}

// TestAnalyzeGoModChange_ForceBlastOnModuleLine — repo rename /
// module path change forces blast.
func TestAnalyzeGoModChange_ForceBlastOnModuleLine(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Removed: []string{"module github.com/old/repo"},
			Added:   []string{"module github.com/new/repo"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if !got.ForceBlast {
		t.Error("module path change should force blast")
	}
}

// TestAnalyzeGoModChange_Replace — replace directive records LHS.
func TestAnalyzeGoModChange_Replace(t *testing.T) {
	d := FileDiff{
		Path: "go.mod",
		Hunks: []Hunk{{
			Added: []string{"replace github.com/orig/mod => github.com/fork/mod v0.1.0"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if len(got.AffectedModules) != 1 || got.AffectedModules[0] != "github.com/orig/mod" {
		t.Errorf("got %v, want [github.com/orig/mod]", got.AffectedModules)
	}
}

// TestAnalyzeGoModChange_NestedGoMod — handles go.mod at nested paths.
func TestAnalyzeGoModChange_NestedGoMod(t *testing.T) {
	d := FileDiff{
		Path: "some/nested/module/go.mod",
		Hunks: []Hunk{{
			Added: []string{"	github.com/x/y v1"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if len(got.AffectedModules) != 1 {
		t.Errorf("nested go.mod should still analyze; got %v", got.AffectedModules)
	}
}

// TestAnalyzeGoModChange_NonGoMod — called on a non-go.mod file,
// returns empty.
func TestAnalyzeGoModChange_NonGoMod(t *testing.T) {
	d := FileDiff{
		Path: "src/main.go",
		Hunks: []Hunk{{
			Added: []string{"	github.com/x/y v1"},
		}},
	}
	got := AnalyzeGoModChange(d)
	if len(got.AffectedModules) != 0 || got.ForceBlast {
		t.Errorf("non-go.mod should be a no-op, got %+v", got)
	}
}
