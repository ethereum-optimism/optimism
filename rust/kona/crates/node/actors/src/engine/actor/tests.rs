use super::*;
use crate::{
    engine::{
        NoopBlockSink, SyncStartError,
        test_utils::{TestAttributesBuilder, test_block_info},
    },
    l1::{
        self,
        test_utils::{TestChain, descendants},
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, FixedBytes, b256};
use alloy_rpc_types_engine::{
    ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV1, ExecutionPayloadV3,
    ForkchoiceUpdated, PayloadStatus,
};
use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
use kona_genesis::ChainGenesis;
use kona_protocol::BlockInfo;
use kona_sources::test_utils::{RpcMock, test_engine_api};
use op_alloy_consensus::OpTxType;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
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
    engine: EngineApi,
    l1: l1::Handle,
}

/// Returns an actor's clients, with the scripted L2 transport behind them, and an L1 chain that
/// holds only the default L1 block, the L1 origin of the default L2 block.
fn clients(config: Arc<RollupConfig>) -> (Clients, RpcMock) {
    clients_on_l1(config, &[BlockInfo::default()])
}

/// Returns an actor's clients, with the scripted L2 transport behind them, and an L1 chain of
/// `l1`, oldest first; the oldest is finalized.
fn clients_on_l1(config: Arc<RollupConfig>, l1: &[BlockInfo]) -> (Clients, RpcMock) {
    clients_with_l1(config, TestChain::fixed(l1))
}

/// Returns an actor's clients, with the scripted L2 transport behind them, and `l1`.
fn clients_with_l1(config: Arc<RollupConfig>, l1: l1::Handle) -> (Clients, RpcMock) {
    let (engine, l2) = test_engine_api(config);
    (Clients { engine, l1 }, l2)
}

/// What the engine did with a [`TestSafety`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Driven {
    /// The engine reset safety onto its safe head.
    Reset(L2BlockInfo),
    /// The engine flushed the last block made safe.
    Flush,
}

/// Safety that returns what the test sends it, and records how the engine drove it.
#[derive(Debug)]
struct TestSafety {
    derived: mpsc::UnboundedReceiver<Derived>,
    driven: Arc<Mutex<Vec<Driven>>>,
    asked: Arc<Mutex<Vec<L2BlockInfo>>>,
}

/// Finality that returns the blocks the test sends it, and records the safe heads it was given.
#[derive(Debug)]
struct TestFinality {
    finalized: mpsc::UnboundedReceiver<L2BlockInfo>,
    given: Arc<Mutex<Vec<Safe>>>,
}

/// The test's side of a [`TestSafety`] and a [`TestFinality`].
#[derive(Debug)]
struct Scripts {
    derived: mpsc::UnboundedSender<Derived>,
    driven: Arc<Mutex<Vec<Driven>>>,
    finalized: mpsc::UnboundedSender<L2BlockInfo>,
    given: Arc<Mutex<Vec<Safe>>>,
    asked: Arc<Mutex<Vec<L2BlockInfo>>>,
}

impl Scripts {
    /// Returns the safe heads the engine asked safety to derive on so far, each once in a row.
    fn asked(&self) -> Vec<L2BlockInfo> {
        self.asked.lock().unwrap().clone()
    }

    /// Makes safety return `derived`, once the engine asks.
    fn derive(&self, derived: Derived) {
        self.derived.send(derived).unwrap();
    }

    /// Makes finality return `finalized`, once the engine asks.
    fn finalize(&self, finalized: L2BlockInfo) {
        self.finalized.send(finalized).unwrap();
    }

    /// Returns how the engine drove safety so far.
    fn driven(&self) -> Vec<Driven> {
        self.driven.lock().unwrap().clone()
    }

    /// Returns the safe heads the engine gave finality so far.
    fn given(&self) -> Vec<Safe> {
        self.given.lock().unwrap().clone()
    }
}

