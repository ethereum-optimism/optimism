use super::*;
use alloy_eips::BlockId;
use alloy_primitives::{B256, Sealed};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use jsonrpsee::rpc_params;
use kona_engine::{
    NoopBlockSink,
    test_utils::{
        ExpectedCall, MockEngineClient, Script, TestAttributesBuilder, TestEngineStateBuilder,
        test_block_info,
    },
};
use kona_protocol::{BlockInfo, L1BlockInfoBedrock};
use kona_rpc::{AdminApiServer, AdminRpc};
use op_alloy_consensus::{OpBlock, OpTxEnvelope, TxDeposit};
use tokio::sync::oneshot;

fn valid() -> PayloadStatus {
    PayloadStatus::from_status(PayloadStatusEnum::Valid)
}
fn fcu() -> ForkchoiceUpdated {
    ForkchoiceUpdated::new(valid())
}
fn build_reply() -> ForkchoiceUpdated {
    ForkchoiceUpdated { payload_id: Some(PayloadId::new([1; 8])), ..fcu() }
}

fn ready<C: EngineClient + 'static>(
    client: Arc<C>,
    cfg: Arc<RollupConfig>,
    parent: L2BlockInfo,
) -> (EngineActor<C>, EngineHandle, watch::Sender<L1State>) {
    let (l1, observations) = watch::channel(L1State::default());
    let (mut actor, handle) =
        EngineActor::with_capacities(client, cfg, Arc::new(NoopBlockSink), observations, 2, 2);
    actor.state = TestEngineStateBuilder::new()
        .with_unsafe_head(parent)
        .with_safe_head(parent)
        .with_finalized_head(parent)
        .build();
    actor.state.phase = EnginePhase::Ready;
    actor.lifecycle = Lifecycle::Ready;
    actor.checked_origins = actor.origins();
    actor.publish();
    (actor, handle, l1)
}

fn enqueue(handle: &EngineHandle, message: Message) {
    handle.commands.try_send(message).unwrap();
    handle.pending.send_modify(|count| *count += 1);
}

fn request_reset(handle: &EngineHandle) -> oneshot::Receiver<EngineClientResult<()>> {
    let (tx, rx) = oneshot::channel();
    handle.resets.try_send(tx).unwrap();
    handle.pending.send_modify(|count| *count += 1);
    rx
}

fn payload(number: u64, parent_hash: B256) -> OpExecutionPayloadEnvelope {
    let info = L1BlockInfoBedrock::new_from_sequence_number(number - 1);
    let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: info.encode_calldata(),
        ..Default::default()
    }));
    let mut block = OpBlock::default();
    block.header.number = number;
    block.header.timestamp = number * 2;
    block.header.parent_hash = parent_hash;
    block.body.transactions.push(deposit);
    let mut envelope = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
    let rebuilt = envelope.clone().try_into_block::<OpTxEnvelope>().unwrap();
    envelope.as_v1_mut().block_hash = rebuilt.header.hash_slow();
    envelope
}

struct ResetFixture {
    client: Arc<MockEngineClient>,
    config: Arc<RollupConfig>,
    genesis: L2BlockInfo,
    l1: watch::Sender<L1State>,
}
impl ResetFixture {
    fn new() -> Self {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let l1: alloy_rpc_types_eth::Block = alloy_rpc_types_eth::Block::new(
            alloy_rpc_types_eth::Header::new(Default::default()),
            alloy_rpc_types_eth::BlockTransactions::Hashes(vec![]),
        );
        let mut config = RollupConfig::default();
        config.genesis.l1 =
            alloy_eips::BlockNumHash { number: l1.header.number, hash: l1.header.hash };
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let genesis = L2BlockInfo {
            block_info: BlockInfo { hash: config.genesis.l2.hash, ..Default::default() },
            l1_origin: config.genesis.l1,
            seq_num: 0,
        };
        let config = Arc::new(config);
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_l2_block(BlockId::finalized(), block.clone())
                .with_l2_block(BlockId::safe(), block.clone())
                .with_l2_block(config.genesis.l2.hash.into(), block.clone())
                .with_l2_block(config.genesis.l2.number.into(), block.clone())
                .with_l2_block(BlockId::latest(), block)
                .with_l1_block(BlockId::latest(), l1.clone())
                .with_l1_block(config.genesis.l1.number.into(), l1)
                .with_fork_choice_updated_v3_response(fcu())
                .build(),
        );
        let (l1, _) = watch::channel(L1State::default());
        Self { client, config, genesis, l1 }
    }
}

