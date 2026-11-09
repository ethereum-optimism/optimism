//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use reth_optimism_flashblocks::PendingFlashBlock;
use reth_primitives_traits::NodePrimitives;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert, RpcNodeCore, RpcNodeCoreExt,
    helpers::{LoadPendingBlock, SpawnBlocking, pending_block::PendingEnvBuilder},
};
use reth_rpc_eth_types::{
    EthApiError, PendingBlock, block::BlockAndReceipts, builder::config::PendingBlockKind,
    error::FromEthApiError,
};
use reth_storage_api::{
    BlockReaderIdExt, StateProviderBox, StateProviderFactory, errors::ProviderResult,
};

#[inline]
const fn pending_state_history_lookup_hash<N: NodePrimitives>(
    pending_block: &PendingFlashBlock<N>,
) -> B256 {
    pending_block.canonical_anchor_hash
}

/// Returns the state after `pending_block`: its canonical anchor's state with the block on top.
fn pending_flashblock_state<P, N>(
    provider: &P,
    pending_block: PendingFlashBlock<N>,
) -> ProviderResult<StateProviderBox>
where
    N: NodePrimitives,
    P: StateProviderFactory<Primitives = N>,
{
    provider.state_with_block_appended(
        pending_state_history_lookup_hash(&pending_block),
        pending_block.pending.executed_block,
    )
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
        let state = pending_flashblock_state(self.provider(), pending_block)
            .map_err(Self::Error::from_eth_err)?;

        Ok(Some(state))
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
            .map(|(block, receipts)| BlockAndReceipts::new(block, receipts));
        Ok(latest)
    }
}

#[cfg(test)]
mod tests {
    use super::{pending_flashblock_state, pending_state_history_lookup_hash};
    use alloy_consensus::Header;
    use alloy_primitives::{Address, B256, U256};
    use reth_chain_state::ExecutedBlock;
    use reth_db_common::init::init_genesis;
    use reth_ethereum_primitives::{Block, EthPrimitives};
    use reth_execution_types::BlockExecutionOutput;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_primitives_traits::{RecoveredBlock, SealedBlock};
    use reth_provider::{providers::BlockchainProvider, test_utils::create_test_provider_factory};
    use reth_rpc_eth_types::PendingBlock;
    use reth_storage_api::AccountReader;
    use reth_trie_common::{ComputedTrieData, HashedPostState, KeccakKeyHasher};
    use revm::{database::BundleState, state::AccountInfo};
    use std::{sync::Arc, time::Instant};

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

    /// Builds pending blocks N and N+1 on the canonical genesis N-1 and reads the tip's state the
    /// way `local_pending_state` does. Block `n` sets the balance of account `n` to `n`. As in the
    /// flashblocks worker, the speculative tip's bundle holds every write since the anchor.
    ///
    /// Fails for now: `state_with_block_appended` asserts that the tip's header parent (N) is the
    /// hash it was given (the anchor N-1), so the state cannot be opened at all.
    #[test]
    #[should_panic(expected = "assertion `left == right` failed")]
    fn pending_flashblock_state_includes_every_speculative_block() {
        let blocks = 2;
        let factory = create_test_provider_factory();
        let anchor = init_genesis(&factory).unwrap();
        let provider = BlockchainProvider::new(factory).unwrap();

        let account = |number: u64| Address::with_last_byte(number as u8);
        let mut bundle = BundleState::builder(1..=blocks);
        let mut parent_hash = anchor;
        let mut tip = None;
        for number in 1..=blocks {
            bundle = bundle.state_present_account_info(
                account(number),
                AccountInfo { balance: U256::from(number), ..Default::default() },
            );
            let header = Header { number, parent_hash, ..Default::default() };
            let block = SealedBlock::seal_slow(Block { header, body: Default::default() });
            parent_hash = block.hash();
            tip = Some(block);
        }
        let bundle = bundle.build();
        let hashed_state = HashedPostState::from_bundle_state::<KeccakKeyHasher>(bundle.state());
        let executed = ExecutedBlock::<EthPrimitives>::new(
            Arc::new(RecoveredBlock::new_sealed(tip.expect("at least one block"), Vec::new())),
            Arc::new(BlockExecutionOutput { state: bundle, ..Default::default() }),
            ComputedTrieData::new(Arc::new(hashed_state.into_sorted()), Arc::default()),
        );
        let pending = PendingFlashBlock::new(
            PendingBlock::with_executed_block(Instant::now(), executed),
            anchor,
            0,
            B256::ZERO,
            false,
        );

        let state = pending_flashblock_state(&provider, pending).unwrap();
        for number in 1..=blocks {
            let balance = state.basic_account(&account(number)).unwrap().map(|a| a.balance);
            assert_eq!(balance, Some(U256::from(number)), "account {number}");
        }
    }
}
