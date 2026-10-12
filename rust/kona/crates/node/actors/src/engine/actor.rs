//! Engine construction and execution.

use super::{
    ActorError, Capacity, Handle,
    handle::{BuildRequest, Message, SealRequest},
};
use crate::EngineDerivationClient;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    ConsolidateInput, Engine, EngineClient, EngineState, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeBlockId, ImportedBlockSink,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{future::Future, pin::Pin, sync::Arc};
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
            let engine = Engine::new(client, rollup, block_sink, EngineState::default(), state_tx);
            Actor::new(derivation_client, engine, messages_rx).run().await
        }
    }
}

/// The delay before the first retry of work that failed temporarily.
const INITIAL_RETRY_DELAY: Duration = Duration::from_millis(100);
/// The longest delay between retries, capped to keep recovery responsive.
const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);
/// How long an operation may keep failing temporarily before it is abandoned and the engine reset.
const TASK_RETRY_TIMEOUT: Duration = Duration::from_secs(60);

/// An engine operation in progress, boxed so the actor's future stays `Send`.
type Operation<'a, E> = Pin<Box<dyn Future<Output = Result<(), E>> + Send + 'a>>;

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
/// Builds and seals reply to their caller, which decides whether to retry. Other operations retry
/// temporary failures in place with backoff, so later requests wait in the handle's bounded queue.
/// An operation that keeps failing for [`TASK_RETRY_TIMEOUT`] is abandoned and the engine is
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
    /// Performs Engine API operations and publishes the engine state.
    engine: Engine,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
}

impl<DerivationClient> Actor<DerivationClient>
where
    DerivationClient: EngineDerivationClient + 'static,
{
    fn new(
        derivation_client: DerivationClient,
        engine: Engine,
        messages_rx: mpsc::Receiver<Message>,
    ) -> Self {
        Self {
            derivation_client,
            el_sync_complete: false,
            last_safe_head_sent: L2BlockInfo::default(),
            engine,
            messages_rx,
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            match self.messages_rx.recv().await.ok_or(ActorError::ChannelClosed)? {
                Message::Build(request) => self.handle_build(*request).await,
                Message::Seal(request) => self.handle_seal(*request).await,
                Message::Consolidate(input) => self.handle_consolidate(input).await?,
                Message::Finalize(block_id) => self.handle_finalize(*block_id).await?,
                Message::InsertUnsafeBlock(payload) => self.handle_insert(*payload).await?,
                Message::Reset(result_tx) => self.handle_reset(result_tx).await?,
            }
            self.sync_derivation().await?;
        }
    }

    /// Starts a build and replies with its payload ID or failure; the caller decides whether to
    /// retry.
    async fn handle_build(&mut self, request: BuildRequest) {
        let BuildRequest { attributes, result_tx } = request;
        // A caller that stopped waiting does not cancel the build.
        let _ = result_tx.send(self.engine.build(attributes).await);
    }

    /// Seals a build and replies with the payload or failure; the caller decides whether to retry.
    async fn handle_seal(&mut self, request: SealRequest) {
        let SealRequest { payload_id, attributes, result_tx } = request;
        // A caller that stopped waiting does not cancel the seal.
        let _ = result_tx.send(self.engine.seal(payload_id, attributes).await);
    }

    async fn handle_consolidate(&mut self, input: ConsolidateInput) -> Result<(), ActorError> {
        self.retry(|engine| Box::pin(engine.consolidate(input.clone()))).await
    }

    async fn handle_finalize(&mut self, block_id: FinalizeBlockId) -> Result<(), ActorError> {
        self.retry(|engine| Box::pin(engine.finalize(block_id))).await
    }

    async fn handle_insert(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), ActorError> {
        self.retry(|engine| Box::pin(engine.insert_unsafe(payload.clone()))).await
    }

    async fn handle_reset(&mut self, result_tx: oneshot::Sender<()>) -> Result<(), ActorError> {
        warn!(target: "engine", "Received reset request");
        self.reset(Some(result_tx)).await
    }

    /// Performs an operation, retrying temporary failures with backoff, and acts on its other
    /// failures.
    async fn retry<E>(
        &mut self,
        mut operation: impl for<'a> FnMut(&'a mut Engine) -> Operation<'a, E> + Send,
    ) -> Result<(), ActorError>
    where
        E: EngineTaskError + std::fmt::Debug + Send,
        ActorError: From<E>,
    {
        let deadline = Instant::now() + TASK_RETRY_TIMEOUT;
        let mut backoff = Backoff::default();
        while let Err(err) = operation(&mut self.engine).await {
            match err.severity() {
                EngineTaskErrorSeverity::Temporary if Instant::now() < deadline => {
                    trace!(target: "engine", ?err, "Temporary error performing engine operation");
                    backoff.wait().await;
                }
                EngineTaskErrorSeverity::Temporary => {
                    warn!(target: "engine", ?err, "Engine operation kept failing; abandoning it and resetting");
                    return self.reset(None).await;
                }
                EngineTaskErrorSeverity::Critical => {
                    error!(target: "engine", ?err, "Critical error performing engine operation");
                    return Err(err.into());
                }
                EngineTaskErrorSeverity::Reset => {
                    warn!(target: "engine", ?err, "Engine operation requires a reset");
                    return self.reset(None).await;
                }
                EngineTaskErrorSeverity::Flush => {
                    // This error is encountered when the payload is marked INVALID by the engine
                    // api. Post-holocene, the payload is replaced by a "deposits-only" block, which
                    // completes the operation. At the same time, the channel and any remaining
                    // buffered batches are flushed.
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
        Ok(())
    }

    /// Sends derivation the safe head if it changed, and reports a completed EL sync.
    ///
    /// An EL sync that completes without a finalized head resets the engine to find one. After a
    /// reset, which reports both itself, this does nothing.
    async fn sync_derivation(&mut self) -> Result<(), ActorError> {
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
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
    };
    use kona_engine::{
        NoopBlockSink,
        test_utils::{
            RpcMock, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
            test_engine_client,
        },
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
        let engine =
            Engine::new(Arc::new(client), config, Arc::new(NoopBlockSink), state, state_tx);
        let actor =
            Actor::new(QueuedEngineDerivationClient::new(derivation_tx), engine, messages_rx);
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

    /// An operation that fails temporarily is retried after a backoff without another request.
    #[tokio::test(start_paused = true)]
    async fn temporary_failure_is_retried_until_the_operation_succeeds() {
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

    /// Build and seal callers that stop waiting for their replies do not stop the engine.
    #[tokio::test(start_paused = true)]
    async fn abandoned_replies_do_not_stop_the_engine() {
        let mut config = RollupConfig::default();
        config.hardforks.ecotone_time = Some(0);
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let started = || {
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid))
                .with_payload_id(PayloadId::new([1; 8]))
        };
        l2.expect_with_delay("engine_forkchoiceUpdatedV3", started(), Duration::from_secs(1));
        l2.expect("engine_forkchoiceUpdatedV3", started());
        let (derivation_tx, _derivation_rx) = mpsc::channel(8);
        let (handle, actor) = spawn(client, config, derivation_tx, EngineState::default());
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
    /// the engine is reset so derivation starts over from the execution layer's heads.
    #[tokio::test(start_paused = true)]
    async fn persistent_temporary_failure_abandons_the_operation_and_resets() {
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