/// Recovery retains the original acknowledgement, including a reset taking longer than its
/// retry interval. No additional L1 event or producer request is needed to finish it.
#[rstest::rstest]
#[case::l1(true, Duration::ZERO)]
#[case::l2(false, Duration::ZERO)]
#[case::slow_l2(false, Duration::from_secs(4))]
#[tokio::test(start_paused = true)]
async fn registered_reset_recovers_and_publishes_before_acknowledgement(
    #[case] l1_failure: bool,
    #[case] read_delay: Duration,
) {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    {
        let mut data = storage.write().await;
        if l1_failure {
            data.l1_read_error = Some("L1 timeout".into());
        } else {
            data.l2_read_error = Some("L2 timeout".into());
        }
    }
    let (mut actor, handle) = EngineActor::new(
        fixture.client,
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    // Keep a real pipeline actor unpolled with a full L1 notification queue: RPC completion
    // depends on engine application and publication, never on derivation draining callbacks.
    let (l1, requests) = mpsc::channel(1);
    l1.try_send(crate::DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::default())).unwrap();
    let _derivation = crate::DerivationActor::new(
        handle.clone(),
        requests,
        kona_derive::test_utils::new_test_pipeline(),
        watch::channel(kona_rpc::DerivationStatus::default()).0,
    );
    let rpc = AdminRpc::new(handle.clone()).into_rpc();
    let reply = rpc.call::<_, ()>("admin_resetDerivationPipeline", rpc_params![]);
    tokio::pin!(reply);
    assert!(futures::poll!(reply.as_mut()).is_pending());
    actor.step().await.unwrap();
    actor.step().await.unwrap();
    assert!(futures::poll!(reply.as_mut()).is_pending());
    assert_ne!(handle.state().phase, EnginePhase::Ready);
    let epoch = handle.state().epoch;
    {
        let mut data = storage.write().await;
        data.l1_read_error = None;
        data.l2_read_error = None;
        data.l2_read_delay = read_delay;
    }
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    // The published state must already be complete before polling the RPC result again.
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    reply.await.unwrap();
    assert_eq!(l1.capacity(), 0);
    assert_eq!(handle.state().epoch, epoch);
    assert_eq!(handle.state().sync_state.safe_head(), fixture.genesis);
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
}

#[tokio::test(start_paused = true)]
async fn full_unsafe_admission_does_not_block_reset_and_old_epoch_work_is_purged() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let (mut actor, handle, _l1) = ready(fixture.client, fixture.config, fixture.genesis);
    let first = payload(1, fixture.genesis.hash());
    handle.try_insert_unsafe(first.clone()).unwrap();
    // Block the first RPC so admission and reset control are exercised during an active attempt.
    let locked = storage.write().await;
    let old_context = EngineContext { epoch: 0, parent: fixture.genesis };
    let (tx, old_reply) = oneshot::channel();
    let reset_reply;
    {
        let attempt = actor.step();
        tokio::pin!(attempt);
        assert!(futures::poll!(attempt.as_mut()).is_pending());
        handle.try_insert_unsafe(first.clone()).unwrap();
        handle.try_insert_unsafe(first.clone()).unwrap();
        assert_eq!(handle.try_insert_unsafe(first), Err(super::super::UnsafeAdmissionError::Full));
        enqueue(
            &handle,
            Message::Start(
                old_context,
                TestAttributesBuilder::new().with_parent(fixture.genesis).build(),
                tx,
            ),
        );
        reset_reply = request_reset(&handle);
        assert_eq!(handle.state().epoch, 0);
        drop(locked);
        attempt.await.unwrap();
    }
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Resetting);
    assert_eq!(handle.state().epoch, 1);
    assert!(actor.active_unsafe.is_none());
    assert!(actor.unsafe_payloads.is_empty());
    actor.step().await.unwrap();
    reset_reply.await.unwrap().unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head(), fixture.genesis);
    actor.step().await.unwrap();
    assert!(matches!(old_reply.await.unwrap(), Err(EngineClientError::Stale)));
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
}

