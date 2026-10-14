use super::*;
use crate::EngineClientError;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, FixedBytes, b256};
use alloy_rpc_types_engine::{
    ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV1, ExecutionPayloadV3,
    ForkchoiceUpdated, PayloadStatus,
};
use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
use kona_engine::{
    NoopBlockSink,
    test_utils::{RpcMock, TestAttributesBuilder, test_block_info, test_engine_client},
};
use kona_genesis::ChainGenesis;
use kona_protocol::BlockInfo;
use op_alloy_consensus::OpTxType;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpPayloadAttributes};
use rstest::rstest;
use std::sync::Mutex;
use tokio::sync::oneshot;

/// Builds engine states for tests, defaulting every head to the unsafe head.
struct TestEngineStateBuilder {
    unsafe_head: L2BlockInfo,
    safe_head: Option<L2BlockInfo>,
    finalized_head: Option<L2BlockInfo>,
}

impl TestEngineStateBuilder {
    /// Starts with every head at the default block.
    fn new() -> Self {
        Self { unsafe_head: L2BlockInfo::default(), safe_head: None, finalized_head: None }
    }

    const fn with_unsafe_head(mut self, block: L2BlockInfo) -> Self {
        self.unsafe_head = block;
        self
    }

    const fn with_safe_head(mut self, block: L2BlockInfo) -> Self {
        self.safe_head = Some(block);
        self
    }

    const fn with_finalized_head(mut self, block: L2BlockInfo) -> Self {
        self.finalized_head = Some(block);
        self
    }

    fn build(self) -> EngineState {
        EngineState::default().apply_update(HeadsUpdate {
            unsafe_head: Some(self.unsafe_head),
            local_safe_head: Some(self.unsafe_head),
            safe_head: Some(self.safe_head.unwrap_or(self.unsafe_head)),
            finalized_head: Some(self.finalized_head.unwrap_or(self.unsafe_head)),
        })
    }
}

type L2Block = <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse;

/// The clients an actor talks to the execution layer and L1 with.
struct Clients {
    engine: EngineClient,
    l1: RootProvider,
}

/// Returns an actor's clients, with the scripted L1 and L2 transports behind them.
fn clients(config: Arc<RollupConfig>) -> (Clients, RpcMock, RpcMock) {
    let (engine, l2) = test_engine_client(config);
    let l1 = RpcMock::default();
    (Clients { engine, l1: l1.provider() }, l1, l2)
}

/// Returns an actor that is not running, for calling its methods directly, with the receiver of
/// its published state.
fn actor(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
    block_sink: Arc<dyn ImportedBlockSink>,
) -> (Actor, watch::Receiver<EngineState>) {
    let (state_tx, state_rx) = watch::channel(state);
    let (resets_tx, _) = watch::channel(state);
    let (_, messages_rx) = mpsc::channel(1);
    let Clients { engine, l1 } = clients;
    (
        Actor::new(Arc::new(engine), l1, config, block_sink, state_tx, resets_tx, messages_rx),
        state_rx,
    )
}

/// Spawns an actor with the initial engine state and returns its handle and task.
fn spawn(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
) -> (Handle, tokio::task::JoinHandle<Result<(), ActorError>>) {
    let (messages, messages_rx) = mpsc::channel(8);
    let (state_tx, state_rx) = watch::channel(state);
    let (resets_tx, resets_rx) = watch::channel(state);
    let actor = Actor::new(
        Arc::new(clients.engine),
        clients.l1,
        config,
        Arc::new(NoopBlockSink),
        state_tx,
        resets_tx,
        messages_rx,
    );
    (Handle::new(messages, state_rx, resets_rx), tokio::spawn(actor.run()))
}

/// Returns a rollup configuration whose genesis is the default L2 block.
fn config() -> Arc<RollupConfig> {
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = L2Block::default().header.inner.hash_slow();
    Arc::new(config)
}

/// Expects the requests of a reset that finds the default L2 block as every head.
fn expect_reset(l1: &RpcMock, l2: &RpcMock, read_delay: Duration) {
    for _ in 0..3 {
        l2.expect_with_delay("eth_getBlockByNumber", L2Block::default(), read_delay);
    }
    // The reset checks that the unsafe head's L1 origin, the genesis L1 block, is on L1.
    l1.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([B256::ZERO, false]),
        alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
    );
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
}

/// Returns an unsafe payload, and a configuration whose genesis is the payload's block.
fn genesis_payload() -> (OpExecutionPayloadEnvelope, Arc<RollupConfig>) {
    let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
    ));
    let block: OpBlock = payload.clone().try_into_block().unwrap();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = block.header.hash_slow();
    (payload, Arc::new(config))
}

/// Records the blocks the engine hands over after making them canonical.
#[derive(Debug, Default)]
struct RecordingSink(Mutex<Vec<(B256, u64)>>);

