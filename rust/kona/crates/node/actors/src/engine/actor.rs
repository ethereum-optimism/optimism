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
    sync::{mpsc, watch},
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
            let engine = Engine::new(EngineState::default(), state_tx);
            Actor::new(client, rollup, derivation_client, engine, messages_rx, block_sink)
                .run()
                .await
        }
    }
}

/// Responsible for managing the operations sent to the execution layer's Engine API. To accomplish
/// this, it uses the [`Engine`] task queue to order Engine API  interactions based off of
/// the [`Ord`] implementation of [`EngineTask`].
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
    /// The [`Engine`] task queue.
    engine: Engine,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
    /// Where to hand every imported block, so the derivation providers can read it locally
    /// instead of fetching it back from the execution layer.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// A reset is retained until successful; RPC failure must not lose the request.
    reset_pending: bool,
    /// Callers waiting for the retained reset to finish.
    reset_waiters: Vec<mpsc::Sender<crate::EngineClientResult<()>>>,
    /// Earliest next attempt after a dependency failure.
    retry_at: Instant,
    /// Exponentially increasing retry delay, capped to keep recovery responsive.
    retry_delay: Duration,
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
            reset_pending: false,
            reset_waiters: Vec::new(),
            retry_at: Instant::now(),
            retry_delay: Duration::from_millis(100),
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            self.step().await?;
        }
    }

    fn defer_retry(&mut self) {
        self.retry_at = Instant::now() + self.retry_delay;
        self.retry_delay = (self.retry_delay * 2).min(Duration::from_secs(5));
    }

    fn begin_reset(&mut self) {
        if !self.reset_pending {
            self.engine.clear();
            self.reset_pending = true;
            self.retry_at = Instant::now();
        }
    }

    async fn attempt_reset(&mut self) -> Result<(), ActorError> {
        match self.reset().await {
            Ok(()) => {
                self.reset_pending = false;
                self.retry_delay = Duration::from_millis(100);
                if !self.el_sync_complete && self.engine.state().el_sync_finished {
                    self.mark_el_sync_complete_and_notify_derivation_actor().await?;
                }
            }
            Err(ActorError::EngineReset(err)) if err.is_temporary() => {
                warn!(target: "engine", ?err, "Reset dependency unavailable; retaining reset");
                self.defer_retry();
            }
            Err(err) => return Err(err),
        }
        Ok(())
    }

    /// Resets the inner [`Engine`] and propagates the reset to the derivation actor.
    async fn reset(&mut self) -> Result<(), ActorError> {
        // Reset the engine.
        let l2_safe_head = self.engine.reset(self.client.clone(), self.rollup.clone()).await?;

        // Derivation may be awaiting this reply with its inbound queue full of L1 updates.
        // Release those callers before awaiting delivery to that same queue. They transition
        // to AwaitingSignal before consuming the queued reset signal below.
        for waiter in self.reset_waiters.drain(..) {
            let _ = waiter.try_send(Ok(()));
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

        Ok(())
    }

    /// Drains the inner [`Engine`] task queue and attempts to update the safe head.
    async fn drain(&mut self) -> Result<(), ActorError> {
        match self.engine.drain().await {
            Ok(_) => {
                trace!(target: "engine", "[ENGINE] tasks drained");
                self.retry_delay = Duration::from_millis(100);
            }
            Err(err) => {
                match err.severity() {
                    EngineTaskErrorSeverity::Critical => {
                        error!(target: "engine", ?err, "Critical error draining engine tasks");
                        return Err(err.into());
                    }
                    EngineTaskErrorSeverity::Reset => {
                        warn!(target: "engine", ?err, "Received reset request");
                        self.begin_reset();
                        return Ok(());
                    }
                    EngineTaskErrorSeverity::Flush => {
                        // This error is encountered when the payload is marked INVALID
                        // by the engine api. Post-holocene, the payload is replaced by
                        // a "deposits-only" block and re-executed. At the same time,
                        // the channel and any remaining buffered batches are flushed.
                        warn!(target: "engine", ?err, "Invalid payload, Flushing derivation pipeline.");
                        match self.derivation_client.send_signal(Signal::FlushChannel).await {
                            Ok(_) => {
                                debug!(target: "engine", "Sent flush signal to derivation actor")
                            }
                            Err(err) => {
                                error!(target: "engine", ?err, "Failed to send flush signal to the derivation actor.");
                                return Err(ActorError::ChannelClosed);
                            }
                        }
                    }
                    EngineTaskErrorSeverity::Temporary => {
                        trace!(target: "engine", ?err, "Temporary error draining engine tasks");
                        self.defer_retry();
                        return Ok(());
                    }
                }
            }
        }

        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            if self.engine.state().sync_state.finalized_head() == L2BlockInfo::default() {
                self.begin_reset();
            } else {
                self.mark_el_sync_complete_and_notify_derivation_actor().await?;
            }
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

    /// Runs due retries, then waits for and enqueues the next request.
    async fn step(&mut self) -> Result<(), ActorError> {
        if Instant::now() >= self.retry_at {
            if self.reset_pending {
                self.attempt_reset().await?;
            } else {
                self.drain().await?;
            }
        }

        // Wait for the next processing request.
        // Retry without needing a new inbound message, while still accepting requests during
        // backoff. Bound queued work so a prolonged outage cannot grow memory without limit.
        let message = tokio::select! {
            message = self.messages_rx.recv(),
                if self.engine.len() < 1024 && self.reset_waiters.len() < 1024 => {
                message.ok_or(ActorError::ChannelClosed)?
            }
            _ = time::sleep_until(self.retry_at),
                if self.reset_pending || !self.engine.is_empty() => return Ok(()),
        };

        match message {
            Message::Build(build_request) => {
                let BuildRequest { attributes, result_tx } = *build_request;
                let task = EngineTask::Build(Box::new(BuildTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    attributes,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
            Message::Consolidate(safe_signal) => {
                let task = EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    safe_signal,
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            Message::Finalize(finalized_l2_block_id) => {
                // Finalize the L2 block identified by the provided [`FinalizeBlockId`].
                let task = EngineTask::Finalize(Box::new(FinalizeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *finalized_l2_block_id,
                )));
                self.engine.enqueue(task);
            }
            Message::InsertUnsafeBlock(envelope) => {
                let task = EngineTask::Insert(Box::new(InsertTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *envelope,
                    false, /* The payload is not derived in this case. This is an unsafe
                            * block. */
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            Message::Reset(result_tx) => {
                warn!(target: "engine", "Received reset request");

                self.begin_reset();
                self.reset_waiters.push(result_tx);
            }
            Message::Seal(seal_request) => {
                let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                let task = EngineTask::Seal(Box::new(SealTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload_id,
                    attributes,
                    // The payload is not derived in this case.
                    false,
                    BuildSealCoupling::Detached,
                    Some(result_tx),
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{DerivationActorRequest, QueuedEngineDerivationClient};
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use kona_engine::{NoopBlockSink, test_utils::test_engine_client};

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

    /// Both L1 and L2 outages during a reset must retain the original request and finish it
    /// after recovery, without a new reset request or L1 head to wake the actor.
    #[rstest::rstest]
    #[case(true, Duration::ZERO)]
    #[case(false, Duration::ZERO)]
    #[case(false, Duration::from_secs(4))]
    #[tokio::test(start_paused = true)]
    async fn reset_recovers_and_completes_original_request(
        #[case] l1_failure: bool,
        #[case] read_delay: Duration,
    ) {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let client = Arc::new(client);
        if l1_failure {
            for tag in ["finalized", "safe", "latest"] {
                l2.expect_params("eth_getBlockByNumber", serde_json::json!([tag, true]), &block);
            }
            l1.expect_error("eth_getBlockByHash");
        } else {
            l2.expect_error("eth_getBlockByNumber");
        }
        let (derivation_tx, mut derivation_rx) = mpsc::channel(1);
        // Derivation waits for the reset reply and cannot drain its full queue until then.
        derivation_tx
            .send(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::default()))
            .await
            .unwrap();
        let derivation =
            QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx };
        let (messages, messages_rx) = mpsc::channel(8);
        let (state_tx, state_rx) = tokio::sync::watch::channel(EngineState::default());
        let handle = Handle::new(messages, state_rx);
        let genesis = config.genesis.l2.hash;
        let reader = handle.clone();
        let mut actor = Actor::new(
            client,
            config,
            derivation,
            Engine::new(EngineState::default(), state_tx),
            messages_rx,
            Arc::new(NoopBlockSink),
        );
        let reset = tokio::spawn(async move { handle.reset().await });
        actor.step().await.unwrap(); // retain the reset
        actor.step().await.unwrap(); // failed attempt, then wait for backoff
        assert!(!reset.is_finished());
        for _ in 0..3 {
            l2.expect_with_delay("eth_getBlockByNumber", &block, read_delay);
        }
        l1.expect(
            "eth_getBlockByHash",
            alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
        );
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        // A step performs recovery and then waits for another message; the response arrives first.
        let mut tasks = tokio::task::JoinSet::new();
        tasks.spawn(actor.run());
        time::timeout(Duration::from_secs(20), reset).await.unwrap().unwrap().unwrap();
        // The reset's heads are published before the reply.
        assert_eq!(reader.state().borrow().sync_state.unsafe_head().block_info.hash, genesis);
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
        tasks.abort_all();
        l1.assert_finished();
        l2.assert_finished();
    }
}
