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
    ///
    /// Only block headers are read: an output root needs the block's hash and roots, not its
    /// transactions.
    pub async fn output_at_ancestor(
        &self,
        block_number: u64,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(BlockNumHash, OutputRoot)>, EngineClientError> {
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
    ) -> Result<Option<(BlockNumHash, OutputRoot)>, EngineClientError> {
        trace!(?recorded_head, "Reading output of the recorded head");
        let Some(output) = self.header_output_at_id(BlockId::hash(recorded_head.hash)).await?
        else {
            debug!(?recorded_head, "EL does not have the recorded head");
            return Ok(None);
        };
        trace!(?recorded_head, "Read output of the recorded head");
        if output.0 != recorded_head {
            warn!(
                ?recorded_head,
                returned = ?output.0,
                "EL returned a different block for the recorded head's hash"
            );
            return Ok(None);
        }
        Ok(self.is_canonical(recorded_head).await?.then_some(output))
    }

    /// A block below the recorded head is read by number, between two checks that the recorded
    /// head is canonical. Unlike a walk of parent hashes, those checks cannot tell a chain that
    /// switched away and back between them from one that never switched.
    async fn output_below_recorded_head(
        &self,
        block_number: u64,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(BlockNumHash, OutputRoot)>, EngineClientError> {
        if !self.is_canonical(recorded_head).await? {
            return Ok(None);
        }
        trace!(block_number, "Reading output of the canonical block at this height");
        let Some(output) = self.header_output_at_id(BlockId::number(block_number)).await? else {
            debug!(block_number, ?recorded_head, "EL has no canonical block at this height");
            return Ok(None);
        };
        trace!(block = ?output.0, "Read output of the canonical block");
        if output.0.number != block_number {
            warn!(block_number, returned = ?output.0, "EL returned a block at a different height");
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

    /// Reads the block with its transactions, which [`L2BlockInfo`] needs for the L1 origin and
    /// sequence number.
    async fn output_at_id(
        &self,
        block: BlockId,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        let Some(output_block) = self.engine.get_block(block).full().await? else {
            return Ok(None);
        };
        let output = self.output_of(&output_block.header).await?;
        let consensus_block = output_block.into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis::<op_alloy_consensus::OpTxEnvelope>(
            &consensus_block.map_transactions(|transaction| transaction.inner.inner.into_inner()),
            &self.cfg.genesis,
        )?;
        Ok(Some((block_info, output)))
    }

    /// Reads only the block's header. The hash the EL reports must be the header's own hash, as
    /// op-node requires, since nothing else checks the hash of a block read by number.
    async fn header_output_at_id(
        &self,
        block: BlockId,
    ) -> Result<Option<(BlockNumHash, OutputRoot)>, EngineClientError> {
        let Some(output_block) = self.engine.get_block(block).hashes().await? else {
            return Ok(None);
        };
        let header = &output_block.header;
        let computed = header.hash_slow();
        if computed != header.hash {
            return Err(EngineClientError::BlockHashMismatch { reported: header.hash, computed });
        }
        let output = self.output_of(header).await?;
        Ok(Some((BlockNumHash::new(header.number, header.hash), output)))
    }

    async fn output_of(
        &self,
        header: &alloy_rpc_types_eth::Header,
    ) -> Result<OutputRoot, EngineClientError> {
        let message_passer_storage_root = if self.cfg.is_isthmus_active(header.timestamp) {
            header.withdrawals_root.ok_or(EngineClientError::NoWithdrawalsRoot)?
        } else {
            // The proof must describe the fetched block even if its height changes branches.
            self.engine
                .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                .block_id(BlockId::hash(header.hash))
                .await?
                .storage_hash
        };
        Ok(OutputRoot::from_parts(header.state_root, message_passer_storage_root, header.hash))
    }
}
