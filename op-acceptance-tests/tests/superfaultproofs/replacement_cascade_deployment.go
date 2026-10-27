package superfaultproofs

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunReplacementCascadeDeploymentTest proves the consolidation of a block replaced in cascade
// that calls a contract deployed on top of an earlier replacement.
//
// The blocks after a replaced block are not re-derived: their batches name the replaced block as
// parent, so derivation drops them, and the sequencer re-sequences their transactions instead.
// Chain B is therefore built in two phases:
//   - Y emits an initiating message and carries an invalid executing message, so Y is replaced.
//   - On top of the replacement, block D deploys a contract, and block D+1 calls it and executes
//     Y's initiating message. Y's replacement dropped that message, so D+1 is replaced in cascade.
//
// Consolidating D+1 makes the host re-execute it in a nested backend on top of D, so the nested
// execution reads the contract's code, which only exists from D on.
func RunReplacementCascadeDeploymentTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required to force-include the cascade's executing message")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2)
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID())
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID())
	blockTime := chains[1].Cfg.BlockTime
	t.Require().Equal(blockTime, chains[0].Cfg.BlockTime, "the cascade layout assumes equal block times")
	rng := rand.New(rand.NewSource(20260727))

	// Fund and deploy before freezeChains stops the sequencers.
	initiator := sys.FunderB.NewFundedEOA(eth.OneEther)
	deployer := sys.FunderB.NewFundedEOA(eth.OneEther)
	caller := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerB := initiator.DeployEventLogger()

	// sequenceBlockPair builds the next block on both chains, which must share a head timestamp: an
	// empty one on chain A and one with the given transactions on chain B. It returns chain B's
	// new block.
	sequenceBlockPair := func(sender *dsl.EOA, txs ...*txplan.PlannedTx) eth.L2BlockRef {
		headA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		headB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		t.Require().Equal(headA.Time, headB.Time, "chains must share a head timestamp")
		sys.TestSequencer.SequenceBlock(t, chains[0].ID, headA.Hash)
		sys.TestSequencer.SequenceBlockWithPlannedTxs(t, headB.Hash, sender, txs...)
		block := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		t.Require().Equal(headB.Number+1, block.Number)
		t.Require().Equal(headB.Time+blockTime, block.Time)
		return block
	}
	// alignChains drives both frozen chains to a shared, validated head timestamp after which both
	// produce a block, and returns that next timestamp.
	alignChains := func() uint64 {
		next := nextTimestampAfterSafeHeads(t, chains)
		advanceUnsafeToTimestamp(t, sys, chains, next-blockTime)
		advanceSafeToCurrentUnsafe(t, chains[0])
		advanceSafeToCurrentUnsafe(t, chains[1])
		sys.SuperRoots.AwaitValidatedTimestamp(next - blockTime)
		return next
	}

	// Y: the initiating message, and an executing message referencing a log that chain A's empty
	// block at Y's timestamp never emits.
	freezeChains(chains)
	firstTimestamp := alignChains()
	firstNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number + 1
	init := initiator.PrepareSameTimestampInit(rng, eventLoggerB, firstNum, 0, firstTimestamp)
	payload := crypto.Keccak256([]byte("replacement cascade deployment fabricated msg"))
	invalidMsg := messages.Message{
		Identifier: messages.Identifier{
			Origin:      eventLoggerB,
			BlockNumber: sys.L2ELA.BlockRefByLabel(eth.Unsafe).Number + 1,
			LogIndex:    0,
			Timestamp:   firstTimestamp,
			ChainID:     chains[0].ID,
		},
		PayloadHash: crypto.Keccak256Hash(payload),
	}
	sequenceBlockPair(initiator, init.SubmitInit(), dsl.SubmitExecForMessage(invalidMsg, initiator))
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(firstTimestamp)
	sys.L2CLB.Reached(safety.CrossSafe, firstNum, 60)
	queryAPI := sys.SuperRoots.QueryAPI()
	requireBlockReplaced(t, sys.L2ELB, queryAPI, chains[1].ID, firstNum, firstTimestamp)

	// D: the deployment, on top of Y's replacement. The rewind that replaced Y restarted chain B's
	// sequencer, so take control of block production again first.
	sys.L2CLB.AwaitSequencerActive()
	refreezeChains(chains)
	deployTimestamp := alignChains()
	contract := crypto.CreateAddress(deployer.Address(), deployer.PendingNonce())
	runtime := program.New().Push(0xc0de).Push(0).Op(vm.MSTORE).Return(0, 32).Bytes()
	initCode := program.New().ReturnViaCodeCopy(runtime).Bytes()
	deployBlock := sequenceBlockPair(deployer,
		txplan.NewPlannedTx(deployer.Plan(), txplan.WithData(initCode), txplan.WithGasLimit(200_000)))
	t.Require().Equal(deployTimestamp, deployBlock.Time)
	code, err := sys.L2ELB.Escape().EthClient().CodeAtHash(t.Ctx(), contract, deployBlock.Hash)
	t.Require().NoError(err)
	t.Require().Equal(runtime, code, "D must deploy the contract")

	// D+1: the call into the deployed contract, and the execution of Y's initiating message, which
	// Y's replacement dropped. The test sequencer force-includes it; the mempool would reject it.
	cascadeBlock := sequenceBlockPair(caller,
		txplan.NewPlannedTx(caller.Plan(), txplan.WithTo(&contract), txplan.WithGasLimit(100_000)),
		init.SubmitExecTo(caller))
	endTimestamp := cascadeBlock.Time

	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLB.Reached(safety.CrossSafe, cascadeBlock.Number, 60)
	requireBlockReplaced(t, sys.L2ELB, queryAPI, chains[1].ID, cascadeBlock.Number, endTimestamp)

	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	runScenarioProofs(t, &sys.SingleChainInterop, chains, &scenarioProofData{
		fpvmTransitions:    replacedBlockTransitions(t, chains, queryAPI, endTimestamp-1, endTimestamp, l1Head),
		fpvmStartTimestamp: endTimestamp - 1,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, endTimestamp, true, runners),
	}, runners...)
}