#[tokio::test(start_paused = true)]
async fn unsafe_retry_keeps_fifo_order_while_commands_make_progress() {
    let script = Script::new([
        ExpectedCall::error("engine_newPayloadV1", -32000),
        ExpectedCall::success("engine_forkchoiceUpdatedV2", build_reply()),
        ExpectedCall::success("engine_newPayloadV1", valid()),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
        ExpectedCall::success("engine_newPayloadV1", valid()),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
    ]);
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    let first = payload(2, parent.hash());
    let second = payload(3, first.block_hash());
    handle.try_insert_unsafe(first.clone()).unwrap();
    handle.try_insert_unsafe(second.clone()).unwrap();
    actor.step().await.unwrap();
    assert_eq!(actor.active_unsafe.as_ref().unwrap().1.block_hash(), first.block_hash());
    let (tx, reply) = oneshot::channel();
    enqueue(
        &handle,
        Message::Start(
            EngineContext { epoch: 0, parent },
            TestAttributesBuilder::new().with_parent(parent).build(),
            tx,
        ),
    );
    actor.step().await.unwrap();
    assert_eq!(reply.await.unwrap().unwrap().id, PayloadId::new([1; 8]));
    assert_eq!(handle.state().sync_state.unsafe_head(), parent);
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head().hash(), first.block_hash());
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head().hash(), second.block_hash());
    let numbers: Vec<_> = script
        .calls()
        .into_iter()
        .filter(|call| call.method == "engine_newPayloadV1")
        .map(|call| call.params[0]["blockNumber"].clone())
        .collect();
    assert_eq!(
        numbers,
        vec![serde_json::json!("0x2"), serde_json::json!("0x2"), serde_json::json!("0x3")]
    );
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
    script.assert_finished();
}

#[tokio::test(start_paused = true)]
async fn accepted_safe_head_supersedes_a_retained_unsafe_retry() {
    let script = Script::new([
        ExpectedCall::error("engine_newPayloadV1", -32000),
        ExpectedCall::success("eth_getBlockByNumber", Option::<serde_json::Value>::None),
        ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
    ]);
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let first = payload(2, parent.hash());
    let block = first.clone().try_into_block::<OpTxEnvelope>().unwrap();
    let safe = L2BlockInfo::from_block_and_genesis(&block, &config.genesis).unwrap();
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    handle.try_insert_unsafe(first).unwrap();
    actor.step().await.unwrap();
    let (tx, reply) = oneshot::channel();
    enqueue(&handle, Message::FollowSafe(EngineContext { epoch: 0, parent }, safe, tx));
    actor.step().await.unwrap();
    reply.await.unwrap().unwrap();
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert!(actor.active_unsafe.is_none());
    assert_eq!(handle.state().sync_state.safe_head(), safe);
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
    script.assert_finished();
}

#[tokio::test]
async fn temporary_reply_observes_an_accepted_build_parent_even_without_a_payload_id() {
    let script = Script::new([
        ExpectedCall::success("eth_getBlockByNumber", Option::<serde_json::Value>::None),
        ExpectedCall::success("engine_forkchoiceUpdatedV2", fcu()),
    ]);
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    actor.state.sync_state = actor.state.sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(test_block_info(5)),
        ..Default::default()
    });
    actor.state.el_sync_finished = false;
    actor.publish();
    let mut attrs = TestAttributesBuilder::new().with_parent(parent).build();
    attrs.derived_from = Some(BlockInfo::default());
    let (tx, reply) = oneshot::channel();
    enqueue(&handle, Message::Consolidate(EngineContext { epoch: 0, parent }, attrs, tx));
    actor.step().await.unwrap();
    assert!(matches!(reply.await.unwrap(), Err(EngineClientError::Temporary(_))));
    assert_eq!(handle.state().sync_state.unsafe_head(), parent);
    assert!(handle.state().el_sync_finished);
    assert_eq!(handle.state().sync_state.safe_head(), parent);
    script.assert_finished();
}

