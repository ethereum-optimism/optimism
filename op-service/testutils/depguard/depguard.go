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

// RequireNoTransitiveImportExcept asserts that no package matched by pattern
// transitively imports any of the forbidden import paths, except the packages
// named in allowed (by full package path).
//
// Prefer this over listing the guarded packages when the invariant should hold
// by default: pass a broad pattern such as "./..." and enumerate only the
// exceptions, so a newly added package is guarded automatically rather than
// silently unguarded.
//
// The allowlist is checked in both directions. An allowed package that no
// longer reaches any forbidden import is reported too, so entries cannot
// outlive the dependency that justified them.
func RequireNoTransitiveImportExcept(t testing.TB, pattern string, allowed []string, forbidden ...string) {
	t.Helper()
	offenders, staleAllowed, err := findForbiddenChainsExcept(pattern, allowed, forbidden...)
	require.NoError(t, err)
	require.Empty(t, offenders, "forbidden transitive import(s):\n%s", strings.Join(offenders, "\n"))
	require.Empty(t, staleAllowed,
		"allowlist entries that no longer reach a forbidden import — remove them:\n%s",
		strings.Join(staleAllowed, "\n"))
}

// findForbiddenChainsExcept walks each matched root package separately, so a
// package is judged on its own closure rather than one shared traversal.
func findForbiddenChainsExcept(pattern string, allowed []string, forbidden ...string) (offenders, staleAllowed []string, err error) {
	roots, err := loadRoots(pattern)
	if err != nil {
		return nil, nil, err
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}
	forbiddenSet := make(map[string]struct{}, len(forbidden))
	for _, f := range forbidden {
		forbiddenSet[f] = struct{}{}
	}

	usedAllowed := make(map[string]struct{}, len(allowed))
	seenRoot := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		if _, dup := seenRoot[root.PkgPath]; dup {
			continue
		}
		seenRoot[root.PkgPath] = struct{}{}

		chain := firstForbiddenChain(root, forbiddenSet)
		_, isAllowed := allowedSet[root.PkgPath]
		switch {
		case chain != "" && !isAllowed:
			offenders = append(offenders, chain)
		case chain != "" && isAllowed:
			usedAllowed[root.PkgPath] = struct{}{}
		}
	}

	for _, a := range allowed {
		if _, used := usedAllowed[a]; !used {
			staleAllowed = append(staleAllowed, "  "+a)
		}
	}
	sort.Strings(offenders)
	sort.Strings(staleAllowed)
	return offenders, staleAllowed, nil
}

// firstForbiddenChain returns an offending import chain rooted at root, or "".
func firstForbiddenChain(root *packages.Package, forbidden map[string]struct{}) string {
	seen := make(map[string]bool)
	var found string
	var walk func(p *packages.Package, trail []string)
	walk = func(p *packages.Package, trail []string) {
		if found != "" || seen[p.ID] {
			return
		}
		seen[p.ID] = true
		trail = append(trail, p.PkgPath)
		if _, ok := forbidden[p.PkgPath]; ok {
			found = strings.Join(trail, " -> ")
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
	walk(root, nil)
	return found
}

// loadRoots loads the packages matched by pattern, failing loudly rather than
// letting a mistyped pattern or a broken package silently pass a guard.
func loadRoots(pattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("pattern %s matched no packages", pattern)
	}
	for _, p := range pkgs {
		for _, e := range p.Errors {
			return nil, fmt.Errorf("loading %s: %s", p.PkgPath, e.Msg)
		}
	}
	return pkgs, nil
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
