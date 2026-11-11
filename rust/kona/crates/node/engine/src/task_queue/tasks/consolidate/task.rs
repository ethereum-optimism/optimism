//! A task to consolidate the engine state.

use super::el_sync_progress::{ElSyncProgress, ElSyncReading, ElSyncWaitOutcome, ElSyncWaitStep};
use crate::{
    ConsolidateTaskError, DelegatedSafeHeads, EngineClient, EngineState, ExecutableEngineTask,
    FinalityViolation, ImportedBlockSink,
    state::EngineSyncStateUpdate,
    task_queue::{
        ForkchoiceDelivery, ForkchoiceReply, build_and_import, record_forkchoice_reply,
        send_forkchoice, synchronize,
    },
};
use alloy_primitives::B256;
use alloy_provider::Provider;
use alloy_rpc_types_eth::Block;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types::Transaction;
use std::{
    sync::{Arc, Mutex, MutexGuard, PoisonError},
    time::Duration,
};
use tokio::time::Instant;

/// Input for consolidation - either derived attributes or delegated safe heads
#[derive(Debug, Clone)]
pub enum ConsolidateInput {
    /// Consolidate based on derived attributes.
    Attributes(Box<OpAttributesWithParent>),
    /// Derivation Delegation: consolidate onto the safe heads a derivation delegate reports.
    Delegated(Box<DelegatedSafeHeads>),
}

impl From<DelegatedSafeHeads> for ConsolidateInput {
    fn from(heads: DelegatedSafeHeads) -> Self {
        Self::Delegated(Box::new(heads))
    }
}

impl From<OpAttributesWithParent> for ConsolidateInput {
    fn from(v: OpAttributesWithParent) -> Self {
        Self::Attributes(Box::new(v))
    }
}

impl ConsolidateInput {
    /// Returns the block number for this consolidation input.
    const fn l2_block_number(&self) -> u64 {
        match self {
            Self::Attributes(attributes) => attributes.block_number(),
            Self::Delegated(heads) => heads.local_safe_head().block_info.number,
        }
    }

    /// Checks if the block is consistent with this consolidation input.
    fn is_consistent_with_block(&self, cfg: &RollupConfig, block: &Block<Transaction>) -> bool {
        match self {
            Self::Attributes(attributes) => {
                crate::AttributesMatch::check(cfg, attributes, block).is_match()
            }
            Self::Delegated(heads) => block.header.hash == heads.local_safe_head().block_info.hash,
        }
    }

    /// Whether these are derived attributes for a block before the last block of its span batch.
    const fn is_intermediate_span_block(&self) -> bool {
        matches!(self, Self::Attributes(attributes) if !attributes.is_last_in_span)
    }

    /// Derived attributes move the safe head on every consolidation, so an unchanged forkchoice was
    /// already sent. A delegated safe head can repeat a forkchoice the execution layer answered
    /// `SYNCING` or lost, so it is always sent.
    const fn forkchoice_delivery(&self) -> ForkchoiceDelivery {
        match self {
            Self::Attributes(_) => ForkchoiceDelivery::WhenChanged,
            Self::Delegated(_) => ForkchoiceDelivery::Always,
        }
    }
}

/// What a consolidation waiting for the execution layer does with a derived block it lacked.
#[derive(Debug)]
enum DerivedBlock {
    /// Build the block from the derived attributes on its parent, which the execution layer has.
    BuildFromAttributes,
    /// Consolidate the block the execution layer committed since the lookup.
    Committed(Box<Block<Transaction>>),
}

/// The [`ConsolidateTask`] attempts to consolidate the engine state
/// using the specified derived attributes or delegated safe heads.
#[derive(Debug, Clone)]
pub struct ConsolidateTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The input for consolidation (either derived attributes or delegated safe heads).
    pub input: ConsolidateInput,
    /// Where to hand the decoded block once the engine has canonicalized it.
    pub block_sink: Arc<dyn ImportedBlockSink>,
    /// The execution layer's progress across the retries of this task while it waits for the
    /// execution layer to sync.
    el_sync_progress: Arc<Mutex<ElSyncProgress>>,
}

impl ConsolidateTask {
    /// Creates a new [`ConsolidateTask`] with the specified input. A wait for the execution layer
    /// to sync resets the engine once its sync reports no progress for `el_sync_stall_timeout`.
    pub fn new(
        client: Arc<EngineClient>,
        cfg: Arc<RollupConfig>,
        input: ConsolidateInput,
        block_sink: Arc<dyn ImportedBlockSink>,
        el_sync_stall_timeout: Duration,
    ) -> Self {
        let el_sync_progress = Arc::new(Mutex::new(ElSyncProgress::new(el_sync_stall_timeout)));
        Self { block_sink, cfg, client, el_sync_progress, input }
    }

