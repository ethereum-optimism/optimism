package coverage

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// IngestReports reads coverage reports and creates `covers` edges in the graph.
// Each report creates edges from the test node to the source nodes it covers,
// with line ranges stored as edge properties.
func IngestReports(g *graph.Graph, reports []*Report) error {
	for _, r := range reports {
		testNodeID := testToNodeID(r.Test, r.Language)

		// Ensure the test node exists (it might not if the graph was built without this test)
		if g.GetNode(testNodeID) == nil {
			continue
		}

		for sourceFile, ranges := range r.Covers {
			sourceNodeID := sourceToNodeID(sourceFile, r.Language)
			if g.GetNode(sourceNodeID) == nil {
				continue
			}

			// Create a covers edge from test → source with line ranges.
			// Profile lets the optimizer pick the right profile(s) when
			// scoping tests. Freshness stamps (test_sha, source_sha,
			// generated_at) let the selector down-weight this edge if
			// the underlying content has changed since collection.
			props := map[string]any{
				"line_ranges": ranges,
			}
			if r.Profile != "" {
				props["profile"] = r.Profile
			}
			if r.GeneratedAt != "" {
				props["generated_at"] = r.GeneratedAt
			}
			if r.TestSha != "" {
				props["test_sha"] = r.TestSha
			}
			if sha, ok := r.SourceShas[sourceFile]; ok && sha != "" {
				props["source_sha"] = sha
			}
			_ = g.AddEdge(&graph.Edge{
				From:       testNodeID,
				To:         sourceNodeID,
				Kind:       graph.EdgeTestedBy,
				Source:     graph.SourceCoverage,
				Confidence: 1.0,
				Strength:   coverageStrength(ranges),
				Properties: props,
			})
		}
	}
	return nil
}

// coverageStrength computes a signal strength from line ranges.
// More covered lines = higher strength. Capped at 1.0.
func coverageStrength(ranges [][2]int) float64 {
	totalLines := 0
	for _, r := range ranges {
		totalLines += r[1] - r[0] + 1
	}
	// Heuristic: 1 line = 0.1, 10+ lines = 1.0
	s := float64(totalLines) * 0.1
	if s > 1.0 {
		s = 1.0
	}
	return s
}

func testToNodeID(test, language string) string {
	switch language {
	case "solidity":
		return "sol:" + test
	case "go":
		return "go:" + strings.TrimSuffix(strings.TrimPrefix(test, "./"), "/...")
	case "rust":
		return "rs:" + test
	}
	return test
}

func sourceToNodeID(source, language string) string {
	switch language {
	case "solidity":
		return "sol:" + source
	case "go":
		return "go:" + source
	case "rust":
		return "rs:" + source
	}
	return source
}

// ComputeOverlaps analyzes coverage reports and computes overlap fractions
// between test pairs. Returns overlap data that can be added as edges.
type Overlap struct {
	TestA    string
	TestB    string
	Fraction float64 // what fraction of A's coverage overlaps with B's
}

// FindOverlaps computes pairwise coverage overlap between reports.
func FindOverlaps(reports []*Report) []Overlap {
	// Build line sets per test
	type lineKey struct {
		file string
		line int
	}

	testLines := make(map[string]map[lineKey]bool)
	for _, r := range reports {
		lines := make(map[lineKey]bool)
		for file, ranges := range r.Covers {
			for _, rng := range ranges {
				for l := rng[0]; l <= rng[1]; l++ {
					lines[lineKey{file, l}] = true
				}
			}
		}
		testLines[r.Test] = lines
	}

	// Compute pairwise overlap
	var overlaps []Overlap
	tests := make([]string, 0, len(testLines))
	for t := range testLines {
		tests = append(tests, t)
	}

	for i := 0; i < len(tests); i++ {
		for j := i + 1; j < len(tests); j++ {
			a, b := tests[i], tests[j]
			linesA, linesB := testLines[a], testLines[b]

			shared := 0
			for k := range linesA {
				if linesB[k] {
					shared++
				}
			}

			if shared > 0 {
				fractionA := float64(shared) / float64(len(linesA))
				fractionB := float64(shared) / float64(len(linesB))

				if fractionA > 0.1 {
					overlaps = append(overlaps, Overlap{
						TestA: a, TestB: b, Fraction: fractionA,
					})
				}
				if fractionB > 0.1 {
					overlaps = append(overlaps, Overlap{
						TestA: b, TestB: a, Fraction: fractionB,
					})
				}
			}
		}
	}

	return overlaps
}