#[tokio::test]
async fn dropped_reply_receiver_does_not_stop_the_owner() {
    let script = Script::new([ExpectedCall::success("engine_forkchoiceUpdatedV2", build_reply())]);
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    let (tx, reply) = oneshot::channel();
    drop(reply);
    enqueue(
        &handle,
        Message::Start(
            EngineContext { epoch: 0, parent },
            TestAttributesBuilder::new().with_parent(parent).build(),
            tx,
        ),
    );
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
    script.assert_finished();
}

async fn build_failure_invalidates_epoch<C: EngineClient + 'static>(
    client: Arc<C>,
    config: Arc<RollupConfig>,
) {
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(client, config, parent);
    let (tx, reply) = oneshot::channel();
    enqueue(
        &handle,
        Message::Start(
            EngineContext { epoch: 0, parent },
            TestAttributesBuilder::new().with_parent(parent).build(),
            tx,
        ),
    );
    actor.step().await.unwrap();
    assert!(matches!(reply.await.unwrap(), Err(EngineClientError::Temporary(_))));
    assert_eq!(handle.state().phase, EnginePhase::Resetting);
    assert_eq!(handle.state().epoch, 1);
    assert_eq!(handle.state().sync_state.unsafe_head(), parent);
}

#[tokio::test]
async fn invalid_forkchoice_starts_an_owner_reset() {
    let script = Script::new([ExpectedCall::error("engine_forkchoiceUpdatedV2", -38002)]);
    let config = Arc::new(RollupConfig::default());
    build_failure_invalidates_epoch(Arc::new(script.client(config.clone())), config).await;
    script.assert_finished();
}

#[tokio::test]
async fn ambiguous_forkchoice_transport_failure_starts_an_owner_reset() {
    // An unset mock response returns a transport error, leaving FCU application unknown.
    let config = Arc::new(RollupConfig::default());
    let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
    build_failure_invalidates_epoch(client, config).await;
}

#[tokio::test]
async fn registered_unsafe_admission_waits_for_engine_capacity_without_executing_early() {
    let script = Script::new((0..3).flat_map(|_| {
        [
            ExpectedCall::success("engine_newPayloadV1", valid()),
            ExpectedCall::success("engine_forkchoiceUpdatedV3", fcu()),
        ]
    }));
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    let rpc = AdminRpc::new(handle.clone()).into_rpc();
    let first = payload(2, parent.hash());
    let second = payload(3, first.block_hash());
    let third = payload(4, second.block_hash());
    for block in [&first, &second] {
        rpc.call::<_, ()>("admin_postUnsafePayload", rpc_params![block]).await.unwrap();
    }
    assert_eq!(actor.unsafe_payloads.len(), 2);
    assert_eq!(handle.state().sync_state.unsafe_head(), parent);
    assert!(script.calls().is_empty());
    let admission = rpc.call::<_, ()>("admin_postUnsafePayload", rpc_params![third.clone()]);
    tokio::pin!(admission);
    assert!(futures::poll!(admission.as_mut()).is_pending());
    actor.step().await.unwrap();
    admission.await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head().hash(), first.block_hash());
    assert_eq!(*handle.queue_length_subscribe().borrow(), 2);
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head().hash(), second.block_hash());
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head().hash(), third.block_hash());
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
    script.assert_finished();
}