    /// The execution layer's progress across the retries of this task.
    fn el_sync_progress(&self) -> MutexGuard<'_, ElSyncProgress> {
        self.el_sync_progress.lock().unwrap_or_else(PoisonError::into_inner)
    }

    /// This is used when the [`ConsolidateTask`] fails to consolidate the engine state
    async fn build_and_import_attributes(
        &self,
        state: &mut EngineState,
        attributes: &OpAttributesWithParent,
    ) -> Result<(), ConsolidateTaskError> {
        build_and_import(
            self.client.as_ref(),
            &self.cfg,
            state,
            attributes.clone(),
            self.block_sink.as_ref(),
        )
        .await?;
        Ok(())
    }

    /// The hash of the execution layer's canonical block at `number`, or `None` when it has none
    /// there. Only the hash is needed, so the block is fetched without its transactions.
    async fn el_block_hash(&self, number: u64) -> Result<Option<B256>, ConsolidateTaskError> {
        debug!(target: "engine", number, "Looking up the execution layer's block hash");
        let block = self.client.get_l2_block(number.into()).await.map_err(|err| {
            warn!(target: "engine", ?err, number, "Failed to look up the execution layer's block");
            ConsolidateTaskError::FailedToFetchDelegatedSafeBlock(number)
        })?;
        let el_hash = block.map(|block| block.header.hash);
        debug!(target: "engine", number, ?el_hash, "Looked up the execution layer's block hash");
        Ok(el_hash)
    }

    /// The hash of the engine's unsafe or established finalized head at `number`, if one sits at
    /// that height. Only these two pin the engine's chain: the unsafe head is the head the
    /// execution layer follows, and finality cannot move, while the safe labels can stay on blocks
    /// that a delegate's reorg abandoned.
    fn chain_pinning_label_hash_at(state: &EngineState, number: u64) -> Option<B256> {
        let sync_state = &state.sync_state;
        Some(sync_state.unsafe_head())
            .filter(|unsafe_head| *unsafe_head != L2BlockInfo::default())
            .map(|unsafe_head| unsafe_head.block_info.id())
            .into_iter()
            .chain(sync_state.established_finalized_head())
            .find(|label| label.number == number)
            .map(|label| label.hash)
    }

    /// The safe label when the cross-safe head cannot be checked against the engine's chain: the
    /// cross-safe head only when it respects finality and advances the label.
    fn unverified_advancing_safe_head(
        state: &EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Option<L2BlockInfo> {
        Self::check_cross_safe_finality(state, heads)
            .ok()
            .and_then(|()| heads.advancing_safe_head(&state.sync_state.safe_head()))
    }

    /// The one check of the delegated cross-safe head against finality.
    fn check_cross_safe_finality(
        state: &EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<(), FinalityViolation> {
        state.sync_state.ensure_respects_finality(heads.safe_head().block_info.id())
    }

    /// The safe label when the unsafe head moves onto a chain the current safe label is not proven
    /// to be on: the cross-safe head, clamped to the finalized head; a conflict with finality
    /// skips.
    fn safe_label_off_engine_chain(
        state: &EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<L2BlockInfo, ConsolidateTaskError> {
        match Self::check_cross_safe_finality(state, heads) {
            Ok(()) => Ok(heads.safe_head()),
            Err(FinalityViolation::BehindFinalizedHead { .. }) => {
                Ok(state.sync_state.finalized_head())
            }
            Err(violation @ FinalityViolation::ConflictsWithFinalizedHead { .. }) => {
                Err(violation.into())
            }
        }
    }

    /// The safe label for delegated safe heads whose local-safe head is on the engine's chain: the
    /// cross-safe head when it advances the label, or when it replaces, at the label's height, a
    /// safe label that a reorg left on abandoned blocks, and `None`, which keeps the label.
    async fn safe_label_on_engine_chain(
        &self,
        state: &EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<Option<L2BlockInfo>, ConsolidateTaskError> {
        if let Err(violation) = Self::check_cross_safe_finality(state, heads) {
            debug!(target: "engine", %violation, ?heads, "Delegated cross-safe head breaks finality; keeping the safe label");
            return Ok(None);
        }
        let safe = state.sync_state.safe_head();
        let cross_safe = heads.safe_head();
        // Below the safe label the cross-safe head can neither advance nor replace it.
        if cross_safe.block_info.number < safe.block_info.number {
            return Ok(None);
        }

        let is_known_on_engine_chain = cross_safe.block_info.id() ==
            heads.local_safe_head().block_info.id() ||
            cross_safe.block_info.id() == safe.block_info.id();
        let chain_hash = if is_known_on_engine_chain {
            Some(cross_safe.block_info.hash)
        } else {
            self.engine_chain_hash_at(state, cross_safe.block_info.number).await?
        };
        if chain_hash.is_some_and(|hash| hash != cross_safe.block_info.hash) {
            debug!(target: "engine", ?heads, "Delegated cross-safe head is not on the chain of its local-safe head; keeping the safe label");
            return Ok(None);
        }
        let is_proven_on_engine_chain = chain_hash.is_some();
        let replaces_stale_safe_label = is_proven_on_engine_chain &&
            cross_safe.block_info.number == safe.block_info.number &&
            cross_safe.block_info.hash != safe.block_info.hash;
        Ok(if replaces_stale_safe_label {
            Some(cross_safe)
        } else {
            heads.advancing_safe_head(&safe)
        })
    }

    /// The hash of the engine's chain at `number`: its unsafe or finalized head when one sits
    /// there, and otherwise the execution layer's canonical block. `None` when nothing can tell.
    async fn engine_chain_hash_at(
        &self,
        state: &EngineState,
        number: u64,
    ) -> Result<Option<B256>, ConsolidateTaskError> {
        if let Some(label_hash) = Self::chain_pinning_label_hash_at(state, number) {
            return Ok(Some(label_hash));
        }
        self.el_canonical_hash_at(state, number).await
    }

    /// The hash of the execution layer's canonical block at `number`, only while that chain is the
    /// engine's: the execution layer applied the engine's last forkchoice, and `number` is at or
    /// below the unsafe head. Otherwise its canonical blocks may predate a sync, or sit above a
    /// rewound unsafe head, and prove nothing, so `None` without a lookup.
    async fn el_canonical_hash_at(
        &self,
        state: &EngineState,
        number: u64,
    ) -> Result<Option<B256>, ConsolidateTaskError> {
        let is_el_canonical_chain_current = state.el_applied_last_forkchoice &&
            number <= state.sync_state.unsafe_head().block_info.number;
        if is_el_canonical_chain_current { self.el_block_hash(number).await } else { Ok(None) }
    }

    /// Moves the unsafe and local-safe heads to the delegated local-safe head, and the safe label
    /// to `safe_head`.
    async fn reconcile_to_delegated_safe_heads(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
        safe_head: Option<L2BlockInfo>,
    ) -> Result<(), ConsolidateTaskError> {
        debug!(
            target: "engine",
            local_safe_l2 = %heads.local_safe_head(),
            safe_l2 = %heads.safe_head(),
            ?safe_head,
            "Apply safe head"
        );

        // We intentionally set the unsafe head to the local-safe head to ensure the engine observes
        // a self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the safe head.
        self.apply_delegated_safe_heads(
            state,
            heads,
            EngineSyncStateUpdate {
                local_safe_head: Some(heads.local_safe_head()),
                safe_head,
                unsafe_head: Some(heads.local_safe_head()),
                ..Default::default()
            },
            "Updated safe head via follow safe",
        )
        .await
    }

    /// Reorgs the engine onto delegated safe heads whose chain differs from the engine's, moving
    /// the safe label to the cross-safe head even when lower, since the old safe block may be on
    /// the abandoned chain, or to the finalized head when the cross-safe head is below it.
    async fn reorg_onto_delegated_safe_heads(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<(), ConsolidateTaskError> {
        let safe_head = Self::safe_label_off_engine_chain(state, heads)?;
        warn!(
            target: "engine",
            local_safe_l2 = %heads.local_safe_head(),
            %safe_head,
            "Reorging onto the delegate's chain"
        );
        self.reconcile_to_delegated_safe_heads(state, heads, Some(safe_head)).await
    }

    /// Moves the engine to delegated safe heads at or above the unsafe head. Only the unsafe head
    /// itself or its child extends the engine's chain; any other local-safe head has unknown
    /// ancestry and reorgs, so the safe label follows it and stays an ancestor of the new head.
    async fn extend_onto_delegated_safe_heads(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<(), ConsolidateTaskError> {
        let unsafe_head = state.sync_state.unsafe_head().block_info;
        let local_safe = heads.local_safe_head().block_info;
        let is_at_unsafe_height = local_safe.number == unsafe_head.number;
        if is_at_unsafe_height && local_safe.hash != unsafe_head.hash {
            return self.reorg_onto_delegated_safe_heads(state, heads).await;
        }
        let is_child_of_unsafe_head = local_safe.number == unsafe_head.number + 1 &&
            local_safe.parent_hash == unsafe_head.hash;
        let is_unproven_extension = !is_at_unsafe_height && !is_child_of_unsafe_head;
        let would_lower_safe_label =
            heads.safe_head().block_info.number < state.sync_state.safe_head().block_info.number;
        if is_unproven_extension &&
            !(would_lower_safe_label && self.is_safe_label_on_el_chain(state).await?)
        {
            return self.reorg_onto_delegated_safe_heads(state, heads).await;
        }

        let safe_head = self.safe_label_on_engine_chain(state, heads).await?;
        self.reconcile_to_delegated_safe_heads(state, heads, safe_head).await
    }

    /// Whether the execution layer's canonical chain holds the engine's safe label. A local-safe
    /// head several blocks above the unsafe head has ancestry the engine cannot check, but a
    /// delegate catching up builds on the safe label the execution layer holds, so that label stays
    /// rather than dropping to a lower cross-safe head.
    async fn is_safe_label_on_el_chain(
        &self,
        state: &EngineState,
    ) -> Result<bool, ConsolidateTaskError> {
        let safe = state.sync_state.safe_head().block_info;
        Ok(self.el_canonical_hash_at(state, safe.number).await? == Some(safe.hash))
    }

    /// Moves the safe labels to a delegated local-safe head that the syncing execution layer does
    /// not have yet, keeping the unsafe head as its sync target; the cross-safe head only advances.
    async fn follow_safe_head_during_el_sync(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<(), ConsolidateTaskError> {
        warn!(
            target: "engine",
            local_safe_l2 = %heads.local_safe_head(),
            unsafe_number = state.sync_state.unsafe_head().block_info.number,
            "Execution layer does not have the delegated safe head yet; keeping its sync target"
        );

        let safe_head = Self::unverified_advancing_safe_head(state, heads);
        self.apply_delegated_safe_heads(
            state,
            heads,
            EngineSyncStateUpdate {
                local_safe_head: Some(heads.local_safe_head()),
                safe_head,
                ..Default::default()
            },
            "Updated safe head while the execution layer syncs",
        )
        .await
    }

    /// Applies `update`, whose labels follow the delegated `heads`, and always sends its
    /// forkchoice update: the execution layer may have answered `SYNCING` to the same labels
    /// before, or lost them in a restart.
    async fn apply_delegated_safe_heads(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
        update: EngineSyncStateUpdate,
        outcome: &'static str,
    ) -> Result<(), ConsolidateTaskError> {
        synchronize(self.client.as_ref(), state, update, ForkchoiceDelivery::Always)
            .await
            .map_err(|e| {
                warn!(target: "engine", ?e, ?heads, "Applying the delegated safe heads failed");
                e
            })?;

        info!(
            target: "engine",
            local_safe_hash = %heads.local_safe_head().block_info.hash,
            local_safe_number = heads.local_safe_head().block_info.number,
            safe_hash = %state.sync_state.safe_head().block_info.hash,
            safe_number = state.sync_state.safe_head().block_info.number,
            "{outcome}"
        );

        Ok(())
    }

    /// Follows delegated safe heads below the engine's local-safe head. Only a different block at
    /// that height in a synced execution layer proves divergence and reorgs; otherwise the
    /// cross-safe head may only move the safe label. After a delegate L1 reorg whose fork point is
    /// above its reported head, the labels stay on the abandoned blocks until it passes that point.
    async fn follow_lagging_delegated_safe_heads(
        &self,
        state: &mut EngineState,
        heads: &DelegatedSafeHeads,
    ) -> Result<(), ConsolidateTaskError> {
        let local_safe = heads.local_safe_head().block_info;
        let el_hash = self.el_canonical_hash_at(state, local_safe.number).await?;
        let safe_head = match el_hash {
            Some(el_hash) if el_hash != local_safe.hash => {
                debug!(target: "engine", ?heads, %el_hash, "Delegated local-safe head behind the engine's diverges from the execution layer");
                return self.reorg_onto_delegated_safe_heads(state, heads).await;
            }
            Some(_) => self.safe_label_on_engine_chain(state, heads).await?,
            None => Self::unverified_advancing_safe_head(state, heads),
        };
        let Some(safe_head) = safe_head else {
            debug!(
                target: "engine",
                ?heads,
                local_safe_number = state.sync_state.local_safe_head().block_info.number,
                "Ignoring delegated safe heads behind the local-safe head"
            );
            return Ok(());
        };
        self.apply_delegated_safe_heads(
            state,
            heads,
            EngineSyncStateUpdate { safe_head: Some(safe_head), ..Default::default() },
            "Updated safe head from a delegate behind the local-safe head",
        )
        .await
    }

    /// Waits for the execution layer to sync a derived block that it did not have, at or below the
    /// unsafe head, and returns the block if the execution layer has it after all.
    ///
    /// Retries between checks only rely on the block lookup that precedes this call. A check reads
    /// the execution layer's sync progress and resends the engine's unchanged forkchoice:
    ///
    /// - `SYNCING`: the execution layer is still syncing toward the unsafe head and will fill the
    ///   block in, so the task waits. Resetting instead would restart from the execution layer's
    ///   own labels, which are genesis after a datadir wipe, and re-derive the chain in a loop.
    /// - `VALID`: the execution layer has the unsafe head, so it has the block unless it lost it.
    ///   It may have committed the block since the lookup, so the block is looked up once more.
    ///
    /// An execution layer that answers `SYNCING` while it reports no sync and sits at the derived
    /// block's parent is not fetching anything, as without peers, so the block is built from the
    /// derived attributes instead. A `SYNCING` answer after the progress has stalled for the bound
    /// resets the engine. Only a `VALID` answer changes the engine state, so a task that then waits
    /// has changed nothing.
    async fn derived_block_once_el_syncs(
        &self,
        state: &mut EngineState,
        block_number: u64,
    ) -> Result<DerivedBlock, ConsolidateTaskError> {
        let unsafe_number = state.sync_state.unsafe_head().block_info.number;
        if !self.el_sync_progress().is_check_due(Instant::now()) {
            debug!(
                target: "engine",
                block_number,
                unsafe_number,
                "Still waiting for the execution layer to sync the derived block"
            );
            return Err(ConsolidateTaskError::MissingUnsafeL2BlockDuringElSync(block_number));
        }

        let reading = self.el_sync_reading().await;
        // Every check probes, a stalled one included: the execution layer may have reached the
        // unsafe head since the last check, and then the block is consolidated instead of reset.
        if self.el_has_unsafe_head(state, block_number).await? {
            record_forkchoice_reply(state, ForkchoiceReply::Valid);
            return self
                .derived_block_committed_since_lookup(block_number)
                .await
                .map(|block| DerivedBlock::Committed(Box::new(block)));
        }
        // The reading counts, and its report goes out, only once the probe is answered.
        let now = Instant::now();
        self.el_sync_progress().check_answered(now);
        let step = self.el_sync_progress().record(reading.clone(), now);
        if reading
            .as_ref()
            .is_some_and(|reading| reading.is_idle_at(block_number.saturating_sub(1)))
        {
            info!(
                target: "engine",
                block_number,
                ?reading,
                unsafe_number,
                "Execution layer is idle at the derived block's parent; building the derived block"
            );
            return Ok(DerivedBlock::BuildFromAttributes);
        }
        if step.outcome == ElSyncWaitOutcome::Stalled {
            warn!(
                target: "engine",
                block_number,
                ?reading,
                unsafe_number,
                "Execution layer stopped making progress toward the unsafe head; resetting"
            );
            return Err(ConsolidateTaskError::MissingUnsafeL2Block(block_number));
        }
        Self::log_wait_for_el_sync(step, reading.as_ref(), block_number, unsafe_number);
        Err(ConsolidateTaskError::MissingUnsafeL2BlockDuringElSync(block_number))
    }

    /// Resends the engine's unchanged forkchoice and returns whether the execution layer applied
    /// it, rather than syncing toward its head.
    async fn el_has_unsafe_head(
        &self,
        state: &EngineState,
        block_number: u64,
    ) -> Result<bool, ConsolidateTaskError> {
        debug!(target: "engine", block_number, "Asking the execution layer whether it has the unsafe head");
        let reply = send_forkchoice(self.client.as_ref(), &state.sync_state).await.map_err(|err| {
            warn!(target: "engine", ?err, block_number, "Forkchoice update for a missing derived block failed");
            ConsolidateTaskError::MissingUnsafeL2BlockProbeFailed(err)
        })?;
        debug!(target: "engine", block_number, ?reply, "Execution layer answered the forkchoice update");
        Ok(reply == ForkchoiceReply::Valid)
    }

    /// Logs a retry that keeps waiting: at `info!` when the wait starts and when progress is
    /// reported, at `warn!` when no progress is reported on that cadence, and at `debug!`
    /// otherwise.
    fn log_wait_for_el_sync(
        step: ElSyncWaitStep,
        reading: Option<&ElSyncReading>,
        block_number: u64,
        unsafe_number: u64,
    ) {
        let time_until_reset = step.time_until_reset;
        match step.outcome {
            ElSyncWaitOutcome::Started => info!(
                target: "engine",
                block_number,
                ?reading,
                ?time_until_reset,
                unsafe_number,
                "Execution layer is syncing toward the unsafe head; waiting for the derived block"
            ),
            ElSyncWaitOutcome::Progressed if step.reports_progress => info!(
                target: "engine",
                block_number,
                ?reading,
                ?time_until_reset,
                unsafe_number,
                "Execution layer sync is progressing; still waiting for the derived block"
            ),
            _ if step.reports_progress => warn!(
                target: "engine",
                block_number,
                ?reading,
                ?time_until_reset,
                unsafe_number,
                "Execution layer reports no sync progress; the engine resets if none comes in time"
            ),
            _ => debug!(
                target: "engine",
                block_number,
                ?reading,
                ?time_until_reset,
                "Still waiting for the execution layer to sync the derived block"
            ),
        }
    }

    /// What the execution layer reports about its sync: the progress of the range it syncs, or
    /// its latest block number when it syncs none. `None` if that cannot be read.
    async fn el_sync_reading(&self) -> Option<ElSyncReading> {
        trace!(target: "engine", "Fetching the execution layer's sync status");
        match self.client.client().request_noparams::<serde_json::Value>("eth_syncing").await {
            Ok(serde_json::Value::Object(status)) => Some(ElSyncReading::from_sync_status(status)),
            Ok(serde_json::Value::Bool(false)) => self
                .el_latest_block_number()
                .await
                .map(|latest_number| ElSyncReading::NotSyncing { latest_number }),
            Ok(status) => {
                warn!(target: "engine", %status, "Execution layer returned an unexpected sync status");
                None
            }
            Err(err) => {
                warn!(target: "engine", ?err, "Failed to fetch the execution layer's sync status");
                None
            }
        }
    }

    /// The number of the execution layer's latest block, or `None` if it cannot be read.
    async fn el_latest_block_number(&self) -> Option<u64> {
        trace!(target: "engine", "Fetching the execution layer's latest block number");
        self.client
            .get_block_number()
            .await
            .map_err(|err| {
                warn!(target: "engine", ?err, "Failed to fetch the execution layer's latest block number");
            })
            .ok()
    }

    /// Looks up the derived block again after the execution layer answered `VALID` for the unsafe
    /// head; the engine resets if the block is still missing.
    async fn derived_block_committed_since_lookup(
        &self,
        block_number: u64,
    ) -> Result<Block<Transaction>, ConsolidateTaskError> {
        match self.client.l2_block_by_label(block_number.into()).await {
            Ok(Some(block)) => {
                info!(target: "engine", block_number, "Execution layer committed the derived block since the lookup");
                Ok(block)
            }
            Ok(None) => {
                warn!(target: "engine", block_number, "Execution layer has the unsafe head but not the derived block");
                Err(ConsolidateTaskError::MissingUnsafeL2Block(block_number))
            }
            Err(err) => {
                warn!(target: "engine", ?err, block_number, "Failed to look up the derived block again");
                Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block)
            }
        }
    }

    /// The safe label after consolidating onto `consolidated`, which the execution layer has: that
    /// block for derived attributes, and the judged cross-safe head for a delegate.
    async fn consolidated_safe_head(
        &self,
        state: &EngineState,
        consolidated: L2BlockInfo,
    ) -> Result<Option<L2BlockInfo>, ConsolidateTaskError> {
        let ConsolidateInput::Delegated(heads) = &self.input else {
            return Ok(Some(consolidated));
        };
        self.safe_label_on_engine_chain(state, heads).await
    }

    /// Handles a block that does not match the input: derived attributes build their own block,
    /// and delegated safe heads reorg the engine onto the delegate's chain.
    async fn reconcile_mismatch(
        &self,
        state: &mut EngineState,
    ) -> Result<(), ConsolidateTaskError> {
        match &self.input {
            ConsolidateInput::Attributes(attributes) => {
                self.build_and_import_attributes(state, attributes).await
            }
            ConsolidateInput::Delegated(heads) => {
                self.reorg_onto_delegated_safe_heads(state, heads).await
            }
        }
    }

    /// Handles input at or above the unsafe head: derived attributes build their block, and
    /// delegated safe heads move the engine onto them.
    async fn reconcile_unsafe_to_safe(
        &self,
        state: &mut EngineState,
    ) -> Result<(), ConsolidateTaskError> {
        match &self.input {
            ConsolidateInput::Attributes(attributes) => {
                self.build_and_import_attributes(state, attributes).await
            }
            ConsolidateInput::Delegated(heads) => {
                self.extend_onto_delegated_safe_heads(state, heads).await
            }
        }
    }

    /// Attempts consolidation on the engine state.
    #[tracing::instrument(
        name = "consolidate",
        target = "engine",
        level = "info",
        skip_all,
        fields(number = self.input.l2_block_number())
    )]
    pub async fn consolidate(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        // Fetch the unsafe L2 block
        let block_num = self.input.l2_block_number();
        let block = match self.client.l2_block_by_label(block_num.into()).await {
            Ok(Some(block)) => block,
            Ok(None) => {
                if let ConsolidateInput::Delegated(heads) = &self.input {
                    return self.follow_safe_head_during_el_sync(state, heads).await;
                }
                match self.derived_block_once_el_syncs(state, block_num).await? {
                    DerivedBlock::Committed(block) => *block,
                    DerivedBlock::BuildFromAttributes => {
                        return self.reconcile_mismatch(state).await;
                    }
                }
            }
            Err(_) => {
                return Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block);
            }
        };
        let block_hash = block.header.hash;

        if !self.input.is_consistent_with_block(&self.cfg, &block) {
            // A different block from an execution layer that has not applied the engine's
            // forkchoice may be from before its sync, so it does not prove the delegate diverged.
            if let ConsolidateInput::Delegated(heads) = &self.input &&
                !state.el_applied_last_forkchoice
            {
                return self.follow_safe_head_during_el_sync(state, heads).await;
            }
            debug!(
                target: "engine",
                input = ?self.input,
                block_hash = %block_hash,
                "ConsolidateInput mismatch! Initiating reorg",
            );
            return self.reconcile_mismatch(state).await;
        }

        trace!(
            target: "engine",
            input = ?self.input,
            block_hash = %block_hash,
            "Consolidating engine state",
        );
        let consensus_block =
            block.into_consensus().map_transactions(|t| t.inner.inner.into_inner());
        let block_info = match (
            L2BlockInfo::from_block_and_genesis(&consensus_block, &self.cfg.genesis),
            &self.input,
        ) {
            (Ok(block_info), _) => block_info,
            // The block matches the delegated local-safe head by hash, so the delegate's info
            // describes it, and the block stays where it is on the engine's chain.
            (Err(e), ConsolidateInput::Delegated(heads)) => {
                warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, using the delegated local-safe head's");
                heads.local_safe_head()
            }
            (Err(e), ConsolidateInput::Attributes(attributes)) => {
                // Continue on to build the block since we failed to construct the block info.
                warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, proceeding to build task");
                return self.build_and_import_attributes(state, attributes).await;
            }
        };

        // A span batch's blocks before its last one only move the safe label; the last block's
        // forkchoice update delivers them all. This avoids sending a FCU call for every block in
        // the span batch.
        if self.input.is_intermediate_span_block() {
            // The next attributes built are this block's child, and ask for its config.
            self.block_sink.block_imported(consensus_block, block_info);

            // Apply a transient update to the safe head.
            state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
                local_safe_head: Some(block_info),
                safe_head: Some(block_info),
                ..Default::default()
            });

            info!(
                target: "engine",
                hash = %block_info.block_info.hash,
                number = block_info.block_info.number,
                "Updated safe head via L1 consolidation"
            );

            return Ok(());
        }

        let safe_head = self.consolidated_safe_head(state, block_info).await?;
        // The next attributes built are this block's child, and ask for its config.
        self.block_sink.block_imported(consensus_block, block_info);

        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate {
                local_safe_head: Some(block_info),
                safe_head,
                ..Default::default()
            },
            self.input.forkchoice_delivery(),
        )
        .await
        .map_err(|e| {
            warn!(target: "engine", ?e, "Consolidation failed");
            e
        })?;

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Updated safe head via L1 consolidation"
        );

        Ok(())
    }
}

