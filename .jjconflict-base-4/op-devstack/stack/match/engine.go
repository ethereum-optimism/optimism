package match

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

func WithEngine(engine stack.ComponentID) stack.ComponentIDMatcher[stack.L2CLNode] {
	return ComponentMatchElemFn[stack.L2CLNode](func(elem stack.L2CLNode) bool {
		for _, el := range elem.ELs() {
			if el.ID() == engine {
				return true
			}
		}
		rbID := stack.ComponentID(engine)
		for _, rb := range elem.RollupBoostNodes() {
			if rb.ID() == rbID {
				return true
			}
		}
		oprbID := stack.ComponentID(engine)
		for _, oprb := range elem.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}

func EngineFor(cl stack.L2CLNode) stack.ComponentIDMatcher[stack.L2ELNode] {
	return ComponentMatchElemFn[stack.L2ELNode](func(elem stack.L2ELNode) bool {
		for _, el := range cl.ELs() {
			if el.ID() == elem.ID() {
				return true
			}
		}
		rbID := stack.ComponentID(elem.ID())
		for _, rb := range cl.RollupBoostNodes() {
			if rb.ID() == rbID {
				return true
			}
		}
		oprbID := stack.ComponentID(elem.ID())
		for _, oprb := range cl.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				return true
			}
		}
		return false
	})
}
