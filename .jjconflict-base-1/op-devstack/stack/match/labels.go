package match

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func WithLabel[I comparable, E interface {
	stack.ComponentIdentifiable
	Label(key string) string
}](key, value string) stack.ComponentIDMatcher[E] {
	return ComponentMatchElemFn[E](func(elem E) bool {
		return elem.Label(key) == value
	})
}

const (
	LabelVendor = "vendor"
)

type Vendor string

const (
	Geth                Vendor = "geth"
	OpReth              Vendor = "op-reth"
	OpGeth              Vendor = "op-geth"
	Proxyd              Vendor = "proxyd"
	FlashblocksWSClient Vendor = "flashblocks-websocket-proxy"
	OpNode              Vendor = "op-node"
	KonaNode            Vendor = "kona-node"
)

func (v Vendor) Match(elems []stack.L2ELNode) []stack.L2ELNode {
	return WithLabel[stack.ComponentID, stack.L2ELNode](LabelVendor, string(v)).Match(elems)
}

func (v Vendor) String() string {
	return string(v)
}
