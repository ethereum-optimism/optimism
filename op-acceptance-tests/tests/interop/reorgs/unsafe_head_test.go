package reorgs

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/stretchr/testify/require"
)

// TestReorgUnsafeHead starts an interop chain with an op-test-sequencer, which takes control over sequencing the L2 chain and introduces a reorg on the unsafe head
func TestReorgUnsafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()

	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	l := sys.Log

	ia := sys.TestSequencer.Escape().ControlAPI(sys.L2A.ChainID())

	// stop batcher on chain A
	sys.L2BatcherA.Stop()

	// two EOAs for a sample transfer tx used later in a conflicting block
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.Wallet.NewEOA(sys.L2ELA)

	sys.L1Network.WaitForBlock()

	sys.L2A.WaitForBlock()
	// waiting for two blocks in order to make sure we are not jumping ahead of a L1 origin (i.e. can't build a chain with L1Origin gaps)
	sys.L2A.WaitForBlock()
	sys.L2A.WaitForBlock()

	unsafeHead := sys.L2ACL.StopSequencer()

	var divergenceBlockNumber_A uint64
	var originalRef_A eth.L2BlockRef
	// prepare and sequence a conflicting block for the L2A chain
	{
		unsafeHeadRef := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

		l.Info("Current unsafe ref", "unsafeHead", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash, "l1_origin", unsafeHeadRef.L1Origin)

		l.Info("Expect to reorg the chain on current unsafe block", "number", unsafeHeadRef.Number, "head", unsafeHead, "parent", unsafeHeadRef.ParentID().Hash)
		divergenceBlockNumber_A = unsafeHeadRef.Number
		originalRef_A = unsafeHeadRef

		parentOfUnsafeHead := unsafeHeadRef.ParentID()

		l.Info("Sequencing a conflicting block", "unsafeHead", unsafeHeadRef, "parent", parentOfUnsafeHead)

		// sequence a conflicting block with a simple transfer tx, based on the parent of the parent of the unsafe head
		{
			err := ia.New(ctx, seqtypes.BuildOpts{
				Parent:   parentOfUnsafeHead.Hash,
				L1Origin: nil,
			})
			require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

			// include simple transfer tx in opened block
			{
				to := alice.PlanTransfer(bob.Address(), eth.OneGWei)
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

	// start batcher on chain A
	sys.L2BatcherA.Start()

	// Don't read eth.Unsafe ("latest") for the parent: it can lag the EL's
	// post-forkchoice canonical swap and return the original head, and building on
	// that stale parent would forkchoice the EL back onto the original chain,
	// undoing the reorg. Wait until the divergence height has actually reorged,
	// then extend that confirmed conflicting block explicitly.
	sys.L2ELA.ReorgExact(originalRef_A, 30)
	conflictingHead := sys.L2ELA.BlockRefByNumber(divergenceBlockNumber_A)
	l.Info("EL reflects conflicting block at divergence height",
		"number", conflictingHead.Number, "hash", conflictingHead.Hash, "parent", conflictingHead.ParentID().Hash)

	// sequence a second block extending the confirmed conflicting head
	{
		l.Info("Sequencing with op-test-sequencer (no L1 origin override)")
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   conflictingHead.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
	}

	// Before resuming op-node sequencing, wait until the CL's sync status reflects the
	// test-sequencer's final committed block. The EL is updated synchronously by
	// CommitBlock (via engine.NewPayload + forkchoice update), but the op-node's
	// StatusTracker and Sequencer.latestHead are updated via asynchronous events.
	// Without this wait, StartSequencer() can observe a stale local unsafe head, pass
	// that stale hash to Sequencer.Start (which validates against its equally-stale
	// latestHead), and then build on top of the original chain — silently reorging
	// the EL back off the test-sequencer's conflicting fork.
	expectedUnsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	l.Info("Waiting for op-node local-unsafe to match test-sequencer's committed head",
		"number", expectedUnsafe.Number, "hash", expectedUnsafe.Hash)
	waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
	err := wait.For(waitCtx, 100*time.Millisecond, func() (bool, error) {
		return sys.L2ACL.SyncStatus().UnsafeL2.Hash == expectedUnsafe.Hash, nil
	})
	waitCancel()
	require.NoError(t, err, "op-node never observed test-sequencer's committed unsafe head %s", expectedUnsafe.Hash)

	// continue sequencing with consensus node (op-node)
	sys.L2ACL.StartSequencer()

	sys.L2A.WaitForBlock()

	// Poll for the reorg rather than reading once: ReorgExact waits until the
	// divergence height has a different hash with the same parent. The generous
	// attempt budget tolerates a slow-to-canonicalize op-reth under CI load.
	l.Info("Asserting chain A reorged on divergence block number",
		"number", divergenceBlockNumber_A, "original", originalRef_A.Hash, "parent", originalRef_A.ParentID().Hash)
	sys.L2ELA.ReorgExact(originalRef_A, 30)

	err = wait.For(ctx, 5*time.Second, func() (bool, error) {
		safeL2Head_A_sequencer := sys.L2ACL.SafeL2BlockRef()

		if safeL2Head_A_sequencer.Number <= divergenceBlockNumber_A {
			l.Info("Safe ref number is still behind divergence block number", "divergence", divergenceBlockNumber_A, "safe", safeL2Head_A_sequencer.Number)
			return false, nil
		}
		l.Info("Safe ref advanced past divergence", "sequencer", safeL2Head_A_sequencer.Hash)

		return true, nil
	})
	require.NoError(t, err, "Expected safe ref to advance past divergence")
}