impl ImportedBlockSink for RecordingSink {
    fn block_imported(&self, block: OpBlock, info: L2BlockInfo) {
        self.0.lock().unwrap().push((info.block_info.hash, block.header.number));
    }
}

#[test]
fn construction_and_build_do_not_require_a_runtime() {
    let config = Arc::new(RollupConfig::default());
    let (clients, _, _) = clients(config.clone());
    let builder = Builder::new(Capacity::try_from(1).unwrap());
    let handle = builder.handle();
    let task = builder.build(Arc::new(clients.engine), clients.l1, config, Arc::new(NoopBlockSink));
    assert_eq!(*handle.state().borrow(), EngineState::default());
    drop(task);
}

/// Both L1 and L2 outages during a reset must retain the original request and finish it
/// after recovery, retrying without another request.
#[rstest]
#[case(true, Duration::ZERO)]
#[case(false, Duration::ZERO)]
#[case(false, Duration::from_secs(4))]
#[tokio::test(start_paused = true)]
async fn reset_recovers_and_completes_original_request(
    #[case] l1_failure: bool,
    #[case] read_delay: Duration,
) {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    // The first attempt fails; the recovery is scripted up front.
    if l1_failure {
        for tag in ["finalized", "safe", "latest"] {
            l2.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([tag, true]),
                L2Block::default(),
            );
        }
        l1.expect_error("eth_getBlockByHash");
    } else {
        l2.expect_error("eth_getBlockByNumber");
    }
    expect_reset(&l1, &l2, read_delay);
    let (handle, actor) = spawn(clients, config.clone(), EngineState::default());
    let resets = handle.resets();

    let started = Instant::now();
    let state = time::timeout(Duration::from_secs(20), handle.reset()).await.unwrap().unwrap();
    assert!(started.elapsed() >= INITIAL_RETRY_DELAY, "the failed attempt is retried later");
    assert_eq!(state.safe_head().block_info.hash, config.genesis.l2.hash);
    // The reset and its state are published before the reply.
    assert!(resets.has_changed().unwrap());
    assert_eq!(*resets.borrow(), state);
    assert_eq!(*handle.state().borrow(), state);
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// A reset that asks for a synced execution layer sends the current forkchoice again, and is
/// refused unless the execution layer validates it. A rejected forkchoice lets the reset proceed,
/// because the reset replaces it.
#[tokio::test(start_paused = true)]
async fn reset_if_el_synced_asks_the_el_whether_it_synced() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    let forkchoice_updated = |status| ForkchoiceUpdated::new(PayloadStatus::from_status(status));
    let invalid = PayloadStatusEnum::Invalid { validation_error: "invalid block".into() };
    // The execution layer syncs, and still syncs when the engine asks.
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(PayloadStatusEnum::Syncing));
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(PayloadStatusEnum::Syncing));
    // It finished syncing when the engine asks again, so the reset proceeds.
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(PayloadStatusEnum::Valid));
    expect_reset(&l1, &l2, Duration::ZERO);
    // It validates a later forkchoice, then falls back into syncing.
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(PayloadStatusEnum::Valid));
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(PayloadStatusEnum::Syncing));
    // It rejects the forkchoice, so the reset proceeds.
    l2.expect("engine_forkchoiceUpdatedV3", forkchoice_updated(invalid));
    expect_reset(&l1, &l2, Duration::ZERO);
    let (handle, actor) = spawn(clients, config.clone(), EngineState::default());

    let resets = handle.resets();
    // With no heads, there is no forkchoice to ask with.
    assert!(matches!(handle.reset_if_el_synced().await, Err(EngineClientError::ElSyncing)));
    // A refused reset is not published.
    assert!(!resets.has_changed().unwrap());
    handle.follow_safe_head(test_block_info(1)).await.unwrap();
    assert!(matches!(handle.reset_if_el_synced().await, Err(EngineClientError::ElSyncing)));
    let state = handle.reset_if_el_synced().await.unwrap();
    assert_eq!(state.safe_head().block_info.hash, config.genesis.l2.hash);
    handle.follow_safe_head(test_block_info(1)).await.unwrap();
    assert!(matches!(handle.reset_if_el_synced().await, Err(EngineClientError::ElSyncing)));
    handle.reset_if_el_synced().await.unwrap();
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// An operation that fails temporarily is retried after a backoff without another request.
#[tokio::test(start_paused = true)]
async fn temporary_failure_is_retried_until_the_operation_succeeds() {
    let (payload, config) = genesis_payload();
    let hash = config.genesis.l2.hash;
    let (clients, l1, l2) = clients(config.clone());
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    l2.expect_error("engine_newPayloadV1");
    l2.expect("engine_newPayloadV1", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    let (handle, actor) = spawn(clients, config, EngineState::default());

    let started = Instant::now();
    handle.insert_unsafe_block(payload).await.unwrap();
    let mut state = handle.state();
    time::timeout(
        Duration::from_secs(10),
        state.wait_for(|state| state.unsafe_head().block_info.hash == hash),
    )
    .await
    .unwrap()
    .unwrap();
    assert!(started.elapsed() >= INITIAL_RETRY_DELAY, "the failed attempt is retried later");
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// Build and seal callers that stop waiting for their replies do not stop the engine.
#[tokio::test(start_paused = true)]
async fn abandoned_replies_do_not_stop_the_engine() {
    let mut config = RollupConfig::default();
    config.hardforks.ecotone_time = Some(0);
    let config = Arc::new(config);
    let (clients, l1, l2) = clients(config.clone());
    let started = || {
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid))
            .with_payload_id(PayloadId::new([1; 8]))
    };
    l2.expect_with_delay("engine_forkchoiceUpdatedV3", started(), Duration::from_secs(1));
    l2.expect("engine_forkchoiceUpdatedV3", started());
    let (handle, actor) = spawn(clients, config, EngineState::default());
    let wait = Duration::from_millis(100);

    // Abandon a build while the engine performs it, and a seal queued behind it. The seal's
    // parent is not the unsafe head, so it fails without a request to the execution layer.
    let build = handle.start_build(TestAttributesBuilder::new().build());
    assert!(time::timeout(wait, build).await.is_err());
    let stale = TestAttributesBuilder::new().with_parent(test_block_info(1)).build();
    assert!(time::timeout(wait, handle.seal(PayloadId::new([1; 8]), stale)).await.is_err());

    // The engine still answers later requests.
    let payload_id = handle.start_build(TestAttributesBuilder::new().build()).await.unwrap();
    assert_eq!(payload_id, PayloadId::new([1; 8]));
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// An operation that keeps failing temporarily is abandoned after [`TASK_RETRY_TIMEOUT`], and
/// the engine is reset to the execution layer's heads, which it publishes.
#[tokio::test(start_paused = true)]
async fn persistent_temporary_failure_abandons_the_operation_and_resets() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    // Fail every attempt made before the timeout, and the first one after it.
    let mut attempt_at = Duration::ZERO;
    let mut delay = INITIAL_RETRY_DELAY;
    l2.expect_error("engine_newPayloadV1");
    while attempt_at < TASK_RETRY_TIMEOUT {
        attempt_at += delay;
        delay = (delay * 2).min(MAX_RETRY_DELAY);
        l2.expect_error("engine_newPayloadV1");
    }
    expect_reset(&l1, &l2, Duration::ZERO);
    let (handle, actor) = spawn(clients, config.clone(), EngineState::default());

    let started = Instant::now();
    handle.insert_unsafe_block(genesis_payload().0).await.unwrap();
    let mut state = handle.state();
    time::timeout(
        TASK_RETRY_TIMEOUT * 2,
        state.wait_for(|state| state.safe_head().block_info.hash == config.genesis.l2.hash),
    )
    .await
    .unwrap()
    .unwrap();
    assert!(started.elapsed() >= TASK_RETRY_TIMEOUT);
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// How the execution layer treats the blocks a consolidation builds.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Verdicts {
    /// It accepts the block derived from the attributes.
    Accepted,
    /// It rejects that block, and accepts the deposits-only replacement.
    Replaced,
    /// It rejects both.
    BothRejected,
}

