package privateinterop

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/lmittmann/w3"
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
	resender := sys.FunderB.NewFundedEOA(eth.OneEther)
	receiver := sys.FunderA.NewFundedEOA(eth.OneEther)
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

	// A forced claim must not poison the range cursor and prevent the batcher from recovering.
	calldata, err = render.EncodePostClaim(&codec.RangeClaim{LastBlock: ^uint64(0)})
	require.NoError(err)
	depositor.DepositTxExpectRevert(predeploys.ClaimRegistryAddr, calldata, "ClaimRegistry_NotBatcher()")

	// Even a successful direct replay call cannot publish a message through a deposit.
	calldata, err = w3.MustNewFunc("replaySentMessage(uint256,uint256,address,address,bytes)", "bytes32").EncodeArgs(
		sys.L2A.ChainID().ToBig(), big.NewInt(9000), alice.Address(), receiver.Address(), []byte{})
	require.NoError(err)
	forcedReplay := depositor.DepositTx(predeploys.L2toL2CrossDomainMessengerAddr, calldata)
	require.Empty(forcedReplay.Logs, "projection deposits cannot publish initiating events")

	// The private messenger can create this message when it resumes, although the projection's
	// messenger does not execute sendMessage and there is no sequencer batch to publish its event.
	send := &txintent.SendTrigger{
		Emitter:     predeploys.L2toL2CrossDomainMessengerAddr,
		DestChainID: sys.L2A.ChainID(),
		Target:      receiver.Address(),
	}
	calldata, err = send.EncodeInput()
	require.NoError(err)
	missed := depositor.DepositTxExpectRevert(predeploys.L2toL2CrossDomainMessengerAddr, calldata,
		"L2ToL2CrossDomainMessengerReplay_Unsupported()")
	require.Empty(missed.Logs)

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

	sys.L2ELB.Start()
	sys.L2BCL.Start()
	sys.L2BatcherB.Start()
	privateReceipt := sys.L2ELB.WaitForReceipt(missed.TxHash)
	require.Len(privateReceipt.Logs, 1, "the forced send must exist in private state after recovery")
	// Starting the processes is not enough: wait for a fresh published claim, so the resend
	// does not land in another block whose publication window has already expired.
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.LocalSafeL2.Number > lastPrivate.Number
	}, 3*time.Minute, time.Second, "private publication must resume beyond the pre-outage head")
	resent := resendPrivateMessage(t, resender, privateReceipt)
	relayPrivateMessage(t, receiver, sys.L2ELB, sys.L2ASupernodeCL, resent)
}
