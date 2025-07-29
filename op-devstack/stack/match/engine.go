package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

func WithEngine(engine stack.L2ELNodeID) stack.Matcher[stack.L2CLNodeID, stack.L2CLNode] {
	return MatchElemFn[stack.L2CLNodeID, stack.L2CLNode](func(elem stack.L2CLNode) bool {
		for _, el := range elem.ELs() {
			if el.ID() == engine {
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
		return false
	})
}
