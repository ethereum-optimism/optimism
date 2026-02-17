//! Test demonstrating the safe head stall issue when block fetch fails.

use crate::{
    ConsolidateInput, ConsolidateTask, Engine, EngineState, EngineTaskError, EngineTaskErrorSeverity,
    EngineTaskExt, test_utils::MockEngineClient,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use tokio::sync::watch;

/// Demonstrates the stall: when block fetch fails repeatedly,
/// the ConsolidateTask stays in the queue and retries infinitely.
///
/// This test documents the original bug behavior - it will pass with the fix
/// but shows what would happen without it (commented assertion).
#[tokio::test]
async fn test_consolidate_stall_on_missing_block() {
    // Setup: Create engine with a mock client that returns errors for block fetches
    // This triggers FailedToFetchUnsafeL2Block (Temporary error)
    let mock_client = Arc::new(
        MockEngineClient::builder()
            .with_config(Arc::new(RollupConfig::default()))
            .with_l2_block_by_label_error()
            .build(),
    );
    let rollup_config = Arc::new(RollupConfig::default());
    let (state_tx, _) = watch::channel(EngineState::default());
    let (queue_len_tx, _) = watch::channel(0);
    let mut engine = Engine::new(EngineState::default(), state_tx, queue_len_tx);

    // Create a safe head block info that should be consolidated
    let safe_head = L2BlockInfo {
        block_info: kona_protocol::BlockInfo {
            hash: B256::random(),
            number: 100,
            parent_hash: B256::random(),
            timestamp: 1000,
        },
        l1_origin: BlockNumHash { number: 50, hash: B256::random() },
        seq_num: 0,
    };

    // Enqueue a ConsolidateTask for this safe head
    let task = crate::EngineTask::Consolidate(Box::new(ConsolidateTask::new(
        mock_client.clone(),
        rollup_config.clone(),
        ConsolidateInput::BlockInfo(safe_head),
    )));
    engine.enqueue(task);

    // Attempt to drain 10 times - with the fix, task should be removed after 10 retries
    let mut retry_count = 0;
    const MAX_TEST_RETRIES: usize = 10;

    while retry_count < MAX_TEST_RETRIES {
        let result = engine.drain().await;

        // Should fail with FailedToFetchUnsafeL2Block error
        assert!(result.is_err(), "Drain should fail when block fetch fails");

        retry_count += 1;
    }

    // With the fix: task should be removed after MAX_TASK_RETRIES
    // Without fix: task would still be in queue (infinite stall)
    let queue_length = engine.queue_length_subscribe().borrow().clone();

    // This assertion passes with the fix:
    assert_eq!(
        queue_length, 0,
        "Task should be removed from queue after {} retries (fix applied)",
        MAX_TEST_RETRIES
    );
}

/// Verifies the retry limit behavior: after MAX_TASK_RETRIES attempts,
/// a task with temporary errors should be dropped and return MaxRetriesExceeded.
#[tokio::test]
async fn test_consolidate_with_retry_limit() {
    // Setup: Create engine with a mock client that returns errors for block fetches
    let mock_client = Arc::new(
        MockEngineClient::builder()
            .with_config(Arc::new(RollupConfig::default()))
            .with_l2_block_by_label_error()
            .build(),
    );
    let rollup_config = Arc::new(RollupConfig::default());
    let (state_tx, _) = watch::channel(EngineState::default());
    let (queue_len_tx, _) = watch::channel(0);
    let mut engine = Engine::new(EngineState::default(), state_tx, queue_len_tx);

    // Create a safe head block info that should be consolidated
    let safe_head = L2BlockInfo {
        block_info: kona_protocol::BlockInfo {
            hash: B256::random(),
            number: 100,
            parent_hash: B256::random(),
            timestamp: 1000,
        },
        l1_origin: BlockNumHash { number: 50, hash: B256::random() },
        seq_num: 0,
    };

    // Enqueue a ConsolidateTask for this safe head
    let task = crate::EngineTask::Consolidate(Box::new(ConsolidateTask::new(
        mock_client.clone(),
        rollup_config.clone(),
        ConsolidateInput::BlockInfo(safe_head),
    )));
    engine.enqueue(task);

    // First 9 attempts: should return temporary error and keep task in queue
    for i in 0..9 {
        let result = engine.drain().await;
        assert!(result.is_err(), "Drain {} should fail", i + 1);

        // Verify task is still in queue
        let queue_length = engine.queue_length_subscribe().borrow().clone();
        assert_eq!(
            queue_length, 1,
            "Task should remain in queue after retry {}",
            i + 1
        );
    }

    // 10th attempt: should drop task and return MaxRetriesExceeded
    let result = engine.drain().await;
    assert!(result.is_err(), "10th drain should fail with MaxRetriesExceeded");

    let error = result.unwrap_err();

    // Verify error is MaxRetriesExceeded wrapped in ConsolidateTaskError
    let error_string = error.to_string();
    assert!(
        error_string.contains("max retries") || error_string.contains("MaxRetriesExceeded"),
        "Error should mention max retries: {}",
        error_string
    );

    // Verify task was removed from queue
    let queue_length = engine.queue_length_subscribe().borrow().clone();
    assert_eq!(
        queue_length, 0,
        "Task should be removed from queue after max retries"
    );

    // Verify the error has Reset severity (triggers engine reset)
    match error {
        crate::task_queue::EngineTaskErrors::Consolidate(e) => {
            assert_eq!(
                e.severity(),
                EngineTaskErrorSeverity::Reset,
                "MaxRetriesExceeded should have Reset severity"
            );
        }
        _ => panic!("Expected ConsolidateTaskError, got: {:?}", error),
    }
}
