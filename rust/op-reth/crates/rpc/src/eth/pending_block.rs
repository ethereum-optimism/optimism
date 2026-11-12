//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError, eth::memory_overlay::MemoryOverlayStateProvider};
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

/// Returns the pending state: the tip's bundle on its canonical anchor's state.
///
/// The sequencer streams flashblocks ahead of block import, so the flashblocks worker can build
/// past the canonical head. With canonical head N-1, it builds pending N on N-1, then speculative
/// N+1 on N, and so on, all anchored at N-1. A tip on the anchor, such as N, opens through reth's
/// `state_with_block_appended`. A speculative tip opens through the vendored
/// [`MemoryOverlayStateProvider`]: its bundle holds every write since the anchor, so account,
/// storage and code reads need only the tip. Block hash lookups are the exception: they see only
/// the tip, not the pending blocks before it.
///
/// reth's `state_with_block_appended(parent_hash, block)` can't open a speculative tip such as N+1:
///
/// - With N, it fails with `BlockHashNotFound`. N is in neither the database nor the overlay
///   manager, which holds only blocks the engine executed.
/// - With N-1, it panics on the `debug_assert` in `OverlayBuilder::with_appended_block` that the
///   hash is the header's parent.
///
/// Appending the whole chain instead needs it at the RPC
/// (<https://github.com/ethereum-optimism/optimism/issues/23998>) and reth access we lack:
/// `state_with_block_appended` takes one block, and `ProviderFactory::overlay_manager` is
/// crate-private.
fn pending_flashblock_state<P, N>(
    provider: &P,
    pending_block: PendingFlashBlock<N>,
) -> ProviderResult<StateProviderBox>
where
    N: NodePrimitives,
    P: StateProviderFactory<Primitives = N>,
{
    let anchor = pending_state_history_lookup_hash(&pending_block);
    let speculative = pending_block.pending.parent_hash() != anchor;
    let tip = pending_block.pending.executed_block;
    if !speculative {
        return provider.state_with_block_appended(anchor, tip);
    }
    let anchor_historical = provider.history_by_block_hash(anchor)?;
    Ok(MemoryOverlayStateProvider::new(anchor_historical, vec![tip]).boxed())
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

    /// UPSTREAM-MIRROR(override): reth@op-v2.7.0-1
    /// `reth_rpc_eth_api::helpers::pending_block::LoadPendingBlock::local_pending_state`
    ///
    /// Serves the flashblock pending state instead of the pool-built one. Keep upstream's blocking
    /// discipline: callers await this on the async runtime, so the reads that open the state go
    /// through `spawn_blocking_io`, as the default does.
    ///
    /// Returns a [`StateProviderBox`] on a mem-pool built pending block overlaying latest.
    async fn local_pending_state(&self) -> Result<Option<StateProviderBox>, Self::Error>
    where
        Self: SpawnBlocking,
    {
        let Ok(Some(pending_block)) = self.pending_flashblock().await else {
            return Ok(None);
        };
        // Callers resolve pending state before taking a blocking thread, so the reads that open it
        // take one here, as reth's default does.
        self.spawn_blocking_io(move |this| {
            pending_flashblock_state(this.provider(), pending_block)
                .map(Some)
                .map_err(Self::Error::from_eth_err)
        })
        .await
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
    use reth_chainspec::MAINNET;
    use reth_db_common::init::init_genesis;
    use reth_ethereum_primitives::{Block, EthPrimitives};
    use reth_execution_types::BlockExecutionOutput;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_primitives_traits::{RecoveredBlock, SealedBlock};
    use reth_provider::{providers::BlockchainProvider, test_utils::create_test_provider_factory};
    use reth_rpc_eth_types::PendingBlock;
    use reth_storage_api::{AccountReader, StateProvider, StateProviderFactory};
    use reth_trie::{ComputedTrieData, HashedPostState, KeccakKeyHasher};
    use revm::{
        database::{AccountStatus, BundleState},
        state::AccountInfo,
    };
    use rstest::rstest;
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

    /// Builds `blocks` pending blocks on the canonical genesis and reads the tip's state the way
    /// `local_pending_state` does: through reth's overlay for a tip on the anchor, through the
    /// vendored one for a speculative tip. Block `n` sets the balance of account `n` to `n`, and
    /// the first block also writes a storage slot and destroys a genesis account. As in the
    /// flashblocks worker, a speculative tip's bundle holds every write since the anchor.
    ///
    /// These are spot checks that the speculative gap is bridged, not a parity suite against reth's
    /// `OverlayStateProvider`: the vendored overlay is meant to go away (see its module doc).
    #[rstest]
    #[case::canonical(1)]
    #[case::speculative(2)]
    fn pending_flashblock_state_includes_every_pending_block(#[case] blocks: u64) {
        let factory = create_test_provider_factory();
        let anchor = init_genesis(&factory).unwrap();
        let provider = BlockchainProvider::new(factory).unwrap();

        let account = |number: u64| Address::with_last_byte(number as u8);
        let destroyed = *MAINNET.genesis.alloc.keys().next().expect("mainnet genesis has accounts");
        assert!(provider.latest().unwrap().basic_account(&destroyed).unwrap().is_some());
        let (slot, value) = (U256::from(1), U256::from(42));

        let mut bundle = BundleState::builder(1..=blocks)
            .state_storage(account(1), std::iter::once((slot, (U256::ZERO, value))).collect())
            .state_original_account_info(destroyed, AccountInfo::default());
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
        let mut bundle = bundle.build();
        bundle.state.get_mut(&destroyed).expect("in the bundle").status = AccountStatus::Destroyed;
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
        assert_eq!(state.storage(account(1), B256::from(slot)).unwrap(), Some(value));
        assert_eq!(state.basic_account(&destroyed).unwrap(), None, "destroyed account");
    }
}
