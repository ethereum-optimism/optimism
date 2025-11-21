package base

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestFaucetFund(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	tracer := t.Tracer()
	ctx := t.Ctx()

	ctx, span := tracer.Start(ctx, "acquire wallets")
	alice := sys.Wallet.NewEOA(sys.L1EL)
	l1Balance := alice.GetBalance()
	bob := sys.Wallet.NewEOA(sys.L2EL)
	l2Balance := bob.GetBalance()
	span.End()

	_, span = tracer.Start(ctx, "fund wallet")
	fundAmount := eth.OneHundredthEther
	sys.FunderL2.FundNoWait(bob, fundAmount)
	sys.FunderL1.FundNoWait(alice, fundAmount)
	span.End()

	_, span = tracer.Start(ctx, "wait for balance")
	t.Logger().InfoContext(ctx, "funds transferred", "amount", fundAmount)
	alice.WaitForBalance(eth.WeiBig(l1Balance.ToBig().Add(l1Balance.ToBig(), fundAmount.ToBig())))
	bob.WaitForBalance(eth.WeiBig(l2Balance.ToBig().Add(l2Balance.ToBig(), fundAmount.ToBig())))
	span.End()
}
