package match

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type byComponentIndexMatcher[E stack.ComponentIdentifiable] struct {
	index int
}

func (ma byComponentIndexMatcher[E]) Match(elems []E) []E {
	if ma.index < 0 {
		return nil
	}
	if ma.index >= len(elems) {
		return nil
	}
	return elems[ma.index : ma.index+1]
}

func (ma byComponentIndexMatcher[E]) String() string {
	return fmt.Sprintf("ByIndex(%d)", ma.index)
}

// ByIndex matches element i (zero-indexed).
func ByComponentIndex[E stack.ComponentIdentifiable](index int) stack.ComponentIDMatcher[E] {
	return byComponentIndexMatcher[E]{index: index}
}

type byChainIndexMatcher[E stack.ChainIdentifiable] struct {
	index int
}

func (ma byChainIndexMatcher[E]) Match(elems []E) []E {
	if ma.index < 0 {
		return nil
	}
	if ma.index >= len(elems) {
		return nil
	}
	return elems[ma.index : ma.index+1]
}

func (ma byChainIndexMatcher[E]) String() string {
	return fmt.Sprintf("ByIndex(%d)", ma.index)
}

// ByIndex matches element i (zero-indexed).
func ByChainIndex[E stack.ChainIdentifiable](index int) stack.ChainIDMatcher[E] {
	return byChainIndexMatcher[E]{index: index}
}

type lastMatcher[E stack.ComponentIdentifiable] struct{}

func (ma lastMatcher[E]) Match(elems []E) []E {
	if len(elems) == 0 {
		return nil
	}
	return elems[len(elems)-1:]
}

func (ma lastMatcher[E]) String() string {
	return "Last"
}

// Last matches the last element.
func Last[E stack.ComponentIdentifiable]() stack.ComponentIDMatcher[E] {
	return lastMatcher[E]{}
}

type onlyMatcher[E stack.ComponentIdentifiable] struct{}

func (ma onlyMatcher[E]) Match(elems []E) []E {
	if len(elems) != 1 {
		return nil
	}
	return elems
}

func (ma onlyMatcher[E]) String() string {
	return "Only"
}

// Only matches the only value. If there are none, or more than one, then no value is matched.
func Only[E stack.ComponentIdentifiable]() stack.ComponentIDMatcher[E] {
	return onlyMatcher[E]{}
}

type andMatcher[E stack.ComponentIdentifiable] struct {
	inner []stack.ComponentIDMatcher[E]
}

func (ma andMatcher[E]) Match(elems []E) []E {
	for _, matcher := range ma.inner {
		elems = matcher.Match(elems)
	}
	return elems
}

func (ma andMatcher[E]) String() string {
	return fmt.Sprintf("And(%s)", joinStr(ma.inner))
}

// And combines all the matchers, by running them all, narrowing down the set with each application.
// If none are provided, all inputs are matched.
func And[E stack.ComponentIdentifiable](matchers ...stack.ComponentIDMatcher[E]) stack.ComponentIDMatcher[E] {
	return andMatcher[E]{inner: matchers}
}

type orMatcher[E stack.ComponentIdentifiable] struct {
	inner []stack.ComponentIDMatcher[E]
}

func (ma orMatcher[E]) Match(elems []E) []E {
	seen := make(map[stack.ComponentID]struct{})
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

func (ma orMatcher[E]) String() string {
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
func Or[E stack.ComponentIdentifiable](matchers ...stack.ComponentIDMatcher[E]) stack.ComponentIDMatcher[E] {
	return orMatcher[E]{inner: matchers}
}

type notMatcher[E stack.ComponentIdentifiable] struct {
	inner stack.ComponentIDMatcher[E]
}

func (ma notMatcher[E]) Match(elems []E) []E {
	matched := make(map[stack.ComponentID]struct{})
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

func (ma notMatcher[E]) String() string {
	return fmt.Sprintf("Not(%s)", ma.inner)
}

// Not matches the elements that do not match the given matcher.
func Not[E stack.ComponentIdentifiable](matcher stack.ComponentIDMatcher[E]) stack.ComponentIDMatcher[E] {
	return notMatcher[E]{inner: matcher}
}