#[tokio::test(start_paused = true)]
async fn idle_initialization_requires_fresh_valid_and_consumes_bootstrap_only_once() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    storage.write().await.fork_choice_updated_v3_response =
        Some(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)));
    let (mut actor, handle) = EngineActor::new(
        fixture.client,
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    // No producer, L1 notification, or reset request wakes initialization.
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Syncing);
    assert!(matches!(actor.lifecycle, Lifecycle::Bootstrap(SyncStartPolicy::TrustedResume { .. })));
    actor.state.el_sync_finished = true; // A past VALID must not bypass current SYNCING.
    let mut reply = request_reset(&handle);
    let epoch = handle.state().epoch;
    actor.step().await.unwrap();
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Syncing);
    assert!(matches!(reply.try_recv(), Err(oneshot::error::TryRecvError::Empty)));
    storage.write().await.fork_choice_updated_v3_response = Some(fcu());
    time::advance(Duration::from_millis(200)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.safe_head(), fixture.genesis);
    assert_eq!(handle.state().sync_state.finalized_head(), fixture.genesis);
    assert_eq!(handle.state().epoch, epoch);
    reply.await.unwrap().unwrap();
    assert_eq!(actor.lifecycle, Lifecycle::Ready);

    let second = request_reset(&handle);
    actor.step().await.unwrap();
    assert_eq!(
        actor.lifecycle,
        Lifecycle::Reset(SyncStartPolicy::Checkpoint {
            safe: fixture.genesis,
            finalized: fixture.genesis,
        })
    );
    actor.step().await.unwrap();
    second.await.unwrap().unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
}

#[tokio::test(start_paused = true)]
async fn an_initial_probe_failure_retries_without_a_new_event() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    storage.write().await.syncing_error = Some("EL unavailable".into());
    let (mut actor, handle) = EngineActor::new(
        fixture.client,
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    assert!(matches!(actor.lifecycle, Lifecycle::Bootstrap(_)));
    assert_ne!(handle.state().phase, EnginePhase::Ready);
    storage.write().await.syncing_error = None;
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
}

struct TestBlock {
    block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>,
    reference: L2BlockInfo,
    payload: OpExecutionPayloadEnvelope,
}

fn block_with_origin(parent: L2BlockInfo, origin: BlockNumHash) -> TestBlock {
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    let sequence = if parent.l1_origin == origin { parent.seq_num + 1 } else { 0 };
    let info = L1BlockInfoBedrock::new(
        origin.number,
        0,
        0,
        origin.hash,
        sequence,
        Default::default(),
        Default::default(),
        Default::default(),
    );
    let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: info.encode_calldata(),
        ..Default::default()
    }));
    let transactions_root =
        alloy_consensus::proofs::calculate_transaction_root(std::slice::from_ref(&deposit));
    let transaction = op_alloy_rpc_types::Transaction {
        inner: alloy_rpc_types_eth::Transaction {
            inner: alloy_consensus::transaction::Recovered::new_unchecked(
                deposit,
                Default::default(),
            ),
            block_hash: None,
            block_number: Some(parent.block_info.number + 1),
            effective_gas_price: Some(1),
            transaction_index: Some(0),
            block_timestamp: None,
        },
        deposit_nonce: None,
        deposit_receipt_version: None,
    };
    let block = Block::new(
        Header::new(alloy_consensus::Header {
            number: parent.block_info.number + 1,
            ommers_hash: alloy_consensus::EMPTY_OMMER_ROOT_HASH,
            base_fee_per_gas: Some(0),
            transactions_root,
            timestamp: parent.block_info.timestamp + 2,
            parent_hash: parent.hash(),
            ..Default::default()
        }),
        BlockTransactions::Full(vec![transaction]),
    );
    let consensus =
        block.clone().into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
    let reference = L2BlockInfo {
        block_info: BlockInfo {
            hash: block.header.hash,
            number: block.header.number,
            parent_hash: parent.hash(),
            timestamp: block.header.timestamp,
        },
        l1_origin: origin,
        seq_num: sequence,
    };
    let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&consensus));
    TestBlock { block, reference, payload }
}

