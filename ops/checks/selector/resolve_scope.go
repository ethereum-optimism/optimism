package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// scopeSelectedChecks takes the set of check IDs picked by the
// dataflow walker and expands each one into per-scope per-profile
// Candidates. Selection is already decided; this function answers
// "with what scope?" for scopeable checks and "clone under which
// profiles?" for profile-matrix expansion.
//
// Strategy per check:
//   - scopeable: union of coverage walk + import walk. Empty union
//     falls through to an unscoped safety-net candidate.
//   - non-scopeable: one unscoped candidate, signal=1.0.
//
// Profile expansion then clones each scopeable check's main-profile
// candidates under every profile whose ActiveWhenSelected list
// includes this check. The fallback for an empty main-profile scope
// emits one unscoped per-profile candidate so blast-like inputs
// (where coverage/imports found nothing) still fan out across the
// feature matrix, matching pre-cutover semantics.
func scopeSelectedChecks(
	g *graph.Graph,
	cat *catalog.Catalog,
	selectedCheckIDs map[string]bool,
	changedIDs []string,
	changedLines map[string]map[int]bool,
	pol *policy.Policy,
	fresh freshness.Checker,
) []Candidate {
	var out []Candidate
	byCheckMain := make(map[string][]Candidate)

	for i := range cat.CheckTypes {
		ct := &cat.CheckTypes[i]
		if !selectedCheckIDs[ct.ID] {
			continue
		}

		var cands []Candidate
		if ct.Scopeable {
			cov := coverageCandidates(g, ct, changedIDs, changedLines, pol, fresh)
			imp := importScopeCandidates(g, ct, changedIDs)
			cands = mergeScopedCandidates(cov, imp)
			if len(cands) == 0 {
				// Safety-net: dataflow selected this check but scope
				// walks found nothing. Fall back to an unscoped
				// candidate so the check still runs.
				cands = []Candidate{{
					CheckID: ct.ID,
					Signal:  1.0,
					Provenance: []SignalContribution{{
						Source:       graph.SourceStatic,
						Contribution: 1.0,
						Raw:          map[string]any{"reason": "dataflow_unscoped"},
					}},
				}}
			}
			byCheckMain[ct.ID] = cands
		} else {
			cands = []Candidate{{
				CheckID: ct.ID,
				Signal:  1.0,
				Provenance: []SignalContribution{{
					Source:       graph.SourceStatic,
					Contribution: 1.0,
					Raw:          map[string]any{"reason": "dataflow"},
				}},
			}}
		}
		out = append(out, cands...)
	}

	out = append(out, profileTriggerExpand(cat, selectedCheckIDs, byCheckMain)...)
	return out
}

// profileTriggerExpand clones each scopeable check's main-profile
// candidates under every profile whose ActiveWhenSelected list
// includes that check. Replaces the deleted profileTriggerCandidates
// which keyed on profile.Triggers glob-matching filePaths.
func profileTriggerExpand(
	cat *catalog.Catalog,
	selected map[string]bool,
	byCheckMain map[string][]Candidate,
) []Candidate {
	var out []Candidate
	for i := range cat.Profiles {
		p := &cat.Profiles[i]
		if p.Name == "" || p.Name == "main" {
			continue
		}
		if len(p.ActiveWhenSelected) == 0 {
			continue
		}
		activated := false
		for _, id := range p.ActiveWhenSelected {
			if selected[id] {
				activated = true
				break
			}
		}
		if !activated {
			continue
		}
		for _, id := range p.ActiveWhenSelected {
			if !selected[id] {
				continue
			}
			mainCands := byCheckMain[id]
			var mainProfileCands []Candidate
			for _, c := range mainCands {
				if c.Profile == "" {
					mainProfileCands = append(mainProfileCands, c)
				}
			}
			if len(mainProfileCands) == 0 {
				// Empty-main-scope fallback: emit one unscoped
				// per-profile candidate so blast-like inputs still
				// fan the matrix. Matches pre-cutover behavior where
				// blastRadiusCandidates emitted per-profile candidates
				// unconditionally for scopeable checks.
				out = append(out, Candidate{
					CheckID: id,
					Profile: p.Name,
					Signal:  1.0,
					Provenance: []SignalContribution{{
						Source:       graph.SourceStatic,
						Contribution: 1.0,
						Raw: map[string]any{
							"reason":  "profile_activation_unscoped",
							"profile": p.Name,
						},
					}},
				})
				continue
			}
			for _, c := range mainProfileCands {
				clone := c
				clone.Profile = p.Name
				clone.Provenance = []SignalContribution{{
					Source:       graph.SourceStatic,
					Contribution: c.Signal,
					Raw: map[string]any{
						"reason":          "profile_activation",
						"profile":         p.Name,
						"scope_from_main": true,
					},
				}}
				out = append(out, clone)
			}
		}
	}
	return out
}

// applyCorrelationSignal walks EdgeObservedCorrelation edges from each
// changed node; if the target is a check already in the candidate
// set, the signal is lifted and a provenance entry appended.
// Correlation no longer CREATES candidates post-cutover — dataflow
// selection is authoritative; correlation only adjusts signal on
// already-selected candidates.
func applyCorrelationSignal(g *graph.Graph, cands []Candidate, changedIDs []string, fresh freshness.Checker) []Candidate {
	type key struct {
		checkID string
		profile string
	}
	idx := make(map[key]int)
	for i, c := range cands {
		idx[key{c.CheckID, c.Profile}] = i
	}
	for _, nodeID := range changedIDs {
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind != graph.EdgeObservedCorrelation {
				continue
			}
			if !isCheckNode(edge.To) {
				continue
			}
			cid := edge.To[len("check:"):]
			fr := fresh.Assess(edge)
			signal := edge.Strength * edge.Confidence * fr
			// Correlation lifts every candidate for this check_id
			// (across all profiles).
			for k, i := range idx {
				if k.checkID != cid {
					continue
				}
				if signal > cands[i].Signal {
					cands[i].Signal = signal
				}
				cands[i].Provenance = append(cands[i].Provenance, SignalContribution{
					Source:       graph.SourceCIHistory,
					EdgeKind:     graph.EdgeObservedCorrelation,
					Contribution: signal,
					Raw: map[string]any{
						"reason": "correlation_boost",
					},
				})
			}
		}
	}
	return cands
}

func isCheckNode(id string) bool {
	return len(id) > len("check:") && id[:len("check:")] == "check:"
}
