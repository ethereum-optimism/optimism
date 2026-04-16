package rustmeta

import (
	"errors"
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
