package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

// ComponentMatchFn implements stack.Matcher, checking all elements at once.
type ComponentMatchFn[E stack.ComponentIdentifiable] func(elems []E) []E

func (m ComponentMatchFn[E]) Match(elems []E) []E {
	return m(elems)
}

func (m ComponentMatchFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchFn[%T]", x)
}

// ChainMatchFn implements stack.Matcher, checking all elements at once.
type ChainMatchFn[E stack.ChainIdentifiable] func(elems []E) []E

func (m ChainMatchFn[E]) Match(elems []E) []E {
	return m(elems)
}

func (m ChainMatchFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchFn[%T]", x)
}

var _ stack.ChainIDMatcher[stack.L2Network] = ChainMatchFn[stack.L2Network](nil)

// ComponentMatchElemFn implements stack.Matcher, checking one element at a time.
type ComponentMatchElemFn[E stack.ComponentIdentifiable] func(elem E) bool

func (m ComponentMatchElemFn[E]) Match(elems []E) (out []E) {
	for _, elem := range elems {
		if m(elem) {
			out = append(out, elem)
		}
	}
	return out
}

func (m ComponentMatchElemFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchElemFn[%T]", x)
}

// ChainMatchElemFn implements stack.Matcher, checking one element at a time.
type ChainMatchElemFn[E stack.ChainIdentifiable] func(elem E) bool

func (m ChainMatchElemFn[E]) Match(elems []E) (out []E) {
	for _, elem := range elems {
		if m(elem) {
			out = append(out, elem)
		}
	}
	return out
}

func (m ChainMatchElemFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchElemFn[%T]", x)
}

var _ stack.ChainIDMatcher[stack.L2Network] = ChainMatchElemFn[stack.L2Network](nil)
