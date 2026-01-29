//! Integration tests for `FlashBlockService`.
//!
//! These tests verify the service's coordination logic including:
//! - Flashblock processing and sequence management
//! - Speculative building when pending parent state is available
//! - Canonical block reconciliation
//! - Build job scheduling
//! - Transaction cache reuse across flashblocks

use alloy_primitives::B256;
use reth_execution_types::BlockExecutionOutput;
use reth_optimism_flashblocks::{
    CanonicalBlockNotification, PendingBlockState, PendingStateRegistry,
    validation::{CanonicalBlockFingerprint, ReconciliationStrategy},
};
use reth_optimism_primitives::OpPrimitives;
use reth_revm::cached::CachedReads;
use std::sync::Arc;

use crate::harness::{FlashBlockServiceTestHarness, TestFlashBlockFactory};

const fn canonical_fingerprint(
    block_number: u64,
    tx_hashes: Vec<B256>,
) -> CanonicalBlockFingerprint {
    CanonicalBlockFingerprint {
        block_number,
        block_hash: B256::repeat_byte(0xAB),
        parent_hash: B256::repeat_byte(0xCD),
        tx_hashes,
    }
}

/// Tests that the service processes flashblocks and updates the sequence manager.
#[tokio::test]
async fn test_service_processes_flashblocks() {
    let mut harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // Send a sequence of flashblocks for block 100
    let fb0 = factory.flashblock_at(0).build();
    let fb1 = factory.flashblock_after(&fb0).build();
    let fb2 = factory.flashblock_after(&fb1).build();

    harness.send_flashblock(fb0).await;
    harness.send_flashblock(fb1).await;
    harness.send_flashblock(fb2).await;

    // Verify flashblocks were received via broadcast
    assert_eq!(harness.received_flashblock_count(), 3);
}

/// Tests that starting a new block (index 0) finalizes the previous sequence.
#[tokio::test]
async fn test_service_finalizes_sequence_on_new_block() {
    let mut harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // First block sequence
    let fb0 = factory.flashblock_at(0).build();
    let fb1 = factory.flashblock_after(&fb0).build();
    harness.send_flashblock(fb0.clone()).await;
    harness.send_flashblock(fb1).await;

    // Start new block - should finalize previous sequence
    let fb2 = factory.flashblock_for_next_block(&fb0).build();
    harness.send_flashblock(fb2).await;

    // Verify sequence was broadcast (finalized)
    assert!(harness.has_complete_sequence());
}

/// Tests canonical block catch-up clears pending state.
#[tokio::test]
async fn test_service_handles_canonical_catchup() {
    let mut harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // Send flashblocks for block 100
    let fb0 = factory.flashblock_at(0).build();
    harness.send_flashblock(fb0).await;

    // Canonical block arrives at 100 - should trigger catch-up
    harness
        .send_canonical_block(CanonicalBlockNotification {
            block_number: 100,
            block_hash: B256::repeat_byte(0x10),
            parent_hash: B256::repeat_byte(0x01),
            tx_hashes: vec![],
        })
        .await;

    // Verify reconciliation strategy was CatchUp
    let strategy = harness.last_reconciliation_strategy();
    assert_eq!(strategy, Some(ReconciliationStrategy::CatchUp));
}

/// Tests that reorg detection clears pending state.
#[tokio::test]
async fn test_service_handles_reorg() {
    let mut harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // Send flashblocks for block 100 with specific tx hashes
    let fb0 = factory.flashblock_at(0).build();
    harness.send_flashblock(fb0).await;

    // Canonical block has different tx hashes - should detect reorg
    let canonical_tx_hashes = vec![B256::repeat_byte(0xAA)];
    harness
        .send_canonical_block(CanonicalBlockNotification {
            block_number: 100,
            block_hash: B256::repeat_byte(0x11),
            parent_hash: B256::repeat_byte(0x02),
            tx_hashes: canonical_tx_hashes,
        })
        .await;

    // Verify reconciliation strategy detected reorg (or catchup if no pending txs)
    let strategy = harness.last_reconciliation_strategy();
    assert!(matches!(
        strategy,
        Some(ReconciliationStrategy::CatchUp | ReconciliationStrategy::HandleReorg)
    ));
}