#[async_trait]
impl ExecutableEngineTask for ConsolidateTask {
    type Output = ();

    type Error = ConsolidateTaskError;

    // Behavior depends on how the safe head is provided:
    //
    // - `Attributes`: The safe head is advanced through the normal derivation flow, where the
    //   DerivationActor and EngineActor coordinate both safe and unsafe heads. In this case, we
    //   consolidate as long as the unsafe head has not fallen behind.
    //
    // - `Delegated`: The safe heads are injected externally by the DerivationActor while delegating
    //   derivation, and are not coordinated with the EngineActor's heads. A local-safe head below
    //   the engine's comes from a lagging or diverged delegate; one below the unsafe head is
    //   consolidated against the execution layer's block; one at or above the unsafe head moves the
    //   unsafe chain onto it.
    async fn execute(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        let ConsolidateInput::Delegated(heads) = &self.input else {
            return if state.sync_state.safe_head().block_info.number <
                state.sync_state.unsafe_head().block_info.number
            {
                self.consolidate(state).await
            } else {
                self.reconcile_unsafe_to_safe(state).await
            };
        };

        let local_safe_number = heads.local_safe_head().block_info.number;
        state.sync_state.ensure_respects_finality(heads.local_safe_head().block_info.id())?;
        // Labels with the unsafe head below the safe labels take the paths below, which move them
        // back into order; ignoring a lagging delegate would keep them broken until a finalization
        // above the unsafe head fails.
        let unsafe_number = state.sync_state.unsafe_head().block_info.number;
        let are_labels_consistent = unsafe_number >=
            state.sync_state.local_safe_head().block_info.number &&
            unsafe_number >= state.sync_state.safe_head().block_info.number;
        if are_labels_consistent &&
            local_safe_number < state.sync_state.local_safe_head().block_info.number
        {
            self.follow_lagging_delegated_safe_heads(state, heads).await
        } else if local_safe_number < state.sync_state.unsafe_head().block_info.number {
            self.consolidate(state).await
        } else {
            self.reconcile_unsafe_to_safe(state).await
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineSyncState, EngineTaskError,
        EngineTaskErrorSeverity::{self, Reset, Temporary, Wait},
        FinalityViolation, ImportedBlockSink, NoopBlockSink, SealTaskError,
        test_utils::{
            RpcMock, TestAttributesBuilder, TestEngineStateBuilder, delegated_safe_heads,
            l1_info_deposit, test_block_info, test_engine_client, test_rpc_block_from_attributes,
        },
    };
    use alloy_eips::{BlockNumHash, Encodable2718};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        ForkchoiceUpdated, INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatus, PayloadStatusEnum,
    };
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use kona_genesis::ChainGenesis;
    use kona_protocol::BlockInfo;

