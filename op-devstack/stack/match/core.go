package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

// MatchFn implements stack.Matcher, checking all elements at once.
type MatchFn[E stack.Identifiable] func(elems []E) []E

func (m MatchFn[E]) Match(elems []E) []E {
	return m(elems)
}

func (m MatchFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchFn[%T]", x)
}

var _ stack.Matcher[stack.L2Network] = MatchFn[stack.L2Network](nil)

// MatchElemFn implements stack.Matcher, checking one element at a time.
type MatchElemFn[E stack.Identifiable] func(elem E) bool

func (m MatchElemFn[E]) Match(elems []E) (out []E) {
	for _, elem := range elems {
		if m(elem) {
			out = append(out, elem)
		}
	}
	return out
}

func (m MatchElemFn[E]) String() string {
	var x E
	return fmt.Sprintf("MatchElemFn[%T]", x)
}

var _ stack.Matcher[stack.L2Network] = MatchElemFn[stack.L2Network](nil)
