package golang

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"golang.org/x/mod/modfile"
)

// GoAdapter builds a package-level import graph using `go list -json ./...`.
//
// In addition to the intra-module package nodes and import edges, the
// adapter emits `mod:<path>` nodes for every module declared in
// go.mod's require block and `imports` edges from each intra-module
// package to the external modules it imports (resolved via longest-
// prefix match against the require list).
//
// This lets go.mod changes feed into Phase 1 as reverse-walk roots:
// a go.mod version bump for module M becomes "changed node mod:M",
// and transitiveConsumers(g, ["mod:M"]) finds every package that
// imports M via the same incoming-edge walk that test-helper changes
// use.
type GoAdapter struct{}

// New returns a new GoAdapter.
func New() *GoAdapter { return &GoAdapter{} }

// Name returns "go".
func (a *GoAdapter) Name() string { return "go" }

// goPackage represents the JSON output of `go list -json`.
type goPackage struct {
	ImportPath  string   `json:"ImportPath"`
	Dir         string   `json:"Dir"`
	GoFiles     []string `json:"GoFiles"`
	Imports     []string `json:"Imports"`
	TestGoFiles []string `json:"TestGoFiles"`
	TestImports []string `json:"TestImports"`
	Module      *struct {
		Path string `json:"Path"`
	} `json:"Module"`
}

// modInfo holds parsed go.mod state used to classify imports.
type modInfo struct {
	ModulePath       string   // this repo's module path
	RequiredModules  []string // direct + indirect requires, sorted longest-first
}

// Analyze discovers every in-tree Go module (go.mod file) under
// rootDir, then for each module runs `go list -json ./...`, creates a
// source node per package within that module, and creates import edges
// between packages plus `mod:` nodes + external-import edges per
// required module.
//
// Multi-module repos (op-stack has a nested `ops/checks/go.mod`, plus
// submodules) were previously invisible because a single
// `go list ./...` at the root only sees packages in the root module.
// Missing nodes meant triggers couldn't match ops/checks/ file paths,
// so diffs under those sub-modules selected nothing. Walking once and
// analyzing each module fixes that without requiring any cross-module
// import resolution: internal/external classification uses the union
// of every in-tree module's packages.
func (a *GoAdapter) Analyze(g *graph.Graph, rootDir string) error {
	modDirs, err := findGoModules(rootDir)
	if err != nil {
		return fmt.Errorf("finding go.mod files: %w", err)
	}
	if len(modDirs) == 0 {
		return fmt.Errorf("no go.mod found under %s", rootDir)
	}

	type moduleData struct {
		dir      string
		info     *modInfo
		packages []goPackage
	}
	var modules []moduleData
	for _, dir := range modDirs {
		info, err := readGoMod(dir)
		if err != nil {
			return fmt.Errorf("reading go.mod in %s: %w", dir, err)
		}
		pkgs, err := listPackages(dir)
		if err != nil {
			return fmt.Errorf("listing packages in %s: %w", dir, err)
		}
		modules = append(modules, moduleData{dir: dir, info: info, packages: pkgs})
	}

	// Union of internal package import paths across every in-tree
	// module. Cross-module imports from one in-tree module to another
	// resolve as internal.
	allInternalPackages := make(map[string]bool)
	for _, m := range modules {
		for _, pkg := range m.packages {
			if strings.HasPrefix(pkg.ImportPath, m.info.ModulePath) {
				allInternalPackages[pkg.ImportPath] = true
			}
		}
	}

	// Emit mod: nodes for each required external, deduped across modules.
	modNodesEmitted := make(map[string]bool)
	for _, m := range modules {
		for _, mp := range m.info.RequiredModules {
			if modNodesEmitted[mp] {
				continue
			}
			modNodesEmitted[mp] = true
			_ = g.AddNode(&graph.Node{
				ID:          "mod:" + mp,
				Kind:        graph.KindModule,
				Granularity: "module",
				Name:        mp,
			})
		}
	}

	// Emit package + file nodes per module.
	for _, m := range modules {
		for _, pkg := range m.packages {
			if !strings.HasPrefix(pkg.ImportPath, m.info.ModulePath) {
				continue
			}
			pkgNodeID := "go:" + pkg.ImportPath
			_ = g.AddNode(&graph.Node{
				ID:          pkgNodeID,
				Kind:        graph.KindSource,
				Granularity: "package",
				Name:        pkg.ImportPath,
				Properties: map[string]any{
					"dir":        pkg.Dir,
					"file_count": len(pkg.GoFiles),
					"has_tests":  len(pkg.TestGoFiles) > 0,
				},
			})

			for _, files := range [][]string{pkg.GoFiles, pkg.TestGoFiles} {
				for _, file := range files {
					_ = g.AddNode(&graph.Node{
						ID:          pkgNodeID + "/" + file,
						Kind:        graph.KindSource,
						Granularity: "file",
						Name:        pkg.ImportPath + "/" + file,
						Properties: map[string]any{
							"package": pkgNodeID,
							"dir":     pkg.Dir,
						},
					})
				}
			}
		}
	}

	// Emit edges per module. Imports are classified against the union
	// of internal packages, so a cross-module internal import still
	// produces a package-to-package edge.
	for _, m := range modules {
		for _, pkg := range m.packages {
			if !strings.HasPrefix(pkg.ImportPath, m.info.ModulePath) {
				continue
			}
			fromID := "go:" + pkg.ImportPath
			for _, imp := range pkg.Imports {
				addImportEdge(g, fromID, imp, allInternalPackages, m.info, false)
			}
			for _, imp := range pkg.TestImports {
				addImportEdge(g, fromID, imp, allInternalPackages, m.info, true)
			}
		}
	}

	return nil
}

