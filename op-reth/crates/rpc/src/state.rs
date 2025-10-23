//! State provider factory for OP Proofs ExEx.

use alloy_eips::BlockId;
use derive_more::Constructor;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{provider::OpProofsStateProviderRef, OpProofsStorage, OpProofsStore};
use reth_provider::{BlockIdReader, ProviderError, ProviderResult, StateProvider};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;

/// Creates a factory for state providers using OP Proofs external proofs storage.
#[derive(Debug, Constructor)]
pub struct OpStateProviderFactory<Eth, P> {
    eth_api: Eth,
    preimage_store: OpProofsStorage<P>,
}

impl<'a, Eth, P> OpStateProviderFactory<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'a,
{
    /// Creates a state provider for the given block id.
    pub async fn state_provider(
        &'a self,
        block_id: Option<BlockId>,
    ) -> ProviderResult<Box<dyn StateProvider + 'a>> {
        let block_id = block_id.unwrap_or_default();
        // Check whether the distance to the block exceeds the maximum configured window.
        let block_number = self
            .eth_api
            .provider()
            .block_number_for_id(block_id)?
            .ok_or(EthApiError::HeaderNotFound(block_id))
            .map_err(ProviderError::other)?;

        let historical_provider =
            self.eth_api.state_at_block_id(block_id).await.map_err(ProviderError::other)?;

        let (Some((latest_block_number, _)), Some((earliest_block_number, _))) = (
            self.preimage_store
                .get_latest_block_number()
                .await
                .map_err(|e| ProviderError::Database(e.into()))?,
            self.preimage_store
                .get_earliest_block_number()
                .await
                .map_err(|e| ProviderError::Database(e.into()))?,
        ) else {
            // if no earliest block, db is empty - use historical provider
            return Ok(historical_provider);
        };

        if block_number < earliest_block_number || block_number > latest_block_number {
            return Ok(historical_provider);
        }

        let external_overlay_provider =
            OpProofsStateProviderRef::new(historical_provider, &self.preimage_store, block_number);

        Ok(Box::new(external_overlay_provider))
    }
}
