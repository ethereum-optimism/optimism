package rules

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/flashblocks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestBoostPriorityOrdering validates that transactions to addresses with higher
// boost weights appear earlier in blocks than transactions to lower-weight addresses.
//
// Rules configuration (from init_test.go):
// - HighPriorityRecipient (0x2222...): weight 5000
// - MediumPriorityRecipient (0x3333...): weight 2000
// - LowPriorityRecipient (0x4444...): weight 500
// - No boost for other addresses: weight 0
//
// Expected ordering: High (5000) > Medium (2000) > Low (500) > Normal (0)
// We verify this by checking TransactionIndex in the block - lower index = earlier in block.
func TestBoostPriorityOrdering(gt *testing.T) {
	t := devtest.SerialT(gt)
	skipIfRulesNotEnabled(t)

	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()

	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test boost priority ordering")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 90*time.Second)
	defer cancel()

	flashblocks.DriveViaTestSequencer(t, sys, 3)

	const maxRetries = 3
	err := retry.Do0(ctx, maxRetries, &retry.FixedStrategy{Dur: 0}, func() error {
		fundAmount := eth.Ether(1)
		senderHigh := sys.FunderL2.NewFundedEOA(fundAmount)
		senderMedium := sys.FunderL2.NewFundedEOA(fundAmount)
		senderLow := sys.FunderL2.NewFundedEOA(fundAmount)
		senderNormal := sys.FunderL2.NewFundedEOA(fundAmount)

		normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
		normalRecipientAddr := normalRecipient.Address()

		sendAmount := eth.OneHundredthEther
		var wg sync.WaitGroup
		var txHigh, txMedium, txLow, txNormal *txplan.PlannedTx

		wg.Add(4)
		go func() {
			defer wg.Done()
			txHigh = senderHigh.Transact(
				senderHigh.Plan(),
				txplan.WithTo(&HighPriorityRecipient),
				txplan.WithValue(sendAmount),
			)
		}()
		go func() {
			defer wg.Done()
			txMedium = senderMedium.Transact(
				senderMedium.Plan(),
				txplan.WithTo(&MediumPriorityRecipient),
				txplan.WithValue(sendAmount),
			)
		}()
		go func() {
			defer wg.Done()
			txLow = senderLow.Transact(
				senderLow.Plan(),
				txplan.WithTo(&LowPriorityRecipient),
				txplan.WithValue(sendAmount),
			)
		}()
		go func() {
			defer wg.Done()
			txNormal = senderNormal.Transact(
				senderNormal.Plan(),
				txplan.WithTo(&normalRecipientAddr),
				txplan.WithValue(sendAmount),
			)
		}()
		wg.Wait()

		receiptHigh, err := txHigh.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("high priority tx inclusion: %w", err)
		}
		receiptMedium, err := txMedium.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("medium priority tx inclusion: %w", err)
		}
		receiptLow, err := txLow.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("low priority tx inclusion: %w", err)
		}
		receiptNormal, err := txNormal.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("normal tx inclusion: %w", err)
		}

		logger.Info("All transactions confirmed",
			"high_block", receiptHigh.BlockNumber, "high_index", receiptHigh.TransactionIndex,
			"medium_block", receiptMedium.BlockNumber, "medium_index", receiptMedium.TransactionIndex,
			"low_block", receiptLow.BlockNumber, "low_index", receiptLow.TransactionIndex,
			"normal_block", receiptNormal.BlockNumber, "normal_index", receiptNormal.TransactionIndex,
		)

		sameBlock := receiptHigh.BlockNumber.Cmp(receiptMedium.BlockNumber) == 0 &&
			receiptMedium.BlockNumber.Cmp(receiptLow.BlockNumber) == 0 &&
			receiptLow.BlockNumber.Cmp(receiptNormal.BlockNumber) == 0

		if !sameBlock {
			return fmt.Errorf("transactions landed in different blocks: high=%d, medium=%d, low=%d, normal=%d",
				receiptHigh.BlockNumber, receiptMedium.BlockNumber, receiptLow.BlockNumber, receiptNormal.BlockNumber)
		}

		require.Less(t, receiptHigh.TransactionIndex, receiptMedium.TransactionIndex,
			"high priority (weight 5000) should have lower tx index than medium priority (weight 2000)")
		require.Less(t, receiptMedium.TransactionIndex, receiptLow.TransactionIndex,
			"medium priority (weight 2000) should have lower tx index than low priority (weight 500)")
		require.Less(t, receiptLow.TransactionIndex, receiptNormal.TransactionIndex,
			"low priority (weight 500) should have lower tx index than normal (no boost)")

		logger.Info("Boost priority ordering verified successfully",
			"order", fmt.Sprintf("high(idx=%d) < medium(idx=%d) < low(idx=%d) < normal(idx=%d)",
				receiptHigh.TransactionIndex, receiptMedium.TransactionIndex,
				receiptLow.TransactionIndex, receiptNormal.TransactionIndex),
		)
		return nil
	})
	require.NoError(t, err, "boost priority ordering verification failed")
}

