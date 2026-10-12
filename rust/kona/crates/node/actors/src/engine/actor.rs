//! Engine construction and execution.

use super::{
    ActorError, Capacity, Handle,
    handle::{BuildRequest, Message, SealRequest},
};
use crate::EngineDerivationClient;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    BuildSealCoupling, BuildTask, ConsolidateTask, Engine, EngineClient, EngineState, EngineTask,
    EngineTaskError, EngineTaskErrorSeverity, FinalizeTask, ImportedBlockSink, InsertTask,
    SealTask,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{future::Future, sync::Arc};
use tokio::{
    sync::{mpsc, oneshot, watch},
    time::{self, Duration, Instant},
};

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages_rx: mpsc::Receiver<Message>,
    state_tx: watch::Sender<EngineState>,
}

impl Builder {
    /// Creates the builder with the default engine state.
    pub fn new(capacity: Capacity) -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(capacity.get());
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        Self { handle: Handle::new(messages_tx, state_rx), messages_rx, state_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The caller must retain a handle to keep the request channel open.
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor.
    pub fn build<DerivationClient: EngineDerivationClient + 'static>(
        self,
        client: Arc<EngineClient>,
        rollup: Arc<RollupConfig>,
        derivation_client: DerivationClient,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages_rx, state_tx } = self;
        async move {
            let engine = Engine::new(
                Arc::clone(&client),
                Arc::clone(&rollup),
                Arc::clone(&block_sink),
                EngineState::default(),
                state_tx,
            );
            Actor::new(client, rollup, derivation_client, engine, messages_rx, block_sink)
                .run()
                .await
        }
    }
}

/// The delay before the first retry of work that failed temporarily.
const INITIAL_RETRY_DELAY: Duration = Duration::from_millis(100);
/// The longest delay between retries, capped to keep recovery responsive.
const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);
/// How long a task may keep failing temporarily before it is dropped and the engine reset.
const TASK_RETRY_TIMEOUT: Duration = Duration::from_secs(60);

/// Exponentially increasing delays between retries of one piece of work.
#[derive(Debug)]
struct Backoff {
    delay: Duration,
}

impl Default for Backoff {
    fn default() -> Self {
        Self { delay: INITIAL_RETRY_DELAY }
    }
}

impl Backoff {
    /// Waits out the current delay and doubles the next one, up to [`MAX_RETRY_DELAY`].
    async fn wait(&mut self) {
        time::sleep(self.delay).await;
        self.delay = (self.delay * 2).min(MAX_RETRY_DELAY);
    }
}

/// Executes the operations sent to the execution layer's Engine API, one request at a time in the
/// order they arrive.
///
/// Temporary failures are retried in place with backoff, so later requests wait in the handle's
/// bounded queue. A task that keeps failing for [`TASK_RETRY_TIMEOUT`] is dropped and the engine is
/// reset; a reset retries until it succeeds.
#[derive(Debug)]
struct Actor<DerivationClient>
where
    DerivationClient: EngineDerivationClient,
{
    /// The client used to send messages to the [`crate::DerivationActor`].
    derivation_client: DerivationClient,
    /// Whether the EL sync is complete. This should only ever go from false to true.
    el_sync_complete: bool,
    /// The last safe head update sent.
    last_safe_head_sent: L2BlockInfo,

    /// The [`RollupConfig`] used to build tasks.
    rollup: Arc<RollupConfig>,
    /// An [`EngineClient`] used for creating engine tasks.
    client: Arc<EngineClient>,
    /// The engine state and the publisher handle readers subscribe to.
    engine: Engine,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
    /// Where to hand every imported block, so the derivation providers can read it locally
    /// instead of fetching it back from the execution layer.
    block_sink: Arc<dyn ImportedBlockSink>,
}

