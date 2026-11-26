use super::*;
use crate::QueuedEngineDerivationClient;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceState, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    ConsolidateInput, EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    OpStackRequest, ResetRequest,
    test_utils::{
        RpcMock, TestEngineStateBuilder, TestRpcBlock, expect_sync_start, test_block_info,
        test_engine_client, test_l2_block, test_rpc_block,
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
        .send(EngineActorRequest::ProcessSafeL2Signal(ConsolidateInput::BlockInfo(
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
    requests.send(EngineActorRequest::ProcessSafeL2Signal(restored_safe.into())).await.unwrap();
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

    actor.begin_reset();
    assert_eq!(actor.last_skipped_task_error, None);
    l1.assert_finished();
    l2.assert_finished();
}

/// A chain of twenty blocks, its config, and the engine's view of them.
fn chain_of_twenty() -> (Vec<OpBlock>, Arc<RollupConfig>) {
    let mut blocks: Vec<OpBlock> = Vec::new();
    for number in 0..=20 {
        let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
        blocks.push(test_l2_block(number, parent, 0));
    }
    let mut config = RollupConfig::default();
    config.genesis.l2 = alloy_eips::BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() };
    (blocks, Arc::new(config))
}

/// Runs one request through a verifier-following engine actor over `client`, and returns the
/// signals derivation heard after the engine's local-safe head, which it hears first.
async fn run_following_verifier(
    client: EngineClient,
    config: Arc<RollupConfig>,
    state: EngineState,
    request: EngineActorRequest,
    local_safe_before: L2BlockInfo,
) -> (EngineActor<QueuedEngineDerivationClient>, watch::Receiver<EngineState>, Vec<Signal>) {
    let (state_tx, state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    )
    .following_verifier();
    requests.send(request).await.unwrap();
    time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));

    let mut signals = Vec::new();
    while let Ok(request) = derivation_rx.try_recv() {
        match request {
            crate::DerivationActorRequest::ProcessEngineSignalRequest(signal) => {
                signals.push(*signal)
            }
            crate::DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(head) |
            crate::DerivationActorRequest::ProcessEngineSyncCompletionRequest(head) => {
                assert!(signals.is_empty(), "no head may follow the restart, got {head:?}");
                assert_eq!(*head, local_safe_before);
            }
            other => panic!("unexpected derivation request {other:?}"),
        }
    }
    (actor, state_rx, signals)
}

/// A denied block the engine has derived makes the actor move local-safe to the block's parent and
/// restart derivation from there, so derivation replaces the block as it derives it again.
#[tokio::test(start_paused = true)]
async fn denied_local_safe_block_restarts_derivation_from_its_parent() {
    let (blocks, config) = chain_of_twenty();
    let info = |number: usize| {
        L2BlockInfo::from_block_and_genesis(&blocks[number], &config.genesis).unwrap()
    };
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(info(20))
        .with_safe_head(info(15))
        .with_finalized_head(info(5))
        .build();
    let rewound = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(info(11)),
        ..Default::default()
    });
    let (client, l1, l2) = test_engine_client(config.clone());
    l2.expect_block_header_by_number(12, Some(test_rpc_block(&blocks[12])));
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([info(11).hash(), true]),
        test_rpc_block(&blocks[11]),
    );
    l2.expect_forkchoice_updated(rewound.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let request = EngineActorRequest::ProcessDeniedBlocks(vec![info(12).block_info.id()]);

    let (actor, state_rx, signals) =
        run_following_verifier(client, config.clone(), state, request, info(15)).await;

    assert_eq!(signals, vec![Signal::Reset(ResetSignal { l2_safe_head: info(11) })]);
    assert_eq!(state_rx.borrow().sync_state.local_safe_head(), info(11));
    assert_eq!(state_rx.borrow().sync_state.unsafe_head(), info(20));
    assert!(actor.engine.is_empty());
    l1.assert_finished();
    l2.assert_finished();
}

