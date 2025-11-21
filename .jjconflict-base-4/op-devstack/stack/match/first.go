package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

func FirstComponent[E stack.ComponentIdentifiable]() stack.ComponentIDMatcher[E] {
	return ByComponentIndex[E](0)
}

func FirstChain[E stack.ChainIdentifiable]() stack.ChainIDMatcher[E] {
	return ByChainIndex[E](0)
}

var FirstL2EL = FirstComponent[stack.L2ELNode]()
var FirstL2CL = FirstComponent[stack.L2CLNode]()
var FirstL2Batcher = FirstComponent[stack.L2Batcher]()
var FirstL2Proposer = FirstComponent[stack.L2Proposer]()
var FirstL2Challenger = FirstComponent[stack.L2Challenger]()

var FirstTestSequencer = FirstComponent[stack.TestSequencer]()
var FirstSupervisor = FirstComponent[stack.Supervisor]()

var FirstL1EL = FirstComponent[stack.L1ELNode]()
var FirstL1CL = FirstComponent[stack.L1CLNode]()

var FirstL1Network = FirstChain[stack.L1Network]()
var FirstL2Network = FirstChain[stack.L2Network]()
var FirstSuperchain = FirstComponent[stack.Superchain]()

var FirstFaucet = FirstComponent[stack.Faucet]()
var FirstSyncTester = FirstComponent[stack.SyncTester]()

var FirstOPRBuilderNode = FirstComponent[stack.OPRBuilderNode]()
var FirstRollupBoostNode = FirstComponent[stack.RollupBoostNode]()
