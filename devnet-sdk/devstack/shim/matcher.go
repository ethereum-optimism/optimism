package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

// findMatch checks if the matcher is an ID for direct lookup. If not, then it will search the list of values for a matching element.
// If multiple elements match, the first found is returned.
// The values function is used to lazy-fetch values in sorted order, such that the search is deterministic.
func findMatch[I comparable, E stack.Identifiable[I]](m stack.Matcher[I, E], getValue func(I) (E, bool), values func() []E) (out E, found bool) {
	id, ok := m.(I)
	if ok {
		return getValue(id)
	}
	got := m.Match(values())
	if len(got) == 0 {
		return
	}
	return got[0], true
}
