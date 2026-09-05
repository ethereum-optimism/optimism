package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
)

// A deposit can initiate a message; the publisher includes it, and anyone can resend it later.
func TestPrivateETHDepositCanSendAndResendInterop(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithPrivateInteropChain())
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	privateAlice := alice.AsEL(sys.L2ELB)
	bridge := dsl.NewStandardBridge(t, sys.L2B, sys.L1EL)
	bridge.Deposit(eth.OneTenthEther, alice)
	privateAlice.VerifyBalanceExact(eth.OneTenthEther)

	send := &txintent.SendTrigger{
		Emitter:     predeploys.L2toL2CrossDomainMessengerAddr,
		DestChainID: sys.L2A.ChainID(),
		Target:      alice.Address(),
	}
	calldata, err := send.EncodeInput()
	t.Require().NoError(err)
	depositor := privateAlice.ViaDepositTx(alice, sys.L2ELB, sys.L2B)
	receipt := depositor.DepositTx(predeploys.L2toL2CrossDomainMessengerAddr, calldata)
	t.Require().Len(receipt.Logs, 1)
	var published txintent.InteropOutput
	ref := sys.L2ELB.BlockRefByNumber(bigs.Uint64Strict(receipt.BlockNumber))
	t.Require().NoError(published.FromReceipt(t.Ctx(), receipt, ref.BlockRef(), sys.L2B.ChainID()))
	t.Require().Len(published.Entries, 1, "the online publisher includes forced initiating messages")

	// A different caller can re-emit the same authenticated payload without repeating the deposit.
	resender := sys.FunderB.NewFundedEOA(eth.OneEther)
	resent := resendPrivateMessage(t, resender, receipt)
	t.Require().Equal(receipt.Logs[0].Topics, resent.Logs[0].Topics)
	t.Require().Equal(receipt.Logs[0].Data, resent.Logs[0].Data)
	t.Require().Greater(bigs.Uint64Strict(resent.BlockNumber), bigs.Uint64Strict(receipt.BlockNumber))
	relayPrivateMessage(t, sys.FunderA.NewFundedEOA(eth.OneEther), sys.L2ELB, sys.L2ASupernodeCL, resent)
}

func resendPrivateMessage(t devtest.T, sender *dsl.EOA, original *types.Receipt) *types.Receipt {
	t.Helper()
	t.Require().Len(original.Logs, 1)
	event := original.Logs[0]
	sent, err := render.DecodeSentMessage(event.Topics, event.Data)
	t.Require().NoError(err)
	calldata, err := w3.MustNewFunc("resendMessage(uint256,uint256,address,address,bytes)", "bytes32").EncodeArgs(
		sent.Destination, sent.Nonce, sent.Sender, sent.Target, sent.Message)
	t.Require().NoError(err)
	messenger := predeploys.L2toL2CrossDomainMessengerAddr
	tx := sender.Transact(sender.Plan(), txplan.WithTo(&messenger), txplan.WithData(calldata))
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Len(receipt.Logs, 1)
	return receipt
}

func relayPrivateMessage(t devtest.T, receiver *dsl.EOA, privateEL *dsl.L2ELNode, publicCL *dsl.L2CLNode, receipt *types.Receipt) {
	t.Helper()
	ref := privateEL.BlockRefByNumber(bigs.Uint64Strict(receipt.BlockNumber))
	var output txintent.InteropOutput
	t.Require().NoError(output.FromReceipt(t.Ctx(), receipt, ref.BlockRef(), privateEL.ChainID()))
	tx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](receiver.Plan())
	tx.Content.Set(&txintent.RelayTrigger{
		ExecTrigger: txintent.ExecTrigger{
			Executor: predeploys.L2toL2CrossDomainMessengerAddr,
			Msg:      output.Entries[0],
		},
		Payload: messages.LogToMessagePayload(receipt.Logs[0]),
	})
	relayed, err := tx.PlannedTx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Equal(types.ReceiptStatusSuccessful, relayed.Status)
	dsl.CheckAll(t, publicCL.ReachedWithProgressFn(safety.CrossSafe, safety.LocalUnsafe,
		bigs.Uint64Strict(relayed.BlockNumber), 6*time.Minute, 90*time.Second))
}
