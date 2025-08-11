package erc20bridge

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestERC20Bridge(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for this test")

	// Create users with same identity on both chains
	l1User := sys.FunderL1.NewFundedEOA(eth.OneTenthEther)
	l2User := l1User.AsEL(sys.L2EL)
	sys.FunderL2.FundAtLeast(l2User, eth.OneHundredthEther)

	l1TokenAddress := l1User.DeployWETH()
	t.Logger().Info("Deployed WETH token on L1", "address", l1TokenAddress)

	wethContract := bindings.NewBindings[bindings.WETH](
		bindings.WithTest(t),
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(l1TokenAddress),
	)

	mintAmount := eth.OneHundredthEther
	t.Logger().Info("Minting WETH tokens on L1", "amount", mintAmount)
	depositCall := wethContract.Deposit()
	contract.Write(l1User, depositCall, txplan.WithValue(mintAmount))

	l1User.VerifyTokenBalance(l1TokenAddress, mintAmount)
	t.Logger().Info("User has WETH tokens on L1", "balance", mintAmount)

	bridge := dsl.NewStandardBridge(t, sys.L2Chain, nil, sys.L1EL)
	l2TokenAddress := bridge.CreateL2Token(l1TokenAddress, "L2 WETH", "L2WETH", l2User)
	t.Logger().Info("Created L2 token", "address", l2TokenAddress)

	l2User.VerifyTokenBalance(l2TokenAddress, eth.ZeroWei)

	l1BridgeAddress := sys.L2Chain.Escape().Deployment().L1StandardBridgeProxyAddr()

	t.Logger().Info("Approving L1 bridge to spend tokens")
	l1User.ApproveToken(l1TokenAddress, l1BridgeAddress, mintAmount)

	bridgeAmount := eth.GWei(1_000_000) // 0.001 ETH worth
	t.Logger().Info("Bridging tokens from L1 to L2", "amount", bridgeAmount)

	deposit := bridge.ERC20Deposit(l1TokenAddress, l2TokenAddress, bridgeAmount, l1User)
	t.Logger().Info("Bridge deposit confirmed on L1", "gas_cost", deposit.GasCost())

	t.Logger().Info("Waiting for deposit to be processed on L2...")
	l2User.WaitForTokenBalance(l2TokenAddress, bridgeAmount)

	l2User.VerifyTokenBalance(l2TokenAddress, bridgeAmount)
	t.Logger().Info("Successfully verified tokens on L2", "balance", bridgeAmount)

	t.Logger().Info("ERC20 bridge test completed successfully!")
}
