package match

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func WithEngine(engine stack.ComponentID) stack.Matcher[stack.L2CLNode] {
	return MatchElemFn[stack.L2CLNode](func(elem stack.L2CLNode) bool {
		for _, el := range elem.ELs() {
			if el.ID() == engine {
				return true
			}
		}
		// Check RollupBoost nodes with matching key/chainID
		rbID := stack.NewRollupBoostNodeID(engine.Key(), engine.ChainID())
		for _, rb := range elem.RollupBoostNodes() {
			if rb.ID() == rbID {
				return true
			}
		}
		// Check OPRBuilder nodes with matching key/chainID
		oprbID := stack.NewOPRBuilderNodeID(engine.Key(), engine.ChainID())
		for _, oprb := range elem.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}

func EngineFor(cl stack.L2CLNode) stack.Matcher[stack.L2ELNode] {
	return MatchElemFn[stack.L2ELNode](func(elem stack.L2ELNode) bool {
		for _, el := range cl.ELs() {
			if el.ID() == elem.ID() {
				return true
			}
		}
		// Check RollupBoost nodes with matching key/chainID
		rbID := stack.NewRollupBoostNodeID(elem.ID().Key(), elem.ID().ChainID())
		for _, rb := range cl.RollupBoostNodes() {
			if rb.ID() == rbID {
				return true
			}
		}
		// Check OPRBuilder nodes with matching key/chainID
		oprbID := stack.NewOPRBuilderNodeID(elem.ID().Key(), elem.ID().ChainID())
		for _, oprb := range cl.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}
