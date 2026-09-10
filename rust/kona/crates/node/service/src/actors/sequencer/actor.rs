//! The [`SequencerActor`].

use crate::{
    NodeActor, SequencerAdminQuery, UnsafePayloadGossipClient,
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
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::{InsertTaskError, SealTaskError, SynchronizeTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{
    sync::Arc,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};
use tokio::{select, sync::mpsc, time::Interval};

/// The handle to a block that has been started but not sealed.
#[derive(Debug)]
struct UnsealedPayloadHandle {
    /// The [`PayloadId`] of the unsealed payload.
    payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] used to start block building.
    attributes_with_parent: OpAttributesWithParent,
}

/// Work retained between block-building ticks.
#[derive(Debug)]
enum PendingPayload {
    /// An engine build that has not yet been accepted by the conductor.
    Unsealed(Box<UnsealedPayloadHandle>),
    /// The exact payload accepted by the conductor but not yet canonicalized locally.
    Committed(Box<OpExecutionPayloadEnvelope>),
}

/// The [`SequencerActor`] is responsible for building L2 blocks on top of the current unsafe head
/// and scheduling them to be signed and gossipped by the P2P layer, extending the L2 chain with new
/// blocks.
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
    /// Receiver for admin API requests.
    pub admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
    /// The attributes builder used for block building.
    pub attributes_builder: AttributesBuilder_,
    /// The optional conductor RPC client.
    pub conductor: Option<Conductor_>,
    /// The struct used to interact with the engine.
    pub engine_client: SequencerEngineClient_,
    /// Whether the sequencer is active.
    pub is_active: bool,
    /// Whether the sequencer is in recovery mode.
    pub in_recovery_mode: bool,
    /// The struct used to determine the next L1 origin.
    pub origin_selector: OriginSelector_,
    /// The rollup configuration.
    pub rollup_config: Arc<RollupConfig>,
    /// A client to asynchronously sign and gossip built payloads to the network actor.
    pub unsafe_payload_gossip_client: UnsafePayloadGossipClient_,

    /// Ticker that paces block-building attempts.
    build_ticker: Interval,
    /// The payload work to resume on the next block-building tick.
    pending_payload: Option<PendingPayload>,
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
        admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
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
        Self {
            admin_api_rx,
            attributes_builder,
            conductor,
            engine_client,
            is_active,
            in_recovery_mode,
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
            build_ticker,
            pending_payload: None,
            last_seal_duration: Duration::from_secs(0),
            started: false,
        }
    }

    /// Seals an engine build and commits the exact payload to the conductor.
    async fn seal_and_commit_payload(
        &self,
        unsealed_payload_handle: &UnsealedPayloadHandle,
    ) -> Result<OpExecutionPayloadEnvelope, SequencerActorError> {
        let seal_request_start = Instant::now();
        let payload = self
            .engine_client
            .seal_block(
                unsealed_payload_handle.payload_id,
                unsealed_payload_handle.attributes_with_parent.clone(),
            )
            .await?;

        update_seal_duration_metrics(seal_request_start.elapsed());
        update_total_transactions_sequenced(
            unsealed_payload_handle.attributes_with_parent.count_transactions(),
        );

        if let Some(conductor) = &self.conductor {
            let conductor_commitment_start = Instant::now();
            conductor.commit_unsafe_payload(&payload).await.inspect_err(|err| {
                error!(target: "sequencer", ?err, "Failed to commit unsafe payload to conductor");
            })?;
            update_conductor_commitment_duration_metrics(conductor_commitment_start.elapsed());
        }

        Ok(payload)
    }

    /// Canonicalizes a conductor-approved payload and then publishes it to peers.
    async fn canonicalize_and_gossip_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), SequencerActorError> {
        self.engine_client.canonicalize_block(payload.clone()).await?;
        self.unsafe_payload_gossip_client
            .schedule_execution_payload_gossip(payload.clone())
            .await
            .map_err(Into::into)
    }

    /// Handles a block-building tick.
    async fn handle_build_tick(&mut self) -> Result<(), SequencerActorError> {
        let pending = self.pending_payload.take();
        let committed_payload = match pending {
            Some(PendingPayload::Unsealed(unsealed)) => {
                let seal_start = Instant::now();
                match self.seal_and_commit_payload(&unsealed).await {
                    Ok(payload) => {
                        self.last_seal_duration = seal_start.elapsed();
                        Some(Box::new(payload))
                    }
                    Err(SequencerActorError::Conductor(err)) => {
                        // Match op-node's temporary-error behavior: retain the build so the same
                        // payload can be sealed and committed again after a short backoff.
                        error!(target: "sequencer", ?err, "Failed to commit unsafe payload to conductor; backing off sequencer");
                        self.pending_payload = Some(PendingPayload::Unsealed(unsealed));
                        self.build_ticker.reset_after(Duration::from_secs(1));
                        return Ok(());
                    }
                    Err(SequencerActorError::EngineError(EngineClientError::SealError(err))) => {
                        if is_seal_task_err_fatal(&err) {
                            error!(target: "sequencer", ?err, "Critical seal task error occurred");
                            return Err(EngineClientError::SealError(err).into());
                        }
                        self.build_ticker.reset_immediately();
                        return Ok(());
                    }
                    Err(other_err) => {
                        error!(target: "sequencer", err = ?other_err, "Unexpected error sealing payload");
                        return Err(other_err);
                    }
                }
            }
            Some(PendingPayload::Committed(payload)) => Some(payload),
            None => None,
        };

        if let Some(payload) = committed_payload {
            match self.canonicalize_and_gossip_payload(&payload).await {
                Ok(()) => {}
                Err(SequencerActorError::EngineError(EngineClientError::CanonicalizeError(
                    err,
                ))) => match canonicalization_error_action(&err) {
                    CanonicalizationErrorAction::Retry => {
                        error!(target: "sequencer", ?err, "Failed to canonicalize conductor-approved payload; backing off sequencer");
                        self.pending_payload = Some(PendingPayload::Committed(payload));
                        self.build_ticker.reset_after(Duration::from_secs(1));
                        return Ok(());
                    }
                    CanonicalizationErrorAction::DropStale => {
                        warn!(target: "sequencer", ?err, "Dropping stale conductor-approved payload");
                        self.build_ticker.reset_immediately();
                        return Ok(());
                    }
                    CanonicalizationErrorAction::Fatal => {
                        return Err(EngineClientError::CanonicalizeError(err).into());
                    }
                },
                Err(other_err) => {
                    error!(target: "sequencer", err = ?other_err, "Unexpected error canonicalizing or gossiping payload");
                    return Err(other_err);
                }
            }
        }

        self.pending_payload =
            self.build_unsealed_payload().await?.map(Box::new).map(PendingPayload::Unsealed);

        if let Some(PendingPayload::Unsealed(payload)) = self.pending_payload.as_ref() {
            let next_block_seconds = payload
                .attributes_with_parent
                .parent()
                .block_info
                .timestamp
                .saturating_add(self.rollup_config.block_time);
            // Next block time is last + block_time - time it takes to seal.
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
        let l1_origin = match self
            .origin_selector
            .next_l1_origin(unsafe_head, self.in_recovery_mode)
            .await
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
        if self.in_recovery_mode {
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
            self.update_metrics();
            // Reset the engine state prior to beginning block building.
            self.schedule_initial_reset().await?;
            self.started = true;
        }

        select! {
            // We are using a biased select here to ensure that the admin queries are given priority over the block building task.
            // This is important to limit the occurrence of race conditions where a stopped query is received when a sequencer is building a new block.
            biased;
            Some(query) = self.admin_api_rx.recv() => {
                let active_before = self.is_active;

                self.handle_admin_query(query).await;

                // immediately attempt to build a block if the sequencer was just started
                if !active_before && self.is_active {
                    self.build_ticker.reset_immediately();
                }
                Ok(())
            }
            // The sequencer must be active to build new blocks.
            _ = self.build_ticker.tick(), if self.is_active => self.handle_build_tick().await,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum CanonicalizationErrorAction {
    /// Retry the exact conductor-approved payload after a backoff.
    Retry,
    /// Drop a payload whose parent is no longer the unsafe head.
    DropStale,
    /// Stop the node because retrying cannot recover safely.
    Fatal,
}

const fn canonicalization_error_action(err: &InsertTaskError) -> CanonicalizationErrorAction {
    match err {
        InsertTaskError::StalePayload { .. } => CanonicalizationErrorAction::DropStale,
        InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => {
            CanonicalizationErrorAction::Retry
        }
        InsertTaskError::ForkchoiceUpdateFailed(err) => match err {
            SynchronizeTaskError::FinalizedAheadOfUnsafe(_, _) => {
                CanonicalizationErrorAction::Fatal
            }
            SynchronizeTaskError::ForkchoiceUpdateFailed(_) |
            SynchronizeTaskError::InvalidForkchoiceState |
            SynchronizeTaskError::UnexpectedPayloadStatus(_) => CanonicalizationErrorAction::Retry,
        },
        InsertTaskError::FromBlockError(_) |
        InsertTaskError::L2BlockInfoConstruction(_) |
        InsertTaskError::MpscSend(_) => CanonicalizationErrorAction::Fatal,
    }
}

// Determines whether the provided [`SealTaskError`] is fatal for the sequencer.
//
// NB: We could use `err.severity()`, but that gives EngineActor control over this classification.
// `SequencerActor` may have different interpretations of severity, and it is not clear when making
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
            InsertTaskError::FromBlockError(_) |
            InsertTaskError::L2BlockInfoConstruction(_) |
            InsertTaskError::MpscSend(_) => true,
            InsertTaskError::StalePayload { .. } |
            InsertTaskError::InsertFailed(_) |
            InsertTaskError::UnexpectedPayloadStatus(_) => false,
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
#[path = "tests/actor_test.rs"]
mod tests;