/// Returns safety and finality that do nothing until the test sends them something.
fn scripted() -> (TestSafety, TestFinality, Scripts) {
    let (derived_tx, derived) = mpsc::unbounded_channel();
    let (finalized_tx, finalized) = mpsc::unbounded_channel();
    let driven = Arc::new(Mutex::default());
    let given = Arc::new(Mutex::default());
    let asked = Arc::new(Mutex::default());
    (
        TestSafety { derived, driven: driven.clone(), asked: asked.clone() },
        TestFinality { finalized, given: given.clone() },
        Scripts { derived: derived_tx, driven, finalized: finalized_tx, given, asked },
    )
}

#[async_trait::async_trait]
impl Safety for TestSafety {
    type Error = std::convert::Infallible;

    async fn next(&mut self, safe_head: L2BlockInfo) -> Result<Derived, Self::Error> {
        {
            let mut asked = self.asked.lock().unwrap();
            if asked.last() != Some(&safe_head) {
                asked.push(safe_head);
            }
        }
        match self.derived.recv().await {
            Some(derived) => Ok(derived),
            None => std::future::pending().await,
        }
    }

    async fn reset(&mut self, safe: L2BlockInfo) -> Result<(), Self::Error> {
        self.driven.lock().unwrap().push(Driven::Reset(safe));
        Ok(())
    }

    async fn flush(&mut self) -> Result<(), Self::Error> {
        self.driven.lock().unwrap().push(Driven::Flush);
        Ok(())
    }
}

#[async_trait::async_trait]
impl Finality for TestFinality {
    type Error = std::convert::Infallible;

    async fn next(&mut self, safe: Safe) -> Result<L2BlockInfo, Self::Error> {
        {
            let mut given = self.given.lock().unwrap();
            if given.last() != Some(&safe) {
                given.push(safe);
            }
        }
        match self.finalized.recv().await {
            Some(finalized) => Ok(finalized),
            None => std::future::pending().await,
        }
    }
}

/// The actor the tests run.
type TestActor = Actor<TestSafety, TestFinality>;

/// Returns an actor that is not running, for calling its methods directly, with the receiver of
/// its published state.
fn actor(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
    block_sink: Arc<dyn ImportedBlockSink>,
) -> (TestActor, watch::Receiver<EngineState>) {
    let (state_tx, state_rx) = watch::channel(state);
    let (resets_tx, _) = watch::channel(state);
    let (_, messages_rx) = mpsc::channel(1);
    let Clients { engine, l1 } = clients;
    let (safety, finality, _) = scripted();
    let dependencies =
        Dependencies { client: Arc::new(engine), l1, config, block_sink, safety, finality };
    (Actor::new(dependencies, state_tx, resets_tx, messages_rx), state_rx)
}

/// Spawns an actor with the initial engine state and returns its handle and task.
fn spawn(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
) -> (Handle, tokio::task::JoinHandle<Result<(), ActorError>>) {
    let (handle, task, _) = spawn_with(clients, config, state, false);
    (handle, task)
}

/// Spawns an actor with the initial engine state, as after its startup reset, driving safety and
/// finality if `deriving`, and returns its handle, its task, and the test's side of its safety and
/// finality.
fn spawn_with(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
    deriving: bool,
) -> (Handle, tokio::task::JoinHandle<Result<(), ActorError>>, Scripts) {
    let (handle, mut actor, scripts) = unstarted(clients, config, state);
    actor.deriving = if deriving { Deriving::Running } else { Deriving::NotStarted };
    (handle, tokio::spawn(actor.serve()), scripts)
}

/// Spawns an actor that starts from the initial engine state, with its startup reset, and returns
/// its handle, its task, and the test's side of its safety and finality.
fn start(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
) -> (Handle, tokio::task::JoinHandle<Result<(), ActorError>>, Scripts) {
    let (handle, actor, scripts) = unstarted(clients, config, state);
    (handle, tokio::spawn(actor.run()), scripts)
}

