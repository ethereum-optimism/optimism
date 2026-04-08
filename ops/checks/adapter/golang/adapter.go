package golang

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// GoAdapter builds a package-level import graph using `go list -json ./...`.
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

// Analyze runs `go list -json ./...` from rootDir, creates a source node for
// each package within the module, and creates import edges between packages.
func (a *GoAdapter) Analyze(g *graph.Graph, rootDir string) error {
	modulePath, err := readModulePath(rootDir)
	if err != nil {
		return fmt.Errorf("reading module path: %w", err)
	}

	packages, err := listPackages(rootDir)
	if err != nil {
		return fmt.Errorf("listing packages: %w", err)
	}

	// Track which packages exist in this module
	modulePackages := make(map[string]bool)
	for _, pkg := range packages {
		if strings.HasPrefix(pkg.ImportPath, modulePath) {
			modulePackages[pkg.ImportPath] = true
		}
	}

	// Create nodes
	for _, pkg := range packages {
		if !strings.HasPrefix(pkg.ImportPath, modulePath) {
			continue
		}

		nodeID := "go:" + pkg.ImportPath
		err := g.AddNode(&graph.Node{
			ID:          nodeID,
			Kind:        graph.KindSource,
			Granularity: "package",
			Name:        pkg.ImportPath,
			Properties: map[string]any{
				"dir":        pkg.Dir,
				"file_count": len(pkg.GoFiles),
				"has_tests":  len(pkg.TestGoFiles) > 0,
			},
		})
		if err != nil {
			// Skip duplicate (already added)
			continue
		}
	}

	// Create import edges
	for _, pkg := range packages {
		if !strings.HasPrefix(pkg.ImportPath, modulePath) {
			continue
		}
		fromID := "go:" + pkg.ImportPath

		// Regular imports
		for _, imp := range pkg.Imports {
			if !modulePackages[imp] {
				continue
			}
			toID := "go:" + imp
			_ = g.AddEdge(&graph.Edge{
				From:       fromID,
				To:         toID,
				Kind:       graph.EdgeImports,
				Source:     graph.SourceStatic,
				Confidence: 1.0,
				Strength:   0.8,
			})
		}

		// Test imports
		for _, imp := range pkg.TestImports {
			if !modulePackages[imp] {
				continue
			}
			toID := "go:" + imp
			_ = g.AddEdge(&graph.Edge{
				From:       fromID,
				To:         toID,
				Kind:       graph.EdgeImports,
				Source:     graph.SourceStatic,
				Confidence: 1.0,
				Strength:   0.7,
				Properties: map[string]any{"test_import": true},
			})
		}
	}

	return nil
}

func readModulePath(rootDir string) (string, error) {
	modFile := filepath.Join(rootDir, "go.mod")
	data, err := os.ReadFile(modFile)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
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
