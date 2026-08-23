//! A task to consolidate the engine state.

use crate::{
    ConsolidateTaskError, EngineClient, EngineState, EngineTaskExt, ImportedBlockSink,
    SharedDenyList, SynchronizeTask,
    state::{EngineSyncStateUpdate, LocalSafeHead, LocalSafeOrigin},
    task_queue::build_and_seal,
};
use alloy_rpc_types_eth::Block;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types::Transaction;
use std::{sync::Arc, time::Instant};

/// Input for consolidation - either derived attributes or a local-safe L2 block
#[derive(Debug, Clone)]
pub enum ConsolidateInput {
    /// Consolidate based on derived attributes.
    Attributes(Box<OpAttributesWithParent>),
    /// Derivation Delegation: consolidate based on local-safe L2 block info.
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

/// Whether an RPC failure from the unsafe-block fetch heuristically says the block is absent
/// rather than the fetch having failed.
///
/// A correct execution-layer implementation answers a block-by-number request for a block it does
/// not have with an empty result, never an error, so this only hardens against wrong
/// implementations — the mirror of op-node's `MaybeAsNotFoundErr`
/// (`op-service/eth/errors.go:10-27`), which wraps every payload fetch
/// (`op-service/sources/eth_client.go:269-274`), string set included.
fn is_block_not_found_rpc_error(err: &crate::EngineClientError) -> bool {
    let msg = err.to_string().to_lowercase();
    msg.contains("block not found") ||
        msg.contains("header not found") ||
        msg.contains("unknown block")
}

impl ConsolidateInput {
    /// The L1 origin to pair with a local-safe head written from this input.
    ///
    /// The delegation path carries a bare [`L2BlockInfo`] injected by the delegating derivation
    /// actor, with no attributes and so no L1 origin to pair with — explicitly unpaired rather than
    /// inheriting whatever origin the previous head had.
    pub(super) const fn local_safe_origin(&self) -> LocalSafeOrigin {
        match self {
            Self::Attributes(attributes) => match attributes.derived_from {
                Some(l1) => LocalSafeOrigin::DerivedFrom(l1),
                None => LocalSafeOrigin::Unpaired,
            },
            Self::BlockInfo(_) => LocalSafeOrigin::Unpaired,
        }
    }

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
    /// The super-authority deny list, when the node runs under one.
    ///
    /// Consulted before an existing block is adopted as local-safe: a denied block must be reorged
    /// out and rebuilt, never consolidated back in (op-node's
    /// `op-node/rollup/attributes/attributes.go:241-256`).
    pub deny: Option<SharedDenyList>,
    /// Where to hand the decoded block once the engine has canonicalized it.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl<EngineClient_: EngineClient> ConsolidateTask<EngineClient_> {
    /// Creates a new [`ConsolidateTask`] with the specified input
    pub const fn new(
        client: Arc<EngineClient_>,
        cfg: Arc<RollupConfig>,
        input: ConsolidateInput,
        deny: Option<SharedDenyList>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, cfg, input, deny, block_sink }
    }

    /// This is used when the [`ConsolidateTask`] fails to consolidate the engine state
    async fn execute_build_and_seal_tasks(
        &self,
        state: &mut EngineState,
        attributes: &OpAttributesWithParent,
    ) -> Result<(), ConsolidateTaskError> {
        build_and_seal(
            state,
            self.client.clone(),
            self.cfg.clone(),
            attributes.clone(),
            true,
            self.deny.clone(),
            self.block_sink.clone(),
        )
        .await?;

        Ok(())
    }

