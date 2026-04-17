package coverage

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// IngestReports reads coverage reports and creates `covers` edges in the graph.
// Each report creates edges from the test node to the source nodes it covers,
// with line ranges stored as edge properties.
//
// Node ID resolution is graph-aware: Go coverage reports identify
// packages as `./pkg/...` and sources as full import paths, while the
// Go adapter emits `go:<full-import-path>` nodes — so we look up by
// dir-suffix match. Rust reports use bare crate names + absolute
// filesystem paths, while the adapter emits `rs:<crate>` and
// `rs:<crate>/<rel>` nodes — so we look up by dir-prefix match. Naive
// string-prefix mapping produced zero Go/Rust coverage edges in the
// graph, which silently broke all Go/Rust scoping.
func IngestReports(g *graph.Graph, reports []*Report) error {
	index := newNodeIndex(g)

	for _, r := range reports {
		testNodeID := index.testToNodeID(r.Test, r.Language)
		if testNodeID == "" || g.GetNode(testNodeID) == nil {
			continue
		}

		for sourceFile, ranges := range r.Covers {
			sourceNodeID := index.sourceToNodeID(sourceFile, r.Language)
			if sourceNodeID == "" || g.GetNode(sourceNodeID) == nil {
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

// nodeIndex caches per-language dir-keyed maps of graph nodes so
// coverage ingestion can resolve report identifiers (./pkg/... for Go,
// crate-name/abs-path for Rust) to the adapter's ID format without
// an O(nodes) scan per lookup.
type nodeIndex struct {
	// Go: repo-relative dir ("op-node/rollup/derive") → go: package node ID.
	goPkgByDir map[string]string
	// Rust: crate absolute directory (prefix) → crate name. Sorted
	// longest-first so nested workspaces match the inner crate first.
	rustCrateDirs []rustCrate
}

type rustCrate struct {
	dir  string // absolute
	name string
}

func newNodeIndex(g *graph.Graph) *nodeIndex {
	idx := &nodeIndex{goPkgByDir: make(map[string]string)}
	for _, n := range g.NodesOfKind(graph.KindSource) {
		dir, _ := n.Properties["dir"].(string)
		if dir == "" {
			continue
		}
		switch {
		case strings.HasPrefix(n.ID, "go:") && n.Granularity != "file":
			// The dir property is the package's absolute path. We want
			// to key by repo-relative path; the simplest proxy is the
			// trailing portion of dir since Go coverage test IDs are
			// repo-relative. Store the whole absolute dir — lookup will
			// suffix-match.
			idx.goPkgByDir[dir] = n.ID
		case strings.HasPrefix(n.ID, "rs:") && n.Granularity == "crate":
			crateName := strings.TrimPrefix(n.ID, "rs:")
			idx.rustCrateDirs = append(idx.rustCrateDirs, rustCrate{dir: dir, name: crateName})
		}
	}
	// Sort Rust crate dirs longest-first so nested-workspace lookup
	// resolves to the innermost crate rather than a parent.
	sortByLenDesc(idx.rustCrateDirs)
	return idx
}

func sortByLenDesc(rs []rustCrate) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && len(rs[j].dir) > len(rs[j-1].dir); j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}

func (idx *nodeIndex) testToNodeID(test, language string) string {
	switch language {
	case "solidity":
		return "sol:" + test
	case "go":
		rel := strings.TrimSuffix(strings.TrimPrefix(test, "./"), "/...")
		// Find a go: node whose dir ends with "/" + rel.
		for dir, nodeID := range idx.goPkgByDir {
			if strings.HasSuffix(dir, "/"+rel) {
				return nodeID
			}
		}
		// Fallback: literal prefix (works if caller already passes a full import path).
		return "go:" + rel
	case "rust":
		// Rust test is a bare crate name; adapter stores it as rs:<crate>.
		return "rs:" + test
	}
	return test
}

func (idx *nodeIndex) sourceToNodeID(source, language string) string {
	switch language {
	case "solidity":
		return "sol:" + source
	case "go":
		// Go source paths are full import paths — prepend prefix.
		return "go:" + source
	case "rust":
		// Rust source is absolute; find the owning crate (longest-prefix
		// match against crate dir) and compose rs:<crate>/<rel>.
		for _, c := range idx.rustCrateDirs {
			if source == c.dir {
				return "rs:" + c.name
			}
			if strings.HasPrefix(source, c.dir+"/") {
				rel := strings.TrimPrefix(source, c.dir+"/")
				return "rs:" + c.name + "/" + rel
			}
		}
		return ""
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