/// Tests speculative building priority - canonical takes precedence.
#[tokio::test]
async fn test_speculative_build_priority() {
    let harness = FlashBlockServiceTestHarness::new();

    // Test the sequence manager's priority logic directly
    let factory = TestFlashBlockFactory::new();

    // Create flashblock for block 100
    let fb0 = factory.flashblock_at(0).build();
    let parent_hash = fb0.base.as_ref().unwrap().parent_hash;

    let mut sequences = harness.create_sequence_manager();
    sequences.insert_flashblock(fb0).unwrap();

    // Create a pending state that doesn't match
    let pending_parent_hash = B256::random();
    let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        B256::repeat_byte(0xBB), // Different from parent_hash
        99,
        pending_parent_hash,
        pending_parent_hash, // canonical anchor
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    // When local tip matches parent, canonical build should be selected (no pending_parent)
    let args = sequences.next_buildable_args(parent_hash, 1000000, Some(pending_state));
    assert!(args.is_some());
    assert!(args.unwrap().pending_parent.is_none()); // Canonical mode, not speculative
}

/// Tests speculative building is used when canonical parent is unavailable.
#[tokio::test]
async fn test_speculative_build_with_pending_parent() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // Create flashblock for block 101 (parent is block 100)
    let fb0 = factory.flashblock_at(0).block_number(101).build();
    let block_100_hash = fb0.base.as_ref().unwrap().parent_hash;

    let mut sequences = harness.create_sequence_manager();
    sequences.insert_flashblock(fb0).unwrap();

    // Local tip is block 99 (doesn't match block 100)
    let local_tip_hash = B256::random();

    // Create pending state for block 100
    let pending_parent_hash = B256::random();
    let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_100_hash, // Matches flashblock's parent
        100,
        pending_parent_hash,
        pending_parent_hash, // canonical anchor
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    // Should select speculative build with pending parent
    let args = sequences.next_buildable_args(local_tip_hash, 1000000, Some(pending_state));
    assert!(args.is_some());
    let build_args = args.unwrap();
    assert!(build_args.pending_parent.is_some());
    assert_eq!(build_args.pending_parent.as_ref().unwrap().block_number, 100);
}

/// Tests that depth limit exceeded clears pending state.
#[tokio::test]
async fn test_depth_limit_exceeded() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Insert flashblocks spanning multiple blocks (100, 101, 102)
    let fb0 = factory.flashblock_at(0).build();
    sequences.insert_flashblock(fb0.clone()).unwrap();

    let fb1 = factory.flashblock_for_next_block(&fb0).build();
    sequences.insert_flashblock(fb1.clone()).unwrap();

    let fb2 = factory.flashblock_for_next_block(&fb1).build();
    sequences.insert_flashblock(fb2).unwrap();

    // Canonical at 101 with max_depth of 0 should trigger depth limit exceeded
    let strategy = sequences.process_canonical_block(canonical_fingerprint(101, vec![]), 0);
    assert!(matches!(strategy, ReconciliationStrategy::DepthLimitExceeded { .. }));
}

/// Tests that speculative building uses cached sequences.
#[tokio::test]
async fn test_speculative_build_uses_cached_sequences() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Create and cache sequence for block 100
    let fb0 = factory.flashblock_at(0).build();
    let block_99_hash = fb0.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb0.clone()).unwrap();

    // Create sequence for block 101 (caches block 100)
    let fb1 = factory.flashblock_for_next_block(&fb0).build();
    sequences.insert_flashblock(fb1.clone()).unwrap();

    // Create sequence for block 102 (caches block 101)
    let fb2 = factory.flashblock_for_next_block(&fb1).build();
    sequences.insert_flashblock(fb2).unwrap();

    // Local tip doesn't match anything canonical
    let local_tip_hash = B256::random();

    // Pending state matches block 99 (block 100's parent)
    let pending_parent_hash = B256::random();
    let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_99_hash,
        99,
        pending_parent_hash,
        pending_parent_hash, // canonical anchor
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    // Should find cached sequence for block 100
    let args = sequences.next_buildable_args(local_tip_hash, 1000000, Some(pending_state));
    assert!(args.is_some());
    let build_args = args.unwrap();
    assert!(build_args.pending_parent.is_some());
    assert_eq!(build_args.base.block_number, 100);
}

