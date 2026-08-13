package sequencing

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// SequencingWindowSystem contains the components needed to exercise sequencing-window expiry.
// It allows the same scenario to run against a standalone consensus client or an op-supernode
// virtual node.
type SequencingWindowSystem struct {
	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode
	L2Network *dsl.L2Network
	L2CL      *dsl.L2CLNode
	L2EL      *dsl.L2ELNode
	L2Batcher *dsl.L2Batcher
	L2Funder  *dsl.FunderEOA
}

// SequencingWindowExpiryOptions configures the shortened sequencing window and batch format used
// by RunSequencingWindowExpiry.
func SequencingWindowExpiryOptions() []presets.Option {
	return []presets.Option{
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// Span-batches during recovery don't appear to align well with the starting-point.
			// It can be off by ~6 L2 blocks, possibly due to off-by-one in L1 block sync
			// considerations in batcher stop or start.
			cfg.BatchType = derive.SingularBatchType
		}),
	}
}

// RunSequencingWindowExpiry tests that the sequencing window may expire, the chain reorgs because
// of it, and the chain then recovers. The supplied consensus client must be an active sequencer.
// This scenario can take 3 minutes to run.
func RunSequencingWindowExpiry(t devtest.T, sys SequencingWindowSystem) {
	require := t.Require()

	alice := sys.L2Funder.NewFundedEOA(eth.OneHundredthEther)

	// Send a random tx, to ensure there is some activity pre-reorg
	tx1 := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	receipt1, err := tx1.Included.Eval(t.Ctx())
	require.NoError(err)
	t.Logger().Info("Confirmed tx 1", "tx", receipt1.TxHash, "block", receipt1.BlockHash, "number", receipt1.BlockNumber)

	// Wait for the first transaction to become safe.
	// We are not interested in the sequencing window expiring and reverting all the way back to 0.
	require.Eventually(func() bool {
		stat, err := sys.L2CL.Escape().RollupAPI().SyncStatus(t.Ctx())
		if err != nil {
			t.Logger().Error("Failed to fetch sync status", "err", err)
			return false
		}
		return stat.SafeL2.Number > bigs.Uint64Strict(receipt1.BlockNumber)
	}, time.Second*45, time.Second, "wait for tx 1 to be safe")
	t.Logger().Info("Tx 1 is safe now")

	// Stop the batcher, so the L2 unsafe blocks will not get submitted.
	// This may take a while, since the batcher is still actively submitting new data.
	sys.L2Batcher.Stop()

	stoppedAt := sys.L1Network.WaitForBlock() // wait for new block, in case there is any batch left
	// Make sure the CL has synced enough of the L1, for the local-safe query to work.
	sys.L2CL.AwaitMinL1Processed(stoppedAt.Number)

	// The latest local-safe L2 block is derived from the L1 chain.
	// After this L1 block the sequence-window expiry starts ticking.
	status := sys.L2CL.SyncStatus()
	lastLocalSafe := status.LocalSafeL2

	t.Logger().Info("Safe when stopping batch-submitter",
		"l1_origin", lastLocalSafe.L1Origin, "local_safe", lastLocalSafe.ID())
	seqWindowSize := sys.L2Network.Escape().RollupConfig().SeqWindowSize
	estimatedExpiryNum := lastLocalSafe.L1Origin.Number + seqWindowSize
	lastRef, err := sys.L1EL.Escape().EthClient().BlockRefByHash(t.Ctx(), lastLocalSafe.L1Origin.Hash)
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
	old := eth.L2BlockRef{Hash: receipt2.BlockHash, Number: bigs.Uint64Strict(receipt2.BlockNumber)}
	t.Logger().Info("Confirmed tx 2, which will be reorged out later",
		"tx", receipt2.TxHash, "l2Block", old)
	// The logs will show a "Chain reorg detected" from the execution client.

	// Once this happens we don't want to continue to try and include the tx in a block again,
	// since that will then be reorged out again.
	// We need to enter recovery-mode to not continue to build an incompatible chain that will not get submitted.
	// It may reorg once more, but then stays compatible.
	// For a while we'll have to build blocks that are not going to be reorged out due to subtle L1 origin divergence.
	t.Logger().Info("Turning on recovery-mode")
	require.NoError(sys.L2CL.SetSequencerRecoverMode(true))

	t.Logger().Info("Waiting for sequencing window expiry induced reorg now", "windowDuration", windowDuration)

	// Monitor that the old unsafe chain is reorged out as expected
	sys.L2EL.ReorgExact(old, 50)

	// Wait for the tx to no longer be included.
	// The tx-indexer may still return the old block or be stale.
	// So instead, lookup the tx nonce
	require.Eventually(func() bool {
		latestNonce, err := sys.L2EL.Escape().EthClient().NonceAt(t.Ctx(), alice.Address(), nil)
		if err != nil {
			t.Logger().Error("Failed to look up pending nonce")
			return false
		}
		t.Logger().Info("Checking tx 2 nonce", "latest", latestNonce, "tx2", tx2.Nonce.Value())
		return latestNonce <= tx2.Nonce.Value()
	}, windowDuration+time.Second*60, 5*time.Second, "tx should be reorged out and not come back")

	t.Logger().Info("Waiting for CL to surpass pre-reorg chain now")
	// Monitor that the CL can continue to sync.
	// A lot more blocks will expire first; the local-safe chain will be entirely force-derived blocks.
	require.Eventually(func() bool {
		stat, err := sys.L2CL.Escape().RollupAPI().SyncStatus(t.Ctx())
		if err != nil {
			t.Logger().Error("Failed to fetch sync status", "err", err)
			return false
		}
		return stat.SafeL2.L1Origin.Number > estimatedExpiryNum
	}, windowDuration+time.Second*60, 5*time.Second, "expecting CL to sync safe data after resolving sequencing window expiry")

	t.Logger().Info("Sanity-checking now")
	// Sanity-check the unsafe head is also updated
	syncStatus := sys.L2CL.SyncStatus()
	require.True(syncStatus.UnsafeL2.L1Origin.Number > estimatedExpiryNum)
	// Sanity-check we are on the right chain
	safeHead := syncStatus.SafeL2
	other := sys.L2EL.BlockRefByNumber(safeHead.Number)
	require.Equal(safeHead.Hash, other.Hash, "CL safe must match chain with EL")

	t.Logger().Info("Re-enabling batch-submitter")
	// re-enable the batcher now that we are done with the test.
	sys.L2Batcher.Start()
	// TODO(#16036): batcher submits future span batch, misses a L2 block.
	// For now it uses singular batches to work-around.

	// Build the missing blocks, catch up on local-safe chain
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 20, 100),
		sys.L2CL.AdvancedFn(safety.LocalUnsafe, 20, 100),
	)

	syncStatus = sys.L2CL.SyncStatus()
	t.Logger().Info("Sync status for L2CL", "local-unsafe", syncStatus.UnsafeL2, "local-safe", syncStatus.LocalSafeL2)

	// Once we have enough margin to not get reorged again before the batch-submitter acts,
	// exit recovery mode, so we can include txs again.
	t.Logger().Info("Exiting recovery mode")
	require.NoError(sys.L2CL.SetSequencerRecoverMode(false))

	// Now confirm a tx, chain should be healthy again.
	tx3 := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	receipt3, err := tx3.Included.Eval(t.Ctx())
	require.NoError(err)
	t.Logger().Info("Confirmed tx 3", "tx", receipt3.TxHash, "block", receipt3.BlockHash)

	// Wait for the transaction to become safe.
	// We are not interested in the sequencing window expiring and reverting all the way back to 0.
	require.Eventually(func() bool {
		status := sys.L2CL.SyncStatus()
		t.Logger().Info("Awaiting tx safety",
			"local-unsafe", status.UnsafeL2, "local-safe", status.LocalSafeL2)
		return status.SafeL2.Number > bigs.Uint64Strict(receipt3.BlockNumber)
	}, time.Second*60, time.Second, "wait for tx 3 to be safe")
	t.Logger().Info("Tx 3 is safe now")
	// Sanity check the block the tx was included is really still canonical
	got := sys.L2EL.BlockRefByNumber(bigs.Uint64Strict(receipt3.BlockNumber))
	require.Equal(receipt3.BlockHash, got.Hash, "tx 3 was included in canonical block")
}
