package superfaultproofs

import (
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunReplacedBlockSameChainMessageTest proves the consolidation of a replaced block whose only
// transaction executes a message naming its own parent block on the same chain.
//
// The message is fabricated (its parent has no such log), so the supernode replaces the block.
// Consolidating it resolves the initiating header by walking back from the replaced block, the
// chain's local safe head, which no node serves state for.
func RunReplacedBlockSameChainMessageTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required to build the replaced block")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2)
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID())
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID())

	// Fund before freezeChains stops the sequencers.
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	freezeChains(chains)

	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	startTimestamp := endTimestamp - 1
	advanceUnsafeToTimestamp(t, sys, chains, startTimestamp)
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(startTimestamp)

	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	parentB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(endTimestamp, unsafeA.Time+chains[0].Cfg.BlockTime)
	t.Require().Equal(endTimestamp, parentB.Time+chains[1].Cfg.BlockTime)

	sys.TestSequencer.SequenceBlock(t, chains[0].ID, unsafeA.Hash)
	t.Require().Equal(endTimestamp, sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time)

	invalidMsg := messages.Message{
		Identifier: messages.Identifier{
			Origin:      bob.Address(),
			BlockNumber: parentB.Number,
			LogIndex:    0,
			Timestamp:   parentB.Time,
			ChainID:     chains[1].ID,
		},
		PayloadHash: crypto.Keccak256Hash([]byte("replaced block same chain message fabricated msg")),
	}
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob, dsl.SubmitExecForMessage(invalidMsg, bob))
	replacedNum := parentB.Number + 1
	t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)

	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLB.Reached(safety.CrossSafe, replacedNum, 60)

	queryAPI := sys.SuperRoots.QueryAPI()
	requireBlockReplaced(t, sys.L2ELB, queryAPI, chains[1].ID, replacedNum, endTimestamp)

	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	runScenarioProofs(t, &sys.SingleChainInterop, chains, &scenarioProofData{
		fpvmTransitions:    replacedBlockTransitions(t, chains, queryAPI, startTimestamp, endTimestamp, l1Head),
		fpvmStartTimestamp: startTimestamp,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, endTimestamp, true, runners),
	}, runners...)
}
