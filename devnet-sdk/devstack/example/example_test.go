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

func TestExampleTracing(gt *testing.T) {
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()
	require := t.Require()
	tracer := t.Tracer()
	logger := t.Logger()

	ctx, acquiring := tracer.Start(ctx, "acquiring interop sys")
	sys := SimpleInterop(t)
	acquiring.End()

	ctx, funded := tracer.Start(ctx, "acquiring funded eoa")
	pre := eth.OneEther
	alice := sys.FunderA.NewFundedEOA(pre)
	funded.End()

	ctx, unfunded := tracer.Start(ctx, "acquiring unfunded eoa")
	bob := sys.Wallet.NewEOA(sys.L2ELA)
	bob.VerifyBalanceExact(eth.ZeroWei)
	unfunded.End()

	ctx, transfer := tracer.Start(ctx, "transferring")
	transferred := eth.GWei(42)
	tx := alice.Transfer(bob.Address(), transferred)
	logger.WithContext(ctx).Info("transferred", "amount", transferred, "gas", tx.Included.Value().GasUsed)
	require.Equal(params.TxGas, tx.Included.Value().GasUsed, "transfers cost 21k gas")
	transfer.End()

	_, verifying := tracer.Start(ctx, "verifying")
	alice.VerifyBalanceLessThan(pre.Sub(transferred)) // less than, because of the tx fee
	bob.VerifyBalanceExact(transferred)
	verifying.End()
}
