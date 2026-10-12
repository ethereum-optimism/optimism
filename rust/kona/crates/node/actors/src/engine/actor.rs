//! Engine construction and execution.

use super::{
    ActorError, BuildTaskError, Capacity, ConsolidateInput, ConsolidateTaskError, EngineBuildError,
    EngineResetError, EngineTaskError, EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTaskError,
    Handle, InsertTaskError, SealTaskError, SynchronizeTaskError,
    handle::{BuildRequest, Message, SealRequest},
};
use crate::EngineDerivationClient;
use alloy_eips::BlockId;
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadId, PayloadStatusEnum};
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    AttributesMatch, EngineClient, EngineState, EngineSyncStateUpdate, ImportedBlockSink, Metrics,
    find_starting_forkchoice,
};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{future::Future, pin::Pin, sync::Arc};
use tokio::{
    sync::{mpsc, oneshot, watch},
    time::{self, Duration, Instant},
};

#[cfg(test)]
mod tests;

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
        Actor::new(derivation_client, client, rollup, block_sink, state_tx, messages_rx).run()
    }
}

/// The delay before the first retry of work that failed temporarily.
const INITIAL_RETRY_DELAY: Duration = Duration::from_millis(100);
/// The longest delay between retries, capped to keep recovery responsive.
const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);
/// How long an operation may keep failing temporarily before it is abandoned and the engine reset.
const TASK_RETRY_TIMEOUT: Duration = Duration::from_secs(60);

/// One attempt at an operation, boxed so the actor's future stays `Send`.
type Attempt<'a, E> = Pin<Box<dyn Future<Output = Result<(), E>> + Send + 'a>>;

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

/// Performs the operations sent to the execution layer's Engine API, one request at a time in the
/// order they arrive, and publishes the engine state after each request and before replying.
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
    /// The Engine API client.
    client: Arc<EngineClient>,
    /// The rollup configuration.
    config: Arc<RollupConfig>,
    /// Where to hand every block the engine makes canonical.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// The state of the engine. Operations update it, and it is published after every request.
    state: EngineState,
    /// Publishes the state to handles.
    state_tx: watch::Sender<EngineState>,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
}

