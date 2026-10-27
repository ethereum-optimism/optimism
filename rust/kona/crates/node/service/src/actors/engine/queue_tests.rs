use super::*;
use crate::QueuedEngineDerivationClient;
use alloy_primitives::Sealed;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    NoopBlockSink,
    test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
};
use kona_protocol::L1BlockInfoBedrock;
use op_alloy_consensus::{OpBlock, OpTxEnvelope, TxDeposit};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Exercise the real actor request/drain loop with forty consecutive unsafe blocks, without
/// depending on the removed dev RPC subscriptions. Observe EVERY enqueue, including a burst
/// already waiting in the inbound channel: healthy processing must keep the task queue <= 1.
#[rstest::rstest]
#[case(false)]
#[case(true)]
#[tokio::test(start_paused = true)]
async fn healthy_unsafe_imports_keep_engine_queue_bounded(#[case] pending: bool) {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let initial_head = test_block_info(0);
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(initial_head).build();
    if pending {
        state.forkchoice_status =
            kona_engine::ForkchoiceStatus::Syncing(state.sync_state.create_forkchoice_state());
    }
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
    actor.forkchoice_retry_at = Instant::now() + Duration::from_secs(1);
    let mut expected_heads = vec![initial_head];
    for number in 1..=40 {
        let mut block = OpBlock::default();
        block.header.number = number;
        block.header.timestamp = number * 2;
        block.header.parent_hash = expected_heads.last().unwrap().hash();
        block.body.transactions.push(OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: L1BlockInfoBedrock::default().encode_calldata(),
            ..Default::default()
        })));
        let payload = ExecutionPayloadV1::from_block_slow(&block);
        let imported: OpBlock =
            OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
        expected_heads.push(
            L2BlockInfo::from_block_and_genesis(&imported, &RollupConfig::default().genesis)
                .unwrap(),
        );
        l2.expect_params(
            "engine_newPayloadV1",
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

/// Retry a pending forkchoice while the request channel remains open and idle. The retry
/// completes initial EL synchronization without requiring another unsafe block or L1 update.
#[tokio::test(start_paused = true)]
async fn pending_forkchoice_recovers_on_idle_actor_and_notifies_derivation() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(10))
        .with_finalized_head(test_block_info(5))
        .with_el_sync_finished(false)
        .build();
    state.forkchoice_status =
        kona_engine::ForkchoiceStatus::Syncing(state.sync_state.create_forkchoice_state());
    let target = state.sync_state.create_forkchoice_state();
    for status in [PayloadStatusEnum::Syncing, PayloadStatusEnum::Valid] {
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([target, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(status)),
        );
    }
    let (state_tx, state_rx) = watch::channel(state);
    let (_requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let start = Instant::now();
    // First attempt stays SYNCING. Its timer yields back to the actor after one second.
    actor.step().await.unwrap();
    assert_eq!(Instant::now() - start, Duration::from_secs(1));
    assert!(actor.engine.has_pending_forkchoice());
    assert!(!actor.el_sync_complete);
    // A successful retry leaves the actor waiting for work, but emits completion first.
    let mut tasks = tokio::task::JoinSet::new();
    tasks.spawn(async move {
        actor.step().await?;
        Ok::<(), EngineError>(())
    });
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        crate::DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(_)
    ));
    assert!(matches!(
        time::timeout(Duration::from_secs(2), derivation_rx.recv()).await.unwrap().unwrap(),
        crate::DerivationActorRequest::ProcessEngineSyncCompletionRequest(_)
    ));
    assert!(state_rx.borrow().el_sync_finished);
    assert_eq!(state_rx.borrow().forkchoice_status, kona_engine::ForkchoiceStatus::Applied(target));
    assert!(derivation_rx.try_recv().is_err(), "unchanged safe labels must not be re-notified");
    tasks.abort_all();
    l1.assert_finished();
    l2.assert_finished();
}

/// A ready inbound stream cannot suppress the independent pending-FCU timer. Its retry uses
/// the current target after processing queued work, rather than restoring an old unsafe head.
#[tokio::test(start_paused = true)]
async fn pending_forkchoice_timer_progresses_with_ready_requests() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(10))
        .with_finalized_head(test_block_info(5))
        .build();
    state.forkchoice_status =
        kona_engine::ForkchoiceStatus::Syncing(state.sync_state.create_forkchoice_state());
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
    actor.forkchoice_retry_at = Instant::now() + Duration::from_secs(1);
    let safe = test_block_info(30);
    for target in [safe, test_block_info(40)] {
        requests.send(EngineActorRequest::ProcessSafeL2Signal(target.into())).await.unwrap();
    }
    actor.step().await.unwrap(); // enqueue safe 30 without waiting for the timer
    let target = state
        .sync_state
        .apply_update(kona_engine::EngineSyncStateUpdate {
            unsafe_head: Some(safe),
            safe_head: Some(safe),
            local_safe_head: Some(safe),
            ..Default::default()
        })
        .create_forkchoice_state();
    for status in [PayloadStatusEnum::Syncing, PayloadStatusEnum::Valid] {
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([target, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(status)),
        );
    }
    time::advance(Duration::from_secs(1)).await;
    actor.step().await.unwrap(); // process safe 30, retry it, then enqueue ready safe 40
    assert!(!actor.engine.has_pending_forkchoice());
    assert_eq!(actor.engine.state().sync_state.unsafe_head(), safe);
    assert_eq!(actor.engine.len(), 1);
    l1.assert_finished();
    l2.assert_finished();
}
