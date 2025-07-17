//! Loads OP pending block for a RPC response.

use std::sync::Arc;

use crate::OpEthApi;
use alloy_eips::BlockNumberOrTag;
use reth_chainspec::ChainSpecProvider;
use reth_evm::ConfigureEvm;
use reth_node_api::NodePrimitives;
use reth_primitives_traits::RecoveredBlock;
use reth_rpc_eth_api::{
    helpers::{pending_block::PendingEnvBuilder, LoadPendingBlock, SpawnBlocking},
    types::RpcTypes,
    EthApiTypes, FromEthApiError, FromEvmError, RpcConvert, RpcNodeCore,
};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{
    BlockReader, BlockReaderIdExt, ProviderBlock, ProviderHeader, ProviderReceipt, ProviderTx,
    ReceiptProvider, StateProviderFactory,
};
use reth_transaction_pool::{PoolTransaction, TransactionPool};

impl<N, Rpc> LoadPendingBlock for OpEthApi<N, Rpc>
where
    Self: SpawnBlocking
        + EthApiTypes<
            NetworkTypes: RpcTypes<
                Header = alloy_rpc_types_eth::Header<ProviderHeader<Self::Provider>>,
            >,
            Error: FromEvmError<Self::Evm>,
            RpcConvert: RpcConvert<Network = Self::NetworkTypes>,
        >,
    N: RpcNodeCore<
        Provider: BlockReaderIdExt + ChainSpecProvider + StateProviderFactory,
        Pool: TransactionPool<Transaction: PoolTransaction<Consensus = ProviderTx<N::Provider>>>,
        Evm: ConfigureEvm<Primitives = Self::Primitives>,
        Primitives: NodePrimitives<
            BlockHeader = ProviderHeader<Self::Provider>,
            SignedTx = ProviderTx<Self::Provider>,
            Receipt = ProviderReceipt<Self::Provider>,
            Block = ProviderBlock<Self::Provider>,
        >,
    >,
    Rpc: RpcConvert,
{
    #[inline]
    fn pending_block(
        &self,
    ) -> &tokio::sync::Mutex<
        Option<PendingBlock<ProviderBlock<Self::Provider>, ProviderReceipt<Self::Provider>>>,
    > {
        self.inner.eth_api.pending_block()
    }

    #[inline]
    fn pending_env_builder(&self) -> &dyn PendingEnvBuilder<Self::Evm> {
        self.inner.eth_api.pending_env_builder()
    }

    /// Returns the locally built pending block
    async fn local_pending_block(
        &self,
    ) -> Result<
        Option<(
            Arc<RecoveredBlock<ProviderBlock<Self::Provider>>>,
            Arc<Vec<ProviderReceipt<Self::Provider>>>,
        )>,
        Self::Error,
    > {
        // See: <https://github.com/ethereum-optimism/op-geth/blob/f2e69450c6eec9c35d56af91389a1c47737206ca/miner/worker.go#L367-L375>
        let latest = self
            .provider()
            .latest_header()
            .map_err(Self::Error::from_eth_err)?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let block_id = latest.hash().into();
        let block = self
            .provider()
            .recovered_block(block_id, Default::default())
            .map_err(Self::Error::from_eth_err)?
            .ok_or(EthApiError::HeaderNotFound(block_id.into()))?;

        let receipts = self
            .provider()
            .receipts_by_block(block_id)
            .map_err(Self::Error::from_eth_err)?
            .ok_or(EthApiError::ReceiptsNotFound(block_id.into()))?;

        Ok(Some((Arc::new(block), Arc::new(receipts))))
    }
}
