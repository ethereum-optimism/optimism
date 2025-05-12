package base

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestFaucetFund(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)
	tracer := t.Tracer()
	ctx := t.Ctx()

	ctx, span := tracer.Start(ctx, "acquire wallets")
	funded := sys.Funder.NewFundedEOA(eth.Ether(2))
	unfunded := sys.Wallet.NewEOA(sys.L2EL)
	span.End()

	_, span = tracer.Start(ctx, "transfer funds")
	amount := eth.OneEther
	funded.Transfer(unfunded.Address(), amount)
	t.Logger().WithContext(ctx).Info("funds transferred", "amount", amount)
	span.End()
}
