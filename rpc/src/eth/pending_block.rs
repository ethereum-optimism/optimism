//! Loads OP pending block for a RPC response.

use crate::{OpEthApi, OpEthApiError};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumberOrTag;
use reth_chain_state::BlockState;
use reth_rpc_eth_api::{
    helpers::{pending_block::PendingEnvBuilder, LoadPendingBlock, SpawnBlocking},
    FromEvmError, RpcConvert, RpcNodeCore,
};
use reth_rpc_eth_types::{
    block::BlockAndReceipts, builder::config::PendingBlockKind, error::FromEthApiError,
    EthApiError, PendingBlock,
};
use reth_storage_api::{
    BlockReader, BlockReaderIdExt, ReceiptProvider, StateProviderBox, StateProviderFactory,
};
use std::sync::Arc;

impl<N, Rpc> LoadPendingBlock for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives>,
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

    /// Returns the locally built pending block
    async fn local_pending_block(
        &self,
    ) -> Result<Option<BlockAndReceipts<Self::Primitives>>, Self::Error> {
        if let Ok(Some(pending)) = self.pending_flashblock() {
            return Ok(Some(pending.into_block_and_receipts()));
        }

        // See: <https://github.com/ethereum-optimism/op-geth/blob/f2e69450c6eec9c35d56af91389a1c47737206ca/miner/worker.go#L367-L375>
        let latest = self
            .provider()
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let block_id = latest.hash().into();
        let block = self
            .provider()
            .recovered_block(block_id, Default::default())?
            .ok_or(EthApiError::HeaderNotFound(block_id.into()))?;

        let receipts = self
            .provider()
            .receipts_by_block(block_id)?
            .ok_or(EthApiError::ReceiptsNotFound(block_id.into()))?;

        Ok(Some(BlockAndReceipts { block: Arc::new(block), receipts: Arc::new(receipts) }))
    }

    /// Returns a [`StateProviderBox`] on a mem-pool built pending block overlaying latest.
    async fn local_pending_state(&self) -> Result<Option<StateProviderBox>, Self::Error>
    where
        Self: SpawnBlocking,
    {
        let Ok(Some(pending_block)) = self.pending_flashblock() else {
            return Ok(None);
        };

        let latest_historical = self
            .provider()
            .history_by_block_hash(pending_block.block().parent_hash())
            .map_err(Self::Error::from_eth_err)?;

        let state = BlockState::from(pending_block);

        Ok(Some(Box::new(state.state_provider(latest_historical)) as StateProviderBox))
    }
}