/// A consolidation that cannot make an unsafe block safe builds and seals the derived attributes,
/// and the L1 block they were derived from becomes the safe head's source.
///
/// After Holocene, a derived payload the execution layer rejects is replaced by a deposits-only
/// block built from the same attributes, without retrying the consolidation. If the replacement is
/// accepted, the engine replies that derivation must flush its channel, with the replacement as
/// the safe head. If it is rejected too, the engine actor stops.
#[rstest]
#[case::accepted(Verdicts::Accepted)]
#[case::replaced(Verdicts::Replaced)]
#[case::both_rejected(Verdicts::BothRejected)]
#[tokio::test(start_paused = true)]
async fn consolidation_builds_and_seals_derived_attributes(#[case] verdicts: Verdicts) {
    let root = B256::repeat_byte(0x42);
    let payload = OpExecutionPayloadEnvelopeV3 {
        execution_payload: ExecutionPayloadV3::from_block_slow(&alloy_consensus::Block::<
            op_alloy_consensus::OpTxEnvelope,
        >::default()),
        block_value: Default::default(),
        blobs_bundle: Default::default(),
        should_override_builder: false,
        parent_beacon_block_root: root,
    };
    let block: OpBlock = OpExecutionPayloadEnvelope::V3 {
        payload: payload.execution_payload.clone(),
        parent_beacon_block_root: root,
    }
    .try_into_block()
    .unwrap();
    // Pin genesis to the payload's block, so its block info needs no L1-info deposit.
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = block.header.hash_slow();
    config.hardforks.holocene_time = Some(0);
    let config = Arc::new(config);

    // A user transaction, which the deposits-only replacement drops.
    let mut attributes = TestAttributesBuilder::new()
        .with_transactions(vec![vec![OpTxType::Legacy as u8, 0xbb].into()])
        .build();
    let source = BlockInfo { number: 7, ..Default::default() };
    attributes.derived_from = Some(source);
    // The attributes extend the safe head, which is also the unsafe head.
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(attributes.parent)
        .with_finalized_head(L2BlockInfo::default())
        .build();
    let forkchoice = state.create_forkchoice_state();

    let (clients, l1, l2) = clients(config.clone());
    let started = || fcu_for_payload(Some(PayloadId::new([1; 8])), PayloadStatusEnum::Valid);
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    let invalid = || {
        PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "invalid transaction".into(),
        })
    };
    // The derived payload is built and sealed.
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([forkchoice, attributes.attributes]),
        started(),
    );
    l2.expect("engine_getPayloadV3", payload.clone());
    if verdicts == Verdicts::Accepted {
        l2.expect("engine_newPayloadV3", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    } else {
        l2.expect("engine_newPayloadV3", invalid());
        // The deposits-only replacement is built and sealed.
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, attributes.as_deposits_only().attributes]),
            started(),
        );
        l2.expect("engine_getPayloadV3", payload);
        if verdicts == Verdicts::Replaced {
            l2.expect("engine_newPayloadV3", valid());
            l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        } else {
            l2.expect("engine_newPayloadV3", invalid());
        }
    }
    let (handle, actor) = spawn(clients, config.clone(), state);

    let consolidation = handle.consolidate(attributes).await;

    if verdicts == Verdicts::BothRejected {
        assert!(consolidation.is_err(), "the engine stops without replying");
        let result = time::timeout(Duration::from_secs(10), actor).await.unwrap().unwrap();
        assert!(matches!(
            result,
            Err(ActorError::Consolidate(ConsolidateTaskError::SealTaskFailed(
                SealTaskError::DepositOnlyPayloadReattemptFailed
            )))
        ));
    } else {
        let Consolidation { state, flush } = consolidation.unwrap();
        assert_eq!(flush, verdicts == Verdicts::Replaced);
        assert_eq!(state.safe_head().block_info.hash, config.genesis.l2.hash);
        assert_eq!(state.safe_head_source(), Some(source));
        actor.abort();
    }
    l1.assert_finished();
    l2.assert_finished();
}