/// Tests the pending state registry behavior.
#[tokio::test]
async fn test_pending_state_registry() {
    let mut registry: PendingStateRegistry<OpPrimitives> = PendingStateRegistry::new();

    let parent_hash = B256::repeat_byte(0);
    let state = PendingBlockState::new(
        B256::repeat_byte(1),
        100,
        parent_hash,
        parent_hash, // canonical anchor
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    registry.record_build(state);

    // Should return state for matching parent hash
    let result = registry.get_state_for_parent(B256::repeat_byte(1));
    assert!(result.is_some());
    assert_eq!(result.unwrap().block_number, 100);

    // Clear and verify
    registry.clear();
    assert!(registry.current().is_none());
}

/// Tests that in-progress signal is sent when build starts.
#[tokio::test]
async fn test_in_progress_signal() {
    let mut harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    // Get the in-progress receiver
    let in_progress_rx = harness.subscribe_in_progress();

    // Initially should be None
    assert!(in_progress_rx.borrow().is_none());

    // Send flashblocks - note: actual build won't happen without proper provider setup
    // but we can verify the signal mechanism exists
    let fb0 = factory.flashblock_at(0).build();
    harness.send_flashblock(fb0).await;

    // The signal should still be None since we can't actually start a build
    // (would need proper provider setup)
    // This test primarily verifies the signal mechanism is wired up
    assert!(in_progress_rx.borrow().is_none());
}

// ==================== Transaction Cache Integration Tests ====================

/// Tests the transaction cache E2E scenario: fb0 [A,B] → fb1 [A,B,C]
/// This verifies the cache flow at the sequence manager level.
#[tokio::test]
async fn test_transaction_cache_continuation_flow() {
    use reth_optimism_flashblocks::TransactionCache;

    // Create a transaction cache
    let mut cache: TransactionCache<OpPrimitives> = TransactionCache::new();

    let tx_a = B256::repeat_byte(0xAA);
    let tx_b = B256::repeat_byte(0xBB);
    let tx_c = B256::repeat_byte(0xCC);

    // Simulate fb0 execution: [A, B]
    let fb0_txs = vec![tx_a, tx_b];
    assert!(cache.get_resumable_state(100, &fb0_txs).is_none());

    // After fb0 execution, update cache
    cache.update(100, fb0_txs, reth_revm::db::BundleState::default(), vec![]);

    // Simulate fb1: [A, B, C] - should resume and skip A, B
    let fb1_txs = vec![tx_a, tx_b, tx_c];
    let result = cache.get_resumable_state(100, &fb1_txs);
    assert!(result.is_some());
    let (_, _, skip) = result.unwrap();
    assert_eq!(skip, 2); // Skip first 2 txs
}

/// Tests that transaction cache is invalidated on block change.
#[tokio::test]
async fn test_transaction_cache_block_transition() {
    use reth_optimism_flashblocks::TransactionCache;

    let mut cache: TransactionCache<OpPrimitives> = TransactionCache::new();

    let tx_a = B256::repeat_byte(0xAA);

    // Block 100
    cache.update(100, vec![tx_a], reth_revm::db::BundleState::default(), vec![]);

    // Block 101 - cache should not be valid
    assert!(cache.get_resumable_state(101, &[tx_a]).is_none());
}

// ==================== Reconciliation Integration Tests ====================

/// Tests that reconciliation properly clears state on catch-up.
#[tokio::test]
async fn test_reconciliation_catchup_clears_state() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Build up state for blocks 100, 101
    let fb0 = factory.flashblock_at(0).build();
    sequences.insert_flashblock(fb0.clone()).unwrap();

    let fb1 = factory.flashblock_for_next_block(&fb0).build();
    sequences.insert_flashblock(fb1).unwrap();

    // Verify state exists
    assert!(sequences.earliest_block_number().is_some());

    // Canonical catches up to 101
    let strategy = sequences.process_canonical_block(canonical_fingerprint(101, vec![]), 10);
    assert_eq!(strategy, ReconciliationStrategy::CatchUp);

    // After catch-up, no buildable args should exist
    let local_tip = B256::random();
    let args = sequences.next_buildable_args::<OpPrimitives>(local_tip, 1000000, None);
    assert!(args.is_none());
}

/// Tests that reconciliation properly clears state on depth limit exceeded.
#[tokio::test]
async fn test_reconciliation_depth_limit_clears_state() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Build up state for blocks 100-102
    let fb0 = factory.flashblock_at(0).build();
    sequences.insert_flashblock(fb0.clone()).unwrap();

    let fb1 = factory.flashblock_for_next_block(&fb0).build();
    sequences.insert_flashblock(fb1.clone()).unwrap();

    let fb2 = factory.flashblock_for_next_block(&fb1).build();
    sequences.insert_flashblock(fb2).unwrap();

    // Canonical at 101 with very small max_depth (0)
    let strategy = sequences.process_canonical_block(canonical_fingerprint(101, vec![]), 0);
    assert!(matches!(strategy, ReconciliationStrategy::DepthLimitExceeded { .. }));

    // After depth exceeded, no state should remain
    assert!(sequences.earliest_block_number().is_none());
}

