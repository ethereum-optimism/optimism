//! State provider factory for OP Proofs ExEx.

use alloy_eips::BlockId;
use derive_more::Constructor;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStorageError, OpProofsStore, api::OpProofsProviderRO,
    provider::OpProofsStateProviderRef,
};
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

        let provider_ro = self.preimage_store.provider_ro().map_err(ProviderError::from)?;

        // An empty proof window is semantically equivalent to an out-of-window lookup for
        // the proofs RPC: there's no proof for that block. Any other storage error is real
        // and propagates up.
        let window = match provider_ro.get_proof_window() {
            Ok(w) => w,
            Err(OpProofsStorageError::NoBlocksFound) => {
                return Err(ProviderError::StateForNumberNotFound(block_number));
            }
            Err(err) => return Err(ProviderError::from(err)),
        };

        if block_number < window.earliest.number || block_number > window.latest.number {
            return Err(ProviderError::StateForNumberNotFound(block_number));
        }

        let external_overlay_provider =
            OpProofsStateProviderRef::new(historical_provider, provider_ro, block_number);

        Ok(Box::new(external_overlay_provider))
    }
}