/// A consolidation the engine abandons for a reset publishes the reset, then replies with the
/// state after it.
#[tokio::test(start_paused = true)]
async fn consolidation_that_resets_replies_with_the_reset_state() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    // The unsafe block at the attributes' height is missing, which needs a reset.
    l2.expect("eth_getBlockByNumber", Option::<RpcBlock<op_alloy_rpc_types::Transaction>>::None);
    expect_reset(&l1, &l2, Duration::ZERO);
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(2))
        .with_safe_head(L2BlockInfo::default())
        .with_finalized_head(L2BlockInfo::default())
        .build();
    let (handle, actor) = spawn(clients, config.clone(), state);
    let resets = handle.resets();

    let attributes = TestAttributesBuilder::new().with_parent(L2BlockInfo::default()).build();
    let Consolidation { state, flush } = handle.consolidate(attributes).await.unwrap();

    assert!(!flush);
    // The reset is published before the reply.
    assert!(resets.has_changed().unwrap());
    assert_eq!(*resets.borrow(), state);
    // The reset's safe head was not derived from a known L1 block.
    assert_eq!(state.safe_head_source(), None);
    assert_eq!(state.safe_head().block_info.hash, config.genesis.l2.hash);
    assert_eq!(*handle.state().borrow(), state);
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// Attributes derived before a reset do not extend the reset's safe head, so the engine skips
/// them and replies with the reset's state.
#[tokio::test(start_paused = true)]
async fn consolidation_skips_attributes_derived_before_a_reset() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    expect_reset(&l1, &l2, Duration::ZERO);
    let safe_head = test_block_info(1);
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(2))
        .with_safe_head(safe_head)
        .with_finalized_head(L2BlockInfo::default())
        .build();
    let (handle, actor) = spawn(clients, config, state);

    // The reset is queued before the consolidation.
    let attributes = TestAttributesBuilder::new().with_parent(safe_head).build();
    let (reset, consolidation) = tokio::join!(handle.reset(), handle.consolidate(attributes));

    let reset = reset.unwrap();
    assert_eq!(consolidation.unwrap(), Consolidation { state: reset, flush: false });
    actor.abort();
    l1.assert_finished();
    l2.assert_finished();
}

