//! Sequencer construction and execution.

use super::Capacity;
use crate::{
    SequencerEngineClient, UnsafePayloadGossipClient,
    engine::EngineClientError,
    sequencer::{
        Handle, HandleError, State,
        conductor::Conductor,
        error::ActorError,
        handle::Message,
        metrics::{
            update_attributes_build_duration_metrics, update_block_build_duration_metrics,
            update_conductor_commitment_duration_metrics, update_seal_duration_metrics,
            update_total_transactions_sequenced,
        },
        origin_selector::{L1OriginSelectorError, OriginSelector},
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::{InsertTaskError, SealTaskError, SynchronizeTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::{
    future::{Future, pending},
    sync::Arc,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};
use tokio::{
    select,
    sync::{mpsc, watch},
    time::Interval,
};

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder<Conductor_> {
    handle: Handle,
    messages: mpsc::Receiver<Message>,
    published: watch::Sender<State>,
    conductor: Option<Conductor_>,
}

impl<Conductor_: Conductor + 'static> Builder<Conductor_> {
    /// Creates the builder with its initial state.
    pub fn new(
        capacity: Capacity,
        conductor: Option<Conductor_>,
        is_active: bool,
        in_recovery_mode: bool,
    ) -> Self {
        let (messages_tx, messages) = mpsc::channel(capacity.get());
        let (published, state) = watch::channel(State {
            active: is_active,
            conductor_enabled: conductor.is_some(),
            recovery_mode: in_recovery_mode,
        });
        Self { handle: Handle::new(state, messages_tx), messages, published, conductor }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor.
    pub fn build<
        AttributesBuilder_: AttributesBuilder + Sync + 'static,
        OriginSelector_: OriginSelector + 'static,
        SequencerEngineClient_: SequencerEngineClient + 'static,
        UnsafePayloadGossipClient_: UnsafePayloadGossipClient + Sync + 'static,
    >(
        self,
        attributes_builder: AttributesBuilder_,
        engine_client: SequencerEngineClient_,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages, published, conductor } = self;
        let state = *published.borrow();
        async move {
            Actor::new(
                messages,
                published,
                state,
                attributes_builder,
                conductor,
                engine_client,
                origin_selector,
                rollup_config,
                unsafe_payload_gossip_client,
            )
            .run()
            .await
        }
    }
}

/// The handle to a block that has been started but not sealed.
#[derive(Debug)]
struct UnsealedPayloadHandle {
    /// The [`PayloadId`] of the unsealed payload.
    payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] used to start block building.
    attributes_with_parent: OpAttributesWithParent,
}

/// The return payload of the `seal_last_and_start_next` function. This allows the sequencer
/// to make an informed decision about when to seal and build the next block.
#[derive(Debug)]
struct SealLastStartNextResult {
    /// The [`UnsealedPayloadHandle`] that was built.
    unsealed_payload_handle: Option<UnsealedPayloadHandle>,
    /// How long it took to execute the seal operation.
    seal_duration: Duration,
}

/// The [`Actor`] is responsible for building L2 blocks on top of the current unsafe head
/// and handing them to the signer through [`signer::Handle`](crate::signer::Handle) to be signed
/// and gossipped, extending the L2 chain with new blocks.
#[derive(Debug)]
struct Actor<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Receives messages from handles.
    messages: mpsc::Receiver<Message>,
    /// Sequencer state shared with admin RPC readers.
    state: State,
    /// Publishes state to handle readers.
    published: watch::Sender<State>,
    /// The attributes builder used for block building.
    attributes_builder: AttributesBuilder_,
    /// The optional conductor RPC client.
    conductor: Option<Conductor_>,
    /// The struct used to interact with the engine.
    engine_client: SequencerEngineClient_,
    /// The struct used to determine the next L1 origin.
    origin_selector: OriginSelector_,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// A client that hands built payloads to the signer actor, which signs them for the network
    /// actor to gossip.
    unsafe_payload_gossip_client: UnsafePayloadGossipClient_,

    /// Ticker that paces block-building attempts.
    build_ticker: Interval,
    /// The handle for the payload built on the previous tick that is waiting to be sealed.
    next_payload_to_seal: Option<UnsealedPayloadHandle>,
    /// Duration of the most recent seal operation, used to back-pressure the build ticker.
    last_seal_duration: Duration,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    Actor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    #[allow(clippy::too_many_arguments)]
    fn new(
        messages: mpsc::Receiver<Message>,
        published: watch::Sender<State>,
        state: State,
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        let build_ticker = tokio::time::interval(Duration::from_secs(rollup_config.block_time));
        Self {
            messages,
            state,
            published,
            attributes_builder,
            conductor,
            engine_client,
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
            build_ticker,
            next_payload_to_seal: None,
            last_seal_duration: Duration::ZERO,
        }
    }

    /// Publishes sequencer state and updates its metrics.
    fn update_state(&self) {
        let state_flags = [
            ("active", self.state.active.to_string()),
            ("recovery", self.state.recovery_mode.to_string()),
        ];
        metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags).set(1);
        self.published.send_replace(self.state);
    }

    /// Seals and commits the last pending block, if one exists and starts the build job for the
    /// next L2 block, on top of the current unsafe head.
    ///
    /// If a new block was started, it will return the associated [`UnsealedPayloadHandle`] so
    /// that it may be sealed and committed in a future call to this function.
    async fn seal_last_and_start_next(
        &mut self,
        payload_to_seal: Option<&UnsealedPayloadHandle>,
    ) -> Result<SealLastStartNextResult, ActorError> {
        let seal_duration = match payload_to_seal {
            Some(to_seal) => {
                let seal_start = Instant::now();
                self.seal_and_commit_payload_if_applicable(to_seal).await?;
                seal_start.elapsed()
            }
            None => Duration::default(),
        };

        let unsealed_payload_handle = self.build_unsealed_payload().await?;

        Ok(SealLastStartNextResult { unsealed_payload_handle, seal_duration })
    }

    /// Sends a seal request to seal the provided [`UnsealedPayloadHandle`], committing and
    /// gossiping the resulting block, if one is built.
    async fn seal_and_commit_payload_if_applicable(
        &self,
        unsealed_payload_handle: &UnsealedPayloadHandle,
    ) -> Result<(), ActorError> {
        let seal_request_start = Instant::now();

        // Send the seal request to the engine to seal the unsealed block.
        let payload = self
            .engine_client
            .seal_and_canonicalize_block(
                unsealed_payload_handle.payload_id,
                unsealed_payload_handle.attributes_with_parent.clone(),
            )
            .await?;

        update_seal_duration_metrics(seal_request_start.elapsed());

        let payload_transaction_count =
            unsealed_payload_handle.attributes_with_parent.count_transactions();
        update_total_transactions_sequenced(payload_transaction_count);

        // If the conductor is available, commit the payload to it.
        if let Some(conductor) = &self.conductor {
            let _conductor_commitment_start = Instant::now();
            if let Err(err) = conductor.commit_unsafe_payload(&payload).await {
                error!(target: "sequencer", ?err, "Failed to commit unsafe payload to conductor");
            }

            update_conductor_commitment_duration_metrics(_conductor_commitment_start.elapsed());
        }

        self.unsafe_payload_gossip_client
            .schedule_execution_payload_gossip(payload)
            .await
            .map_err(Into::into)
    }

    /// Starts building an L2 block by creating and populating payload attributes referencing the
    /// correct L1 origin block and sending them to the block engine.
    async fn build_unsealed_payload(
        &mut self,
    ) -> Result<Option<UnsealedPayloadHandle>, ActorError> {
        let unsafe_head = self.engine_client.get_unsafe_head().await?;

        let Some(l1_origin) = self.get_next_payload_l1_origin(unsafe_head).await? else {
            // Temporary error - retry on next tick.
            return Ok(None);
        };

        info!(
            target: "sequencer",
            parent_num = unsafe_head.block_info.number,
            l1_origin_num = l1_origin.number,
            "Started sequencing new block"
        );

        // Build the payload attributes for the next block.
        let attributes_build_start = Instant::now();

        let Some(attributes_with_parent) = self.build_attributes(unsafe_head, l1_origin).await?
        else {
            // Temporary error or reset - retry on next tick.
            return Ok(None);
        };

        update_attributes_build_duration_metrics(attributes_build_start.elapsed());

        // Send the built attributes to the engine to be built.
        let build_request_start = Instant::now();

        let payload_id =
            self.engine_client.start_build_block(attributes_with_parent.clone()).await?;

        update_block_build_duration_metrics(build_request_start.elapsed());

        Ok(Some(UnsealedPayloadHandle { payload_id, attributes_with_parent }))
    }

    /// Determines and validates the L1 origin block for the provided L2 unsafe head.
    /// Returns `Ok(None)` for temporary errors that should be retried.
    async fn get_next_payload_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
    ) -> Result<Option<BlockInfo>, ActorError> {
        let recovery_mode = self.state.recovery_mode;
        let l1_origin = match self.origin_selector.next_l1_origin(unsafe_head, recovery_mode).await
        {
            Ok(l1_origin) => l1_origin,
            Err(L1OriginSelectorError::OriginNotFound(hash)) => {
                warn!(
                    target: "sequencer",
                    %hash,
                    "L1 origin block not found, resetting engine"
                );
                self.engine_client.reset_engine_forkchoice().await?;
                return Ok(None);
            }
            Err(err) => {
                warn!(
                    target: "sequencer",
                    ?err,
                    "Temporary error occurred while selecting next L1 origin. Re-attempting on next tick."
                );
                return Ok(None);
            }
        };

        if unsafe_head.l1_origin.hash != l1_origin.parent_hash &&
            unsafe_head.l1_origin.hash != l1_origin.hash
        {
            warn!(
                target: "sequencer",
                l1_origin = ?l1_origin,
                unsafe_head_hash = %unsafe_head.l1_origin.hash,
                unsafe_head_l1_origin = ?unsafe_head.l1_origin,
                "Cannot build new L2 block on inconsistent L1 origin, resetting engine"
            );
            self.engine_client.reset_engine_forkchoice().await?;
            return Ok(None);
        }
        Ok(Some(l1_origin))
    }

    /// Builds the `OpAttributesWithParent` for the next block to build. If None is returned, it
    /// indicates that no attributes could be built at this time but future attempts may be made.
    async fn build_attributes(
        &mut self,
        unsafe_head: L2BlockInfo,
        l1_origin: BlockInfo,
    ) -> Result<Option<OpAttributesWithParent>, ActorError> {
        let mut attributes = match self
            .attributes_builder
            .prepare_payload_attributes(unsafe_head, l1_origin.id())
            .await
        {
            Ok(attrs) => attrs,
            Err(PipelineErrorKind::Temporary(_)) => {
                // Temporary error - retry on next tick.
                return Ok(None);
            }
            Err(PipelineErrorKind::Reset(_)) => {
                if let Err(err) = self.engine_client.reset_engine_forkchoice().await {
                    error!(target: "sequencer", ?err, "Failed to reset engine");
                    return Err(ActorError::ChannelClosed);
                }

                warn!(
                    target: "sequencer",
                    "Resetting engine due to pipeline error while preparing payload attributes"
                );
                return Ok(None);
            }
            Err(err @ PipelineErrorKind::Critical(_)) => {
                error!(target: "sequencer", ?err, "Failed to prepare payload attributes");
                return Err(err.into());
            }
        };

        attributes.no_tx_pool = Some(!self.should_use_tx_pool(l1_origin, &attributes));

        let attrs_with_parent = OpAttributesWithParent::new(attributes, unsafe_head, None, false);
        Ok(Some(attrs_with_parent))
    }

    /// Determines, for the provided L1 origin block and payload attributes being constructed, if
    /// transaction pool transactions should be enabled.
    fn should_use_tx_pool(&self, l1_origin: BlockInfo, attributes: &OpPayloadAttributes) -> bool {
        if self.state.recovery_mode {
            warn!(target: "sequencer", "Sequencer is in recovery mode, producing empty block");
            return false;
        }

        // If the next L2 block is beyond the sequencer drift threshold, we must produce an empty
        // block.
        if attributes.payload_attributes.timestamp >
            l1_origin.timestamp + self.rollup_config.max_sequencer_drift(l1_origin.timestamp)
        {
            return false;
        }

        // Do not include transactions in the first Ecotone block.
        if self.rollup_config.is_first_ecotone_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing ecotone upgrade block");
            return false;
        }

        // Do not include transactions in the first Fjord block.
        if self.rollup_config.is_first_fjord_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing fjord upgrade block");
            return false;
        }

        // Do not include transactions in the first Granite block.
        if self.rollup_config.is_first_granite_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing granite upgrade block");
            return false;
        }

        // Do not include transactions in the first Holocene block.
        if self.rollup_config.is_first_holocene_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing holocene upgrade block");
            return false;
        }

        // Do not include transactions in the first Isthmus block.
        if self.rollup_config.is_first_isthmus_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing isthmus upgrade block");
            return false;
        }

        // Do not include transactions in the first Jovian block.
        // See: `<https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/derivation.md#activation-block-rules>`
        if self.rollup_config.is_first_jovian_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing jovian upgrade block");
            return false;
        }

        // Do not include transactions in the first Karst block.
        // See: `<https://github.com/ethereum-optimism/specs/tree/main/specs/protocol/karst>`
        if self.rollup_config.is_first_karst_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing karst upgrade block");
            return false;
        }

        // Do not include transactions in the first Lagoon block.
        if self.rollup_config.is_first_interop_block(attributes.payload_attributes.timestamp) {
            info!(target: "sequencer", "Sequencing lagoon upgrade block");
            return false;
        }

        // Transaction pool transactions are enabled if none of the reasons to disable are satisfied
        // above.
        true
    }

    /// Schedules the initial engine reset request and waits for the unsafe head to be updated.
    async fn schedule_initial_reset(&self) -> Result<(), ActorError> {
        // Reset the engine, in order to initialize the engine state.
        // NB: this call waits for confirmation that the reset succeeded and we can proceed with
        // post-reset logic.
        self.engine_client.reset_engine_forkchoice().await.map_err(|err| {
            error!(target: "sequencer", ?err, "Failed to send reset request to engine");
            err.into()
        })
    }
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    Actor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder + Sync + 'static,
    Conductor_: Conductor + Sync + 'static,
    OriginSelector_: OriginSelector + Sync + 'static,
    SequencerEngineClient_: SequencerEngineClient + Sync + 'static,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient + Sync + 'static,
{
    async fn run(mut self) -> Result<(), ActorError> {
        // Publish the initial state and metrics before beginning block building.
        self.update_state();
        // Reset the engine state prior to beginning block building.
        self.schedule_initial_reset().await?;
        loop {
            select! {
                // Prioritize admin messages over block building.
                biased;
                Some(message) = self.messages.recv() => self.handle_message(message).await,
                _ = self.build_ticker.tick(), if self.state.active => self.build().await?,
                // A stopped actor with no command handles stays pending until dropped.
                else => pending().await,
            }
        }
    }

    async fn handle_message(&mut self, message: Message) {
        // A dropped RPC response receiver does not cancel an accepted command.
        match message {
            Message::StartSequencer(tx) => {
                self.state.active = true;
                self.update_state();
                let _ = tx.send(Ok(()));
            }
            Message::StopSequencer(tx) => {
                // Publish before awaiting the unsafe head: sequencing is stopped even if that read
                // fails.
                self.state.active = false;
                self.update_state();
                let result =
                    self.engine_client.get_unsafe_head().await.map(|h| h.hash()).map_err(|_| {
                        HandleError::ErrorAfterSequencerWasStopped(
                            "current unsafe hash is unavailable.".to_string(),
                        )
                    });
                let _ = tx.send(result);
            }
            Message::SetRecoveryMode(mode, tx) => {
                self.state.recovery_mode = mode;
                self.update_state();
                let _ = tx.send(Ok(()));
            }
            Message::OverrideLeader(tx) => {
                let result = match self.conductor.as_mut() {
                    Some(conductor) => conductor
                        .override_leader()
                        .await
                        .map_err(|e| HandleError::LeaderOverrideError(e.to_string())),
                    None => {
                        Err(HandleError::LeaderOverrideError("No conductor configured".to_string()))
                    }
                };
                let _ = tx.send(result);
            }
        }
    }

    async fn build(&mut self) -> Result<(), ActorError> {
        if !self.unsafe_payload_gossip_client.has_capacity() {
            warn!(target: "sequencer", "Sequencing tick, gossip queue full, not building a block");
            return Ok(());
        }
        info!(target: "sequencer", "Sequencing tick, building block");
        // Move the pending payload out of self so the &mut self call below doesn't conflict
        // with the &self read of self.next_payload_to_seal.
        let pending = self.next_payload_to_seal.take();
        match self.seal_last_and_start_next(pending.as_ref()).await {
            Ok(res) => {
                self.next_payload_to_seal = res.unsealed_payload_handle;
                self.last_seal_duration = res.seal_duration;
            }
            Err(ActorError::EngineError(EngineClientError::SealError(err))) => {
                if is_seal_task_err_fatal(&err) {
                    error!(target: "sequencer", err=?err, "Critical seal task error occurred");
                    return Err(ActorError::EngineError(EngineClientError::SealError(err)));
                }
                self.next_payload_to_seal = None;
            }
            Err(other_err) => {
                error!(target: "sequencer", err = ?other_err, "Unexpected error building or sealing payload");
                return Err(other_err);
            }
        }

        if let Some(payload) = self.next_payload_to_seal.as_ref() {
            let next_block_seconds = payload
                .attributes_with_parent
                .parent()
                .block_info
                .timestamp
                .saturating_add(self.rollup_config.block_time);
            // next block time is last + block_time - time it takes to seal.
            let next_block_time =
                UNIX_EPOCH + Duration::from_secs(next_block_seconds) - self.last_seal_duration;
            match next_block_time.duration_since(SystemTime::now()) {
                Ok(duration) => self.build_ticker.reset_after(duration),
                Err(_) => self.build_ticker.reset_immediately(),
            };
        } else {
            self.build_ticker.reset_immediately();
        }
        Ok(())
    }
}

