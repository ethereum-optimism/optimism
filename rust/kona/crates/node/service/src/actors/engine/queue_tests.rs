use super::*;
use crate::{DerivationActorRequest, QueuedEngineDerivationClient};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceState, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    ConsolidateInput, EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    ResetRequest,
    test_utils::{
        RpcMock, TestEngineStateBuilder, TestRpcBlock, expect_sync_start,
        expect_sync_start_answered, test_block_info, test_engine_client, test_l2_block,
        test_rpc_block,
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
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
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
            "engine_newPayloadV2",
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
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
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
            .send(EngineActorRequest::ProcessDelegatedSafeHead(Box::new(delegated_safe_head)))
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
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
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
        .send(EngineActorRequest::ProcessDelegatedSafeHead(Box::new(delegated_safe_head)))
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
    config.genesis.l1.hash = alloy_consensus::Header::default().hash_slow();
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
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
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
    l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
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
        .send(EngineActorRequest::ProcessDelegatedSafeHead(Box::new(restored_safe)))
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
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
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

    actor.begin_reset();
    assert_eq!(actor.last_skipped_task_error, None);
    l1.assert_finished();
    l2.assert_finished();
}

/// Derivation can send attributes and finalizations from its old pipeline after the engine
/// reset, while the reset signal is still queued for it. They carry the reset epoch derivation
/// had then, and are dropped on arrival instead of running against the reset head.
#[tokio::test]
async fn work_derived_before_the_latest_reset_is_dropped() {
    let config = Arc::new(RollupConfig::default());
    let (client, _l1, _l2) = test_engine_client(config.clone());
    let head = test_block_info(5);
    let state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
    let (state_tx, _state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(2);
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    actor.reset_epoch = 2;
    let attributes = |reset_epoch| EngineActorRequest::ProcessDerivedAttributes {
        attributes: Box::new(
            kona_engine::test_utils::TestAttributesBuilder::new().with_parent(head).build(),
        ),
        reset_epoch,
    };
    let finalization =
        |reset_epoch| EngineActorRequest::ProcessDerivedFinalization { number: 5, reset_epoch };

    for request in [attributes(1), finalization(1)] {
        requests.send(request).await.unwrap();
        actor.step().await.unwrap();
        assert!(actor.engine.is_empty());
    }
    // Hold the queue so the accepted work is counted, not run.
    actor.retry_at = Instant::now() + Duration::from_secs(3600);
    for (request, queued) in [(attributes(2), 1), (finalization(2), 2)] {
        requests.send(request).await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(actor.engine.len(), queued);
    }
}

/// A drain can move the local-safe head and then hit a task that asks for a reset. The new head
/// is reported before the reset starts, or derivation would never record it.
#[tokio::test]
async fn local_safe_head_is_reported_before_a_reset() {
    let config = Arc::new(RollupConfig::default());
    let (client, _l1, _l2) = test_engine_client(config.clone());
    let client = Arc::new(client);
    let head = test_block_info(5);
    let state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
    let (state_tx, _state_rx) = watch::channel(state);
    let (_requests, request_rx) = mpsc::channel(1);
    let (derivation_tx, mut derivation_rx) = mpsc::unbounded_channel();
    let mut actor = EngineActor::new(
        client.clone(),
        config.clone(),
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    // As if earlier tasks in this drain moved the local-safe head past the last head sent.
    actor.last_engine_safe_head_sent = test_block_info(4);
    // Attributes that don't extend the local-safe head ask for a reset without an engine call.
    actor.engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
        client,
        config,
        ConsolidateInput::Attributes(Box::new(
            kona_engine::test_utils::TestAttributesBuilder::new()
                .with_parent(test_block_info(9))
                .build(),
        )),
        Arc::new(NoopBlockSink),
    ))));

    actor.drain().await.unwrap();

    assert!(actor.reset_pending);
    let Ok(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(sent)) =
        derivation_rx.try_recv()
    else {
        panic!("expected the local-safe head before the reset");
    };
    assert_eq!(*sent, head);
}

