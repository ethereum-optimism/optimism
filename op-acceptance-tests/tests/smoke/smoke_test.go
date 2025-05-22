package smoke

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TestWrapETH checks WETH interactions, testing both reading and writing on the chain.
// This demonstrates the usage of DSL for contract bindings
func TestWrapETH(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	sys := presets.NewMinimal(t)

	alice := sys.Funder.NewFundedEOA(eth.ThousandEther)
	bob := sys.Funder.NewFundedEOA(eth.ThousandEther)

	client := sys.L2EL.Escape().EthClient()

	// Contract binding preparation
	// Embed EL client for reading the chain
	factory := bindings.NewWETHCallFactory(bindings.WithClient(client))
	// Use default WETH address
	factory.WithDefaultAddr()
	// We can bind other options such as tests later
	factory.ApplyFactoryOptions(bindings.WithTest(t))

	// Initialize bindings from binding factory
	weth := bindings.NewWETH(factory)

	// Fetch default WETH address from binding
	wethAddr, _ := weth.To()

	// Basic sanity check
	require.NotEqual(alice.Address(), bob.Address())

	// Alice and Bob has zero WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(alice.Address())))
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice wraps 1 WETH
	alice.Transfer(*wethAddr, eth.OneEther)

	// Read: Alice has 1 WETH
	require.Equal(eth.OneEther, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice wraps 1 WETH again
	alice.Transfer(*wethAddr, eth.OneEther)

	// Read: Alice has 2 WETH
	require.Equal(eth.Ether(2), contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(bob.Address())))

	// Read not using the DSL. Therefore you need to manually error handle and also set context
	_, err := contractio.Read(weth.Transfer(bob.Address(), eth.OneEther), t.Ctx())
	// Will revert because tx.sender is not set
	require.Error(err)
	// Provide tx.sender using txplan
	// Success because tx.sender(Alice) has enough WETH
	require.True(contract.Read(weth.Transfer(bob.Address(), eth.OneEther), txplan.WithSender(alice.Address())))

	// Write: Alice sends Bob 1 WETH
	contract.Write(alice, weth.Transfer(bob.Address(), eth.OneEther))

	// Read: Alice has 1 WETH
	require.Equal(eth.OneEther, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 1 WETH
	require.Equal(eth.OneEther, contract.Read(weth.BalanceOf(bob.Address())))

	// Write: Alice sends Bob 1 WETH
	contract.Write(alice, weth.Transfer(bob.Address(), eth.OneEther))

	// Read: Alice has 0 WETH
	require.Equal(eth.ZeroWei, contract.Read(weth.BalanceOf(alice.Address())))
	// Read: Bob has 2 WETH
	require.Equal(eth.Ether(2), contract.Read(weth.BalanceOf(bob.Address())))
}
