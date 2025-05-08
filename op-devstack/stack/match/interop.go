package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// L2ChainA is an alias for the first L2 network.
var L2ChainA = First[stack.L2NetworkID, stack.L2Network]()

// L2ChainB is an alias for the second L2 network.
var L2ChainB = Second[stack.L2NetworkID, stack.L2Network]()
