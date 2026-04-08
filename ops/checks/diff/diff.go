package diff

import (
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// blastRadiusPatterns are file patterns that affect everything.
// Aligned with ops/check-changed/main.py REBUILD_ALL_PATTERNS + GO_PATTERNS.
var blastRadiusPatterns = []string{
	".circleci/",
	".github/",
	"package.json",
	"mise.toml",
	"ops/check-changed/",
	"go.mod",
	"go.sum",
	"foundry.toml",
	"Dockerfile",
}

// ChangedFiles parses `git diff --name-only` output into a list of file paths.
func ChangedFiles(diffOutput string) []string {
	var files []string
	for _, line := range strings.Split(diffOutput, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// FilesToNodeIDs maps changed file paths to graph node IDs.
func FilesToNodeIDs(g *graph.Graph, files []string) (nodeIDs []string, unknown []string) {
	seen := make(map[string]bool)

	for _, f := range files {
		matched := false

		// Try Go: map file to its package node
		if strings.HasSuffix(f, ".go") {
			dir := filepath.Dir(f)
			// Search for a go: node whose directory matches
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if !strings.HasPrefix(node.ID, "go:") {
					continue
				}
				nodeDir, _ := node.Properties["dir"].(string)
				if nodeDir != "" && strings.HasSuffix(nodeDir, "/"+dir) {
					if !seen[node.ID] {
						nodeIDs = append(nodeIDs, node.ID)
						seen[node.ID] = true
					}
					matched = true
					break
				}
				// Fallback: match by package name in path
				goPath := strings.TrimPrefix(node.ID, "go:")
				if matchesPath(goPath, dir) {
					if !seen[node.ID] {
						nodeIDs = append(nodeIDs, node.ID)
						seen[node.ID] = true
					}
					matched = true
					break
				}
			}
		}

		// Try Solidity: map file to sol: node
		if strings.HasSuffix(f, ".sol") {
			// Try with and without "packages/contracts-bedrock/" prefix
			candidates := []string{f}
			if strings.HasPrefix(f, "packages/contracts-bedrock/") {
				candidates = append(candidates, strings.TrimPrefix(f, "packages/contracts-bedrock/"))
			}
			for _, c := range candidates {
				nodeID := "sol:" + c
				if g.GetNode(nodeID) != nil && !seen[nodeID] {
					nodeIDs = append(nodeIDs, nodeID)
					seen[nodeID] = true
					matched = true
					break
				}
			}
		}

		if !matched {
			unknown = append(unknown, f)
		}
	}

	return
}

// BlastRadiusFiles returns true if any of the changed files are "blast radius everything" files.
func BlastRadiusFiles(files []string) (bool, []string) {
	var matches []string
	for _, f := range files {
		for _, pattern := range blastRadiusPatterns {
			if strings.HasPrefix(f, pattern) || f == pattern || strings.TrimSuffix(pattern, "/") == f {
				matches = append(matches, f)
				break
			}
		}
	}
	return len(matches) > 0, matches
}

func matchesPath(goImportPath, relDir string) bool {
	// Check if the Go import path ends with the relative directory path
	return strings.HasSuffix(goImportPath, "/"+relDir) || strings.HasSuffix(goImportPath, relDir)
}
