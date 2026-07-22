// Package depguard provides test helpers to assert that a package's transitive
// build closure stays free of forbidden imports.
package depguard

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// RequireNoTransitiveImport asserts that no package matched by pattern
// transitively imports any of the forbidden import paths. On failure it reports
// an offending import chain, e.g. "a -> b -> forbidden/pkg". Direct-import-only
// checks miss dependencies introduced through an intermediate package; this
// walks the full closure.
//
// Only the production build closure is walked: the guarded invariant protects
// packages that import the matched package, and they never build its test files.
func RequireNoTransitiveImport(t testing.TB, pattern string, forbidden ...string) {
	t.Helper()
	chains, err := findForbiddenChains(pattern, forbidden...)
	require.NoError(t, err)
	require.Empty(t, chains, "forbidden transitive import(s):\n%s", strings.Join(chains, "\n"))
}

func findForbiddenChains(pattern string, forbidden ...string) ([]string, error) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return []string{"pattern " + pattern + " matched no packages"}, nil
	}
	// A root that failed to load (e.g. a typo'd pattern yielding a stub package)
	// has an empty import graph and would silently pass the guard.
	for _, p := range pkgs {
		for _, e := range p.Errors {
			return nil, fmt.Errorf("loading %s: %s", p.PkgPath, e.Msg)
		}
	}

	forbiddenSet := make(map[string]struct{}, len(forbidden))
	for _, f := range forbidden {
		forbiddenSet[f] = struct{}{}
	}

	// A package already visited on a non-offending trail can be skipped: every
	// forbidden path reachable through it was found on the first visit.
	seen := make(map[string]bool)
	var chains []string
	var walk func(p *packages.Package, trail []string)
	walk = func(p *packages.Package, trail []string) {
		if seen[p.ID] {
			return
		}
		seen[p.ID] = true
		trail = append(trail, p.PkgPath)
		if _, ok := forbiddenSet[p.PkgPath]; ok {
			chains = append(chains, strings.Join(trail, " -> "))
			return
		}
		imports := make([]string, 0, len(p.Imports))
		for path := range p.Imports {
			imports = append(imports, path)
		}
		sort.Strings(imports)
		for _, path := range imports {
			walk(p.Imports[path], trail)
		}
	}
	for _, p := range pkgs {
		walk(p, nil)
	}
	return chains, nil
}
