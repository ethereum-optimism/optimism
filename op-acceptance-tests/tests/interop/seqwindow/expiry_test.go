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
func TestSequencingWindowExpiry(gt *testing.T) {
	t := devtest.SerialT(gt)
	// TODO(#15769): fix sequencing-window expiry interop functionality
	t.Skip("Interop sequencing-window expiry interaction is known to not fully work yet.")

	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)

	// Send a random tx, to ensure there is some activity pre-reorg
	tx1 := alice.Transfer(common.HexToAddress("0x7777"), eth.GWei(100))
	receipt1, err := tx1.Included.Eval(t.Ctx())
	require.NoError(err)
	t.Logger().Info("Confirmed tx 1", "tx", receipt1.TxHash, "block", receipt1.BlockHash)

	// Wait for the first tx to become cross-safe.
	// We are not interested in the sequencing window to expire and revert all the way back to 0.
	require.Eventually(func() bool {
		stat, err := sys.L2CLA.Escape().RollupAPI().SyncStatus(t.Ctx())
		require.NoError(err)
		return stat.SafeL2.Number > receipt1.BlockNumber.Uint64()
	}, time.Second*30, time.Second, "wait for tx 1 to be safe")
	t.Logger().Info("Tx 1 is safe now")

	// Stop the batcher of chain A, so the L2 unsafe blocks will not get submitted.
	require.NoError(sys.L2BatcherA.ActivityAPI().StopBatcher(t.Ctx()))
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

	t.Logger().Info("Waiting for sequencing window expiry induced reorg now")
	// Monitor that the old unsafe chain is reorged out as expected
	require.Eventually(func() bool {
		latest := sys.L2ELA.BlockRefByNumber(old.Number)
		return latest.Hash != old.Hash
	}, windowDuration+time.Second*20, time.Second, "expecting old block to be reorged out")

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

	// re-enable the batcher now that we are done with the test.
	t.Require().NoError(sys.L2BatcherA.ActivityAPI().StartBatcher(t.Ctx()))
}
