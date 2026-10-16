//! The [`SequencerActor`].

use crate::{
    NodeActor, UnsafePayloadGossipClient,
    actors::{
        SequencerEngineClient,
        engine::EngineClientError,
        sequencer::{
            conductor::Conductor,
            error::SequencerActorError,
            metrics::{
                update_attributes_build_duration_metrics, update_block_build_duration_metrics,
                update_conductor_commitment_duration_metrics, update_seal_duration_metrics,
                update_total_transactions_sequenced,
            },
            origin_selector::{L1OriginSelectorError, OriginSelector},
        },
    },
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use async_trait::async_trait;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::{CanonicalizeTaskError, InsertTaskError, SealTaskError, SynchronizeTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_rpc::{SequencerAdminAPIError, SequencerAdminCommand, SequencerState};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{
    sync::Arc,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};
use tokio::{
    select,
    sync::{mpsc, watch},
    time::Interval,
};

/// The handle to a block that has been started but not sealed.
#[derive(Debug)]
struct UnsealedPayloadHandle {
    /// The [`PayloadId`] of the unsealed payload.
    payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] used to start block building.
    attributes_with_parent: OpAttributesWithParent,
}

/// A sequenced block that has been sealed and committed to the conductor, if one is configured,
/// but not yet canonicalized and gossiped.
#[derive(Debug)]
struct CommittedBlock {
    /// The committed payload. Retries must canonicalize this exact envelope.
    pub envelope: OpExecutionPayloadEnvelope,
    /// The unsafe head on which the block was built.
    pub parent: L2BlockInfo,
    /// The number of transactions in the block.
    pub tx_count: u64,
}

/// The block the sequencer is working on between ticks.
// The sequencer holds at most one of these, so the variants' size difference is irrelevant.
#[allow(clippy::large_enum_variant)]
#[derive(Debug)]
enum InFlightBlock {
    /// Built but not yet sealed. Commit failures retry the seal with this payload ID.
    Building(UnsealedPayloadHandle),
    /// Sealed and committed. Retries must canonicalize this exact envelope.
    Committed(CommittedBlock),
}

/// The [`SequencerActor`] is responsible for building L2 blocks on top of the current unsafe head
/// and handing them to the [`SignerActor`](crate::SignerActor) to be signed and gossipped,
/// extending the L2 chain with new blocks.
#[derive(Debug)]
pub struct SequencerActor<
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
    /// Receiver for sequencer admin commands.
    pub admin_command_rx: mpsc::Receiver<SequencerAdminCommand>,
    /// Sequencer state shared with admin RPC readers.
    state: watch::Sender<SequencerState>,
    /// The attributes builder used for block building.
    pub attributes_builder: AttributesBuilder_,
    /// The optional conductor RPC client.
    conductor: Option<Conductor_>,
    /// The struct used to interact with the engine.
    pub engine_client: SequencerEngineClient_,
    /// The struct used to determine the next L1 origin.
    pub origin_selector: OriginSelector_,
    /// The rollup configuration.
    pub rollup_config: Arc<RollupConfig>,
    /// A client that hands built payloads to the signer actor, which signs them for the network
    /// actor to gossip.
    pub unsafe_payload_gossip_client: UnsafePayloadGossipClient_,

    /// Ticker that paces block-building attempts.
    build_ticker: Interval,
    /// The block carried over from a previous tick, if any.
    in_flight: Option<InFlightBlock>,
    /// Duration of the most recent seal operation, used to back-pressure the build ticker.
    last_seal_duration: Duration,
    /// Whether the one-shot startup work (metrics + initial engine reset) has run.
    started: bool,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencerActor<
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
    /// Instantiate a new [`SequencerActor`].
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        admin_command_rx: mpsc::Receiver<SequencerAdminCommand>,
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        is_active: bool,
        in_recovery_mode: bool,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        let build_ticker = tokio::time::interval(Duration::from_secs(rollup_config.block_time));
        let (state, _) = watch::channel(SequencerState {
            active: is_active,
            conductor_enabled: conductor.is_some(),
            recovery_mode: in_recovery_mode,
        });
        Self {
            admin_command_rx,
            state,
            attributes_builder,
            conductor,
            engine_client,
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
            build_ticker,
            in_flight: None,
            last_seal_duration: Duration::from_secs(0),
            started: false,
        }
    }

    /// Subscribe to the sequencer's state.
    pub fn admin_state_receiver(&self) -> watch::Receiver<SequencerState> {
        self.state.subscribe()
    }

    /// Copy the current state so no watch borrow is held across an await.
    fn state(&self) -> SequencerState {
        *self.state.borrow()
    }

    /// Update and publish sequencer state together with its metrics.
    fn update_state(&self, update: impl FnOnce(&mut SequencerState)) {
        self.state.send_modify(|state| {
            update(state);
            let state_flags = [
                ("active", state.active.to_string()),
                ("recovery", state.recovery_mode.to_string()),
            ];
            metrics::gauge!(crate::Metrics::SEQUENCER_STATE, &state_flags).set(1);
        });
    }

    /// Starts the sequencer in an idempotent fashion.
    fn start_sequencer(&self) {
        if self.state().active {
            info!(target: "sequencer", "received request to start sequencer, but it is already started");
            return;
        }

        info!(target: "sequencer", "Starting sequencer");
        self.update_state(|state| state.active = true);
    }

    /// Stops the sequencer in an idempotent fashion.
    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        info!(target: "sequencer", "Stopping sequencer");
        // Publish before awaiting the unsafe head: sequencing is stopped even if that read fails.
        self.update_state(|state| state.active = false);

        self.engine_client.get_unsafe_head().await
            .map(|h| h.hash())
            .map_err(|e| {
                error!(target: "sequencer", err=?e, "Error fetching unsafe head after stopping sequencer, which should never happen.");
                SequencerAdminAPIError::ErrorAfterSequencerWasStopped("current unsafe hash is unavailable.".to_string())
            })
    }

    /// Sets the recovery mode of the sequencer in an idempotent fashion.
    fn set_recovery_mode(&self, mode: bool) {
        self.update_state(|state| state.recovery_mode = mode);
        info!(target: "sequencer", is_active = mode, "Updated recovery mode");
    }

    /// Overrides the leader, if the conductor is enabled.
    /// If not, an error will be returned.
    async fn override_leader(&mut self) -> Result<(), SequencerAdminAPIError> {
        let Some(conductor) = self.conductor.as_mut() else {
            return Err(SequencerAdminAPIError::LeaderOverrideError(
                "No conductor configured".to_string(),
            ));
        };

        if let Err(e) = conductor.override_leader().await {
            error!(target: "sequencer::rpc", "Failed to override leader: {}", e);
            return Err(SequencerAdminAPIError::LeaderOverrideError(e.to_string()));
        }
        info!(target: "sequencer", "Overrode leader via the conductor service");

        Ok(())
    }

    /// Seals the provided [`UnsealedPayloadHandle`] and commits the resulting payload to the
    /// conductor, if one is configured.
    async fn seal_and_commit(
        &self,
        handle: &UnsealedPayloadHandle,
    ) -> Result<CommittedBlock, SequencerActorError> {
        let seal_request_start = Instant::now();
        let envelope = self
            .engine_client
            .seal_block(handle.payload_id, handle.attributes_with_parent.clone())
            .await?;
        update_seal_duration_metrics(seal_request_start.elapsed());

        if let Some(conductor) = &self.conductor {
            let commitment_start = Instant::now();
            let result = conductor.commit_unsafe_payload(&envelope).await;
            update_conductor_commitment_duration_metrics(commitment_start.elapsed());
            // Retain the build job, not this uncommitted envelope. Like op-node, retry
            // getPayload with the same payload ID on the next attempt.
            result?;
        }

        Ok(CommittedBlock {
            envelope,
            parent: handle.attributes_with_parent.parent,
            tx_count: handle.attributes_with_parent.count_transactions(),
        })
    }

    /// Canonicalizes the provided [`CommittedBlock`] and schedules it for gossip.
    async fn canonicalize_and_gossip(
        &self,
        block: &CommittedBlock,
    ) -> Result<(), SequencerActorError> {
        self.engine_client.canonicalize_block(block.envelope.clone(), block.parent).await?;
        update_total_transactions_sequenced(block.tx_count);

        self.unsafe_payload_gossip_client
            .schedule_execution_payload_gossip(block.envelope.clone())
            .await
            .map_err(Into::into)
    }

    /// Starts building an L2 block by creating and populating payload attributes referencing the
    /// correct L1 origin block and sending them to the block engine.
    async fn build_unsealed_payload(
        &mut self,
    ) -> Result<Option<UnsealedPayloadHandle>, SequencerActorError> {
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
    ) -> Result<Option<BlockInfo>, SequencerActorError> {
        let recovery_mode = self.state().recovery_mode;
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
    ) -> Result<Option<OpAttributesWithParent>, SequencerActorError> {
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
                    return Err(SequencerActorError::ChannelClosed);
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
        if self.state().recovery_mode {
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
    async fn schedule_initial_reset(&self) -> Result<(), SequencerActorError> {
        // Reset the engine, in order to initialize the engine state.
        // NB: this call waits for confirmation that the reset succeeded and we can proceed with
        // post-reset logic.
        self.engine_client.reset_engine_forkchoice().await.map_err(|err| {
            error!(target: "sequencer", ?err, "Failed to send reset request to engine");
            err.into()
        })
    }
}

#[async_trait]
impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> NodeActor
    for SequencerActor<
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
    type Error = SequencerActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        if !self.started {
            // Publish the initial state and metrics before beginning block building.
            self.update_state(|_| {});
            // Reset the engine state prior to beginning block building.
            self.schedule_initial_reset().await?;
            self.started = true;
        }

        select! {
            // We are using a biased select here to ensure that the admin commands are given priority over the block building task.
            // This is important to limit the occurrence of race conditions where a stop command is received when a sequencer is building a new block.
            biased;
            Some(command) = self.admin_command_rx.recv() => {
                // A dropped RPC response receiver does not cancel an accepted command.
                match command {
                    SequencerAdminCommand::StartSequencer(tx) => {
                        self.start_sequencer();
                        let _ = tx.send(Ok(()));
                    }
                    SequencerAdminCommand::StopSequencer(tx) => {
                        let _ = tx.send(self.stop_sequencer().await);
                    }
                    SequencerAdminCommand::SetRecoveryMode(mode, tx) => {
                        self.set_recovery_mode(mode);
                        let _ = tx.send(Ok(()));
                    }
                    SequencerAdminCommand::OverrideLeader(tx) => {
                        let _ = tx.send(self.override_leader().await);
                    }
                }

                Ok(())
            }
            // The sequencer must be active to build new blocks.
            _ = self.build_ticker.tick(), if self.state().active => {
                if !self.unsafe_payload_gossip_client.has_capacity() {
                    warn!(target: "sequencer", "Sequencing tick, gossip queue full, not building a block");
                    return Ok(());
                }
                info!(target: "sequencer", "Sequencing tick, building block");
                // Advance the block carried over from the previous tick. A block that was already
                // committed skips straight to canonicalization. On failure, keep the state the
                // block was in when it failed so that a retry resumes from there.
                let publish_start = self.in_flight.is_some().then(Instant::now);
                let committed = match self.in_flight.take() {
                    None => None,
                    Some(InFlightBlock::Committed(block)) => Some(block),
                    Some(InFlightBlock::Building(handle)) => match self.seal_and_commit(&handle).await {
                        Ok(block) => Some(block),
                        Err(SequencerActorError::ConductorCommit(err)) => {
                            warn!(target: "sequencer", ?err, "Conductor commit failed; retrying the build job before canonicalization or gossip");
                            self.in_flight = Some(InFlightBlock::Building(handle));
                            self.build_ticker.reset_after(Duration::from_secs(1));
                            return Ok(());
                        }
                        Err(SequencerActorError::EngineError(EngineClientError::SealError(err))) => match err {
                            // The build is stale, or the EL may have expired or discarded its
                            // payload ID. Drop it and rebuild on the next tick.
                            SealTaskError::UnsafeHeadChangedSinceBuild | SealTaskError::GetPayloadFailed(_) => {
                                self.build_ticker.reset_immediately();
                                return Ok(());
                            }
                            // Only produced when sealing also imports the payload, or within the
                            // engine itself.
                            SealTaskError::PayloadInsertionFailed(_) |
                            SealTaskError::DepositOnlyPayloadFailed |
                            SealTaskError::DepositOnlyPayloadReattemptFailed |
                            SealTaskError::HoloceneInvalidFlush |
                            SealTaskError::MpscSend(_) |
                            SealTaskError::ClockWentBackwards => {
                                error!(target: "sequencer", ?err, "Critical seal task error occurred");
                                return Err(SequencerActorError::EngineError(EngineClientError::SealError(err)));
                            }
                        },
                        Err(err) => {
                            error!(target: "sequencer", ?err, "Unexpected error sealing payload");
                            return Err(err);
                        }
                    },
                };
                if let Some(block) = committed {
                    match self.canonicalize_and_gossip(&block).await {
                        Ok(()) => {}
                        Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(err))) => match &err {
                            // The block no longer extends the unsafe head. Drop it and rebuild on
                            // the next tick.
                            CanonicalizeTaskError::UnsafeHeadChangedSinceBuild => {
                                self.build_ticker.reset_immediately();
                                return Ok(());
                            }
                            CanonicalizeTaskError::PayloadInsertionFailed(insert_err) if is_insert_task_err_fatal(insert_err) => {
                                error!(target: "sequencer", ?err, "Critical canonicalization error occurred");
                                return Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(err)));
                            }
                            // The engine rejected the block. Drop it and rebuild on the next tick.
                            CanonicalizeTaskError::PayloadInsertionFailed(insert_err) if matches!(
                                **insert_err,
                                InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. })
                            ) => {
                                self.build_ticker.reset_immediately();
                                return Ok(());
                            }
                            // Retry temporary insertion failures with the exact committed envelope.
                            CanonicalizeTaskError::PayloadInsertionFailed(_) => {
                                self.in_flight = Some(InFlightBlock::Committed(block));
                                self.build_ticker.reset_after(Duration::from_secs(1));
                                return Ok(());
                            }
                            CanonicalizeTaskError::MpscSend(_) => {
                                error!(target: "sequencer", ?err, "Critical canonicalization error occurred");
                                return Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(err)));
                            }
                        },
                        Err(err) => {
                            error!(target: "sequencer", ?err, "Unexpected error canonicalizing or gossiping payload");
                            return Err(err);
                        }
                    }
                }
                self.last_seal_duration = publish_start.map_or(Duration::ZERO, |start| start.elapsed());

                // Start the build job for the next L2 block on top of the current unsafe head.
                match self.build_unsealed_payload().await {
                    Ok(handle) => self.in_flight = handle.map(InFlightBlock::Building),
                    Err(err) => {
                        error!(target: "sequencer", ?err, "Unexpected error building payload");
                        return Err(err);
                    }
                }

                if let Some(InFlightBlock::Building(handle)) = self.in_flight.as_ref() {
                    let next_block_seconds = handle.attributes_with_parent.parent().block_info.timestamp.saturating_add(self.rollup_config.block_time);
                    // next block time is last + block_time - time it takes to seal.
                    let next_block_time = UNIX_EPOCH + Duration::from_secs(next_block_seconds) - self.last_seal_duration;
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
    }
}

// Determines whether the provided [`InsertTaskError`] is fatal for the sequencer.
//
// NB: We could use `err.severity()`, but that gives EngineActor control over this classification.
// `SequencerActor` may have different interpretations of severity, and it is not clear when making
// a change in that area of the codebase that it will affect this area. When a new task error is
// added, this approach guarantees compilation will fail until it is handled here. Seal and
// canonicalization errors are matched exhaustively where they occur in `step`.
const fn is_insert_task_err_fatal(err: &InsertTaskError) -> bool {
    match err {
        InsertTaskError::ForkchoiceUpdateFailed(synchronize_error) => match synchronize_error {
            SynchronizeTaskError::FinalizedAheadOfUnsafe(_, _) => true,
            SynchronizeTaskError::ForkchoiceUpdateFailed(_) |
            SynchronizeTaskError::InvalidForkchoiceState |
            SynchronizeTaskError::UnexpectedPayloadStatus(_) => false,
        },
        InsertTaskError::FromBlockError(_) | InsertTaskError::L2BlockInfoConstruction(_) => true,
        InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => false,
    }
}

#[cfg(test)]
mod tests;
