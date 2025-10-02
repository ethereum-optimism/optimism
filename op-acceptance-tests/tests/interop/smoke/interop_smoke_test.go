package smoke

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	txib "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestInteropSystemNoop(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.SerialT(gt)
	_ = presets.NewMinimal(t)
	t.Log("noop")
}

func TestSmokeTest(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	ctx := t.Ctx()

	user := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)

	l2Client := sys.L2EL.Escape().EthClient()
	weth := txib.NewBindings[txib.WETH](
		txib.WithClient(l2Client),
		txib.WithTo(predeploys.WETHAddr),
		txib.WithTest(t),
	)

	initialBalance, err := contractio.Read(weth.BalanceOf(user.Address()), ctx)
	require.NoError(err)
	t.Logf("Initial WETH balance: %s", initialBalance)

	depositAmount := eth.OneHundredthEther

	tx := user.Transfer(predeploys.WETHAddr, depositAmount)
	receipt, err := tx.Included.Eval(ctx)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	t.Logf("Deposited %s ETH to WETH contract", depositAmount)

	finalBalance, err := contractio.Read(weth.BalanceOf(user.Address()), ctx)
	require.NoError(err)
	t.Logf("Final WETH balance: %s", finalBalance)

	expectedBalance := initialBalance.Add(depositAmount)
	require.Equal(expectedBalance, finalBalance, "WETH balance should have increased by deposited amount")
}

func TestSmokeTestFailure(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	ctx := t.Ctx()

	user := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)

	l2Client := sys.L2EL.Escape().EthClient()
	weth := txib.NewBindings[txib.WETH](
		txib.WithClient(l2Client),
		txib.WithTo(predeploys.WETHAddr),
		txib.WithTest(t),
	)

	initialBalance, err := contractio.Read(weth.BalanceOf(user.Address()), ctx)
	require.NoError(err)
	t.Logf("Initial WETH balance: %s", initialBalance)

	depositAmount := eth.OneEther

	userBalance := user.GetBalance()
	t.Logf("User balance: %s", userBalance)

	require.True(userBalance.Lt(depositAmount), "user should have insufficient funds for this transaction")

	t.Logf("user has insufficient funds: balance=%s, required=%s", userBalance, depositAmount)

	finalBalance, err := contractio.Read(weth.BalanceOf(user.Address()), ctx)
	require.NoError(err)
	t.Logf("Final WETH balance: %s", finalBalance)

	require.Equal(initialBalance, finalBalance, "WETH balance should not have changed")
}
