package match

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

func First[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return ByIndex[I, E](0)
}

func Second[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return ByIndex[I, E](1)
}

// ByIndex matches element i (zero-indexed).
func ByIndex[I comparable, E stack.Identifiable[I]](index int) stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		if index < 0 {
			return nil
		}
		if index >= len(elems) {
			return nil
		}
		return elems[index : index+1]
	})
}

// Last matches the last element.
func Last[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		if len(elems) == 0 {
			return nil
		}
		return elems[len(elems)-1:]
	})
}

// Random matches a singular random element.
func Random[I comparable, E stack.Identifiable[I]](rng *rand.Rand) stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		if len(elems) == 0 {
			return nil
		}
		i := rng.Intn(len(elems))
		return elems[i : i+1]
	})
}

// Only matches the only value. If there are none, or more than one, then no value is matched.
func Only[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		if len(elems) != 1 {
			return nil
		}
		return elems
	})
}

// Combine combines all the matchers, by running them all, narrowing down the set with each application.
// If none are provided, all inputs are matched.
func Combine[I comparable, E stack.Identifiable[I]](matchers ...stack.Matcher[I, E]) stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		for _, matcher := range matchers {
			elems = matcher.Match(elems)
		}
		return elems
	})
}

// Or returns each of the inputs that have a match with any of the matchers.
// All inputs are applied to all matchers, even if matched previously.
func Or[I comparable, E stack.Identifiable[I]](matchers ...stack.Matcher[I, E]) stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		seen := make(map[I]struct{})
		for _, matcher := range matchers {
			for _, elem := range matcher.Match(elems) {
				seen[elem.ID()] = struct{}{}
			}
		}
		// preserve sort order and duplicates by iterating the original list
		out := make([]E, 0, len(seen))
		for _, elem := range elems {
			if _, ok := seen[elem.ID()]; ok {
				out = append(out, elem)
			}
		}
		return out
	})
}

// Not matches the elements that do not match the given matcher.
func Not[I comparable, E stack.Identifiable[I]](matcher stack.Matcher[I, E]) stack.Matcher[I, E] {
	return MatchFn[I, E](func(elems []E) []E {
		matched := make(map[I]struct{})
		for _, elem := range matcher.Match(elems) {
			matched[elem.ID()] = struct{}{}
		}
		out := make([]E, 0, len(elems))
		for _, elem := range elems {
			if _, ok := matched[elem.ID()]; !ok {
				out = append(out, elem)
			}
		}
		return out
	})
}
