//! Engine construction and execution.

use super::{
    ActorError, BuildTaskError, ConsolidateTaskError, Derived, EngineBuildError, EngineResetError,
    EngineState, EngineTaskError, EngineTaskErrorSeverity, Finality, FinalizeTaskError, Handle,
    HeadsUpdate, ImportedBlockSink, InsertTaskError, Safe, Safety, SealTaskError,
    SynchronizeTaskError,
    handle::{BuildRequest, CanonicalizeRequest, Message, SealRequest},
    sync::find_starting_forkchoice,
};
use crate::{Metrics, l1};
use alloy_eips::BlockNumHash;
use alloy_rpc_types_engine::{
    ForkchoiceState, INVALID_FORK_CHOICE_STATE_ERROR, PayloadId, PayloadStatusEnum,
};
use alloy_rpc_types_eth::Block as RpcBlock;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_sources::EngineApi;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types::Transaction;
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
    /// Creates the builder with the engine starting from `state`, the execution layer's heads as
    /// [`execution_layer_state`](super::execution_layer_state) reads them. Its handles read those
    /// heads until the engine's first reset replaces them.
    ///
    /// The request queue holds one request: a sender waits for room in it, and the network drops
    /// the unsafe blocks it receives over gossip while it is full.
    pub fn new(state: EngineState) -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(1);
        let (state_tx, state_rx) = watch::channel(state);
        Self { handle: Handle::new(messages_tx, state_rx), messages_rx, state_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// From its first reset, the engine makes safe what `safety` decides, and finalizes what
    /// `finality` decides.
    ///
    /// The caller must retain a handle to keep the request channel open.
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor.
    pub fn build<S: Safety, F: Finality>(
        self,
        client: Arc<EngineApi>,
        l1: l1::Handle,
        rollup: Arc<RollupConfig>,
        block_sink: Arc<dyn ImportedBlockSink>,
        safety: S,
        finality: F,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages_rx, state_tx } = self;
        let dependencies =
            Dependencies { client, l1, config: rollup, block_sink, safety, finality };
        Actor::new(dependencies, state_tx, messages_rx).run()
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

/// What the engine actor talks to.
#[derive(Debug)]
struct Dependencies<S, F> {
    /// The Engine API client.
    client: Arc<EngineApi>,
    /// The L1 chain.
    l1: l1::Handle,
    /// The rollup configuration.
    config: Arc<RollupConfig>,
    /// Where to hand every block the engine makes canonical.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// What decides the safe head.
    safety: S,
    /// What decides the finalized head.
    finality: F,
}

/// Performs the operations sent to the execution layer's Engine API, one request at a time in the
/// order they arrive, and publishes the engine state after each request and before replying.
///
/// Builds and seals reply to their caller, which decides whether to retry. Other operations retry
/// temporary failures in place with backoff, so later requests wait in the handle's bounded queue.
/// An operation that keeps failing for [`TASK_RETRY_TIMEOUT`] is abandoned and the engine is
/// reset; a reset retries until it succeeds.
///
/// The engine resets itself when it starts, and from then on follows L1: whenever the L1 head or
/// the L1 blocks its heads rest on change, it resets if L1 reorged out one of those blocks. It
/// drives safety and finality once the execution layer has synced.
#[derive(Debug)]
struct Actor<S, F> {
    /// The Engine API client.
    client: Arc<EngineApi>,
    /// The L1 chain, which a reset checks to find a plausible forkchoice, and which the engine
    /// follows for reorgs.
    l1: l1::Handle,
    /// The rollup configuration.
    config: Arc<RollupConfig>,
    /// Where to hand every block the engine makes canonical.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// What decides the safe head, which the engine drives once the execution layer has synced.
    safety: S,
    /// What decides the finalized head, which the engine drives once the execution layer has
    /// synced.
    finality: F,
    /// The state of the engine. Operations update it, and it is published after every request.
    state: EngineState,
    /// Whether the engine has sent the execution layer a forkchoice. It starts from the execution
    /// layer's own heads, which it has not chosen until it sends them.
    forkchoice_sent: bool,
    /// Whether the engine drives safety and finality yet.
    deriving: Deriving,
    /// The latest L1 head the engine checked its heads against, and the L1 blocks they rested on
    /// then. Once its heads rest on other blocks, the engine checks them again.
    l1_checked: Option<L1Checked>,
    /// Publishes the state to handles.
    state_tx: watch::Sender<EngineState>,
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
}

impl<S: Safety, F: Finality> Actor<S, F> {
    /// Creates the actor, starting from the state that `state_tx` holds.
    fn new(
        dependencies: Dependencies<S, F>,
        state_tx: watch::Sender<EngineState>,
        messages_rx: mpsc::Receiver<Message>,
    ) -> Self {
        let Dependencies { client, l1, config, block_sink, safety, finality } = dependencies;
        let state = *state_tx.borrow();
        Self {
            client,
            l1,
            config,
            block_sink,
            safety,
            finality,
            state,
            forkchoice_sent: false,
            deriving: Deriving::NotStarted,
            l1_checked: None,
            state_tx,
            messages_rx,
        }
    }

    /// Resets the engine, which checks the execution layer's heads against L1, and then serves.
    ///
    /// An execution layer that has finalized a block past genesis is not doing its first sync, so
    /// safety and finality start at once. Otherwise they start once the execution layer has synced
    /// past the reset's unsafe head: until then, deriving would build blocks the execution layer is
    /// still syncing, and move its sync target.
    async fn run(mut self) -> Result<(), ActorError> {
        let first_sync =
            self.state.finalized_head().block_info.number <= self.config.genesis.l2.number;
        self.reset().await?;
        self.deriving = if first_sync {
            info!(target: "engine", "Waiting for the execution layer's first sync before deriving");
            Deriving::AfterSyncPast(self.state.unsafe_head().block_info.number)
        } else {
            Deriving::Synced
        };
        self.serve().await
    }

    /// Serves requests, checks L1, and drives safety and finality once they start, for as long as
    /// the actor runs.
    async fn serve(mut self) -> Result<(), ActorError> {
        loop {
            if self.deriving == Deriving::Synced {
                info!(target: "engine", "The execution layer synced; deriving");
                self.deriving = Deriving::Running;
                self.safety
                    .reset(self.state.safe_head())
                    .await
                    .map_err(|_| ActorError::SafetyStopped)?;
            }
            let safe = self.safe();
            let deriving = self.deriving == Deriving::Running;
            tokio::select! {
                // Check L1 before more requests, which could otherwise delay it for as long as
                // they keep coming.
                biased;
                checked = Self::next_l1_check(&self.l1, self.state, self.l1_checked) => {
                    let L1Check { head, reorged } = checked.map_err(|_| ActorError::L1Stopped)?;
                    if let Some(block) = reorged {
                        warn!(target: "engine", ?block, l1_head = ?head.id(), "L1 reorged out a block the engine's heads rest on; resetting");
                        self.reset().await?;
                    }
                    self.l1_checked = Some(L1Checked { head, rested_on: rests_on(&self.state) });
                }
                message = self.messages_rx.recv() => {
                    let message = message.ok_or(ActorError::ChannelClosed)?;
                    self.handle_message(message).await?;
                }
                derived = self.safety.next(self.state.safe_head()), if deriving => {
                    let derived = derived.map_err(|_| ActorError::SafetyStopped)?;
                    let result = self.adopt(derived).await;
                    // Publish the state even if adopting failed: it can update the state before a
                    // later step fails.
                    self.publish_state();
                    result?;
                }
                // Finality learns each safe head whose source the engine knows: none since a
                // reset until the engine makes another block safe.
                finalized = self.finality.next(safe), if deriving && self.state.safe_head_source().is_some() => {
                    let finalized = finalized.map_err(|_| ActorError::FinalityStopped)?;
                    let result = self.adopt_finalized(finalized).await;
                    self.publish_state();
                    result?;
                }
            }
        }
    }

    /// Returns the safe head and the L1 block it was derived from, or the default block where that
    /// is unknown.
    fn safe(&self) -> Safe {
        Safe {
            head: self.state.safe_head(),
            source: self.state.safe_head_source().unwrap_or_default(),
        }
    }

    /// Makes safe what safety decided: builds the next safe block from attributes, or makes a
    /// block the execution layer has safe.
    async fn adopt(&mut self, derived: Derived) -> Result<(), ActorError> {
        match derived {
            Derived::Attributes(attributes) => {
                let recovery = self
                    .retry(Metrics::CONSOLIDATE_TASK_LABEL, |actor| {
                        Box::pin(actor.build_and_seal(attributes.clone(), true))
                    })
                    .await?;
                if recovery == Recovery::Flushed {
                    self.safety.flush().await.map_err(|_| ActorError::SafetyStopped)?;
                }
            }
            Derived::Safe(Safe { head, source }) => {
                if head != self.state.safe_head() {
                    let recovery = self
                        .retry(Metrics::CONSOLIDATE_TASK_LABEL, |actor| {
                            Box::pin(actor.handle_follow_safe_head(head))
                        })
                        .await?;
                    // A reset replaced the safe head.
                    if recovery == Recovery::Reset {
                        return Ok(());
                    }
                }
                self.state = self.state.with_safe_head_source(Some(source));
            }
        }
        Ok(())
    }

    /// Finalizes what finality decided, if it is past the finalized head.
    async fn adopt_finalized(&mut self, finalized: L2BlockInfo) -> Result<(), ActorError> {
        if finalized.block_info.number <= self.state.finalized_head().block_info.number {
            return Ok(());
        }
        let block = finalized.block_info.id();
        self.retry(Metrics::FINALIZE_TASK_LABEL, |actor| Box::pin(actor.handle_finalize(block)))
            .await?;
        Ok(())
    }

    /// Checks the L1 blocks the heads of `state` rest on, which [`rests_on`] lists, against the L1
    /// head: at once if they are not the blocks `checked` checked, and otherwise once the L1 head
    /// changes.
    ///
    /// A block after the L1 head cannot be checked yet; a later L1 head checks it. A block before
    /// the finalized L1 block cannot reorg any more, and is not checked again. Cancelling the check
    /// loses nothing.
    async fn next_l1_check(
        l1: &l1::Handle,
        state: EngineState,
        checked: Option<L1Checked>,
    ) -> Result<L1Check, l1::HandleError> {
        let blocks = rests_on(&state);
        let head = match checked {
            Some(checked) if checked.rested_on == blocks => l1.next_head(checked.head).await?,
            _ => l1.status()?.head,
        };
        let finalized = l1.status()?.finalized.number;
        // The L1 actor reads blocks before the finalized block over RPC.
        for block in blocks.into_iter().filter(|block| block.number >= finalized) {
            let canonical = l1.canonical(block.number).await?;
            if canonical.is_some_and(|canonical| canonical.hash != block.hash) {
                return Ok(L1Check { head, reorged: Some(block) });
            }
        }
        Ok(L1Check { head, reorged: None })
    }

    /// Handles a request, and publishes the state afterwards.
    async fn handle_message(&mut self, message: Message) -> Result<(), ActorError> {
        let result = match message {
            Message::Build(request) => {
                self.handle_build(*request).await;
                Ok(())
            }
            Message::Seal(request) => {
                self.handle_seal(*request).await;
                Ok(())
            }
            Message::Canonicalize(request) => {
                self.handle_canonicalize(*request).await;
                Ok(())
            }
            Message::InsertUnsafeBlock(payload) => self
                .retry(Metrics::INSERT_TASK_LABEL, |actor| {
                    Box::pin(actor.handle_insert((*payload).clone()))
                })
                .await
                .map(|_| ()),
            Message::Reset(result_tx) => self.handle_reset(result_tx).await,
        };
        // Publish the state even if the request failed: it can update the state before a
        // later step fails.
        self.publish_state();
        result
    }

    /// Resets the engine and replies with the state afterwards.
    async fn handle_reset(
        &mut self,
        result_tx: oneshot::Sender<EngineState>,
    ) -> Result<(), ActorError> {
        warn!(target: "engine", "Received reset request");
        self.reset().await?;
        // The reply's state must match the published one.
        self.publish_state();
        let _ = result_tx.send(self.state);
        Ok(())
    }

    /// Starts a build and replies with its payload ID or failure; the caller decides whether to
    /// retry.
    async fn handle_build(&self, request: BuildRequest) {
        let BuildRequest { attributes, result_tx } = request;
        let result = record(Metrics::BUILD_TASK_LABEL, self.start_build(attributes).await);
        // A caller that stopped waiting does not cancel the build.
        let _ = result_tx.send(result);
    }

    /// Seals a build without importing it, and replies with the payload or failure; the caller
    /// decides whether to retry, and whether the block becomes canonical.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head moved since
    /// the build started.
    async fn handle_seal(&self, request: SealRequest) {
        let SealRequest { payload_id, attributes, result_tx } = request;
        let result = match self.check_parent(&attributes) {
            Ok(()) => {
                let timestamp = attributes.attributes().payload_attributes.timestamp;
                self.client.get_payload(payload_id, timestamp).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })
            }
            Err(error) => Err(error),
        };
        // A caller that stopped waiting does not cancel the seal.
        let _ = result_tx.send(record(Metrics::SEAL_TASK_LABEL, result));
    }

    /// Imports a sealed block and makes it the unsafe head, and replies with the failure, if any;
    /// the caller decides whether to retry.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head is no longer
    /// the block's parent. Unlike a derived block, one the execution layer rejects is not replaced
    /// with a deposits-only block: the conductor has not committed the replacement. The sequencer
    /// drops the block and builds another at its height.
    async fn handle_canonicalize(&mut self, request: CanonicalizeRequest) {
        let CanonicalizeRequest { payload, attributes, result_tx } = request;
        let result = match self.check_parent(&attributes) {
            Ok(()) => self
                .insert_payload(payload, false)
                .await
                .map(|block| {
                    info!(
                        target: "engine",
                        l2_number = block.block_info.number,
                        l2_time = block.block_info.timestamp,
                        "Built and imported new unsafe block"
                    );
                })
                .map_err(|e| SealTaskError::PayloadInsertionFailed(Box::new(e))),
            Err(error) => Err(error),
        };
        // The sequencer builds the next block on the unsafe head as soon as it gets the reply.
        self.publish_state();
        // A caller that stopped waiting does not cancel the import.
        let _ = result_tx.send(record(Metrics::CANONICALIZE_TASK_LABEL, result));
    }

    /// Checks that the attributes' parent is still the unsafe head.
    fn check_parent(&self, attributes: &OpAttributesWithParent) -> Result<(), SealTaskError> {
        let unsafe_head = self.state.unsafe_head().block_info;
        let parent = attributes.parent.block_info;
        if unsafe_head.id() == parent.id() {
            return Ok(());
        }
        info!(
            target: "engine",
            unsafe_block_info = ?unsafe_head,
            parent_block_info = ?parent,
            "Attributes parent does not match unsafe head, returning rebuild error"
        );
        Err(SealTaskError::UnsafeHeadChangedSinceBuild)
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

    /// Makes safe a block safety reported: one derivation matched against the unsafe chain, or a
    /// trusted source's safe head.
    ///
    /// If the unsafe block at its height is that block, it becomes safe, with a forkchoice update.
    /// Otherwise, including when the safe head is ahead of the unsafe head, it becomes the unsafe
    /// and safe head.
    async fn handle_follow_safe_head(
        &mut self,
        safe_head: L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        // The unsafe head itself went to the block sink when it was inserted.
        if safe_head == self.state.unsafe_head() {
            let update = HeadsUpdate {
                safe_head: Some(safe_head),
                local_safe_head: Some(safe_head),
                ..Default::default()
            };
            return Ok(self.synchronize(update).await?);
        }
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
            if matches && self.try_make_unsafe_block_safe(block).await? {
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

    /// Makes the L2 block `block` the finalized head, and dispatches a forkchoice update to
    /// finalize it. A block that is not safe, or that the execution layer's canonical chain does
    /// not hold, such as one finality decided on before a reset replaced it, is not finalized.
    async fn handle_finalize(&mut self, block: BlockNumHash) -> Result<(), FinalizeTaskError> {
        // Only a safe block can be finalized. A reset can move the safe head back; finality
        // decides again from the safe heads after it.
        let safe_head = self.state.safe_head().block_info.number;
        if safe_head < block.number {
            warn!(target: "engine", block_number = block.number, safe_head, "Not finalizing a block that is not safe");
            return Ok(());
        }

        // The canonical block at a safe height is an ancestor of the safe head.
        let canonical = self
            .client
            .get_l2_block(block.number.into())
            .full()
            .await
            .map_err(FinalizeTaskError::TransportError)?
            .ok_or(FinalizeTaskError::BlockNotFound(block.number))?
            .into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis(&canonical, &self.config.genesis)
            .map_err(FinalizeTaskError::FromBlock)?;
        if block_info.block_info.hash != block.hash {
            warn!(target: "engine", ?block, canonical = ?block_info.block_info.id(), "Not finalizing a block the canonical chain does not hold");
            return Ok(());
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
        self.publish_state();
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);
        // Safety starts over from the reset's safe head, if it started.
        if self.deriving == Deriving::Running {
            self.safety
                .reset(self.state.safe_head())
                .await
                .map_err(|_| ActorError::SafetyStopped)?;
        }
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
        let valid = self.update_forkchoice(forkchoice).await?;
        self.state = new_state;
        self.forkchoice_sent = true;
        if let Deriving::AfterSyncPast(number) = self.deriving &&
            valid &&
            new_state.unsafe_head().block_info.number > number
        {
            self.deriving = Deriving::Synced;
        }
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

    /// Promotes an unsafe block that matches the safe head to safe with a forkchoice update, and
    /// hands it to the block sink. Returns `false`, changing nothing, if the block's info cannot be
    /// constructed.
    async fn try_make_unsafe_block_safe(
        &mut self,
        block: RpcBlock<Transaction>,
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
                warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, not making it safe");
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
        self.synchronize(update)
            .await
            .inspect_err(|e| warn!(target: "engine", ?e, "Making the unsafe block safe failed"))?;

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Made the unsafe block safe"
        );

        Ok(true)
    }
}

/// Whether the engine drives safety and finality yet.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Deriving {
    /// Not before the engine's startup reset.
    NotStarted,
    /// Not until the execution layer validates a forkchoice whose unsafe head is past this block
    /// number: it is doing its first sync.
    AfterSyncPast(u64),
    /// The execution layer synced, so they start next.
    Synced,
    /// The engine drives them.
    Running,
}

/// Returns the L1 blocks the heads of `state` rest on: the unsafe head's L1 origin, and the L1
/// block the safe head was derived from, or the safe head's L1 origin where that is unknown.
fn rests_on(state: &EngineState) -> [BlockNumHash; 2] {
    let safe =
        state.safe_head_source().map_or_else(|| state.safe_head().l1_origin, |source| source.id());
    [state.unsafe_head().l1_origin, safe]
}

/// An L1 head the engine checked its heads against, and the L1 blocks they rested on.
#[derive(Debug, Clone, Copy)]
struct L1Checked {
    /// The L1 head checked against.
    head: BlockInfo,
    /// The L1 blocks the heads rested on, as [`rests_on`] lists them.
    rested_on: [BlockNumHash; 2],
}

/// What checking the engine's heads against an L1 head found.
#[derive(Debug)]
struct L1Check {
    /// The L1 head checked against.
    head: BlockInfo,
    /// The block the heads rest on that L1 reorged out, if any.
    reorged: Option<BlockNumHash>,
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