/// An undo moves local-safe and the safe label back to the finalized head, restarts derivation
/// from it, and reports that the labels moved back. Derivation then consults the new denied
/// blocks.
#[tokio::test(start_paused = true)]
async fn an_undo_derives_again_from_the_finalized_head() {
    let (blocks, config) = chain_of_twenty();
    let config = Arc::new(RollupConfig {
        hardforks: kona_genesis::HardForkConfig { lagoon_time: Some(0), ..config.hardforks },
        ..(*config).clone()
    });
    let info = |number: usize| {
        L2BlockInfo::from_block_and_genesis(&blocks[number], &config.genesis).unwrap()
    };
    let state = TestEngineStateBuilder::new()
        .with_safe_label(kona_engine::SafeLabel::Verified { activation_timestamp: 0 })
        .with_unsafe_head(info(20))
        .with_local_safe_head(info(15))
        .with_safe_head(info(12))
        .with_finalized_head(info(5))
        .build();
    let undone = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(info(5)),
        safe_head: Some(info(5)),
        ..Default::default()
    });
    let (client, l1, l2) = test_engine_client(config.clone());
    l2.expect_forkchoice_updated(undone.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let (done, mut heard) = mpsc::channel(1);
    let denied = vec![info(13).block_info.id()];
    let request = EngineActorRequest::UndoVerifications(Box::new(kona_engine::UndoRequest {
        denied: denied.clone(),
        done,
    }));

    let (actor, state_rx, signals) =
        run_following_verifier(client, config.clone(), state, request, info(15)).await;

    assert_eq!(heard.try_recv(), Ok(()));
    assert_eq!(signals, vec![Signal::Reset(ResetSignal { l2_safe_head: info(5) })]);
    let state = state_rx.borrow().sync_state;
    assert_eq!((state.local_safe_head(), state.safe_head()), (info(5), info(5)));
    assert_eq!(state.unsafe_head(), info(20));
    assert_eq!(actor.denied.contains(denied[0], 0), Ok(true));
    assert!(actor.engine.is_empty());
    l1.assert_finished();
    l2.assert_finished();
}

/// A verifier's finalized head whose L1 block is not on the node's L1 chain, as for a moment
/// after an L1 reorg, finalizes nothing.
#[tokio::test(start_paused = true)]
async fn a_finalized_head_off_the_nodes_l1_chain_finalizes_nothing() {
    let (blocks, config) = chain_of_twenty();
    let info = |number: usize| {
        L2BlockInfo::from_block_and_genesis(&blocks[number], &config.genesis).unwrap()
    };
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(info(20))
        .with_safe_head(info(15))
        .with_finalized_head(info(5))
        .build();
    let (client, l1, l2) = test_engine_client(config.clone());
    let mut other = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
    other.header.inner.number = 3;
    other.header.hash = other.header.inner.hash_slow();
    l1.expect_params("eth_getBlockByNumber", serde_json::json!(["0x3", false]), other);
    let head = VerifierHead::Verified {
        block: info(10).block_info.id(),
        timestamp: 20,
        l1: alloy_eips::BlockNumHash { number: 3, hash: B256::repeat_byte(3) },
    };
    let request = EngineActorRequest::ProcessVerifiedFinalizedHead(Box::new(head));

    let (actor, state_rx, signals) =
        run_following_verifier(client, config.clone(), state, request, info(15)).await;

    assert_eq!(signals, vec![]);
    assert!(actor.engine.is_empty(), "no finalization queued");
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), info(5));
    l1.assert_finished();
    l2.assert_finished();
}

/// Following a verifier, the verifier's finalized head is final only once the node's own L1
/// finality covers it, and it bounds what L1 finality finalizes. Without a verifier, L1 finality
/// finalizes alone.
#[test]
fn verifier_finality_is_bounded_by_the_nodes_l1_finality() {
    let mut config = RollupConfig { block_time: 2, ..Default::default() };
    config.hardforks.lagoon_time = Some(0);
    let config = Arc::new(config);
    let (client, _l1, _l2) = test_engine_client(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(30))
        .with_safe_head(test_block_info(20))
        .with_finalized_head(test_block_info(10))
        .build();
    let (state_tx, _state_rx) = watch::channel(state);
    let (_requests, request_rx) = mpsc::channel(8);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let verified = test_block_info(15).block_info.id();
    let l1 = |number| FinalityUpdate::L1(FinalizeBlockId::ByNumber(number));

    let mut without_verifier = actor;
    assert_eq!(without_verifier.finality_target(l1(12)), Some(FinalizeBlockId::ByNumber(12)));

    let mut actor = without_verifier.following_verifier();
    let head = VerifierHead::Verified { block: verified, timestamp: 30, l1: Default::default() };
    assert_eq!(actor.finality_target(FinalityUpdate::Verifier(head)), None, "no L1 finality yet");
    assert_eq!(actor.finality_target(l1(12)), Some(FinalizeBlockId::ByNumber(12)));
    assert_eq!(actor.finality_target(l1(18)), Some(FinalizeBlockId::ByHash(verified)));
    assert_eq!(
        actor.finality_target(l1(8)),
        Some(FinalizeBlockId::ByHash(verified)),
        "lower L1 finality does not lower the bound"
    );
    assert_eq!(actor.finality_target(l1(9)).map(|t| t.number()), Some(15));
    let behind = VerifierHead::Verified {
        block: test_block_info(9).block_info.id(),
        timestamp: 18,
        l1: Default::default(),
    };
    assert_eq!(actor.finality_target(FinalityUpdate::Verifier(behind)), None, "behind finality");
}

