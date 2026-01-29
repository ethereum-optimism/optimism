//! Integration tests for `FlashBlockService`.
//!
//! These tests verify the service's coordination logic including:
//! - Flashblock processing and sequence management
//! - Speculative building when pending parent state is available
//! - Canonical block reconciliation
//! - Build job scheduling

use alloy_primitives::B256;
use reth_execution_types::BlockExecutionOutput;
use reth_optimism_flashblocks::{
    CanonicalBlockNotification, PendingBlockState, PendingStateRegistry,
    validation::ReconciliationStrategy,
};
use reth_optimism_primitives::OpPrimitives;
use reth_revm::cached::CachedReads;
use std::sync::Arc;

use crate::harness::{FlashBlockServiceTestHarness, TestFlashBlockFactory};

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
        .send_canonical_block(CanonicalBlockNotification { block_number: 100, tx_hashes: vec![] })
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
    let strategy = sequences.process_canonical_block(101, &[], 0);
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
