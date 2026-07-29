package flashblocks

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/stretchr/testify/require"
)

// txConfirmTimeout bounds how long the test waits for Alice's transfer to land on-chain.
// L2 block time is 2s and the tx is straightforward, so 60s is generous enough to absorb
// CI load and L1 origin churn while still failing fast (with a clear error) when the
// sequencer or builder cannot include user transactions at all.
const txConfirmTimeout = 60 * time.Second

// flashblockObserveTimeout bounds how long the test waits for op-rbuilder to publish a
// flashblock for the confirmed inclusion block. Once the on-chain inclusion block is
// known, the matching flashblock should already have been emitted (it precedes block
// sealing). This timeout exists only to absorb propagation latency from op-rbuilder's
// websocket publisher to the in-process subscriber.
const flashblockObserveTimeout = 30 * time.Second

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
//
// The test is structured as two sequential, separately-timed phases so that a
// failure points clearly at the broken stage:
//
//  1. Wait for Alice's tx to confirm on-chain (bounded by txConfirmTimeout).
//  2. Wait for op-rbuilder to publish a flashblock for the confirmed block
//     (bounded by flashblockObserveTimeout).
//
// If we instead used a single package-level timeout for both phases, every failure
// mode (sequencer broken, builder broken, websocket dead, …) would surface as the
// same misleading "never found the transaction in flashblock receipts" message
// after a 30-minute hang.
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

	// balancesByBlock collects every flashblock receipt that mentions Alice's tx hash,
	// keyed by block number. A speculative flashblock may carry the tx for block N,
	// but if that block is rebuilt the tx surfaces again for a later block. We keep
	// the entire history so that, once we know the on-chain inclusion block, we can
	// look up the matching flashblock without racing the WS stream.
	//
	// observedBlocks is the ordered set of block numbers that emitted a flashblock
	// (regardless of whether they referenced our tx). It feeds the diagnostic message
	// when the test times out — distinguishing "no flashblocks at all" (op-rbuilder
	// broken) from "flashblocks for other blocks but not our tx's block" (a real
	// flashblock-vs-inclusion mismatch).
	balancesByBlock := make(map[uint64]*big.Int)
	observedBlocks := make(map[uint64]struct{})
	var balancesMu sync.Mutex

	collectorCtx, cancelCollector := context.WithCancel(t.Ctx())
	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for {
			select {
			case fb, ok := <-fbClient.Next():
				if !ok {
					// The websocket subscription terminated. Stop collecting; the main
					// goroutine will report a clear failure if we still need a flashblock.
					return
				}
				balancesMu.Lock()
				observedBlocks[uint64(fb.Metadata.BlockNumber)] = struct{}{}
				if _, found := fb.Metadata.Receipts[txHash]; found {
					var observedBalance *big.Int
					if balanceStr, balOk := fb.Metadata.NewAccountBalances[strings.ToLower(bobAddress.Hex())]; balOk {
						parsed, parseOk := new(big.Int).SetString(strings.TrimPrefix(balanceStr, "0x"), 16)
						if parseOk {
							observedBalance = parsed
						}
					}
					balancesByBlock[uint64(fb.Metadata.BlockNumber)] = observedBalance
				}
				balancesMu.Unlock()
			case <-collectorCtx.Done():
				return
			}
		}
	}()
	// Single defer that cancels first, then waits — must be in this order. The
	// collector goroutine parks in `<-fbClient.Next()` until either a new flashblock
	// arrives or its context is cancelled. fbClient's channel is closed by
	// startClient's t.Cleanup, but Go runs t.Cleanup callbacks AFTER the test
	// function's defers, so we cannot rely on the channel close to unpark the
	// collector. We must cancel collectorCtx ourselves to let it return, then
	// wait for it. Two separate defers (cancel + wait) would deadlock because LIFO
	// ordering would run Wait before cancel.
	defer func() {
		cancelCollector()
		collectorWG.Wait()
	}()

	// Phase 1: wait for the tx to land on-chain.
	//
	// Bound this with a short, dedicated timeout so a sequencer/builder pipeline
	// that never produces user blocks fails the test in ~60s with the actual
	// reason ("never confirmed") rather than dragging out to the package timeout
	// and reporting the misleading "never found the transaction in flashblock
	// receipts" message.
	confirmCtx, cancelConfirm := context.WithTimeout(t.Ctx(), txConfirmTimeout)
	defer cancelConfirm()
	t.Require().NoError(waitConfirmed(confirmCtx, tx),
		"Alice's transfer never confirmed on the sequencer EL within %s — likely a sequencer or txpool issue, not a flashblock issue",
		txConfirmTimeout)

	txBlock, err := tx.IncludedBlock.Eval(t.Ctx())
	t.Require().NoError(err)

	// Phase 2: wait for op-rbuilder to emit a flashblock for the inclusion block
	// that carries Alice's tx hash. The collector goroutine is already buffering
	// matching flashblocks in balancesByBlock; we just need to either find an
	// existing match or block until a new one arrives.
	observeCtx, cancelObserve := context.WithTimeout(t.Ctx(), flashblockObserveTimeout)
	defer cancelObserve()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		balancesMu.Lock()
		_, ok := balancesByBlock[txBlock.Number]
		balancesMu.Unlock()
		if ok {
			break
		}
		select {
		case <-observeCtx.Done():
			balancesMu.Lock()
			seenForTxBlock := false
			obs := make([]uint64, 0, len(observedBlocks))
			for n := range observedBlocks {
				obs = append(obs, n)
				if n == txBlock.Number {
					seenForTxBlock = true
				}
			}
			balancesMu.Unlock()
			slices.Sort(obs)
			if len(obs) == 0 {
				t.Require().FailNowf(
					"no flashblocks observed",
					"op-rbuilder published zero flashblocks during the test (tx confirmed at block %d). "+
						"This indicates op-rbuilder isn't producing flashblocks at all — investigate the builder, not the test.",
					txBlock.Number,
				)
			}
			t.Require().FailNowf(
				"never found the transaction in flashblock receipts for the confirmed block",
				"tx %s confirmed at block %d but no flashblock for that block contained the tx receipt. "+
					"flashblocks were observed for blocks %v, seenForTxBlock=%v.",
				txHash, txBlock.Number, obs, seenForTxBlock,
			)
		case <-ticker.C:
		}
	}

	expectedBalance, err := sys.L2EL.EthClient().BalanceAt(t.Ctx(), bob.Address(), new(big.Int).SetUint64(txBlock.Number))
	t.Require().NoError(err)
	balancesMu.Lock()
	observed := balancesByBlock[txBlock.Number]
	balancesMu.Unlock()
	require.Equal(t, expectedBalance, observed, "Bob's balance must match the on-chain balance at the transaction inclusion block when reported in flashblock metadata")
}

// waitConfirmed evaluates tx.Success under a bounded context and returns the
// result, annotating context-deadline errors with a clearer message. The bounded
// context guarantees the wait fails fast when the tx genuinely cannot be included,
// rather than dragging out to the package-wide timeout (which produced a misleading
// "never found the transaction in flashblock receipts" message — see the package
// doc on the test for context).
func waitConfirmed(ctx context.Context, tx *txplan.PlannedTx) error {
	_, err := tx.Success.Eval(ctx)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errors.New("tx never confirmed within the per-phase timeout — see logs for sequencer/builder state")
	}
	return err
}
