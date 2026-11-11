use super::*;
use crate::QueuedEngineDerivationClient;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceState, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink, ResetRequest,
    test_utils::{
        RpcMock, TestEngineStateBuilder, TestRpcBlock, delegated_safe_heads, expect_sync_start,
        test_block_info, test_engine_client, test_l2_block, test_rpc_block,
    },
};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
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
        let block = test_l2_block(number, expected_heads.last().unwrap().hash(), 0);
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
        l2.expect_forkchoice_updated(
            forkchoice(*expected_heads.last().unwrap(), initial_head, initial_head),
            PayloadStatusEnum::Valid,
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
            .send(EngineActorRequest::ProcessSafeL2Signal(delegated_safe_heads(
                delegated_safe_head,
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

/// A delegate poll while the execution layer syncs toward the unsafe head: the delegate's safe
/// head is below the unsafe head but not in the execution layer yet, and neither is its finalized
/// block. The consolidate moves the safe labels and the finalize is skipped, so the actor goes on.
#[tokio::test(start_paused = true)]
async fn delegated_poll_during_el_sync_moves_safe_labels_and_skips_finalization() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_local_safe_head(test_block_info(5))
        .with_safe_head(test_block_info(5))
        .with_finalized_head(test_block_info(3))
        .build();
    let delegated_safe_head = test_block_info(10);
    let delegated_finalized_head = test_block_info(8);
    let expected_sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(delegated_safe_head),
        safe_head: Some(delegated_safe_head),
        ..Default::default()
    });
    l2.expect_block_by_number(delegated_safe_head.block_info.number, None);
    l2.expect_forkchoice_updated(
        expected_sync_state.create_forkchoice_state(),
        PayloadStatusEnum::Syncing,
    );
    l2.expect_block_by_number(delegated_finalized_head.block_info.number, None);
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

    // One poll: the safe head, then the finalized block pinned by hash. Each step drains the
    // previous request's task and then accepts the next request.
    requests
        .send(EngineActorRequest::ProcessSafeL2Signal(delegated_safe_heads(
            delegated_safe_head,
            delegated_safe_head,
        )))
        .await
        .unwrap();
    requests
        .send(EngineActorRequest::ProcessFinalizedL2Block(Box::new(FinalizeBlockId::ByHash(
            delegated_finalized_head.block_info.id(),
        ))))
        .await
        .unwrap();
    for _ in 0..2 {
        time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    }
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));

    assert!(actor.engine.is_empty());
    assert!(!actor.reset_pending);
    assert_eq!(state_rx.borrow().sync_state, expected_sync_state);
    l1.assert_finished();
    l2.assert_finished();
}

fn forkchoice(head: L2BlockInfo, safe: L2BlockInfo, finalized: L2BlockInfo) -> ForkchoiceState {
    ForkchoiceState {
        finalized_block_hash: finalized.hash(),
        head_block_hash: head.hash(),
        safe_block_hash: safe.hash(),
    }
}

/// The reset finds the EL's finalized, safe and unsafe heads all at genesis, then synchronizes
/// the EL to them.
fn expect_reset_to_genesis(
    l1: &RpcMock,
    l2: &RpcMock,
    genesis_block: &TestRpcBlock,
    genesis: L2BlockInfo,
) {
    expect_sync_start(l1, l2, genesis_block, forkchoice(genesis, genesis, genesis));
}

fn reset_request() -> (EngineActorRequest, mpsc::Receiver<kona_engine::EngineRequestResult<()>>) {
    let (result_tx, result_rx) = mpsc::channel(1);
    (EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })), result_rx)
}

/// How the finalization target is identified: delegated derivation pins the hash, and
/// finalization from L1 finality uses the block number.
#[derive(Clone, Copy, Debug)]
enum TargetKind {
    ByHash,
    ByNumber,
}

impl TargetKind {
    fn finalize_request(self, target: L2BlockInfo) -> EngineActorRequest {
        let block_id = match self {
            Self::ByHash => FinalizeBlockId::ByHash(target.block_info.id()),
            Self::ByNumber => FinalizeBlockId::ByNumber(target.block_info.number),
        };
        EngineActorRequest::ProcessFinalizedL2Block(Box::new(block_id))
    }
}