impl<DerivationClient> Actor<DerivationClient>
where
    DerivationClient: EngineDerivationClient + 'static,
{
    /// Creates the actor, starting from the state that `state_tx` holds.
    fn new(
        derivation_client: DerivationClient,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        block_sink: Arc<dyn ImportedBlockSink>,
        state_tx: watch::Sender<EngineState>,
        messages_rx: mpsc::Receiver<Message>,
    ) -> Self {
        let state = *state_tx.borrow();
        Self {
            derivation_client,
            el_sync_complete: false,
            last_safe_head_sent: L2BlockInfo::default(),
            client,
            config,
            block_sink,
            state,
            state_tx,
            messages_rx,
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            let result = match self.messages_rx.recv().await.ok_or(ActorError::ChannelClosed)? {
                Message::Build(request) => {
                    self.handle_build(*request).await;
                    Ok(())
                }
                Message::Seal(request) => {
                    self.handle_seal(*request).await;
                    Ok(())
                }
                Message::Consolidate(input) => {
                    self.retry(Metrics::CONSOLIDATE_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_consolidate(input.clone()))
                    })
                    .await
                }
                Message::Finalize(block_id) => {
                    self.retry(Metrics::FINALIZE_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_finalize(*block_id))
                    })
                    .await
                }
                Message::InsertUnsafeBlock(payload) => {
                    self.retry(Metrics::INSERT_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_insert((*payload).clone()))
                    })
                    .await
                }
                Message::Reset(result_tx) => {
                    warn!(target: "engine", "Received reset request");
                    self.reset(Some(result_tx)).await
                }
            };
            // Publish the state even if the request failed: it can update the state before a
            // later step fails.
            self.publish_state();
            result?;
            self.sync_derivation().await?;
        }
    }

    /// Starts a build and replies with its payload ID or failure; the caller decides whether to
    /// retry.
    async fn handle_build(&self, request: BuildRequest) {
        let BuildRequest { attributes, result_tx } = request;
        let result = record(Metrics::BUILD_TASK_LABEL, self.start_build(attributes).await);
        // A caller that stopped waiting does not cancel the build.
        let _ = result_tx.send(result);
    }

    /// Seals a build, makes it the unsafe head, and replies with the payload or failure; the caller
    /// decides whether to retry.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head moved since
    /// the build started.
    async fn handle_seal(&mut self, request: SealRequest) {
        let SealRequest { payload_id, attributes, result_tx } = request;

        let unsafe_head = self.state.sync_state.unsafe_head().block_info;
        let parent = attributes.parent.block_info;
        let result = if unsafe_head.id() == parent.id() {
            self.seal_payload(payload_id, attributes, false).await
        } else {
            info!(
                target: "engine",
                unsafe_block_info = ?unsafe_head,
                parent_block_info = ?parent,
                "Seal attributes parent does not match unsafe head, returning rebuild error"
            );
            Err(SealTaskError::UnsafeHeadChangedSinceBuild)
        };

        // The sequencer builds the next block on the unsafe head as soon as it gets the reply.
        self.publish_state();
        // A caller that stopped waiting does not cancel the seal.
        let _ = result_tx.send(record(Metrics::SEAL_TASK_LABEL, result));
    }

    /// Inserts an unsafe payload received from gossip or the admin API.
    ///
    /// A payload the execution layer reports as `INVALID` or `ACCEPTED` is dropped: retrying
    /// returns the same verdict.
    async fn handle_insert(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), InsertTaskError> {
        match self.insert_payload(payload, false).await {
            Err(InsertTaskError::UnexpectedPayloadStatus(
                status @ (PayloadStatusEnum::Invalid { .. } | PayloadStatusEnum::Accepted),
            )) => {
                warn!(target: "engine", %status, "Dropping rejected unsafe payload");
                Ok(())
            }
            result => result.map(|_| ()),
        }
    }

    /// Advances the safe head to the input, reorging the unsafe chain to it if necessary.
    ///
    /// Behavior depends on how the safe head is provided:
    ///
    /// - `Attributes`: The safe head is advanced through the normal derivation flow, where the
    ///   derivation and engine actors coordinate both safe and unsafe heads. In this case, we
    ///   consolidate as long as the unsafe head has not fallen behind.
    ///
    /// - `BlockInfo`: The safe head is injected externally by the derivation actor while delegating
    ///   derivation, and is not coordinated with the engine actor's safe and unsafe heads. If the
    ///   injected safe head is ahead of the engine actor's unsafe head, we reconcile the unsafe
    ///   chain up to the safe head instead of consolidating.
    async fn handle_consolidate(
        &mut self,
        input: ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        let (safe_head_number, block_num) = match &input {
            ConsolidateInput::Attributes(attributes) => {
                (self.state.sync_state.safe_head().block_info.number, attributes.block_number())
            }
            ConsolidateInput::BlockInfo(info) => (info.block_info.number, info.block_info.number),
        };
        if safe_head_number >= self.state.sync_state.unsafe_head().block_info.number {
            return self.reconcile_unsafe_to_safe(&input).await;
        }

        // Fetch the unsafe L2 block
        let block = match self.client.l2_block_by_label(block_num.into()).await {
            Ok(Some(block)) => block,
            Ok(None) => {
                warn!(target: "engine", "Received `None` block for {}", block_num);
                return Err(ConsolidateTaskError::MissingUnsafeL2Block(block_num));
            }
            Err(_) => {
                warn!(target: "engine", "Failed to fetch unsafe l2 block for consolidation");
                return Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block);
            }
        };
        let block_hash = block.header.hash;

        let is_consistent = match &input {
            ConsolidateInput::Attributes(attributes) => {
                AttributesMatch::check(&self.config, attributes, &block).is_match()
            }
            ConsolidateInput::BlockInfo(info) => block_hash == info.block_info.hash,
        };
        if !is_consistent {
            debug!(
                target: "engine",
                input = ?input,
                block_hash = %block_hash,
                "ConsolidateInput mismatch! Initiating reorg",
            );
            return self.reconcile_unsafe_to_safe(&input).await;
        }

        trace!(
            target: "engine",
            input = ?input,
            block_hash = %block_hash,
            "Consolidating engine state",
        );
        let consensus_block =
            block.into_consensus().map_transactions(|t| t.inner.inner.into_inner());
        let block_info = match L2BlockInfo::from_block_and_genesis(
            &consensus_block,
            &self.config.genesis,
        ) {
            Ok(block_info) => block_info,
            Err(e) => {
                // Build the block instead, since its block info cannot be constructed.
                warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, proceeding to build");
                debug!(
                    target: "engine",
                    input = ?input,
                    block_hash = %block_hash,
                    "ConsolidateInput mismatch! Initiating reorg",
                );
                return self.reconcile_unsafe_to_safe(&input).await;
            }
        };

        // The next attributes built are this block's child, and ask for its config.
        self.block_sink.block_imported(consensus_block, block_info);

        let update = EngineSyncStateUpdate {
            safe_head: Some(block_info),
            local_safe_head: Some(block_info),
            ..Default::default()
        };
        match &input {
            // Only issue a forkchoice update if the attributes are the last in the span batch.
            // This is an optimization to avoid sending a FCU call for every block in the span
            // batch.
            ConsolidateInput::Attributes(attributes) if attributes.is_last_in_span => {
                self.synchronize(update)
                    .await
                    .inspect_err(|e| warn!(target: "engine", ?e, "Consolidation failed"))?;
            }
            // Apply a transient update to the safe head.
            _ => self.state.sync_state = self.state.sync_state.apply_update(update),
        }

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Updated safe head via L1 consolidation"
        );

        Ok(())
    }

    /// Fetches the [`L2BlockInfo`] identified by `block_id`, makes it the finalized head, and
    /// dispatches a forkchoice update to finalize the block.
    async fn handle_finalize(
        &mut self,
        block_id: FinalizeBlockId,
    ) -> Result<(), FinalizeTaskError> {
        let block_number = block_id.number();

        // Sanity check that the block that is being finalized is at least safe.
        if self.state.sync_state.safe_head().block_info.number < block_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        // Look up by hash when the caller pinned a specific hash (delegated polling supplies
        // `(number, hash)`); otherwise look up by number. Finalization is irreversible, so a stale
        // hash must produce a hard error, not silently finalize whatever the engine has at the
        // same height.
        let lookup: BlockId = match block_id {
            FinalizeBlockId::ByHash(id) => id.hash.into(),
            FinalizeBlockId::ByNumber(n) => n.into(),
        };

        let block = self
            .client
            .get_l2_block(lookup)
            .full()
            .await
            .map_err(FinalizeTaskError::TransportError)?
            .ok_or(FinalizeTaskError::BlockNotFound(block_number))?
            .into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.client.cfg().genesis)
            .map_err(FinalizeTaskError::FromBlock)?;

        // For ByHash, also assert the returned block's height matches the caller's claim. The
        // engine should never serve a block at a different height for a given hash, but a height
        // mismatch indicates either an upstream protocol bug or a misuse of the API and must
        // never lead to silent finalization.
        if let FinalizeBlockId::ByHash(id) = block_id &&
            block_info.block_info.number != id.number
        {
            return Err(FinalizeTaskError::BlockNotFound(id.number));
        }

        // Dispatch a forkchoice update.
        self.synchronize(EngineSyncStateUpdate {
            finalized_head: Some(block_info),
            ..Default::default()
        })
        .await?;

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Updated finalized head"
        );

        Ok(())
    }

    /// Performs an operation, retrying temporary failures with backoff, and acts on its other
    /// failures.
    async fn retry<E>(
        &mut self,
        label: &'static str,
        mut attempt: impl for<'a> FnMut(&'a mut Self) -> Attempt<'a, E> + Send,
    ) -> Result<(), ActorError>
    where
        E: EngineTaskError + std::fmt::Debug + Send,
        ActorError: From<E>,
    {
        let deadline = Instant::now() + TASK_RETRY_TIMEOUT;
        let mut backoff = Backoff::default();
        while let Err(err) = record(label, attempt(self).await) {
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
                    // Derivation continues from the deposits-only block, so publish it first.
                    self.publish_state();
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

    /// Publishes the state to handles.
    ///
    /// The actor publishes after every request, and before any reply or derivation signal whose
    /// recipient reads the state.
    fn publish_state(&self) {
        self.state_tx.send_replace(self.state);
    }

    /// Sends derivation the safe head if it changed, and reports a completed EL sync.
    ///
    /// An EL sync that completes without a finalized head resets the engine to find one. After a
    /// reset, which reports both itself, this does nothing.
    async fn sync_derivation(&mut self) -> Result<(), ActorError> {
        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.state.el_sync_finished {
            if self.state.sync_state.finalized_head() == L2BlockInfo::default() {
                return self.reset(None).await;
            }
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    /// Resets the engine to a sync starting point found with [`find_starting_forkchoice`], and
    /// reorgs the execution layer to it. Then publishes the state, replies to the caller, if any,
    /// and propagates the reset to the derivation actor.
    ///
    /// Temporary failures are retried until the reset succeeds: losing it would leave derivation
    /// waiting for a reset signal.
    async fn reset(&mut self, result_tx: Option<oneshot::Sender<()>>) -> Result<(), ActorError> {
        let mut backoff = Backoff::default();
        let l2_safe_head = loop {
            let attempt = async {
                let start = find_starting_forkchoice(&self.config, self.client.as_ref()).await?;
                self.synchronize(EngineSyncStateUpdate {
                    unsafe_head: Some(start.un_safe),
                    local_safe_head: Some(start.safe),
                    safe_head: Some(start.safe),
                    finalized_head: Some(start.finalized),
                })
                .await?;
                Ok::<_, EngineResetError>(start.safe)
            };
            match attempt.await {
                Ok(l2_safe_head) => break l2_safe_head,
                Err(err) if err.is_temporary() => {
                    warn!(target: "engine", ?err, "Reset dependency unavailable; retrying reset");
                    backoff.wait().await;
                }
                Err(err) => return Err(err.into()),
            }
        };
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        // The caller reads the reset's heads as soon as it gets the reply.
        self.publish_state();

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

        if !self.el_sync_complete && self.state.el_sync_finished {
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    async fn mark_el_sync_complete_and_notify_derivation_actor(
        &mut self,
    ) -> Result<(), ActorError> {
        self.el_sync_complete = true;

        self.derivation_client
            .notify_sync_completed(self.state.sync_state.safe_head())
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                ActorError::ChannelClosed
            })?
    }

    /// Attempts to send the [`crate::DerivationActor`] the safe head if updated.
    async fn send_derivation_actor_safe_head_if_updated(&mut self) -> Result<(), ActorError> {
        let engine_safe_head = self.state.sync_state.safe_head();
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

    /// Applies a sync state update and sends the resulting forkchoice to the execution layer with
    /// `engine_forkchoiceUpdated`, without payload attributes.
    ///
    /// Inserting, consolidating, and finalizing blocks, and resetting, all use this to move the
    /// forkchoice. Starting a build sends its own forkchoice update with payload attributes
    /// instead.
    async fn synchronize(
        &mut self,
        update: EngineSyncStateUpdate,
    ) -> Result<(), SynchronizeTaskError> {
        // Apply the sync state update to the engine state.
        let new_sync_state = self.state.sync_state.apply_update(update);

        // Check if a forkchoice update is not needed, return early.
        // A forkchoice update is not needed if...
        // 1. The engine state is not default (initial forkchoice state has been emitted), and
        // 2. The new sync state is the same as the current sync state (no changes to the sync
        //    state).
        //
        // NOTE:
        // We shouldn't retry the forkchoice update there. Since the `sync_state` is only updated
        // here (except during consolidation, when the block is not the last in the batch) - the
        // engine would get stuck retrying the forkchoice update.
        if self.state.sync_state != Default::default() && self.state.sync_state == new_sync_state {
            debug!(target: "engine", ?new_sync_state, "No forkchoice update needed");
            return Ok(());
        }

        // Check if the head is behind the finalized head.
        if new_sync_state.unsafe_head().block_info.number <
            new_sync_state.finalized_head().block_info.number
        {
            return Err(SynchronizeTaskError::FinalizedAheadOfUnsafe(
                new_sync_state.unsafe_head().block_info.number,
                new_sync_state.finalized_head().block_info.number,
            ));
        }

        // Send the forkchoice update through the input.
        let forkchoice = new_sync_state.create_forkchoice_state();

        // Handle the forkchoice update result.
        let response = self.client.forkchoice_updated(forkchoice, None).await;

        let valid_response = response.map_err(|e| {
            // Fatal forkchoice update error.
            let error = e
                .as_error_resp()
                .and_then(|e| {
                    (e.code == INVALID_FORK_CHOICE_STATE_ERROR as i64)
                        .then_some(SynchronizeTaskError::InvalidForkchoiceState)
                })
                .unwrap_or_else(|| SynchronizeTaskError::ForkchoiceUpdateFailed(e));

            debug!(target: "engine", error = ?error, "Unexpected forkchoice update error");

            error
        })?;

        match &valid_response.payload_status.status {
            PayloadStatusEnum::Valid => {
                if !self.state.el_sync_finished {
                    info!(
                        target: "engine",
                        "Finished execution layer sync."
                    );
                    self.state.el_sync_finished = true;
                }
            }
            PayloadStatusEnum::Syncing => {
                // If we're not building a new payload, we're driving EL sync.
                debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
            }
            status => {
                // Other codes are not expected.
                return Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone()));
            }
        }

        // Apply the new sync state to the engine state.
        self.state.sync_state = new_sync_state;

        debug!(
            target: "engine",
            forkchoice = ?forkchoice,
            response = ?valid_response,
            "Forkchoice updated"
        );

        Ok(())
    }

    /// Starts building a block by sending `engine_forkchoiceUpdated` with the payload attributes,
    /// and returns the [`PayloadId`] to seal it with.
    ///
    /// The forkchoice advertises the attributes' parent as the unsafe head to build on. A
    /// `SYNCING` execution layer cannot start the build, which may be retried later.
    async fn start_build(
        &self,
        attributes_envelope: OpAttributesWithParent,
    ) -> Result<PayloadId, BuildTaskError> {
        debug!(
            target: "engine_builder",
            txs = attributes_envelope.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = attributes_envelope.is_deposits_only(),
            "Starting new build job"
        );

        // Sanity check if the head is behind the finalized head. If it is, this is a critical
        // error.
        if self.state.sync_state.unsafe_head().block_info.number <
            self.state.sync_state.finalized_head().block_info.number
        {
            return Err(BuildTaskError::EngineBuildError(
                EngineBuildError::FinalizedAheadOfUnsafe(
                    self.state.sync_state.unsafe_head().block_info.number,
                    self.state.sync_state.finalized_head().block_info.number,
                ),
            ));
        }

        // When inserting a payload, we advertise the parent's unsafe head as the current unsafe
        // head to build on top of.
        let new_forkchoice = self
            .state
            .sync_state
            .apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(attributes_envelope.parent),
                ..Default::default()
            })
            .create_forkchoice_state();

        let update = self
            .client
            .forkchoice_updated(new_forkchoice, Some(attributes_envelope.attributes))
            .await
            .map_err(|e| {
                error!(target: "engine_builder", "Forkchoice update failed: {}", e);
                BuildTaskError::EngineBuildError(EngineBuildError::AttributesInsertionFailed(e))
            })?;

        match update.payload_status.status {
            PayloadStatusEnum::Valid => {}
            PayloadStatusEnum::Invalid { validation_error } => {
                error!(target: "engine_builder", "Forkchoice update failed: {}", validation_error);
                return Err(BuildTaskError::EngineBuildError(EngineBuildError::InvalidPayload(
                    validation_error,
                )));
            }
            PayloadStatusEnum::Syncing => {
                warn!(target: "engine_builder", "Forkchoice update failed temporarily: EL is syncing");
                return Err(BuildTaskError::EngineBuildError(EngineBuildError::EngineSyncing));
            }
            status @ PayloadStatusEnum::Accepted => {
                // Other codes are never returned by `engine_forkchoiceUpdate`
                return Err(BuildTaskError::EngineBuildError(
                    EngineBuildError::UnexpectedPayloadStatus(status),
                ));
            }
        }

        debug!(
            target: "engine_builder",
            unsafe_hash = new_forkchoice.head_block_hash.to_string(),
            safe_hash = new_forkchoice.safe_block_hash.to_string(),
            finalized_hash = new_forkchoice.finalized_block_hash.to_string(),
            "Forkchoice update with attributes successful"
        );

        // Fetch the payload ID from the FCU. If no payload ID was returned, something went wrong -
        // the block building job on the EL should have been initiated.
        let payload_id = update
            .payload_id
            .ok_or(BuildTaskError::EngineBuildError(EngineBuildError::MissingPayloadId))?;

        info!(
            target: "engine_builder",
            "block build started"
        );

        Ok(payload_id)
    }

    /// Inserts a payload with `engine_newPayload` and makes it the unsafe head.
    ///
    /// A payload derived from a safe block (`is_payload_safe`) also becomes the safe head.
    async fn insert_payload(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
        is_payload_safe: bool,
    ) -> Result<L2BlockInfo, InsertTaskError> {
        // Insert the new payload.
        let response = self
            .client
            .new_payload(payload.clone())
            .await
            .inspect_err(|e| warn!(target: "engine", "Failed to insert new payload: {e}"))
            .map_err(InsertTaskError::InsertFailed)?;
        if !matches!(response.status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
        }

        let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
        let new_unsafe_ref = L2BlockInfo::from_block_and_genesis(&block, &self.config.genesis)
            .map_err(InsertTaskError::L2BlockInfoConstruction)?;

        // Send a FCU to canonicalize the imported block.
        self.synchronize(EngineSyncStateUpdate {
            unsafe_head: Some(new_unsafe_ref),
            local_safe_head: is_payload_safe.then_some(new_unsafe_ref),
            safe_head: is_payload_safe.then_some(new_unsafe_ref),
            ..Default::default()
        })
        .await?;

        // The block is now canonical, so anything reading the L2 chain locally can rely on it.
        self.block_sink.block_imported(block, new_unsafe_ref);

        info!(
            target: "engine",
            hash = %new_unsafe_ref.block_info.hash,
            number = new_unsafe_ref.block_info.number,
            "Inserted new unsafe block"
        );

        Ok(new_unsafe_ref)
    }

    /// Seals the payload built from `attributes` and imports it, returning the sealed payload.
    ///
    /// Retrieves the built payload using `engine_getPayload`, then imports it with
    /// `engine_newPayload` and a forkchoice update. After Holocene, a payload the execution layer
    /// rejects is replaced by a deposits-only block, and the seal fails with
    /// [`SealTaskError::HoloceneInvalidFlush`] to flush the derivation pipeline.
    /// `is_attributes_derived` says whether the attributes were derived or created by the
    /// sequencer.
    async fn seal_payload(
        &mut self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
        is_attributes_derived: bool,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        debug!(
            target: "engine",
            txs = attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = attributes.is_deposits_only(),
            "Starting new seal job"
        );

        // Fetch the payload just built from the EL and import it into the engine.
        let timestamp = attributes.attributes().payload_attributes.timestamp;
        debug!(
            target: "engine",
            payload_id = payload_id.to_string(),
            l2_time = timestamp,
            "Sealing payload"
        );
        let payload = self.client.get_payload(payload_id, timestamp).await.map_err(|e| {
            error!(target: "engine", "Payload fetch failed: {e}");
            SealTaskError::GetPayloadFailed(e)
        })?;

        let new_block_ref = match self.insert_payload(payload.clone(), is_attributes_derived).await
        {
            Ok(new_block_ref) => {
                info!(target: "engine", "Successfully imported payload");
                new_block_ref
            }
            Err(InsertTaskError::UnexpectedPayloadStatus(e)) if attributes.is_deposits_only() => {
                error!(target: "engine", error = ?e, "Critical: Deposit-only payload import failed");
                return Err(SealTaskError::DepositOnlyPayloadFailed);
            }
            Err(InsertTaskError::UnexpectedPayloadStatus(e))
                if self.config.is_holocene_active(timestamp) =>
            {
                warn!(target: "engine", error = ?e, "Re-attempting payload import with deposits only.");

                // HOLOCENE: Re-attempt payload import with deposits only
                // First build the deposits-only payload, then seal it
                let deposits_only_attrs = attributes.as_deposits_only();

                return match self.build_and_seal(deposits_only_attrs, is_attributes_derived).await {
                    Ok(()) => {
                        info!(target: "engine", "Successfully imported deposits-only payload");
                        Err(SealTaskError::HoloceneInvalidFlush)
                    }
                    Err(_) => Err(SealTaskError::DepositOnlyPayloadReattemptFailed),
                };
            }
            Err(e) => {
                error!(target: "engine", "Payload import failed: {e}");
                return Err(Box::new(e).into());
            }
        };

        info!(
            target: "engine",
            l2_number = new_block_ref.block_info.number,
            l2_time = new_block_ref.block_info.timestamp,
            "Built and imported new {} block",
            if is_attributes_derived { "safe" } else { "unsafe" },
        );

        Ok(payload)
    }

    /// Builds and seals a payload in sequence, so no other request can advance the unsafe head
    /// between them.
    ///
    /// Unlike a seal request, this does not check that the attributes' parent is the unsafe head:
    /// a derivation-driven reorg legitimately seals a payload whose parent differs from it.
    /// Consolidation uses this to build derived attributes that do not match the unsafe chain, and
    /// sealing uses it for the Holocene deposits-only fallback.
    ///
    /// `is_attributes_derived` says whether the attributes were derived or created by the
    /// sequencer.
    async fn build_and_seal(
        &mut self,
        attributes: OpAttributesWithParent,
        is_attributes_derived: bool,
    ) -> Result<(), ConsolidateTaskError> {
        let payload_id = self.start_build(attributes.clone()).await?;

        // Sealing can fall back to building and sealing a deposits-only block, so box the
        // recursive call.
        Box::pin(self.seal_payload(payload_id, attributes, is_attributes_derived)).await?;

        Ok(())
    }

    /// Makes the unsafe chain agree with a safe head that does not match it.
    ///
    /// Derived attributes are built and sealed on their parent. A delegated safe block becomes
    /// the unsafe and safe head.
    async fn reconcile_unsafe_to_safe(
        &mut self,
        input: &ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        let safe_l2 = match input {
            ConsolidateInput::Attributes(attributes) => {
                return self.build_and_seal((**attributes).clone(), true).await;
            }
            ConsolidateInput::BlockInfo(safe_l2) => safe_l2,
        };

        warn!(
            target: "engine",
            safe_l2 = %safe_l2,
            "Apply safe head"
        );

        // We intentionally set the unsafe head to safe_l2 to ensure the engine observes a
        // self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the safe head.
        self.synchronize(EngineSyncStateUpdate {
            unsafe_head: Some(*safe_l2),
            safe_head: Some(*safe_l2),
            local_safe_head: Some(*safe_l2),
            ..Default::default()
        })
        .await
        .inspect_err(|e| warn!(target: "engine", ?e, "Apply safe head failed"))?;

        info!(
            target: "engine",
            hash = %safe_l2.block_info.hash,
            number = safe_l2.block_info.number,
            "Updated safe head via follow safe"
        );

        Ok(())
    }
}

/// Records the outcome of one attempt at an operation, and returns it.
fn record<T, E: EngineTaskError>(label: &'static str, result: Result<T, E>) -> Result<T, E> {
    match &result {
        Ok(_) => {
            metrics::counter!(Metrics::ENGINE_TASK_SUCCESS, "type" => label).increment(1);
        }
        Err(err) => {
            metrics::counter!(Metrics::ENGINE_TASK_FAILURE, label => err.severity().to_string())
                .increment(1);
        }
    }
    result
}
