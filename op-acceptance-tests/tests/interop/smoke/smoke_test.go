package smoke

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

// TestWrapETH checks WETH interactions, testing both reading and writing on the chain.
// This demonstrates the usage of DSL for contract bindings
func TestWrapETH(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	sys := presets.NewMinimal(t)

	// alice and bob are funded with 0.1 ETH
	alice := sys.Funder.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Funder.NewFundedEOA(eth.OneTenthEther)

	client := sys.L2EL.Escape().EthClient()

	wethAddr := common.HexToAddress(predeploys.WETH)
	// Contract binding preparation
	weth := bindings.NewBindings[bindings.WETH](bindings.WithClient(client), bindings.WithTest(t), bindings.WithTo(wethAddr))

	// Basic sanity check
	require.NotEqual(alice.Address(), bob.Address())

	// Alice and Bob has zero WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(alice.Address())))
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice wraps 0.01 WETH
	alice.Transfer(wethAddr, eth.OneHundredthEther)

	// Read: Alice has 0.01 WETH
	require.Equal(eth.OneHundredthEther, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice wraps 0.01 WETH again
	alice.Transfer(wethAddr, eth.OneHundredthEther)

	// Read: Alice has 0.02 WETH
	require.Equal(eth.TwoHundredthsEther, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Read not using the DSL. Therefore you need to manually error handle and also set context
	_, err := contractio.Read(weth.Transfer(bob.Address(), eth.OneHundredthEther), t.Ctx())
	// Will revert because tx.sender is not set
	require.Error(err)
	// Provide tx.sender using txplan
	// Success because tx.sender(Alice) has enough WETH
	require.True(contract.Read(weth.Transfer(bob.Address(), eth.OneHundredthEther), txplan.WithSender(alice.Address())))

	// Write: Alice sends Bob 0.01 WETH
	contract.Write(alice, weth.Transfer(bob.Address(), eth.OneHundredthEther))

	// Read: Alice has 0.01 WETH
	require.Equal(eth.OneHundredthEther, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0.01 WETH
	require.Equal(eth.OneHundredthEther, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice sends Bob 0.01 WETH
	contract.Write(alice, weth.Transfer(bob.Address(), eth.OneHundredthEther))

	// Read: Alice has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0.02 WETH
	require.Equal(eth.TwoHundredthsEther, contract.Read(weth.BalanceOf(bob.Address())))

	// Throw away WETH for cleanup
	contract.Write(bob, weth.Transfer(common.MaxAddress, eth.TwoHundredthsEther))
}
