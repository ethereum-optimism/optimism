package superfaultproofs

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
)

// deletionSlots is the number of storage slots the deletion contract's constructor fills.
// The runtime code zeroes every slot but slot 0, so the storage trie collapses onto slot 0's
// leaf, which no executed transaction reads or writes.
const deletionSlots = 32

// RunReplacedBlockStorageDeletionTest proves the consolidation of a replaced block whose
// transactions delete storage.
//
// Chain B's block at endTimestamp zeroes 31 of a contract's 32 storage slots and carries an
// invalid executing message, so the supernode replaces it. Consolidating the replaced block makes
// the host re-execute it in a nested backend. Applying the deletions collapses the contract's
// storage trie onto slot 0's leaf; that leaf is off every account and storage proof path, so the
// host can only serve it from the block's execution witness.
func RunReplacedBlockStorageDeletionTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	chains := replacedBlockChains(t, sys)

	// Deploy before freezeChains stops the sequencers.
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	deleter := deployContract(t, bob, storageDeletionInitCode().Hex())

	freezeChains(chains)
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	parentA, parentB := advanceToReplacedBlockParents(t, sys, chains, endTimestamp)

	// Chain A's block at endTimestamp is empty, so the message chain B executes below references
	// a log that never exists.
	sys.TestSequencer.SequenceBlock(t, chains[0].ID, parentA.Hash)
	t.Require().Equal(endTimestamp, sys.L2ELA.BlockRefByLabel(eth.Unsafe).Time)

	payload := crypto.Keccak256([]byte("replaced block storage deletion fabricated msg"))
	invalidMsg := messages.Message{
		Identifier: messages.Identifier{
			Origin:      deleter,
			BlockNumber: parentA.Number + 1,
			LogIndex:    0,
			Timestamp:   endTimestamp,
			ChainID:     chains[0].ID,
		},
		PayloadHash: crypto.Keccak256Hash(payload),
	}
	deleteTx := txplan.NewPlannedTx(bob.Plan(), txplan.WithTo(&deleter), txplan.WithGasLimit(500_000))
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob,
		deleteTx, dsl.SubmitExecForMessage(invalidMsg, bob))
	replacedNum := parentB.Number + 1
	t.Require().Equal(endTimestamp, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Time)

	// The deletions are in the block that will be replaced.
	requireStorageSlot(t, sys.L2ELB, deleter, 0, replacedNum, 1)
	requireStorageSlot(t, sys.L2ELB, deleter, 1, replacedNum, 0)

	awaitSecondChainBlockReplaced(t, sys, chains, parentA, parentB, endTimestamp)
	// The replacement dropped the deletions.
	requireStorageSlot(t, sys.L2ELB, deleter, 1, replacedNum, 2)

	proveReplacedBlockConsolidation(t, sys, chains, endTimestamp, runners...)
}

// storageDeletionInitCode returns init code that sets slot i to i+1 for every i below
// deletionSlots, and deploys runtime code that zeroes every slot but slot 0.
func storageDeletionInitCode() *program.Program {
	runtime := program.New()
	for slot := 1; slot < deletionSlots; slot++ {
		runtime.Sstore(slot, 0)
	}
	initCode := program.New()
	for slot := range deletionSlots {
		initCode.Sstore(slot, slot+1)
	}
	return initCode.ReturnViaCodeCopy(runtime.Bytes())
}

func requireStorageSlot(t devtest.T, el *dsl.L2ELNode, addr common.Address, slot int64, blockNum uint64, want int64) {
	got, err := el.Escape().EthClient().GetStorageAt(t.Ctx(), addr, common.BigToHash(big.NewInt(slot)), hexutil.EncodeUint64(blockNum))
	t.Require().NoError(err)
	t.Require().Equalf(common.BigToHash(big.NewInt(want)), got,
		"slot %d of %s at block %d", slot, addr, blockNum)
}

// replacedBlockChains returns the two interop chains of a scenario that replaces a block on the
// second chain.
func replacedBlockChains(t devtest.T, sys *presets.SimpleInterop) []*chain {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required to build the replaced block")

	chains := orderedChains(sys)
	t.Require().Len(chains, 2)
	t.Require().Equal(chains[0].ID, sys.L2ChainA.ChainID())
	t.Require().Equal(chains[1].ID, sys.L2ChainB.ChainID())
	return chains
}

// advanceToReplacedBlockParents advances both frozen chains to their last block before
// endTimestamp, makes it safe and waits for the supernode to validate it. It returns those blocks,
// the parents of the blocks the test sequences at endTimestamp.
func advanceToReplacedBlockParents(t devtest.T, sys *presets.SimpleInterop, chains []*chain, endTimestamp uint64) (parentA, parentB eth.L2BlockRef) {
	advanceUnsafeToTimestamp(t, sys, chains, endTimestamp-1)
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp - 1)

	parentA = chains[0].EL.BlockRefByLabel(eth.Unsafe)
	parentB = chains[1].EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(endTimestamp, parentA.Time+chains[0].Cfg.BlockTime)
	t.Require().Equal(endTimestamp, parentB.Time+chains[1].Cfg.BlockTime)
	return parentA, parentB
}

