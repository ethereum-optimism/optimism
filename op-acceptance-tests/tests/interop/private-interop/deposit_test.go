package privateinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// Ordinary ETH deposits fund the private chain, but cannot initiate interop through its messenger.
func TestPrivateETHDepositCannotSendInterop(gt *testing.T) {
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
	receipt := depositor.DepositTxExpectRevert(predeploys.L2toL2CrossDomainMessengerAddr, calldata,
		"L2ToL2CrossDomainMessenger_UnpaidMessage()")
	t.Require().Empty(receipt.Logs, "forced messaging must not emit interop logs")
}
