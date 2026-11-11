//! A task to consolidate the engine state.

use super::el_sync_progress::{ElSyncProgress, ElSyncReading, ElSyncWaitOutcome, ElSyncWaitStep};
use crate::{
    ConsolidateTaskError, EngineClient, EngineState, ExecutableEngineTask, ImportedBlockSink,
    state::EngineSyncStateUpdate,
    task_queue::{
        ForkchoiceReply, build_and_import, record_forkchoice_reply, send_forkchoice, synchronize,
        synchronize_unconditionally,
    },
};
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

/// Input for consolidation - either derived attributes or safe L2 block
#[derive(Debug, Clone)]
pub enum ConsolidateInput {
    /// Consolidate based on derived attributes.
    Attributes(Box<OpAttributesWithParent>),
    /// Derivation Delegation: consolidate based on safe L2 block info.
    BlockInfo(L2BlockInfo),
}

impl From<L2BlockInfo> for ConsolidateInput {
    fn from(v: L2BlockInfo) -> Self {
        Self::BlockInfo(v)
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
            Self::BlockInfo(info) => info.block_info.number,
        }
    }

    /// Checks if the block is consistent with this consolidation input.
    fn is_consistent_with_block(&self, cfg: &RollupConfig, block: &Block<Transaction>) -> bool {
        match self {
            Self::Attributes(attributes) => {
                crate::AttributesMatch::check(cfg, attributes, block).is_match()
            }
            Self::BlockInfo(info) => block.header.hash == info.block_info.hash,
        }
    }

    /// Whether these are derived attributes for a block before the last block of its span batch.
    const fn is_intermediate_span_block(&self) -> bool {
        matches!(self, Self::Attributes(attributes) if !attributes.is_last_in_span)
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
/// using the specified payload attributes or block info.
#[derive(Debug, Clone)]
pub struct ConsolidateTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The input for consolidation (either attributes or block info).
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

    /// This provides symmetric fallback behavior with `build_and_import`.
    async fn reconcile_to_safe_head(
        &self,
        state: &mut EngineState,
        safe_l2: &L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        warn!(
            target: "engine",
            safe_l2 = %safe_l2,
            "Apply safe head"
        );

        // We intentionally set the unsafe head to safe_l2 to ensure the engine observes a
        // self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the safe head.
        self.apply_delegated_safe_head(
            state,
            safe_l2,
            EngineSyncStateUpdate {
                local_safe_head: Some(*safe_l2),
                safe_head: Some(*safe_l2),
                unsafe_head: Some(*safe_l2),
                ..Default::default()
            },
            "Updated safe head via follow safe",
        )
        .await
    }

    /// Moves the safe labels to a delegated safe head that the execution layer does not have yet,
    /// leaving the unsafe head in place.
    ///
    /// A block missing below the unsafe head means the execution layer is still syncing toward the
    /// unsafe head. Resetting would restart that sync from the execution layer's old labels, and
    /// moving the unsafe head back would retarget it. Like op-node's follow-source mode, only the
    /// safe labels follow the delegate, and the forkchoice update carries them to the execution
    /// layer.
    async fn follow_safe_head_during_el_sync(
        &self,
        state: &mut EngineState,
        safe_l2: &L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        warn!(
            target: "engine",
            safe_l2 = %safe_l2,
            unsafe_number = state.sync_state.unsafe_head().block_info.number,
            "Execution layer does not have the delegated safe head yet; keeping its sync target"
        );

        self.apply_delegated_safe_head(
            state,
            safe_l2,
            EngineSyncStateUpdate {
                local_safe_head: Some(*safe_l2),
                safe_head: Some(*safe_l2),
                ..Default::default()
            },
            "Updated safe head while the execution layer syncs",
        )
        .await
    }

    /// Applies `update`, whose labels follow the delegated `safe_l2`, and always sends its
    /// forkchoice update: the execution layer may have answered `SYNCING` to the same labels
    /// before, or lost them in a restart.
    async fn apply_delegated_safe_head(
        &self,
        state: &mut EngineState,
        safe_l2: &L2BlockInfo,
        update: EngineSyncStateUpdate,
        outcome: &'static str,
    ) -> Result<(), ConsolidateTaskError> {
        synchronize_unconditionally(self.client.as_ref(), state, update)
            .await
            .map_err(|e| {
                warn!(target: "engine", ?e, safe_l2 = %safe_l2, "Applying the delegated safe head failed");
                e
            })?;

        info!(
            target: "engine",
            hash = %safe_l2.block_info.hash,
            number = safe_l2.block_info.number,
            "{outcome}"
        );

        Ok(())
    }

    /// Follows a delegated safe head below the safe head. Only a different block at its height in
    /// a synced execution layer proves the delegate diverged and reorgs the engine onto it; the
    /// same block (a lagging delegate) or a missing one leaves the labels. After a delegate L1
    /// reorg whose fork point is above the head it reports, the labels stay on the abandoned
    /// blocks until it reports a head above that point.
    async fn follow_delegated_safe_head_behind_safe_head(
        &self,
        state: &mut EngineState,
        safe_l2: &L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        // Until the execution layer applies the engine's forkchoice, its canonical block at that
        // height may predate the engine's own labels, so it proves nothing about the delegate.
        if !state.el_applied_last_forkchoice {
            debug!(
                target: "engine",
                safe_l2 = %safe_l2,
                "Ignoring a delegated safe head behind the safe head while the execution layer syncs"
            );
            return Ok(());
        }
        debug!(
            target: "engine",
            safe_l2 = %safe_l2,
            "Looking up the execution layer's block at a delegated safe head behind the safe head"
        );
        let el_block =
            self.client.get_l2_block(safe_l2.block_info.number.into()).await.map_err(|err| {
                warn!(
                    target: "engine",
                    ?err,
                    safe_l2 = %safe_l2,
                    "Failed to look up the execution layer's block at a delegated safe head"
                );
                ConsolidateTaskError::FailedToFetchDelegatedSafeBlock(safe_l2.block_info.number)
            })?;
        debug!(
            target: "engine",
            safe_l2 = %safe_l2,
            el_hash = ?el_block.as_ref().map(|block| block.header.hash),
            "Looked up the execution layer's block at a delegated safe head"
        );
        match el_block {
            Some(el_block) if el_block.header.hash != safe_l2.block_info.hash => {
                debug!(
                    target: "engine",
                    safe_l2 = %safe_l2,
                    el_hash = %el_block.header.hash,
                    "Delegated safe head behind the safe head diverges from the execution layer"
                );
                self.reconcile_to_safe_head(state, safe_l2).await
            }
            _ => {
                debug!(
                    target: "engine",
                    safe_l2 = %safe_l2,
                    safe_number = state.sync_state.safe_head().block_info.number,
                    "Ignoring a delegated safe head behind the safe head"
                );
                Ok(())
            }
        }
    }

    /// Handles the fallback case when the block doesn't match the input or does not exist.
    async fn reconcile_unsafe_to_safe(
        &self,
        state: &mut EngineState,
    ) -> Result<(), ConsolidateTaskError> {
        match &self.input {
            ConsolidateInput::Attributes(attributes) => {
                self.build_and_import_attributes(state, attributes).await
            }
            ConsolidateInput::BlockInfo(safe_l2) => {
                self.reconcile_to_safe_head(state, safe_l2).await
            }
        }
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
                if let ConsolidateInput::BlockInfo(safe_l2) = &self.input {
                    return self.follow_safe_head_during_el_sync(state, safe_l2).await;
                }
                match self.derived_block_once_el_syncs(state, block_num).await? {
                    DerivedBlock::Committed(block) => *block,
                    DerivedBlock::BuildFromAttributes => {
                        return self.reconcile_unsafe_to_safe(state).await;
                    }
                }
            }
            Err(_) => {
                return Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block);
            }
        };
        let block_hash = block.header.hash;

        if self.input.is_consistent_with_block(&self.cfg, &block) {
            trace!(
                target: "engine",
                input = ?self.input,
                block_hash = %block_hash,
                "Consolidating engine state",
            );
            let consensus_block =
                block.into_consensus().map_transactions(|t| t.inner.inner.into_inner());
            match L2BlockInfo::from_block_and_genesis(&consensus_block, &self.cfg.genesis) {
                // A span batch's blocks before its last one only move the safe label; the last
                // block's forkchoice update delivers them all. This avoids sending a FCU call for
                // every block in the span batch.
                Ok(block_info) if self.input.is_intermediate_span_block() => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    // Apply a transient update to the safe head.
                    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
                        safe_head: Some(block_info),
                        local_safe_head: Some(block_info),
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
                Ok(block_info) => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    let update = EngineSyncStateUpdate {
                        safe_head: Some(block_info),
                        local_safe_head: Some(block_info),
                        ..Default::default()
                    };
                    let synchronized = match &self.input {
                        // Derived attributes move the safe head on every consolidation, so an
                        // unchanged forkchoice was already sent.
                        ConsolidateInput::Attributes(_) => {
                            synchronize(self.client.as_ref(), state, update).await
                        }
                        // A delegated safe head can repeat a forkchoice the execution layer
                        // answered `SYNCING` or lost, so it is always sent.
                        ConsolidateInput::BlockInfo(_) => {
                            synchronize_unconditionally(self.client.as_ref(), state, update).await
                        }
                    };
                    synchronized.map_err(|e| {
                        warn!(target: "engine", ?e, "Consolidation failed");
                        e
                    })?;

                    info!(
                        target: "engine",
                        hash = %block_info.block_info.hash,
                        number = block_info.block_info.number,
                        "Updated safe head via L1 consolidation"
                    );

                    return Ok(());
                }
                Err(e) => {
                    // Continue on to build the block since we failed to construct the block info.
                    warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, proceeding to build task");
                }
            }
        }

        // A different block from an execution layer that has not applied the engine's forkchoice
        // may be from before its sync, so it does not prove the delegate diverged.
        if let ConsolidateInput::BlockInfo(safe_l2) = &self.input &&
            !state.el_applied_last_forkchoice
        {
            return self.follow_safe_head_during_el_sync(state, safe_l2).await;
        }

        debug!(
            target: "engine",
            input = ?self.input,
            block_hash = %block_hash,
            "ConsolidateInput mismatch! Initiating reorg",
        );
        // Handle mismatch case - called when consistency check fails
        // or when L2BlockInfo construction fails in Attributes branch
        self.reconcile_unsafe_to_safe(state).await
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
    // - `BlockInfo`: The safe head is injected externally by the DerivationActor while delegating
    //   derivation, and is not coordinated with the EngineActor's safe/unsafe heads. If the
    //   injected safe head is ahead of the EngineActor's unsafe head, we reconcile the unsafe chain
    //   up to the safe head instead of consolidating.
    async fn execute(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        if let ConsolidateInput::BlockInfo(safe_l2) = &self.input {
            state.sync_state.ensure_respects_finality(safe_l2.block_info.id())?;
            // Labels with the unsafe head below the safe head take the paths below, which move the
            // labels back to a consistent state; ignoring a lagging delegate would keep them broken
            // until a finalization above the unsafe head fails.
            let are_labels_consistent = state.sync_state.unsafe_head().block_info.number >=
                state.sync_state.safe_head().block_info.number;
            if are_labels_consistent &&
                safe_l2.block_info.number < state.sync_state.safe_head().block_info.number
            {
                return self.follow_delegated_safe_head_behind_safe_head(state, safe_l2).await;
            }
        }
        let safe_head_number = match &self.input {
            ConsolidateInput::Attributes { .. } => state.sync_state.safe_head().block_info.number,
            ConsolidateInput::BlockInfo(safe_block_info) => safe_block_info.block_info.number,
        };
        if safe_head_number < state.sync_state.unsafe_head().block_info.number {
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
            RpcMock, TestAttributesBuilder, TestEngineStateBuilder, l1_info_deposit,
            test_block_info, test_engine_client, test_rpc_block_from_attributes,
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
            ConsolidateInput::BlockInfo(safe_l2),
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

    /// Either `target` itself or an older block, for a label that may already sit at the target.
    fn target_or_older_block(is_at_target: bool, target: L2BlockInfo) -> L2BlockInfo {
        if is_at_target { target } else { test_block_info(5) }
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
        let mut state = engine_state_with_safe_head(target_or_older_block(
            starts_at_target,
            delegated_safe_head,
        ));
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
            ConsolidateInput::BlockInfo(delegated_safe_head),
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
        state.el_sync_finished = false;
        let sync_state_before = state.sync_state;
        el_valid_and_recheck_fails(&l2, &state);

        let err = task.execute(&mut state).await.unwrap_err();

        assert_eq!(err.severity(), Temporary, "got {err:?}");
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

    /// Where a delegated safe head sits against the finalized head.
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

    /// The outcome of checking a delegated safe head against finality.
    #[derive(Clone, Copy, Debug, PartialEq, Eq)]
    enum FinalityCheck {
        Accepted,
        BehindFinalizedHead,
        ConflictsWithFinalizedHead,
    }

    impl FinalityCheck {
        fn of(result: &Result<(), ConsolidateTaskError>) -> Self {
            match result {
                Ok(()) => Self::Accepted,
                Err(ConsolidateTaskError::BreaksFinality(
                    FinalityViolation::BehindFinalizedHead { .. },
                )) => Self::BehindFinalizedHead,
                Err(ConsolidateTaskError::BreaksFinality(
                    FinalityViolation::ConflictsWithFinalizedHead { .. },
                )) => Self::ConflictsWithFinalizedHead,
                Err(err) => panic!("unexpected error {err:?}"),
            }
        }
    }

    /// A delegated safe head that breaks finality, as a stale delegate snapshot reports, is skipped
    /// before the execution layer is asked anything, and leaves every label where it was.
    #[rstest::rstest]
    #[case::below(TargetAgainstFinalized::Below, FinalityCheck::BehindFinalizedHead)]
    #[case::conflicting(
        TargetAgainstFinalized::ConflictingAt,
        FinalityCheck::ConflictsWithFinalizedHead
    )]
    #[tokio::test]
    async fn delegated_safe_head_breaking_finality_is_skipped(
        #[case] target: TargetAgainstFinalized,
        #[case] expected: FinalityCheck,
    ) {
        let cfg = Arc::new(RollupConfig::default());
        let mut state = engine_state_with_safe_head(test_block_info(12));
        let state_before = state;
        let delegated_safe_head = target.delegated_safe_head(state.sync_state.finalized_head());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        let result = task.execute(&mut state).await;

        assert_eq!(FinalityCheck::of(&result), expected);
        assert_eq!(result.unwrap_err().severity(), crate::EngineTaskErrorSeverity::Skip);
        assert_eq!(state, state_before);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A delegated safe head from a lagging delegate, between the finalized head, inclusive, and
    /// the safe head, does not move the safe labels back, whether or not the execution layer has
    /// its block, and sends no forkchoice update.
    #[rstest::rstest]
    #[case::el_has_block(true, false)]
    #[case::el_missing_block(false, false)]
    #[case::at_finalized_head(true, true)]
    #[tokio::test]
    async fn lagging_delegated_safe_head_leaves_labels_unchanged(
        #[case] el_has_block: bool,
        #[case] finalized_at_target: bool,
    ) {
        let (block, cfg) = genesis_pinned_block(10);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_safe_head(test_block_info(12))
            .with_finalized_head(target_or_older_block(finalized_at_target, delegated_safe_head))
            .build();
        let state_before = state;
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_block_header_by_number(10, el_block(el_has_block, block));
        let sink = Arc::new(RecordingSink::default());
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            sink.clone(),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state, state_before);
        assert!(sink.0.lock().unwrap().is_empty());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A delegated safe head behind the safe head whose block differs from the execution layer's
    /// means the delegate's chain diverged above it, so the engine still reorgs onto it, without
    /// moving finalized.
    #[tokio::test]
    async fn diverged_delegated_safe_head_behind_safe_head_reorgs_the_engine() {
        let (block, cfg) = genesis_pinned_block(10);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = engine_state_with_safe_head(test_block_info(12));
        let finalized_before = state.sync_state.finalized_head();
        let expected_sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(delegated_safe_head),
            safe_head: Some(delegated_safe_head),
            unsafe_head: Some(delegated_safe_head),
            ..Default::default()
        });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_block_header_by_number(10, Some(other_chain_block(10)));
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        assert_eq!(state.sync_state.finalized_head(), finalized_before);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A delegated safe head behind the safe head is not judged against an execution layer that has
    /// not applied the engine's last forkchoice, whether it is in its first sync or syncing again
    /// after answering `SYNCING`: its canonical block there may predate the engine's labels, so no
    /// lookup is made and the labels stay.
    #[rstest::rstest]
    #[case::initial_sync(false, false)]
    #[case::syncing_again(true, false)]
    #[tokio::test]
    async fn delegated_safe_head_behind_safe_head_is_ignored_while_the_el_syncs(
        #[case] el_sync_finished: bool,
        #[case] el_applied_last_forkchoice: bool,
    ) {
        let (block, cfg) = genesis_pinned_block(10);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_safe_head(test_block_info(12))
            .with_finalized_head(test_block_info(5))
            .with_el_sync_finished(el_sync_finished)
            .with_el_applied_last_forkchoice(el_applied_last_forkchoice)
            .build();
        let state_before = state;
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state, state_before);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An empty block at `number` whose hash differs from [`genesis_pinned_block`]'s at the same
    /// height, standing for a block on another chain.
    fn other_chain_block(number: u64) -> RpcBlock<Transaction> {
        RpcBlock::new(
            RpcHeader::new(alloy_consensus::Header { number, timestamp: 1, ..Default::default() }),
            BlockTransactions::Full(vec![]),
        )
    }

    /// A reorg onto a diverged delegated safe head that the execution layer refuses as an invalid
    /// forkchoice state resets the engine, which rebuilds the labels from the execution layer.
    #[tokio::test]
    async fn rejected_reorg_onto_a_delegated_safe_head_resets_the_engine() {
        let (block, cfg) = genesis_pinned_block(10);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = engine_state_with_safe_head(test_block_info(12));
        let state_before = state;
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_block_header_by_number(10, Some(other_chain_block(10)));
        l2.expect_error_code(
            "engine_forkchoiceUpdatedV3",
            i64::from(alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR),
        );
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        let err = task.execute(&mut state).await.unwrap_err();

        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Reset, "got {err:?}");
        assert_eq!(state, state_before);
        l1.assert_finished();
        l2.assert_finished();
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

    /// Labels with the unsafe head rewound below the safe head are moved back into order by any
    /// delegated safe head above finality instead of being kept: at or above the unsafe head the
    /// engine reorgs onto it, and below it the matching block consolidates. Keeping them would let
    /// a later finalization above the unsafe head fail fatally.
    #[rstest::rstest]
    #[case::at_or_above_unsafe_head(12, false)]
    #[case::below_unsafe_head(8, true)]
    #[tokio::test]
    async fn delegated_safe_head_repairs_an_unsafe_head_below_the_safe_head(
        #[case] delegated_number: u64,
        #[case] is_below_unsafe_head: bool,
    ) {
        let (block, cfg) = genesis_pinned_block(delegated_number);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(10))
            .with_safe_head(test_block_info(15))
            .with_finalized_head(test_block_info(5))
            .build();
        let expected_sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(delegated_safe_head),
            safe_head: Some(delegated_safe_head),
            unsafe_head: (!is_below_unsafe_head).then_some(delegated_safe_head),
            ..Default::default()
        });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        if is_below_unsafe_head {
            l2.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([format!("{delegated_number:#x}"), true]),
                block,
            );
        }
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        assert!(
            state.sync_state.unsafe_head().block_info.number >=
                state.sync_state.safe_head().block_info.number
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A delegated safe head whose block differs from the execution layer's is not treated as a
    /// divergence while the execution layer has not applied the engine's last forkchoice: only the
    /// safe labels move to it, and the unsafe head stays the sync target.
    #[tokio::test]
    async fn mismatched_delegated_safe_head_does_not_reorg_while_the_el_syncs() {
        let (block, cfg) = genesis_pinned_block(15);
        let delegated_safe_head = block_info_of(&cfg, &block);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_safe_head(test_block_info(12))
            .with_finalized_head(test_block_info(5))
            .with_el_applied_last_forkchoice(false)
            .build();
        let expected_sync_state = with_safe_head(&state, delegated_safe_head);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!(["0xf", true]),
            other_chain_block(15),
        );
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::BlockInfo(delegated_safe_head),
            Arc::new(NoopBlockSink),
            STALL,
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
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