    /// Records the blocks the engine hands over after consolidating.
    #[derive(Debug, Default)]
    struct RecordingSink(std::sync::Mutex<Vec<B256>>);

    impl ImportedBlockSink for RecordingSink {
        fn block_imported(&self, _: op_alloy_consensus::OpBlock, info: L2BlockInfo) {
            self.0.lock().unwrap().push(info.block_info.hash);
        }
    }

    #[tokio::test]
    async fn consolidated_blocks_are_handed_to_the_block_sink() {
        // Consolidation reads the unsafe block in full to compare it against the derived
        // attributes; the next attributes built ask for this same block's system config, so it
        // must reach the sink rather than being dropped.
        let (block, cfg) = genesis_pinned_block(0);
        let block_hash = block.header.hash;
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x0", true]), block);
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        let client = Arc::new(client);

        let safe_l2 = L2BlockInfo {
            block_info: BlockInfo { hash: block_hash, number: 0, ..Default::default() },
            ..Default::default()
        };
        let sink = Arc::new(RecordingSink::default());
        let task = ConsolidateTask::new(
            client,
            cfg,
            delegated_safe_heads(safe_l2, safe_l2),
            sink.clone(),
            STALL,
        );

        task.consolidate(&mut EngineState::default()).await.unwrap();

        assert_eq!(sink.0.lock().unwrap().as_slice(), &[block_hash]);
        l1.assert_finished();
        l2.assert_finished();
    }

    const DEPOSIT: u8 = op_alloy_consensus::DEPOSIT_TX_TYPE_ID;

    fn holocene_cfg() -> Arc<RollupConfig> {
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        cfg.hardforks.holocene_time = Some(0);
        Arc::new(cfg)
    }

    fn derived(transactions: Vec<alloy_primitives::Bytes>) -> OpAttributesWithParent {
        crate::test_utils::TestAttributesBuilder::new()
            .with_parent(crate::test_utils::test_block_info(10))
            .with_transactions(transactions)
            .build()
    }

    async fn build_derived(
        client: Arc<EngineClient>,
        cfg: Arc<RollupConfig>,
        attributes: &OpAttributesWithParent,
    ) -> ConsolidateTaskError {
        let mut state = crate::test_utils::TestEngineStateBuilder::new()
            .with_unsafe_head(attributes.parent)
            .build();
        let task = ConsolidateTask::new(
            client,
            cfg,
            ConsolidateInput::Attributes(Box::new(attributes.clone())),
            Arc::new(crate::NoopBlockSink),
            STALL,
        );
        task.build_and_import_attributes(&mut state, attributes)
            .await
            .expect_err("the mock never completes a seal")
    }

