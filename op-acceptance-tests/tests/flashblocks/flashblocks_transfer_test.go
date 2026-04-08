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

// TestFlashblocksTransfer checks that a transfer gets reflected in a flashblock before the transaction is confirmed in a block
//
// Expectations:
//
//   - There must have been a Flashblock containing a new_account_balance corresponding to Bob's
//     account. This flashblock would be representative of the flashblock including Alice-to-Bob
//     transaction.
//   - Bob's balance reported in the flashblock must match the on-chain balance after confirmation.
//   - The transaction's confirmed block number must match the flashblock's block number.
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

	fbClient := sources.NewFlashblockClient(
		sys.L2RollupBoost.FlashblocksClient(),
		t.Logger().With("stream_source", "rollup-boost"),
		100,
	)
	startClient(t, fbClient)

	bob := sys.Wallet.NewEOA(sys.L2EL)
	txCh := make(chan *txplan.PlannedTx)
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		alice := sys.FunderL2.NewFundedEOA(eth.ThreeHundredthsEther)
		bobAddress := bob.Address()
		txCh <- alice.Transact(alice.Plan(), txplan.WithTo(&bobAddress), txplan.WithValue(eth.OneHundredthEther))
		close(txCh)
	}()

	var blockNumber int
	var observedBalance *big.Int
outer:
	for {
		select {
		case fb, ok := <-fbClient.Next():
			t.Require().True(ok, "client channel closed before we found the transaction")
			balanceStr, found := fb.Metadata.NewAccountBalances[strings.ToLower(bob.Address().Hex())]
			if found {
				blockNumber = fb.Metadata.BlockNumber
				observedBalance, ok = new(big.Int).SetString(balanceStr[2:], 16)
				t.Require().True(ok)
				break outer
			}
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "never found the transaction")
		}
	}

	var txBlock eth.BlockRef
	select {
	case tx, ok := <-txCh:
		t.Require().True(ok)
		var err error
		txBlock, err = tx.IncludedBlock.Eval(t.Ctx())
		t.Require().NoError(err)
	case <-t.Ctx().Done():
		t.Require().NoError(t.Ctx().Err())
	}

	expectedBalance, err := sys.L2EL.EthClient().BalanceAt(t.Ctx(), bob.Address(), new(big.Int).SetUint64(txBlock.Number))
	t.Require().NoError(err)
	require.Equal(t, expectedBalance, observedBalance, "Bob's balance must be correct as per exactly what Alice transferred to them")
	require.Equal(t, int(txBlock.Number), blockNumber, "the transaction's block number should be the same as the flashblock's block number")
}