/// A reset forgets the safe head's source, even when it keeps the safe head.
#[tokio::test]
async fn reset_forgets_the_safe_head_source() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    expect_reset(&l1, &l2, Duration::ZERO);
    // The second reset finds the same heads, so it sends no forkchoice update.
    for _ in 0..3 {
        l2.expect("eth_getBlockByNumber", L2Block::default());
    }
    l1.expect(
        "eth_getBlockByHash",
        alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
    );
    let (mut actor, _) = actor(clients, config, EngineState::default(), Arc::new(NoopBlockSink));

    actor.reset().await.unwrap();
    actor.state =
        actor.state.with_safe_head_source(Some(BlockInfo { number: 1, ..Default::default() }));
    actor.reset().await.unwrap();

    assert_eq!(actor.state.safe_head_source(), None);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn inserted_blocks_are_handed_to_the_block_sink() {
    let (payload, config) = genesis_payload();
    let (clients, l1, l2) = clients(config.clone());
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    l2.expect("engine_newPayloadV1", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    let sink = Arc::new(RecordingSink::default());
    let (mut actor, _) = actor(clients, config.clone(), EngineState::default(), sink.clone());

    actor.handle_insert(payload).await.unwrap();
    assert_eq!(
        sink.0.lock().unwrap().as_slice(),
        &[(config.genesis.l2.hash, 0)],
        "a successfully imported block must reach the sink"
    );
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest]
#[case::invalid(PayloadStatusEnum::Invalid { validation_error: "invalid transaction".into() })]
#[case::accepted(PayloadStatusEnum::Accepted)]
#[tokio::test]
async fn insert_drops_rejected_payloads(#[case] status: PayloadStatusEnum) {
    let (payload, config) = genesis_payload();
    let (clients, l1, l2) = clients(config.clone());
    l2.expect("engine_newPayloadV1", PayloadStatus::from_status(status));
    let sink = Arc::new(RecordingSink::default());
    let (mut actor, _) = actor(clients, config, EngineState::default(), sink.clone());

    actor.handle_insert(payload).await.unwrap();
    assert_eq!(actor.state, EngineState::default());
    assert!(sink.0.lock().unwrap().is_empty(), "a rejected block must not reach the sink");
    l1.assert_finished();
    l2.assert_finished();
}

fn fcu_for_payload(payload_id: Option<PayloadId>, status: PayloadStatusEnum) -> ForkchoiceUpdated {
    ForkchoiceUpdated {
        payload_status: PayloadStatus { status, latest_valid_hash: Some(FixedBytes([2u8; 32])) },
        payload_id,
    }
}

#[derive(Debug, PartialEq, Eq)]
enum BuildErr {
    AttributesInsertionFailed,
    EngineSyncing,
    FinalizedAheadOfUnsafe,
    InvalidPayload,
    MissingPayloadId,
    Unexpected,
}

// Wraps real errors, ignoring details so we can easily match on results.
fn simplify(result: Result<PayloadId, BuildTaskError>) -> Result<PayloadId, BuildErr> {
    match result {
        Ok(payload_id) => Ok(payload_id),
        Err(BuildTaskError::EngineBuildError(e)) => match e {
            EngineBuildError::AttributesInsertionFailed(_) => {
                Err(BuildErr::AttributesInsertionFailed)
            }
            EngineBuildError::EngineSyncing => Err(BuildErr::EngineSyncing),
            EngineBuildError::FinalizedAheadOfUnsafe(_, _) => Err(BuildErr::FinalizedAheadOfUnsafe),
            EngineBuildError::InvalidPayload(_) => Err(BuildErr::InvalidPayload),
            EngineBuildError::MissingPayloadId => Err(BuildErr::MissingPayloadId),
            EngineBuildError::UnexpectedPayloadStatus(_) => Err(BuildErr::Unexpected),
        },
    }
}

#[rstest]
#[case::success(Some(PayloadStatusEnum::Valid), true, None)]
#[case::missing_id(Some(PayloadStatusEnum::Valid), false, Some(BuildErr::MissingPayloadId))]
#[case::fcu_fail(None, false, Some(BuildErr::AttributesInsertionFailed))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Invalid{validation_error: String::new()}), false, Some(BuildErr::InvalidPayload))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Syncing), false, Some(BuildErr::EngineSyncing))]
#[case::fcu_status_fail(Some(PayloadStatusEnum::Accepted), false, Some(BuildErr::Unexpected))]
#[tokio::test]
async fn build_variants(
    // NB: none = failure
    #[case] fcu_status: Option<PayloadStatusEnum>,
    // NB: none = failure
    #[case] payload_id_present: bool,
    // NB: none = success
    #[case] expected_err: Option<BuildErr>,
) {
    let payload_id = payload_id_present.then(|| PayloadId::new([1u8; 8]));

    let parent_block = test_block_info(0);
    let unsafe_block = test_block_info(1);
    let attributes_timestamp = unsafe_block.block_info.timestamp;

    // The engine client selects the forkchoice method version, and is tested for it.
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let cfg = Arc::new(cfg);
    let (clients, l1, l2) = clients(cfg.clone());
    if let Some(status) = fcu_status {
        l2.expect("engine_forkchoiceUpdatedV3", fcu_for_payload(payload_id, status));
    } else {
        l2.expect_error("engine_forkchoiceUpdatedV3");
    }

    let attributes = TestAttributesBuilder::new()
        .with_parent(parent_block)
        .with_timestamp(attributes_timestamp)
        .build();

    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(unsafe_block)
        .with_safe_head(parent_block)
        .with_finalized_head(parent_block)
        .build();

    let (actor, _) = actor(clients, cfg, state, Arc::new(NoopBlockSink));
    let result = simplify(actor.start_build(attributes).await);

    if expected_err.is_some() {
        assert_eq!(expected_err, result.err());
    } else {
        assert!(result.is_ok());
        assert!(payload_id.is_some(), "Payload id none when it should be some.");
        assert_eq!(result.unwrap(), payload_id.unwrap(), "Should return the correct payload ID");
    }
    l1.assert_finished();
    l2.assert_finished();
}

/// A build on an unsafe head behind the finalized head fails without asking the execution layer.
#[tokio::test]
async fn build_rejects_an_unsafe_head_behind_the_finalized_head() {
    let config = Arc::new(RollupConfig::default());
    let (clients, l1, l2) = clients(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(1))
        .with_finalized_head(test_block_info(2))
        .build();
    let (actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));

    let result = simplify(actor.start_build(TestAttributesBuilder::new().build()).await);
    assert_eq!(result, Err(BuildErr::FinalizedAheadOfUnsafe));
    l1.assert_finished();
    l2.assert_finished();
}

