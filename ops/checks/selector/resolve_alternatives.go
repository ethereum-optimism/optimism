package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// applyStagePolicy filters Candidates by stage acceptance rules and
// collapses assertion groups to one discharger per group.
//
// Two concerns compose here:
//
//  1. Tag acceptance. A check is accepted at a stage iff either
//     stage.AcceptTags is empty, the check has no Tags, or at least
//     one of the check's Tags is in stage.AcceptTags. Candidates whose
//     CheckType is not accepted are dropped.
//
//  2. Assertion collapse. After filtering, checks sharing the same
//     CheckType.Assertion are alternatives. Exactly one survives per
//     assertion group, chosen by:
//       - best tag preference in stage.AcceptTags (earliest position)
//       - tiebreaker: lower CheckType.AvgDuration (cheaper runs win)
//       - tiebreaker: lexical CheckID (stable across runs)
//
// The single-authority check-level decision keeps the downstream
// fan-out (scope tiers, profiles, prereqs) coherent: candidates for a
// dropped CheckID disappear as a unit.
func applyStagePolicy(candidates []Candidate, cat *catalog.Catalog, stage Stage) []Candidate {
	if cat == nil || len(candidates) == 0 {
		return candidates
	}
	keep := make(map[string]bool)
	for _, c := range candidates {
		if _, ok := keep[c.CheckID]; ok {
			continue
		}
		ct := cat.ByID(c.CheckID)
		if ct == nil {
			keep[c.CheckID] = true
			continue
		}
		keep[c.CheckID] = stage.AcceptsTags(ct.Tags)
	}

	groups := make(map[string][]string)
	for id, ok := range keep {
		if !ok {
			continue
		}
		ct := cat.ByID(id)
		if ct == nil || ct.Assertion == "" {
			continue
		}
		groups[ct.Assertion] = append(groups[ct.Assertion], id)
	}
	for _, ids := range groups {
		if len(ids) <= 1 {
			continue
		}
		winner := pickAssertionWinner(ids, cat, stage)
		for _, id := range ids {
			if id != winner {
				keep[id] = false
			}
		}
	}

	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if keep[c.CheckID] {
			out = append(out, c)
		}
	}
	return out
}

// pickAssertionWinner picks one CheckID from an assertion group.
// Caller guarantees len(ids) >= 2 and every id is in cat.
func pickAssertionWinner(ids []string, cat *catalog.Catalog, stage Stage) string {
	best := ids[0]
	for _, id := range ids[1:] {
		if compareAlternatives(id, best, cat, stage) < 0 {
			best = id
		}
	}
	return best
}

// compareAlternatives returns negative if a is preferred over b.
// Preference order:
//  1. Better tag preference (earlier position in stage.AcceptTags).
//  2. Lower AvgDuration.
//  3. Lexically earlier ID.
func compareAlternatives(a, b string, cat *catalog.Catalog, stage Stage) int {
	ctA, ctB := cat.ByID(a), cat.ByID(b)
	prefA := bestTagPreference(ctA, stage)
	prefB := bestTagPreference(ctB, stage)
	if prefA != prefB {
		return prefA - prefB
	}
	if ctA.AvgDuration != ctB.AvgDuration {
		return ctA.AvgDuration - ctB.AvgDuration
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// bestTagPreference returns the minimum TagPreference across a
// check's tags. Untagged checks get the worst preference so tagged
// alternatives outrank them inside an assertion group. The preference
// is only meaningful relative to peers in the same group.
func bestTagPreference(ct *catalog.CheckType, stage Stage) int {
	if len(ct.Tags) == 0 {
		return len(stage.AcceptTags) + 1
	}
	best := len(stage.AcceptTags)
	for _, t := range ct.Tags {
		if p := stage.TagPreference(t); p < best {
			best = p
		}
	}
	return best
}