/// Returns an actor with the initial engine state that has not started, its handle, and the test's
/// side of its safety and finality.
fn unstarted(
    clients: Clients,
    config: Arc<RollupConfig>,
    state: EngineState,
) -> (Handle, TestActor, Scripts) {
    let (messages, messages_rx) = mpsc::channel(8);
    let (state_tx, state_rx) = watch::channel(state);
    let (resets_tx, resets_rx) = watch::channel(state);
    let (safety, finality, scripts) = scripted();
    let Clients { engine, l1 } = clients;
    let block_sink = Arc::new(NoopBlockSink);
    let dependencies =
        Dependencies { client: Arc::new(engine), l1, config, block_sink, safety, finality };
    let actor = Actor::new(dependencies, state_tx, resets_tx, messages_rx);
    (Handle::new(messages, state_rx, resets_rx), actor, scripts)
}

/// Returns a rollup configuration whose genesis is the default L2 block.
fn config() -> Arc<RollupConfig> {
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = L2Block::default().header.inner.hash_slow();
    Arc::new(config)
}

/// Expects the requests of a reset that finds the default L2 block as every head.
fn expect_reset(l2: &RpcMock, read_delay: Duration) {
    for _ in 0..3 {
        l2.expect_with_delay("eth_getBlockByNumber", L2Block::default(), read_delay);
    }
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
    let (clients, _) = clients(config.clone());
    let builder = Builder::new(EngineState::default());
    let handle = builder.handle();
    let (safety, finality, _) = scripted();
    let task = builder.build(
        Arc::new(clients.engine),
        clients.l1,
        config,
        Arc::new(NoopBlockSink),
        safety,
        finality,
    );
    assert_eq!(*handle.state().borrow(), EngineState::default());
    drop(task);
}

/// An L2 outage during a reset must retain the original request and finish it after recovery,
/// retrying without another request. The L1 actor waits out L1 outages itself.
#[rstest]
#[case(Duration::ZERO)]
#[case(Duration::from_secs(4))]
#[tokio::test(start_paused = true)]
async fn reset_recovers_and_completes_original_request(#[case] read_delay: Duration) {
    let config = config();
    let (clients, l2) = clients(config.clone());
    // The first attempt fails; the recovery is scripted up front.
    l2.expect_error("eth_getBlockByNumber");
    expect_reset(&l2, read_delay);
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
    l2.assert_finished();
}

/// The engine starts from the execution layer's heads, publishes them, and resets at once, which
/// sends a forkchoice even when it settles on the same heads. An execution layer that has finalized
/// no block past genesis is doing its first sync, so safety does not start yet.
#[tokio::test(start_paused = true)]
async fn starts_from_the_execution_layers_heads_and_resets() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    for _ in 0..3 {
        l2.expect("eth_getBlockByNumber", L2Block::default());
    }
    let state = crate::engine::execution_layer_state(&config, &clients.engine).await.unwrap();
    l2.assert_finished();
    let genesis = state.finalized_head();
    assert_eq!(genesis.block_info.hash, config.genesis.l2.hash);
    assert_eq!(
        [state.unsafe_head(), state.local_safe_head(), state.safe_head()],
        [genesis, genesis, genesis]
    );

    expect_reset(&l2, Duration::ZERO);
    let (handle, actor, scripts) = start(clients, config, state);
    let mut resets = handle.resets();
    assert_eq!(*handle.state().borrow(), state);
    assert_eq!(next_reset(&mut resets).await, state);
    settle().await;
    assert_eq!(scripts.driven(), []);
    actor.abort();
    l2.assert_finished();
}

/// An execution layer that has finalized a block past genesis is not doing its first sync, so
/// safety starts after the startup reset, onto the reset's safe head.
#[tokio::test(start_paused = true)]
async fn derives_at_once_when_the_execution_layer_finalized_past_genesis() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    expect_reset(&l2, Duration::ZERO);
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(5)).build();
    let (handle, actor, scripts) = start(clients, config, state);
    let reset = next_reset(&mut handle.resets()).await;
    settle().await;
    assert_eq!(scripts.driven(), [Driven::Reset(reset.safe_head())]);
    actor.abort();
    l2.assert_finished();
}

