//go:build !ci

package flashblocks

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// skipIfRulesNotEnabled skips the test if FLASHBLOCKS_RULES_TEST env var is not set.
// Rule ordering tests require the rules-enabled orchestrator setup.
func skipIfRulesNotEnabled(t devtest.T) {
	if os.Getenv("FLASHBLOCKS_RULES_TEST") == "" {
		t.Skip("Skipping rule ordering test: FLASHBLOCKS_RULES_TEST env var not set")
	}
}

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

	// Drive initial blocks to ensure the system is ready
	driveViaTestSequencer(t, sys, 3)

	// Create funded senders - one for each priority level
	fundAmount := eth.Ether(1)
	senderHigh := sys.FunderL2.NewFundedEOA(fundAmount)
	senderMedium := sys.FunderL2.NewFundedEOA(fundAmount)
	senderLow := sys.FunderL2.NewFundedEOA(fundAmount)
	senderNormal := sys.FunderL2.NewFundedEOA(fundAmount)

	// Create a normal (non-boosted) recipient for comparison
	normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
	normalRecipientAddr := normalRecipient.Address()

	logger.Info("Test accounts created",
		"sender_high", senderHigh.Address().Hex(),
		"sender_medium", senderMedium.Address().Hex(),
		"sender_low", senderLow.Address().Hex(),
		"sender_normal", senderNormal.Address().Hex(),
		"high_priority_recipient", HighPriorityRecipient.Hex(),
		"medium_priority_recipient", MediumPriorityRecipient.Hex(),
		"low_priority_recipient", LowPriorityRecipient.Hex(),
		"normal_recipient", normalRecipientAddr.Hex(),
	)

	// Send all transactions concurrently with the same value (to ensure fair comparison)
	sendAmount := eth.OneHundredthEther
	var wg sync.WaitGroup
	var txHigh, txMedium, txLow, txNormal *txplan.PlannedTx

	wg.Add(4)

	// Transaction to high priority recipient (weight 5000)
	go func() {
		defer wg.Done()
		txHigh = senderHigh.Transact(
			senderHigh.Plan(),
			txplan.WithTo(&HighPriorityRecipient),
			txplan.WithValue(sendAmount),
		)
	}()

	// Transaction to medium priority recipient (weight 2000)
	go func() {
		defer wg.Done()
		txMedium = senderMedium.Transact(
			senderMedium.Plan(),
			txplan.WithTo(&MediumPriorityRecipient),
			txplan.WithValue(sendAmount),
		)
	}()

	// Transaction to low priority recipient (weight 500)
	go func() {
		defer wg.Done()
		txLow = senderLow.Transact(
			senderLow.Plan(),
			txplan.WithTo(&LowPriorityRecipient),
			txplan.WithValue(sendAmount),
		)
	}()

	// Transaction to normal recipient (no boost, weight 0)
	go func() {
		defer wg.Done()
		txNormal = senderNormal.Transact(
			senderNormal.Plan(),
			txplan.WithTo(&normalRecipientAddr),
			txplan.WithValue(sendAmount),
		)
	}()

	wg.Wait()

	// Wait for all transactions to be included
	receiptHigh, err := txHigh.Included.Eval(ctx)
	require.NoError(t, err, "high priority tx should be included")
	receiptMedium, err := txMedium.Included.Eval(ctx)
	require.NoError(t, err, "medium priority tx should be included")
	receiptLow, err := txLow.Included.Eval(ctx)
	require.NoError(t, err, "low priority tx should be included")
	receiptNormal, err := txNormal.Included.Eval(ctx)
	require.NoError(t, err, "normal tx should be included")

	logger.Info("All transactions confirmed",
		"high_block", receiptHigh.BlockNumber, "high_index", receiptHigh.TransactionIndex,
		"medium_block", receiptMedium.BlockNumber, "medium_index", receiptMedium.TransactionIndex,
		"low_block", receiptLow.BlockNumber, "low_index", receiptLow.TransactionIndex,
		"normal_block", receiptNormal.BlockNumber, "normal_index", receiptNormal.TransactionIndex,
	)

	// Verify ordering based on boost weights using transaction index in the block
	// Lower TransactionIndex = earlier in the block = higher priority
	sameBlock := receiptHigh.BlockNumber.Cmp(receiptMedium.BlockNumber) == 0 &&
		receiptMedium.BlockNumber.Cmp(receiptLow.BlockNumber) == 0 &&
		receiptLow.BlockNumber.Cmp(receiptNormal.BlockNumber) == 0

	if sameBlock {
		logger.Info("All transactions in same block - verifying boost ordering via transaction index")

		// Verify ordering: high < medium < low < normal (lower index = earlier = higher priority)
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
	} else {
		// Transactions in different blocks - compare transaction indices within blocks where applicable
		logger.Info("Transactions in different blocks - comparing indices for transactions in same block",
			"high_block", receiptHigh.BlockNumber,
			"medium_block", receiptMedium.BlockNumber,
			"low_block", receiptLow.BlockNumber,
			"normal_block", receiptNormal.BlockNumber,
		)

		// If in same block, higher priority should have lower index
		if receiptHigh.BlockNumber.Cmp(receiptMedium.BlockNumber) == 0 {
			require.Less(t, receiptHigh.TransactionIndex, receiptMedium.TransactionIndex,
				"high priority should have lower tx index than medium in same block")
		}
		if receiptMedium.BlockNumber.Cmp(receiptLow.BlockNumber) == 0 {
			require.Less(t, receiptMedium.TransactionIndex, receiptLow.TransactionIndex,
				"medium priority should have lower tx index than low in same block")
		}
		if receiptLow.BlockNumber.Cmp(receiptNormal.BlockNumber) == 0 {
			require.Less(t, receiptLow.TransactionIndex, receiptNormal.TransactionIndex,
				"low priority should have lower tx index than normal in same block")
		}
	}
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

	ctx, cancel := context.WithTimeout(topLevelCtx, 60*time.Second)
	defer cancel()

	// Drive initial blocks
	driveViaTestSequencer(t, sys, 2)

	// Create two funded senders
	fundAmount := eth.ThreeHundredthsEther
	senderBoosted := sys.FunderL2.NewFundedEOA(fundAmount)
	senderNormal := sys.FunderL2.NewFundedEOA(fundAmount)

	// Create a normal recipient
	normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
	normalRecipientAddr := normalRecipient.Address()

	// Define gas fee parameters:
	// - Low priority fee: 1 gwei (for boosted tx)
	// - High priority fee: 100 gwei (for non-boosted tx) - 100x higher!
	lowGasTip := big.NewInt(1_000_000_000)    // 1 gwei
	highGasTip := big.NewInt(100_000_000_000) // 100 gwei
	// Set fee cap high enough to cover base fee + tip
	highGasFeeCap := big.NewInt(200_000_000_000) // 200 gwei

	logger.Info("Test accounts created",
		"sender_boosted", senderBoosted.Address().Hex(),
		"sender_normal", senderNormal.Address().Hex(),
		"boosted_recipient", BoostedRecipient.Hex(),
		"normal_recipient", normalRecipientAddr.Hex(),
		"boosted_tx_gas_tip", lowGasTip.String(),
		"normal_tx_gas_tip", highGasTip.String(),
	)

	// Send transactions concurrently
	sendAmount := eth.OneHundredthEther
	var wg sync.WaitGroup
	var txBoosted, txNormal *txplan.PlannedTx

	wg.Add(2)

	// Transaction to boosted recipient (weight 1000) with LOW priority fee
	go func() {
		defer wg.Done()
		txBoosted = senderBoosted.Transact(
			senderBoosted.Plan(),
			txplan.WithTo(&BoostedRecipient),
			txplan.WithValue(sendAmount),
			txplan.WithGasTipCap(lowGasTip),
		)
	}()

	// Transaction to normal recipient (no boost) with HIGH priority fee
	// This tx has 100x higher gas tip but should still come AFTER the boosted tx
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

	// Wait for transactions to be included
	receiptBoosted, err := txBoosted.Included.Eval(ctx)
	require.NoError(t, err, "boosted tx should be included")
	receiptNormal, err := txNormal.Included.Eval(ctx)
	require.NoError(t, err, "normal tx should be included")

	logger.Info("Transactions confirmed",
		"boosted_hash", receiptBoosted.TxHash.Hex(),
		"boosted_block", receiptBoosted.BlockNumber,
		"boosted_index", receiptBoosted.TransactionIndex,
		"boosted_gas_tip", "1 gwei",
		"normal_hash", receiptNormal.TxHash.Hex(),
		"normal_block", receiptNormal.BlockNumber,
		"normal_index", receiptNormal.TransactionIndex,
		"normal_gas_tip", "100 gwei",
	)

	// If both transactions are in the same block, boosted should have lower index (come first)
	// DESPITE the normal tx having 100x higher gas tip
	if receiptBoosted.BlockNumber.Cmp(receiptNormal.BlockNumber) == 0 {
		require.Less(t, receiptBoosted.TransactionIndex, receiptNormal.TransactionIndex,
			"boosted transaction (weight 1000, 1 gwei tip) should have lower tx index than "+
				"normal transaction (no boost, 100 gwei tip) - proving rules > gas priority")

		logger.Info("Rule-based boost precedence over gas priority verified!",
			"boosted_index", receiptBoosted.TransactionIndex,
			"normal_index", receiptNormal.TransactionIndex,
			"boosted_gas_tip", "1 gwei",
			"normal_gas_tip", "100 gwei",
			"conclusion", "rules-based ordering takes precedence over economic incentives",
		)
	} else {
		logger.Info("Transactions in different blocks - ordering verification skipped",
			"boosted_block", receiptBoosted.BlockNumber,
			"normal_block", receiptNormal.BlockNumber,
		)
	}
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
	driveViaTestSequencer(t, sys, 2)

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
		require.LessOrEqual(t, receipt0.BlockNumber.Uint64(), receipt1.BlockNumber.Uint64(),
			"tx0 must be in same or earlier block than tx1")
	}

	if receipt1.BlockNumber.Cmp(receipt2.BlockNumber) == 0 {
		require.Less(t, receipt1.TransactionIndex, receipt2.TransactionIndex,
			"tx1 (nonce N+1) must have lower index than tx2 (nonce N+2)")
	} else {
		require.LessOrEqual(t, receipt1.BlockNumber.Uint64(), receipt2.BlockNumber.Uint64(),
			"tx1 must be in same or earlier block than tx2")
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

	ctx, cancel := context.WithTimeout(topLevelCtx, 90*time.Second)
	defer cancel()

	// Drive initial blocks (use 2 to avoid timestamp drift exceeding 5s gossip threshold)
	driveViaTestSequencer(t, sys, 2)

	fundAmount := eth.ThreeHundredthsEther

	// Create normal recipient for non-boosted transactions
	normalRecipient := sys.Wallet.NewEOA(sys.L2EL)
	normalRecipientAddr := normalRecipient.Address()

	// Create senders with their target recipients and weights
	type senderConfig struct {
		eoa       *dsl.EOA
		priority  string
		recipient common.Address
		weight    int
	}

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
		eoa := sys.FunderL2.NewFundedEOA(fundAmount)
		senders[i] = senderConfig{
			eoa:       eoa,
			priority:  cfg.priority,
			recipient: cfg.recipient,
			weight:    cfg.weight,
		}
		logger.Debug("Created sender",
			"index", i,
			"address", eoa.Address().Hex(),
			"priority", cfg.priority,
			"recipient", cfg.recipient.Hex(),
		)
	}

	// Send all transactions concurrently
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

	// Wait for all to be included and collect results
	type txResult struct {
		receipt  *types.Receipt
		priority string
		weight   int
	}
	results := make([]txResult, len(senders))

	for i := range senders {
		receipt, err := plannedTxs[i].Included.Eval(ctx)
		require.NoError(t, err, fmt.Sprintf("tx%d should be included", i))
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

	// Group by block and verify ordering within each block
	blockGroups := make(map[uint64][]txResult)
	for _, r := range results {
		blockNum := r.receipt.BlockNumber.Uint64()
		blockGroups[blockNum] = append(blockGroups[blockNum], r)
	}

	for blockNum, txs := range blockGroups {
		if len(txs) < 2 {
			continue
		}

		logger.Info("Verifying ordering in block", "block", blockNum, "tx_count", len(txs))

		// Within the same block, higher weight should mean lower tx index
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

	logger.Info("Multiple senders mixed priorities test completed")
}
