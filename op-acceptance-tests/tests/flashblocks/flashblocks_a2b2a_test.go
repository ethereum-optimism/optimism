package flashblocks

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/core/types"
)

// A test that transfers 1 ETH from A to B and then 1 ETH from B to A.
// Should fail if B transfers before B receives the 1.
func TestA2B2A(gt *testing.T) {
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()
	sysgo.SkipOnKonaNode(t, "not supported (fail to get user rpc)")
	sys := presets.NewSingleChainWithFlashblocks(t)

	// Drive a couple blocks on the test sequencer so the faucet L2 funding tx has a chance to land before we rely on it.
	driveViaTestSequencer(t, sys, 2)

	// Subscribe directly to op-rbuilder here: rollup-boost may intentionally drop
	// flashblocks, but this test needs to observe the flashblock carrying Alice's
	// transfer to Bob.
	fbClient := sources.NewFlashblockClient(
		sys.L2OPRBuilder.FlashblocksClient(),
		t.Logger().With("stream_source", "op-rbuilder"),
		100,
	)
	startClient(t, fbClient)

	// 1. Fund Alice.  Alice gets 1 Eth.
	amount := eth.OneEther
	zero := eth.ZeroWei
	alice := sys.FunderL2.NewFundedEOA(amount)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	// 1. Assertions
	t.Require().Equal(amount, alice.GetBalance())
	t.Require().Equal(zero, bob.GetBalance())

	// 2. Alice transfers 0.5 Eth to Bob
	txA2B := alice.Transfer(bob.Address(), amount.Div(2))
	receiptA2B, errA2B := txA2B.Included.Eval(ctx)
	t.Require().NoError(errA2B)
	t.Require().Equal(types.ReceiptStatusSuccessful, receiptA2B.Status)

	// 2. Assertions
	// Alice has at least 0.25 Eth left.
	alice.VerifyBalanceAtLeast(amount.Div(4))
	// Bob now has exactly 0.5 Eth.
	bob.VerifyBalanceExact(amount.Div(2))

	// 3. Bob transfers 0.25 Eth back to Alice
	txB2A := bob.Transfer(alice.Address(), amount.Div(4))
	receiptB2A, errB2A := txB2A.Included.Eval(ctx)
	t.Require().NoError(errB2A)
	t.Require().Equal(types.ReceiptStatusSuccessful, receiptB2A.Status)

	// 3. Assertions
	// Alice now has at lest 0.5 Eth.
	alice.VerifyBalanceAtLeast(amount.Div(2))
	// Bob has at least 0.125 Eth left.
	bob.VerifyBalanceLessThan(amount.Div(8))

}
