//! The [`SequencerActor`].

use super::handle::{Message, SequencerHandle, SequencerState};
use crate::{
    BuildToken, EngineContext, NodeActor, Sealed, UnsafePayloadGossipClient,
    actors::{
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
use async_trait::async_trait;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::EnginePhase;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
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
pub(super) struct UnsealedPayloadHandle {
    /// The engine-owned identity of the unsealed payload.
    pub job: PendingBuild,
    /// The [`OpAttributesWithParent`] used to start block building.
    pub attributes_with_parent: OpAttributesWithParent,
}

/// Retrieval and import retries retain the exact original engine work.
#[derive(Debug)]
pub(super) enum PendingBuild {
    Building(BuildToken),
    /// A positive conductor acknowledgement, or explicit local publication permission.
    Committed(Box<Sealed>),
}

/// The [`SequencerActor`] is responsible for building L2 blocks on top of the current unsafe head
/// and handing them to the [`SignerActor`](crate::SignerActor) to be signed and gossipped,
/// extending the L2 chain with new blocks.
#[derive(Debug)]
pub struct SequencerActor<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Private state-changing commands from the sequencer handle.
    pub(super) commands: mpsc::Receiver<Message>,
    /// Authoritative administrative state; publish before acknowledging mutations.
    pub(super) state: SequencerState,
    state_tx: watch::Sender<SequencerState>,
    /// The attributes builder used for block building.
    pub attributes_builder: AttributesBuilder_,
    /// The optional conductor RPC client.
    pub(super) conductor: Option<Conductor_>,
    /// Process-local disaster recovery permission; the configured conductor remains observable.
    pub(super) conductor_overridden: bool,
    /// The struct used to interact with the engine.
    pub engine_client: crate::EngineHandle,
    /// The struct used to determine the next L1 origin.
    pub origin_selector: OriginSelector_,
    /// The rollup configuration.
    pub rollup_config: Arc<RollupConfig>,
    /// A client that hands built payloads to the signer actor, which signs them for the network
    /// actor to gossip.
    pub unsafe_payload_gossip_client: UnsafePayloadGossipClient_,

    /// Ticker that paces block-building attempts.
    pub(super) build_ticker: Interval,
    /// The handle for the payload built on the previous tick that is waiting to be sealed.
    next_payload_to_seal: Option<UnsealedPayloadHandle>,
    /// Duration of the most recent seal operation, used to back-pressure the build ticker.
    pub(super) last_seal_duration: Duration,
    /// Whether the one-shot startup work (metrics + initial engine reset) has run.
    started: bool,
}

impl<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_>
    SequencerActor<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_>
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Instantiate a new [`SequencerActor`].
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: crate::EngineHandle,
        is_active: bool,
        in_recovery_mode: bool,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> (Self, SequencerHandle) {
        let (command_tx, commands) = mpsc::channel(32);
        let state = SequencerState {
            active: is_active && conductor.is_none(),
            recovery_mode: in_recovery_mode,
        };
        let (state_tx, state_rx) = watch::channel(state);
        let handle = SequencerHandle::new(command_tx, state_rx, conductor.is_some());
        let build_ticker = tokio::time::interval(Duration::from_secs(rollup_config.block_time));
        let actor = Self {
            commands,
            state,
            state_tx,
            attributes_builder,
            conductor,
            conductor_overridden: false,
            engine_client,
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
            build_ticker,
            next_payload_to_seal: None,
            last_seal_duration: Duration::from_secs(0),
            started: false,
        };
        (actor, handle)
    }

    pub(super) fn publish(&self) {
        self.state_tx.send_replace(self.state);
        self.update_metrics();
    }

    #[cfg(test)]
    pub(super) fn test_pending(&mut self, pending: UnsealedPayloadHandle) {
        self.next_payload_to_seal = Some(pending);
        self.started = true;
    }

    #[cfg(test)]
    pub(super) const fn has_pending(&self) -> bool {
        self.next_payload_to_seal.is_some()
    }

    /// A reset or competing head makes this job unusable, even when the reset kept the same heads.
    fn context_is_current(&self, context: EngineContext) -> bool {
        let state = self.engine_client.state();
        state.epoch == context.epoch &&
            state.sync_state.unsafe_head() == context.parent &&
            state.phase != EnginePhase::Resetting
    }

    /// Retrieve until positively acknowledged, then import only that exact envelope.
    /// `false` retains an unacknowledged job for another getPayload/commit attempt.
    pub(super) async fn publish_payload(
        &self,
        pending: &mut UnsealedPayloadHandle,
    ) -> Result<bool, SequencerActorError> {
        if let PendingBuild::Building(token) = &pending.job {
            let sealed = self.engine_client.seal(token.clone()).await?;
            if !self.context_is_current(sealed.context()) {
                return Err(EngineClientError::Stale.into());
            }
            // Unmanaged or locally overridden sequencing owns publication permission.
            if !self.conductor_overridden &&
                let Some(conductor) = &self.conductor
            {
                let started = Instant::now();
                let result = conductor.commit_unsafe_payload(sealed.payload()).await;
                update_conductor_commitment_duration_metrics(started.elapsed());
                if let Err(error) = result {
                    warn!(target: "sequencer", %error, "Conductor did not acknowledge payload; retaining build job");
                    return Ok(false);
                }
            }
            pending.job = PendingBuild::Committed(Box::new(sealed));
        }
        self.import_committed(pending).await?;
        Ok(true)
    }

    /// Engine publication precedes signing/gossip admission and any Stop acknowledgement.
    async fn import_committed(
        &self,
        pending: &UnsealedPayloadHandle,
    ) -> Result<(), SequencerActorError> {
        let PendingBuild::Committed(sealed) = &pending.job else {
            unreachable!("only acknowledged or locally authorized payloads may be imported")
        };
        let started = Instant::now();
        let payload = self.engine_client.canonicalize(sealed.as_ref().clone()).await?;
        update_seal_duration_metrics(started.elapsed());
        update_total_transactions_sequenced(pending.attributes_with_parent.count_transactions());
        self.unsafe_payload_gossip_client.schedule_execution_payload_gossip(payload).await?;
        Ok(())
    }

    /// Stop owns settlement even if its RPC caller drops the reply while this method waits.
    pub(super) async fn settle_committed(&mut self) -> Result<(), SequencerActorError> {
        let Some(pending) = self.next_payload_to_seal.take() else { return Ok(()) };
        let PendingBuild::Committed(sealed) = &pending.job else { return Ok(()) };
        loop {
            if !self.context_is_current(sealed.context()) {
                return Ok(());
            }
            if !self.conductor_overridden &&
                let Some(conductor) = &self.conductor
            {
                match conductor.leader().await {
                    Ok(true) => {}
                    Ok(false) | Err(_) => return Ok(()),
                }
            }
            // Leadership I/O may have crossed a reset. The engine also rechecks on application.
            if !self.context_is_current(sealed.context()) {
                return Ok(());
            }
            match self.import_committed(&pending).await {
                Err(SequencerActorError::EngineError(EngineClientError::Temporary(_))) => {
                    tokio::time::sleep(Duration::from_secs(1)).await;
                }
                Ok(()) |
                Err(SequencerActorError::EngineError(
                    EngineClientError::Stale |
                    EngineClientError::Expired |
                    EngineClientError::Rejected(_),
                )) => return Ok(()),
                Err(error) => return Err(error),
            }
        }
    }

    /// Commands queued during a bounded action run before any following build admission.
    async fn drain_commands(&mut self) -> Result<bool, SequencerActorError> {
        let mut handled = false;
        while let Ok(command) = self.commands.try_recv() {
            handled = true;
            self.handle_command(command).await?;
        }
        Ok(handled)
    }

    /// Starts building an L2 block by creating and populating payload attributes referencing the
    /// correct L1 origin block and sending them to the block engine.
    pub(super) async fn build_unsealed_payload(
        &mut self,
    ) -> Result<Option<UnsealedPayloadHandle>, SequencerActorError> {
        let state = self.engine_client.state();
        if matches!(state.phase, EnginePhase::Initializing | EnginePhase::Resetting) {
            return Ok(None);
        }
        let unsafe_head = state.sync_state.unsafe_head();
        let context = EngineContext { epoch: state.epoch, parent: unsafe_head };

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

        // A Stop or recovery-mode change queued during preparation supersedes these attributes.
        if self.drain_commands().await? || !self.state.active {
            return Ok(None);
        }

        // Send the built attributes to the engine to be built.
        let build_request_start = Instant::now();

        let token = self.engine_client.start_build(context, attributes_with_parent.clone()).await?;

        update_block_build_duration_metrics(build_request_start.elapsed());

        Ok(Some(UnsealedPayloadHandle {
            job: PendingBuild::Building(token),
            attributes_with_parent,
        }))
    }

    /// Determines and validates the L1 origin block for the provided L2 unsafe head.
    /// Returns `Ok(None)` for temporary errors that should be retried.
    async fn get_next_payload_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
    ) -> Result<Option<BlockInfo>, SequencerActorError> {
        let l1_origin = match self
            .origin_selector
            .next_l1_origin(unsafe_head, self.state.recovery_mode)
            .await
        {
            Ok(l1_origin) => l1_origin,
            Err(L1OriginSelectorError::OriginNotFound(hash)) => {
                warn!(
                    target: "sequencer",
                    %hash,
                    "L1 origin block not found, resetting engine"
                );
                self.engine_client.reset().await?;
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
            self.engine_client.reset().await?;
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
                if let Err(err) = self.engine_client.reset().await {
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
    async fn schedule_initial_reset(&self) -> Result<(), SequencerActorError> {
        // Reset the engine, in order to initialize the engine state.
        // NB: this call waits for confirmation that the reset succeeded and we can proceed with
        // post-reset logic.
        self.engine_client.reset().await.map_err(|err| {
            error!(target: "sequencer", ?err, "Failed to send reset request to engine");
            err.into()
        })
    }
}

#[async_trait]
impl<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_> NodeActor
    for SequencerActor<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_>
where
    AttributesBuilder_: AttributesBuilder + Sync + 'static,
    Conductor_: Conductor + Sync + 'static,
    OriginSelector_: OriginSelector + Sync + 'static,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient + Sync + 'static,
{
    type Error = SequencerActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        if !self.started {
            self.publish();
            // Reset the engine state prior to beginning block building.
            self.schedule_initial_reset().await?;
            self.started = true;
        }

        select! {
            // We are using a biased select here to ensure that the admin queries are given priority over the block building task.
            // This is important to limit the occurrence of race conditions where a stopped query is received when a sequencer is building a new block.
            biased;
            Some(command) = self.commands.recv() => {
                self.handle_command(command).await
            }
            // The sequencer must be active to build new blocks.
            _ = self.build_ticker.tick(), if self.state.active => {
                if !self.unsafe_payload_gossip_client.has_capacity() {
                    warn!(target: "sequencer", "Sequencing tick, gossip queue full, not building a block");
                    return Ok(());
                }
                info!(target: "sequencer", "Sequencing tick, building block");
                let mut may_build = true;
                if let Some(mut pending) = self.next_payload_to_seal.take() {
                    let started = Instant::now();
                    match self.publish_payload(&mut pending).await {
                        Ok(true) => self.last_seal_duration = started.elapsed(),
                        Ok(false) | Err(SequencerActorError::EngineError(EngineClientError::Temporary(_))) => {
                            self.next_payload_to_seal = Some(pending);
                            may_build = false;
                        }
                        Err(SequencerActorError::EngineError(error @ (EngineClientError::Stale | EngineClientError::Expired | EngineClientError::Rejected(_)))) => {
                            debug!(target:"sequencer",?error,"Discarding obsolete or rejected job");
                            may_build = false;
                        }
                        Err(error) => return Err(error),
                    }
                }
                self.drain_commands().await?;
                if !self.state.active || !may_build {
                    self.build_ticker.reset_after(Duration::from_secs(1));
                    return Ok(());
                }
                match self.build_unsealed_payload().await {
                    Ok(payload) => self.next_payload_to_seal = payload,
                    Err(SequencerActorError::EngineError(error @ (EngineClientError::Temporary(_) | EngineClientError::Stale | EngineClientError::Expired | EngineClientError::Rejected(_)))) => {
                        debug!(target:"sequencer",?error,"Build unavailable; retry on next tick");
                        self.build_ticker.reset();
                        return Ok(());
                    }
                    Err(error) => return Err(error),
                }

                if let Some(payload) = self.next_payload_to_seal.as_ref() {
                    let next_block_seconds = payload.attributes_with_parent.parent().block_info.timestamp.saturating_add(self.rollup_config.block_time);
                    // next block time is last + block_time - time it takes to seal.
                    let next_block_time = UNIX_EPOCH + Duration::from_secs(next_block_seconds) - self.last_seal_duration;
                    match next_block_time.duration_since(SystemTime::now()) {
                        Ok(duration) => self.build_ticker.reset_after(duration),
                        Err(_) => self.build_ticker.reset_immediately(),
                    };
                } else {
                    self.build_ticker.reset();
                }
                Ok(())
            }
            else => std::future::pending().await,
        }
    }
}
