package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

func TestDepositCannotValidateMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	sys.L1Network.WaitForOnline()
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	depositor := alice.AsEL(sys.L2ELA).ViaDepositTx(alice, sys.L2ELA, sys.L2ChainA)

	inbox := bindings.NewBindings[bindings.CrossL2Inbox]()
	calldata, err := inbox.ValidateMessage(messages.Identifier{}, eth.Bytes32{}).EncodeInputLambda()
	require.NoError(err, "failed to encode validateMessage calldata")

	depositor.DepositTxExpectRevert(predeploys.CrossL2InboxAddr, calldata, "CrossL2Inbox_NoExecutingDeposits()")
}
