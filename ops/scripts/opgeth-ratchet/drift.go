package main

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// collectDrift type-checks every package (including tests) matching patterns in dir and returns
// the errors of the upstream-clean packages of module modPath: those whose imports reach no
// package with a fork-API use in uses. Positions are relative to dir.
//
// A package's test files count only for its test variants, so a use in X's tests leaves X clean
// for its importers.
func collectDrift(ctx context.Context, dir, modPath string, uses map[Use]bool, buildFlags, patterns []string) ([]string, error) {
	forkUsers := make(map[string]bool)
	for u := range uses {
		forkUsers[u.Package] = true
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Context:    ctx,
		Dir:        dir,
		Tests:      true,
		BuildFlags: buildFlags,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	inModule := func(pkg *packages.Package) bool { return pkg.Module != nil && pkg.Module.Path == modPath }
	usesFork := func(pkg *packages.Package) bool {
		rel := relPackage(pkg, modPath)
		return forkUsers[rel] || (hasTestFiles(pkg) && forkUsers[rel+testsSuffix])
	}
	dirty := make(map[*packages.Package]bool)
	var reaches func(pkg *packages.Package) bool
	reaches = func(pkg *packages.Package) bool {
		d, ok := dirty[pkg]
		if ok {
			return d
		}
		dirty[pkg] = false // import cycles are type errors, reported by the package itself
		d = usesFork(pkg)
		for _, imp := range pkg.Imports {
			if d {
				break
			}
			d = inModule(imp) && reaches(imp)
		}
		dirty[pkg] = d
		return d
	}

	var clean []*packages.Package
	for _, pkg := range pkgs {
		if inModule(pkg) && !reaches(pkg) {
			clean = append(clean, pkg)
		}
	}
	// Errors in files that several variants of a package share are reported once.
	seen := make(map[string]bool)
	add := func(s string) { seen[s] = true }
	// The imports of a clean package are clean too.
	packages.Visit(clean, nil, func(pkg *packages.Package) {
		if !inModule(pkg) {
			// A dependency that fails leaves its importers type-checked against an incomplete API.
			for _, e := range pkg.Errors {
				add(fmt.Sprintf("%s: %s", pkg.ID, e))
			}
			return
		}
		for _, e := range pkg.Errors {
			if e.Kind == packages.TypeError || isExportBuildOutput(e) {
				continue // reported from pkg.TypeErrors below
			}
			add(fmt.Sprintf("%s: %s", pkg.ID, e))
		}
		for _, te := range pkg.TypeErrors {
			pos := te.Fset.Position(te.Pos)
			if rel, err := filepath.Rel(dir, pos.Filename); err == nil {
				pos.Filename = rel
			}
			add(fmt.Sprintf("%s: %s", pos, te.Msg))
		}
	})
	drift := make([]string, 0, len(seen))
	for s := range seen {
		drift = append(drift, s)
	}
	slices.Sort(drift)
	return drift, nil
}

func hasTestFiles(pkg *packages.Package) bool {
	return slices.ContainsFunc(pkg.GoFiles, func(f string) bool { return strings.HasSuffix(f, "_test.go") })
}
