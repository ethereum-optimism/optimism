package superfaultproofs

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunReplacedBlockSameChainMessageTest proves the consolidation of a replaced block whose only user
// transaction executes a fabricated message naming its parent on the same chain.
//
// Resolving the parent's header must not read the replaced block's state: no node serves it, so the
// host would retry the request until the proof times out.
func RunReplacedBlockSameChainMessageTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	runReplacedBlockEarlierMessageTest(t, sys, false, runners...)
}

// RunReplacedBlockCrossChainMessageTest proves the consolidation of a replaced block when another
// chain executes a message from an earlier block of the replaced chain.
//
// Chain B's replaced block executes a fabricated copy of a message initiated in an earlier chain B
// block, and chain A's block at the same timestamp executes the real message. Both messages name
// the same chain B block, which the guest resolves by walking back from chain B's replaced block
// through EIP-2935 history. After the replacement, chain A's message is checked again, walking back
// from the deposit-only block.
func RunReplacedBlockCrossChainMessageTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	runReplacedBlockEarlierMessageTest(t, sys, true, runners...)
}

func runReplacedBlockEarlierMessageTest(t devtest.T, sys *presets.SimpleInterop, crossChain bool, runners ...ProofRunner) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required to build the replaced block")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2)
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID())
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID())

	// Fund and deploy before freezeChains stops the sequencers.
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	var alice *dsl.EOA
	var initMsg messages.Message
	if crossChain {
		alice = sys.FunderA.NewFundedEOA(eth.OneEther)
		eventLogger := bob.DeployEventLogger()
		sent := bob.SendRandomInitMessage(rand.New(rand.NewSource(46)), eventLogger, 2, 10)
		out, err := sent.Tx.Result.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().NotEmpty(out.Entries)
		initMsg = out.Entries[0]
	}

	freezeChains(chains)

	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	if crossChain {
		// The initiating block can be the frozen head. Skip a block so the replaced block's parent
		// sits above the initiating block.
		endTimestamp = advanceToCommonBlockBoundary(t, chains, endTimestamp+1)
	}
	startTimestamp := endTimestamp - 1
	advanceUnsafeToTimestamp(t, sys, chains, startTimestamp)
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(startTimestamp)

	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	parentB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(endTimestamp, unsafeA.Time+chains[0].Cfg.BlockTime)
	t.Require().Equal(endTimestamp, parentB.Time+chains[1].Cfg.BlockTime)

	payloadHash := crypto.Keccak256Hash([]byte("replaced block earlier message fabricated msg"))
	invalidMsg := messages.Message{
		Identifier: messages.Identifier{
			Origin:      bob.Address(),
			BlockNumber: parentB.Number,
			LogIndex:    0,
			Timestamp:   parentB.Time,
			ChainID:     chains[1].ID,
		},
		PayloadHash: payloadHash,
	}
	if crossChain {
		// Starting below chain B's parent block makes the lookup read EIP-2935 history.
		t.Require().Less(initMsg.Identifier.BlockNumber, parentB.Number)
		sys.TestSequencer.SequenceBlockWithPlannedTxs(t, unsafeA.Hash, alice, dsl.SubmitExecForMessage(initMsg, alice))
		invalidMsg = initMsg
		invalidMsg.PayloadHash = payloadHash
	} else {
		sys.TestSequencer.SequenceBlock(t, chains[0].ID, unsafeA.Hash)
	}
	t.Require().Equal(endTimestamp, sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time)

	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob, dsl.SubmitExecForMessage(invalidMsg, bob))
	replacedNum := parentB.Number + 1
	t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)

	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	sys.L2CLA.Reached(safety.CrossSafe, unsafeA.Number+1, 60)
	sys.L2CLB.Reached(safety.CrossSafe, replacedNum, 60)

	queryAPI := sys.SuperRoots.QueryAPI()
	requireBlockReplaced(t, sys.L2ELB, queryAPI, chains[1].ID, replacedNum, endTimestamp)
	if crossChain {
		t.Require().Equal(optimisticBlockAtTimestamp(t, queryAPI, chains[0].ID, endTimestamp).BlockHash,
			sys.L2ELA.BlockRefByNumber(unsafeA.Number+1).Hash,
			"chain A's block executes a valid message and must stay canonical")
	}

	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	runScenarioProofs(t, &sys.SingleChainInterop, chains, &scenarioProofData{
		fpvmTransitions:    replacedBlockTransitions(t, chains, queryAPI, startTimestamp, endTimestamp, l1Head),
		fpvmStartTimestamp: startTimestamp,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, endTimestamp, true, runners),
	}, runners...)
}