/// A reset drops queued sequencer work. Each requester's reply channel closes rather than staying
/// open, so the sequencer learns that its request was dropped instead of waiting forever.
#[tokio::test(start_paused = true)]
async fn reset_closes_the_reply_channels_of_queued_seal_and_canonicalize() {
    let config = Arc::new(RollupConfig::default());
    let (client, _l1, _l2) = test_engine_client(config.clone());
    let head = test_block_info(5);
    let state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
    let (state_tx, _state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(3);
    let (derivation_tx, _derivation_rx) = mpsc::unbounded_channel();
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    // Hold the queue so the requests are queued, not run.
    actor.retry_at = Instant::now() + Duration::from_secs(3600);
    let (seal_tx, mut seal_rx) = mpsc::channel(1);
    let (canonicalize_tx, mut canonicalize_rx) = mpsc::channel(1);
    let payload = ExecutionPayloadV1::from_block_slow(&test_l2_block(6, head.hash(), 0));
    for request in [
        EngineActorRequest::Seal(Box::new(SealRequest {
            payload_id: PayloadId::new([1; 8]),
            attributes: kona_engine::test_utils::TestAttributesBuilder::new()
                .with_parent(head)
                .build(),
            result_tx: seal_tx,
        })),
        EngineActorRequest::Canonicalize(Box::new(CanonicalizeRequest {
            payload: OpExecutionPayloadEnvelope::V1(payload),
            parent: head,
            result_tx: canonicalize_tx,
        })),
    ] {
        requests.send(request).await.unwrap();
        actor.step().await.unwrap();
    }
    assert_eq!(actor.engine.len(), 2);

    let (reset, _reset_result) = reset_request();
    requests.send(reset).await.unwrap();
    actor.step().await.unwrap();

    assert!(actor.reset_pending);
    assert!(actor.engine.is_empty());
    assert!(seal_rx.recv().await.is_none());
    assert!(canonicalize_rx.recv().await.is_none());
}

/// A rollup config whose L2 genesis is the default block, with that block and its info.
fn genesis_at_default_block() -> (Arc<RollupConfig>, TestRpcBlock, L2BlockInfo) {
    let genesis_block = TestRpcBlock::default();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = genesis_block.header.inner.hash_slow();
    config.genesis.l1.hash = alloy_consensus::Header::default().hash_slow();
    let genesis = L2BlockInfo::from_block_and_genesis(
        &genesis_block.clone().into_consensus(),
        &config.genesis,
    )
    .unwrap();
    (Arc::new(config), genesis_block, genesis)
}

/// An engine actor with no engine state yet, as at node startup, with its request sender and the
/// receiver of its messages to derivation.
fn starting_actor(
    config: Arc<RollupConfig>,
    client: EngineClient,
) -> (
    EngineActor<QueuedEngineDerivationClient>,
    mpsc::Sender<EngineActorRequest>,
    mpsc::UnboundedReceiver<DerivationActorRequest>,
) {
    let (requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, derivation_rx) = mpsc::unbounded_channel();
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(EngineState::default(), state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    (actor, requests, derivation_rx)
}

/// Takes the next message to derivation, which must be a reset signal, and returns its safe head.
fn reset_signal_safe_head(
    derivation_rx: &mut mpsc::UnboundedReceiver<DerivationActorRequest>,
) -> L2BlockInfo {
    let Ok(DerivationActorRequest::ProcessEngineSignalRequest(signal)) = derivation_rx.try_recv()
    else {
        panic!("expected a reset signal");
    };
    let kona_derive::Signal::Reset(reset) = *signal else {
        panic!("expected a reset signal");
    };
    reset.l2_safe_head
}

/// Takes the next message to derivation, which must be the EL-sync completion, and returns its
/// safe head.
fn sync_completion_safe_head(
    derivation_rx: &mut mpsc::UnboundedReceiver<DerivationActorRequest>,
) -> L2BlockInfo {
    let Ok(DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head)) =
        derivation_rx.try_recv()
    else {
        panic!("expected the EL-sync completion");
    };
    *safe_head
}

/// In consensus-layer sync mode, a validator resets at startup with no request to prompt it. A
/// `VALID` answer to the reset's forkchoice update completes EL sync, so derivation is reset to the
/// recovered safe head and starts without waiting for an unsafe payload.
#[tokio::test(start_paused = true)]
async fn startup_reset_completes_el_sync_without_any_request() {
    let (config, genesis_block, genesis) = genesis_at_default_block();
    let (client, l1, l2) = test_engine_client(config.clone());
    let (actor, _requests, mut derivation_rx) = starting_actor(config, client);
    let mut actor = actor.with_startup_reset();
    expect_reset_to_genesis(&l1, &l2, &genesis_block, genesis);

    // The step resets and then waits for a request that never comes.
    assert!(time::timeout(Duration::from_secs(1), actor.step()).await.is_err());

    assert!(!actor.reset_pending);
    assert!(actor.el_sync_complete);
    assert_eq!(reset_signal_safe_head(&mut derivation_rx), genesis);
    assert_eq!(sync_completion_safe_head(&mut derivation_rx), genesis);
    assert!(derivation_rx.try_recv().is_err());
    l1.assert_finished();
    l2.assert_finished();
}

/// When the execution layer answers the startup reset's forkchoice update with `SYNCING`, the
/// reset completes without completing EL sync and is not retried, which would hold back queued
/// work. The queued unsafe payload is then imported, and its `VALID` forkchoice update completes
/// EL sync as it does without a startup reset.
#[tokio::test(start_paused = true)]
async fn startup_reset_answered_syncing_leaves_el_sync_to_unsafe_payloads() {
    let (config, genesis_block, genesis) = genesis_at_default_block();
    let (client, l1, l2) = test_engine_client(config.clone());
    let (actor, requests, mut derivation_rx) = starting_actor(config.clone(), client);
    let mut actor = actor.with_startup_reset();
    expect_sync_start_answered(
        &l1,
        &l2,
        &genesis_block,
        forkchoice(genesis, genesis, genesis),
        PayloadStatusEnum::Syncing,
    );
    let unsafe_payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &test_l2_block(1, genesis.hash(), 0),
    ));
    let unsafe_head = L2BlockInfo::from_block_and_genesis(
        &unsafe_payload.clone().try_into_block::<OpTxEnvelope>().unwrap(),
        &config.genesis,
    )
    .unwrap();
    requests
        .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(unsafe_payload)))
        .await
        .unwrap();

    // The reset runs first; the unsafe payload is queued behind it.
    actor.step().await.unwrap();
    assert!(!actor.reset_pending, "the startup reset must not be retried");
    assert!(!actor.el_sync_complete);
    assert_eq!(actor.engine.len(), 1, "the unsafe payload waits in the queue");
    assert_eq!(reset_signal_safe_head(&mut derivation_rx), genesis);
    assert!(derivation_rx.try_recv().is_err(), "EL sync is not complete yet");
    l1.assert_finished();
    l2.assert_finished();

    l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect_forkchoice_updated(
        forkchoice(unsafe_head, genesis, genesis),
        PayloadStatusEnum::Valid,
    );
    assert!(time::timeout(Duration::from_secs(1), actor.step()).await.is_err());

    assert!(actor.el_sync_complete);
    assert!(actor.engine.is_empty());
    assert_eq!(sync_completion_safe_head(&mut derivation_rx), genesis);
    assert!(derivation_rx.try_recv().is_err(), "EL sync completes without another reset");
    l1.assert_finished();
    l2.assert_finished();
}

