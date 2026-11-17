use super::*;
use crate::QueuedEngineDerivationClient;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    ConsolidateInput, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    test_utils::{TestEngineStateBuilder, l1_info_deposit, test_block_info, test_engine_client},
};
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Exercise the real actor request/drain loop with forty consecutive unsafe blocks, without
/// depending on the removed dev RPC subscriptions. Observe EVERY enqueue, including a burst
/// already waiting in the inbound channel: healthy processing must keep the task queue <= 1.
#[tokio::test(start_paused = true)]
async fn healthy_unsafe_imports_keep_engine_queue_bounded() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let initial_head = test_block_info(0);
    let state = TestEngineStateBuilder::new().with_unsafe_head(initial_head).build();
    let (state_tx, state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(40);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let mut expected_heads = vec![initial_head];
    for number in 1..=40 {
        let mut block = OpBlock::default();
        block.header.number = number;
        block.header.timestamp = number * 2;
        block.header.parent_hash = expected_heads.last().unwrap().hash();
        block.body.transactions.push(l1_info_deposit());
        let payload = ExecutionPayloadV1::from_block_slow(&block);
        let imported: OpBlock =
            OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
        expected_heads.push(
            L2BlockInfo::from_block_and_genesis(&imported, &RollupConfig::default().genesis)
                .unwrap(),
        );
        l2.expect_params(
            "engine_newPayloadV2",
            serde_json::json!([payload]),
            PayloadStatus::from_status(PayloadStatusEnum::Valid),
        );
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([
                alloy_rpc_types_engine::ForkchoiceState {
                    head_block_hash: expected_heads.last().unwrap().hash(),
                    safe_block_hash: initial_head.hash(),
                    finalized_block_hash: initial_head.hash(),
                },
                null
            ]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        requests
            .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(
                OpExecutionPayloadEnvelope::V1(payload),
            )))
            .await
            .unwrap();
    }
    for expected in &expected_heads[..40] {
        time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
        assert_eq!(actor.engine.len(), 1, "no backlog may accumulate while the EL is healthy");
        assert_eq!(state_rx.borrow().sync_state.unsafe_head(), *expected);
    }
    // The final step imports block forty before discovering that its input has closed.
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));
    assert!(actor.engine.is_empty());
    assert_eq!(state_rx.borrow().sync_state.unsafe_head(), expected_heads[40]);
    l1.assert_finished();
    l2.assert_finished();
}

/// While the execution layer catches up to a delegated safe head ahead of the local unsafe head,
/// each delegate poll repeats the same target. Every poll sends it, after a `SYNCING` answer and
/// after a `VALID` one, since an execution layer that restarts can lose an applied forkchoice.
#[tokio::test(start_paused = true)]
async fn delegated_safe_head_is_sent_on_every_poll() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(10))
        .with_finalized_head(test_block_info(5))
        .build();
    let delegated_safe_head = test_block_info(30);
    let target = state
        .sync_state
        .apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(delegated_safe_head),
            safe_head: Some(delegated_safe_head),
            unsafe_head: Some(delegated_safe_head),
            ..Default::default()
        })
        .create_forkchoice_state();
    for status in [PayloadStatusEnum::Syncing, PayloadStatusEnum::Valid, PayloadStatusEnum::Valid] {
        l2.expect_forkchoice_updated(target, status);
    }
    let (state_tx, state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );

    // Three polls report the same target. Each step drains the previous poll's task and then
    // accepts the next poll.
    for _ in 0..3 {
        requests
            .send(EngineActorRequest::ProcessSafeL2Signal(ConsolidateInput::BlockInfo(
                delegated_safe_head,
            )))
            .await
            .unwrap();
        time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    }
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));

    assert!(actor.engine.is_empty());
    assert_eq!(state_rx.borrow().sync_state.safe_head(), delegated_safe_head);
    l1.assert_finished();
    l2.assert_finished();
}

/// A finalization the engine skips leaves the actor running, an identical repeat keeps the same
/// skip in memory, and a reset forgets it.
#[tokio::test(start_paused = true)]
async fn skipped_finalization_keeps_the_actor_running_until_a_reset_forgets_it() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(test_block_info(10))
        .build();
    let (state_tx, _state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let stale_finalization =
        || EngineActorRequest::ProcessFinalizedL2Block(Box::new(FinalizeBlockId::ByNumber(5)));

    // Each step drains the previous request's task, then accepts the next request.
    requests.send(stale_finalization()).await.unwrap();
    time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    requests.send(stale_finalization()).await.unwrap();
    time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    let first_skip = actor.last_skipped_task_error.clone();
    assert!(first_skip.is_some(), "the skipped finalization is remembered");

    requests.send(stale_finalization()).await.unwrap();
    time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    assert_eq!(actor.last_skipped_task_error, first_skip);

    actor.begin_reset(ResetKind::Full);
    assert_eq!(actor.last_skipped_task_error, None);
    l1.assert_finished();
    l2.assert_finished();
}
