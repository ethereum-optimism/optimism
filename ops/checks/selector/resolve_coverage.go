package selector

import (
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// coverageCandidates emits one Candidate per (test, profile) whose
// coverage line ranges intersect the diff's changed lines. Signal comes
// from the fraction of changed lines the test touches, floored at
// policy.Coverage.SignalFloor for any hit (rewards hit/no-hit
// discrimination over fine-grained fractions — see 76967a8b5f).
func coverageCandidates(
	g *graph.Graph,
	ct *catalog.CheckType,
	changedIDs []string,
	changedLines map[string]map[int]bool,
	pol *policy.Policy,
	fresh freshness.Checker,
) []Candidate {
	floor := pol.Coverage.SignalFloor

	type key struct {
		testNode string
		profile  string
	}
	type entry struct {
		signal     float64
		hitLines   int
		totalLines int
		freshness  float64 // multiplier actually applied (1.0 = as-generated)
	}
	best := make(map[key]entry)

	update := func(k key, raw entry) {
		raw.signal *= raw.freshness
		if e, ok := best[k]; !ok || raw.signal > e.signal {
			best[k] = raw
		}
	}

	for _, changedID := range changedIDs {
		sourceFile := nodeIDToSourceFile(changedID)
		fileChanged := changedLines[sourceFile]

		for _, edge := range g.EdgesTo(changedID) {
			if edge.Source != graph.SourceCoverage {
				continue
			}
			// Coverage edges are language-typed by their endpoints.
			// A Solidity coverage edge (sol:test → sol:src) must not
			// produce a go-test or rust-test candidate — the scope
			// derivation would use the Solidity test file path, and
			// the resulting command (`go test ./test/X.t.sol`) is
			// nonsense. Skip edges whose test-side node doesn't match
			// this check's language.
			if !testNodeMatchesLanguage(edge.From, ct.Language) {
				continue
			}
			profile := profileFromEdge(edge)
			fr := fresh.Assess(edge)

			if len(fileChanged) == 0 {
				update(key{edge.From, profile}, entry{
					signal:    edge.Strength * edge.Confidence,
					freshness: fr,
				})
				continue
			}

			lineRanges, ok := edge.Properties["line_ranges"]
			if !ok {
				update(key{edge.From, profile}, entry{
					signal:    edge.Strength * edge.Confidence,
					freshness: fr,
				})
				continue
			}

			hitCount := countLineHits(lineRanges, fileChanged)
			if hitCount == 0 {
				continue
			}

			totalChanged := len(fileChanged)
			hitFraction := float64(hitCount) / float64(totalChanged)
			signal := (floor + (1-floor)*hitFraction) * edge.Confidence
			update(key{edge.From, profile}, entry{
				signal:     signal,
				hitLines:   hitCount,
				totalLines: totalChanged,
				freshness:  fr,
			})
		}
	}

	if len(best) == 0 {
		return nil
	}

	var out []Candidate
	for k, e := range best {
		scope := scopeForCandidate(k.testNode, ct)
		if scope == "" {
			continue
		}
		raw := map[string]any{
			"test_node": k.testNode,
		}
		if e.hitLines > 0 {
			raw["hit_lines"] = e.hitLines
			raw["total_changed"] = e.totalLines
		}
		if e.freshness != 0 && e.freshness < 1.0 {
			raw["freshness"] = e.freshness
		}
		out = append(out, Candidate{
			CheckID: ct.ID,
			Scope:   scope,
			Profile: k.profile,
			Signal:  e.signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceCoverage,
				EdgeKind:     graph.EdgeTestedBy,
				Contribution: e.signal,
				Raw:          raw,
			}},
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Signal > out[j].Signal })
	return out
}