#[tokio::test(start_paused = true)]
async fn initial_sync_keeps_the_accepted_unsafe_target_instead_of_stale_latest() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let TestBlock { block, reference: target, payload } =
        block_with_origin(fixture.genesis, fixture.config.genesis.l1);
    {
        let mut data = storage.write().await;
        data.el_syncing = true;
        data.syncing_info.highest_block = alloy_primitives::U256::from(1);
        data.new_payload_v1_response = Some(PayloadStatus::from_status(PayloadStatusEnum::Syncing));
        data.fork_choice_updated_v3_response =
            Some(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)));
    }
    let (mut actor, handle) = EngineActor::new(
        fixture.client.clone(),
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    handle.try_insert_unsafe(payload).unwrap();
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head(), target);
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert!(
        storage
            .read()
            .await
            .forkchoice_calls
            .iter()
            .all(|fcu| fcu.head_block_hash == target.hash())
    );
    {
        let mut data = storage.write().await;
        data.el_syncing = false;
        data.fork_choice_updated_v3_response = Some(fcu());
    }
    fixture.client.set_l2_block(BlockId::latest(), block.clone()).await;
    fixture.client.set_l2_block(target.hash().into(), block).await;
    time::advance(Duration::from_millis(200)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.unsafe_head(), target);
    assert!(
        storage
            .read()
            .await
            .forkchoice_calls
            .iter()
            .all(|fcu| fcu.head_block_hash == target.hash())
    );
}

#[tokio::test(start_paused = true)]
async fn trusted_follow_sync_retains_its_target_without_reopening_bootstrap() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let TestBlock { block, reference: target, .. } =
        block_with_origin(fixture.genesis, fixture.config.genesis.l1);
    let (mut actor, handle, _l1) = ready(fixture.client.clone(), fixture.config, fixture.genesis);
    storage.write().await.fork_choice_updated_v3_response =
        Some(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)));
    let (tx, reply) = oneshot::channel();
    enqueue(
        &handle,
        Message::FollowSafe(EngineContext { epoch: 0, parent: fixture.genesis }, target, tx),
    );
    actor.step().await.unwrap();
    reply.await.unwrap().unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Syncing);
    assert!(matches!(actor.lifecycle, Lifecycle::Following { .. }));
    let requested = actor.sync_target.expect("following retains the exact requested tuple");
    actor.step().await.unwrap();
    assert_ne!(handle.state().phase, EnginePhase::Ready);
    storage.write().await.fork_choice_updated_v3_response = Some(fcu());
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    // A VALID forkchoice is insufficient while the target body is still unavailable.
    assert!(matches!(actor.lifecycle, Lifecycle::Following { .. }));
    assert_ne!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(actor.sync_target, Some(requested));
    fixture.client.set_l2_block(target.block_info.number.into(), block).await;
    time::advance(Duration::from_millis(200)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.safe_head(), target);
    assert!(
        storage
            .read()
            .await
            .forkchoice_calls
            .iter()
            .all(|fcu| fcu.head_block_hash == target.hash())
    );
    actor.begin_reset();
    assert_eq!(
        actor.lifecycle,
        Lifecycle::Reset(SyncStartPolicy::Checkpoint { safe: target, finalized: fixture.genesis })
    );
}

#[tokio::test(start_paused = true)]
async fn origin_holes_retry_and_same_height_reorgs_trigger_one_owner_reset() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let (mut actor, handle, l1) = ready(fixture.client.clone(), fixture.config, fixture.genesis);
    let head = BlockInfo { number: 1, hash: B256::repeat_byte(1), ..Default::default() };
    l1.send_replace(L1State { head: Some(head), ..Default::default() });
    let original = storage.read().await.l1_blocks_by_id.clone();
    storage.write().await.l1_blocks_by_id.clear();
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert!(actor.origins_dirty);
    storage.write().await.l1_blocks_by_id = original;
    time::advance(Duration::from_secs(1)).await;
    actor.step().await.unwrap();
    assert!(!actor.origins_dirty);
    // Four heads share one origin: only one numeric lookup per successful check.
    assert_eq!(storage.read().await.l1_reads.len(), 2);
    let wrong = alloy_rpc_types_eth::Block::new(
        alloy_rpc_types_eth::Header::new(alloy_consensus::Header {
            extra_data: vec![1].into(),
            ..Default::default()
        }),
        alloy_rpc_types_eth::BlockTransactions::Hashes(vec![]),
    );
    fixture.client.set_l1_block(fixture.genesis.l1_origin.number.into(), wrong).await;
    l1.send_replace(L1State {
        head: Some(BlockInfo { hash: B256::repeat_byte(2), ..head }),
        ..Default::default()
    });
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Resetting);
    assert_eq!(handle.state().epoch, 1);
    let epoch = handle.state().epoch;
    actor.begin_reset();
    assert_eq!(handle.state().epoch, epoch);
}

