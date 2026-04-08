package coverage

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// IngestGoCoverage reads a Go coverage profile (from `go test -coverprofile`)
// and creates precise tested_by edges from source packages to test packages.
func IngestGoCoverage(g *graph.Graph, profilePath string) error {
	f, err := os.Open(profilePath)
	if err != nil {
		return fmt.Errorf("opening coverage profile: %w", err)
	}
	defer f.Close()

	// Coverage profile format:
	// mode: set|count|atomic
	// file:startLine.startCol,endLine.endCol statements count
	// e.g.: github.com/org/repo/pkg/file.go:10.2,20.5 3 1

	coveredPackages := make(map[string]int) // package path -> covered statement count
	totalPackages := make(map[string]int)   // package path -> total statement count

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse: file:start,end statements count
		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}
		filePath := line[:colonIdx]

		// Extract package path from file path
		lastSlash := strings.LastIndex(filePath, "/")
		if lastSlash < 0 {
			continue
		}
		pkgPath := filePath[:lastSlash]

		// Parse count (last field)
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 3 {
			continue
		}

		totalPackages[pkgPath]++
		if fields[2] != "0" {
			coveredPackages[pkgPath]++
		}
	}

	// Create tested_by edges with strength proportional to coverage
	for pkgPath, covered := range coveredPackages {
		total := totalPackages[pkgPath]
		if total == 0 {
			continue
		}
		strength := float64(covered) / float64(total)

		fromID := "go:" + pkgPath
		// Find check nodes that test this package
		for _, edge := range g.EdgesTo(fromID) {
			if edge.Kind == graph.EdgeTestedBy {
				continue
			}
		}

		// Add coverage edge from source to any check that covers it
		// The check ID is derived from the package path
		// This is a best-effort mapping — the builder wires more precisely
		_ = g.AddEdge(&graph.Edge{
			From:       fromID,
			To:         fromID, // Placeholder — builder resolves actual check ID
			Kind:       graph.EdgeTestedBy,
			Source:     graph.SourceCoverage,
			Confidence: 1.0,
			Strength:   strength,
			Properties: map[string]any{
				"covered_statements": covered,
				"total_statements":   total,
			},
		})
	}

	return scanner.Err()
}

// IngestForgeCoverage reads a Forge LCOV coverage report and creates
// tested_by edges from Solidity source files to test files.
func IngestForgeCoverage(g *graph.Graph, lcovPath string) error {
	f, err := os.Open(lcovPath)
	if err != nil {
		return fmt.Errorf("opening LCOV report: %w", err)
	}
	defer f.Close()

	// LCOV format:
	// SF:path/to/file.sol
	// DA:lineNum,hitCount
	// ...
	// end_of_record

	var currentFile string
	var totalLines, coveredLines int

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "SF:") {
			currentFile = strings.TrimPrefix(line, "SF:")
			totalLines = 0
			coveredLines = 0
		} else if strings.HasPrefix(line, "DA:") {
			totalLines++
			parts := strings.SplitN(strings.TrimPrefix(line, "DA:"), ",", 2)
			if len(parts) == 2 && parts[1] != "0" {
				coveredLines++
			}
		} else if line == "end_of_record" && currentFile != "" {
			if totalLines > 0 {
				strength := float64(coveredLines) / float64(totalLines)
				fromID := "sol:" + currentFile
				if g.GetNode(fromID) != nil {
					// Store coverage data as node property for later use by scorer
					node := g.GetNode(fromID)
					if node.Properties == nil {
						node.Properties = make(map[string]any)
					}
					node.Properties["coverage_strength"] = strength
					node.Properties["covered_lines"] = coveredLines
					node.Properties["total_lines"] = totalLines
				}
			}
			currentFile = ""
		}
	}

	return scanner.Err()
}
