package seqwindow

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencingWindowExpiry tests that the sequencing window may expire,
// the chain reorgs because of it, and that the chain then recovers.
// This test can take 3 minutes to run.
func TestSequencingWindowExpiry(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)

	// Send a random tx, to ensure there is some activity pre-reorg
	tx1 := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	receipt1, err := tx1.Included.Eval(t.Ctx())
	require.NoError(err)
	t.Logger().Info("Confirmed tx 1", "tx", receipt1.TxHash, "block", receipt1.BlockHash, "number", receipt1.BlockNumber)

	// Wait for the first tx to become cross-safe.
	// We are not interested in the sequencing window to expire and revert all the way back to 0.
	require.Eventually(func() bool {
		stat, err := sys.L2CLA.Escape().RollupAPI().SyncStatus(t.Ctx())
		require.NoError(err)
		return stat.SafeL2.Number > receipt1.BlockNumber.Uint64()
	}, time.Second*45, time.Second, "wait for tx 1 to be safe")
	t.Logger().Info("Tx 1 is safe now")

	// Stop the batcher of chain A, so the L2 unsafe blocks will not get submitted.
	// This may take a while, since the batcher is still actively submitting new data.
	sys.L2BatcherA.Stop()

	stoppedAt := sys.L1Network.WaitForBlock() // wait for new block, in case there is any batch left
	// Make sure the supervisor has synced enough of the L1, for the local-safe query to work.
	sys.Supervisor.AwaitMinL1(stoppedAt.Number)

	// The latest local-safe L2 block is derived from the L1 block with the last batch.
	// After this L1 block the sequence-window expiry starts ticking.
	last, err := sys.Supervisor.Escape().QueryAPI().LocalSafe(t.Ctx(), sys.L2ChainA.ChainID())
	require.NoError(err)

	t.Logger().Info("Safe when stopping batch-submitter",
		"source", last.Source, "derived", last.Derived)
	seqWindowSize := sys.L2ChainA.Escape().RollupConfig().SeqWindowSize
	estimatedExpiryNum := last.Source.Number + seqWindowSize
	lastRef, err := sys.L1EL.Escape().EthClient().BlockRefByHash(t.Ctx(), last.Source.Hash)
	require.NoError(err)
	lastTime := time.Unix(int64(lastRef.Time), 0)
	l1BlockTime := sys.L1EL.EstimateBlockTime()
	windowDuration := l1BlockTime * time.Duration(seqWindowSize)
	t.Logger().Info("Sequencing window expiry",
		"estimateL1Num", estimatedExpiryNum, "windowDuration", windowDuration,
		"fromNow", time.Until(lastTime.Add(windowDuration)))

	// The unsafe L2 block after this last safe block is going to be reorged out
	// once the sequencing window expires.
	// However, since it is empty, it may stay around, because it would be compatible.
	// So let's insert a transaction, then we can be sure it is different.
	tx2 := alice.Transfer(common.HexToAddress("0xdead"), eth.GWei(42))
	receipt2, err := tx2.Included.Eval(t.Ctx())
	require.NoError(err)
	// Now get the block that included the tx. This block will change.
	old := eth.BlockID{Hash: receipt2.BlockHash, Number: receipt2.BlockNumber.Uint64()}
	t.Logger().Info("Confirmed tx 2, which will be reorged out later",
		"tx", receipt2.TxHash, "l2Block", old)
	// The logs will show a "Chain reorg detected" from op-geth.

	// Once this happens we don't want to continue to try and include the tx in a block again,
	// since that will then be reorged out again.
	// We need to enter recovery-mode to not continue to build an incompatible chain that will not get submitted.
	// It may reorg once more, but then stays compatible.
	// For a while we'll have to build blocks that are not going to be reorged out due to subtle L1 origin divergence.
	t.Logger().Info("Turning on recovery-mode")
	t.Require().NoError(sys.L2CLA.Escape().RollupAPI().SetRecoverMode(t.Ctx(), true))

	t.Logger().Info("Waiting for sequencing window expiry induced reorg now")
	// Monitor that the old unsafe chain is reorged out as expected
	require.Eventually(func() bool {
		latest := sys.L2ELA.BlockRefByNumber(old.Number)
		return latest.Hash != old.Hash
	}, windowDuration+time.Second*20, time.Second, "expecting old block to be reorged out")

	// Wait for the tx to no longer be included.
	// The tx-indexer may still return the old block or be stale.
	// So instead, lookup the tx nonce
	require.Eventually(func() bool {
		latestNonce, err := sys.L2ELA.Escape().EthClient().NonceAt(t.Ctx(), alice.Address(), nil)
		if err != nil {
			t.Logger().Error("Failed to look up pending nonce")
			return false
		}
		t.Logger().Info("Checking tx 2 nonce", "latest", latestNonce, "tx2", tx2.Nonce.Value())
		return latestNonce <= tx2.Nonce.Value()
	}, windowDuration+time.Second*20, time.Second, "tx should be reorged out and not come back")

	t.Logger().Info("Waiting for supervisor to surpass pre-reorg chain now")
	// Monitor that the supervisor can continue to sync.
	// A lot more blocks will expire first; the local-safe chain will be entirely force-derived blocks.
	require.Eventually(func() bool {
		safe, err := sys.Supervisor.Escape().QueryAPI().CrossSafe(t.Ctx(), sys.L2ChainA.ChainID())
		require.NoError(err)
		return safe.Source.Number > estimatedExpiryNum
	}, windowDuration+time.Second*20, time.Second, "expecting supervisor to sync cross-safe data, after resolving sequencing window expiry")

	t.Logger().Info("Sanity-checking now")
	// Sanity-check the unsafe head of the supervisor is also updated
	tip, err := sys.Supervisor.Escape().QueryAPI().LocalUnsafe(t.Ctx(), sys.L2ChainA.ChainID())
	require.NoError(err)
	require.True(tip.Number > estimatedExpiryNum)
	// Sanity-check the supervisor is on the right chain
	safe, err := sys.Supervisor.Escape().QueryAPI().CrossSafe(t.Ctx(), sys.L2ChainA.ChainID())
	require.NoError(err)
	other := sys.L2ELA.BlockRefByNumber(safe.Derived.Number)
	require.Equal(safe.Derived.Hash, other.Hash, "supervisor must match chain with EL")

	t.Logger().Info("Re-enabling batch-submitter")
	// re-enable the batcher now that we are done with the test.
	sys.L2BatcherA.Start()
	// TODO(#16036): batcher submits future span batch, misses a L2 block.
	// For now it uses singular batches to work-around.

	// Build the missing blocks, catch up on local-safe chain
	t.Require().Eventually(func() bool {
		status := sys.L2CLA.SyncStatus()
		return status.LocalSafeL2.Number+10 > status.UnsafeL2.Number
	}, windowDuration, time.Second*2, "wait for local-safe to recover near unsafe head")

	// Once we have enough margin to not get reorged again before the batch-submitter acts,
	// exit recovery mode, so we can include txs again.
	t.Logger().Info("Exiting recovery mode")
	t.Require().NoError(sys.L2CLA.Escape().RollupAPI().SetRecoverMode(t.Ctx(), false))

	// Now confirm a tx, chain should be healthy again.
	tx3 := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	receipt3, err := tx3.Included.Eval(t.Ctx())
	require.NoError(err)
	t.Logger().Info("Confirmed tx 3", "tx", receipt3.TxHash, "block", receipt3.BlockHash)

	// Wait for the first tx to become cross-safe.
	// We are not interested in the sequencing window to expire and revert all the way back to 0.
	require.Eventually(func() bool {
		status := sys.L2CLA.SyncStatus()
		t.Logger().Info("Awaiting tx safety",
			"local-unsafe", status.UnsafeL2, "local-safe", status.LocalSafeL2)
		return status.SafeL2.Number > receipt3.BlockNumber.Uint64()
	}, time.Second*60, time.Second, "wait for tx 3 to be safe")
	t.Logger().Info("Tx 3 is safe now")
	// Sanity check the block the tx was included is really still canonical
	got := sys.L2ELA.BlockRefByNumber(receipt3.BlockNumber.Uint64())
	t.Require().Equal(receipt3.BlockHash, got.Hash, "tx 3 was included in canonical block")
}
