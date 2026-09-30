package superfaultproofs

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// RunCycleReplacementPreservesAcyclicPrerequisiteTest checks that cycle replacement excludes chain C.
func RunCycleReplacementPreservesAcyclicPrerequisiteTest(
	t devtest.T,
	sys *presets.ThreeChainInterop,
	runners ...ProofRunner,
) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required for this test")

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	carol := sys.FunderC.NewFundedEOA(eth.OneEther)
	eventLoggerC := carol.DeployEventLogger()

	sys.Supernode().EnsureInteropPaused(10, sys.L2CLA, sys.L2CLB, sys.L2CLC)
	sys.L2CLA.StopSequencer()
	sys.L2CLB.StopSequencer()
	sys.L2CLC.StopSequencer()

	parentA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	parentB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	parentC := sys.L2ELC.BlockRefByLabel(eth.Unsafe)
	nextTimestamp := max(
		sys.L2ChainA.TimestampForBlockNum(parentA.Number+1),
		sys.L2ChainB.TimestampForBlockNum(parentB.Number+1),
		sys.L2ChainC.TimestampForBlockNum(parentC.Number+1),
	)
	parentA = sys.TestSequencer.AdvanceToBlockParent(t, sys.L2ChainA, nextTimestamp)
	parentB = sys.TestSequencer.AdvanceToBlockParent(t, sys.L2ChainB, nextTimestamp)
	parentC = sys.TestSequencer.AdvanceToBlockParent(t, sys.L2ChainC, nextTimestamp)

	blockA, blockB, blockC := parentA.Number+1, parentB.Number+1, parentC.Number+1
	initC := carol.PrepareSameTimestampInit(rand.New(rand.NewSource(22825)), eventLoggerC, blockC, 0, nextTimestamp)
	c0 := dsl.PrecomputeExecEventMessage(initC.Message, sys.L2ChainC.ChainID(), blockC, 1, nextTimestamp)
	a1 := dsl.PrecomputeExecEventMessage(c0, sys.L2ChainA.ChainID(), blockA, 1, nextTimestamp)
	b0 := dsl.PrecomputeExecEventMessage(a1, sys.L2ChainB.ChainID(), blockB, 0, nextTimestamp)

	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentA.Hash, alice,
		dsl.SubmitExecForMessage(b0, alice),
		dsl.SubmitExecForMessage(c0, alice),
	)
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob,
		dsl.SubmitExecForMessage(a1, bob),
	)
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentC.Hash, carol,
		initC.SubmitInit(),
		dsl.SubmitExecForMessage(initC.Message, carol),
	)
	targetA := sys.L2ELA.BlockRefByLabel(eth.Unsafe).Number
	sys.L2BatcherA.Start()
	sys.L2CLA.Reached(safety.LocalSafe, targetA, 60)
	sys.L2BatcherA.Stop()
	targetB := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number
	sys.L2BatcherB.Start()
	sys.L2CLB.Reached(safety.LocalSafe, targetB, 60)
	sys.L2BatcherB.Stop()
	targetC := sys.L2ELC.BlockRefByLabel(eth.Unsafe).Number
	sys.L2BatcherC.Start()
	sys.L2CLC.Reached(safety.LocalSafe, targetC, 60)
	sys.L2BatcherC.Stop()

	optimisticA := sys.Supernode().AwaitOptimisticBlockAtTimestamp(sys.L2ChainA.ChainID(), nextTimestamp)
	optimisticB := sys.Supernode().AwaitOptimisticBlockAtTimestamp(sys.L2ChainB.ChainID(), nextTimestamp)
	optimisticC := sys.Supernode().AwaitOptimisticBlockAtTimestamp(sys.L2ChainC.ChainID(), nextTimestamp)

	sys.Supernode().ResumeInterop()
	sys.SuperRoots.AwaitValidatedTimestamp(nextTimestamp)
	response := sys.SuperRoots.SuperRootAtTimestamp(nextTimestamp)
	t.Require().NotNil(response.Data, "expected verified super root at timestamp %d", nextTimestamp)
	verified, ok := response.Data.Super.(*eth.SuperV1)
	t.Require().True(ok, "verified super root must be SuperV1")
	t.Require().Len(verified.Chains, 3, "verified super root must contain three chains")
	t.Require().Equal(sys.L2ChainA.ChainID(), verified.Chains[0].ChainID, "verified chain A order must match the dependency set")
	t.Require().Equal(sys.L2ChainB.ChainID(), verified.Chains[1].ChainID, "verified chain B order must match the dependency set")
	t.Require().Equal(sys.L2ChainC.ChainID(), verified.Chains[2].ChainID, "verified chain C order must match the dependency set")

	expected := eth.NewSuperV1(
		nextTimestamp,
		verified.Chains[0],
		verified.Chains[1],
		eth.ChainIDAndOutput{ChainID: sys.L2ChainC.ChainID(), Output: optimisticC.OutputRoot},
	)
	startTimestamp := nextTimestamp - 1
	start := *eth.NewSuperV1(
		startTimestamp,
		eth.ChainIDAndOutput{ChainID: sys.L2ChainA.ChainID(), Output: sys.L2CLA.OutputAtBlock(parentA.Number).OutputRoot},
		eth.ChainIDAndOutput{ChainID: sys.L2ChainB.ChainID(), Output: sys.L2CLB.OutputAtBlock(parentB.Number).OutputRoot},
		eth.ChainIDAndOutput{ChainID: sys.L2ChainC.ChainID(), Output: sys.L2CLC.OutputAtBlock(parentC.Number).OutputRoot},
	)
	preConsolidation := marshalTransition(start, consolidateStep, optimisticA, optimisticB, optimisticC)
	l1Head := latestRequiredL1(response)

	t.Require().NotEqual(optimisticA.OutputRoot, verified.Chains[0].Output,
		"supernode preserved cyclic chain A")
	t.Require().NotEqual(optimisticB.OutputRoot, verified.Chains[1].Output,
		"supernode preserved cyclic chain B")
	t.Require().Equal(optimisticC.OutputRoot, verified.Chains[2].Output,
		"supernode did not preserve acyclic prerequisite chain C")

	runThreeChainScenarioProofs(t, sys, &scenarioProofData{
		fpvmTransitions: []*transitionTest{{
			Name:               "PreservesAcyclicPrerequisite",
			AgreedClaim:        preConsolidation,
			DisputedClaim:      expected.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1Head,
			ClaimTimestamp:     nextTimestamp,
			ExpectValid:        true,
		}},
		fpvmStartTimestamp: startTimestamp,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, nextTimestamp, true, runners),
	}, runners...)
}
