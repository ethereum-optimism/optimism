package superfaultproofs

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/crypto"
)

// RunReplacedBlockSameChainMessageTest proves the consolidation of a replaced block whose only user
// transaction executes a fabricated message naming its parent on the same chain.
func RunReplacedBlockSameChainMessageTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	chains := replacedBlockChains(t, sys)

	// Fund before freezeChains stops the sequencers.
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	freezeChains(chains)
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	parentA, parentB := advanceToReplacedBlockParents(t, sys, chains, endTimestamp)

	sys.TestSequencer.SequenceBlock(t, chains[0].ID, parentA.Hash)
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
	t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)

	awaitSecondChainBlockReplaced(t, sys, chains, parentA, parentB, endTimestamp)
	proveReplacedBlockConsolidation(t, sys, chains, endTimestamp, runners...)
}

// RunReplacedBlockCrossChainMessageTest proves the consolidation of a replaced block when another
// chain executes a message from an earlier block of the replaced chain.
//
// Chain B's replaced block executes a fabricated copy of a message initiated in an earlier chain B
// block, below the replaced block's parent. Chain A's block at the same timestamp executes the
// real message.
func RunReplacedBlockCrossChainMessageTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	chains := replacedBlockChains(t, sys)

	// Fund and initiate the message before freezeChains stops the sequencers.
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLogger := bob.DeployEventLogger()
	sent := bob.SendRandomInitMessage(rand.New(rand.NewSource(46)), eventLogger, 2, 10)
	out, err := sent.Tx.Result.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().NotEmpty(out.Entries)
	initMsg := out.Entries[0]

	freezeChains(chains)
	// The initiating block can be the frozen head. Skip a block so the replaced block's parent sits
	// above the initiating block.
	endTimestamp := advanceToCommonBlockBoundary(t, chains, nextTimestampAfterSafeHeads(t, chains)+1)
	parentA, parentB := advanceToReplacedBlockParents(t, sys, chains, endTimestamp)
	t.Require().Less(initMsg.Identifier.BlockNumber, parentB.Number,
		"the initiating block must be below the replaced block's parent")

	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentA.Hash, alice, dsl.SubmitExecForMessage(initMsg, alice))
	t.Require().Equal(endTimestamp, sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time)

	invalidMsg := initMsg
	invalidMsg.PayloadHash = crypto.Keccak256Hash([]byte("replaced block cross chain message fabricated msg"))
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob, dsl.SubmitExecForMessage(invalidMsg, bob))
	t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)

	awaitSecondChainBlockReplaced(t, sys, chains, parentA, parentB, endTimestamp)
	t.Require().Equal(optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[0].ID, endTimestamp).BlockHash,
		sys.L2ELA.BlockRefByNumber(parentA.Number+1).Hash,
		"chain A's block executes a valid message and must stay canonical")

	proveReplacedBlockConsolidation(t, sys, chains, endTimestamp, runners...)
}