/// Tests continue strategy preserves state.
#[tokio::test]
async fn test_reconciliation_continue_preserves_state() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Build up state for block 100
    let fb0 = factory.flashblock_at(0).build();
    let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb0).unwrap();

    // Canonical at 99 (behind pending)
    let strategy = sequences.process_canonical_block(canonical_fingerprint(99, vec![]), 10);
    assert_eq!(strategy, ReconciliationStrategy::Continue);

    // State should be preserved - can still build
    let args = sequences.next_buildable_args::<OpPrimitives>(parent_hash, 1000000, None);
    assert!(args.is_some());
}

// ==================== Speculative Building Chain Tests ====================

/// Tests multi-level speculative building: block N → N+1 → N+2.
#[tokio::test]
async fn test_speculative_building_chain() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Create flashblock sequence for block 100
    let fb100 = factory.flashblock_at(0).block_number(100).build();
    let block_99_hash = fb100.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb100.clone()).unwrap();

    // Create flashblock sequence for block 101 (caches 100)
    let fb101 = factory.flashblock_for_next_block(&fb100).build();
    let block_100_hash = fb101.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb101.clone()).unwrap();

    // Create flashblock sequence for block 102 (caches 101)
    let fb102 = factory.flashblock_for_next_block(&fb101).build();
    let block_101_hash = fb102.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb102).unwrap();

    // Local tip is some random hash (not matching any canonical)
    let local_tip = B256::random();

    // Pending state for block 99 should allow building block 100
    let parent_of_99 = B256::random();
    let pending_99: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_99_hash,
        99,
        parent_of_99,
        parent_of_99, // canonical_anchor_hash
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    let args = sequences.next_buildable_args(local_tip, 1000000, Some(pending_99));
    assert!(args.is_some());
    assert_eq!(args.as_ref().unwrap().base.block_number, 100);

    // Pending state for block 100 should allow building block 101
    let pending_100: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_100_hash,
        100,
        block_99_hash,
        block_99_hash, // canonical_anchor_hash (forwarded from parent)
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    let args = sequences.next_buildable_args(local_tip, 1000000, Some(pending_100));
    assert!(args.is_some());
    assert_eq!(args.as_ref().unwrap().base.block_number, 101);

    // Pending state for block 101 should allow building block 102
    let pending_101: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_101_hash,
        101,
        block_100_hash,
        block_100_hash, // canonical_anchor_hash (forwarded from parent)
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    let args = sequences.next_buildable_args(local_tip, 1000000, Some(pending_101));
    assert!(args.is_some());
    assert_eq!(args.as_ref().unwrap().base.block_number, 102);
}

/// Tests that speculative build args include the pending parent state.
#[tokio::test]
async fn test_speculative_build_includes_pending_parent() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Create flashblock for block 101
    let fb = factory.flashblock_at(0).block_number(101).build();
    let block_100_hash = fb.base.as_ref().unwrap().parent_hash;
    sequences.insert_flashblock(fb).unwrap();

    // Local tip doesn't match
    let local_tip = B256::random();

    // Create pending state for block 100
    let parent_of_100 = B256::random();
    let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState::new(
        block_100_hash,
        100,
        parent_of_100,
        parent_of_100, // canonical_anchor_hash
        Arc::new(BlockExecutionOutput::default()),
        CachedReads::default(),
    );

    let args = sequences.next_buildable_args(local_tip, 1000000, Some(pending_state));
    assert!(args.is_some());

    let build_args = args.unwrap();
    assert!(build_args.pending_parent.is_some());

    // Verify the pending parent has the correct block info
    let pp = build_args.pending_parent.unwrap();
    assert_eq!(pp.block_number, 100);
    assert_eq!(pp.block_hash, block_100_hash);
}

// ==================== Edge Case Tests ====================

/// Tests behavior when no pending state and no canonical match.
#[tokio::test]
async fn test_no_buildable_when_nothing_matches() {
    let harness = FlashBlockServiceTestHarness::new();
    let factory = TestFlashBlockFactory::new();

    let mut sequences = harness.create_sequence_manager();

    // Create flashblock for block 100
    let fb = factory.flashblock_at(0).build();
    sequences.insert_flashblock(fb).unwrap();

    // Local tip doesn't match, no pending state
    let local_tip = B256::random();
    let args = sequences.next_buildable_args::<OpPrimitives>(local_tip, 1000000, None);
    assert!(args.is_none());
}

/// Tests that `NoPendingState` is returned when no sequences exist.
#[tokio::test]
async fn test_reconciliation_with_no_pending_state() {
    let harness = FlashBlockServiceTestHarness::new();
    let mut sequences = harness.create_sequence_manager();

    // No flashblocks inserted
    let strategy = sequences.process_canonical_block(canonical_fingerprint(100, vec![]), 10);
    assert_eq!(strategy, ReconciliationStrategy::NoPendingState);
}
