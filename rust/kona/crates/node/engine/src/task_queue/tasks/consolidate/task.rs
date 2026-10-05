//! A task to consolidate the engine state.

use crate::{
    ConsolidateTaskError, EngineClient, EngineState, EngineTaskExt, ImportedBlockSink,
    SealTaskError, SynchronizeTask,
    state::EngineSyncStateUpdate,
    task_queue::{BuildAndSealError, build_and_seal},
};
use alloy_rpc_types_eth::Block;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types::Transaction;
use std::{sync::Arc, time::Instant};

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

    /// Returns true if this is `Attributes` and `attributes.is_last_in_span` is true.
    const fn is_attributes_last_in_span(&self) -> bool {
        matches!(
            self,
            Self::Attributes(attributes)
                if attributes.is_last_in_span
        )
    }
}

/// The [`ConsolidateTask`] attempts to consolidate the engine state
/// using the specified payload attributes or block info.
#[derive(Debug, Clone)]
pub struct ConsolidateTask<EngineClient_: EngineClient> {
    /// The engine client.
    pub client: Arc<EngineClient_>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The input for consolidation (either attributes or block info).
    pub input: ConsolidateInput,
    /// Where to hand the decoded block once the engine has canonicalized it.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl<EngineClient_: EngineClient> ConsolidateTask<EngineClient_> {
    /// Creates a new [`ConsolidateTask`] with the specified input
    pub const fn new(
        client: Arc<EngineClient_>,
        cfg: Arc<RollupConfig>,
        input: ConsolidateInput,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, cfg, input, block_sink }
    }

    /// This is used when the [`ConsolidateTask`] fails to consolidate the engine state
    async fn execute_build_and_seal_tasks(
        &self,
        state: &mut EngineState,
        attributes: &OpAttributesWithParent,
    ) -> Result<(), ConsolidateTaskError> {
        match build_and_seal(
            state,
            self.client.clone(),
            self.cfg.clone(),
            attributes.clone(),
            true,
            self.block_sink.clone(),
        )
        .await
        {
            Err(BuildAndSealError::Build(err))
                if err.is_invalid_attributes() &&
                    (attributes.is_deposits_only() ||
                        self.cfg.is_holocene_active(
                            attributes.attributes().payload_attributes.timestamp,
                        )) =>
            {
                warn!(target: "engine", %err, "Derived attributes rejected at forkchoiceUpdated");
                if attributes.is_deposits_only() {
                    return Err(SealTaskError::DepositOnlyPayloadFailed.into());
                }
                let result = build_and_seal(
                    state,
                    self.client.clone(),
                    self.cfg.clone(),
                    attributes.as_deposits_only(),
                    true,
                    self.block_sink.clone(),
                )
                .await;
                Err(match result {
                    Ok(()) => SealTaskError::HoloceneInvalidFlush,
                    Err(_) => SealTaskError::DepositOnlyPayloadReattemptFailed,
                }
                .into())
            }
            result => result.map_err(Into::into),
        }
    }

    /// This provides symmetric fallback behavior to with `build_and_seal`.
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

        let fcu_start = Instant::now();

        // We intentionally set the unsafe head to safe_l2 to ensure the engine observes a
        // self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the safe head.
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.cfg.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(*safe_l2),
                safe_head: Some(*safe_l2),
                local_safe_head: Some(*safe_l2),
                ..Default::default()
            },
        )
        .execute(state)
        .await
        .map_err(|e| {
            warn!(target: "engine", ?e, "Apply safe head failed");
            e
        })?;

        let fcu_duration = fcu_start.elapsed();

        info!(
            target: "engine",
            hash = %safe_l2.block_info.hash,
            number = safe_l2.block_info.number,
            fcu_duration = ?fcu_duration,
            "Updated safe head via follow safe"
        );

        Ok(())
    }

    /// Handles the fallback case when the block doesn't match the input or does not exist.
    async fn reconcile_unsafe_to_safe(
        &self,
        state: &mut EngineState,
    ) -> Result<(), ConsolidateTaskError> {
        match &self.input {
            ConsolidateInput::Attributes(attributes) => {
                self.execute_build_and_seal_tasks(state, attributes).await
            }
            ConsolidateInput::BlockInfo(safe_l2) => {
                self.reconcile_to_safe_head(state, safe_l2).await
            }
        }
    }

    /// Attempts consolidation on the engine state.
    pub async fn consolidate(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        let global_start = Instant::now();

        // Fetch the unsafe L2 block
        let block_num = self.input.l2_block_number();
        let fetch_start = Instant::now();
        let block = match self.client.l2_block_by_label(block_num.into()).await {
            Ok(Some(block)) => block,
            Ok(None) => {
                warn!(target: "engine", "Received `None` block for {}", block_num);
                return Err(ConsolidateTaskError::MissingUnsafeL2Block(block_num));
            }
            Err(_) => {
                warn!(target: "engine", "Failed to fetch unsafe l2 block for consolidation");
                return Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block);
            }
        };
        let block_fetch_duration = fetch_start.elapsed();
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
                // Only issue a forkchoice update if the attributes are the last in the span
                // batch. This is an optimization to avoid sending a FCU
                // call for every block in the span batch.
                Ok(block_info) if !self.input.is_attributes_last_in_span() => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    let total_duration = global_start.elapsed();

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
                        ?total_duration,
                        ?block_fetch_duration,
                        "Updated safe head via L1 consolidation"
                    );

                    return Ok(());
                }
                Ok(block_info) => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    let fcu_start = Instant::now();

                    SynchronizeTask::new(
                        Arc::clone(&self.client),
                        self.cfg.clone(),
                        EngineSyncStateUpdate {
                            safe_head: Some(block_info),
                            local_safe_head: Some(block_info),
                            ..Default::default()
                        },
                    )
                    .execute(state)
                    .await
                    .map_err(|e| {
                        warn!(target: "engine", ?e, "Consolidation failed");
                        e
                    })?;

                    let fcu_duration = fcu_start.elapsed();
                    let total_duration = global_start.elapsed();

                    info!(
                        target: "engine",
                        hash = %block_info.block_info.hash,
                        number = block_info.block_info.number,
                        ?total_duration,
                        ?block_fetch_duration,
                        fcu_duration = ?fcu_duration,
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
impl<EngineClient_: EngineClient> EngineTaskExt for ConsolidateTask<EngineClient_> {
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
    use crate::{EngineTaskError, ImportedBlockSink, test_utils::MockEngineClient};
    use alloy_eips::{BlockNumHash, BlockNumberOrTag};
    use alloy_primitives::B256;
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
        let header = RpcHeader::new(alloy_consensus::Header::default());
        let block_hash = header.hash;
        let block = RpcBlock::new(header, BlockTransactions::Full(vec![]));

        // Pin genesis to this block so its L2BlockInfo needs no L1-info deposit.
        let cfg = Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, number: 0 },
                ..Default::default()
            },
            ..Default::default()
        });
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(cfg.clone())
                .with_l2_block_by_label(BlockNumberOrTag::Number(0), block)
                .build(),
        );

        let safe_l2 = L2BlockInfo {
            block_info: BlockInfo { hash: block_hash, number: 0, ..Default::default() },
            ..Default::default()
        };
        let sink = Arc::new(RecordingSink::default());
        let task =
            ConsolidateTask::new(client, cfg, ConsolidateInput::BlockInfo(safe_l2), sink.clone());

        task.consolidate(&mut EngineState::default()).await.unwrap();

        assert_eq!(sink.0.lock().unwrap().as_slice(), &[block_hash]);
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
        client: Arc<MockEngineClient>,
        attributes: &OpAttributesWithParent,
    ) -> ConsolidateTaskError {
        let mut state = crate::test_utils::TestEngineStateBuilder::new()
            .with_unsafe_head(attributes.parent)
            .build();
        let task = ConsolidateTask::new(
            client,
            holocene_cfg(),
            ConsolidateInput::Attributes(Box::new(attributes.clone())),
            Arc::new(crate::NoopBlockSink),
        );
        task.execute_build_and_seal_tasks(&mut state, attributes)
            .await
            .expect_err("the mock never completes a seal")
    }

    // SDM-H1: -38003 triggers deposits-only recovery.
    #[tokio::test]
    async fn attributes_rejected_at_fcu_are_replaced_with_deposits_only() {
        let deposit = alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]);
        let attributes = derived(vec![deposit.clone(), alloy_primitives::Bytes::from(vec![0x02])]);
        let client = Arc::new(
            crate::test_utils::test_engine_client_builder()
                .with_fork_choice_updated_v3_rejecting_non_deposit_attributes()
                .with_fork_choice_updated_v3_response(alloy_rpc_types_engine::ForkchoiceUpdated {
                    payload_status: alloy_rpc_types_engine::PayloadStatus::from_status(
                        alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                    ),
                    payload_id: Some(alloy_rpc_types_engine::PayloadId::new([1; 8])),
                })
                .build(),
        );

        let err = build_derived(client.clone(), &attributes).await;

        let calls = client.fork_choice_updated_v3_attributes().await;
        assert_eq!(calls.len(), 2, "one fcU for the batch, one for its deposits-only replacement");
        assert_eq!(
            calls[1].as_ref().and_then(|attrs| attrs.transactions.clone()),
            Some(vec![deposit]),
            "the replacement keeps only the deposits",
        );
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

    #[tokio::test]
    async fn transport_failure_at_fcu_is_retried_not_replaced() {
        let attributes = derived(vec![
            alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]),
            alloy_primitives::Bytes::from(vec![0x02]),
        ]);
        let client = Arc::new(crate::test_utils::test_engine_client_builder().build());

        let err = build_derived(client.clone(), &attributes).await;

        assert_eq!(client.fork_choice_updated_v3_attributes().await.len(), 1);
        assert!(matches!(err, ConsolidateTaskError::BuildTaskFailed(_)), "got {err:?}");
        assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Temporary);
    }
}
