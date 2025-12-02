package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// L2ChainA is an alias for the first L2 network.
var L2ChainA = First[stack.L2NetworkID, stack.L2Network]()

// L2ChainB is an alias for the second L2 network.
var L2ChainB = Second[stack.L2NetworkID, stack.L2Network]()

// L2ChainById returns a matcher for the L2 network with the given ID.
func L2ChainById(id stack.L2NetworkID) stack.Matcher[stack.L2NetworkID, stack.L2Network] {
	return byID[stack.L2NetworkID, stack.L2Network](id)
}
