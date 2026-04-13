package flashblocks

import (
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/stretchr/testify/require"
)

// TestFlashblocksTransfer checks that a transfer is reflected in the op-rbuilder
// flashblock stream for the same block that eventually includes the transaction.
//
// Expectations:
//
//   - There must have been a Flashblock whose metadata.receipts contains Alice's transaction hash.
//     This identifies the flashblock that carried Alice's transfer to Bob.
//   - The transaction's confirmed block number must match the flashblock's block number.
//   - If Bob's balance is reported in new_account_balances for that flashblock, it must match the
//     on-chain balance at the transaction's inclusion block.
func TestFlashblocksTransfer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-30|22:44:52.250]
	// assertions.go:387:             	Error Trace:	/op-devstack/sysgo/l2_cl_kona.go:99
	// assertions.go:387:             	            				/op-devstack/sysgo/mixed_runtime.go:456
	// assertions.go:387:             	            				/op-devstack/sysgo/singlechain_build.go:182
	// assertions.go:387:             	            				/op-devstack/sysgo/singlechain_build.go:276
	// assertions.go:387:             	            				/op-devstack/sysgo/singlechain_flashblocks.go:36
	// assertions.go:387:             	            				/op-devstack/sysgo/singlechain_runtime.go:105
	// assertions.go:387:             	            				/op-devstack/sysgo/singlechain_flashblocks.go:53
	// assertions.go:387:             	            				/op-devstack/presets/flashblocks.go:43
	// assertions.go:387:             	            				/op-acceptance-tests/tests/flashblocks/flashblocks_transfer_test.go:38
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	context deadline exceeded
	// assertions.go:387:             	Test:       	TestFlashblocksTransfer
	// assertions.go:387:             	Messages:   	need user RPC
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

	bob := sys.Wallet.NewEOA(sys.L2EL)
	alice := sys.FunderL2.NewFundedEOA(eth.ThreeHundredthsEther)
	bobAddress := bob.Address()
	tx := txplan.NewPlannedTx(alice.Plan(), txplan.WithTo(&bobAddress), txplan.WithValue(eth.OneHundredthEther))
	signedTx, err := tx.Signed.Eval(t.Ctx())
	t.Require().NoError(err)
	txHash := strings.ToLower(signedTx.Hash().Hex())

	// Buffer the result so cleanup cannot deadlock if we time out before reading it.
	txCh := make(chan error, 1)
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := tx.Success.Eval(t.Ctx())
		txCh <- err
		close(txCh)
	}()

	var blockNumber int
	var observedBalance *big.Int
outer:
	for {
		select {
		case fb, ok := <-fbClient.Next():
			t.Require().True(ok, "client channel closed before we found the transaction")
			if _, found := fb.Metadata.Receipts[txHash]; found {
				blockNumber = fb.Metadata.BlockNumber
				if balanceStr, ok := fb.Metadata.NewAccountBalances[strings.ToLower(bob.Address().Hex())]; ok {
					observedBalance, ok = new(big.Int).SetString(balanceStr[2:], 16)
					t.Require().True(ok)
				}
				break outer
			}
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "never found the transaction in flashblock receipts")
		}
	}

	select {
	case err := <-txCh:
		t.Require().NoError(err)
	case <-t.Ctx().Done():
		t.Require().NoError(t.Ctx().Err())
	}

	txBlock, err := tx.IncludedBlock.Eval(t.Ctx())
	t.Require().NoError(err)
	require.Equal(t, int(txBlock.Number), blockNumber, "the transaction's block number should be the same as the flashblock's block number")

	if observedBalance == nil {
		t.Require().Fail("matched flashblock via receipts but Bob was absent from new_account_balances", "bob=%s txHash=%s", bob.Address(), txHash)
	}

	expectedBalance, err := sys.L2EL.EthClient().BalanceAt(t.Ctx(), bob.Address(), new(big.Int).SetUint64(txBlock.Number))
	t.Require().NoError(err)
	require.Equal(t, expectedBalance, observedBalance, "Bob's balance must match the on-chain balance at the transaction inclusion block when reported in flashblock metadata")
}
