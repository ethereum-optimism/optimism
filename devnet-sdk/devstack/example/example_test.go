package example

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
)

// TestExample1 starts an interop chain and verifies that the local unsafe head advances.
func TestExample1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := SimpleInterop(t)

	t.Require().NotEqual(sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID(), "sanity-check we have two different chains")
	sys.Supervisor.VerifySyncStatus(dsl.WithAllLocalUnsafeHeadsAdvancedBy(10))
}

func TestExample2(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := SimpleInterop(t)

	sys.Supervisor.VerifySyncStatus(dsl.WithAllLocalUnsafeHeadsAdvancedBy(4))
}

func TestExampleTxs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := SimpleInterop(t)
	require := t.Require()

	pre := eth.OneEther
	alice := sys.FunderA.NewFundedEOA(pre)

	bob := sys.Wallet.NewEOA(sys.L2ELA)
	bob.VerifyBalanceExact(eth.ZeroWei)

	transferred := eth.GWei(42)
	tx := alice.Transfer(bob.Address(), transferred)
	require.Equal(params.TxGas, tx.Included.Value().GasUsed, "transfers cost 21k gas")

	alice.VerifyBalanceLessThan(pre.Sub(transferred)) // less than, because of the tx fee
	bob.VerifyBalanceExact(transferred)
}
