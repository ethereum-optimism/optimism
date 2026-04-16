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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// Crate is one Cargo workspace member.
type Crate struct {
	Name        string
	ManifestDir string // absolute path to the directory containing Cargo.toml
	// Dependencies are the direct deps declared in this crate's
	// Cargo.toml — union of [dependencies], [dev-dependencies], and
	// [build-dependencies]. Names are the *package* names (respecting
	// the `package = "..."` aliasing convention). No classification
	// here — callers decide internal vs external by intersecting
	// against the workspace member set.
	Dependencies []string
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
		manifestDir := filepath.Dir(p.ManifestPath)
		deps, _ := parseCargoDependencies(p.ManifestPath)
		crates = append(crates, Crate{
			Name:         p.Name,
			ManifestDir:  manifestDir,
			Dependencies: deps,
		})
	}
	return crates, nil
}

// parseCargoDependencies reads a Cargo.toml and returns the union of
// [dependencies], [dev-dependencies], and [build-dependencies] as a
// sorted, de-duplicated list of package names. Respects the `package
// = "..."` aliasing convention (so `alloy = { package = "alloy-
// primitives" }` records "alloy-primitives", not "alloy").
//
// Returns (nil, nil) on read/parse failure — deps are best-effort; a
// malformed Cargo.toml shouldn't fail graph construction.
func parseCargoDependencies(manifestPath string) ([]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var m struct {
		Dependencies      map[string]any `toml:"dependencies"`
		DevDependencies   map[string]any `toml:"dev-dependencies"`
		BuildDependencies map[string]any `toml:"build-dependencies"`
	}
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, table := range []map[string]any{m.Dependencies, m.DevDependencies, m.BuildDependencies} {
		for name, val := range table {
			pkg := name
			// Aliased: name = { package = "...", ... }
			if sub, ok := val.(map[string]any); ok {
				if override, ok := sub["package"].(string); ok && override != "" {
					pkg = override
				}
			}
			seen[pkg] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
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