// Determines whether the provided [`SealTaskError`] is fatal for the sequencer.
//
// NB: We could use `err.severity()`, but that gives EngineActor control over this classification.
// `Actor` may have different interpretations of severity, and it is not clear when making
// a change in that area of the codebase that it will affect this area. When a new task error is
// added, this approach guarantees compilation will fail until it is handled here.
fn is_seal_task_err_fatal(err: &SealTaskError) -> bool {
    match err {
        SealTaskError::PayloadInsertionFailed(insert_err) => match &**insert_err {
            InsertTaskError::ForkchoiceUpdateFailed(synchronize_error) => match synchronize_error {
                SynchronizeTaskError::FinalizedAheadOfUnsafe(_, _) => true,
                SynchronizeTaskError::ForkchoiceUpdateFailed(_) |
                SynchronizeTaskError::InvalidForkchoiceState |
                SynchronizeTaskError::UnexpectedPayloadStatus(_) => false,
            },
            InsertTaskError::FromBlockError(_) | InsertTaskError::L2BlockInfoConstruction(_) => {
                true
            }
            InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => false,
        },
        SealTaskError::GetPayloadFailed(_) |
        SealTaskError::HoloceneInvalidFlush |
        SealTaskError::UnsafeHeadChangedSinceBuild => false,
        SealTaskError::DepositOnlyPayloadFailed |
        SealTaskError::DepositOnlyPayloadReattemptFailed |
        SealTaskError::MpscSend(_) |
        SealTaskError::ClockWentBackwards => true,
    }
}

#[cfg(test)]
mod tests;