#[tokio::test(start_paused = true)]
async fn initialization_prefers_a_known_safe_checkpoint_to_trusted_resume() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let TestBlock { block, reference: safe, .. } =
        block_with_origin(fixture.genesis, fixture.config.genesis.l1);
    fixture.client.set_l2_block(BlockId::safe(), block.clone()).await;
    fixture.client.set_l2_block(BlockId::latest(), block.clone()).await;
    fixture.client.set_l2_block(safe.hash().into(), block).await;
    storage.write().await.fork_choice_updated_v3_response =
        Some(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)));
    let (mut actor, handle) = EngineActor::new(
        fixture.client,
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    assert_eq!(
        actor.lifecycle,
        Lifecycle::Bootstrap(SyncStartPolicy::Checkpoint { safe, finalized: fixture.genesis })
    );
    storage.write().await.fork_choice_updated_v3_response = Some(fcu());
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert!(handle.state().sync_state.safe_head().block_info.number <= safe.block_info.number);
    assert_eq!(handle.state().sync_state.finalized_head(), fixture.genesis);
}

#[rstest::rstest]
#[case::observed_head(true)]
#[case::no_observation(false)]
#[tokio::test]
async fn changed_applied_origins_are_checked_even_without_a_new_l1_head(#[case] observed: bool) {
    let fixture = ResetFixture::new();
    let (mut actor, handle, l1) = ready(fixture.client, fixture.config, fixture.genesis);
    if observed {
        l1.send_replace(L1State {
            head: Some(BlockInfo { number: 1, ..Default::default() }),
            ..Default::default()
        });
        actor.step().await.unwrap();
    }
    let mut next = fixture.genesis;
    next.l1_origin.hash = B256::repeat_byte(9);
    actor.state.pending_safe_head = next;
    actor.publish();
    let (tx, _reply) = oneshot::channel();
    enqueue(
        &handle,
        Message::Start(
            EngineContext { epoch: 0, parent: fixture.genesis },
            TestAttributesBuilder::new().with_parent(fixture.genesis).build(),
            tx,
        ),
    );
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Resetting);
    assert_eq!(actor.commands.len(), 1, "canonical checks precede another producer operation");
}

#[tokio::test]
async fn an_origin_ahead_of_observed_l1_is_plausible() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let (mut actor, handle, l1) = ready(fixture.client, fixture.config, fixture.genesis);
    let mut ahead = fixture.genesis;
    ahead.l1_origin = BlockNumHash { number: 2, hash: B256::repeat_byte(2) };
    actor.state.pending_safe_head = ahead;
    actor.publish();
    l1.send_replace(L1State {
        head: Some(BlockInfo { number: 1, ..Default::default() }),
        ..Default::default()
    });
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert!(storage.read().await.l1_reads.iter().all(|id| *id != BlockId::number(2)));
}

#[tokio::test]
async fn a_syncing_build_invalidates_current_readiness_despite_historical_valid() {
    let script = Script::new([ExpectedCall::success(
        "engine_forkchoiceUpdatedV2",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
    )]);
    let config = Arc::new(RollupConfig::default());
    let parent = test_block_info(1);
    let (mut actor, handle, _l1) = ready(Arc::new(script.client(config.clone())), config, parent);
    let (tx, reply) = oneshot::channel();
    enqueue(
        &handle,
        Message::Start(
            EngineContext { epoch: 0, parent },
            TestAttributesBuilder::new().with_parent(parent).build(),
            tx,
        ),
    );
    actor.step().await.unwrap();
    assert!(matches!(reply.await.unwrap(), Err(EngineClientError::Temporary(_))));
    assert_eq!(handle.state().phase, EnginePhase::Syncing);
    assert!(handle.state().el_sync_finished);
    assert_eq!(
        actor.lifecycle,
        Lifecycle::Reset(SyncStartPolicy::Checkpoint { safe: parent, finalized: parent })
    );
    assert_eq!(handle.state().epoch, 1);
    script.assert_finished();
}

