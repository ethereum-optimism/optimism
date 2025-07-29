package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type assume[I comparable, E stack.Identifiable[I]] struct {
	t     devtest.T
	inner stack.Matcher[I, E]
}

func (a *assume[I, E]) Match(elems []E) []E {
	elems = a.inner.Match(elems)
	a.t.Gate().NotEmpty(elems, "must match something to continue, but matched nothing with %s", a.inner)
	return elems
}

func (a *assume[I, E]) String() string {
	return fmt.Sprintf("Assume(%s)", a.inner)
}

// Assume skips the test if no elements were matched with the inner matcher
func Assume[I comparable, E stack.Identifiable[I]](t devtest.T, inner stack.Matcher[I, E]) stack.Matcher[I, E] {
	return &assume[I, E]{
		t:     t,
		inner: inner,
	}
}