/// A block committed over the opstack API goes through the engine queue, and is answered once it
/// is the unsafe head, which the node's sequencer and the RPC readers see then.
#[tokio::test(start_paused = true)]
async fn opstack_commit_runs_through_the_engine_queue() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let initial_head = test_block_info(0);
    let state = TestEngineStateBuilder::new().with_unsafe_head(initial_head).build();
    let (state_tx, state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(1);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let (unsafe_head_tx, unsafe_head_rx) = watch::channel(initial_head);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        Some(unsafe_head_tx),
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let payload = ExecutionPayloadV1::from_block_slow(&test_l2_block(1, initial_head.hash(), 0));
    let imported: OpBlock =
        OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
    let head =
        L2BlockInfo::from_block_and_genesis(&imported, &RollupConfig::default().genesis).unwrap();
    l2.expect_params(
        "engine_newPayloadV2",
        serde_json::json!([payload]),
        PayloadStatus::from_status(PayloadStatusEnum::Valid),
    );
    l2.expect_forkchoice_updated(
        forkchoice(head, initial_head, initial_head),
        PayloadStatusEnum::Valid,
    );
    let (result_tx, mut result_rx) = mpsc::channel(1);
    requests
        .send(EngineActorRequest::OpStack(Box::new(OpStackRequest::Commit {
            payload: Box::new(OpExecutionPayloadEnvelope::V1(payload)),
            result_tx,
        })))
        .await
        .unwrap();

    // The first step takes the request, the second runs it.
    actor.step().await.unwrap();
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));

    result_rx.recv().await.unwrap().unwrap();
    assert_eq!(state_rx.borrow().sync_state.unsafe_head(), head);
    assert_eq!(*unsafe_head_rx.borrow(), head);
    l1.assert_finished();
    l2.assert_finished();
}

/// A stalled engine keeps taking the verifier's polls without queuing a task per poll: the newest
/// safe and finalized heads replace the queued ones, so the inbox is drained and cannot fill up.
#[tokio::test(start_paused = true)]
async fn polled_heads_do_not_pile_up_while_the_engine_is_stalled() {
    // An unsafe import whose execution layer fails keeps the engine retrying it.
    let mut block = alloy_consensus::Block::<OpTxEnvelope>::default();
    block.header.number = 1;
    let payload = ExecutionPayloadV1::from_block_slow(&block);
    let imported: OpBlock =
        OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
    let mut config = RollupConfig::default();
    config.genesis.l2 = alloy_eips::BlockNumHash { number: 1, hash: imported.header.hash_slow() };
    let config = Arc::new(config);
    let (client, l1, l2) = test_engine_client(config.clone());
    l2.expect_error("engine_newPayloadV2");
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(0)).build();
    let (state_tx, _state_rx) = watch::channel(state);
    let polls = 50;
    let (requests, request_rx) = mpsc::channel(2 * polls + 1);
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
    requests
        .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(OpExecutionPayloadEnvelope::V1(
            payload,
        ))))
        .await
        .unwrap();
    for poll in 1..=polls as u64 {
        let block = alloy_eips::BlockNumHash { number: poll, hash: B256::repeat_byte(poll as u8) };
        let head = kona_engine::VerifierHead::Verified { block, timestamp: poll, l1: block };
        requests.send(EngineActorRequest::ProcessVerifiedSafeHead(Box::new(head))).await.unwrap();
        requests
            .send(EngineActorRequest::ProcessFinalizedL2Block(Box::new(FinalizeBlockId::ByNumber(
                poll,
            ))))
            .await
            .unwrap();
    }

    for _ in 0..=2 * polls {
        actor.step().await.unwrap();
    }

    assert_eq!(requests.capacity(), requests.max_capacity(), "the engine took every request");
    // The retried import, and one safe and one finalized head.
    assert_eq!(actor.engine.len(), 3);
    l1.assert_finished();
    l2.assert_finished();
}
