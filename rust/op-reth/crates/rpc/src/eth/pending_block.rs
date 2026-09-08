//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use reth_chain_state::BlockState;
use reth_optimism_flashblocks::PendingFlashBlock;
use reth_primitives_traits::AlloyBlockHeader;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert, RpcNodeCore, RpcNodeCoreExt,
    helpers::{LoadPendingBlock, SpawnBlocking, pending_block::PendingEnvBuilder},
};
use reth_rpc_eth_types::{
    EthApiError, PendingBlock, block::BlockAndReceipts, builder::config::PendingBlockKind,
    error::FromEthApiError,
};
use reth_storage_api::{BlockReaderIdExt, StateProviderBox, StateProviderFactory};
use tracing::debug;

#[inline]
const fn pending_state_history_lookup_hash<N: reth_primitives_traits::NodePrimitives>(
    pending_block: &PendingFlashBlock<N>,
) -> B256 {
    pending_block.canonical_anchor_hash
}

/// Returns the pending flashblock only while the canonical chain has not caught up with it.
///
/// A flashblocks upstream can stop delivering without closing its connection, which leaves the
/// last flashblock in place indefinitely while the node keeps importing canonical blocks. Once
/// the canonical tip reaches that height the flashblock only describes a block that is already
/// sealed, and answering `pending` from its partial state reports *behind* `latest` - an
/// `eth_getTransactionCount(addr, "pending")` below the same call against `latest`, which is not
/// a legal result and hands callers an already-consumed nonce. Dropping it makes those reads fall
/// back to `latest`, which is complete and self-consistent.
///
/// The test is against the canonical tip rather than against the flashblock's age on purpose: a
/// flashblock that is merely old but still ahead of the tip is the best answer available, and
/// discarding it would *lower* the nonce that `pending` had already reported - the very failure
/// this guards against.
///
/// See <https://github.com/ethereum-optimism/optimism/issues/22816>.
pub(super) fn unsuperseded_pending_flashblock<N: reth_primitives_traits::NodePrimitives>(
    pending_block: Option<&PendingFlashBlock<N>>,
    latest_block_number: u64,
) -> Option<PendingFlashBlock<N>> {
    let pending_block = pending_block?;
    let pending_block_number = pending_block.pending.block().header().number();

    if pending_block_number <= latest_block_number {
        debug!(
            target: "flashblocks",
            pending_block_number,
            latest_block_number,
            flashblock_index = pending_block.last_flashblock_index,
            "Pending flashblock superseded by the canonical chain, falling back to latest"
        );

        return None;
    }

    Some(pending_block.clone())
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
    use super::{pending_state_history_lookup_hash, unsuperseded_pending_flashblock};
    use alloy_consensus::Header;
    use alloy_primitives::B256;
    use reth_chain_state::ExecutedBlock;
    use reth_optimism_flashblocks::PendingFlashBlock;
    use reth_optimism_primitives::OpPrimitives;
    use reth_primitives_traits::RecoveredBlock;
    use reth_rpc_eth_types::PendingBlock;
    use std::{sync::Arc, time::Instant};

    fn pending_flashblock(block_number: u64) -> PendingFlashBlock<OpPrimitives> {
        let block = alloy_consensus::Block::new(
            Header { number: block_number, ..Default::default() },
            Default::default(),
        );
        let executed_block = ExecutedBlock::<OpPrimitives> {
            recovered_block: Arc::new(RecoveredBlock::new_unhashed(block, Vec::new())),
            ..Default::default()
        };

        PendingFlashBlock::new(
            PendingBlock::<OpPrimitives>::with_executed_block(Instant::now(), executed_block),
            B256::ZERO,
            0,
            B256::ZERO,
            false,
        )
    }

    #[test]
    fn pending_flashblock_ahead_of_the_canonical_tip_is_served() {
        let pending = pending_flashblock(101);

        assert!(
            unsuperseded_pending_flashblock(Some(&pending), 100).is_some(),
            "a flashblock building the next block is exactly what `pending` is for"
        );
    }

    #[test]
    fn pending_flashblock_at_the_canonical_tip_is_not_served() {
        let pending = pending_flashblock(100);

        assert!(
            unsuperseded_pending_flashblock(Some(&pending), 100).is_none(),
            "once the block is sealed canonically its partial flashblock state is behind `latest`"
        );
    }

    #[test]
    fn pending_flashblock_behind_the_canonical_tip_is_not_served() {
        // The frozen-feed case: the flashblock stopped advancing while the chain kept importing.
        let pending = pending_flashblock(100);

        assert!(
            unsuperseded_pending_flashblock(Some(&pending), 4_000).is_none(),
            "a flashblock the chain has long passed must not answer `pending`"
        );
    }

    #[test]
    fn missing_pending_flashblock_stays_missing() {
        assert!(unsuperseded_pending_flashblock::<OpPrimitives>(None, 100).is_none());
    }

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