/// During its first sync, the execution layer has synced once it validates a forkchoice whose
/// unsafe head is past the startup reset's: only then do safety and finality start.
#[rstest]
#[case::still_syncing(PayloadStatusEnum::Syncing, 6, false)]
#[case::validated_at_the_reset(PayloadStatusEnum::Valid, 5, false)]
#[case::validated_past_it(PayloadStatusEnum::Valid, 6, true)]
#[tokio::test]
async fn first_sync_ends_with_a_valid_forkchoice_past_the_reset(
    #[case] status: PayloadStatusEnum,
    #[case] unsafe_head: u64,
    #[case] synced: bool,
) {
    let config = config();
    let (clients, l2) = clients(config.clone());
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(status)),
    );
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(4)).build();
    let (mut actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));
    actor.deriving = Deriving::AfterSyncPast(5);

    let update =
        HeadsUpdate { unsafe_head: Some(test_block_info(unsafe_head)), ..Default::default() };
    actor.synchronize(update).await.unwrap();

    let expected = if synced { Deriving::Synced } else { Deriving::AfterSyncPast(5) };
    assert_eq!(actor.deriving, expected);
    l2.assert_finished();
}

/// An operation that fails temporarily is retried after a backoff without another request.
#[tokio::test(start_paused = true)]
async fn temporary_failure_is_retried_until_the_operation_succeeds() {
    let (payload, config) = genesis_payload();
    let hash = config.genesis.l2.hash;
    let (clients, l2) = clients(config.clone());
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
    l2.assert_finished();
}

/// Build and seal callers that stop waiting for their replies do not stop the engine.
#[tokio::test(start_paused = true)]
async fn abandoned_replies_do_not_stop_the_engine() {
    let mut config = RollupConfig::default();
    config.hardforks.ecotone_time = Some(0);
    let config = Arc::new(config);
    let (clients, l2) = clients(config.clone());
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
    l2.assert_finished();
}

/// An operation that keeps failing temporarily is abandoned after [`TASK_RETRY_TIMEOUT`], and
/// the engine is reset to the execution layer's heads, which it publishes.
#[tokio::test(start_paused = true)]
async fn persistent_temporary_failure_abandons_the_operation_and_resets() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    // Fail every attempt made before the timeout, and the first one after it.
    let mut attempt_at = Duration::ZERO;
    let mut delay = INITIAL_RETRY_DELAY;
    l2.expect_error("engine_newPayloadV1");
    while attempt_at < TASK_RETRY_TIMEOUT {
        attempt_at += delay;
        delay = (delay * 2).min(MAX_RETRY_DELAY);
        l2.expect_error("engine_newPayloadV1");
    }
    expect_reset(&l2, Duration::ZERO);
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
    l2.assert_finished();
}

/// How the execution layer treats the blocks the engine builds from derived attributes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Verdicts {
    /// It accepts the block derived from the attributes.
    Accepted,
    /// It rejects that block, and accepts the deposits-only replacement.
    Replaced,
    /// It rejects both.
    BothRejected,
}

/// Derived attributes are built and sealed into the next safe block, and the L1 block they were
/// derived from becomes the safe head's source.
///
/// After Holocene, a derived payload the execution layer rejects is replaced by a deposits-only
/// block built from the same attributes, without retrying. If the replacement is accepted, the
/// engine flushes safety, with the replacement as the safe head. If it is rejected too, the engine
/// actor stops.
#[rstest]
#[case::accepted(Verdicts::Accepted)]
#[case::replaced(Verdicts::Replaced)]
#[case::both_rejected(Verdicts::BothRejected)]
#[tokio::test(start_paused = true)]
async fn builds_and_seals_derived_attributes(#[case] verdicts: Verdicts) {
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

    let (clients, l2) = clients(config.clone());
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
    let (handle, actor, scripts) = spawn_with(clients, config.clone(), state, true);

    scripts.derive(Derived::Attributes(attributes));

    if verdicts == Verdicts::BothRejected {
        let result = time::timeout(Duration::from_secs(10), actor).await.unwrap().unwrap();
        assert!(matches!(
            result,
            Err(ActorError::Consolidate(ConsolidateTaskError::SealTaskFailed(
                SealTaskError::DepositOnlyPayloadReattemptFailed
            )))
        ));
    } else {
        let mut states = handle.state();
        let made_safe =
            |state: &EngineState| state.safe_head().block_info.hash == config.genesis.l2.hash;
        let state = *time::timeout(Duration::from_secs(10), states.wait_for(made_safe))
            .await
            .unwrap()
            .unwrap();
        assert_eq!(state.safe_head_source(), Some(source));
        // Safety is asked on top of the new safe head next.
        settle().await;
        assert_eq!(scripts.asked().last(), Some(&state.safe_head()));
        // Derivation flushes the channel of a replaced block.
        let flushed = if verdicts == Verdicts::Replaced { vec![Driven::Flush] } else { vec![] };
        assert_eq!(scripts.driven(), flushed);
        actor.abort();
    }
    l2.assert_finished();
}

