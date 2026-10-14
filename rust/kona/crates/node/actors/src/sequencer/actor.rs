//! Sequencer construction and execution.

use crate::{
    SequencerEngineClient,
    engine::{
        BuildTaskError, EngineBuildError, EngineClientError, InsertTaskError, SealTaskError,
        SynchronizeTaskError,
    },
    origin::OriginError,
    sequencer::{
        Handle, HandleError, Signer,
        conductor::Conductor,
        error::ActorError,
        handle::Message,
        metrics::{
            update_attributes_build_duration_metrics, update_block_build_duration_metrics,
            update_conductor_commitment_duration_metrics, update_seal_duration_metrics,
            update_total_transactions_sequenced,
        },
        origin::Origin,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::{
    future::{Future, pending},
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{
    select,
    sync::{mpsc, watch},
    time::{self, Instant},
};

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages_rx: mpsc::Receiver<Message>,
    is_active_tx: watch::Sender<bool>,
}

impl Builder {
    /// Creates the builder with its initial state.
    ///
    /// The command queue holds one command: each command waits for its reply anyway.
    pub fn new() -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(1);
        let (is_active_tx, is_active_rx) = watch::channel(false);
        Self { handle: Handle::new(is_active_rx, messages_tx), messages_rx, is_active_tx }
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
        Conductor_: Conductor + 'static,
        Origin_: Origin + 'static,
        SequencerEngineClient_: SequencerEngineClient + 'static,
        Signer_: Signer + 'static,
    >(
        self,
        attributes_builder: AttributesBuilder_,
        engine_client: SequencerEngineClient_,
        origin: Origin_,
        rollup_config: Arc<RollupConfig>,
        conductor: Option<Conductor_>,
        signer: Signer_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages_rx, is_active_tx } = self;
        async move {
            Actor::new(
                messages_rx,
                is_active_tx,
                attributes_builder,
                conductor,
                engine_client,
                origin,
                rollup_config,
                signer,
            )
            .run()
            .await
        }
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}

/// The handle to a block that has been started but not sealed.
#[derive(Debug, PartialEq, Eq)]
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
/// and handing them to the [`Signer`] to be signed
/// and gossipped, extending the L2 chain with new blocks.
#[derive(Debug)]
struct Actor<AttributesBuilder_, Conductor_, Origin_, SequencerEngineClient_, Signer_>
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    Origin_: Origin,
    SequencerEngineClient_: SequencerEngineClient,
    Signer_: Signer,
{
    /// Receives messages from handles.
    messages_rx: mpsc::Receiver<Message>,
    /// Whether sequencing is active.
    is_active: bool,
    /// Publishes state to handle readers.
    is_active_tx: watch::Sender<bool>,
    /// The attributes builder used for block building.
    attributes_builder: AttributesBuilder_,
    /// The optional conductor RPC client.
    conductor: Option<Conductor_>,
    /// The struct used to interact with the engine.
    engine_client: SequencerEngineClient_,
    /// The struct used to determine the next L1 origin.
    origin: Origin_,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// Queues built payloads for signing.
    signer: Signer_,
}

/// What a build left behind for the next one: the block to seal, and what to time it from.
#[derive(Debug, PartialEq, Eq)]
enum LastBuild {
    /// It left no block to seal.
    NothingStarted,
    /// It started this block, which the next build seals. Sealing the block before it took
    /// `seal_duration`.
    Started {
        /// The started block.
        block: Box<UnsealedPayloadHandle>,
        /// How long sealing the block before it took.
        seal_duration: Duration,
    },
    /// The engine refused to start a block at this instant.
    Refused(Instant),
}

impl<AttributesBuilder_, Conductor_, Origin_, SequencerEngineClient_, Signer_>
    Actor<AttributesBuilder_, Conductor_, Origin_, SequencerEngineClient_, Signer_>
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    Origin_: Origin,
    SequencerEngineClient_: SequencerEngineClient,
    Signer_: Signer,
{
    #[allow(clippy::too_many_arguments)]
    fn new(
        messages_rx: mpsc::Receiver<Message>,
        is_active_tx: watch::Sender<bool>,
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        origin: Origin_,
        rollup_config: Arc<RollupConfig>,
        signer: Signer_,
    ) -> Self {
        let is_active = *is_active_tx.borrow();
        Self {
            messages_rx,
            is_active,
            is_active_tx,
            attributes_builder,
            conductor,
            engine_client,
            origin,
            rollup_config,
            signer,
        }
    }

    /// Publishes sequencer state and updates its metrics.
    fn update_state(&self) {
        let state_flags = [("active", self.is_active.to_string())];
        metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags).set(1);
        self.is_active_tx.send_replace(self.is_active);
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

        self.signer.send(payload).await.map_err(|error| ActorError::Signer(error.to_string()))
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
        &self,
        unsafe_head: L2BlockInfo,
    ) -> Result<Option<BlockInfo>, ActorError> {
        let l1_origin = match self.origin.next(unsafe_head) {
            Ok(l1_origin) => l1_origin,
            Err(OriginError::Reorged(origin)) => {
                warn!(
                    target: "sequencer",
                    ?origin,
                    "L1 reorged out the unsafe head's L1 origin, resetting engine"
                );
                self.engine_client.reset_engine_forkchoice().await?;
                return Ok(None);
            }
            Err(OriginError::Unavailable) => {
                info!(target: "sequencer", "No L1 origin to build on yet; retrying on the next tick");
                return Ok(None);
            }
            Err(OriginError::Stopped) => return Err(ActorError::OriginStopped),
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
}

impl<AttributesBuilder_, Conductor_, Origin_, SequencerEngineClient_, Signer_>
    Actor<AttributesBuilder_, Conductor_, Origin_, SequencerEngineClient_, Signer_>
where
    AttributesBuilder_: AttributesBuilder + Sync + 'static,
    Conductor_: Conductor + Sync + 'static,
    Origin_: Origin + Sync + 'static,
    SequencerEngineClient_: SequencerEngineClient + Sync + 'static,
    Signer_: Signer + 'static,
{
    async fn run(mut self) -> Result<(), ActorError> {
        // Publish the initial state and metrics before beginning block building.
        self.update_state();
        let block_time = Duration::from_secs(self.rollup_config.block_time);
        let mut last = LastBuild::NothingStarted;
        loop {
            select! {
                // Prioritize messages over block building.
                biased;
                Some(message) = self.messages_rx.recv() => self.handle_message(message).await,
                ready = Self::next_build(self.is_active, &self.signer, &last, block_time) => {
                    ready.map_err(|error| ActorError::Signer(error.to_string()))?;
                    last = self.build(last).await?;
                }
            }
        }
    }

    async fn handle_message(&mut self, message: Message) {
        // A dropped RPC response receiver does not cancel an accepted command.
        match message {
            Message::StartSequencer(tx) => {
                self.is_active = true;
                self.update_state();
                let _ = tx.send(Ok(()));
            }
            Message::StopSequencer(tx) => {
                // Publish before awaiting the unsafe head: sequencing is stopped even if that read
                // fails.
                self.is_active = false;
                self.update_state();
                let result =
                    self.engine_client.get_unsafe_head().await.map(|h| h.hash()).map_err(|_| {
                        HandleError::ErrorAfterSequencerWasStopped(
                            "current unsafe hash is unavailable.".to_string(),
                        )
                    });
                let _ = tx.send(result);
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

    /// Waits until the next build may start: never while sequencing is stopped, and otherwise once
    /// both the signer has room for the build's payload and the build is due.
    ///
    /// This is the whole of the sequencer's timing. It keeps no state, so it can be dropped at any
    /// point without losing anything: the signer's `ready` reserves nothing, and the deadline is
    /// worked out afresh from `last` on every call.
    async fn next_build(
        active: bool,
        signer: &Signer_,
        last: &LastBuild,
        block_time: Duration,
    ) -> Result<(), Signer_::Error> {
        if !active {
            return pending().await;
        }
        let due = match last {
            // Nothing waits to be sealed, so the next block starts at once.
            LastBuild::NothingStarted => Instant::now(),
            // The started block is sealed when its time comes, as early as the last seal took.
            LastBuild::Started { block, seal_duration } => {
                let parent_timestamp = block.attributes_with_parent.parent().block_info.timestamp;
                let timestamp = parent_timestamp.saturating_add(block_time.as_secs());
                let due = UNIX_EPOCH + Duration::from_secs(timestamp) - *seal_duration;
                Instant::now() + due.duration_since(SystemTime::now()).unwrap_or_default()
            }
            // The engine may keep refusing until its state changes, so wait a block time.
            LastBuild::Refused(at) => *at + block_time,
        };
        // The deadline is absolute, so waiting for room and then for it ends at whichever of the
        // two comes later.
        signer.ready().await?;
        time::sleep_until(due).await;
        Ok(())
    }

    /// Seals the block the last build started, if any, and starts the next, returning what it
    /// left behind.
    async fn build(&mut self, last: LastBuild) -> Result<LastBuild, ActorError> {
        info!(target: "sequencer", "Sequencing tick, building block");
        let to_seal = match last {
            LastBuild::Started { block, .. } => Some(*block),
            LastBuild::NothingStarted | LastBuild::Refused(_) => None,
        };
        match self.seal_last_and_start_next(to_seal.as_ref()).await {
            Ok(res) => Ok(match res.unsealed_payload_handle {
                Some(block) => {
                    LastBuild::Started { block: Box::new(block), seal_duration: res.seal_duration }
                }
                None => LastBuild::NothingStarted,
            }),
            Err(ActorError::EngineError(EngineClientError::SealError(err))) => {
                if is_seal_task_err_fatal(&err) {
                    error!(target: "sequencer", err=?err, "Critical seal task error occurred");
                    return Err(ActorError::EngineError(EngineClientError::SealError(err)));
                }
                Ok(LastBuild::NothingStarted)
            }
            Err(ActorError::EngineError(EngineClientError::StartBuildError(err))) => {
                if is_build_task_err_fatal(&err) {
                    error!(target: "sequencer", err=?err, "Critical build task error occurred");
                    return Err(ActorError::EngineError(EngineClientError::StartBuildError(err)));
                }
                warn!(target: "sequencer", err=?err, "Failed to start building a block");
                Ok(LastBuild::Refused(Instant::now()))
            }
            Err(other_err) => {
                error!(target: "sequencer", err = ?other_err, "Unexpected error building or sealing payload");
                Err(other_err)
            }
        }
    }
}

// Determines whether the provided [`BuildTaskError`] is fatal for the sequencer.
//
// Matches exhaustively for the same reason as `is_seal_task_err_fatal`.
const fn is_build_task_err_fatal(err: &BuildTaskError) -> bool {
    match err {
        BuildTaskError::EngineBuildError(build_error) => match build_error {
            EngineBuildError::FinalizedAheadOfUnsafe(_, _) => true,
            EngineBuildError::AttributesInsertionFailed(_) |
            EngineBuildError::InvalidPayload(_) |
            EngineBuildError::UnexpectedPayloadStatus(_) |
            EngineBuildError::MissingPayloadId |
            EngineBuildError::EngineSyncing => false,
        },
    }
}

// Determines whether the provided [`SealTaskError`] is fatal for the sequencer.
//
// NB: We could use `err.severity()`, but that gives the engine actor control over this
// classification. `Actor` may have different interpretations of severity, and it is not clear when
// making a change in that area of the codebase that it will affect this area. When a new task error
// is added, this approach guarantees compilation will fail until it is handled here.
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
        SealTaskError::ClockWentBackwards => true,
    }
}

#[cfg(test)]
mod tests;