/// Sends a seal request and returns the reply.
async fn seal(
    actor: &mut Actor,
    payload_id: PayloadId,
    attributes: OpAttributesWithParent,
) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
    let (result_tx, result_rx) = oneshot::channel();
    actor.handle_seal(SealRequest { payload_id, attributes, result_tx }).await;
    result_rx.await.unwrap()
}

/// A sealed block is published before the reply, because the sequencer builds the next block on
/// the unsafe head as soon as it gets the reply.
#[tokio::test]
async fn seal_publishes_the_new_unsafe_head_before_replying() {
    let (payload, config) = genesis_payload();
    let OpExecutionPayloadEnvelope::V1(payload_v1) = payload else {
        unreachable!("the genesis payload is a V1 payload")
    };
    let (clients, l1, l2) = clients(config.clone());
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    l2.expect(
        "engine_getPayloadV2",
        ExecutionPayloadEnvelopeV2 {
            execution_payload: ExecutionPayloadFieldV2::V1(payload_v1),
            block_value: Default::default(),
        },
    );
    l2.expect("engine_newPayloadV1", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    let attributes = TestAttributesBuilder::new().build();
    let state = TestEngineStateBuilder::new().with_unsafe_head(attributes.parent).build();
    let (mut actor, state_rx) = actor(clients, config.clone(), state, Arc::new(NoopBlockSink));

    let (result_tx, mut result_rx) = oneshot::channel();
    actor
        .handle_seal(SealRequest { payload_id: PayloadId::new([1; 8]), attributes, result_tx })
        .await;
    result_rx.try_recv().unwrap().unwrap();
    // The seal handler published the new unsafe head itself, before replying.
    assert!(state_rx.has_changed().unwrap());
    assert_eq!(state_rx.borrow().unsafe_head().block_info.hash, config.genesis.l2.hash);
    l1.assert_finished();
    l2.assert_finished();
}

/// How a payload is sealed.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Sealing {
    /// A seal request, separate from its build, so another request may have moved the unsafe
    /// head in between.
    Request,
    /// Building and sealing in sequence, as a derivation-driven reorg does: the payload's parent
    /// legitimately differs from the unsafe head.
    Reorg,
}

/// The two paths the unsafe-head check can steer a seal into: aborting the seal as stale, or
/// proceeding to the payload fetch — whose injected RPC failure surfaces as
/// [`SealTaskError::GetPayloadFailed`].
#[derive(Debug, PartialEq, Eq)]
enum SealOutcome {
    AbortedAsStale,
    ProceededToSeal,
}