// TestBoostedVsNonBoostedOrdering validates that a boosted transaction appears before
// a non-boosted transaction even when the non-boosted transaction has a MUCH HIGHER
// priority fee (gas tip). This proves that rule-based boost takes precedence over
// economic incentives (EIP-1559 priority fees).
//
// Test setup:
// - Boosted tx: to BoostedRecipient (weight 1000), LOW priority fee (1 gwei tip)
// - Normal tx: to normal recipient (no boost), HIGH priority fee (100 gwei tip)
//
// Expected: Despite 100x higher gas tip, the normal tx should come AFTER the boosted tx.
func TestBoostedVsNonBoostedOrdering(gt *testing.T) {
	t := devtest.SerialT(gt)
	skipIfRulesNotEnabled(t)

	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()

	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test boosted vs non-boosted ordering")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 90*time.Second)
	defer cancel()

	flashblocks.DriveViaTestSequencer(t, sys, 2)

	lowGasTip := big.NewInt(1_000_000_000)
	highGasTip := big.NewInt(100_000_000_000)
	highGasFeeCap := big.NewInt(200_000_000_000)

	const maxRetries = 3
	err := retry.Do0(ctx, maxRetries, &retry.FixedStrategy{Dur: 0}, func() error {
		fundAmount := eth.ThreeHundredthsEther
		senderBoosted := sys.FunderL2.NewFundedEOA(fundAmount)
		senderNormal := sys.FunderL2.NewFundedEOA(fundAmount)

		normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
		normalRecipientAddr := normalRecipient.Address()

		sendAmount := eth.OneHundredthEther
		var wg sync.WaitGroup
		var txBoosted, txNormal *txplan.PlannedTx

		wg.Add(2)
		go func() {
			defer wg.Done()
			txBoosted = senderBoosted.Transact(
				senderBoosted.Plan(),
				txplan.WithTo(&BoostedRecipient),
				txplan.WithValue(sendAmount),
				txplan.WithGasTipCap(lowGasTip),
			)
		}()
		go func() {
			defer wg.Done()
			txNormal = senderNormal.Transact(
				senderNormal.Plan(),
				txplan.WithTo(&normalRecipientAddr),
				txplan.WithValue(sendAmount),
				txplan.WithGasTipCap(highGasTip),
				txplan.WithGasFeeCap(highGasFeeCap),
			)
		}()
		wg.Wait()

		receiptBoosted, err := txBoosted.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("boosted tx inclusion: %w", err)
		}
		receiptNormal, err := txNormal.Included.Eval(ctx)
		if err != nil {
			return fmt.Errorf("normal tx inclusion: %w", err)
		}

		logger.Info("Transactions confirmed",
			"boosted_block", receiptBoosted.BlockNumber,
			"boosted_index", receiptBoosted.TransactionIndex,
			"normal_block", receiptNormal.BlockNumber,
			"normal_index", receiptNormal.TransactionIndex,
		)

		if receiptBoosted.BlockNumber.Cmp(receiptNormal.BlockNumber) != 0 {
			return fmt.Errorf("transactions landed in different blocks: boosted=%d, normal=%d",
				receiptBoosted.BlockNumber, receiptNormal.BlockNumber)
		}

		require.Less(t, receiptBoosted.TransactionIndex, receiptNormal.TransactionIndex,
			"boosted transaction (weight 1000, 1 gwei tip) should have lower tx index than "+
				"normal transaction (no boost, 100 gwei tip) - proving rules > gas priority")

		logger.Info("Rule-based boost precedence over gas priority verified!",
			"boosted_index", receiptBoosted.TransactionIndex,
			"normal_index", receiptNormal.TransactionIndex,
		)
		return nil
	})
	require.NoError(t, err, "boosted vs non-boosted ordering verification failed")
}

