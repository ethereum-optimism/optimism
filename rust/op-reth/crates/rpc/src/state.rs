//! State provider factory for OP Proofs ExEx.

use alloy_eips::BlockId;
use derive_more::Constructor;
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStorageError, OpProofsStore, api::OpProofsProviderRO,
    provider::OpProofsStateProviderRef,
};
use reth_provider::{BlockHashReader, BlockIdReader, ProviderError, ProviderResult, StateProvider};
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
    Eth: FullEthApi + 'static,
    P: OpProofsStore + Clone + 'a,
{
    /// Creates a state provider for the given block id.
    pub async fn state_provider(
        &'a self,
        block_id: BlockId,
    ) -> ProviderResult<Box<dyn StateProvider + 'a>> {
        let block_number = canonical_block_number(&self.eth_api, block_id)?;

        let historical_provider =
            self.eth_api.state_at_block_id(block_id).await.map_err(ProviderError::other)?;

        let provider_ro = self.preimage_store.provider_ro().map_err(ProviderError::from)?;

        // Check the window in the transaction the overlay reads from, so pruning cannot move it
        // in between.
        ensure_in_proof_window(&provider_ro, block_number)?;

        let external_overlay_provider =
            OpProofsStateProviderRef::new(historical_provider, provider_ro, block_number);

        Ok(Box::new(external_overlay_provider))
    }
}

/// Resolves `block_id` to a canonical block number inside the proofs storage window.
///
/// Callers that serve a proof check this before taking a proof permit, so out-of-window requests
/// are rejected without holding a permit or a blocking-task slot.
/// [`OpStateProviderFactory::state_provider`] checks the window again in the transaction it
/// reads from, because pruning can move it in between.
pub fn ensure_within_proofs_window<Eth, P>(
    eth_api: &Eth,
    preimage_store: &OpProofsStorage<P>,
    block_id: BlockId,
) -> ProviderResult<u64>
where
    Eth: FullEthApi,
    P: OpProofsStore,
{
    let block_number = canonical_block_number(eth_api, block_id)?;
    let provider_ro = preimage_store.provider_ro().map_err(ProviderError::from)?;
    ensure_in_proof_window(&provider_ro, block_number)?;
    Ok(block_number)
}

/// Resolves `block_id` to a block number, rejecting a hash that names a non-canonical block.
fn canonical_block_number<Eth: FullEthApi>(
    eth_api: &Eth,
    block_id: BlockId,
) -> ProviderResult<u64> {
    let block_number = eth_api
        .provider()
        .block_number_for_id(block_id)?
        .ok_or(EthApiError::HeaderNotFound(block_id))
        .map_err(ProviderError::other)?;

    // The proofs storage is canonical-only and is overlaid by block number, so a hash that
    // names a non-canonical block would be served canonical state for its height.
    if let BlockId::Hash(hash) = block_id &&
        eth_api.provider().block_hash(block_number)? != Some(hash.block_hash)
    {
        return Err(ProviderError::other(EthApiError::HeaderNotFound(block_id)));
    }

    Ok(block_number)
}

/// Rejects a block outside the proofs storage window visible to `provider`.
fn ensure_in_proof_window(
    provider: &impl OpProofsProviderRO,
    block_number: u64,
) -> ProviderResult<()> {
    // An empty proof window is semantically equivalent to an out-of-window lookup for the proofs
    // RPC: there's no proof for that block. Any other storage error is real and propagates up.
    let window = match provider.get_proof_window() {
        Ok(w) => w,
        Err(OpProofsStorageError::NoBlocksFound) => {
            return Err(ProviderError::StateForNumberNotFound(block_number));
        }
        Err(err) => return Err(ProviderError::from(err)),
    };

    if block_number < window.earliest.number || block_number > window.latest.number {
        return Err(ProviderError::StateForNumberNotFound(block_number));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use reth_optimism_trie::{InMemoryProofsStorage, OpProofsInitProvider};

    #[test]
    fn proof_window_check_rejects_blocks_outside_the_stored_window() {
        let storage = InMemoryProofsStorage::new();
        assert!(matches!(
            ensure_in_proof_window(&storage.provider_ro().unwrap(), 10),
            Err(ProviderError::StateForNumberNotFound(10))
        ));

        let init = storage.initialization_provider().unwrap();
        init.set_initial_state_anchor(BlockNumHash::new(10, B256::repeat_byte(1))).unwrap();
        init.commit_initial_state().unwrap();

        let provider = storage.provider_ro().unwrap();
        ensure_in_proof_window(&provider, 10).unwrap();
        for block_number in [9, 11] {
            assert!(matches!(
                ensure_in_proof_window(&provider, block_number),
                Err(ProviderError::StateForNumberNotFound(n)) if n == block_number
            ));
        }
    }
}