fn classify(err: &SealTaskError) -> SealOutcome {
    match err {
        SealTaskError::UnsafeHeadChangedSinceBuild => SealOutcome::AbortedAsStale,
        SealTaskError::GetPayloadFailed(_) => SealOutcome::ProceededToSeal,
        other => panic!("unexpected seal error: {other:?}"),
    }
}

#[rstest]
#[case::request_with_moved_unsafe_head(Sealing::Request, false, SealOutcome::AbortedAsStale)]
#[case::request_with_current_unsafe_head(Sealing::Request, true, SealOutcome::ProceededToSeal)]
#[case::reorg_with_moved_unsafe_head(Sealing::Reorg, false, SealOutcome::ProceededToSeal)]
#[case::reorg_with_current_unsafe_head(Sealing::Reorg, true, SealOutcome::ProceededToSeal)]
#[tokio::test]
async fn unsafe_head_check_variants(
    #[case] sealing: Sealing,
    #[case] unsafe_head_at_parent: bool,
    #[case] expected: SealOutcome,
) {
    let parent_block = test_block_info(10);
    let unsafe_head = if unsafe_head_at_parent { parent_block } else { test_block_info(15) };

    let attributes = TestAttributesBuilder::new().with_parent(parent_block).build();
    let state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();

    let cfg = Arc::new(RollupConfig::default());
    let (clients, l1, l2) = clients(cfg.clone());
    let payload_id = PayloadId::new([1u8; 8]);
    if sealing == Sealing::Reorg {
        l2.expect(
            "engine_forkchoiceUpdatedV2",
            fcu_for_payload(Some(payload_id), PayloadStatusEnum::Valid),
        );
    }
    if unsafe_head_at_parent || sealing == Sealing::Reorg {
        l2.expect_error("engine_getPayloadV2");
    }
    let (mut actor, _) = actor(clients, cfg, state, Arc::new(NoopBlockSink));

    let result = match sealing {
        Sealing::Request => seal(&mut actor, payload_id, attributes).await,
        Sealing::Reorg => match actor.build_and_seal(attributes, true).await {
            Err(ConsolidateTaskError::SealTaskFailed(err)) => Err(err),
            other => panic!("unexpected build and seal result: {other:?}"),
        },
    };
    assert_eq!(classify(&result.expect_err("seal should fail against the mock")), expected);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn consolidated_blocks_are_handed_to_the_block_sink() {
    // Consolidation reads the unsafe block in full to compare it against the safe head; the next
    // attributes built ask for this same block's system config, so it must reach the sink rather
    // than being dropped. Derived attributes and delegated safe heads share this step.
    let header = RpcHeader::new(alloy_consensus::Header::default());
    let block_hash = header.hash;
    let block: RpcBlock<op_alloy_rpc_types::Transaction> =
        RpcBlock::new(header, BlockTransactions::Full(vec![]));

    // Pin genesis to this block so its L2BlockInfo needs no L1-info deposit.
    let cfg = Arc::new(RollupConfig {
        genesis: ChainGenesis {
            l2: BlockNumHash { hash: block_hash, number: 0 },
            ..Default::default()
        },
        ..Default::default()
    });
    let (clients, l1, l2) = clients(cfg.clone());
    l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x0", true]), block);
    let safe_l2 = L2BlockInfo {
        block_info: BlockInfo { hash: block_hash, number: 0, ..Default::default() },
        ..Default::default()
    };
    // The unsafe head is ahead of the safe block, so it is consolidated rather than applied.
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(1)).build();
    let sink = Arc::new(RecordingSink::default());
    let (mut actor, _) = actor(clients, cfg, state, sink.clone());

    actor.handle_follow_safe_head(safe_l2).await.unwrap();

    assert_eq!(sink.0.lock().unwrap().as_slice(), &[(block_hash, 0)]);
    l1.assert_finished();
    l2.assert_finished();
}

/// A consolidation whose attributes match the unsafe block at their height makes that block safe,
/// with the attributes' L1 block as its source. Only the last attributes of a span send a
/// forkchoice update.
#[rstest]
#[case::last_in_span(true)]
#[case::within_a_span(false)]
#[tokio::test]
async fn consolidation_makes_a_matching_unsafe_block_safe(#[case] is_last_in_span: bool) {
    let header = RpcHeader::new(alloy_consensus::Header::default());
    let block_hash = header.hash;
    let block: RpcBlock<op_alloy_rpc_types::Transaction> =
        RpcBlock::new(header, BlockTransactions::Full(vec![]));
    // Pin genesis to this block so its L2BlockInfo needs no L1-info deposit.
    let cfg = Arc::new(RollupConfig {
        genesis: ChainGenesis {
            l2: BlockNumHash { hash: block_hash, number: 0 },
            ..Default::default()
        },
        ..Default::default()
    });
    let (clients, l1, l2) = clients(cfg.clone());
    l2.expect("eth_getBlockByNumber", block);
    if is_last_in_span {
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
    }
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(1))
        .with_safe_head(L2BlockInfo::default())
        .with_finalized_head(L2BlockInfo::default())
        .build();
    let (mut actor, _) = actor(clients, cfg, state, Arc::new(NoopBlockSink));
    let source = BlockInfo { number: 7, ..Default::default() };
    // Attributes that match the default block.
    let attributes = OpAttributesWithParent::new(
        OpPayloadAttributes { gas_limit: Some(0), ..Default::default() },
        L2BlockInfo::default(),
        Some(source),
        is_last_in_span,
    );

    actor.handle_consolidate(attributes).await.unwrap();

    assert_eq!(actor.state.safe_head().block_info.hash, block_hash);
    assert_eq!(actor.state.safe_head_source(), Some(source));
    l1.assert_finished();
    l2.assert_finished();
}

