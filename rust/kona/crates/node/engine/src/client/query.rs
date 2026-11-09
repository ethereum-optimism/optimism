use super::EngineClientError;
use alloy_eips::{BlockId, BlockNumHash, BlockNumberOrTag};
use alloy_provider::{Provider, RootProvider};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, Predeploys};
use op_alloy_network::Optimism;
use std::{cmp::Ordering, sync::Arc};
use tracing::{debug, trace, warn};

/// Read-only execution-layer handle that computes L2 outputs without Engine API mutations.
#[derive(Clone, Debug)]
pub struct EngineQueryClient {
    pub(super) cfg: Arc<RollupConfig>,
    pub(super) engine: RootProvider<Optimism>,
}

impl EngineQueryClient {
    /// Reads the L2 block and computes its output using the active fork rules.
    pub async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> Result<(L2BlockInfo, OutputRoot), EngineClientError> {
        self.output_at_id(block.into()).await?.ok_or(EngineClientError::NoL2BlockFound(block))
    }

    /// Computes the output of block `block_number` on the canonical chain through `recorded_head`.
    ///
    /// Returns `None` when the EL cannot confirm that block right now, for example because
    /// `recorded_head` is not canonical or the chain was rewound during the query. A retry once
    /// the chain settles can succeed.
    pub async fn output_at_ancestor(
        &self,
        block_number: u64,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        match block_number.cmp(&recorded_head.number) {
            Ordering::Equal => self.output_at_recorded_head(recorded_head).await,
            Ordering::Less => self.output_below_recorded_head(block_number, recorded_head).await,
            Ordering::Greater => Ok(None),
        }
    }

    /// The recorded head is read by its hash, so it only has to be canonical once the read is done.
    async fn output_at_recorded_head(
        &self,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        trace!(?recorded_head, "Reading output of the recorded head");
        let Some(output) = self.output_at_id(BlockId::hash(recorded_head.hash)).await? else {
            warn!(?recorded_head, "EL does not have the recorded head");
            return Ok(None);
        };
        trace!(?recorded_head, "Read output of the recorded head");
        if output.0.block_info.id() != recorded_head {
            warn!(
                ?recorded_head,
                returned = ?output.0.block_info.id(),
                "EL returned a different block for the recorded head's hash"
            );
            return Ok(None);
        }
        Ok(self.is_canonical(recorded_head).await?.then_some(output))
    }

    /// A block below the recorded head is read by number, between two checks that the recorded
    /// head is canonical. Unlike a walk of parent hashes, those checks cannot tell a chain that
    /// switched away and back between them from one that never switched. While the local-safe
    /// head at or above `recorded_head` stays the same, such a switch would need the rollup node
    /// to rewind and re-derive every block back up to it within this one read, so in practice it
    /// does not happen.
    async fn output_below_recorded_head(
        &self,
        block_number: u64,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        if !self.is_canonical(recorded_head).await? {
            return Ok(None);
        }
        trace!(block_number, "Reading output of the canonical block at this height");
        let Some(output) = self.output_at_id(BlockId::number(block_number)).await? else {
            warn!(
                block_number,
                ?recorded_head,
                "EL has no block at this height below the canonical recorded head"
            );
            return Ok(None);
        };
        trace!(block = ?output.0.block_info.id(), "Read output of the canonical block");
        if output.0.block_info.number != block_number {
            warn!(
                block_number,
                returned = ?output.0.block_info.id(),
                "EL returned a block at a different height"
            );
            return Ok(None);
        }
        Ok(self.is_canonical(recorded_head).await?.then_some(output))
    }

    async fn is_canonical(&self, block: BlockNumHash) -> Result<bool, EngineClientError> {
        trace!(?block, "Checking block is canonical");
        let canonical = self.engine.get_block_by_number(block.number.into()).await?;
        let is_canonical = canonical.is_some_and(|canonical| canonical.header.hash == block.hash);
        if is_canonical {
            trace!(?block, "Block is canonical");
        } else {
            debug!(?block, "Block is not canonical");
        }
        Ok(is_canonical)
    }

    async fn output_at_id(
        &self,
        block: BlockId,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        let Some(output_block) = self.engine.get_block(block).full().await? else {
            return Ok(None);
        };
        let consensus_block = output_block.clone().into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis::<op_alloy_consensus::OpTxEnvelope>(
            &consensus_block.map_transactions(|transaction| transaction.inner.inner.into_inner()),
            &self.cfg.genesis,
        )?;
        let message_passer_storage_root =
            if self.cfg.is_isthmus_active(output_block.header.timestamp) {
                output_block.header.withdrawals_root.ok_or(EngineClientError::NoWithdrawalsRoot)?
            } else {
                // The proof must describe the fetched block even if its height changes branches.
                self.engine
                    .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                    .block_id(BlockId::hash(output_block.header.hash))
                    .await?
                    .storage_hash
            };
        let output = OutputRoot::from_parts(
            output_block.header.state_root,
            message_passer_storage_root,
            output_block.header.hash,
        );
        Ok(Some((block_info, output)))
    }
}