/// A safe block the engine abandons for a reset, here one its execution layer is missing, resets
/// safety onto the reset's safe head, after the engine publishes the reset.
#[tokio::test(start_paused = true)]
async fn abandoning_a_safe_block_resets_safety() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    // The unsafe block at the safe block's height is missing, which needs a reset.
    l2.expect("eth_getBlockByNumber", Option::<RpcBlock<op_alloy_rpc_types::Transaction>>::None);
    expect_reset(&l2, Duration::ZERO);
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(2))
        .with_safe_head(L2BlockInfo::default())
        .with_finalized_head(L2BlockInfo::default())
        .build();
    let (handle, actor, scripts) = spawn_with(clients, config.clone(), state, true);
    let mut resets = handle.resets();

    scripts.derive(Derived::Safe(Safe { head: test_block_info(1), source: BlockInfo::default() }));

    let reset = next_reset(&mut resets).await;
    // The reset's safe head was not derived from a known L1 block.
    assert_eq!(reset.safe_head_source(), None);
    assert_eq!(reset.safe_head().block_info.hash, config.genesis.l2.hash);
    settle().await;
    assert_eq!(scripts.driven(), [Driven::Reset(reset.safe_head())]);
    actor.abort();
    l2.assert_finished();
}

/// A reset keeps an unsafe head whose L1 origin is after the L1 head, where it may yet be
/// canonical.
#[tokio::test]
async fn reset_keeps_an_unsafe_head_whose_l1_origin_is_after_the_l1_head() {
    let mut config = (*config()).clone();
    config.genesis.l1 = BlockNumHash { number: 5, hash: B256::repeat_byte(5) };
    let config = Arc::new(config);
    let (clients, l2) = clients(config.clone());
    expect_reset(&l2, Duration::ZERO);
    let (mut actor, _) =
        actor(clients, config.clone(), EngineState::default(), Arc::new(NoopBlockSink));

    actor.reset().await.unwrap();

    assert_eq!(actor.state.unsafe_head().block_info.hash, config.genesis.l2.hash);
    l2.assert_finished();
}

/// A reset walks back from an unsafe head whose L1 origin L1 reorged out, though L1 can still
/// serve the origin by hash.
#[tokio::test]
async fn reset_walks_back_from_an_unsafe_head_whose_l1_origin_was_reorged_out() {
    let config = config();
    // The canonical L1 block at the origin's height is another block.
    let other = BlockInfo { hash: B256::repeat_byte(9), ..Default::default() };
    let (clients, l2) = clients_on_l1(config.clone(), &[other]);
    for _ in 0..3 {
        l2.expect("eth_getBlockByNumber", L2Block::default());
    }
    // The genesis block has no parent to walk back to.
    l2.expect("eth_getBlockByHash", Option::<L2Block>::None);
    let (mut actor, _) = actor(clients, config, EngineState::default(), Arc::new(NoopBlockSink));

    assert!(matches!(
        actor.reset().await,
        Err(ActorError::EngineReset(EngineResetError::SyncStart(SyncStartError::BlockNotFound(_))))
    ));
    l2.assert_finished();
}