    // An FCU rejection with -38003 (invalid payload attributes) triggers deposits-only recovery.
    #[tokio::test]
    async fn attributes_rejected_at_fcu_are_replaced_with_deposits_only() {
        let deposit = alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]);
        let attributes = derived(vec![deposit.clone(), alloy_primitives::Bytes::from(vec![0x02])]);
        let cfg = holocene_cfg();
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        let forkchoice = crate::test_utils::TestEngineStateBuilder::new()
            .with_unsafe_head(attributes.parent)
            .build()
            .sync_state
            .create_forkchoice_state();
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, attributes.as_deposits_only().attributes]),
            alloy_rpc_types_engine::ForkchoiceUpdated {
                payload_status: alloy_rpc_types_engine::PayloadStatus::from_status(
                    alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                ),
                payload_id: Some(alloy_rpc_types_engine::PayloadId::new([1; 8])),
            },
        );
        l2.expect_error("engine_getPayloadV3");
        let err = build_derived(Arc::new(client), cfg, &attributes).await;
        l1.assert_finished();
        l2.assert_finished();
        // The mock fails the replacement's getPayload.
        assert!(
            matches!(
                err,
                ConsolidateTaskError::SealTaskFailed(
                    SealTaskError::DepositOnlyPayloadReattemptFailed
                )
            ),
            "got {err:?}",
        );
    }

    // An internal error at FCU is a failure of the execution layer, not of the batch.
    #[tokio::test]
    async fn internal_error_at_fcu_is_retried_not_replaced() {
        let attributes = derived(vec![
            alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]),
            alloy_primitives::Bytes::from(vec![0x02]),
        ]);
        let cfg = holocene_cfg();
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_error("engine_forkchoiceUpdatedV3");
        let err = build_derived(Arc::new(client), cfg, &attributes).await;
        l1.assert_finished();
        l2.assert_finished();
        assert!(matches!(err, ConsolidateTaskError::BuildTaskFailed(_)), "got {err:?}");
        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Temporary);
    }

    #[tokio::test]
    async fn transport_failure_at_fcu_is_retried_not_replaced() {
        let attributes = derived(vec![
            alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]),
            alloy_primitives::Bytes::from(vec![0x02]),
        ]);
        let cfg = holocene_cfg();
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_transport_error("engine_forkchoiceUpdatedV3");
        let err = build_derived(Arc::new(client), cfg, &attributes).await;
        l1.assert_finished();
        l2.assert_finished();
        assert!(matches!(err, ConsolidateTaskError::BuildTaskFailed(_)), "got {err:?}");
        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Temporary);
    }

    // Before Holocene, nothing replaces rejected attributes, so the task keeps the rejection.
    #[tokio::test]
    async fn attributes_rejected_at_fcu_before_holocene_are_not_replaced() {
        let attributes = derived(vec![
            alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]),
            alloy_primitives::Bytes::from(vec![0x02]),
        ]);
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        let cfg = Arc::new(cfg);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        let err = build_derived(Arc::new(client), cfg, &attributes).await;
        l1.assert_finished();
        l2.assert_finished();
        assert!(
            matches!(&err, ConsolidateTaskError::BuildTaskFailed(build) if build.is_invalid_attributes()),
            "got {err:?}"
        );
    }

    /// An engine state with the unsafe head at 20, the finalized head at 5 and `safe_head` between.
    fn engine_state_with_safe_head(safe_head: L2BlockInfo) -> EngineState {
        TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_local_safe_head(safe_head)
            .with_safe_head(safe_head)
            .with_finalized_head(test_block_info(5))
            .build()
    }

    fn with_safe_head(state: &EngineState, safe_head: L2BlockInfo) -> EngineSyncState {
        state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(safe_head),
            safe_head: Some(safe_head),
            ..Default::default()
        })
    }

    fn block_info_of(cfg: &RollupConfig, block: &RpcBlock<Transaction>) -> L2BlockInfo {
        let consensus_block =
            block.clone().into_consensus().map_transactions(|t| t.inner.inner.into_inner());
        L2BlockInfo::from_block_and_genesis(&consensus_block, &cfg.genesis).unwrap()
    }

    /// An empty block at `number`, with genesis pinned to it so its `L2BlockInfo` needs no L1-info
    /// deposit.
    fn genesis_pinned_block(number: u64) -> (RpcBlock<Transaction>, Arc<RollupConfig>) {
        let header = RpcHeader::new(alloy_consensus::Header { number, ..Default::default() });
        let cfg = Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: header.hash, number },
                ..Default::default()
            },
            ..Default::default()
        });
        (RpcBlock::new(header, BlockTransactions::Full(vec![])), cfg)
    }

    /// What the execution layer answers for the block at the delegated safe head's height.
    fn el_block(el_has_block: bool, block: RpcBlock<Transaction>) -> Option<RpcBlock<Transaction>> {
        el_has_block.then_some(block)
    }

    /// The safe head before consolidation: either already the delegated target, as a `SYNCING`
    /// answer or an earlier poll leaves it, or an older block.
    fn initial_safe_head(starts_at_target: bool, target: L2BlockInfo) -> L2BlockInfo {
        if starts_at_target { target } else { test_block_info(5) }
    }

    /// Below the unsafe head, a delegated safe head always reaches the execution layer: when the
    /// labels already hold it, and when the execution layer is still syncing and does not have its
    /// block yet. A missing block then moves only the safe labels instead of resetting, and only a
    /// block the execution layer has reaches the block sink.
    #[rstest::rstest]
    #[case::new_target(true, false, PayloadStatusEnum::Valid)]
    #[case::repeated_target(true, true, PayloadStatusEnum::Valid)]
    #[case::new_target_missing_from_syncing_el(false, false, PayloadStatusEnum::Syncing)]
    #[case::repeated_target_missing_from_syncing_el(false, true, PayloadStatusEnum::Syncing)]
    #[tokio::test]
    async fn delegated_safe_head_below_unsafe_head_is_always_sent(
        #[case] el_has_block: bool,
        #[case] starts_at_target: bool,
        #[case] reply: PayloadStatusEnum,
    ) {
        let (block, cfg) = genesis_pinned_block(10);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state =
            engine_state_with_safe_head(initial_safe_head(starts_at_target, delegated_safe_head));
        let expected_sync_state = with_safe_head(&state, delegated_safe_head);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!(["0xa", true]),
            el_block(el_has_block, block),
        );
        l2.expect_forkchoice_updated(expected_sync_state.create_forkchoice_state(), reply);
        let sink = Arc::new(RecordingSink::default());
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            delegated_safe_heads(delegated_safe_head, delegated_safe_head),
            sink.clone(),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        assert_eq!(sink.0.lock().unwrap().len(), usize::from(el_has_block));
        l1.assert_finished();
        l2.assert_finished();
    }

    const STALL_SECS: u64 = 3600;
    const STALL: Duration = Duration::from_secs(STALL_SECS);

    /// A derived-attributes task for block 10 and an engine state whose unsafe head is above it.
    fn task_for_missing_derived_block(client: EngineClient) -> (ConsolidateTask, EngineState) {
        let attributes = derived_attributes(true);
        let state = engine_state_with_safe_head(attributes.parent);
        let task = ConsolidateTask::new(
            Arc::new(client),
            Arc::new(RollupConfig::default()),
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
            STALL,
        );
        (task, state)
    }

    /// The execution layer lacks block 10, reports no sync in progress, and answers the unchanged
    /// forkchoice with `reply`.
    fn el_lacks_derived_block(l2: &RpcMock, state: &EngineState, reply: PayloadStatusEnum) {
        el_lacks_derived_block_and_reports(l2, IdleAt(100));
        l2.expect_forkchoice_updated(state.sync_state.create_forkchoice_state(), reply);
    }

    /// The execution layer lacks block 10 and reports `report` about its sync.
    fn el_lacks_derived_block_and_reports(l2: &RpcMock, report: ElReports) {
        l2.expect_block_by_number(10, None);
        report.expect_on(l2);
    }

    fn el_syncing(l2: &RpcMock, state: &EngineState) {
        el_lacks_derived_block(l2, state, PayloadStatusEnum::Syncing);
    }

    fn el_valid_and_block_still_missing(l2: &RpcMock, state: &EngineState) {
        el_lacks_derived_block(l2, state, PayloadStatusEnum::Valid);
        l2.expect_block_by_number(10, None);
    }

    fn el_valid_and_recheck_fails(l2: &RpcMock, state: &EngineState) {
        el_lacks_derived_block(l2, state, PayloadStatusEnum::Valid);
        l2.expect_error("eth_getBlockByNumber");
    }

    fn el_invalid(l2: &RpcMock, state: &EngineState) {
        el_lacks_derived_block(
            l2,
            state,
            PayloadStatusEnum::Invalid { validation_error: "bad block".into() },
        );
    }

    fn forkchoice_update_fails(l2: &RpcMock, _: &EngineState) {
        el_lacks_derived_block_and_reports(l2, IdleAt(100));
        l2.expect_error("engine_forkchoiceUpdatedV3");
    }

    fn forkchoice_state_rejected(l2: &RpcMock, _: &EngineState) {
        el_lacks_derived_block_and_reports(l2, IdleAt(100));
        l2.expect_error_code("engine_forkchoiceUpdatedV3", INVALID_FORK_CHOICE_STATE_ERROR as i64);
    }

    /// Derived attributes whose block is missing below the unsafe head resend the unchanged
    /// forkchoice. An execution layer still syncing toward the unsafe head makes the task wait, and
    /// a failed RPC is retried on the failure backoff; one that has the unsafe head but still not
    /// the block, or rejects the forkchoice, resets the engine. The labels stay unchanged.
    #[rstest::rstest]
    #[case::el_syncing_waits(el_syncing, Wait)]
    #[case::el_valid_and_block_still_missing_resets(el_valid_and_block_still_missing, Reset)]
    #[case::el_valid_and_recheck_failing_is_retried(el_valid_and_recheck_fails, Temporary)]
    #[case::el_invalid_resets(el_invalid, Reset)]
    #[case::forkchoice_update_failing_is_retried(forkchoice_update_fails, Temporary)]
    #[case::forkchoice_state_rejected_resets(forkchoice_state_rejected, Reset)]
    #[tokio::test]
    async fn derived_block_missing_from_el_asks_whether_it_syncs(
        #[case] el_answers: fn(&RpcMock, &EngineState),
        #[case] expected_severity: EngineTaskErrorSeverity,
    ) {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        let state_before = state;
        el_answers(&l2, &state);

        let err = task.execute(&mut state).await.unwrap_err();

        assert_eq!(err.severity(), expected_severity, "got {err:?}");
        assert_eq!(state, state_before);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A probe answered `VALID` tells the engine the execution layer applied its forkchoice and
    /// finished syncing, even when the recheck of the block then fails; only the labels stay.
    #[tokio::test]
    async fn valid_probe_records_the_el_sync_even_when_the_recheck_fails() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        state.el_applied_last_forkchoice = false;
        state.el_sync_finished = false;
        let sync_state_before = state.sync_state;
        el_valid_and_recheck_fails(&l2, &state);

        let err = task.execute(&mut state).await.unwrap_err();

        assert_eq!(err.severity(), Temporary, "got {err:?}");
        assert!(state.el_applied_last_forkchoice);
        assert!(state.el_sync_finished);
        assert_eq!(state.sync_state, sync_state_before);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An execution layer that commits the derived block between the lookup and the forkchoice
    /// update answers `VALID`; the block is then consolidated instead of resetting the engine.
    #[tokio::test]
    async fn derived_block_committed_during_the_probe_is_consolidated() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        let ConsolidateInput::Attributes(attributes) = &task.input else { unreachable!() };
        let block = test_rpc_block_from_attributes(attributes);
        let expected_sync_state =
            with_safe_head(&state, block_info_of(&RollupConfig::default(), &block));
        el_lacks_derived_block(&l2, &state, PayloadStatusEnum::Valid);
        l2.expect_block_by_number(10, Some(block));
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// What the execution layer reports about its sync on one retry.
    #[derive(Debug, Clone, Copy)]
    enum ElReports {
        /// No sync in progress, at this latest block.
        IdleAt(u64),
        /// A sync in progress, at this current block.
        SyncingAt(u64),
    }
    use ElReports::{IdleAt, SyncingAt};

    impl ElReports {
        fn expect_on(self, l2: &RpcMock) {
            match self {
                IdleAt(latest_number) => l2.expect_el_not_syncing_at(latest_number),
                SyncingAt(current_number) => l2.expect_el_syncing_at(current_number),
            }
        }
    }

    /// One retry of a task waiting for block 10, `elapsed_secs` after the previous one. The
    /// execution layer lacks the block; on a check it also reports `check` about its sync and
    /// answers the resent forkchoice with `SYNCING`.
    #[derive(Debug, Clone, Copy)]
    struct Retry {
        check: Option<ElReports>,
        elapsed_secs: u64,
    }

    /// A retry that checks on the execution layer.
    const fn checking(elapsed_secs: u64, report: ElReports) -> Retry {
        Retry { check: Some(report), elapsed_secs }
    }

    /// A retry between checks, which only looks the block up.
    const fn looking_up(elapsed_secs: u64) -> Retry {
        Retry { check: None, elapsed_secs }
    }

    impl Retry {
        fn expect_on(self, l2: &RpcMock, state: &EngineState) {
            l2.expect_block_by_number(10, None);
            if let Some(report) = self.check {
                report.expect_on(l2);
                l2.expect_forkchoice_updated(
                    state.sync_state.create_forkchoice_state(),
                    PayloadStatusEnum::Syncing,
                );
            }
        }
    }

    const CHECK_SECS: u64 = super::super::el_sync_progress::CHECK_INTERVAL.as_secs();

    /// Across the retries of one task, the execution layer reports its sync on each check. Each
    /// retry yields one severity: the task waits while the report changes, and resets once it has
    /// stayed the same for the bound and the final check's forkchoice is still answered
    /// `SYNCING`. Retries between checks make no RPC but the block lookup.
    #[rstest::rstest]
    #[case::progressing_el_keeps_waiting(&[checking(0, IdleAt(100)), checking(STALL_SECS - 1, IdleAt(101)), checking(STALL_SECS - 1, IdleAt(102))], &[Wait, Wait, Wait])]
    #[case::stuck_el_resets_after_the_bound(&[checking(0, IdleAt(100)), checking(STALL_SECS - CHECK_SECS, IdleAt(100)), checking(CHECK_SECS, IdleAt(100))], &[Wait, Wait, Reset])]
    #[case::stuck_sync_resets_after_the_bound(&[checking(0, SyncingAt(100)), checking(STALL_SECS, SyncingAt(100))], &[Wait, Reset])]
    #[case::retries_between_checks_only_look_up(&[checking(0, IdleAt(100)), looking_up(2), looking_up(2)], &[Wait, Wait, Wait])]
    #[case::check_interval_elapsing_checks(&[checking(0, IdleAt(100)), looking_up(2), checking(CHECK_SECS - 2, IdleAt(100))], &[Wait, Wait, Wait])]
    #[tokio::test(start_paused = true)]
    async fn wait_for_el_sync_is_bounded_by_its_progress(
        #[case] retries: &[Retry],
        #[case] expected_severities: &[EngineTaskErrorSeverity],
    ) {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        let mut severities = Vec::new();
        for retry in retries {
            tokio::time::advance(Duration::from_secs(retry.elapsed_secs)).await;
            retry.expect_on(&l2, &state);
            severities.push(task.execute(&mut state).await.unwrap_err().severity());
        }

        assert_eq!(severities, expected_severities);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The check that finds the progress stalled still resends the forkchoice. An execution layer
    /// that reached the unsafe head since the last check answers `VALID`, and the derived block is
    /// consolidated instead of resetting the engine.
    #[tokio::test(start_paused = true)]
    async fn stalled_el_that_reached_the_unsafe_head_is_consolidated() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        checking(0, IdleAt(100)).expect_on(&l2, &state);
        let first = task.execute(&mut state).await.unwrap_err().severity();

        tokio::time::advance(STALL).await;
        let ConsolidateInput::Attributes(attributes) = &task.input else { unreachable!() };
        let block = test_rpc_block_from_attributes(attributes);
        let expected_sync_state =
            with_safe_head(&state, block_info_of(&RollupConfig::default(), &block));
        el_lacks_derived_block(&l2, &state, PayloadStatusEnum::Valid);
        l2.expect_block_by_number(10, Some(block));
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
        task.execute(&mut state).await.unwrap();

        assert_eq!(first, Wait);
        assert_eq!(state.sync_state, expected_sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A check whose forkchoice update fails is not counted as answered, so the next retry checks
    /// again instead of waiting for the check interval.
    #[tokio::test(start_paused = true)]
    async fn failed_probe_is_retried_on_the_next_retry() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        forkchoice_update_fails(&l2, &state);
        let first = task.execute(&mut state).await.unwrap_err().severity();

        tokio::time::advance(Duration::from_secs(2)).await;
        checking(0, IdleAt(100)).expect_on(&l2, &state);
        let second = task.execute(&mut state).await.unwrap_err().severity();

        assert_eq!([first, second], [Temporary, Wait]);
        l1.assert_finished();
        l2.assert_finished();
    }

    fn derived_attributes(is_last_in_span: bool) -> OpAttributesWithParent {
        let mut attributes = TestAttributesBuilder::new()
            .with_parent(test_block_info(9))
            .with_transactions(vec![l1_info_deposit().encoded_2718().into()])
            .build();
        attributes.is_last_in_span = is_last_in_span;
        attributes
    }

    /// Derived blocks inside a span batch only move the safe label; the span's last block sends
    /// the forkchoice update that delivers it.
    #[rstest::rstest]
    #[case::intermediate_block_defers_forkchoice(false, 0)]
    #[case::last_block_sends_forkchoice(true, 1)]
    #[tokio::test]
    async fn span_batch_sends_forkchoice_only_at_its_last_block(
        #[case] is_last_in_span: bool,
        #[case] forkchoice_updates: usize,
    ) {
        let cfg = Arc::new(RollupConfig::default());
        let attributes = derived_attributes(is_last_in_span);
        let block = test_rpc_block_from_attributes(&attributes);
        let derived_safe_head = block_info_of(&cfg, &block);
        let mut state = engine_state_with_safe_head(attributes.parent);
        let expected_sync_state = with_safe_head(&state, derived_safe_head);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0xa", true]), block);
        for _ in 0..forkchoice_updates {
            l2.expect_forkchoice_updated(
                expected_sync_state.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        }
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Where a delegated local-safe head sits against the finalized head.
    #[derive(Clone, Copy, Debug)]
    enum TargetAgainstFinalized {
        Below,
        ConflictingAt,
    }

    impl TargetAgainstFinalized {
        fn delegated_safe_head(self, finalized: L2BlockInfo) -> L2BlockInfo {
            let number = finalized.block_info.number;
            match self {
                Self::Below => test_block_info(number - 1),
                Self::ConflictingAt => L2BlockInfo {
                    block_info: BlockInfo {
                        hash: !finalized.block_info.hash,
                        ..finalized.block_info
                    },
                    ..finalized
                },
            }
        }
    }

    fn is_behind_finality(result: &Result<(), ConsolidateTaskError>) -> bool {
        matches!(
            result,
            Err(ConsolidateTaskError::BreaksFinality(
                FinalityViolation::BehindFinalizedHead { .. }
            ))
        )
    }

    fn is_conflicting_with_finality(result: &Result<(), ConsolidateTaskError>) -> bool {
        matches!(
            result,
            Err(ConsolidateTaskError::BreaksFinality(
                FinalityViolation::ConflictsWithFinalizedHead { .. }
            ))
        )
    }

    /// A delegated local-safe head that breaks finality, as a stale delegate snapshot reports, is
    /// skipped before the execution layer is asked anything, and leaves every label where it was.
    #[rstest::rstest]
    #[case::below(TargetAgainstFinalized::Below, is_behind_finality)]
    #[case::conflicting(TargetAgainstFinalized::ConflictingAt, is_conflicting_with_finality)]
    #[tokio::test]
    async fn delegated_local_safe_head_breaking_finality_is_skipped(
        #[case] target: TargetAgainstFinalized,
        #[case] is_expected: fn(&Result<(), ConsolidateTaskError>) -> bool,
    ) {
        let mut state = engine_state_with_safe_head(test_block_info(12));
        let state_before = state;
        let delegated = target.delegated_safe_head(state.sync_state.finalized_head());

        let result = run_delegated(&mut state, delegated, delegated, |_| {}).await;

        assert!(is_expected(&result), "got {result:?}");
        assert_eq!(result.unwrap_err().severity(), crate::EngineTaskErrorSeverity::Skip);
        assert_eq!(state, state_before);
    }

    /// A block at `number` on the chain identified by `chain`, and its info. Blocks on different
    /// chains at the same height have different hashes.
    fn chain_block(number: u64, chain: u64) -> (RpcBlock<Transaction>, L2BlockInfo) {
        let header = RpcHeader::new(alloy_consensus::Header {
            number,
            timestamp: chain,
            ..Default::default()
        });
        let info = L2BlockInfo {
            block_info: BlockInfo { hash: header.hash, number, ..Default::default() },
            ..Default::default()
        };
        (RpcBlock::new(header, BlockTransactions::Full(vec![])), info)
    }

    /// The engine's chain.
    const ENGINE_CHAIN: u64 = 0;
    /// A chain that diverged from the engine's.
    const OTHER_CHAIN: u64 = 1;

    fn engine_chain_block(number: u64) -> L2BlockInfo {
        chain_block(number, ENGINE_CHAIN).1
    }

    fn other_chain_block(number: u64) -> L2BlockInfo {
        chain_block(number, OTHER_CHAIN).1
    }

    /// What the execution layer holds at a height: the engine chain's block, another chain's, or
    /// nothing.
    #[derive(Clone, Copy, Debug)]
    enum ElHas {
        EngineChain,
        Missing,
        OtherChain,
    }

    impl ElHas {
        fn block(self, number: u64) -> Option<RpcBlock<Transaction>> {
            match self {
                Self::EngineChain => Some(chain_block(number, ENGINE_CHAIN).0),
                Self::Missing => None,
                Self::OtherChain => Some(chain_block(number, OTHER_CHAIN).0),
            }
        }
    }

    /// An engine state with the unsafe head at 20 and the finalized head at 5 on the engine's
    /// chain, and its local-safe and safe heads at the given heights, which differ under interop.
    fn engine_state_with_safe_labels(local_safe: u64, safe: u64) -> EngineState {
        TestEngineStateBuilder::new()
            .with_unsafe_head(engine_chain_block(20))
            .with_local_safe_head(engine_chain_block(local_safe))
            .with_safe_head(engine_chain_block(safe))
            .with_finalized_head(engine_chain_block(5))
            .build()
    }

    /// Runs a consolidation of the delegated `local_safe` and `cross_safe` heads against an
    /// execution layer that `expect_engine_calls` scripts, and checks every scripted call was made.
    /// Genesis is pinned to the local-safe head, so its empty block decodes without an L1-info
    /// deposit.
    async fn run_delegated(
        state: &mut EngineState,
        local_safe: L2BlockInfo,
        cross_safe: L2BlockInfo,
        expect_engine_calls: impl FnOnce(&RpcMock),
    ) -> Result<(), ConsolidateTaskError> {
        let cfg = Arc::new(RollupConfig {
            genesis: ChainGenesis { l2: local_safe.block_info.id(), ..Default::default() },
            ..Default::default()
        });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        expect_engine_calls(&l2);
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            delegated_safe_heads(local_safe, cross_safe),
            Arc::new(NoopBlockSink),
            STALL,
        );

        let result = task.execute(state).await;

        l1.assert_finished();
        l2.assert_finished();
        result
    }

    /// The labels after the engine moves its local-safe head, and its safe head to `safe`.
    fn labels_after(
        state: &EngineState,
        local_safe: L2BlockInfo,
        safe: L2BlockInfo,
    ) -> EngineSyncState {
        state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            safe_head: Some(safe),
            ..Default::default()
        })
    }

    /// The labels after a reorg onto the delegated heads.
    fn labels_after_reorg(
        state: &EngineState,
        local_safe: L2BlockInfo,
        cross_safe: L2BlockInfo,
    ) -> EngineSyncState {
        state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            safe_head: Some(cross_safe),
            unsafe_head: Some(local_safe),
            ..Default::default()
        })
    }

    /// Below the unsafe head, a delegated local-safe head on the engine's chain that the execution
    /// layer has becomes the local-safe label, and a cross-safe head on that chain above the safe
    /// label moves it once the execution layer's block at its height confirms it.
    #[tokio::test]
    async fn advancing_cross_safe_head_moves_the_safe_label() {
        let (local_safe, cross_safe) = (engine_chain_block(10), engine_chain_block(8));
        let mut state = engine_state_with_safe_labels(9, 7);
        let expected = labels_after(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_by_number(10, ElHas::EngineChain.block(10));
            l2.expect_block_header_by_number(8, ElHas::EngineChain.block(8));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// The safe label's own block is known, so a cross-safe head there needs no lookup.
    fn cross_safe_at_safe_label(l2: &RpcMock) {
        l2.expect_block_by_number(10, ElHas::EngineChain.block(10));
    }

    /// A cross-safe head behind the safe label can neither advance nor replace it, so it is not
    /// looked up.
    fn cross_safe_behind_safe_label(l2: &RpcMock) {
        l2.expect_block_by_number(10, ElHas::EngineChain.block(10));
    }

    /// The execution layer still syncs toward the local-safe head, so the cross-safe head is not
    /// looked up.
    fn el_syncing_cross_safe_behind_safe_label(l2: &RpcMock) {
        l2.expect_block_by_number(10, ElHas::Missing.block(10));
    }

    /// The execution layer has another block at the cross-safe height than the delegate's.
    fn cross_safe_off_the_local_safe_chain(l2: &RpcMock) {
        l2.expect_block_by_number(10, ElHas::EngineChain.block(10));
        l2.expect_block_header_by_number(8, ElHas::OtherChain.block(8));
    }

    /// A cross-safe head at or below the safe label leaves it, so a load-balanced delegate whose
    /// cross-safe head lags does not drop the forkchoice update's safe block; so does one that is
    /// not on the chain of its own local-safe head, which the execution layer has.
    #[rstest::rstest]
    #[case::at_safe_label(7, cross_safe_at_safe_label)]
    #[case::behind_safe_label(6, cross_safe_behind_safe_label)]
    #[case::el_syncing_behind_safe_label(6, el_syncing_cross_safe_behind_safe_label)]
    #[case::off_the_local_safe_chain(8, cross_safe_off_the_local_safe_chain)]
    #[tokio::test]
    async fn cross_safe_head_not_advancing_keeps_the_safe_label(
        #[case] cross_safe_number: u64,
        #[case] expect_lookups: fn(&RpcMock),
    ) {
        let local_safe = engine_chain_block(10);
        let mut state = engine_state_with_safe_labels(9, 7);
        let expected = labels_after(&state, local_safe, state.sync_state.safe_head());

        run_delegated(&mut state, local_safe, engine_chain_block(cross_safe_number), |l2| {
            expect_lookups(l2);
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A cross-safe head behind, or conflicting with, finality leaves the safe label without a
    /// lookup, while the local-safe head from the same status still moves.
    #[rstest::rstest]
    #[case::behind_finalized(engine_chain_block(4))]
    #[case::conflicting_with_finalized(other_chain_block(5))]
    #[tokio::test]
    async fn cross_safe_head_breaking_finality_keeps_only_the_safe_label(
        #[case] cross_safe: L2BlockInfo,
    ) {
        let local_safe = engine_chain_block(10);
        let mut state = engine_state_with_safe_labels(9, 7);
        let expected = labels_after(&state, local_safe, state.sync_state.safe_head());

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_by_number(10, ElHas::EngineChain.block(10));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// The execution layer has the lagging local-safe head at 10, and confirms the cross-safe head
    /// at 9.
    fn el_has_local_safe_10_and_cross_safe_9(l2: &RpcMock) {
        l2.expect_block_header_by_number(10, ElHas::EngineChain.block(10));
        l2.expect_block_header_by_number(9, ElHas::EngineChain.block(9));
    }

    /// The execution layer lacks the lagging local-safe head at 10, so the cross-safe head is not
    /// looked up.
    fn el_missing_local_safe_10(l2: &RpcMock) {
        l2.expect_block_header_by_number(10, ElHas::Missing.block(10));
    }

    /// A delegated local-safe head behind the engine's whose block the execution layer has, or
    /// lacks, from a lagging delegate, leaves the local-safe and unsafe heads in place. Its
    /// cross-safe head still moves the safe label when above it.
    #[rstest::rstest]
    #[case::el_has_local_safe(el_has_local_safe_10_and_cross_safe_9)]
    #[case::el_missing_local_safe(el_missing_local_safe_10)]
    #[tokio::test]
    async fn lagging_delegate_moves_only_an_advancing_safe_label(
        #[case] expect_lookups: fn(&RpcMock),
    ) {
        let (local_safe, cross_safe) = (engine_chain_block(10), engine_chain_block(9));
        let mut state = engine_state_with_safe_labels(12, 8);
        let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
            safe_head: Some(cross_safe),
            ..Default::default()
        });

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            expect_lookups(l2);
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A delegate before interop, whose heads are the same block, that lags the engine leaves every
    /// label where it is and sends no forkchoice update, whether or not the execution layer has the
    /// block.
    #[rstest::rstest]
    #[case::el_has_block(10, ElHas::EngineChain)]
    #[case::el_missing_block(10, ElHas::Missing)]
    #[case::at_finalized_head(5, ElHas::EngineChain)]
    #[tokio::test]
    async fn lagging_delegate_before_interop_changes_nothing(
        #[case] delegated_number: u64,
        #[case] el_at_local_safe: ElHas,
    ) {
        let delegated = engine_chain_block(delegated_number);
        let mut state = engine_state_with_safe_labels(12, 12);
        let state_before = state;

        run_delegated(&mut state, delegated, delegated, |l2| {
            l2.expect_block_header_by_number(
                delegated_number,
                el_at_local_safe.block(delegated_number),
            );
        })
        .await
        .unwrap();

        assert_eq!(state, state_before);
    }

    /// A failed lookup of a lagging delegated head changes nothing and is retried on the failure
    /// backoff.
    #[tokio::test]
    async fn failed_lookup_of_a_lagging_delegated_head_is_retried() {
        let delegated = engine_chain_block(10);
        let mut state = engine_state_with_safe_labels(12, 12);
        let state_before = state;

        let err = run_delegated(&mut state, delegated, delegated, |l2| {
            l2.expect_error("eth_getBlockByNumber");
        })
        .await
        .unwrap_err();

        assert!(
            matches!(err, ConsolidateTaskError::FailedToFetchDelegatedSafeBlock(10)),
            "got {err:?}"
        );
        assert_eq!(err.severity(), Temporary);
        assert_eq!(state, state_before);
    }

    /// A lagging delegate whose cross-safe head is behind the safe label leaves every label where
    /// it is and sends no forkchoice update, without looking the cross-safe head up.
    #[tokio::test]
    async fn lagging_delegate_with_cross_safe_head_behind_the_safe_label_changes_nothing() {
        let mut state = engine_state_with_safe_labels(12, 12);
        let state_before = state;

        run_delegated(&mut state, engine_chain_block(10), engine_chain_block(7), |l2| {
            l2.expect_block_header_by_number(10, ElHas::EngineChain.block(10));
        })
        .await
        .unwrap();

        assert_eq!(state, state_before);
    }

    /// A lagging delegate whose cross-safe head conflicts with the finalized head leaves every
    /// label where it is: the cross-safe head cannot become the safe label.
    #[tokio::test]
    async fn lagging_delegate_with_cross_safe_head_conflicting_with_finality_changes_nothing() {
        let mut state = engine_state_with_safe_labels(12, 12);
        let state_before = state;

        run_delegated(&mut state, engine_chain_block(10), other_chain_block(5), |l2| {
            l2.expect_block_header_by_number(10, ElHas::EngineChain.block(10));
        })
        .await
        .unwrap();

        assert_eq!(state, state_before);
    }

    /// A lagging delegate whose local-safe block the execution layer lacks cannot be judged against
    /// the execution layer's canonical blocks, which may predate its sync, so a cross-safe head off
    /// them is not looked up and reorgs nothing; not above the safe label, it changes nothing.
    #[tokio::test]
    async fn lagging_delegate_missing_from_the_el_never_reorgs() {
        let mut state = engine_state_with_safe_labels(12, 12);
        let state_before = state;

        run_delegated(&mut state, other_chain_block(10), other_chain_block(8), |l2| {
            l2.expect_block_header_by_number(10, ElHas::Missing.block(10));
        })
        .await
        .unwrap();

        assert_eq!(state, state_before);
    }

    /// A lagging delegate is not judged against an execution layer that has not applied the
    /// engine's last forkchoice, even after its first sync finished: its canonical block at the
    /// local-safe height may predate the engine's labels, so no lookup is made and a delegate on
    /// another chain reorgs nothing.
    #[tokio::test]
    async fn lagging_delegate_is_not_judged_against_a_syncing_el() {
        let mut state = engine_state_with_safe_labels(12, 12);
        state.el_applied_last_forkchoice = false;
        let state_before = state;

        run_delegated(&mut state, other_chain_block(10), other_chain_block(8), |_| {})
            .await
            .unwrap();

        assert_eq!(state, state_before);
    }

    /// Lagging delegate: the execution layer has another block at its local-safe height.
    fn lagging_local_safe_off_the_el_chain(l2: &RpcMock) {
        l2.expect_block_header_by_number(10, ElHas::EngineChain.block(10));
    }

    /// Below the unsafe head: the execution layer has another block at the local-safe height.
    fn local_safe_off_the_el_chain(l2: &RpcMock) {
        l2.expect_block_by_number(18, ElHas::EngineChain.block(18));
    }

    /// At or above the unsafe head: the local-safe head is another block at the unsafe height.
    const fn local_safe_off_the_unsafe_head(_: &RpcMock) {}

    /// Above the unsafe head but not its child: the local-safe head's ancestry is unknown, and the
    /// execution layer does not hold the engine's safe label either.
    fn local_safe_above_the_unsafe_head(l2: &RpcMock) {
        l2.expect_block_header_by_number(12, ElHas::OtherChain.block(12));
    }

    /// Delegated heads on a chain that diverged from the engine's, which an execution-layer block
    /// or the unsafe head at the local-safe height proves, or above the unsafe head without being
    /// its child while the execution layer does not hold the safe label, reorg the engine onto
    /// them: the unsafe and local-safe heads move to the local-safe head, and the safe label to the
    /// cross-safe head even when lower, since the old safe block is on the abandoned chain.
    /// Finalized stays.
    #[rstest::rstest]
    #[case::lagging_local_safe(10, lagging_local_safe_off_the_el_chain)]
    #[case::local_safe_below_unsafe(18, local_safe_off_the_el_chain)]
    #[case::local_safe_at_unsafe(20, local_safe_off_the_unsafe_head)]
    #[case::local_safe_above_unsafe(25, local_safe_above_the_unsafe_head)]
    #[tokio::test]
    async fn delegated_heads_off_the_engine_chain_reorg_the_engine(
        #[case] local_safe_number: u64,
        #[case] expect_lookups: fn(&RpcMock),
    ) {
        let (local_safe, cross_safe) = (other_chain_block(local_safe_number), other_chain_block(8));
        let mut state = engine_state_with_safe_labels(12, 12);
        let finalized_before = state.sync_state.finalized_head();
        let expected = labels_after_reorg(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            expect_lookups(l2);
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
        assert_eq!(state.sync_state.finalized_head(), finalized_before);
    }

    /// Delegated heads at or above the unsafe head on the engine's chain move the unsafe and
    /// local-safe heads to the local-safe head, and a cross-safe head on that chain above the safe
    /// label moves it.
    #[rstest::rstest]
    #[case::at_unsafe_head(engine_chain_block(20))]
    #[case::child_of_unsafe_head(child_of_engine_unsafe_head())]
    #[tokio::test]
    async fn delegated_heads_extending_the_engine_chain_move_an_advancing_safe_label(
        #[case] local_safe: L2BlockInfo,
    ) {
        let cross_safe = engine_chain_block(17);
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = labels_after_reorg(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_header_by_number(17, ElHas::EngineChain.block(17));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A safe label left on blocks a reorg abandoned does not prove the engine's chain: the
    /// execution layer confirms the cross-safe head at the safe label's height, and the safe label
    /// moves to it, keeping it an ancestor of the unsafe head.
    #[tokio::test]
    async fn cross_safe_head_replaces_a_stale_safe_label_at_its_height() {
        let (local_safe, cross_safe) = (engine_chain_block(20), engine_chain_block(12));
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(engine_chain_block(20))
            .with_local_safe_head(other_chain_block(12))
            .with_safe_head(other_chain_block(12))
            .with_finalized_head(engine_chain_block(5))
            .build();
        let expected = labels_after_reorg(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_header_by_number(12, ElHas::EngineChain.block(12));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// While the execution layer has not finished syncing, its canonical blocks may be from before
    /// the sync, so the cross-safe head is not checked against them and only advances the safe
    /// label.
    #[tokio::test]
    async fn cross_safe_head_is_not_judged_against_a_syncing_el() {
        let (local_safe, cross_safe) = (engine_chain_block(20), engine_chain_block(17));
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(engine_chain_block(20))
            .with_safe_head(engine_chain_block(15))
            .with_finalized_head(engine_chain_block(5))
            .with_el_sync_finished(false)
            .build();
        let expected = labels_after_reorg(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// The engine chain's block at 21, the child of the engine's unsafe head at 20.
    fn child_of_engine_unsafe_head() -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                parent_hash: engine_chain_block(20).block_info.hash,
                ..engine_chain_block(21).block_info
            },
            ..Default::default()
        }
    }

    /// A delegated local-safe head that is the unsafe head's child extends the engine's chain, so
    /// a cross-safe head below the safe label keeps it.
    #[tokio::test]
    async fn child_of_the_unsafe_head_keeps_the_safe_label() {
        let local_safe = child_of_engine_unsafe_head();
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            unsafe_head: Some(local_safe),
            ..Default::default()
        });

        run_delegated(&mut state, local_safe, engine_chain_block(12), |l2| {
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A reorg whose cross-safe head is below the finalized head moves the safe label to the
    /// finalized head, which the delegate's chain shares with the engine's.
    #[tokio::test]
    async fn reorg_with_cross_safe_head_behind_finality_moves_safe_to_finalized() {
        let local_safe = other_chain_block(20);
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = labels_after_reorg(&state, local_safe, state.sync_state.finalized_head());

        run_delegated(&mut state, local_safe, other_chain_block(4), |l2| {
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// Delegated heads at the unsafe head on the engine's chain, as when the engine has no unsafe
    /// gossip, keep the safe label for a cross-safe head at or below it, so a lagging backend
    /// behind a load balancer does not move safe back.
    #[rstest::rstest]
    #[case::at_safe_label(15, no_lookup)]
    #[case::behind_safe_label(12, no_lookup)]
    #[tokio::test]
    async fn delegated_heads_at_the_unsafe_head_keep_the_safe_label(
        #[case] cross_safe_number: u64,
        #[case] expect_cross_safe_lookup: fn(&RpcMock),
    ) {
        let local_safe = engine_chain_block(20);
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            unsafe_head: Some(local_safe),
            ..Default::default()
        });

        run_delegated(&mut state, local_safe, engine_chain_block(cross_safe_number), |l2| {
            expect_cross_safe_lookup(l2);
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A cross-safe head at one of the engine's labels is not looked up.
    const fn no_lookup(_: &RpcMock) {}

    /// The execution layer has the engine chain's block at 15, which the safe label alone cannot
    /// prove, since it can stay on blocks a reorg abandoned.
    fn cross_safe_15_on_the_engine_chain(l2: &RpcMock) {
        l2.expect_block_header_by_number(15, ElHas::EngineChain.block(15));
    }

    fn expect_invalid_forkchoice_state(l2: &RpcMock) {
        l2.expect_error_code(
            "engine_forkchoiceUpdatedV3",
            i64::from(alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR),
        );
    }

    /// A forkchoice update for delegated safe heads, whether a reorg off the engine's chain or an
    /// extension of it, that the execution layer refuses as an invalid forkchoice state resets the
    /// engine, which rebuilds the labels from the execution layer.
    #[rstest::rstest]
    #[case::reorg(other_chain_block(20))]
    #[case::extension(engine_chain_block(25))]
    #[tokio::test]
    async fn rejected_forkchoice_for_delegated_heads_resets_the_engine(
        #[case] delegated: L2BlockInfo,
    ) {
        let mut state = engine_state_with_safe_labels(15, 15);
        let state_before = state;

        let err = run_delegated(&mut state, delegated, delegated, expect_invalid_forkchoice_state)
            .await
            .unwrap_err();

        assert_eq!(err.severity(), EngineTaskErrorSeverity::Reset, "got {err:?}");
        assert_eq!(state, state_before);
    }

    /// A reorg whose cross-safe head breaks finality is skipped before the execution layer is
    /// asked: the cross-safe head cannot become the safe label, and the old safe label would sit
    /// above the new unsafe head.
    #[tokio::test]
    async fn reorg_with_cross_safe_head_breaking_finality_is_skipped() {
        let mut state = engine_state_with_safe_labels(15, 15);
        let state_before = state;

        let err = run_delegated(&mut state, other_chain_block(20), other_chain_block(5), |_| {})
            .await
            .unwrap_err();

        assert!(matches!(err, ConsolidateTaskError::BreaksFinality(_)), "got {err:?}");
        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Skip);
        assert_eq!(state, state_before);
    }

    /// While the execution layer syncs, its canonical blocks are still those from before the sync,
    /// so a cross-safe head is not compared with them: it only advances the safe label, and never
    /// reorgs the engine or retargets the sync.
    #[rstest::rstest]
    #[case::other_chain_below_safe_label(other_chain_block(12), engine_chain_block(15))]
    #[case::advancing(engine_chain_block(16), engine_chain_block(16))]
    #[tokio::test]
    async fn el_sync_moves_the_safe_label_only_forward_without_lookups(
        #[case] cross_safe: L2BlockInfo,
        #[case] expected_safe: L2BlockInfo,
    ) {
        let local_safe = engine_chain_block(18);
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = labels_after(&state, local_safe, expected_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_by_number(18, ElHas::Missing.block(18));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A cross-safe head that is the engine's safe label leaves it in place without asking the
    /// execution layer. One that differs from the safe label at its height is checked against the
    /// execution layer, which has the engine chain's block there, so the safe label stays too.
    #[rstest::rstest]
    #[case::at_safe_label(engine_chain_block(15), no_lookup)]
    #[case::conflicting_with_safe_label(other_chain_block(15), cross_safe_15_on_the_engine_chain)]
    #[tokio::test]
    async fn cross_safe_head_off_the_engine_chain_keeps_the_safe_label(
        #[case] cross_safe: L2BlockInfo,
        #[case] expect_cross_safe_lookup: fn(&RpcMock),
    ) {
        let local_safe = engine_chain_block(18);
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = labels_after(&state, local_safe, engine_chain_block(15));

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_by_number(18, ElHas::EngineChain.block(18));
            expect_cross_safe_lookup(l2);
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A delegated local-safe head whose block the execution layer has, but whose info cannot be
    /// built from it, is consolidated with the delegate's info instead of reorging the unsafe head
    /// back to it.
    #[tokio::test]
    async fn undecodable_delegated_block_is_consolidated_without_a_reorg() {
        let (block, local_safe) = chain_block(18, ENGINE_CHAIN);
        let cfg = Arc::new(RollupConfig::default());
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = labels_after(&state, local_safe, local_safe);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_block_by_number(18, Some(block));
        l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            delegated_safe_heads(local_safe, local_safe),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A delegated block that differs from the execution layer's is not a divergence while the
    /// execution layer has not applied the engine's last forkchoice: only the safe labels move, the
    /// cross-safe head only advancing, and the unsafe head stays the sync target.
    #[tokio::test]
    async fn delegated_block_off_a_syncing_el_moves_only_the_safe_labels() {
        let (local_safe, cross_safe) = (engine_chain_block(18), engine_chain_block(17));
        let mut state = engine_state_with_safe_labels(15, 15);
        state.el_applied_last_forkchoice = false;
        let expected = labels_after(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_by_number(18, ElHas::OtherChain.block(18));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Syncing,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// A delegate catching up several blocks above the unsafe head keeps the engine's safe label
    /// when the execution layer holds it, rather than dropping it to a lower cross-safe head; only
    /// a safe label the execution layer does not hold falls to the cross-safe head.
    #[rstest::rstest]
    #[case::el_holds_safe_label(ElHas::EngineChain, 15)]
    #[case::el_off_safe_label(ElHas::OtherChain, 12)]
    #[tokio::test]
    async fn catch_up_above_the_unsafe_head_keeps_a_safe_label_the_el_holds(
        #[case] el_at_safe_label: ElHas,
        #[case] expected_safe: u64,
    ) {
        let (local_safe, cross_safe) = (engine_chain_block(23), engine_chain_block(12));
        let mut state = engine_state_with_safe_labels(15, 15);
        let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            safe_head: Some(engine_chain_block(expected_safe)),
            unsafe_head: Some(local_safe),
            ..Default::default()
        });

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_block_header_by_number(15, el_at_safe_label.block(15));
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// Labels with the unsafe head rewound below the safe labels are moved back into order by
    /// delegated heads above finality instead of being kept as a lagging delegate's; the execution
    /// layer is not asked about heights above the rewound unsafe head.
    #[tokio::test]
    async fn delegated_heads_repair_an_unsafe_head_below_the_safe_labels() {
        let (local_safe, cross_safe) = (engine_chain_block(12), engine_chain_block(12));
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(engine_chain_block(10))
            .with_safe_head(engine_chain_block(15))
            .with_finalized_head(engine_chain_block(5))
            .build();
        let expected = labels_after_reorg(&state, local_safe, cross_safe);

        run_delegated(&mut state, local_safe, cross_safe, |l2| {
            l2.expect_forkchoice_updated(
                expected.create_forkchoice_state(),
                PayloadStatusEnum::Valid,
            );
        })
        .await
        .unwrap();

        assert_eq!(state.sync_state, expected);
    }

    /// An execution layer that answers `SYNCING` while it reports no sync and sits at the derived
    /// block's parent fetches nothing, as without peers, so the block is built from the derived
    /// attributes on that parent instead of waiting: the build's forkchoice update follows.
    #[tokio::test]
    async fn el_idle_at_the_parent_builds_the_derived_block() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        el_lacks_derived_block_and_reports(&l2, IdleAt(9));
        l2.expect_forkchoice_updated(
            state.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_error("engine_forkchoiceUpdatedV2");

        let err = task.execute(&mut state).await.unwrap_err();

        assert!(matches!(err, ConsolidateTaskError::BuildTaskFailed(_)), "got {err:?}");
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A check whose probe fails records nothing: the wait has not started, so the next check's
    /// reading starts it and its report still goes out.
    #[tokio::test(start_paused = true)]
    async fn failed_probe_does_not_start_the_wait() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let (task, mut state) = task_for_missing_derived_block(client);
        forkchoice_update_fails(&l2, &state);
        task.execute(&mut state).await.unwrap_err();
        let has_started_after_failure = task.el_sync_progress().has_started();

        checking(0, IdleAt(100)).expect_on(&l2, &state);
        task.execute(&mut state).await.unwrap_err();

        assert!(!has_started_after_failure);
        assert!(task.el_sync_progress().has_started());
        l1.assert_finished();
        l2.assert_finished();
    }
}
