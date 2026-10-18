use super::*;
use alloy_eips::BlockId;
use alloy_primitives::{B256, Sealed};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    NoopBlockSink,
    test_utils::{
        ExpectedCall, MockEngineClient, Script, TestAttributesBuilder, TestEngineStateBuilder,
        test_block_info,
    },
};
use kona_protocol::{BlockInfo, L1BlockInfoBedrock};
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
) -> (EngineActor<C>, EngineHandle) {
    let (mut actor, handle) =
        EngineActor::with_capacities(client, cfg, Arc::new(NoopBlockSink), 2, 2);
    actor.state =
        TestEngineStateBuilder::new().with_unsafe_head(parent).with_finalized_head(parent).build();
    actor.state.phase = EnginePhase::Ready;
    actor.publish();
    (actor, handle)
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
                .with_l2_block(BlockId::latest(), block)
                .with_l1_block(BlockId::latest(), l1.clone())
                .with_l1_block(config.genesis.l1.number.into(), l1)
                .with_fork_choice_updated_v3_response(fcu())
                .build(),
        );
        Self { client, config, genesis }
    }
}

/// Recovery retains the original acknowledgement, including a reset taking longer than its
/// retry interval. No additional L1 event or producer request is needed to finish it.
#[rstest::rstest]
#[case::l1(true, Duration::ZERO)]
#[case::l2(false, Duration::ZERO)]
#[case::slow_l2(false, Duration::from_secs(4))]
#[tokio::test(start_paused = true)]
async fn reset_recovers_and_completes_original_request(
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
    let (mut actor, handle) =
        EngineActor::new(fixture.client, fixture.config, Arc::new(NoopBlockSink));
    let mut reply = request_reset(&handle);
    actor.step().await.unwrap();
    actor.step().await.unwrap();
    assert!(reply.try_recv().is_err());
    assert_eq!(handle.state().phase, EnginePhase::Resetting);
    let epoch = handle.state().epoch;
    {
        let mut data = storage.write().await;
        data.l1_read_error = None;
        data.l2_read_error = None;
        data.l2_read_delay = read_delay;
    }
    time::advance(Duration::from_millis(100)).await;
    actor.step().await.unwrap();
    reply.await.unwrap().unwrap();
    assert_eq!(handle.state().phase, EnginePhase::Ready);
    assert_eq!(handle.state().epoch, epoch);
    assert_eq!(handle.state().sync_state.safe_head(), fixture.genesis);
    assert_eq!(*handle.queue_length_subscribe().borrow(), 0);
}

#[tokio::test(start_paused = true)]
async fn full_unsafe_admission_does_not_block_reset_and_old_epoch_work_is_purged() {
    let fixture = ResetFixture::new();
    let storage = fixture.client.storage();
    let (mut actor, handle) = ready(fixture.client, fixture.config, fixture.genesis);
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
    let (mut actor, handle) = ready(Arc::new(script.client(config.clone())), config, parent);
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
    let (mut actor, handle) = ready(Arc::new(script.client(config.clone())), config, parent);
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
    let (mut actor, handle) = ready(Arc::new(script.client(config.clone())), config, parent);
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
    let (mut actor, handle) = ready(Arc::new(script.client(config.clone())), config, parent);
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
    let (mut actor, handle) = ready(client, config, parent);
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