/// A reset forgets the safe head's source, even when it keeps the safe head.
#[tokio::test]
async fn reset_forgets_the_safe_head_source() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    expect_reset(&l2, Duration::ZERO);
    // The second reset finds the same heads, so it sends no forkchoice update.
    for _ in 0..3 {
        l2.expect("eth_getBlockByNumber", L2Block::default());
    }
    let (mut actor, _) = actor(clients, config, EngineState::default(), Arc::new(NoopBlockSink));

    actor.reset().await.unwrap();
    actor.state =
        actor.state.with_safe_head_source(Some(BlockInfo { number: 1, ..Default::default() }));
    actor.reset().await.unwrap();

    assert_eq!(actor.state.safe_head_source(), None);
    l2.assert_finished();
}

/// Which block of an engine state's heads an L1 block is.
#[derive(Debug, Clone, Copy)]
enum RestsOn {
    /// The unsafe head's L1 origin.
    UnsafeOrigin,
    /// The L1 block the safe head was derived from.
    SafeSource,
    /// The safe head's L1 origin, whose source is unknown.
    SafeOrigin,
}

/// Returns an engine state whose heads rest on `block` as `rests_on` says, and otherwise on the
/// default L1 block.
fn resting_on(block: BlockInfo, rests_on: RestsOn) -> EngineState {
    let on = |origin: BlockInfo| L2BlockInfo { l1_origin: origin.id(), ..Default::default() };
    let state = TestEngineStateBuilder::new()
        .with_safe_head(L2BlockInfo::default())
        .with_finalized_head(L2BlockInfo::default());
    match rests_on {
        RestsOn::UnsafeOrigin => state.with_unsafe_head(on(block)).build(),
        RestsOn::SafeSource => state.build().with_safe_head_source(Some(block)),
        RestsOn::SafeOrigin => state.with_safe_head(on(block)).build(),
    }
}

/// Lets the actor handle what changed.
async fn settle() {
    time::sleep(Duration::from_secs(1)).await;
}

/// Waits for the engine's next reset, and returns its state.
async fn next_reset(resets: &mut watch::Receiver<EngineState>) -> EngineState {
    time::timeout(Duration::from_secs(10), resets.changed()).await.unwrap().unwrap();
    *resets.borrow_and_update()
}

/// From its first reset, the engine resets when L1 reorgs out a block its heads rest on.
#[rstest]
#[case::unsafe_origin(RestsOn::UnsafeOrigin)]
#[case::safe_source(RestsOn::SafeSource)]
#[case::safe_origin(RestsOn::SafeOrigin)]
#[tokio::test(start_paused = true)]
async fn resets_when_l1_reorgs_out_a_block_its_heads_rest_on(#[case] rests_on: RestsOn) {
    let config = config();
    let first = descendants(&BlockInfo::default(), 3, 1);
    let l1 = TestChain::new(&[&[BlockInfo::default()][..], &first].concat());
    let (clients, l2) = clients_with_l1(config.clone(), l1.handle());
    let (handle, actor, _) =
        spawn_with(clients, config.clone(), resting_on(first[1], rests_on), true);
    let mut resets = handle.resets();
    settle().await;
    assert!(!resets.has_changed().unwrap());

    // L1 replaces the block, and the engine resets onto the default L2 block.
    let second = descendants(&first[0], 3, 2);
    expect_reset(&l2, Duration::ZERO);
    l1.set_head(&second);
    let reset = next_reset(&mut resets).await;
    assert_eq!(reset.unsafe_head().block_info.hash, config.genesis.l2.hash);
    actor.abort();
    l2.assert_finished();
}

/// Heads that come to rest on an L1 block L1 already reorged out, such as a safe head derived from
/// data read before the reorg, are checked at once, without waiting for another L1 head.
#[tokio::test(start_paused = true)]
async fn checks_heads_that_change_between_l1_heads() {
    let config = config();
    let first = descendants(&BlockInfo::default(), 3, 1);
    let second = descendants(&first[0], 2, 2);
    let l1 = TestChain::new(&[&[BlockInfo::default()][..], &first[..1], &second].concat());
    let (clients, l2) = clients_with_l1(config.clone(), l1.handle());
    let (handle, actor, scripts) =
        spawn_with(clients, config.clone(), EngineState::default(), true);
    let mut resets = handle.resets();
    settle().await;

    // A safe head derived from the replaced block.
    let safe = L2BlockInfo {
        block_info: BlockInfo { number: 1, ..Default::default() },
        l1_origin: first[1].id(),
        seq_num: 0,
    };
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    expect_reset(&l2, Duration::ZERO);
    scripts.derive(Derived::Safe(Safe { head: safe, source: first[1] }));
    let reset = next_reset(&mut resets).await;
    assert_eq!(reset.safe_head().block_info.hash, config.genesis.l2.hash);
    actor.abort();
    l2.assert_finished();
}

