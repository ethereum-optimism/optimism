package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
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

	ia := sys.Sequencer.Escape().IndividualAPI(sys.L2ChainA.ChainID())

	l.Info("Stopping batcher")
	err := sys.L2BatcherA.Escape().ActivityAPI().StopBatcher(ctx)
	require.NoError(t, err, "Expected to be able to call StopBatcher API, but got error")

	// two eoas for a sample transfer tx used later in a conflicting block
	alice := sys.FunderA.NewFundedEOA(eth.ThousandEther)
	bob := sys.Wallet.NewEOA(sys.L2ELA)

	active, err := sys.L2CLNodeA.Escape().RollupAPI().SequencerActive(ctx)
	require.NoError(t, err, "Expected to be able to call SequencerActive API, but got error")
	l.Info("Rollup node sequencer status", "active", active)

	sys.L1Network.WaitForBlock()

	sys.L2ChainA.WaitForBlock()
	// waiting for two blocks in order to make sure we are not jumping ahead of a L1 origin (i.e. can't build a chain with L1Origin gaps)
	sys.L2ChainA.WaitForBlock()
	sys.L2ChainA.WaitForBlock()

	unsafeHead, err := sys.L2CLNodeA.Escape().RollupAPI().StopSequencer(ctx)
	require.NoError(t, err, "Expected to be able to call StopSequencer API, but got error")

	// wait for the sequencer to become inactive
	err = wait.For(ctx, 1*time.Second, func() (bool, error) {
		active, err = sys.L2CLNodeA.Escape().RollupAPI().SequencerActive(ctx)
		return !active, err
	})
	require.NoError(t, err, "Expected to be able to call SequencerActive API, and wait for inactive state for sequencer, but got error")

	l.Info("Rollup node sequencer status", "active", active, "unsafeHead", unsafeHead)

	var divergenceBlockNumber uint64
	var originalRef eth.L2BlockRef
	// prepare and sequencer a conflicting block for the L2A chain
	{
		unsafeHeadRef, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByHash(ctx, unsafeHead)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		l.Info("Current unsafe ref", "unsafeHead", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash, "l1_origin", unsafeHeadRef.L1Origin)

		l.Info("Expect to reorg the chain on current unsafe block", "number", unsafeHeadRef.Number, "head", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash)
		divergenceBlockNumber = unsafeHeadRef.Number
		originalRef = unsafeHeadRef

		sys.L2ChainA.PrintChain()

		l1Origin, err := sys.L1Network.Escape().L1ELNode(match.FirstL1EL).EthClient().InfoByLabel(ctx, "latest")
		require.NoError(t, err, "Expected to get latest block from L1 execution client")

		l1OriginHash := l1Origin.Hash()

		parentOfUnsafeHead := unsafeHeadRef.ParentID()
		parentsL1Origin, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByHash(ctx, parentOfUnsafeHead.Hash)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		if l1Origin.NumberU64() == parentsL1Origin.L1Origin.Number {
			l.Info("Wait for a new L1 block, as current L1 head is the same as the parent of the unsafe head")
			sys.L1Network.WaitForBlock()

			l1Origin, err := sys.L1Network.Escape().L1ELNode(match.FirstL1EL).EthClient().InfoByLabel(ctx, "latest")
			require.NoError(t, err, "Expected to get latest block from L1 execution client")

			l1OriginHash = l1Origin.Hash()
		}

		l.Info("Sequencing a conflicting block", "unsafeHead", unsafeHeadRef, "parent", parentOfUnsafeHead, "l1_origin", eth.InfoToL1BlockRef(l1Origin))

		// sequence a conflicting block with a simple transfer tx, based on the parent of the parent of the unsafe head
		{
			err = ia.New(ctx, seqtypes.BuildOpts{
				Parent:   parentOfUnsafeHead.Hash,
				L1Origin: &l1OriginHash,
			})
			require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

			// include simple transfer tx in opened block
			{
				to := alice.PlanTransfer(bob.Address(), eth.OneEther)
				opt := txplan.Combine(to)
				ptx := txplan.NewPlannedTx(opt)
				signed_tx, err := ptx.Signed.Eval(ctx)
				require.NoError(t, err, "Expected to be able to evaluate a planned transaction on op-test-sequencer, but got error")
				txdata, err := signed_tx.MarshalBinary()
				require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")

				err = ia.IncludeTx(ctx, txdata)
				require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")
			}

			err = ia.Next(ctx)
			require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		}
	}

	l.Info("Conflicting block has been produced, sequence a second block with op-test-sequencer")

	{
		currentUnsafeRef := sys.L2ChainA.UnsafeHeadRef()
		l.Info("Current unsafe ref", "unsafeHead", currentUnsafeRef)

		l.Info("Starting batcher")
		err = sys.L2BatcherA.Escape().ActivityAPI().StartBatcher(ctx)
		require.NoError(t, err, "Expected to be able to call StartBatcher API, but got error")

		l.Info("Sequencing with op-test-sequencer (no L1 origin override)")
		err = ia.New(ctx, seqtypes.BuildOpts{
			Parent:   currentUnsafeRef.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)
	}

	newUnsafeHeadRef := sys.L2ChainA.UnsafeHeadRef()
	l.Info("Continue sequencing with consensus node (op-node)", "unsafeHead", newUnsafeHeadRef)

	err = sys.L2CLNodeA.Escape().RollupAPI().StartSequencer(ctx, newUnsafeHeadRef.Hash)
	require.NoError(t, err, "Expected to be able to start sequencer on rollup node")

	// wait for the sequencer to become active
	err = wait.For(ctx, 1*time.Second, func() (bool, error) {
		active, err = sys.L2CLNodeA.Escape().RollupAPI().SequencerActive(ctx)
		return active, err
	})
	require.NoError(t, err, "Expected to be able to call SequencerActive API, and wait for an active state for sequencer, but got error")

	l.Info("Rollup node sequencer", "active", active)

	sys.L2ChainA.WaitForBlock()

	reorgedRef, err := sys.L2ELA.Escape().EthClient().BlockRefByNumber(ctx, divergenceBlockNumber)
	require.NoError(t, err, "Expected to be able to call BlockRefByNumber API, but got error")

	sys.L2ChainA.PrintChain()

	l.Info("Reorged chain on divergence block number (prior the reorg)", "number", divergenceBlockNumber, "head", originalRef.Hash, "parent", originalRef.ParentID().Hash)
	l.Info("Reorged chain on divergence block number (after the reorg)", "number", divergenceBlockNumber, "head", reorgedRef.Hash, "parent", reorgedRef.ParentID().Hash)
	require.NotEqual(t, originalRef.Hash, reorgedRef.Hash, "Expected to get different heads on divergence block number, but got the same hash, so no reorg happened")
	require.Equal(t, originalRef.ParentID().Hash, reorgedRef.ParentHash, "Expected to get same parent hashes on divergence block number, but got different hashes")

	err = wait.For(ctx, 5*time.Second, func() (bool, error) {
		var safeL2Head_supervisor common.Hash
		var safeL2Head_sequencer eth.L2BlockRef

		// get supervisor safe L2 head
		{
			safeBlockID := sys.Supervisor.SafeBlockID(sys.L2ChainA.ChainID())
			safeL2Head_supervisor = safeBlockID.Hash
		}
		// get sequencer safe L2 head
		{
			safeL2Head_sequencer = sys.L2CLNodeA.SafeL2BlockRef()
		}

		if safeL2Head_sequencer.Number <= divergenceBlockNumber {
			l.Info("Safe ref number is still behind divergence block number", "divergence", divergenceBlockNumber, "safe", safeL2Head_sequencer.Number)
			return false, nil
		}

		if safeL2Head_sequencer.Hash.Cmp(safeL2Head_supervisor) == 0 {
			l.Info("Safe ref is the same on both supervisor and sequencer", "supervisor", safeL2Head_supervisor, "sequencer", safeL2Head_sequencer.Hash)
			return true, nil
		}

		l.Info("Safe ref still not the same on supervisor and sequencer", "supervisor", safeL2Head_supervisor, "sequencer", safeL2Head_sequencer.Hash)
		return false, nil
	})
	require.NoError(t, err, "Expected to get same safe ref on both supervisor and sequencer eventually")
	sys.L2ChainA.PrintChain()
}
