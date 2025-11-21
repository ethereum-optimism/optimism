package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

func SecondComponent[E stack.ComponentIdentifiable]() stack.ComponentIDMatcher[E] {
	return ByComponentIndex[E](1)
}

func SecondChain[E stack.ChainIdentifiable]() stack.ChainIDMatcher[E] {
	return ByChainIndex[E](1)
}

var SecondL2EL = SecondComponent[stack.L2ELNode]()
var SecondL2CL = SecondComponent[stack.L2CLNode]()

var SecondSupervisor = SecondComponent[stack.Supervisor]()
var SecondL2Network = SecondChain[stack.L2Network]()