/// The engine gives finality each safe head whose source it knows, none since its last reset
/// until it makes another block safe, and finalizes only a block past its finalized head.
#[tokio::test(start_paused = true)]
async fn drives_finality_with_safe_heads_whose_source_it_knows() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    let (safe_head, finalized) = (test_block_info(6), test_block_info(5));
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(10))
        .with_safe_head(safe_head)
        .with_finalized_head(finalized)
        .build();
    let (handle, actor, scripts) = spawn_with(clients, config, state, true);
    settle().await;
    assert_eq!(scripts.given(), []);

    // Safety reports the safe head the engine has, with its source.
    let safe = Safe { head: safe_head, source: BlockInfo { number: 3, ..Default::default() } };
    scripts.derive(Derived::Safe(safe));
    settle().await;
    assert_eq!(handle.state().borrow().safe_head_source(), Some(safe.source));
    assert_eq!(scripts.given(), [safe]);

    // A block at or before the finalized head is not finalized again.
    scripts.finalize(test_block_info(4));
    settle().await;
    assert_eq!(handle.state().borrow().finalized_head(), finalized);
    actor.abort();
    l2.assert_finished();
}

/// An L1 origin after the L1 head cannot be checked yet. The engine checks it once L1 reaches its
/// height, and resets if L1 has another block there.
#[tokio::test(start_paused = true)]
async fn checks_an_l1_origin_after_the_l1_head_once_l1_reaches_it() {
    let config = config();
    let first = descendants(&BlockInfo::default(), 4, 1);
    let l1 = TestChain::new(&[&[BlockInfo::default()][..], &first[..3]].concat());
    let (clients, l2) = clients_with_l1(config.clone(), l1.handle());
    let state = resting_on(first[3], RestsOn::UnsafeOrigin);
    let (handle, actor, _) = spawn_with(clients, config, state, true);
    let mut resets = handle.resets();
    settle().await;

    // L1 reaches the origin.
    l1.set_head(&first[3..]);
    settle().await;
    assert!(!resets.has_changed().unwrap());

    // L1 replaces the origin.
    let second = descendants(&first[2], 2, 2);
    expect_reset(&l2, Duration::ZERO);
    l1.set_head(&second);
    next_reset(&mut resets).await;
    actor.abort();
    l2.assert_finished();
}

#[tokio::test]
async fn inserted_blocks_are_handed_to_the_block_sink() {
    let (payload, config) = genesis_payload();
    let (clients, l2) = clients(config.clone());
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
    l2.assert_finished();
}

#[rstest]
#[case::invalid(PayloadStatusEnum::Invalid { validation_error: "invalid transaction".into() })]
#[case::accepted(PayloadStatusEnum::Accepted)]
#[tokio::test]
async fn insert_drops_rejected_payloads(#[case] status: PayloadStatusEnum) {
    let (payload, config) = genesis_payload();
    let (clients, l2) = clients(config.clone());
    l2.expect("engine_newPayloadV1", PayloadStatus::from_status(status));
    let sink = Arc::new(RecordingSink::default());
    let (mut actor, _) = actor(clients, config, EngineState::default(), sink.clone());

    actor.handle_insert(payload).await.unwrap();
    assert_eq!(actor.state, EngineState::default());
    assert!(sink.0.lock().unwrap().is_empty(), "a rejected block must not reach the sink");
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

    // The Engine API client selects the forkchoice method version, and is tested for it.
    let mut cfg = RollupConfig::default();
    cfg.hardforks.ecotone_time = Some(0);
    let cfg = Arc::new(cfg);
    let (clients, l2) = clients(cfg.clone());
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
    l2.assert_finished();
}

