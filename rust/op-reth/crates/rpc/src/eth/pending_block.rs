//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use reth_chain_state::BlockState;
use reth_optimism_flashblocks::PendingFlashBlock;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert, RpcNodeCore, RpcNodeCoreExt,
    helpers::{LoadPendingBlock, SpawnBlocking, pending_block::PendingEnvBuilder},
};
use reth_rpc_eth_types::{
    EthApiError, PendingBlock, block::BlockAndReceipts, builder::config::PendingBlockKind,
    error::FromEthApiError,
};
use reth_storage_api::{BlockReaderIdExt, StateProviderBox, StateProviderFactory};

#[inline]
const fn pending_state_history_lookup_hash<N: reth_primitives_traits::NodePrimitives>(
    pending_block: &PendingFlashBlock<N>,
) -> B256 {
    pending_block.canonical_anchor_hash
}

impl<N, Rpc> LoadPendingBlock for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
    #[inline]
    fn pending_block(&self) -> &tokio::sync::Mutex<Option<PendingBlock<N::Primitives>>> {
        self.inner.eth_api.pending_block()
    }

    #[inline]
    fn pending_env_builder(&self) -> &dyn PendingEnvBuilder<Self::Evm> {
        self.inner.eth_api.pending_env_builder()
    }

    #[inline]
    fn pending_block_kind(&self) -> PendingBlockKind {
        self.inner.eth_api.pending_block_kind()
    }

    /// Returns a [`StateProviderBox`] on a mem-pool built pending block overlaying latest.
    async fn local_pending_state(&self) -> Result<Option<StateProviderBox>, Self::Error>
    where
        Self: SpawnBlocking,
    {
        let Ok(Some(pending_block)) = self.pending_flashblock().await else {
            return Ok(None);
        };
        let canonical_anchor_hash = pending_state_history_lookup_hash(&pending_block);
        let state = BlockState::from(pending_block.pending);

        let anchor_historical = self
            .provider()
            .history_by_block_hash(canonical_anchor_hash)
            .map_err(Self::Error::from_eth_err)?;

        Ok(Some(Box::new(state.state_provider(anchor_historical)) as StateProviderBox))
    }

    /// Returns the locally built pending block
    async fn local_pending_block(
        &self,
    ) -> Result<Option<BlockAndReceipts<Self::Primitives>>, Self::Error> {
        if let Ok(Some(pending)) = self.pending_flashblock().await {
            return Ok(Some(pending.pending.into_block_and_receipts()));
        }

        // See: <https://github.com/ethereum-optimism/op-geth/blob/f2e69450c6eec9c35d56af91389a1c47737206ca/miner/worker.go#L367-L375>
        let latest = self
            .provider()
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;

        let latest = self
            .cache()
            .get_block_and_receipts(latest.hash())
            .await
            .map_err(Self::Error::from_eth_err)?
            .map(|(block, receipts)| BlockAndReceipts { block, receipts });
        Ok(latest)
    }
}

#[cfg(test)]
mod tests {
    use super::pending_state_history_lookup_hash;
    use alloy_primitives::B256;
    use reth_chain_state::ExecutedBlock;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_rpc_eth_types::PendingBlock;
    use std::time::Instant;

    #[test]
    fn pending_state_prefers_canonical_anchor_over_parent_hash() {
        let pending = PendingBlock::<OpPrimitives>::with_executed_block(
            Instant::now(),
            ExecutedBlock::<OpPrimitives>::default(),
        );
        let parent_hash = pending.parent_hash();
        let canonical_anchor_hash = B256::from([0x11; 32]);
        assert_ne!(canonical_anchor_hash, parent_hash);

        let pending_flashblock = PendingFlashBlock::<OpPrimitives>::new(
            pending,
            canonical_anchor_hash,
            0,
            B256::ZERO,
            false,
        );

        assert_eq!(pending_state_history_lookup_hash(&pending_flashblock), canonical_anchor_hash);
    }
}