/// In execution-layer sync mode, a validator starts without a reset: it sends nothing to its
/// execution layer or to derivation until a request arrives.
#[tokio::test(start_paused = true)]
async fn without_a_startup_reset_the_actor_waits_for_a_request() {
    let (config, _genesis_block, _genesis) = genesis_at_default_block();
    let (client, l1, l2) = test_engine_client(config.clone());
    let (mut actor, _requests, mut derivation_rx) = starting_actor(config, client);

    assert!(time::timeout(Duration::from_secs(1), actor.step()).await.is_err());

    assert!(!actor.reset_pending);
    assert!(!actor.el_sync_complete);
    assert!(derivation_rx.try_recv().is_err());
    l1.assert_finished();
    l2.assert_finished();
}

/// An execution layer that is unreachable when the startup reset runs fails it with a temporary
/// error. The reset stays pending and holds back the unsafe payload queued behind it, then succeeds
/// after backoff; its `VALID` answer completes EL sync, and the payload is imported afterwards.
#[tokio::test(start_paused = true)]
async fn startup_reset_retries_after_the_execution_layer_is_unreachable() {
    let (config, genesis_block, genesis) = genesis_at_default_block();
    let (client, l1, l2) = test_engine_client(config.clone());
    let (actor, requests, mut derivation_rx) = starting_actor(config.clone(), client);
    let mut actor = actor.with_startup_reset();
    l2.expect_error("eth_getBlockByNumber");
    let unsafe_payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &test_l2_block(1, genesis.hash(), 0),
    ));
    let unsafe_head = L2BlockInfo::from_block_and_genesis(
        &unsafe_payload.clone().try_into_block::<OpTxEnvelope>().unwrap(),
        &config.genesis,
    )
    .unwrap();
    requests
        .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(unsafe_payload)))
        .await
        .unwrap();

    actor.step().await.unwrap();
    assert!(actor.reset_pending, "a failed reset is retained");
    assert_eq!(actor.engine.len(), 1, "the unsafe payload waits behind the reset");
    assert!(!actor.el_sync_complete);
    assert!(derivation_rx.try_recv().is_err());
    l2.assert_finished();

    expect_reset_to_genesis(&l1, &l2, &genesis_block, genesis);
    l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect_forkchoice_updated(
        forkchoice(unsafe_head, genesis, genesis),
        PayloadStatusEnum::Valid,
    );
    // The actor retries after backoff with no new request, then imports the payload.
    let steps = async {
        loop {
            actor.step().await.unwrap();
        }
    };
    assert!(time::timeout(Duration::from_secs(1), steps).await.is_err());

    assert!(!actor.reset_pending);
    assert!(actor.el_sync_complete);
    assert!(actor.engine.is_empty());
    assert_eq!(reset_signal_safe_head(&mut derivation_rx), genesis);
    assert_eq!(sync_completion_safe_head(&mut derivation_rx), genesis);
    assert!(derivation_rx.try_recv().is_err());
    l1.assert_finished();
    l2.assert_finished();
}