// findGoModules walks rootDir looking for go.mod files. Skips
// testdata/, node_modules/, .git/, .bare/, and any path under lib/
// (for git-submodule vendored code). Returns the parent directory
// of each go.mod (i.e. the module root).
func findGoModules(rootDir string) ([]string, error) {
	var dirs []string
	skipParts := map[string]bool{
		"testdata": true, "node_modules": true, ".git": true, ".bare": true,
	}
	err := filepath.Walk(rootDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if skipParts[fi.Name()] {
				return filepath.SkipDir
			}
			// Skip git-submodule areas and git-worktree internal bare clones.
			// Both are independent repos that can have their own go.mod we don't own.
			if fi.Name() == "lib" || fi.Name() == "worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.Name() == "go.mod" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}

// addImportEdge classifies `imp` as internal, external-to-module, or
// stdlib/unresolved and emits the appropriate edge (if any). Internal
// imports go to `go:<pkg>`; external imports go to `mod:<module>`;
// stdlib and unresolved imports are skipped.
func addImportEdge(g *graph.Graph, fromID, imp string, modulePackages map[string]bool, info *modInfo, isTest bool) {
	strength := 0.8
	if isTest {
		strength = 0.7
	}

	if modulePackages[imp] {
		props := map[string]any{}
		if isTest {
			props["test_import"] = true
		}
		var ePropsPtr map[string]any
		if len(props) > 0 {
			ePropsPtr = props
		}
		_ = g.AddEdge(&graph.Edge{
			From:       fromID,
			To:         "go:" + imp,
			Kind:       graph.EdgeImports,
			Source:     graph.SourceStatic,
			Confidence: 1.0,
			Strength:   strength,
			Properties: ePropsPtr,
		})
		return
	}

	if mod := resolveExternalModule(imp, info.RequiredModules); mod != "" {
		props := map[string]any{"import_path": imp}
		if isTest {
			props["test_import"] = true
		}
		_ = g.AddEdge(&graph.Edge{
			From:       fromID,
			To:         "mod:" + mod,
			Kind:       graph.EdgeImports,
			Source:     graph.SourceStatic,
			Confidence: 1.0,
			Strength:   strength,
			Properties: props,
		})
	}
	// else: stdlib or transitive-only dep not listed in go.mod; skip.
}

// resolveExternalModule maps an import path to the module that owns
// it via longest-prefix match against go.mod's require list. Returns
// "" for stdlib or unresolved imports.
func resolveExternalModule(importPath string, requiredModules []string) string {
	for _, m := range requiredModules {
		if importPath == m || strings.HasPrefix(importPath, m+"/") {
			return m
		}
	}
	return ""
}

// readGoMod parses go.mod with x/mod/modfile and returns the module
// path plus the require list sorted longest-first (so resolveExternal
// Module's first match is the most specific).
func readGoMod(rootDir string) (*modInfo, error) {
	modFile := filepath.Join(rootDir, "go.mod")
	data, err := os.ReadFile(modFile)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}
	info := &modInfo{}
	if f.Module != nil {
		info.ModulePath = f.Module.Mod.Path
	}
	for _, r := range f.Require {
		info.RequiredModules = append(info.RequiredModules, r.Mod.Path)
	}
	sort.Slice(info.RequiredModules, func(i, j int) bool {
		return len(info.RequiredModules[i]) > len(info.RequiredModules[j])
	})
	return info, nil
}

func listPackages(rootDir string) ([]goPackage, error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = rootDir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	var packages []goPackage
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	for decoder.More() {
		var pkg goPackage
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decoding package: %w", err)
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}