/// A build on an unsafe head behind the finalized head fails without asking the execution layer.
#[tokio::test]
async fn build_rejects_an_unsafe_head_behind_the_finalized_head() {
    let config = Arc::new(RollupConfig::default());
    let (clients, l2) = clients(config.clone());
    let state = TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(1))
        .with_finalized_head(test_block_info(2))
        .build();
    let (actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));

    let result = simplify(actor.start_build(TestAttributesBuilder::new().build()).await);
    assert_eq!(result, Err(BuildErr::FinalizedAheadOfUnsafe));
    l2.assert_finished();
}

/// Sends a seal request and returns the reply.
async fn seal(
    actor: &mut TestActor,
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
    let (clients, l2) = clients(config.clone());
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
    let (clients, l2) = clients(cfg.clone());
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
    l2.assert_finished();
}

#[tokio::test]
async fn blocks_made_safe_are_handed_to_the_block_sink() {
    // Making an unsafe block safe reads it in full to compare it against the safe head; the next
    // attributes built ask for this same block's system config, so it must reach the sink rather
    // than being dropped.
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
    let (clients, l2) = clients(cfg.clone());
    l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x0", true]), block);
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    let safe_l2 = L2BlockInfo {
        block_info: BlockInfo { hash: block_hash, number: 0, ..Default::default() },
        ..Default::default()
    };
    // The unsafe head is ahead of the safe block, so the safe block is compared rather than
    // applied.
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(1)).build();
    let sink = Arc::new(RecordingSink::default());
    let (mut actor, _) = actor(clients, cfg, state, sink.clone());

    actor.handle_follow_safe_head(safe_l2).await.unwrap();

    assert_eq!(sink.0.lock().unwrap().as_slice(), &[(block_hash, 0)]);
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
    let (clients, l2) = clients(config.clone());
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
    l2.assert_finished();
}

/// A request to finalize a block above the safe head, which a reset can leave behind, is dropped
/// without asking the execution layer, and the engine keeps running.
#[tokio::test]
async fn finalize_drops_a_block_that_is_not_safe() {
    let config = config();
    let (clients, l2) = clients(config.clone());
    let state = TestEngineStateBuilder::new().with_unsafe_head(test_block_info(2)).build();
    let (mut actor, _) = actor(clients, config, state, Arc::new(NoopBlockSink));

    actor.handle_finalize(BlockNumHash { number: 3, ..Default::default() }).await.unwrap();

    assert_eq!(actor.state, state);
    l2.assert_finished();
}

/// The engine finalizes only the block the execution layer's canonical chain holds at the
/// finalized block's height: another, such as one finality decided on before a reset replaced it,
/// is skipped rather than finalized, and the engine keeps running.
///
/// Assert the exact by-number RPC lookup, so a by-hash lookup cannot finalize a block that is not
/// an ancestor of the safe head.
#[tokio::test]
async fn finalize_skips_a_block_the_canonical_chain_does_not_hold() {
    let header = RpcHeader::new(alloy_consensus::Header::default());
    let canonical_hash = header.hash;
    let block: RpcBlock<op_alloy_rpc_types::Transaction> =
        RpcBlock::new(header, BlockTransactions::Full(vec![]));
    // Pin genesis to the canonical block, so its L2BlockInfo needs no L1-info deposit.
    let mut cfg = RollupConfig::default();
    cfg.genesis.l2 = BlockNumHash { number: 0, hash: canonical_hash };
    let cfg = Arc::new(cfg);
    let (clients, l2) = clients(cfg.clone());
    l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x0", true]), block);
    let state = TestEngineStateBuilder::new().build();
    let (mut actor, _) = actor(clients, cfg, state, Arc::new(NoopBlockSink));

    let other = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    actor.handle_finalize(BlockNumHash { number: 0, hash: other }).await.unwrap();

    assert_eq!(actor.state, state);
    l2.assert_finished();
}
