package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

// MatchFn implements stack.Matcher, checking all elements at once.
type MatchFn[I comparable, E stack.Identifiable[I]] func(elems []E) []E

func (m MatchFn[I, E]) Match(elems []E) []E {
	return m(elems)
}

func (m MatchFn[I, E]) String() string {
	var id I
	var x E
	return fmt.Sprintf("MatchFn[%T, %T]", id, x)
}

var _ stack.Matcher[stack.L2NetworkID, stack.L2Network] = MatchFn[stack.L2NetworkID, stack.L2Network](nil)

// MatchElemFn implements stack.Matcher, checking one element at a time.
type MatchElemFn[I comparable, E stack.Identifiable[I]] func(elem E) bool

func (m MatchElemFn[I, E]) Match(elems []E) (out []E) {
	for _, elem := range elems {
		if m(elem) {
			out = append(out, elem)
		}
	}
	return out
}

func (m MatchElemFn[I, E]) String() string {
	var id I
	var x E
	return fmt.Sprintf("MatchElemFn[%T, %T]", id, x)
}

var _ stack.Matcher[stack.L2NetworkID, stack.L2Network] = MatchElemFn[stack.L2NetworkID, stack.L2Network](nil)
