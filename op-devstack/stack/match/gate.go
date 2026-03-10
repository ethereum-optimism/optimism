package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type assume[E stack.Identifiable] struct {
	t     devtest.T
	inner stack.Matcher[E]
}

func (a *assume[E]) Match(elems []E) []E {
	elems = a.inner.Match(elems)
	a.t.Gate().NotEmpty(elems, "must match something to continue, but matched nothing with %s", a.inner)
	return elems
}

func (a *assume[E]) String() string {
	return fmt.Sprintf("Assume(%s)", a.inner)
}

// Assume skips the test if no elements were matched with the inner matcher
func Assume[E stack.Identifiable](t devtest.T, inner stack.Matcher[E]) stack.Matcher[E] {
	return &assume[E]{
		t:     t,
		inner: inner,
	}
}
