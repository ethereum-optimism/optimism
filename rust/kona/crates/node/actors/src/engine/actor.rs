//! Engine construction and execution.

use super::{
    ActorError, BuildTaskError, ConsolidateTaskError, Consolidation, EngineBuildError,
    EngineClientError, EngineResetError, EngineState, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeBlockId, FinalizeTaskError, Handle, HeadsUpdate, ImportedBlockSink, InsertTaskError,
    SealTaskError, SynchronizeTaskError,
    attributes::AttributesMatch,
    handle::{BuildRequest, ConsolidateRequest, Message, ResetRequest, SealRequest},
    sync::find_starting_forkchoice,
};
use crate::{Metrics, l1};
use alloy_eips::BlockId;
use alloy_rpc_types_engine::{
    ForkchoiceState, INVALID_FORK_CHOICE_STATE_ERROR, PayloadId, PayloadStatusEnum,
};
use alloy_rpc_types_eth::Block as RpcBlock;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use kona_sources::EngineApi;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types::Transaction;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{future::Future, pin::Pin, sync::Arc};
use tokio::{
    sync::{mpsc, watch},
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
    resets_tx: watch::Sender<EngineState>,
}

impl Builder {
    /// Creates the builder with the engine starting from `state`, the execution layer's heads as
    /// [`execution_layer_state`](super::execution_layer_state) reads them. Its handles read those
    /// heads until the engine's first reset replaces them.
    ///
    /// The request queue holds one request: a sender waits for room in it, and the network drops
    /// the unsafe blocks it receives over gossip while it is full.
    pub fn new(state: EngineState) -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(1);
        let (state_tx, state_rx) = watch::channel(state);
        let (resets_tx, resets_rx) = watch::channel(state);
        Self {
            handle: Handle::new(messages_tx, state_rx, resets_rx),
            messages_rx,
            state_tx,
            resets_tx,
        }
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
    pub fn build(
        self,
        client: Arc<EngineApi>,
        l1: l1::Handle,
        rollup: Arc<RollupConfig>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages_rx, state_tx, resets_tx } = self;
        Actor::new(client, l1, rollup, block_sink, state_tx, resets_tx, messages_rx).run()
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
struct Actor {
    /// The Engine API client.
    client: Arc<EngineApi>,
    /// The L1 chain, which a reset checks to find a plausible forkchoice.
    l1: l1::Handle,
    /// The rollup configuration.
    config: Arc<RollupConfig>,
    /// Where to hand every block the engine makes canonical.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// The state of the engine. Operations update it, and it is published after every request.
    state: EngineState,
    /// Whether the engine has sent the execution layer a forkchoice. It starts from the execution
    /// layer's own heads, which it has not chosen until it sends them.
    forkchoice_sent: bool,
    /// Publishes the state to handles.
    state_tx: watch::Sender<EngineState>,
    /// Publishes the state after each reset to handles.
    resets_tx: watch::Sender<EngineState>,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
}

impl Actor {
    /// Creates the actor, starting from the state that `state_tx` holds.
    fn new(
        client: Arc<EngineApi>,
        l1: l1::Handle,
        config: Arc<RollupConfig>,
        block_sink: Arc<dyn ImportedBlockSink>,
        state_tx: watch::Sender<EngineState>,
        resets_tx: watch::Sender<EngineState>,
        messages_rx: mpsc::Receiver<Message>,
    ) -> Self {
        let state = *state_tx.borrow();
        Self {
            client,
            l1,
            config,
            block_sink,
            state,
            forkchoice_sent: false,
            state_tx,
            resets_tx,
            messages_rx,
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            let message = self.messages_rx.recv().await.ok_or(ActorError::ChannelClosed)?;
            let result = match message {
                Message::Build(request) => {
                    self.handle_build(*request).await;
                    Ok(())
                }
                Message::Seal(request) => {
                    self.handle_seal(*request).await;
                    Ok(())
                }
                Message::Consolidate(request) => {
                    let ConsolidateRequest { attributes, result_tx } = *request;
                    self.retry(Metrics::CONSOLIDATE_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_consolidate(attributes.clone()))
                    })
                    .await
                    .map(|recovery| {
                        // The reply's state must match the published one.
                        self.publish_state();
                        // A caller that stopped waiting does not undo the consolidation.
                        let _ = result_tx.send(Consolidation {
                            state: self.state,
                            flush: recovery == Recovery::Flushed,
                        });
                    })
                }
                Message::FollowSafeHead(safe_head) => self
                    .retry(Metrics::CONSOLIDATE_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_follow_safe_head(*safe_head))
                    })
                    .await
                    .map(|_| ()),
                Message::Finalize(block_id) => self
                    .retry(Metrics::FINALIZE_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_finalize(*block_id))
                    })
                    .await
                    .map(|_| ()),
                Message::InsertUnsafeBlock(payload) => self
                    .retry(Metrics::INSERT_TASK_LABEL, |actor| {
                        Box::pin(actor.handle_insert((*payload).clone()))
                    })
                    .await
                    .map(|_| ()),
                Message::Reset(request) => self.handle_reset(request).await,
            };
            // Publish the state even if the request failed: it can update the state before a
            // later step fails.
            self.publish_state();
            result?;
        }
    }

    /// Resets the engine and replies with the state afterwards.
    ///
    /// A request that asks for a synced execution layer is refused unless the execution layer
    /// validates the current forkchoice.
    async fn handle_reset(&mut self, request: ResetRequest) -> Result<(), ActorError> {
        let ResetRequest { only_if_el_synced, result_tx } = request;
        if only_if_el_synced && !self.el_synced().await {
            debug!(target: "engine", "Refusing a reset while the execution layer syncs");
            let _ = result_tx.send(Err(EngineClientError::ElSyncing));
            return Ok(());
        }

        warn!(target: "engine", "Received reset request");
        self.reset().await?;
        // The reply's state must match the published one.
        self.publish_state();
        let _ = result_tx.send(Ok(self.state));
        Ok(())
    }

    /// Returns whether the execution layer is synced, by sending it the current forkchoice again.
    ///
    /// Before the engine has sent a forkchoice, the execution layer is taken to be syncing. A
    /// rejected forkchoice counts as synced, because a reset replaces it.
    async fn el_synced(&self) -> bool {
        if !self.forkchoice_sent {
            return false;
        }
        match self.update_forkchoice(self.state.create_forkchoice_state()).await {
            Ok(synced) => synced,
            Err(err) if err.severity() == EngineTaskErrorSeverity::Reset => {
                warn!(target: "engine", ?err, "The execution layer rejected the forkchoice");
                true
            }
            Err(err) => {
                warn!(target: "engine", ?err, "Failed to ask whether the execution layer synced");
                false
            }
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

        let unsafe_head = self.state.unsafe_head().block_info;
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

    /// Advances the safe head with attributes derived from L1.
    ///
    /// The derivation and engine actors coordinate the safe and unsafe heads. While the unsafe head
    /// is ahead of the safe head, the unsafe block at the attributes' height becomes safe if it
    /// matches them. Otherwise the attributes are built and sealed, reorging the unsafe chain.
    ///
    /// Attributes whose parent is not the safe head were derived before a reset, and are skipped.
    /// Otherwise, the L1 block the attributes were derived from becomes the safe head's source.
    async fn handle_consolidate(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> Result<(), ConsolidateTaskError> {
        let safe_head = self.state.safe_head().block_info;
        let parent = attributes.parent.block_info;
        if parent.id() != safe_head.id() {
            warn!(
                target: "engine",
                safe_head = ?safe_head,
                parent = ?parent,
                "Skipping attributes that do not extend the safe head"
            );
            return Ok(());
        }

        if safe_head.number < self.state.unsafe_head().block_info.number {
            let block = self.fetch_unsafe_block(attributes.block_number()).await?;
            let matches = AttributesMatch::check(&self.config, &attributes, &block).is_match();
            if !matches {
                debug!(
                    target: "engine",
                    ?attributes,
                    block_hash = %block.header.hash,
                    "ConsolidateInput mismatch! Initiating reorg",
                );
            }
            // Only issue a forkchoice update if the attributes are the last in the span batch.
            // This is an optimization to avoid sending a FCU call for every block in the span
            // batch.
            if matches && self.try_make_unsafe_block_safe(block, attributes.is_last_in_span).await?
            {
                self.state = self.state.with_safe_head_source(attributes.derived_from);
                return Ok(());
            }
        }

        self.build_and_seal(attributes, true).await
    }

    /// Follows a safe head supplied by the derivation delegate.
    ///
    /// The safe head is not coordinated with the engine actor's safe and unsafe heads. If the
    /// unsafe block at its height matches it, that block becomes safe. Otherwise, including when
    /// the safe head is not behind the unsafe head, it becomes the unsafe and safe head.
    async fn handle_follow_safe_head(
        &mut self,
        safe_head: L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        if safe_head.block_info.number < self.state.unsafe_head().block_info.number {
            let block = self.fetch_unsafe_block(safe_head.block_info.number).await?;
            let matches = block.header.hash == safe_head.block_info.hash;
            if !matches {
                debug!(
                    target: "engine",
                    ?safe_head,
                    block_hash = %block.header.hash,
                    "ConsolidateInput mismatch! Initiating reorg",
                );
            }
            if matches && self.try_make_unsafe_block_safe(block, false).await? {
                return Ok(());
            }
        }

        warn!(
            target: "engine",
            safe_l2 = %safe_head,
            "Apply safe head"
        );

        // We intentionally set the unsafe head to the safe head to ensure the engine observes a
        // self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the safe head.
        self.synchronize(HeadsUpdate {
            unsafe_head: Some(safe_head),
            safe_head: Some(safe_head),
            local_safe_head: Some(safe_head),
            ..Default::default()
        })
        .await
        .inspect_err(|e| warn!(target: "engine", ?e, "Apply safe head failed"))?;

        info!(
            target: "engine",
            hash = %safe_head.block_info.hash,
            number = safe_head.block_info.number,
            "Updated safe head via follow safe"
        );

        Ok(())
    }

    /// Fetches the [`L2BlockInfo`] identified by `block_id`, makes it the finalized head, and
    /// dispatches a forkchoice update to finalize the block. A request for a block that is not
    /// safe is dropped.
    async fn handle_finalize(
        &mut self,
        block_id: FinalizeBlockId,
    ) -> Result<(), FinalizeTaskError> {
        let block_number = block_id.number();

        // Only a safe block can be finalized. A reset can move the safe head back after the
        // request was sent; finality asks again once the next L1 block finalizes.
        let safe_head = self.state.safe_head().block_info.number;
        if safe_head < block_number {
            warn!(target: "engine", block_number, safe_head, "Dropping a request to finalize a block that is not safe");
            return Ok(());
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
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.config.genesis)
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
        self.synchronize(HeadsUpdate { finalized_head: Some(block_info), ..Default::default() })
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
    /// failures. Returns how the operation completed.
    async fn retry<E>(
        &mut self,
        label: &'static str,
        mut attempt: impl for<'a> FnMut(&'a mut Self) -> Attempt<'a, E> + Send,
    ) -> Result<Recovery, ActorError>
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
                    self.reset().await?;
                    return Ok(Recovery::Reset);
                }
                EngineTaskErrorSeverity::Critical => {
                    error!(target: "engine", ?err, "Critical error performing engine operation");
                    return Err(err.into());
                }
                EngineTaskErrorSeverity::Reset => {
                    warn!(target: "engine", ?err, "Engine operation requires a reset");
                    self.reset().await?;
                    return Ok(Recovery::Reset);
                }
                EngineTaskErrorSeverity::Flush => {
                    // This error is encountered when the payload is marked INVALID by the engine
                    // api. Post-holocene, the payload is replaced by a "deposits-only" block, which
                    // completes the operation. Derivation must flush the channel and any remaining
                    // buffered batches.
                    warn!(target: "engine", ?err, "Invalid payload replaced by a deposits-only block");
                    return Ok(Recovery::Flushed);
                }
            }
        }
        Ok(Recovery::Completed)
    }

    /// Publishes the state to handles.
    ///
    /// The actor publishes after every request, and before any reply whose recipient reads the
    /// state.
    fn publish_state(&self) {
        self.state_tx.send_if_modified(|state| std::mem::replace(state, self.state) != self.state);
    }

    /// Resets the engine to a sync starting point found with [`find_starting_forkchoice`], reorgs
    /// the execution layer to it, and publishes the state and then the reset.
    ///
    /// Temporary failures are retried until the reset succeeds.
    async fn reset(&mut self) -> Result<(), ActorError> {
        let mut backoff = Backoff::default();
        loop {
            let attempt = async {
                let start =
                    find_starting_forkchoice(&self.config, self.client.as_ref(), &self.l1).await?;
                self.synchronize(HeadsUpdate {
                    unsafe_head: Some(start.un_safe),
                    local_safe_head: Some(start.safe),
                    safe_head: Some(start.safe),
                    finalized_head: Some(start.finalized),
                })
                .await?;
                Ok::<_, EngineResetError>(())
            };
            match attempt.await {
                Ok(()) => break,
                Err(err) if err.is_temporary() => {
                    warn!(target: "engine", ?err, "Reset dependency unavailable; retrying reset");
                    backoff.wait().await;
                }
                Err(err) => return Err(err.into()),
            }
        }
        // The reset's safe head was not derived from a known L1 block.
        self.state = self.state.with_safe_head_source(None);
        // Readers that follow the engine's chain start over from the reset's safe head. The reset
        // is published before the reply to the request that caused it, and after the state, so a
        // reader that sees the reset finds no older state left to read.
        self.publish_state();
        self.resets_tx.send_replace(self.state);
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);
        Ok(())
    }

    /// Applies a heads update and sends the resulting forkchoice to the execution layer with
    /// `engine_forkchoiceUpdated`, without payload attributes.
    ///
    /// Inserting, consolidating, and finalizing blocks, and resetting, all use this to move the
    /// forkchoice. Starting a build sends its own forkchoice update with payload attributes
    /// instead.
    async fn synchronize(&mut self, update: HeadsUpdate) -> Result<(), SynchronizeTaskError> {
        // Apply the update to the engine state's heads.
        let new_state = self.state.apply_update(update);

        // Check if a forkchoice update is not needed, return early.
        // A forkchoice update is not needed if...
        // 1. A forkchoice has been sent, and
        // 2. The update changes no head.
        //
        // NOTE:
        // We shouldn't retry the forkchoice update there. Since the heads are only updated
        // here (except during consolidation, when the block is not the last in the batch, and
        // the safe head's source, which is not part of the forkchoice) - the engine would get
        // stuck retrying the forkchoice update.
        if self.forkchoice_sent && self.state == new_state {
            debug!(target: "engine", ?new_state, "No forkchoice update needed");
            return Ok(());
        }

        // Check if the head is behind the finalized head.
        if new_state.unsafe_head().block_info.number < new_state.finalized_head().block_info.number
        {
            return Err(SynchronizeTaskError::FinalizedAheadOfUnsafe(
                new_state.unsafe_head().block_info.number,
                new_state.finalized_head().block_info.number,
            ));
        }

        let forkchoice = new_state.create_forkchoice_state();
        self.update_forkchoice(forkchoice).await?;
        self.state = new_state;
        self.forkchoice_sent = true;
        Ok(())
    }

    /// Sends the forkchoice to the execution layer with `engine_forkchoiceUpdated`, without payload
    /// attributes. Returns whether the execution layer validated it, rather than still syncing.
    async fn update_forkchoice(
        &self,
        forkchoice: ForkchoiceState,
    ) -> Result<bool, SynchronizeTaskError> {
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

        let synced = match &valid_response.payload_status.status {
            PayloadStatusEnum::Valid => true,
            PayloadStatusEnum::Syncing => {
                // If we're not building a new payload, we're driving EL sync.
                debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
                false
            }
            status => {
                // Other codes are not expected.
                return Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone()));
            }
        };

        debug!(
            target: "engine",
            forkchoice = ?forkchoice,
            response = ?valid_response,
            "Forkchoice updated"
        );

        Ok(synced)
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
        if self.state.unsafe_head().block_info.number <
            self.state.finalized_head().block_info.number
        {
            return Err(BuildTaskError::EngineBuildError(
                EngineBuildError::FinalizedAheadOfUnsafe(
                    self.state.unsafe_head().block_info.number,
                    self.state.finalized_head().block_info.number,
                ),
            ));
        }

        // When inserting a payload, we advertise the parent's unsafe head as the current unsafe
        // head to build on top of.
        let new_forkchoice = self
            .state
            .apply_update(HeadsUpdate {
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
        self.synchronize(HeadsUpdate {
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
                if is_attributes_derived {
                    // A deposits-only replacement is derived from the same L1 block.
                    self.state = self.state.with_safe_head_source(attributes.derived_from);
                }
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

    /// Fetches the unsafe L2 block at `number`, to compare against a safe head.
    async fn fetch_unsafe_block(
        &self,
        number: u64,
    ) -> Result<RpcBlock<Transaction>, ConsolidateTaskError> {
        match self.client.l2_block_by_label(number.into()).await {
            Ok(Some(block)) => Ok(block),
            Ok(None) => {
                warn!(target: "engine", "Received `None` block for {}", number);
                Err(ConsolidateTaskError::MissingUnsafeL2Block(number))
            }
            Err(_) => {
                warn!(target: "engine", "Failed to fetch unsafe l2 block for consolidation");
                Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block)
            }
        }
    }

    /// Promotes an unsafe block that matches the safe head to safe, and hands it to the block
    /// sink.
    ///
    /// Sends a forkchoice update only if `send_forkchoice` is set; otherwise the safe head is
    /// updated in the engine state alone. Returns `false`, changing nothing, if the block's info
    /// cannot be constructed.
    async fn try_make_unsafe_block_safe(
        &mut self,
        block: RpcBlock<Transaction>,
        send_forkchoice: bool,
    ) -> Result<bool, ConsolidateTaskError> {
        trace!(
            target: "engine",
            block_hash = %block.header.hash,
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
                warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, proceeding to build");
                return Ok(false);
            }
        };

        // The next attributes built are this block's child, and ask for its config.
        self.block_sink.block_imported(consensus_block, block_info);

        let update = HeadsUpdate {
            safe_head: Some(block_info),
            local_safe_head: Some(block_info),
            ..Default::default()
        };
        if send_forkchoice {
            self.synchronize(update)
                .await
                .inspect_err(|e| warn!(target: "engine", ?e, "Consolidation failed"))?;
        } else {
            // Apply a transient update to the safe head.
            self.state = self.state.apply_update(update);
        }

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Updated safe head via L1 consolidation"
        );

        Ok(true)
    }
}

/// How an operation completed after [`Actor::retry`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Recovery {
    /// The operation succeeded.
    Completed,
    /// The execution layer rejected the operation's block, and a deposits-only block replaced it.
    Flushed,
    /// The operation was abandoned and the engine reset, publishing the reset.
    Reset,
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
