package match

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

var FirstL2EL = First[stack.L2ELNodeID, stack.L2ELNode]()
var FirstL2CL = First[stack.L2CLNodeID, stack.L2CLNode]()
var FirstL2Batcher = First[stack.L2BatcherID, stack.L2Batcher]()
var FirstL2Proposer = First[stack.L2ProposerID, stack.L2Proposer]()
var FirstL2Challenger = First[stack.L2ChallengerID, stack.L2Challenger]()

var FirstTestSequencer = First[stack.TestSequencerID, stack.TestSequencer]()
var FirstSupervisor = First[stack.SupervisorID, stack.Supervisor]()

var FirstL1EL = First[stack.L1ELNodeID, stack.L1ELNode]()
var FirstL1CL = First[stack.L1CLNodeID, stack.L1CLNode]()

var FirstL1Network = First[stack.L1NetworkID, stack.L1Network]()
var FirstL2Network = First[stack.L2NetworkID, stack.L2Network]()
var FirstSuperchain = First[stack.SuperchainID, stack.Superchain]()
var FirstCluster = First[stack.ClusterID, stack.Cluster]()

var FirstFaucet = First[stack.FaucetID, stack.Faucet]()
var FirstSyncTester = First[stack.SyncTesterID, stack.SyncTester]()