// awaitSecondChainBlockReplaced makes the blocks after parentA and parentB safe, waits for the
// supernode to validate endTimestamp, and asserts that it replaced the second chain's block.
func awaitSecondChainBlockReplaced(t devtest.T, sys *presets.SimpleInterop, chains []*chain, parentA, parentB eth.L2BlockRef, endTimestamp uint64) {
	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)
	chains[0].CLNode.Reached(safety.CrossSafe, parentA.Number+1, 60)
	chains[1].CLNode.Reached(safety.CrossSafe, parentB.Number+1, 60)

	optimistic := optimisticBlockAtTimestamp(t, sys.SuperRoots.QueryAPI(), chains[1].ID, endTimestamp)
	t.Require().NotEqualf(optimistic.BlockHash, chains[1].EL.BlockRefByNumber(parentB.Number+1).Hash,
		"chain %s block %d at timestamp %d must have been replaced; otherwise the host finds the "+
			"disputed optimistic block canonical and skips the nested re-execution",
		chains[1].ID, parentB.Number+1, endTimestamp)
}

// proveReplacedBlockConsolidation runs every proof runner over the transitions from
// endTimestamp-1 to endTimestamp, where the second chain's block at endTimestamp was replaced.
func proveReplacedBlockConsolidation(t devtest.T, sys *presets.SimpleInterop, chains []*chain, endTimestamp uint64, runners ...ProofRunner) {
	queryAPI := sys.SuperRoots.QueryAPI()
	l1Head := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))
	runScenarioProofs(t, &sys.SingleChainInterop, chains, &scenarioProofData{
		fpvmTransitions:    replacedBlockTransitions(t, chains, queryAPI, endTimestamp-1, endTimestamp, l1Head),
		fpvmStartTimestamp: endTimestamp - 1,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, endTimestamp, true, runners),
	}, runners...)
}

// replacedBlockTransitions returns the transitions from startTimestamp to endTimestamp, where
// endTimestamp holds a replaced block on the second chain: the optimistic steps, the padding steps,
// and both outcomes of consolidating the replaced block, all at l1Head.
func replacedBlockTransitions(
	t devtest.T,
	chains []*chain,
	queryAPI apis.SupernodeQueryAPI,
	startTimestamp, endTimestamp uint64,
	l1Head eth.BlockID,
) []*transitionTest {
	start := superRootAtTimestamp(t, chains, startTimestamp)
	crossSafeEnd := superRootAtTimestamp(t, chains, endTimestamp)
	firstOptimistic := optimisticBlockAtTimestamp(t, queryAPI, chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, queryAPI, chains[1].ID, endTimestamp)
	paddingStep := func(step uint64) []byte {
		return marshalTransition(start, step, firstOptimistic, secondOptimistic)
	}
	optimisticEnd := eth.NewSuperV1(endTimestamp,
		eth.ChainIDAndOutput{ChainID: chains[0].ID, Output: firstOptimistic.OutputRoot},
		eth.ChainIDAndOutput{ChainID: chains[1].ID, Output: secondOptimistic.OutputRoot})
	step1 := marshalTransition(start, 1, firstOptimistic)
	step2 := marshalTransition(start, 2, firstOptimistic, secondOptimistic)

	transition := func(name string, agreed, disputed []byte, traceIndex int64, expectValid bool) *transitionTest {
		return &transitionTest{
			Name:               name,
			AgreedClaim:        agreed,
			DisputedClaim:      disputed,
			DisputedTraceIndex: traceIndex,
			L1Head:             l1Head,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        expectValid,
		}
	}
	return []*transitionTest{
		transition("FirstChainOptimisticBlock", start.Marshal(), step1, 0, true),
		transition("SecondChainOptimisticBlock", step1, step2, 1, true),
		transition("FirstPaddingStep", step2, paddingStep(3), 2, true),
		transition("LastPaddingStep", paddingStep(consolidateStep-1), paddingStep(consolidateStep), consolidateStep-1, true),
		transition("Consolidate-ExpectInvalidPendingBlock", paddingStep(consolidateStep), optimisticEnd.Marshal(), consolidateStep, false),
		transition("Consolidate-ReplaceInvalidBlock", paddingStep(consolidateStep), crossSafeEnd.Marshal(), consolidateStep, true),
		transition("AlreadyAtClaimedTimestamp", crossSafeEnd.Marshal(), crossSafeEnd.Marshal(), 5000, true),
	}
}
