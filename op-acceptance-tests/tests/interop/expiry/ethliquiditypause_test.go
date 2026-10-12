package expiry

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	superchainConfigSelector      = crypto.Keccak256([]byte("superchainConfig()"))[:4]
	pauseSelector                 = crypto.Keccak256([]byte("pause(address)"))[:4]
	unpauseSelector               = crypto.Keccak256([]byte("unpause(address)"))[:4]
	syncETHLiquidityPauseSelector = crypto.Keccak256([]byte("syncETHLiquidityPause()"))[:4]
	pausedSelector                = crypto.Keccak256([]byte("paused()"))[:4]
	sendETHSelector               = crypto.Keccak256([]byte("sendETH(address,uint256)"))[:4]
)

// TestL1PauseSyncedToETHLiquidity checks that the chain's L1 pause stops the ETH bridge on L2 once
// anyone syncs it: the guardian pauses the superchain on L1, anyone syncs, and a sendETH on A
// reverts; the guardian unpauses, anyone syncs again, and the send goes through and is relayed.
func TestL1PauseSyncedToETHLiquidity(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()
	l1 := sys.L1EL.EthClient()
	elA := sys.L2ELA.EthClient()

	l1Messenger := sys.L2ChainA.L1CrossDomainMessengerAddr()
	out, err := l1.Call(t.Ctx(), ethereum.CallMsg{To: &l1Messenger, Data: superchainConfigSelector}, rpc.LatestBlockNumber)
	require.NoError(err, "failed to read the SuperchainConfig")
	superchainConfig := common.BytesToAddress(out)
	guardian := dsl.NewKey(t, sys.L2ChainA.Escape().Keys().Secret(
		devkeys.SuperchainOperatorKeys(sys.L1Network.ChainID().ToBig())(devkeys.SuperchainConfigGuardianKey))).
		User(sys.L1EL)
	syncer := sys.FunderL1.NewFundedEOA(eth.OneEther)

	// setGlobalPause pauses or unpauses every chain on L1, as the guardian.
	setGlobalPause := func(selector []byte) {
		guardian.Transact(guardian.Plan(), txplan.WithTo(&superchainConfig), txplan.WithData(slices.Concat(selector, make([]byte, 32))))
	}
	// sync syncs A's L1 pause onto its ETHLiquidity and checks that A applied it.
	sync := func(paused bool) {
		rcpt, err := syncer.Transact(syncer.Plan(), txplan.WithTo(&l1Messenger), txplan.WithData(syncETHLiquidityPauseSelector)).
			Included.Eval(t.Ctx())
		require.NoError(err, "the sync was not included")
		deposit := sys.L2ELA.WaitForDeposit(sys.L2ChainA.DepositContractAddr(), rcpt)
		require.Equal(types.ReceiptStatusSuccessful, deposit.Status, "the sync's deposit must execute on A")
		out, err := elA.Call(t.Ctx(), ethereum.CallMsg{To: &predeploys.ETHLiquidityAddr, Data: pausedSelector},
			rpc.BlockNumber(deposit.BlockNumber.Int64()))
		require.NoError(err, "failed to read whether ETHLiquidity is paused")
		require.Equal(paused, common.BytesToHash(out).Big().Sign() != 0, "ETHLiquidity must mirror the L1 pause")
	}

	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	amount := eth.OneHundredthEther
	sendETH := slices.Concat(sendETHSelector, common.LeftPadBytes(recipient.Address().Bytes(), 32),
		common.LeftPadBytes(sys.L2ChainB.ChainID().ToBig().Bytes(), 32))

	setGlobalPause(pauseSelector)
	sync(true)

	// The send is included and reverts with the pause error, leaving ETHLiquidity untouched. Its
	// gas limit is fixed, as estimation would fail on the revert.
	tx := txplan.NewPlannedTx(sender.Plan(), txplan.WithTo(&predeploys.SuperchainETHBridgeAddr),
		txplan.WithValue(amount), txplan.WithData(sendETH), txplan.WithGasLimit(300_000))
	rcpt, err := tx.Included.Eval(t.Ctx())
	require.NoError(err, "the paused sendETH must be included")
	require.Equal(types.ReceiptStatusFailed, rcpt.Status, "sendETH must revert while ETHLiquidity is paused")
	require.Equal(dsl.ErrorSelector("ETHLiquidity_Paused()"), []byte(sys.L2ELA.TraceCalls(rcpt.TxHash).Output),
		"sendETH must revert with the pause error")
	balanceBefore, err := elA.BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr,
		new(big.Int).Sub(rcpt.BlockNumber, big.NewInt(1)))
	require.NoError(err, "failed to read ETHLiquidity's balance")
	balanceAfter, err := elA.BalanceAt(t.Ctx(), predeploys.ETHLiquidityAddr, rcpt.BlockNumber)
	require.NoError(err, "failed to read ETHLiquidity's balance")
	require.Zero(balanceBefore.Cmp(balanceAfter), "a reverted send must not move ETH into ETHLiquidity")

	setGlobalPause(unpauseSelector)
	sync(false)

	send := dsl.SendETH(sender, recipient.Address(), sys.L2ChainB.ChainID(), amount)
	send.Relay(sys.FunderB.NewFundedEOA(eth.OneEther), sys.Supernode())
	recipient.VerifyBalanceExact(amount)
}