// TestSameSenderNonceOrdering verifies that transactions from the same sender
// maintain nonce ordering regardless of boost rules.
func TestSameSenderNonceOrdering(gt *testing.T) {
	t := devtest.SerialT(gt)
	skipIfRulesNotEnabled(t)

	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()

	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test same sender nonce ordering")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 60*time.Second)
	defer cancel()

	// Drive initial blocks
	flashblocks.DriveViaTestSequencer(t, sys, 2)

	// Create a single funded sender
	sender := sys.FunderL2.NewFundedEOA(eth.Ether(1))

	// Create normal recipient
	normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
	normalRecipientAddr := normalRecipient.Address()

	logger.Info("Test sender created", "address", sender.Address().Hex())

	sendAmount := eth.OneHundredthEther

	// Send 3 sequential transactions from same sender to different recipients
	// TX0 -> Normal recipient (no boost)
	// TX1 -> HighPriorityRecipient (weight 5000)
	// TX2 -> Normal recipient (no boost)
	//
	// Even though TX1 has the highest boost, it must come after TX0 due to nonce ordering

	// TX0: to normal recipient
	tx0 := sender.Transact(
		sender.Plan(),
		txplan.WithTo(&normalRecipientAddr),
		txplan.WithValue(sendAmount),
	)
	receipt0, err := tx0.Included.Eval(ctx)
	require.NoError(t, err, "tx0 should be included")

	// TX1: to high priority recipient
	tx1 := sender.Transact(
		sender.Plan(),
		txplan.WithTo(&HighPriorityRecipient),
		txplan.WithValue(sendAmount),
	)
	receipt1, err := tx1.Included.Eval(ctx)
	require.NoError(t, err, "tx1 should be included")

	// TX2: to normal recipient
	tx2 := sender.Transact(
		sender.Plan(),
		txplan.WithTo(&normalRecipientAddr),
		txplan.WithValue(sendAmount),
	)
	receipt2, err := tx2.Included.Eval(ctx)
	require.NoError(t, err, "tx2 should be included")

	logger.Info("Sequential transactions confirmed",
		"tx0_hash", receipt0.TxHash.Hex(), "tx0_block", receipt0.BlockNumber, "tx0_index", receipt0.TransactionIndex,
		"tx1_hash", receipt1.TxHash.Hex(), "tx1_block", receipt1.BlockNumber, "tx1_index", receipt1.TransactionIndex,
		"tx2_hash", receipt2.TxHash.Hex(), "tx2_block", receipt2.BlockNumber, "tx2_index", receipt2.TransactionIndex,
	)

	// Verify nonce ordering is preserved
	// If transactions are in the same block, their indices must reflect nonce order
	if receipt0.BlockNumber.Cmp(receipt1.BlockNumber) == 0 {
		require.Less(t, receipt0.TransactionIndex, receipt1.TransactionIndex,
			"tx0 (nonce N) must have lower index than tx1 (nonce N+1) despite tx1 having higher boost")
	} else {
		// If in different blocks, tx0's block must be <= tx1's block
		require.LessOrEqual(t,
			bigs.Uint64Strict(receipt0.BlockNumber),
			bigs.Uint64Strict(receipt1.BlockNumber),
			"tx0 must be in same or earlier block than tx1",
		)
	}

	if receipt1.BlockNumber.Cmp(receipt2.BlockNumber) == 0 {
		require.Less(t, receipt1.TransactionIndex, receipt2.TransactionIndex,
			"tx1 (nonce N+1) must have lower index than tx2 (nonce N+2)")
	} else {
		require.LessOrEqual(t,
			bigs.Uint64Strict(receipt1.BlockNumber),
			bigs.Uint64Strict(receipt2.BlockNumber),
			"tx1 must be in same or earlier block than tx2",
		)
	}

	logger.Info("Nonce ordering verified - boost rules do not break same-sender ordering")
}

