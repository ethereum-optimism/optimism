//! A task to consolidate the engine state.

use crate::{
    ConsolidateTaskError, EngineClient, EngineState, ExecutableEngineTask, ImportedBlockSink,
    InsertTaskError, SealTaskError,
    state::EngineSyncStateUpdate,
    task_queue::{build_and_import, synchronize},
};
use alloy_rpc_types_engine::PayloadStatusEnum;
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

    /// Whether this input completes a batch (delegated block info is already safe).
    fn completes_batch(&self, cfg: &RollupConfig) -> bool {
        match self {
            Self::Attributes(attributes) => {
                attributes.is_last_in_span || attributes.is_holocene_active(cfg)
            }
            Self::BlockInfo(_) => true,
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
        let result = build_and_import(
            self.client.as_ref(),
            &self.cfg,
            state,
            attributes.clone(),
            self.block_sink.as_ref(),
        )
        .await
        .map_err(ConsolidateTaskError::from);
        let rejected = match &result {
            Err(ConsolidateTaskError::BuildTaskFailed(err)) => err.is_invalid_attributes(),
            Err(ConsolidateTaskError::SealTaskFailed(SealTaskError::PayloadInsertionFailed(
                err,
            ))) => {
                matches!(
                    **err,
                    InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. })
                )
            }
            _ => false,
        };
        if rejected && !attributes.is_deposits_only() && !attributes.is_holocene_active(&self.cfg) {
            state.need_fcu_call_backup_unsafe_reorg = true;
            return self.reject_batch(state).await;
        }
        result
    }

    /// Restore a consistent forkchoice at the last completed batch. The pipeline keeps its
    /// L1/channel cursor; `BatchQueue` discards the rest of the span when it sees the old parent.
    async fn reject_batch(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        let checkpoint = state.sync_state.local_safe_head();
        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate { unsafe_head: Some(checkpoint), ..Default::default() },
        )
        .await?;
        state.pending_safe_head = None;
        state.need_fcu_call_backup_unsafe_reorg = false;
        warn!(target: "engine", %checkpoint, "Rejected invalid batch and restored completed boundary");
        Err(ConsolidateTaskError::BatchRejected)
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
        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate {
                unsafe_head: Some(*safe_l2),
                safe_head: Some(*safe_l2),
                local_safe_head: Some(*safe_l2),
                ..Default::default()
            },
        )
        .await
        .map_err(|e| {
            warn!(target: "engine", ?e, "Apply safe head failed");
            e
        })?;

        state.pending_safe_head = None;
        info!(
            target: "engine",
            hash = %safe_l2.block_info.hash,
            number = safe_l2.block_info.number,
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
                // Before Holocene, keep the prefix pending and advertise it as safe only once
                // the span completes. Holocene accepts each singular block independently.
                Ok(block_info) if !self.input.completes_batch(&self.cfg) => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    // Only the derivation parent moves until the entire span succeeds.
                    state.pending_safe_head = Some(block_info);

                    info!(
                        target: "engine",
                        hash = %block_info.block_info.hash,
                        number = block_info.block_info.number,
                        "Updated pending-safe head via L1 consolidation"
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
                    )
                    .await
                    .map_err(|e| {
                        warn!(target: "engine", ?e, "Consolidation failed");
                        e
                    })?;
                    state.pending_safe_head = None;

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
        if state.need_fcu_call_backup_unsafe_reorg {
            return self.reject_batch(state).await;
        }
        let safe_head_number = match &self.input {
            ConsolidateInput::Attributes { .. } => state.derivation_parent().block_info.number,
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
        EngineTaskError, ImportedBlockSink, SealTaskError, test_utils::test_engine_client,
    };
    use alloy_eips::BlockNumHash;
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
        let block: RpcBlock<op_alloy_rpc_types::Transaction> =
            RpcBlock::new(header, BlockTransactions::Full(vec![]));

        // Pin genesis to this block so its L2BlockInfo needs no L1-info deposit.
        let cfg = Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, number: 0 },
                ..Default::default()
            },
            ..Default::default()
        });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_params("eth_getBlockByNumber", serde_json::json!(["0x0", true]), block);
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            alloy_rpc_types_engine::ForkchoiceUpdated::new(
                alloy_rpc_types_engine::PayloadStatus::from_status(
                    alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                ),
            ),
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
    #[rstest::rstest]
    #[tokio::test]
    async fn attributes_rejected_at_fcu_are_replaced_with_deposits_only(
        #[values(false, true)] l2_before_activation: bool,
    ) {
        let deposit = alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]);
        let mut attributes =
            derived(vec![deposit.clone(), alloy_primitives::Bytes::from(vec![0x02])]);
        let mut cfg = holocene_cfg();
        if l2_before_activation {
            Arc::make_mut(&mut cfg).hardforks.holocene_time = Some(10);
            attributes.attributes.payload_attributes.timestamp = 2;
            attributes.derived_from =
                Some(kona_protocol::BlockInfo { timestamp: 10, ..Default::default() });
        }
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

    // Before Holocene, rejected attributes roll back and release the next batch.
    #[rstest::rstest]
    #[case::inactive(None)]
    #[case::l2_ahead_of_inclusion(Some(1))]
    #[tokio::test]
    async fn attributes_rejected_at_fcu_before_holocene_are_not_replaced(
        #[case] holocene_time: Option<u64>,
    ) {
        let mut attributes = derived(vec![
            alloy_primitives::Bytes::from(vec![DEPOSIT, 0x01]),
            alloy_primitives::Bytes::from(vec![0x02]),
        ]);
        attributes.derived_from = Some(kona_protocol::BlockInfo::default());
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(0);
        cfg.hardforks.holocene_time = holocene_time;
        let cfg = Arc::new(cfg);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            alloy_rpc_types_engine::ForkchoiceUpdated::new(
                alloy_rpc_types_engine::PayloadStatus::from_status(
                    alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                ),
            ),
        );
        let err = build_derived(Arc::new(client), cfg, &attributes).await;
        l1.assert_finished();
        l2.assert_finished();
        assert!(matches!(err, ConsolidateTaskError::BatchRejected), "got {err:?}");
    }
}

