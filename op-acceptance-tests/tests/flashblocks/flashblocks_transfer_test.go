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
// Because flashblocks are speculative, the tx may first appear in a flashblock for
// block N whose payload is later superseded (e.g. the sequencer calls getPayload
// before the builder updates best_payload). The test therefore collects all matching
// flashblocks and verifies the one whose block number equals the final on-chain
// inclusion block.
//
// Expectations:
//
//   - There must be a flashblock whose metadata.receipts contains Alice's tx hash
//     and whose block_number equals the confirmed inclusion block.
//   - Bob's balance in new_account_balances for that flashblock must match the
//     on-chain balance at the inclusion block.
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

	// Accumulate flashblock receipts for the tx, keyed by block number.
	// A speculative flashblock may carry the tx for block N, but if that block is
	// rebuilt the tx surfaces again for a later block. We keep reading until the tx
	// is confirmed on-chain and we have seen a flashblock for its inclusion block.
	balancesByBlock := make(map[uint64]*big.Int)
	var txBlock eth.BlockRef

outer:
	for {
		select {
		case fb, ok := <-fbClient.Next():
			t.Require().True(ok, "client channel closed before we found the transaction")
			if _, found := fb.Metadata.Receipts[txHash]; found {
				var observedBalance *big.Int
				if balanceStr, ok := fb.Metadata.NewAccountBalances[strings.ToLower(bob.Address().Hex())]; ok {
					observedBalance, ok = new(big.Int).SetString(balanceStr[2:], 16)
					t.Require().True(ok)
				}
				balancesByBlock[uint64(fb.Metadata.BlockNumber)] = observedBalance
			}
		case err := <-txCh:
			t.Require().NoError(err)
			txBlock, err = tx.IncludedBlock.Eval(t.Ctx())
			t.Require().NoError(err)
			txCh = nil
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "never found the transaction in flashblock receipts for the confirmed block")
		}

		if _, ok := balancesByBlock[txBlock.Number]; txCh == nil && ok {
			break outer
		}
	}

	expectedBalance, err := sys.L2EL.EthClient().BalanceAt(t.Ctx(), bob.Address(), new(big.Int).SetUint64(txBlock.Number))
	t.Require().NoError(err)
	require.Equal(t, expectedBalance, balancesByBlock[txBlock.Number], "Bob's balance must match the on-chain balance at the transaction inclusion block when reported in flashblock metadata")
}