// TestMultipleSendersWithMixedPriorities tests a realistic scenario with multiple
// senders sending to different priority recipients concurrently.
func TestMultipleSendersWithMixedPriorities(gt *testing.T) {
	t := devtest.SerialT(gt)
	skipIfRulesNotEnabled(t)

	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()

	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test multiple senders mixed priorities")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 120*time.Second)
	defer cancel()

	flashblocks.DriveViaTestSequencer(t, sys, 2)

	type senderConfig struct {
		eoa       *dsl.EOA
		priority  string
		recipient common.Address
		weight    int
	}

	type txResult struct {
		receipt  *types.Receipt
		priority string
		weight   int
	}

	const maxRetries = 3
	err := retry.Do0(ctx, maxRetries, &retry.FixedStrategy{Dur: 0}, func() error {
		fundAmount := eth.ThreeHundredthsEther

		normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
		normalRecipientAddr := normalRecipient.Address()

		configs := []struct {
			priority  string
			recipient common.Address
			weight    int
		}{
			{"high", HighPriorityRecipient, 5000},
			{"medium", MediumPriorityRecipient, 2000},
			{"low", LowPriorityRecipient, 500},
			{"normal", normalRecipientAddr, 0},
			{"high", HighPriorityRecipient, 5000},
			{"normal", normalRecipientAddr, 0},
		}

		senders := make([]senderConfig, len(configs))
		for i, cfg := range configs {
			senders[i] = senderConfig{
				eoa:       sys.FunderL2.NewFundedEOA(fundAmount),
				priority:  cfg.priority,
				recipient: cfg.recipient,
				weight:    cfg.weight,
			}
		}

		sendAmount := eth.OneHundredthEther
		var wg sync.WaitGroup
		plannedTxs := make([]*txplan.PlannedTx, len(senders))

		for i := range senders {
			wg.Add(1)
			idx := i
			go func() {
				defer wg.Done()
				recipient := senders[idx].recipient
				plannedTxs[idx] = senders[idx].eoa.Transact(
					senders[idx].eoa.Plan(),
					txplan.WithTo(&recipient),
					txplan.WithValue(sendAmount),
				)
			}()
		}
		wg.Wait()

		results := make([]txResult, len(senders))
		for i := range senders {
			receipt, err := plannedTxs[i].Included.Eval(ctx)
			if err != nil {
				return fmt.Errorf("tx%d inclusion: %w", i, err)
			}
			results[i] = txResult{
				receipt:  receipt,
				priority: senders[i].priority,
				weight:   senders[i].weight,
			}
			logger.Info("Transaction confirmed",
				"index", i,
				"priority", senders[i].priority,
				"weight", senders[i].weight,
				"block", receipt.BlockNumber,
				"tx_index", receipt.TransactionIndex,
			)
		}

		blockGroups := make(map[uint64][]txResult)
		for _, r := range results {
			blockNum := bigs.Uint64Strict(r.receipt.BlockNumber)
			blockGroups[blockNum] = append(blockGroups[blockNum], r)
		}

		if len(blockGroups) != 1 {
			blockNumbers := make([]uint64, 0, len(blockGroups))
			for blockNum := range blockGroups {
				blockNumbers = append(blockNumbers, blockNum)
			}
			return fmt.Errorf("transactions landed in %d different blocks: %v", len(blockGroups), blockNumbers)
		}

		for blockNum, txs := range blockGroups {
			logger.Info("Verifying ordering in block", "block", blockNum, "tx_count", len(txs))

			for i := 0; i < len(txs); i++ {
				for j := i + 1; j < len(txs); j++ {
					if txs[i].weight > txs[j].weight {
						require.Less(t, txs[i].receipt.TransactionIndex, txs[j].receipt.TransactionIndex,
							"tx with weight %d should have lower index than tx with weight %d in block %d",
							txs[i].weight, txs[j].weight, blockNum)
					} else if txs[i].weight < txs[j].weight {
						require.Greater(t, txs[i].receipt.TransactionIndex, txs[j].receipt.TransactionIndex,
							"tx with weight %d should have higher index than tx with weight %d in block %d",
							txs[i].weight, txs[j].weight, blockNum)
					}
				}
			}
		}

		logger.Info("Multiple senders mixed priorities test completed successfully")
		return nil
	})
	require.NoError(t, err, "multiple senders mixed priorities verification failed")
}