#[cfg(test)]
mod span_tests {
    use super::*;
    use crate::{
        NoopBlockSink,
        test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
    };
    use alloy_consensus::transaction::Recovered;
    use alloy_eips::{BlockNumHash, Encodable2718};
    use alloy_primitives::{Address, Sealed};
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use kona_protocol::L1BlockInfoBedrock;
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    /// Consolidation advances the derivation parent per block, while only a completed span may
    /// move safe labels or be advertised as safe in an FCU. This differs from rebuilding blocks.
    #[rstest::rstest]
    #[case::pre_holocene(false)]
    #[case::holocene_l2_behind_inclusion(true)]
    #[tokio::test]
    async fn consolidated_span_promotion_obeys_the_inclusion_fork(#[case] holocene: bool) {
        let header = alloy_consensus::Header { number: 1, ..Default::default() };
        let mut cfg = RollupConfig::default();
        cfg.genesis.l2 = BlockNumHash::new(1, header.hash_slow());
        cfg.hardforks.holocene_time = Some(10);
        let cfg = Arc::new(cfg);
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let client = Arc::new(client);
        let boundary = L2BlockInfo::default();
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(5))
            .with_safe_head(boundary)
            .with_finalized_head(boundary)
            .build();
        // Keep local-safe at the completed boundary too.
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(boundary),
            ..Default::default()
        });
        let mut attributes = OpAttributesWithParent::new(
            OpPayloadAttributes {
                gas_limit: Some(0),
                transactions: Some(vec![]),
                ..Default::default()
            },
            boundary,
            Some(kona_protocol::BlockInfo {
                timestamp: if holocene { 10 } else { 0 },
                ..Default::default()
            }),
            false,
        );
        // The fixture pins block 1 to genesis, so decoding requires no L1-info deposit.
        let rpc_block = alloy_rpc_types_eth::Block::<Transaction>::new(
            alloy_rpc_types_eth::Header::new(header.clone()),
            alloy_rpc_types_eth::BlockTransactions::Full(vec![]),
        );
        l2.expect("eth_getBlockByNumber", &rpc_block);
        if holocene {
            l2.expect(
                "engine_forkchoiceUpdatedV3",
                ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
            );
        }
        let mut task =
            ConsolidateTask::new(client, cfg, attributes.clone().into(), Arc::new(NoopBlockSink));
        task.consolidate(&mut state).await.unwrap();
        let prefix = state.derivation_parent();
        assert_eq!(prefix.hash(), header.hash_slow());
        let expected_safe = if holocene { prefix } else { boundary };
        assert_eq!(state.sync_state.local_safe_head(), expected_safe);
        assert_eq!(state.sync_state.safe_head(), expected_safe);
        assert_eq!(state.pending_safe_head.is_none(), holocene);
        let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: L1BlockInfoBedrock::default().encode_calldata(),
            ..Default::default()
        }));
        let last_header = alloy_consensus::Header {
            number: 2,
            timestamp: 2,
            parent_hash: prefix.hash(),
            ..Default::default()
        };
        let last_hash = last_header.hash_slow();
        attributes.parent = prefix;
        attributes.attributes.payload_attributes.timestamp = 2;
        attributes.attributes.transactions = Some(vec![deposit.encoded_2718().into()]);
        attributes.is_last_in_span = true;
        task.input = attributes.into();
        l2.expect(
            "eth_getBlockByNumber",
            alloy_rpc_types_eth::Block::new(
                alloy_rpc_types_eth::Header::new(last_header),
                alloy_rpc_types_eth::BlockTransactions::Full(vec![Transaction::from_transaction(
                    Recovered::new_unchecked(deposit, Address::ZERO),
                    Default::default(),
                )]),
            ),
        );
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([
                alloy_rpc_types_engine::ForkchoiceState {
                    safe_block_hash: last_hash,
                    ..state.sync_state.create_forkchoice_state()
                },
                null
            ]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        task.consolidate(&mut state).await.unwrap();
        assert_eq!(state.sync_state.local_safe_head().hash(), last_hash);
        assert_eq!(state.sync_state.safe_head().hash(), last_hash);
        assert_eq!(state.pending_safe_head, None);
        l1.assert_finished();
        l2.assert_finished();
    }
}
