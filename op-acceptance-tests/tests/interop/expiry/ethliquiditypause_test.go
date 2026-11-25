package expiry

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestL1PauseSyncedToETHLiquidity checks that the chain's L1 pause stops the ETH bridge on L2 once
// anyone syncs it: the guardian pauses the superchain on L1, anyone syncs, and a sendETH on A
// reverts; the guardian unpauses, anyone syncs again, and the send goes through and is relayed.
func TestL1PauseSyncedToETHLiquidity(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	l1Messenger := bindings.NewBindings[bindings.L1CrossDomainMessenger](bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2ChainA.L1CrossDomainMessengerAddr()), bindings.WithTest(t))
	superchainConfig := bindings.NewBindings[bindings.SuperchainConfig](bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(contract.Read(l1Messenger.SuperchainConfig())), bindings.WithTest(t))
	ethLiquidityA := bindings.NewBindings[bindings.ETHLiquidity](bindings.WithClient(sys.L2ELA.EthClient()),
		bindings.WithTo(predeploys.ETHLiquidityAddr), bindings.WithTest(t))
	guardian := dsl.NewKey(t, sys.L2ChainA.Escape().Keys().Secret(
		devkeys.SuperchainOperatorKeys(sys.L1Network.ChainID().ToBig())(devkeys.SuperchainConfigGuardianKey))).
		User(sys.L1EL)
	syncer := sys.FunderL1.NewFundedEOA(eth.OneEther)

	// sync syncs A's L1 pause onto its ETHLiquidity and checks that A applied it.
	sync := func(paused bool) {
		deposit := sys.L2ELA.WaitForDeposit(sys.L2ChainA.DepositContractAddr(),
			contract.Write(syncer, l1Messenger.SyncETHLiquidityPause()))
		require.Equal(types.ReceiptStatusSuccessful, deposit.Status, "the sync's deposit must execute on A")
		require.Equal(paused, contract.Read(ethLiquidityA.Paused()), "ETHLiquidity must mirror the L1 pause")
	}

	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	amount := eth.OneHundredthEther

	contract.Write(guardian, superchainConfig.Pause(common.Address{}))
	sync(true)

	// The send is included and reverts with the pause error. Its gas limit is fixed, as estimation
	// would fail on the revert.
	bridgeA := bindings.NewBindings[bindings.SuperchainETHBridge](bindings.WithClient(sys.L2ELA.EthClient()),
		bindings.WithTo(predeploys.SuperchainETHBridgeAddr), bindings.WithTest(t))
	rcpt, err := contractio.Write(bridgeA.SendETH(recipient.Address(), sys.L2ChainB.ChainID()), t.Ctx(),
		sender.Plan(), txplan.WithValue(amount), txplan.WithGasLimit(300_000))
	require.NoError(err, "the paused sendETH must be included")
	require.Equal(types.ReceiptStatusFailed, rcpt.Status, "sendETH must revert while ETHLiquidity is paused")
	require.Equal(dsl.ErrorSelector("ETHLiquidity_Paused()"), []byte(sys.L2ELA.TraceCalls(rcpt.TxHash).Output),
		"sendETH must revert with the pause error")

	contract.Write(guardian, superchainConfig.Unpause(common.Address{}))
	sync(false)

	send := dsl.SendETH(sender, recipient.Address(), sys.L2ChainB.ChainID(), amount)
	send.Relay(sys.FunderB.NewFundedEOA(eth.OneEther), sys.Supernode())
	recipient.VerifyBalanceExact(amount)
}