/// A delegated safe head that the unsafe chain does not hold becomes the unsafe and safe head, with
/// a forkchoice update.
#[rstest]
#[case::not_behind_the_unsafe_head(false)]
#[case::mismatching_the_unsafe_block(true)]
#[tokio::test]
async fn delegated_safe_head_is_applied_when_the_unsafe_chain_lacks_it(
    #[case] unsafe_head_ahead: bool,
) {
    let config = Arc::new(RollupConfig::default());
    let (clients, l1, l2) = clients(config.clone());
    let safe_head = test_block_info(1);
    let unsafe_head = if unsafe_head_ahead { test_block_info(2) } else { safe_head };
    if unsafe_head_ahead {
        // The unsafe block at the safe head's height is a different block.
        let block: RpcBlock<op_alloy_rpc_types::Transaction> = RpcBlock::new(
            RpcHeader::new(alloy_consensus::Header::default()),
            BlockTransactions::Full(vec![]),
        );
        assert_ne!(block.header.hash, safe_head.block_info.hash);
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x1", true]), block);
    }
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    let genesis = test_block_info(0);
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(unsafe_head)
        .with_safe_head(genesis)
        .with_finalized_head(genesis)
        .build();
    let (mut actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));

    actor.handle_follow_safe_head(safe_head).await.unwrap();

    assert_eq!(actor.state.unsafe_head(), safe_head);
    assert_eq!(actor.state.safe_head(), safe_head);
    l1.assert_finished();
    l2.assert_finished();
}

/// A request to finalize a block above the safe head, which a reset can leave behind, is dropped
/// without asking the execution layer, and the engine keeps running.
#[tokio::test]
async fn finalize_drops_a_block_that_is_not_safe() {
    let config = config();
    let (clients, l1, l2) = clients(config.clone());
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(2)).build();
    let (mut actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));

    actor.handle_finalize(FinalizeBlockId::ByNumber(3)).await.unwrap();

    assert_eq!(actor.state, state);
    l1.assert_finished();
    l2.assert_finished();
}

/// When the engine receives a `ByHash` finalize request for a block hash it doesn't have,
/// finalization must fail with [`FinalizeTaskError::BlockNotFound`] rather than silently
/// finalize whatever it happens to have at the same height.
///
/// Assert the exact by-hash RPC lookup, so a by-number lookup cannot silently finalize a
/// different canonical block at the same height.
#[tokio::test]
async fn finalize_by_hash_errors_when_engine_lacks_hash() {
    const N: u64 = 10;
    let hash_a = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let hash_b = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");

    let cfg = Arc::new(RollupConfig::default());
    let (clients, l1, l2) = clients(cfg.clone());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([hash_b, true]),
        Option::<alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>>::None,
    );

    // Place the safe head at N so the sanity check (`safe_head.number >= block_id.number`) passes
    // and `execute()` reaches the lookup we care about.
    let safe_head = L2BlockInfo {
        block_info: BlockInfo {
            number: N,
            hash: hash_a,
            parent_hash: Default::default(),
            timestamp: N * 2,
        },
        l1_origin: BlockNumHash::default(),
        seq_num: 0,
    };
    let state =
        TestEngineStateBuilder::new().with_unsafe_head(safe_head).with_safe_head(safe_head).build();
    let (mut actor, _) = actor(clients, cfg, state, Arc::new(NoopBlockSink));

    let result = actor
        .handle_finalize(FinalizeBlockId::ByHash(BlockNumHash { number: N, hash: hash_b }))
        .await;

    assert!(
        matches!(result, Err(FinalizeTaskError::BlockNotFound(n)) if n == N),
        "expected BlockNotFound({N}) — got {result:?}. The by-hash lookup must fail loudly when \
         the engine lacks the requested hash; instead, finalization either succeeded (finalizing the \
         wrong block) or surfaced a different error."
    );
    l1.assert_finished();
    l2.assert_finished();
}
