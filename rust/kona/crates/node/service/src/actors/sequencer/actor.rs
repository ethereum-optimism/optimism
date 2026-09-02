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
                update_attributes_build_duration_metrics, update_await_ready_duration_metrics,
                update_block_build_duration_metrics, update_conductor_commitment_duration_metrics,
                update_seal_duration_metrics, update_total_transactions_sequenced,
            },
            origin_selector::{L1OriginSelectorError, OriginSelector},
        },
    },
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::{InsertTaskError, SealTaskError, SealedPayload, SynchronizeTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{
    select,
    sync::mpsc,
    time::{Instant, sleep_until},
};

/// A block the sequencer sealed, and how long sealing it took.
#[derive(Debug)]
struct SealedBlock {
    /// The sealed block.
    block: L2BlockInfo,
    /// How long the engine took to seal it, excluding the wait for the execution layer to
    /// consider the payload worth sealing.
    seal_duration: Duration,
}

/// The handle to a block that has been started but not sealed.
#[derive(Debug)]
pub(super) struct UnsealedPayloadHandle {
    /// The [`PayloadId`] of the unsealed payload.
    pub payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] used to start block building.
    pub attributes_with_parent: OpAttributesWithParent,
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

    /// When the next sequencing slot may start.
    next_slot_at: Instant,
    /// Duration of the most recent seal operation, reserved at the end of a slot so the last
    /// block of a group lands on its timestamp rather than a seal later.
    last_seal_duration: Duration,
    /// The slot timestamp whose failure already bought an immediate retry, so a persistently
    /// failing execution layer cannot spin.
    retried_timestamp: Option<u64>,
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
            next_slot_at: Instant::now(),
            last_seal_duration: Duration::ZERO,
            retried_timestamp: None,
            started: false,
        }
    }

    /// Runs one sequencing slot.
    ///
    /// The slot builds the first block on top of the current unsafe head at
    /// `parent.timestamp + block_time`, then keeps extending the group with siblings carrying that
    /// same timestamp and L1 origin. The group ends at the slot deadline, once it holds
    /// `max_multi_blocks` blocks, on a block built without the transaction pool, on any engine
    /// reset or seal failure, or on an admin request to stop or enter recovery mode.
    pub(super) async fn run_slot(&mut self) -> Result<(), SequencerActorError> {
        let unsafe_head = self.engine_client.get_unsafe_head().await?;
        let timestamp = unsafe_head.block_info.timestamp + self.rollup_config.block_time;
        let deadline = self.slot_deadline(timestamp);
        // Pace the next slot even if this one ends early, so the chain never runs ahead of the
        // wall clock.
        self.next_slot_at = deadline;

        // Siblings only ever extend a group this loop started, so a (re)started sequencer never
        // extends a group it did not build.
        let siblings_allowed = self.rollup_config.siblings_allowed(timestamp);

        // The blocks this slot has sealed. The group length is walked over parent hashes; these
        // answer the walk for the blocks the sequencer built itself, without a round trip to the
        // engine for each of them.
        let mut group = Vec::new();

        // Every block of the group shares one L1 origin, so it is selected once per slot. Any
        // early return below discards it, and the next slot selects afresh.
        let Some(l1_origin) = self.get_next_payload_l1_origin(unsafe_head, timestamp).await? else {
            self.retry_empty_slot(&group, timestamp);
            return Ok(());
        };

        let mut parent = unsafe_head;

        loop {
            let Some(handle) = self.build_unsealed_payload(parent, l1_origin, timestamp).await?
            else {
                self.retry_empty_slot(&group, timestamp);
                return Ok(());
            };

            // A block built without the transaction pool (an upgrade, drift or recovery block) is
            // the only block of its slot.
            let closes_group =
                handle.attributes_with_parent.attributes().no_tx_pool.unwrap_or(false);

            if !siblings_allowed && self.wait_for_deadline(deadline).await {
                return Ok(());
            }

            let ready_deadline = siblings_allowed.then_some(deadline);
            let sealed = match self.seal_and_commit_payload(&handle, ready_deadline).await {
                Ok(sealed) => sealed,
                Err(SequencerActorError::EngineError(EngineClientError::SealError(err))) => {
                    if is_seal_task_err_fatal(&err) {
                        error!(target: "sequencer", err=?err, "Critical seal task error occurred");
                        return Err(SequencerActorError::EngineError(
                            EngineClientError::SealError(err),
                        ));
                    }
                    warn!(target: "sequencer", err=?err, "Failed to seal payload, ending the block group");
                    self.retry_empty_slot(&group, timestamp);
                    return Ok(());
                }
                Err(other_err) => {
                    error!(target: "sequencer", err = ?other_err, "Unexpected error sealing payload");
                    return Err(other_err);
                }
            };
            self.last_seal_duration = sealed.seal_duration;
            group.push(sealed.block.block_info);

            if closes_group || !siblings_allowed || Instant::now() >= deadline {
                return Ok(());
            }

            // A stop or recovery-mode request must take effect before the next sibling, so that
            // `admin_stopSequencer` hands back the block the sequencer actually stopped at.
            if self.poll_admin_requests().await {
                return Ok(());
            }

            if self.group_length(sealed.block.block_info, &group).await? >=
                self.rollup_config.max_multi_blocks()
            {
                return Ok(());
            }

            // The next sibling extends the block just sealed. The engine publishes its unsafe
            // head only after the task queue drains, so reading it back here would race with the
            // seal that has just returned.
            parent = sealed.block;
        }
    }

    /// Lets the next slot start at once after a slot that sealed nothing, instead of idling
    /// until a deadline no block will land on.
    ///
    /// A slot that already sealed a block waits out its deadline, so the next timestamp is never
    /// started ahead of the wall clock, and a timestamp is only ever retried once this way, so an
    /// execution layer that fails every attempt degrades to one attempt per slot rather than
    /// spinning on the auth RPC.
    fn retry_empty_slot(&mut self, group: &[BlockInfo], timestamp: u64) {
        if !group.is_empty() || self.retried_timestamp.replace(timestamp) == Some(timestamp) {
            return;
        }
        self.next_slot_at = Instant::now();
    }

    /// The instant by which the slot's blocks must be sealed: the wall-clock time of the slot's
    /// L2 timestamp, less the duration of the last seal so that the group's final block lands on
    /// its timestamp rather than a seal later. Already in the past when the sequencer is behind
    /// the wall clock, which is what limits a catching-up sequencer to one block per timestamp.
    fn slot_deadline(&self, timestamp: u64) -> Instant {
        let now = Instant::now();
        let Ok(unix_now) = SystemTime::now().duration_since(UNIX_EPOCH) else {
            return now;
        };
        let target = Duration::from_secs(timestamp).saturating_sub(self.last_seal_duration);
        now.checked_add(target.saturating_sub(unix_now)).unwrap_or(now)
    }

    /// Waits until `deadline`, serving admin requests meanwhile. Returns whether the slot must
    /// end because the sequencer was stopped or put into recovery mode.
    async fn wait_for_deadline(&mut self, deadline: Instant) -> bool {
        loop {
            select! {
                biased;
                Some(query) = self.admin_api_rx.recv() => {
                    self.handle_admin_query(query).await;
                    if !self.is_active || self.in_recovery_mode {
                        return true;
                    }
                }
                _ = sleep_until(deadline) => return false,
            }
        }
    }

    /// Serves every admin request that is already pending, without blocking. Returns whether the
    /// block group must end because the sequencer was stopped or put into recovery mode.
    async fn poll_admin_requests(&mut self) -> bool {
        while let Ok(query) = self.admin_api_rx.try_recv() {
            self.handle_admin_query(query).await;
        }
        !self.is_active || self.in_recovery_mode
    }

    /// Returns how many blocks at the tip of the chain share `head`'s timestamp, capped at
    /// `max_multi_blocks`.
    ///
    /// The length is walked back from `head` over parent hashes rather than counted while
    /// building, so it stays correct across restarts, reorgs, and blocks the engine imported from
    /// elsewhere. `known` answers the walk for blocks the caller already holds. A parent the
    /// execution layer cannot produce is reported as a full group, ending the sequencer's group
    /// rather than risking one block too many.
    async fn group_length(
        &self,
        head: BlockInfo,
        known: &[BlockInfo],
    ) -> Result<u64, SequencerActorError> {
        let max = self.rollup_config.max_multi_blocks();
        let group_timestamp = head.timestamp;
        let mut length = 1;
        let mut block = head;
        while length < max {
            let parent = match known.iter().find(|candidate| candidate.hash == block.parent_hash) {
                Some(parent) => *parent,
                None => {
                    let Some(parent) =
                        self.engine_client.l2_block_info_by_hash(block.parent_hash).await?
                    else {
                        warn!(
                            target: "sequencer",
                            hash = %block.parent_hash,
                            "Parent of the unsafe head is unavailable, treating the block group as full"
                        );
                        return Ok(max);
                    };
                    parent
                }
            };
            if parent.timestamp != group_timestamp {
                break;
            }
            length += 1;
            block = parent;
        }
        Ok(length)
    }

    /// Sends a seal request to seal the provided [`UnsealedPayloadHandle`], committing and
    /// gossiping the resulting block.
    ///
    /// Returns the sealed block, which the next sibling of the group builds on.
    async fn seal_and_commit_payload(
        &self,
        unsealed_payload_handle: &UnsealedPayloadHandle,
        ready_deadline: Option<Instant>,
    ) -> Result<SealedBlock, SequencerActorError> {
        let seal_request_start = Instant::now();

        // Send the seal request to the engine to seal the unsealed block.
        let SealedPayload { payload, block, seal_duration } = self
            .engine_client
            .seal_and_canonicalize_block(
                unsealed_payload_handle.payload_id,
                unsealed_payload_handle.attributes_with_parent.clone(),
                ready_deadline,
            )
            .await?;

        // The round trip covers the readiness wait as well; the engine reports the seal alone, so
        // the difference is what the sequencer spent waiting for the payload to become ready.
        let seal_request_duration = seal_request_start.elapsed();
        update_seal_duration_metrics(seal_request_duration);
        update_await_ready_duration_metrics(seal_request_duration.saturating_sub(seal_duration));

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

        self.unsafe_payload_gossip_client.schedule_execution_payload_gossip(payload).await?;

        Ok(SealedBlock { block, seal_duration })
    }

    /// Starts building an L2 block on top of `parent`, at `timestamp` and in the epoch of
    /// `l1_origin`, by creating and populating payload attributes and sending them to the block
    /// engine.
    pub(super) async fn build_unsealed_payload(
        &mut self,
        parent: L2BlockInfo,
        l1_origin: BlockInfo,
        timestamp: u64,
    ) -> Result<Option<UnsealedPayloadHandle>, SequencerActorError> {
        info!(
            target: "sequencer",
            parent_num = parent.block_info.number,
            l1_origin_num = l1_origin.number,
            timestamp,
            "Started sequencing new block"
        );

        // Build the payload attributes for the next block.
        let attributes_build_start = Instant::now();

        let Some(attributes_with_parent) =
            self.build_attributes(parent, l1_origin, timestamp).await?
        else {
            // Temporary error or reset - retry on the next slot.
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

    /// Determines and validates the L1 origin block for the block(s) to be built at `timestamp`
    /// on top of the provided L2 unsafe head.
    /// Returns `Ok(None)` for temporary errors that should be retried.
    async fn get_next_payload_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
        timestamp: u64,
    ) -> Result<Option<BlockInfo>, SequencerActorError> {
        let l1_origin = match self
            .origin_selector
            .next_l1_origin(unsafe_head, timestamp, self.in_recovery_mode)
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
                    "Temporary error occurred while selecting next L1 origin. Re-attempting on the next slot."
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
        parent: L2BlockInfo,
        l1_origin: BlockInfo,
        timestamp: u64,
    ) -> Result<Option<OpAttributesWithParent>, SequencerActorError> {
        let mut attributes = match self
            .attributes_builder
            .prepare_payload_attributes(parent, l1_origin.id(), timestamp)
            .await
        {
            Ok(attrs) => attrs,
            Err(PipelineErrorKind::Temporary(_)) => {
                // Temporary error - retry on the next slot.
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

        let attrs_with_parent = OpAttributesWithParent::new(attributes, parent, None, false);
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
                    self.next_slot_at = Instant::now();
                }
                Ok(())
            }
            // The sequencer must be active to build new blocks.
            _ = sleep_until(self.next_slot_at), if self.is_active => self.run_slot().await,
        }
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
            InsertTaskError::FromBlockError(_) | InsertTaskError::L2BlockInfoConstruction(_) => {
                true
            }
            InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => false,
        },
        SealTaskError::GetPayloadFailed(_) |
        SealTaskError::PayloadJobUnknown(_) |
        SealTaskError::HoloceneInvalidFlush |
        SealTaskError::UnsafeHeadChangedSinceBuild => false,
        SealTaskError::DepositOnlyPayloadFailed |
        SealTaskError::DepositOnlyPayloadReattemptFailed |
        SealTaskError::MpscSend(_) |
        SealTaskError::ClockWentBackwards => true,
    }
}