#[rstest::rstest]
#[case::valid(false)]
#[case::syncing_then_valid(true)]
#[tokio::test(start_paused = true)]
async fn a_fresh_el_waiting_for_its_first_forkchoice_can_initialize_without_gossip(
    #[case] initially_syncing: bool,
) {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    {
        let mut data = storage.write().await;
        data.el_syncing = true;
        if initially_syncing {
            data.fork_choice_updated_v3_response = Some(ForkchoiceUpdated::new(
                PayloadStatus::from_status(PayloadStatusEnum::Syncing),
            ));
        }
    }
    let (mut actor, handle) = EngineActor::new(
        fixture.client.clone(),
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    if initially_syncing {
        assert_eq!(handle.state().phase, EnginePhase::Syncing);
        assert_eq!(actor.sync_target.unwrap().un_safe, fixture.genesis);
        storage.write().await.fork_choice_updated_v3_response = Some(fcu());
        time::advance(Duration::from_millis(100)).await;
        actor.step().await.unwrap();
    }
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.unsafe_head(), fixture.genesis);
    let data = storage.read().await;
    assert!(!data.forkchoice_calls.is_empty());
    assert!(
        data.forkchoice_calls.iter().all(|call| call.head_block_hash == fixture.genesis.hash())
    );
}

#[tokio::test(start_paused = true)]
async fn preexisting_el_sync_without_gossip_resumes_at_fresh_latest() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let TestBlock { block, reference: target, .. } =
        block_with_origin(fixture.genesis, fixture.config.genesis.l1);
    {
        let mut data = storage.write().await;
        data.el_syncing = true;
        data.syncing_info.highest_block = alloy_primitives::U256::from(1);
    }
    let (mut actor, handle) = EngineActor::new(
        fixture.client.clone(),
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    assert!(storage.read().await.forkchoice_calls.is_empty());
    assert!(actor.sync_target.is_none());
    // The EL can complete its own sync independently of CL gossip.
    fixture.client.set_l2_block(BlockId::latest(), block.clone()).await;
    fixture.client.set_l2_block(target.hash().into(), block).await;
    storage.write().await.el_syncing = false;
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.unsafe_head(), target);
    assert!(
        storage
            .read()
            .await
            .forkchoice_calls
            .iter()
            .all(|fcu| fcu.head_block_hash == target.hash())
    );
}

#[tokio::test(start_paused = true)]
async fn missing_initial_safety_labels_do_not_block_the_payload_that_starts_el_sync() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let TestBlock { block, reference: target, payload } =
        block_with_origin(fixture.genesis, fixture.config.genesis.l1);
    let original = storage.read().await.l2_blocks_by_id.clone();
    {
        let mut data = storage.write().await;
        data.l2_blocks_by_id.clear();
        data.el_syncing = true;
        data.syncing_info.highest_block = alloy_primitives::U256::from(1);
        data.new_payload_v1_response = Some(PayloadStatus::from_status(PayloadStatusEnum::Syncing));
        data.fork_choice_updated_v3_response =
            Some(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)));
    }
    let (mut actor, handle) = EngineActor::new(
        fixture.client.clone(),
        fixture.config,
        Arc::new(NoopBlockSink),
        fixture.l1.subscribe(),
    );
    actor.step().await.unwrap();
    assert_eq!(actor.lifecycle, Lifecycle::Loading);
    handle.try_insert_unsafe(payload).unwrap();
    actor.step().await.unwrap();
    assert_eq!(handle.state().sync_state.unsafe_head(), target);
    assert_eq!(handle.state().sync_state.safe_head(), L2BlockInfo::default());
    assert_eq!(handle.state().sync_state.finalized_head(), L2BlockInfo::default());
    {
        let mut data = storage.write().await;
        data.l2_blocks_by_id = original;
        data.el_syncing = false;
        data.fork_choice_updated_v3_response = Some(fcu());
    }
    fixture.client.set_l2_block(BlockId::latest(), block.clone()).await;
    fixture.client.set_l2_block(target.hash().into(), block).await;
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().sync_state.unsafe_head(), target);
    assert_eq!(handle.state().sync_state.finalized_head(), fixture.genesis);
}
