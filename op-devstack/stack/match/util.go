package match

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func First[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return ByIndex[I, E](0)
}

func Second[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return ByIndex[I, E](1)
}

type byIndexMatcher[I comparable, E stack.Identifiable[I]] struct {
	index int
}

func (ma byIndexMatcher[I, E]) Match(elems []E) []E {
	if ma.index < 0 {
		return nil
	}
	if ma.index >= len(elems) {
		return nil
	}
	return elems[ma.index : ma.index+1]
}

func (ma byIndexMatcher[I, E]) String() string {
	return fmt.Sprintf("ByIndex(%d)", ma.index)
}

// ByIndex matches element i (zero-indexed).
func ByIndex[I comparable, E stack.Identifiable[I]](index int) stack.Matcher[I, E] {
	return byIndexMatcher[I, E]{index: index}
}

type lastMatcher[I comparable, E stack.Identifiable[I]] struct{}

func (ma lastMatcher[I, E]) Match(elems []E) []E {
	if len(elems) == 0 {
		return nil
	}
	return elems[len(elems)-1:]
}

func (ma lastMatcher[I, E]) String() string {
	return "Last"
}

// Last matches the last element.
func Last[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return lastMatcher[I, E]{}
}

type onlyMatcher[I comparable, E stack.Identifiable[I]] struct{}

func (ma onlyMatcher[I, E]) Match(elems []E) []E {
	if len(elems) != 1 {
		return nil
	}
	return elems
}

func (ma onlyMatcher[I, E]) String() string {
	return "Only"
}

// Only matches the only value. If there are none, or more than one, then no value is matched.
func Only[I comparable, E stack.Identifiable[I]]() stack.Matcher[I, E] {
	return onlyMatcher[I, E]{}
}

type andMatcher[I comparable, E stack.Identifiable[I]] struct {
	inner []stack.Matcher[I, E]
}

func (ma andMatcher[I, E]) Match(elems []E) []E {
	for _, matcher := range ma.inner {
		elems = matcher.Match(elems)
	}
	return elems
}

func (ma andMatcher[I, E]) String() string {
	return fmt.Sprintf("And(%s)", joinStr(ma.inner))
}

// And combines all the matchers, by running them all, narrowing down the set with each application.
// If none are provided, all inputs are matched.
func And[I comparable, E stack.Identifiable[I]](matchers ...stack.Matcher[I, E]) stack.Matcher[I, E] {
	return andMatcher[I, E]{inner: matchers}
}

type orMatcher[I comparable, E stack.Identifiable[I]] struct {
	inner []stack.Matcher[I, E]
}

func (ma orMatcher[I, E]) Match(elems []E) []E {
	seen := make(map[I]struct{})
	for _, matcher := range ma.inner {
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
}

func (ma orMatcher[I, E]) String() string {
	return fmt.Sprintf("Or(%s)", joinStr(ma.inner))
}

func joinStr[V fmt.Stringer](elems []V) string {
	var out strings.Builder
	for i, e := range elems {
		out.WriteString(e.String())
		if i < len(elems)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

// Or returns each of the inputs that have a match with any of the matchers.
// All inputs are applied to all matchers, even if matched previously.
func Or[I comparable, E stack.Identifiable[I]](matchers ...stack.Matcher[I, E]) stack.Matcher[I, E] {
	return orMatcher[I, E]{inner: matchers}
}

type notMatcher[I comparable, E stack.Identifiable[I]] struct {
	inner stack.Matcher[I, E]
}

func (ma notMatcher[I, E]) Match(elems []E) []E {
	matched := make(map[I]struct{})
	for _, elem := range ma.inner.Match(elems) {
		matched[elem.ID()] = struct{}{}
	}
	out := make([]E, 0, len(elems))
	for _, elem := range elems {
		if _, ok := matched[elem.ID()]; !ok {
			out = append(out, elem)
		}
	}
	return out
}

func (ma notMatcher[I, E]) String() string {
	return fmt.Sprintf("Not(%s)", ma.inner)
}

// Not matches the elements that do not match the given matcher.
func Not[I comparable, E stack.Identifiable[I]](matcher stack.Matcher[I, E]) stack.Matcher[I, E] {
	return notMatcher[I, E]{inner: matcher}
}