    /// This provides symmetric fallback behavior to with `build_and_seal`.
    async fn reconcile_to_local_safe_head(
        &self,
        state: &mut EngineState,
        local_safe_l2: &L2BlockInfo,
    ) -> Result<(), ConsolidateTaskError> {
        warn!(
            target: "engine",
            local_safe_l2 = %local_safe_l2,
            "Apply local-safe head"
        );

        let fcu_start = Instant::now();

        // We intentionally set the unsafe head to local_safe_l2 to ensure the engine observes a
        // self-consistent head state. This is required to correctly handle reorgs (where unsafe
        // may be ahead on a non-canonical fork) and to trigger EL sync when the local unsafe head
        // lags behind the local-safe head.
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.cfg.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(*local_safe_l2),
                // Reached only from the `BlockInfo` arm, where derivation is delegated and no L1
                // origin accompanies the injected head.
                local_safe_head: Some(LocalSafeHead::unpaired(*local_safe_l2)),
                ..Default::default()
            },
        )
        .execute(state)
        .await
        .map_err(|e| {
            warn!(target: "engine", ?e, "Apply local-safe head failed");
            e
        })?;

        let fcu_duration = fcu_start.elapsed();

        info!(
            target: "engine",
            hash = %local_safe_l2.block_info.hash,
            number = local_safe_l2.block_info.number,
            fcu_duration = ?fcu_duration,
            "Updated local-safe head via follow safe"
        );

        Ok(())
    }

    /// Handles the fallback case when the block doesn't match the input or does not exist.
    async fn reconcile_unsafe_to_local_safe(
        &self,
        state: &mut EngineState,
    ) -> Result<(), ConsolidateTaskError> {
        match &self.input {
            ConsolidateInput::Attributes(attributes) => {
                self.execute_build_and_seal_tasks(state, attributes).await
            }
            ConsolidateInput::BlockInfo(local_safe_l2) => {
                self.reconcile_to_local_safe_head(state, local_safe_l2).await
            }
        }
    }

    /// Maps a fetch miss of the unsafe block to consolidate against onto op-node's split
    /// (`op-node/rollup/attributes/attributes.go:204-221`): while the engine's initial EL sync is
    /// in flight the miss is a paced stall — the EL simply has not filled the height in yet, and a
    /// reset would re-target it away from its sync target — and after EL sync it is a reset, so
    /// the walkback realigns the unsafe head with what the execution layer actually has instead of
    /// retrying a fetch that can never succeed.
    fn missing_unsafe_block(&self, state: &EngineState, block_num: u64) -> ConsolidateTaskError {
        if state.el_sync_finished {
            warn!(
                target: "engine",
                block_num,
                "Unsafe L2 block missing for consolidation; requesting a reset to realign with \
                 the execution layer"
            );
            ConsolidateTaskError::MissingUnsafeL2Block(block_num)
        } else {
            debug!(
                target: "engine",
                block_num,
                "Waiting for EL sync to fill in the unsafe L2 block for consolidation"
            );
            ConsolidateTaskError::AwaitingELSyncUnsafeL2Block(block_num)
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
            Ok(None) => return Err(self.missing_unsafe_block(state, block_num)),
            // An error response that says the block is absent is the same miss, only surfaced by
            // a non-conforming EL implementation; op-node folds it into the not-found branch too
            // (`op-service/eth/errors.go:10-27`).
            Err(err) if is_block_not_found_rpc_error(&err) => {
                return Err(self.missing_unsafe_block(state, block_num));
            }
            Err(_) => {
                warn!(target: "engine", "Failed to fetch unsafe l2 block for consolidation");
                return Err(ConsolidateTaskError::FailedToFetchUnsafeL2Block);
            }
        };
        let block_fetch_duration = fetch_start.elapsed();
        let block_hash = block.header.hash;

        if self.input.is_consistent_with_block(&self.cfg, &block) {
            // A denied block can re-enter the canonical chain via unsafe sync, and consolidation
            // would then adopt it as local-safe without ever rebuilding it. Gate here too,
            // forcing the build path on a deny — where the seal-time check turns the rebuild
            // into the deposits-only replacement. The mirror of op-node's consolidation deny
            // check (`op-node/rollup/attributes/attributes.go:241-256`), including its posture:
            // a read error fails CLOSED — without a deny-list answer the block can be neither
            // promoted nor reorged, so the task stalls and retries. Only the attributes path
            // checks, as op-node's does: the delegation path carries no attributes a replacement
            // could be built from.
            let denied = if matches!(&self.input, ConsolidateInput::Attributes(_)) &&
                let Some(deny) = &self.deny
            {
                deny.is_denied(block.header.number, block_hash).map_err(|err| {
                    warn!(
                        target: "engine",
                        %err,
                        block_number = block.header.number,
                        block_hash = %block_hash,
                        "Failed to check the deny list during consolidation; stalling rather \
                         than promoting or reorging"
                    );
                    ConsolidateTaskError::DenyListUnavailable
                })?
            } else {
                false
            };
            if denied {
                warn!(
                    target: "engine",
                    block_number = block.header.number,
                    block_hash = %block_hash,
                    "Consolidated block is denied by the super authority; forcing a reorg to \
                     build the replacement"
                );
                return self.reconcile_unsafe_to_local_safe(state).await;
            }

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

                    // Apply a transient update to the local-safe head.
                    state.sync_state = state.apply_sync_update(EngineSyncStateUpdate {
                        local_safe_head: Some(LocalSafeHead::new(
                            block_info,
                            self.input.local_safe_origin(),
                        )),
                        ..Default::default()
                    });

                    info!(
                        target: "engine",
                        hash = %block_info.block_info.hash,
                        number = block_info.block_info.number,
                        ?total_duration,
                        ?block_fetch_duration,
                        "Updated local-safe head via L1 consolidation"
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
                            local_safe_head: Some(LocalSafeHead::new(
                                block_info,
                                self.input.local_safe_origin(),
                            )),
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
                        "Updated local-safe head via L1 consolidation"
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
        self.reconcile_unsafe_to_local_safe(state).await
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for ConsolidateTask<EngineClient_> {
    type Output = ();

    type Error = ConsolidateTaskError;

    // Behavior depends on how the local-safe head is provided:
    //
    // - `Attributes`: The local-safe head is advanced through the normal derivation flow, where the
    //   DerivationActor and ChainController coordinate both local-safe and unsafe heads. In this
    //   case, we consolidate as long as the unsafe head has not fallen behind.
    //
    // - `BlockInfo`: The local-safe head is injected externally by the DerivationActor while
    //   delegating derivation, and is not coordinated with the ChainController's local-safe/unsafe
    //   heads. If the injected head is ahead of the ChainController's unsafe head, we reconcile the
    //   unsafe chain up to it instead of consolidating.
    async fn execute(&self, state: &mut EngineState) -> Result<(), ConsolidateTaskError> {
        // Attributes that no longer sit on the local-safe head are stale and are dropped, the
        // mirror of op-node's queued-attributes check against the pending-safe head
        // (`op-node/rollup/attributes/attributes.go:156-182`). The drop is not an error case:
        // op-node's comment reads "This is expected after successful processing of these
        // attributes", and that is exactly how the deposits-only fallback ends. Both the Holocene
        // invalid-payload retry and an invalidation's replacement import a block *for* these
        // attributes and advance local-safe past their parent, while the consolidate task itself
        // returns the flush signal as an error — which [`crate::Engine::drain`] answers by keeping
        // the task queued for retry. Without this drop, the retried task re-consolidates the same
        // attributes against the replacement it just imported, mismatches, deterministically
        // rebuilds the denied block, and replaces it again, forever — post-replacement
        // consolidation livelocks instead of converging.
        //
        // A parent at the local-safe *height* whose hash is not the local-safe head is the other
        // arm of op-node's check (`attributes.go:172-182`): reorg inconsistency, answered with a
        // reset rather than a drop.
        if let ConsolidateInput::Attributes(attributes) = &self.input {
            let local_safe = state.sync_state.local_safe_head().block_info;
            let parent = attributes.parent.block_info;
            if parent.number != local_safe.number {
                debug!(
                    target: "engine",
                    parent_number = parent.number,
                    parent_hash = %parent.hash,
                    local_safe_number = local_safe.number,
                    local_safe_hash = %local_safe.hash,
                    "Dropping stale consolidation attributes; the local-safe head moved past them"
                );
                return Ok(());
            }
            if parent.hash != local_safe.hash {
                warn!(
                    target: "engine",
                    parent_number = parent.number,
                    parent_hash = %parent.hash,
                    local_safe_hash = %local_safe.hash,
                    "Consolidation attributes parent conflicts with the local-safe head"
                );
                return Err(ConsolidateTaskError::ParentConflictsWithLocalSafe);
            }
        }

        // Derivation drives consolidation, so the comparison is against the *local*-safe head.
        // Reading cross-safe here would re-consolidate already-consolidated blocks whenever
        // cross-safe lags local-safe under interop.
        let local_safe_head_number = match &self.input {
            ConsolidateInput::Attributes { .. } => {
                state.sync_state.local_safe_head().block_info.number
            }
            ConsolidateInput::BlockInfo(local_safe_block_info) => {
                local_safe_block_info.block_info.number
            }
        };
        if local_safe_head_number < state.sync_state.unsafe_head().block_info.number {
            self.consolidate(state).await
        } else {
            self.reconcile_unsafe_to_local_safe(state).await
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ImportedBlockSink, test_utils::MockEngineClient};
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

    /// A consolidate task over a client with no block at the fetched height.
    fn task_without_block(block_num: u64) -> ConsolidateTask<MockEngineClient> {
        task_with_client(
            block_num,
            MockEngineClient::builder().with_config(Arc::new(RollupConfig::default())).build(),
        )
    }

    fn task_with_client(
        block_num: u64,
        client: MockEngineClient,
    ) -> ConsolidateTask<MockEngineClient> {
        let safe_l2 = L2BlockInfo {
            block_info: BlockInfo { number: block_num, ..Default::default() },
            ..Default::default()
        };
        ConsolidateTask::new(
            Arc::new(client),
            Arc::new(RollupConfig::default()),
            ConsolidateInput::BlockInfo(safe_l2),
            None,
            Arc::new(crate::NoopBlockSink),
        )
    }

    /// A missing unsafe block is answered along op-node's split
    /// (`op-node/rollup/attributes/attributes.go:204-221`): a paced temporary stall while the
    /// initial EL sync is still filling the height in, and a reset once EL sync has finished —
    /// so the walkback realigns the unsafe head instead of the same fetch being retried forever.
    #[tokio::test]
    async fn missing_unsafe_block_stalls_during_el_sync_and_resets_after() {
        use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};

        let task = task_without_block(4);

        let mut state = EngineState { el_sync_finished: false, ..Default::default() };
        let err = task.consolidate(&mut state).await.unwrap_err();
        assert!(matches!(err, ConsolidateTaskError::AwaitingELSyncUnsafeL2Block(4)));
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Temporary);

        state.el_sync_finished = true;
        let err = task.consolidate(&mut state).await.unwrap_err();
        assert!(matches!(err, ConsolidateTaskError::MissingUnsafeL2Block(4)));
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Reset);
    }

    /// An RPC error that heuristically says the block is absent is folded into the same miss
    /// split, hardening against non-conforming execution layers exactly as op-node's
    /// `MaybeAsNotFoundErr` does (`op-service/eth/errors.go:10-27`); any other RPC failure stays
    /// a plain temporary fetch failure (`op-node/rollup/attributes/attributes.go:222-227`).
    #[tokio::test]
    async fn not_found_rpc_errors_map_to_the_miss_split() {
        use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};

        let mut state = EngineState { el_sync_finished: true, ..Default::default() };

        for message in ["block not found", "HEADER NOT FOUND", "Unknown block 0x4"] {
            let client = MockEngineClient::builder()
                .with_config(Arc::new(RollupConfig::default()))
                .with_l2_block_by_label_error(BlockNumberOrTag::Number(4), message)
                .build();
            let err = task_with_client(4, client).consolidate(&mut state).await.unwrap_err();
            assert!(
                matches!(err, ConsolidateTaskError::MissingUnsafeL2Block(4)),
                "{message} must map to the miss split, got {err:?}"
            );
        }

        let client = MockEngineClient::builder()
            .with_config(Arc::new(RollupConfig::default()))
            .with_l2_block_by_label_error(BlockNumberOrTag::Number(4), "connection refused")
            .build();
        let err = task_with_client(4, client).consolidate(&mut state).await.unwrap_err();
        assert!(matches!(err, ConsolidateTaskError::FailedToFetchUnsafeL2Block));
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Temporary);
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
        let task = ConsolidateTask::new(
            client,
            cfg,
            ConsolidateInput::BlockInfo(safe_l2),
            None,
            sink.clone(),
        );

        task.consolidate(&mut EngineState::default()).await.unwrap();

        assert_eq!(sink.0.lock().unwrap().as_slice(), &[block_hash]);
    }
}
