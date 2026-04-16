// Package rustmeta exposes Cargo workspace metadata in a single shape
// shared by the Rust adapter and the Rust coverage collector.
//
// Before this package existed, each consumer had its own `Crate` type
// and its own copy of the `cargo metadata` invocation. The adapter
// needed it once per graph build; the collector re-invoked it per
// test, which on a batch coverage run was hundreds of redundant
// subprocess calls.
//
// The package exposes a stateless Load for one-shot callers and a
// Loader for callers that benefit from caching across calls.
package rustmeta

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
)

// Crate is one Cargo workspace member.
type Crate struct {
	Name        string
	ManifestDir string // absolute path to the directory containing Cargo.toml
}

// Fetcher returns workspace members for a given absolute workspace
// directory. Used as an indirection point for tests.
type Fetcher func(workDir string) ([]Crate, error)

// Load runs `cargo metadata --no-deps --format-version 1` in workDir
// and returns the workspace members. Stateless — every call spawns
// cargo. Use Loader for repeated calls.
func Load(workDir string) ([]Crate, error) {
	cmd := exec.Command("cargo", "metadata", "--no-deps", "--format-version", "1")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo metadata: %w", err)
	}
	var meta struct {
		Packages []struct {
			Name         string `json:"name"`
			ManifestPath string `json:"manifest_path"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("parsing cargo metadata: %w", err)
	}
	crates := make([]Crate, 0, len(meta.Packages))
	for _, p := range meta.Packages {
		crates = append(crates, Crate{
			Name:        p.Name,
			ManifestDir: filepath.Dir(p.ManifestPath),
		})
	}
	return crates, nil
}

// Loader caches Load results per workDir. Safe for concurrent use.
// Construct with NewLoader (default fetcher) or NewLoaderWith (custom
// fetcher, for tests).
type Loader struct {
	fetcher Fetcher

	mu    sync.Mutex
	cache map[string][]Crate
}

// NewLoader returns a Loader backed by the default Load fetcher.
func NewLoader() *Loader {
	return &Loader{fetcher: Load}
}

// NewLoaderWith returns a Loader backed by the given Fetcher.
// Intended for tests.
func NewLoaderWith(fetcher Fetcher) *Loader {
	return &Loader{fetcher: fetcher}
}

// Load returns the workspace members at workDir, reusing a cached
// result if one exists. Cache is keyed by workDir — distinct
// workspaces don't collide.
func (l *Loader) Load(workDir string) ([]Crate, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if crates, ok := l.cache[workDir]; ok {
		return crates, nil
	}
	crates, err := l.fetcher(workDir)
	if err != nil {
		return nil, err
	}
	if l.cache == nil {
		l.cache = make(map[string][]Crate)
	}
	l.cache[workDir] = crates
	return crates, nil
}
