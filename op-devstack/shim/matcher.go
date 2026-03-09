package shim

import (
	"slices"
	"sort"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

// findMatch checks if the matcher is an ID wrapper for direct lookup. If not, then it will search the list of values for a matching element.
// If multiple elements match, the first found is returned.
// The values function is used to lazy-fetch values in sorted order, such that the search is deterministic.
func findMatch[E stack.Identifiable](m stack.Matcher[E], getValue func(stack.ComponentID) (E, bool), values func() []E) (out E, found bool) {
	// Check for idMatcher wrapper (created by stack.ByID)
	if idm, ok := m.(interface{ ID() stack.ComponentID }); ok {
		return getValue(idm.ID())
	}
	got := m.Match(values())
	if len(got) == 0 {
		return
	}
	return got[0], true
}

// sortByID sorts a slice of ComponentIDs.
func sortByID(ids []stack.ComponentID) []stack.ComponentID {
	out := slices.Clone(ids)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Less(out[j])
	})
	return out
}

// sortByIDFunc sorts a slice of elements by extracting their ID.
func sortByIDFunc[T stack.Identifiable](elems []T) []T {
	out := slices.Clone(elems)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID().Less(out[j].ID())
	})
	return out
}