impl<DerivationClient> Actor<DerivationClient>
where
    DerivationClient: EngineDerivationClient + 'static,
{
    fn new(
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        derivation_client: DerivationClient,
        engine: Engine,
        messages_rx: mpsc::Receiver<Message>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self {
            client,
            derivation_client,
            el_sync_complete: false,
            engine,
            last_safe_head_sent: L2BlockInfo::default(),
            rollup: config,
            messages_rx,
            block_sink,
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            let message = self.messages_rx.recv().await.ok_or(ActorError::ChannelClosed)?;
            let task = match message {
                Message::Build(build_request) => {
                    let BuildRequest { attributes, result_tx } = *build_request;
                    EngineTask::Build(Box::new(BuildTask::new(
                        self.client.clone(),
                        self.rollup.clone(),
                        attributes,
                        Some(result_tx),
                    )))
                }
                Message::Consolidate(safe_signal) => {
                    EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                        self.client.clone(),
                        self.rollup.clone(),
                        safe_signal,
                        Arc::clone(&self.block_sink),
                    )))
                }
                Message::Finalize(finalized_l2_block_id) => {
                    // Finalize the L2 block identified by the provided [`FinalizeBlockId`].
                    EngineTask::Finalize(Box::new(FinalizeTask::new(
                        self.client.clone(),
                        self.rollup.clone(),
                        *finalized_l2_block_id,
                    )))
                }
                Message::InsertUnsafeBlock(envelope) => {
                    EngineTask::Insert(Box::new(InsertTask::new(
                        self.client.clone(),
                        self.rollup.clone(),
                        *envelope,
                        false, /* The payload is not derived in this case. This is an unsafe
                                * block. */
                        Arc::clone(&self.block_sink),
                    )))
                }
                Message::Reset(result_tx) => {
                    warn!(target: "engine", "Received reset request");
                    self.reset(Some(result_tx)).await?;
                    continue;
                }
                Message::Seal(seal_request) => {
                    let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                    EngineTask::Seal(Box::new(SealTask::new(
                        self.client.clone(),
                        self.rollup.clone(),
                        payload_id,
                        attributes,
                        // The payload is not derived in this case.
                        false,
                        BuildSealCoupling::Detached,
                        Some(result_tx),
                        Arc::clone(&self.block_sink),
                    )))
                }
            };
            self.execute(task).await?;
        }
    }

    /// Executes a task, retrying temporary failures with backoff, and acts on its outcome.
    async fn execute(&mut self, task: EngineTask) -> Result<(), ActorError> {
        let deadline = Instant::now() + TASK_RETRY_TIMEOUT;
        let mut backoff = Backoff::default();
        while let Err(err) = self.engine.execute(&task).await {
            match err.severity() {
                EngineTaskErrorSeverity::Temporary if Instant::now() < deadline => {
                    trace!(target: "engine", ?err, "Temporary error executing engine task");
                    backoff.wait().await;
                }
                EngineTaskErrorSeverity::Temporary => {
                    warn!(target: "engine", ?err, "Engine task kept failing; dropping it and resetting");
                    return self.reset(None).await;
                }
                EngineTaskErrorSeverity::Critical => {
                    error!(target: "engine", ?err, "Critical error executing engine task");
                    return Err(err.into());
                }
                EngineTaskErrorSeverity::Reset => {
                    warn!(target: "engine", ?err, "Engine task requires a reset");
                    return self.reset(None).await;
                }
                EngineTaskErrorSeverity::Flush => {
                    // This error is encountered when the payload is marked INVALID by the engine
                    // api. Post-holocene, the payload is replaced by a "deposits-only" block, which
                    // completes the task. At the same time, the channel and any remaining buffered
                    // batches are flushed.
                    warn!(target: "engine", ?err, "Invalid payload, Flushing derivation pipeline.");
                    self.derivation_client.send_signal(Signal::FlushChannel).await.map_err(|err| {
                        error!(target: "engine", ?err, "Failed to send flush signal to the derivation actor.");
                        ActorError::ChannelClosed
                    })?;
                    debug!(target: "engine", "Sent flush signal to derivation actor");
                    break;
                }
            }
        }

        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            if self.engine.state().sync_state.finalized_head() == L2BlockInfo::default() {
                return self.reset(None).await;
            }
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    /// Resets the engine and propagates the reset to the derivation actor, then replies to the
    /// caller, if any.
    ///
    /// Temporary failures are retried until the reset succeeds: losing it would leave derivation
    /// waiting for a reset signal.
    async fn reset(&mut self, result_tx: Option<oneshot::Sender<()>>) -> Result<(), ActorError> {
        let mut backoff = Backoff::default();
        let l2_safe_head = loop {
            match self.engine.reset().await {
                Ok(l2_safe_head) => break l2_safe_head,
                Err(err) if err.is_temporary() => {
                    warn!(target: "engine", ?err, "Reset dependency unavailable; retrying reset");
                    backoff.wait().await;
                }
                Err(err) => return Err(err.into()),
            }
        };

        // Derivation may be awaiting this reply with its inbound queue full of L1 updates.
        // Release the caller before awaiting delivery to that same queue. It transitions
        // to AwaitingSignal before consuming the queued reset signal below.
        if let Some(result_tx) = result_tx {
            let _ = result_tx.send(());
        }

        // Signal the derivation actor to reset.
        let signal = Signal::Reset(ResetSignal { l2_safe_head });
        match self.derivation_client.send_signal(signal).await {
            Ok(_) => info!(target: "engine", "Sent reset signal to derivation actor"),
            Err(err) => {
                error!(target: "engine", ?err, "Failed to send reset signal to the derivation actor");
                return Err(ActorError::ChannelClosed);
            }
        }

        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    async fn mark_el_sync_complete_and_notify_derivation_actor(
        &mut self,
    ) -> Result<(), ActorError> {
        self.el_sync_complete = true;

        self.derivation_client
            .notify_sync_completed(self.engine.state().sync_state.safe_head())
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                ActorError::ChannelClosed
            })?
    }

    /// Attempts to send the [`crate::DerivationActor`] the safe head if updated.
    async fn send_derivation_actor_safe_head_if_updated(&mut self) -> Result<(), ActorError> {
        let engine_safe_head = self.engine.state().sync_state.safe_head();
        if engine_safe_head == self.last_safe_head_sent {
            info!(target: "engine", safe_head = ?engine_safe_head, "Safe head unchanged");
            // This was already sent, so do not send it.
            return Ok(());
        }

        self.derivation_client.send_new_engine_safe_head(engine_safe_head).await.map_err(|e| {
            error!(target: "engine", ?e, "Failed to send new engine safe head");
            ActorError::ChannelClosed
        })?;

        info!(target: "engine", safe_head = ?engine_safe_head, "Attempted L2 Safe Head Update");
        self.last_safe_head_sent = engine_safe_head;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{DerivationActorRequest, QueuedEngineDerivationClient};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };
    use kona_engine::{
        NoopBlockSink,
        test_utils::{RpcMock, TestEngineStateBuilder, test_engine_client},
    };
    use kona_protocol::BlockInfo;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

    #[test]
    fn construction_and_build_do_not_require_a_runtime() {
        let config = Arc::new(RollupConfig::default());
        let (client, _, _) = test_engine_client(config.clone());
        let builder = Builder::new(Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let (derivation_tx, _derivation_rx) = mpsc::channel(1);
        let task = builder.build(
            Arc::new(client),
            config,
            QueuedEngineDerivationClient::new(derivation_tx),
            Arc::new(NoopBlockSink),
        );
        assert_eq!(*handle.state().borrow(), EngineState::default());
        drop(task);
    }

    type L2Block = <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse;

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
        l1.expect(
            "eth_getBlockByHash",
            alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
        );
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
    }

    /// Spawns an actor with the initial engine state and returns its handle and task.
    fn spawn(
        client: EngineClient,
        config: Arc<RollupConfig>,
        derivation_tx: mpsc::Sender<DerivationActorRequest>,
        state: EngineState,
    ) -> (Handle, tokio::task::JoinHandle<Result<(), ActorError>>) {
        let (messages, messages_rx) = mpsc::channel(8);
        let (state_tx, state_rx) = watch::channel(state);
        let client = Arc::new(client);
        let block_sink: Arc<dyn ImportedBlockSink> = Arc::new(NoopBlockSink);
        let engine = Engine::new(
            Arc::clone(&client),
            Arc::clone(&config),
            Arc::clone(&block_sink),
            state,
            state_tx,
        );
        let actor = Actor::new(
            client,
            config,
            QueuedEngineDerivationClient::new(derivation_tx),
            engine,
            messages_rx,
            block_sink,
        );
        (Handle::new(messages, state_rx), tokio::spawn(actor.run()))
    }

    /// Returns an unsafe payload that the default configuration accepts as a block at genesis.
    fn unsafe_payload() -> (OpExecutionPayloadEnvelope, B256) {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        (payload, block.header.hash_slow())
    }

    /// Both L1 and L2 outages during a reset must retain the original request and finish it
    /// after recovery, retrying without another request.
    #[rstest::rstest]
    #[case(true, Duration::ZERO)]
    #[case(false, Duration::ZERO)]
    #[case(false, Duration::from_secs(4))]
    #[tokio::test(start_paused = true)]
    async fn reset_recovers_and_completes_original_request(
        #[case] l1_failure: bool,
        #[case] read_delay: Duration,
    ) {
        let config = config();
        let (client, l1, l2) = test_engine_client(config.clone());
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
        let (derivation_tx, mut derivation_rx) = mpsc::channel(1);
        // Derivation waits for the reset reply and cannot drain its full queue until then.
        derivation_tx
            .send(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::default()))
            .await
            .unwrap();
        let (handle, actor) = spawn(client, config.clone(), derivation_tx, EngineState::default());

        let started = Instant::now();
        time::timeout(Duration::from_secs(20), handle.reset()).await.unwrap().unwrap();
        assert!(started.elapsed() >= INITIAL_RETRY_DELAY, "the failed attempt is retried later");
        // The reset's heads are published before the reply.
        assert_eq!(
            handle.state().borrow().sync_state.unsafe_head().block_info.hash,
            config.genesis.l2.hash
        );
        assert!(matches!(
            derivation_rx.recv().await.unwrap(),
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(_)
        ));
        assert!(matches!(
            derivation_rx.recv().await.unwrap(),
            DerivationActorRequest::ProcessEngineSignalRequest(_)
        ));
        assert!(matches!(
            derivation_rx.recv().await.unwrap(),
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(_)
        ));
        assert!(matches!(
            derivation_rx.recv().await.unwrap(),
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(_)
        ));
        actor.abort();
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A task that fails temporarily is retried after a backoff without another request.
    #[tokio::test(start_paused = true)]
    async fn temporary_failure_is_retried_until_the_task_succeeds() {
        let (payload, hash) = unsafe_payload();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = hash;
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect_error("engine_newPayloadV1");
        l2.expect("engine_newPayloadV1", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
        let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
        // A finalized head keeps the completed EL sync from triggering a reset.
        let finalized = L2BlockInfo {
            block_info: BlockInfo { hash: B256::repeat_byte(1), ..Default::default() },
            ..Default::default()
        };
        let state = TestEngineStateBuilder::new().with_finalized_head(finalized).build();
        let (handle, actor) = spawn(client, config, derivation_tx, state);

        let started = Instant::now();
        handle.insert_unsafe_block(payload).await.unwrap();
        let mut state = handle.state();
        time::timeout(
            Duration::from_secs(10),
            state.wait_for(|state| state.sync_state.unsafe_head().block_info.hash == hash),
        )
        .await
        .unwrap()
        .unwrap();
        assert!(started.elapsed() >= INITIAL_RETRY_DELAY, "the failed attempt is retried later");
        assert!(matches!(
            derivation_rx.recv().await.unwrap(),
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(_)
        ));
        actor.abort();
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A task that keeps failing temporarily is dropped after [`TASK_RETRY_TIMEOUT`], and the
    /// engine is reset so derivation starts over from the execution layer's heads.
    #[tokio::test(start_paused = true)]
    async fn persistent_temporary_failure_drops_the_task_and_resets() {
        let config = config();
        let (client, l1, l2) = test_engine_client(config.clone());
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
        let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
        let (handle, actor) = spawn(client, config, derivation_tx, EngineState::default());

        let started = Instant::now();
        handle.insert_unsafe_block(unsafe_payload().0).await.unwrap();
        let signal = time::timeout(TASK_RETRY_TIMEOUT * 2, derivation_rx.recv()).await.unwrap();
        assert!(matches!(signal, Some(DerivationActorRequest::ProcessEngineSignalRequest(_))));
        assert!(started.elapsed() >= TASK_RETRY_TIMEOUT);
        actor.abort();
        l1.assert_finished();
        l2.assert_finished();
    }
}