// TestSingleSenderRandomNonceOrderWithRandomScores sends 10 transactions from a single sender
// with explicit nonces, random gas tips, and random boost recipients. The transactions are
// submitted to the mempool in a shuffled nonce order to verify that op-rbuilder correctly
// sorts by transaction score while still respecting nonce ordering for the same sender.
//
// Setup:
//   - 1 sender, 10 transactions (nonces base+0 through base+9)
//   - Each tx gets a random gas tip (1-50 gwei) AND a random recipient from
//     {HighPriority, MediumPriority, LowPriority, Normal} for varied boost weights
//   - Transactions are shuffled before submission so the mempool receives them out-of-order
//
// Expected: All 10 transactions are included with strict nonce ordering preserved,
// i.e. within the same block tx with nonce N always has a lower TransactionIndex
// than tx with nonce N+1, and across blocks lower nonces are in earlier blocks.
func TestSingleSenderRandomNonceOrderWithRandomScores(gt *testing.T) {
	t := devtest.SerialT(gt)
	skipIfRulesNotEnabled(t)

	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()

	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test single sender random nonce order with random scores")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 120*time.Second)
	defer cancel()

	flashblocks.DriveViaTestSequencer(t, sys, 2)

	const txCount = 10

	type recipientInfo struct {
		addr   common.Address
		weight int
	}
	normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
	normalRecipientAddr := normalRecipient.Address()
	recipients := []recipientInfo{
		{HighPriorityRecipient, 5000},
		{MediumPriorityRecipient, 2000},
		{LowPriorityRecipient, 500},
		{normalRecipientAddr, 0},
	}

	const maxRetries = 3
	err := retry.Do0(ctx, maxRetries, &retry.FixedStrategy{Dur: 0}, func() error {
		sender := sys.FunderL2.NewFundedEOA(eth.Ether(1))
		baseNonce := sender.PendingNonce()

		logger.Info("Test sender created",
			"address", sender.Address().Hex(),
			"baseNonce", baseNonce,
		)

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		type txConfig struct {
			nonce     uint64
			tipGwei   int64
			recipient recipientInfo
		}
		configs := make([]txConfig, txCount)
		for i := 0; i < txCount; i++ {
			tipGwei := int64(1 + rng.Intn(50))
			recip := recipients[rng.Intn(len(recipients))]
			configs[i] = txConfig{
				nonce:     baseNonce + uint64(i),
				tipGwei:   tipGwei,
				recipient: recip,
			}
			logger.Info("Transaction config",
				"index", i,
				"nonce", configs[i].nonce,
				"tipGwei", tipGwei,
				"recipient", recip.addr.Hex(),
				"boostWeight", recip.weight,
			)
		}

		submitOrder := rng.Perm(txCount)
		logger.Info("Shuffled submission order", "order", submitOrder)

		sendAmount := eth.OneHundredthEther
		highFeeCap := new(big.Int).Mul(big.NewInt(200), big.NewInt(1_000_000_000))

		plannedTxs := make([]*txplan.PlannedTx, txCount)
		var wg sync.WaitGroup
		for _, idx := range submitOrder {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				cfg := configs[i]
				tip := new(big.Int).Mul(big.NewInt(cfg.tipGwei), big.NewInt(1_000_000_000))
				recipAddr := cfg.recipient.addr
				plannedTxs[i] = sender.Transact(
					sender.Plan(),
					txplan.WithStaticNonce(cfg.nonce),
					txplan.WithTo(&recipAddr),
					txplan.WithValue(sendAmount),
					txplan.WithGasTipCap(tip),
					txplan.WithGasFeeCap(highFeeCap),
				)
			}(idx)
		}
		wg.Wait()

		type txResult struct {
			nonce   uint64
			tipGwei int64
			weight  int
			receipt *types.Receipt
		}
		results := make([]txResult, txCount)
		for i := 0; i < txCount; i++ {
			receipt, err := plannedTxs[i].Included.Eval(ctx)
			if err != nil {
				return fmt.Errorf("tx%d (nonce %d) inclusion: %w", i, configs[i].nonce, err)
			}
			results[i] = txResult{
				nonce:   configs[i].nonce,
				tipGwei: configs[i].tipGwei,
				weight:  configs[i].recipient.weight,
				receipt: receipt,
			}
			logger.Info("Transaction confirmed",
				"index", i,
				"nonce", configs[i].nonce,
				"tipGwei", configs[i].tipGwei,
				"boostWeight", configs[i].recipient.weight,
				"block", receipt.BlockNumber,
				"txIndex", receipt.TransactionIndex,
			)
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].nonce < results[j].nonce
		})

		// Count how many transactions share a block with at least one other tx.
		// If every tx lands in its own block, nonce ordering is trivially
		// guaranteed by the protocol and the test does not exercise the builder's
		// intra-block ordering logic.  Treat that as a retryable condition so the
		// next attempt (with a fresh sender/nonces) hopefully lands more txs in
		// the same block.
		blockCounts := make(map[uint64]int)
		for _, r := range results {
			blockCounts[bigs.Uint64Strict(r.receipt.BlockNumber)]++
		}
		maxPerBlock := 0
		for _, c := range blockCounts {
			if c > maxPerBlock {
				maxPerBlock = c
			}
		}
		if maxPerBlock < 2 {
			return fmt.Errorf("all %d transactions landed in separate blocks (%d blocks); "+
				"need at least 2 in the same block to validate intra-block nonce ordering",
				txCount, len(blockCounts))
		}

		for i := 0; i < len(results)-1; i++ {
			cur := results[i]
			next := results[i+1]

			if cur.receipt.BlockNumber.Cmp(next.receipt.BlockNumber) == 0 {
				require.Less(t, cur.receipt.TransactionIndex, next.receipt.TransactionIndex,
					"nonce %d (tip=%d gwei, boost=%d, txIdx=%d) must have lower tx index than nonce %d (tip=%d gwei, boost=%d, txIdx=%d) in block %d",
					cur.nonce, cur.tipGwei, cur.weight, cur.receipt.TransactionIndex,
					next.nonce, next.tipGwei, next.weight, next.receipt.TransactionIndex,
					cur.receipt.BlockNumber)
			} else {
				require.Less(
					t,
					bigs.Uint64Strict(cur.receipt.BlockNumber),
					bigs.Uint64Strict(next.receipt.BlockNumber),
					"nonce %d must be in an earlier block than nonce %d (got blocks %d and %d)",
					cur.nonce, next.nonce, cur.receipt.BlockNumber, next.receipt.BlockNumber,
				)
			}
		}

		logger.Info("Single sender random nonce order test passed - nonce ordering preserved despite random scores and shuffled submission",
			"txCount", txCount,
			"blocksUsed", len(blockCounts),
			"maxTxsInOneBlock", maxPerBlock,
			"nonceRange", fmt.Sprintf("%d-%d", results[0].nonce, results[len(results)-1].nonce),
		)
		return nil
	})
	require.NoError(t, err, "single sender random nonce order with random scores verification failed")
}

func skipIfRulesNotEnabled(t devtest.T) {
	if !rulesEnabled() {
		t.Skip("Skipping rule ordering test")
	}
}
