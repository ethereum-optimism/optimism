package match

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

func WithLabel[I comparable, E interface {
	stack.Identifiable[I]
	Label(key string) string
}](key, value string) stack.Matcher[I, E] {
	return MatchElemFn[I, E](func(elem E) bool {
		return elem.Label(key) == value
	})
}

const (
	LabelVendor = "vendor"
)

type L2ELVendor string

const (
	OpReth L2ELVendor = "op-reth"
	OpGeth L2ELVendor = "op-geth"
	Proxyd L2ELVendor = "proxyd"
)

func (v L2ELVendor) Match(elems []stack.L2ELNode) []stack.L2ELNode {
	return WithLabel[stack.L2ELNodeID, stack.L2ELNode](LabelVendor, string(v)).Match(elems)
}

func (v L2ELVendor) String() string {
	return string(v)
}
