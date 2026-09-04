package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

// Public derivation and its counterparty advance after expiry without a private node or batcher.
func TestPrivateOutageDoesNotBlockPublicProgress(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()),
	)
	require := t.Require()
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	require.Eventually(func() bool {
		status, err := sys.L2BSupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.SafeL2.Number > 2
	}, 3*time.Minute, time.Second, "public projection should advance before the outage")

	sys.L2BatcherB.Stop()
	sys.L2BCL.Stop()
	lastPrivate := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.L2ELB.Stop()

	// A failed forced inbox call must not create a dependency, even at projection base fee zero.
	calldata, err := bindings.NewBindings[bindings.CrossL2Inbox]().ValidateMessage(
		messages.Identifier{}, eth.Bytes32{}).EncodeInputLambda()
	require.NoError(err)
	depositor := alice.AsEL(sys.L2BSupernodeEL).ViaDepositTx(alice, sys.L2BSupernodeEL, sys.L2B)
	receipt := depositor.DepositTxExpectRevert(predeploys.CrossL2InboxAddr, calldata, "NotInAccessList()")
	require.Empty(receipt.Logs)

	// Crossing the last private timestamp proves these blocks did not come from queued batches.
	require.Eventually(func() bool {
		projection, err := sys.L2BSupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		if err != nil || projection.SafeL2.Time <= lastPrivate.Time+4 {
			return false
		}
		counterparty, err := sys.L2ASupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && counterparty.SafeL2.Time > lastPrivate.Time+4
	}, 3*time.Minute, time.Second, "both chains must progress beyond the offline private head")

	ref := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
	_, txs, err := sys.L2BSupernodeEL.EthClient().InfoAndTxsByNumber(t.Ctx(), ref.Number)
	require.NoError(err)
	for _, tx := range txs {
		require.True(optypes.IsDepositTx(tx) || optypes.IsPostExecTx(tx), "fallback must contain no sequenced transactions")
	}
	_, receipts, err := sys.L2BSupernodeEL.Escape().L2EthClient().FetchReceipts(t.Ctx(), ref.Hash)
	require.NoError(err)
	for _, receipt := range receipts {
		require.Empty(receipt.Logs, "empty fallback block must contain no interop messages")
	}
}
