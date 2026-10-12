//! Consolidating the safe head.

use super::Engine;
use crate::engine::{ConsolidateInput, ConsolidateTaskError};
use kona_engine::EngineSyncStateUpdate;
use kona_protocol::L2BlockInfo;

impl Engine {
    /// Makes a delegated safe block the unsafe and safe head.
    ///
    /// This provides symmetric fallback behavior to `build_and_seal` for derived attributes.
    async fn reconcile_to_safe_head(
        &mut self,
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
        self.synchronize(EngineSyncStateUpdate {
            unsafe_head: Some(*safe_l2),
            safe_head: Some(*safe_l2),
            local_safe_head: Some(*safe_l2),
            ..Default::default()
        })
        .await
        .map_err(|e| {
            warn!(target: "engine", ?e, "Apply safe head failed");
            e
        })?;

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
        &mut self,
        input: &ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        match input {
            ConsolidateInput::Attributes(attributes) => {
                self.build_and_seal((**attributes).clone(), true).await?;
                Ok(())
            }
            ConsolidateInput::BlockInfo(safe_l2) => self.reconcile_to_safe_head(safe_l2).await,
        }
    }

    /// Consolidates the unsafe block at the input's height if it matches the input, and otherwise
    /// reconciles the unsafe chain to the input.
    pub(super) async fn consolidate_with_unsafe_block(
        &mut self,
        input: &ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        // Fetch the unsafe L2 block
        let block_num = input.l2_block_number();
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
        let block_hash = block.header.hash;

        if input.is_consistent_with_block(&self.config, &block) {
            trace!(
                target: "engine",
                input = ?input,
                block_hash = %block_hash,
                "Consolidating engine state",
            );
            let consensus_block =
                block.into_consensus().map_transactions(|t| t.inner.inner.into_inner());
            match L2BlockInfo::from_block_and_genesis(&consensus_block, &self.config.genesis) {
                // Only issue a forkchoice update if the attributes are the last in the span
                // batch. This is an optimization to avoid sending a FCU
                // call for every block in the span batch.
                Ok(block_info) if !input.is_attributes_last_in_span() => {
                    // The next attributes built are this block's child, and ask for its config.
                    self.block_sink.block_imported(consensus_block, block_info);

                    // Apply a transient update to the safe head.
                    self.state.sync_state =
                        self.state.sync_state.apply_update(EngineSyncStateUpdate {
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

                    self.synchronize(EngineSyncStateUpdate {
                        safe_head: Some(block_info),
                        local_safe_head: Some(block_info),
                        ..Default::default()
                    })
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
                    warn!(target: "engine", ?e, "Failed to construct L2BlockInfo, proceeding to build");
                }
            }
        }

        debug!(
            target: "engine",
            input = ?input,
            block_hash = %block_hash,
            "ConsolidateInput mismatch! Initiating reorg",
        );
        // Handle mismatch case - called when consistency check fails
        // or when L2BlockInfo construction fails in Attributes branch
        self.reconcile_unsafe_to_safe(input).await
    }

    /// Advances the safe head to the input, reorging the unsafe chain to it if necessary.
    ///
    /// Behavior depends on how the safe head is provided:
    ///
    /// - `Attributes`: The safe head is advanced through the normal derivation flow, where the
    ///   derivation and engine actors coordinate both safe and unsafe heads. In this case, we
    ///   consolidate as long as the unsafe head has not fallen behind.
    ///
    /// - `BlockInfo`: The safe head is injected externally by the derivation actor while delegating
    ///   derivation, and is not coordinated with the engine actor's safe and unsafe heads. If the
    ///   injected safe head is ahead of the engine actor's unsafe head, we reconcile the unsafe
    ///   chain up to the safe head instead of consolidating.
    pub(in crate::engine) async fn consolidate_input(
        &mut self,
        input: &ConsolidateInput,
    ) -> Result<(), ConsolidateTaskError> {
        let safe_head_number = match input {
            ConsolidateInput::Attributes { .. } => {
                self.state.sync_state.safe_head().block_info.number
            }
            ConsolidateInput::BlockInfo(safe_block_info) => safe_block_info.block_info.number,
        };
        if safe_head_number < self.state.sync_state.unsafe_head().block_info.number {
            self.consolidate_with_unsafe_block(input).await
        } else {
            self.reconcile_unsafe_to_safe(input).await
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use kona_engine::{EngineState, ImportedBlockSink, test_utils::test_engine_client};
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::BlockInfo;
    use std::sync::Arc;

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
        let safe_l2 = L2BlockInfo {
            block_info: BlockInfo { hash: block_hash, number: 0, ..Default::default() },
            ..Default::default()
        };
        let sink = Arc::new(RecordingSink::default());
        let mut engine = Engine::new(
            Arc::new(client),
            cfg,
            sink.clone(),
            EngineState::default(),
            tokio::sync::watch::channel(EngineState::default()).0,
        );

        engine.consolidate_with_unsafe_block(&ConsolidateInput::BlockInfo(safe_l2)).await.unwrap();

        assert_eq!(sink.0.lock().unwrap().as_slice(), &[block_hash]);
        l1.assert_finished();
        l2.assert_finished();
    }
}
