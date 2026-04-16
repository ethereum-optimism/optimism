package rustmeta

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
)

// TestLoader_Caches — repeated Load calls for the same workDir invoke
// the fetcher once; the cached result is returned thereafter.
func TestLoader_Caches(t *testing.T) {
	var calls int32
	fetcher := func(workDir string) ([]Crate, error) {
		atomic.AddInt32(&calls, 1)
		return []Crate{{Name: "alpha", ManifestDir: "/r/alpha"}}, nil
	}
	l := NewLoaderWith(fetcher)

	for i := 0; i < 5; i++ {
		crates, err := l.Load("/workspace")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(crates) != 1 || crates[0].Name != "alpha" {
			t.Errorf("unexpected crates: %+v", crates)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("fetcher invoked %d times, want 1 (subsequent calls should hit cache)", got)
	}
}

// TestLoader_CachesPerWorkDir — separate workDirs don't collide; each
// gets its own cached entry.
func TestLoader_CachesPerWorkDir(t *testing.T) {
	var calls int32
	fetcher := func(workDir string) ([]Crate, error) {
		atomic.AddInt32(&calls, 1)
		return []Crate{{Name: workDir}}, nil
	}
	l := NewLoaderWith(fetcher)

	if _, err := l.Load("/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Load("/b"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Load("/a"); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("fetcher invoked %d times, want 2 (once per distinct workDir)", got)
	}
}

// TestParseCargoDependencies_UnionsAllTables — [dependencies],
// [dev-dependencies], and [build-dependencies] all contribute to the
// dep list; results are sorted + deduplicated.
func TestParseCargoDependencies_UnionsAllTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cargo.toml")
	contents := `
[package]
name = "x"
version = "0.1.0"

[dependencies]
serde = "1"
alloy-primitives = { version = "0.8" }

[dev-dependencies]
proptest = "1"
serde = { workspace = true }

[build-dependencies]
cc = "1"
`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := parseCargoDependencies(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"alloy-primitives", "cc", "proptest", "serde"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deps = %v, want %v", got, want)
	}
}

// TestParseCargoDependencies_AliasRespected — `alloy = { package =
// "alloy-primitives" }` records the real package name, not the alias.
func TestParseCargoDependencies_AliasRespected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cargo.toml")
	contents := `
[package]
name = "x"

[dependencies]
alloy = { package = "alloy-primitives", version = "0.8" }
`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, _ := parseCargoDependencies(path)
	want := []string{"alloy-primitives"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deps = %v, want %v (alias should map to real package)", got, want)
	}
}

// TestParseCargoDependencies_NoDeps — a manifest with no dep tables
// returns empty, not error.
func TestParseCargoDependencies_NoDeps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cargo.toml")
	if err := os.WriteFile(path, []byte("[package]\nname = \"x\"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := parseCargoDependencies(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("deps = %v, want empty", got)
	}
}

// TestLoader_PropagatesError — fetcher errors are returned and NOT
// cached. A subsequent successful fetcher call should succeed.
func TestLoader_PropagatesError(t *testing.T) {
	var calls int32
	fail := true
	fetcher := func(workDir string) ([]Crate, error) {
		atomic.AddInt32(&calls, 1)
		if fail {
			return nil, errors.New("boom")
		}
		return []Crate{{Name: "ok"}}, nil
	}
	l := NewLoaderWith(fetcher)

	if _, err := l.Load("/w"); err == nil {
		t.Error("expected error on first call")
	}
	fail = false
	if _, err := l.Load("/w"); err != nil {
		t.Errorf("expected success after recovery, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("fetcher invoked %d times, want 2 (error should not be cached)", got)
	}
}
