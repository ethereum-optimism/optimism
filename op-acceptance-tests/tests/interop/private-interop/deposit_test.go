package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
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
	// The same forced transaction is inert on the online public projection. The
	// earlier ETH funding deposit also leaves Alice's public balance untouched.
	ref := sys.L2ELB.BlockRefByNumber(bigs.Uint64Strict(receipt.BlockNumber))
	sys.L2BSupernodeEL.WaitL1OriginReached(eth.Unsafe, ref.L1Origin.Number, 120)
	projected := sys.L2BSupernodeEL.WaitForReceipt(receipt.TxHash)
	t.Require().Equal(types.ReceiptStatusSuccessful, projected.Status)
	t.Require().Zero(projected.GasUsed)
	t.Require().Empty(projected.Logs)
	alice.AsEL(sys.L2BSupernodeEL).VerifyBalanceExact(eth.ZeroWei)
	var published txintent.InteropOutput
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

// Deposits can call existing contracts on both chains. Application logs from those
// calls must not become a prefix that shifts the projection's replayed messages.
func TestPrivateDepositLogsDoNotShiftPublishedMessages(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithPrivateInteropChain())
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	wethDeposit, err := w3.MustNewFunc("deposit()", "").EncodeArgs()
	t.Require().NoError(err)
	type aggregateCall struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	// WETH exists on both chains and emits Deposit even for a zero-value call.
	calls := []aggregateCall{
		{Target: predeploys.WETHAddr, CallData: wethDeposit},
		{Target: predeploys.WETHAddr, CallData: wethDeposit},
	}
	for range 2 {
		send := &txintent.SendTrigger{
			Emitter:     predeploys.L2toL2CrossDomainMessengerAddr,
			DestChainID: sys.L2A.ChainID(), Target: alice.Address(),
		}
		calldata, err := send.EncodeInput()
		t.Require().NoError(err)
		// Swallow the projection messenger's unsupported-call revert. Otherwise
		// that revert would erase the WETH logs and conceal the regression.
		calls = append(calls, aggregateCall{Target: send.Emitter, AllowFailure: true, CallData: calldata})
	}
	calldata, err := w3.MustNewFunc(
		"aggregate3((address target,bool allowFailure,bytes callData)[])", "(bool,bytes)[]",
	).EncodeArgs(calls)
	t.Require().NoError(err)
	portalAddr := sys.L2B.DepositContractAddr()
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L1EL.EthClient()), bindings.WithTo(portalAddr), bindings.WithTest(t))
	l1Receipt, err := contractio.Write(portal.DepositTransaction(
		predeploys.MultiCall3Addr, eth.ZeroWei, 1_000_000, false, calldata), t.Ctx(), alice.Plan())
	t.Require().NoError(err)
	t.Require().Equal(types.ReceiptStatusSuccessful, l1Receipt.Status)
	deposits, err := derive.UserDeposits([]*types.Receipt{l1Receipt}, portalAddr)
	t.Require().NoError(err)
	t.Require().Len(deposits, 1)
	sys.L2ELB.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(l1Receipt.BlockNumber), 120)
	privateReceipt := sys.L2ELB.WaitForReceipt(deposits[0].Hash())
	t.Require().Equal(types.ReceiptStatusSuccessful, privateReceipt.Status)
	t.Require().Len(privateReceipt.Logs, 4, "two application logs followed by two initiating messages")
	for i, addr := range []common.Address{
		predeploys.WETHAddr, predeploys.WETHAddr,
		predeploys.L2toL2CrossDomainMessengerAddr, predeploys.L2toL2CrossDomainMessengerAddr,
	} {
		t.Require().Equal(addr, privateReceipt.Logs[i].Address)
	}
	ref := sys.L2ELB.BlockRefByHash(privateReceipt.BlockHash)
	sys.L2BSupernodeEL.WaitL1OriginReached(eth.Unsafe, ref.L1Origin.Number, 120)
	projectedDeposit := sys.L2BSupernodeEL.WaitForReceipt(privateReceipt.TxHash)
	t.Require().Equal(types.ReceiptStatusSuccessful, projectedDeposit.Status)
	t.Require().Empty(projectedDeposit.Logs, "ordinary deposit logs must not precede replayed messages")
	t.Require().Zero(projectedDeposit.GasUsed)

	var output txintent.InteropOutput
	t.Require().NoError(output.FromReceipt(t.Ctx(), privateReceipt, ref.BlockRef(), sys.L2B.ChainID()))
	publicRef := sys.L2BSupernodeEL.BlockRefByNumber(ref.Number)
	_, receipts, err := sys.L2BSupernodeEL.Escape().L2EthClient().FetchReceipts(t.Ctx(), publicRef.Hash)
	t.Require().NoError(err)
	var publicLogs []*types.Log
	for _, receipt := range receipts.Geth() {
		publicLogs = append(publicLogs, receipt.Logs...)
	}
	t.Require().Len(publicLogs, 2, "inspect all public logs, without filtering out an injected prefix")
	for i, log := range publicLogs {
		t.Require().Equal(uint(i), log.Index)
		t.Require().Equal(uint32(i), output.Entries[i+2].Identifier.LogIndex)
		t.Require().Equal(privateReceipt.Logs[i+2].Address, log.Address)
		t.Require().Equal(privateReceipt.Logs[i+2].Topics, log.Topics)
		t.Require().Equal(privateReceipt.Logs[i+2].Data, log.Data)
	}
	receiver := sys.FunderA.NewFundedEOA(eth.OneEther)
	for _, index := range []int{2, 3} {
		relayPrivateMessageAt(t, receiver, sys.L2ELB, sys.L2ASupernodeCL, privateReceipt, index)
	}
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
	relayPrivateMessageAt(t, receiver, privateEL, publicCL, receipt, 0)
}

func relayPrivateMessageAt(t devtest.T, receiver *dsl.EOA, privateEL *dsl.L2ELNode, publicCL *dsl.L2CLNode, receipt *types.Receipt, index int) {
	t.Helper()
	ref := privateEL.BlockRefByNumber(bigs.Uint64Strict(receipt.BlockNumber))
	var output txintent.InteropOutput
	t.Require().NoError(output.FromReceipt(t.Ctx(), receipt, ref.BlockRef(), privateEL.ChainID()))
	tx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](receiver.Plan())
	tx.Content.Set(&txintent.RelayTrigger{
		ExecTrigger: txintent.ExecTrigger{
			Executor: predeploys.L2toL2CrossDomainMessengerAddr,
			Msg:      output.Entries[index],
		},
		Payload: messages.LogToMessagePayload(receipt.Logs[index]),
	})
	relayed, err := tx.PlannedTx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Equal(types.ReceiptStatusSuccessful, relayed.Status)
	dsl.CheckAll(t, publicCL.ReachedWithProgressFn(safety.CrossSafe, safety.LocalUnsafe,
		bigs.Uint64Strict(relayed.BlockNumber), 6*time.Minute, 90*time.Second))
}