/// A reset can restore a safe head below a finalization target that was safe when it was
/// requested. The target must wait across repeated resets without stopping the node, holding the
/// queue, or triggering another reset, and must finalize once the safe head reaches it.
#[rstest::rstest]
#[case::by_hash(TargetKind::ByHash)]
#[case::by_number(TargetKind::ByNumber)]
#[tokio::test(start_paused = true)]
async fn finalization_target_survives_resets_below_it(#[case] target_kind: TargetKind) {
    let genesis_block = TestRpcBlock::default();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = genesis_block.header.inner.hash_slow();
    let config = Arc::new(config);
    let genesis = L2BlockInfo::from_block_and_genesis(
        &genesis_block.clone().into_consensus(),
        &config.genesis,
    )
    .unwrap();
    let target_block = test_l2_block(15, B256::with_last_byte(14), 0);
    let target = L2BlockInfo::from_block_and_genesis(&target_block, &config.genesis).unwrap();
    let restored_safe = test_block_info(20);
    let unsafe_payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &test_l2_block(1, genesis.hash(), 0),
    ));
    let unsafe_head = L2BlockInfo::from_block_and_genesis(
        &unsafe_payload.clone().try_into_block::<OpTxEnvelope>().unwrap(),
        &config.genesis,
    )
    .unwrap();

    let (client, l1, l2) = test_engine_client(config.clone());
    let (requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, _derivation_rx) = mpsc::channel(32);
    let (state_tx, state_rx) = watch::channel(EngineState::default());
    let mut actor = EngineActor::new(
        Arc::new(client),
        config.clone(),
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(EngineState::default(), state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );

    // Startup reset, with the finalization target arriving right behind it.
    let (reset, mut first_reset_result) = reset_request();
    requests.send(reset).await.unwrap();
    actor.step().await.unwrap();
    expect_reset_to_genesis(&l1, &l2, &genesis_block, genesis);
    requests.send(target_kind.finalize_request(target)).await.unwrap();
    actor.step().await.unwrap();
    assert!(first_reset_result.try_recv().unwrap().is_ok());
    assert_eq!(state_rx.borrow().sync_state.safe_head(), genesis);

    // The target is above the reset safe head: skipped without a lookup or forkchoice update,
    // and dropped from the queue rather than retried, so the unsafe import received in the same
    // step is the only task left.
    requests
        .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(unsafe_payload)))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert_eq!(actor.engine.len(), 1, "only the unsafe import may remain queued");
    assert!(!actor.reset_pending);
    l2.expect("engine_newPayloadV1", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect_forkchoice_updated(
        forkchoice(unsafe_head, genesis, genesis),
        PayloadStatusEnum::Valid,
    );
    let (reset, mut second_reset_result) = reset_request();
    requests.send(reset).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(state_rx.borrow().sync_state.unsafe_head(), unsafe_head);

    // A second reset lowers the heads again, and the same target is requested again.
    expect_reset_to_genesis(&l1, &l2, &genesis_block, genesis);
    requests.send(target_kind.finalize_request(target)).await.unwrap();
    actor.step().await.unwrap();
    assert!(second_reset_result.try_recv().unwrap().is_ok());
    requests
        .send(EngineActorRequest::ProcessSafeL2Signal(delegated_safe_heads(
            restored_safe,
            restored_safe,
        )))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert_eq!(actor.engine.len(), 1, "only the safe head update may remain queued");
    assert!(!actor.reset_pending);
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), genesis);

    // Once the safe head reaches the target, the repeated request finalizes it.
    l2.expect_forkchoice_updated(
        forkchoice(restored_safe, restored_safe, genesis),
        PayloadStatusEnum::Valid,
    );
    requests.send(target_kind.finalize_request(target)).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(state_rx.borrow().sync_state.safe_head(), restored_safe);
    l2.expect_block_by_number(15, Some(test_rpc_block(&target_block)));
    l2.expect_forkchoice_updated(
        forkchoice(restored_safe, restored_safe, target),
        PayloadStatusEnum::Valid,
    );
    // With nothing left to do, the actor waits for requests instead of retrying.
    assert!(time::timeout(Duration::from_secs(60), actor.step()).await.is_err());
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), target);
    assert!(actor.engine.is_empty());
    l1.assert_finished();
    l2.assert_finished();
}
