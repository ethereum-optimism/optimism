package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type assumeComponent[E stack.ComponentIdentifiable] struct {
	t     devtest.T
	inner stack.ComponentIDMatcher[E]
}

func (a *assumeComponent[E]) Match(elems []E) []E {
	elems = a.inner.Match(elems)
	a.t.Gate().NotEmpty(elems, "must match something to continue, but matched nothing with %s", a.inner)
	return elems
}

func (a *assumeComponent[E]) String() string {
	return fmt.Sprintf("Assume(%s)", a.inner)
}

// AssumeComponent skips the test if no elements were matched with the inner matcher
func AssumeComponent[E stack.ComponentIdentifiable](t devtest.T, inner stack.ComponentIDMatcher[E]) stack.ComponentIDMatcher[E] {
	return &assumeComponent[E]{
		t:     t,
		inner: inner,
	}
}

type assumeChain[E stack.ChainIdentifiable] struct {
	t     devtest.T
	inner stack.ChainIDMatcher[E]
}

func (a *assumeChain[E]) Match(elems []E) []E {
	elems = a.inner.Match(elems)
	a.t.Gate().NotEmpty(elems, "must match something to continue, but matched nothing with %s", a.inner)
	return elems
}

func (a *assumeChain[E]) String() string {
	return fmt.Sprintf("Assume(%s)", a.inner)
}

// AssumeChain skips the test if no elements were matched with the inner matcher
func AssumeChain[E stack.ChainIdentifiable](t devtest.T, inner stack.ChainIDMatcher[E]) stack.ChainIDMatcher[E] {
	return &assumeChain[E]{
		t:     t,
		inner: inner,
	}
}
