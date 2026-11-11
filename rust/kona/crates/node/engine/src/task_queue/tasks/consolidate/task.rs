//! A task to consolidate the engine state.

use crate::{
    ConsolidateTaskError, EngineClient, EngineState, ExecutableEngineTask, ImportedBlockSink,
    state::EngineSyncStateUpdate,
    task_queue::{ForkchoiceDelivery, build_and_import, synchronize},
};
use alloy_rpc_types_eth::Block;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types::Transaction;
use std::sync::Arc;

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

    /// Derived attributes move the safe head on every consolidation, so an unchanged forkchoice was
    /// already sent. A delegated safe head can repeat a forkchoice the execution layer answered
    /// `SYNCING` or lost, so it is always sent.
    const fn forkchoice_delivery(&self) -> ForkchoiceDelivery {
        match self {
            Self::Attributes(_) => ForkchoiceDelivery::WhenChanged,
            Self::BlockInfo(_) => ForkchoiceDelivery::Always,
        }
    }
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
}

impl ConsolidateTask {
    /// Creates a new [`ConsolidateTask`] with the specified input
    pub const fn new(
        client: Arc<EngineClient>,
        cfg: Arc<RollupConfig>,
        input: ConsolidateInput,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, cfg, input, block_sink }
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
        synchronize(self.client.as_ref(), state, update, ForkchoiceDelivery::Always)
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
                warn!(target: "engine", "Received `None` block for {}", block_num);
                return Err(ConsolidateTaskError::MissingUnsafeL2Block(block_num));
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

                    synchronize(
                        self.client.as_ref(),
                        state,
                        EngineSyncStateUpdate {
                            safe_head: Some(block_info),
                            local_safe_head: Some(block_info),
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
        EngineSyncState, EngineTaskError, FinalityViolation, ImportedBlockSink, NoopBlockSink,
        SealTaskError,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, l1_info_deposit, test_block_info,
            test_engine_client,
        },
    };
    use alloy_consensus::transaction::Recovered;
    use alloy_eips::{BlockNumHash, Encodable2718};
    use alloy_primitives::{Address, B256};
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
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
        let task =
            ConsolidateTask::new(client, cfg, ConsolidateInput::BlockInfo(safe_l2), sink.clone());

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
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        assert_eq!(sink.0.lock().unwrap().len(), usize::from(el_has_block));
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Unlike a delegated safe head, derived attributes whose block is missing below the unsafe
    /// head still reset the engine.
    #[tokio::test]
    async fn derived_attributes_missing_from_el_reset_the_engine() {
        let cfg = Arc::new(RollupConfig::default());
        let attributes = derived_attributes(true);
        let mut state = engine_state_with_safe_head(attributes.parent);
        let state_before = state;
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!(["0xa", true]),
            None::<RpcBlock<Transaction>>,
        );
        let task = ConsolidateTask::new(
            Arc::new(client),
            cfg,
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
        );

        let err = task.execute(&mut state).await.unwrap_err();

        assert!(matches!(err, ConsolidateTaskError::MissingUnsafeL2Block(10)), "got {err:?}");
        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Reset);
        assert_eq!(state, state_before);
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

    /// The unsafe block that `attributes` derive, so consolidation finds it consistent.
    fn unsafe_block_matching(attributes: &OpAttributesWithParent) -> RpcBlock<Transaction> {
        let payload_attributes = &attributes.attributes.payload_attributes;
        let header = alloy_consensus::Header {
            beneficiary: payload_attributes.suggested_fee_recipient,
            gas_limit: attributes.attributes.gas_limit.unwrap(),
            mix_hash: payload_attributes.prev_randao,
            number: attributes.block_number(),
            parent_beacon_block_root: payload_attributes.parent_beacon_block_root,
            parent_hash: attributes.parent.block_info.hash,
            timestamp: payload_attributes.timestamp,
            ..Default::default()
        };
        let deposit = Transaction::from_transaction(
            Recovered::new_unchecked(l1_info_deposit(), Address::ZERO),
            Default::default(),
        );
        RpcBlock::new(RpcHeader::new(header), BlockTransactions::Full(vec![deposit]))
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
        let block = unsafe_block_matching(&attributes);
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
        );

        let err = task.execute(&mut state).await.unwrap_err();

        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Reset, "got {err:?}");
        assert_eq!(state, state_before);
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
        );

        task.execute(&mut state).await.unwrap();

        assert_eq!(state.sync_state, expected_sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }
}
