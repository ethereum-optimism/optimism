package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

var FirstL2EL = First[stack.L2ELNode]()
var FirstL2CL = First[stack.L2CLNode]()
var FirstL2Batcher = First[stack.L2Batcher]()
var FirstL2Proposer = First[stack.L2Proposer]()
var FirstL2Challenger = First[stack.L2Challenger]()

var FirstTestSequencer = First[stack.TestSequencer]()
var FirstSupervisor = First[stack.Supervisor]()
var FirstSupernode = First[stack.Supernode]()

var FirstL1EL = First[stack.L1ELNode]()
var FirstL1CL = First[stack.L1CLNode]()

var FirstL1Network = First[stack.L1Network]()
var FirstL2Network = First[stack.L2Network]()
var FirstSuperchain = First[stack.Superchain]()
var FirstCluster = First[stack.Cluster]()

var FirstFaucet = First[stack.Faucet]()
var FirstSyncTester = First[stack.SyncTester]()

var FirstOPRBuilderNode = First[stack.OPRBuilderNode]()
var FirstRollupBoostNode = First[stack.RollupBoostNode]()
