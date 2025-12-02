package match

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func WithEngine(engine stack.L2ELNodeID) stack.Matcher[stack.L2CLNodeID, stack.L2CLNode] {
	return MatchElemFn[stack.L2CLNodeID, stack.L2CLNode](func(elem stack.L2CLNode) bool {
		for _, el := range elem.ELs() {
			if el.ID() == engine {
				return true
			}
		}
		rbID := stack.RollupBoostNodeID(engine)
		for _, rb := range elem.RollupBoostNodes() {
			if rb.ID().ChainID() == rbID.ChainID() {
				return true
			}
		}
		oprbID := stack.OPRBuilderNodeID(engine)
		for _, oprb := range elem.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}

func EngineFor(cl stack.L2CLNode) stack.Matcher[stack.L2ELNodeID, stack.L2ELNode] {
	return MatchElemFn[stack.L2ELNodeID, stack.L2ELNode](func(elem stack.L2ELNode) bool {
		for _, el := range cl.ELs() {
			if el.ID() == elem.ID() {
				return true
			}
		}
		rbID := stack.RollupBoostNodeID(elem.ID())
		for _, rb := range cl.RollupBoostNodes() {
			if rb.ID().ChainID() == rbID.ChainID() {
				return true
			}
		}
		oprbID := stack.OPRBuilderNodeID(elem.ID())
		for _, oprb := range cl.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}
