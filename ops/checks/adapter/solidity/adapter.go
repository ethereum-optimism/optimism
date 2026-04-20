package solidity

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// SolidityAdapter parses Solidity import statements to build a file-level
// dependency graph. Uses foundry.toml remappings to resolve import paths.
type SolidityAdapter struct{}

// New returns a new SolidityAdapter.
func New() *SolidityAdapter { return &SolidityAdapter{} }

// Name returns "solidity".
func (a *SolidityAdapter) Name() string { return "solidity" }

// importPattern matches Solidity import statements.
var importPattern = regexp.MustCompile(`import\s+(?:\{[^}]*\}\s+from\s+)?["']([^"']+)["']`)

// Analyze walks the Solidity project directory, finds all .sol files,
// parses import statements, and creates source nodes and import edges.
func (a *SolidityAdapter) Analyze(g *graph.Graph, rootDir string) error {
	// Find the contracts-bedrock directory (or use rootDir if it contains .sol files)
	solDir := rootDir
	contractsDir := filepath.Join(rootDir, "packages", "contracts-bedrock")
	if info, err := os.Stat(contractsDir); err == nil && info.IsDir() {
		solDir = contractsDir
	}

	// Read remappings from foundry.toml
	remappings := readRemappings(solDir)

	// Scan directories for .sol files
	scanDirs := []string{"src", "test", "scripts", "interfaces"}
	var solFiles []string
	for _, dir := range scanDirs {
		fullDir := filepath.Join(solDir, dir)
		if _, err := os.Stat(fullDir); err != nil {
			continue
		}
		filepath.Walk(fullDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".sol") {
				solFiles = append(solFiles, path)
			}
			return nil
		})
	}

	// Create nodes for all .sol files
	for _, path := range solFiles {
		relPath, err := filepath.Rel(solDir, path)
		if err != nil {
			continue
		}
		nodeID := "sol:" + relPath
		_ = g.AddNode(&graph.Node{
			ID:          nodeID,
			Kind:        graph.KindSource,
			Granularity: "file",
			Name:        relPath,
			Properties: map[string]any{
				"language": "solidity",
				"abs_path": path,
			},
		})
	}

	// Parse imports and create edges
	for _, path := range solFiles {
		relPath, _ := filepath.Rel(solDir, path)
		fromID := "sol:" + relPath

		imports := parseImports(path)
		for _, imp := range imports {
			resolved := resolveImport(imp, relPath, remappings)
			toID := "sol:" + resolved

			// Only create edge if target node exists
			if g.GetNode(toID) != nil {
				_ = g.AddEdge(&graph.Edge{
					From:       fromID,
					To:         toID,
					Kind:       graph.EdgeImports,
					Source:     graph.SourceStatic,
					Confidence: 1.0,
					Strength:   0.9,
				})
			}
		}
	}

	// Cross-language dataflow (src → generated-interfaces, src → Go
	// bindings) is now handled by gen-* check_types declaring those
	// artifacts as outputs, plus the builder's bridge-imports step.

	return nil
}


// parseImports reads a Solidity file and extracts import paths.
func parseImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := importPattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				imports = append(imports, m[1])
			}
		}
	}
	return imports
}

// resolveImport resolves an import path using remappings.
//
// Foundry remappings (e.g. `interfaces/=interfaces`,
// `forge-std/=lib/forge-std/src`) have a prefix that usually ends in
// `/` and a target that usually doesn't. Foundry resolves these as
// directory joins, not string concatenation — so `interfaces/L1/IFoo`
// with target `interfaces` becomes `interfaces/L1/IFoo`, not
// `interfacesL1/IFoo`. filepath.Join handles the slash correctly.
func resolveImport(importPath, fromFile string, remappings map[string]string) string {
	// Try remappings first
	for prefix, target := range remappings {
		if strings.HasPrefix(importPath, prefix) {
			rest := strings.TrimPrefix(importPath, prefix)
			return filepath.Join(target, rest)
		}
	}

	// Relative imports
	if strings.HasPrefix(importPath, ".") {
		dir := filepath.Dir(fromFile)
		resolved := filepath.Join(dir, importPath)
		return filepath.Clean(resolved)
	}

	return importPath
}

// readRemappings reads import remappings from foundry.toml.
func readRemappings(solDir string) map[string]string {
	remappings := make(map[string]string)

	// Try foundry.toml first
	tomlPath := filepath.Join(solDir, "foundry.toml")
	if data, err := os.ReadFile(tomlPath); err == nil {
		// Simple parsing of remappings array from foundry.toml
		// Looks for lines like 'remappings = [' or individual entries
		lines := strings.Split(string(data), "\n")
		inRemappings := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "remappings") {
				inRemappings = true
				continue
			}
			if inRemappings {
				if line == "]" {
					break
				}
				// Parse 'prefix/=target/' entries
				line = strings.Trim(line, "' \",")
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					remappings[parts[0]] = parts[1]
				}
			}
		}
	}

	// Fall back to remappings.txt
	if len(remappings) == 0 {
		remapPath := filepath.Join(solDir, "remappings.txt")
		if data, err := os.ReadFile(remapPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					remappings[parts[0]] = parts[1]
				}
			}
		}
	}

	return remappings
}
