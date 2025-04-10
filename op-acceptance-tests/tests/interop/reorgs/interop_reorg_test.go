package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/presets"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/stretchr/testify/require"
)

var SimpleInterop presets.TestSetup[*presets.SimpleInterop]

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// Other setups may be added here, hydrated from the same orchestrator
	presets.DoMain(m, presets.NewSimpleInterop(&SimpleInterop))
}

// TestReorgUnsafeHead starts an interop chain with an op-test-sequencer, which takes control over sequencing the L2 chain and introduces a reorg on the unsafe head
func TestReorgUnsafeHead(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := SimpleInterop(t)
	l := sys.Log

	l1Network := sys.L1Network
	l2Anet := sys.L2ChainA
	elNode := l2Anet.Escape().L2ELNodes()[0]
	rapi := l2Anet.Escape().L2CLNodes()[0].RollupAPI()
	l2RefClient := l2Anet.Escape().L2CLNodes()[0].L2BlockRefByHash()

	l1Client := l1Network.Escape().L1ELNode(match.FirstL1EL).EthClient()

	active, err := rapi.SequencerActive(ctx)
	require.NoError(t, err, "Expected to be able to call SequencerActive API, but got error")
	l.Info("Rollup node sequencer", "active", active)

	l.Info("Wait for a L1 block")
	l1Network.WaitForBlock()

	l.Info("Wait for two L2 blocks")
	l2Anet.WaitForBlock()
	l2Anet.WaitForBlock()

	unsafeHead, err := rapi.StopSequencer(ctx)
	require.NoError(t, err, "Expected to be able to call StopSequencer API, but got error")

	time.Sleep(1 * time.Second)
	active, err = rapi.SequencerActive(ctx)
	require.NoError(t, err, "Expected to be able to call SequencerActive API, but got error")

	l.Info("Rollup node sequencer", "active", active, "unsafeHead", unsafeHead)

	unsafeHeadRef, err := l2RefClient.L2BlockRefByHash(ctx, unsafeHead)
	require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

	l.Info("Current unsafe ref", "unsafeHead", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash, "l1_origin", unsafeHeadRef.L1Origin)

	l.Info("Expect to reorg the chain on current unsafe block", "number", unsafeHeadRef.Number, "head", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash)
	divergenceBlockNumber := unsafeHeadRef.Number
	oldHash := unsafeHeadRef.Hash
	oldParentHash := unsafeHeadRef.ParentID().Hash

	l2Anet.PrintChain()

	l1Origin, err := l1Client.InfoByLabel(ctx, "latest")
	require.NoError(t, err, "Expected to get latest block from L1 execution client")

	l10hash := l1Origin.Hash()

	parentOfUnsafeHead := unsafeHeadRef.ParentID()
	parentsL1Origin, err := l2RefClient.L2BlockRefByHash(ctx, parentOfUnsafeHead.Hash)
	require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

	if l1Origin.NumberU64() == parentsL1Origin.L1Origin.Number {
		l.Info("Wait for a new L1 block, as current L1 head is the same as the parent of the unsafe head")
		l1Network.WaitForBlock()

		l1Origin, err := l1Client.InfoByLabel(ctx, "latest")
		require.NoError(t, err, "Expected to get latest block from L1 execution client")

		l10hash = l1Origin.Hash()
	}

	l.Info("Sequencing with op-test-sequencer (override L1 origin)", "unsafeHead", unsafeHeadRef, "parent", parentOfUnsafeHead, "l1_origin", eth.InfoToL1BlockRef(l1Origin))
	l.Info("Sequencing with op-test-sequencer (override L1 origin)", "parent", parentOfUnsafeHead, "l1_origin_for_parent", parentsL1Origin.L1Origin)

	opts := seqtypes.BuildOpts{
		Parent:   parentOfUnsafeHead.Hash,
		L1Origin: &l10hash,
	}
	seq := sys.Sequencer
	ia := seq.IndividualAPI()

	l.Info("Calling New() on op-test-sequencer")
	err = ia.New(ctx, opts)
	require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

	l.Info("Calling Next() on op-test-sequencer")
	err = ia.Next(ctx)
	require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")

	l.Info("Block has been produced, do another one with op-test-sequencer")

	initialBlockInfo, err := elNode.EthClient().InfoByLabel(ctx, "latest")
	require.NoError(t, err, "Expected to get latest block from L2 execution client")

	l.Info("Current unsafe ref", "unsafeHead", eth.InfoToL1BlockRef(initialBlockInfo))

	l.Info("Sequencing with op-test-sequencer (no L1 origin override)")
	l.Info("Calling New() on op-test-sequencer")
	opts = seqtypes.BuildOpts{
		Parent:   initialBlockInfo.Hash(),
		L1Origin: nil,
	}
	err = ia.New(ctx, opts)
	require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")
	time.Sleep(2 * time.Second)

	l.Info("Calling Next() on op-test-sequencer")
	err = ia.Next(ctx)
	require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
	time.Sleep(2 * time.Second)

	newUnsafeHeadRef := l2Anet.UnsafeHeadRef()
	l.Info("Continue sequencing with consensus node", "unsafeHead", newUnsafeHeadRef)

	err = rapi.StartSequencer(ctx, newUnsafeHeadRef.Hash)
	require.NoError(t, err, "Expected to be able to start sequencer on rollup node")

	active, err = rapi.SequencerActive(ctx)
	require.NoError(t, err, "Expected to be able to call SequencerActive API, but got error")
	time.Sleep(1 * time.Second)

	l.Info("Rollup node sequencer", "active", active)
	l.Info("Wait for a L2 block")
	l2Anet.WaitForBlock()

	reorgedRef, err := elNode.EthClient().BlockRefByNumber(ctx, divergenceBlockNumber)
	require.NoError(t, err, "Expected to be able to call BlockRefByNumber API, but got error")

	l2Anet.PrintChain()

	l.Info("Reorged chain on divergence block number (prior the reorg)", "number", divergenceBlockNumber, "head", oldHash, "parent", oldParentHash)
	l.Info("Reorged chain on divergence block number (after the reorg)", "number", divergenceBlockNumber, "head", reorgedRef.Hash, "parent", reorgedRef.ParentID().Hash)
	require.NotEqual(t, oldHash, reorgedRef.Hash, "Expected to get different heads on divergence block number, but got the same hash, so no reorg happened")
	require.Equal(t, oldParentHash, reorgedRef.ParentHash, "Expected to get same parent hashes on divergence block number, but got different hashes")
}
