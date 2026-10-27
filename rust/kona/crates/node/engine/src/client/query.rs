use super::EngineClientError;
use alloy_eips::{BlockId, BlockNumHash, BlockNumberOrTag};
use alloy_provider::{Provider, RootProvider};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, Predeploys};
use op_alloy_network::Optimism;
use std::sync::Arc;

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
        self.output_at_id(block.into()).await
    }

    /// Computes an ancestor's output, checking the recorded head is canonical before and after.
    /// Returns `None` when the anchor or its ancestry cannot be confirmed.
    pub async fn output_at_ancestor(
        &self,
        block_number: u64,
        recorded_head: BlockNumHash,
    ) -> Result<Option<(L2BlockInfo, OutputRoot)>, EngineClientError> {
        if block_number > recorded_head.number {
            return Ok(None);
        }
        let Some(mut block) = self.engine.get_block_by_number(recorded_head.number.into()).await?
        else {
            return Ok(None);
        };
        if block.header.hash != recorded_head.hash || block.header.number != recorded_head.number {
            return Ok(None);
        }
        while block.header.number > block_number {
            let parent_hash = block.header.parent_hash;
            let parent_number = block.header.number - 1;
            let Some(parent) = self.engine.get_block_by_hash(parent_hash).await? else {
                return Ok(None);
            };
            if parent.header.hash != parent_hash || parent.header.number != parent_number {
                return Ok(None);
            }
            block = parent;
        }
        let output = self.output_at_id(BlockId::hash(block.header.hash)).await?;
        if output.0.block_info.id() != BlockNumHash::new(block.header.number, block.header.hash) {
            return Ok(None);
        }
        let canonical = self.engine.get_block_by_number(recorded_head.number.into()).await?;
        if canonical.is_none_or(|block| block.header.hash != recorded_head.hash) {
            return Ok(None);
        }
        Ok(Some(output))
    }

    async fn output_at_id(
        &self,
        block: BlockId,
    ) -> Result<(L2BlockInfo, OutputRoot), EngineClientError> {
        let output_block =
            self.engine.get_block(block).full().await?.ok_or_else(|| match block {
                BlockId::Number(number) => EngineClientError::NoL2BlockFound(number),
                BlockId::Hash(hash) => EngineClientError::NoL2BlockFoundByHash(hash.block_hash),
            })?;
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
        Ok((block_info, output))
    }
}
