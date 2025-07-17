//! Loads and formats OP block RPC response.

use reth_chainspec::ChainSpecProvider;
use reth_optimism_forks::OpHardforks;
use reth_rpc_eth_api::{
    helpers::{EthBlocks, LoadBlock, LoadPendingBlock, SpawnBlocking},
    RpcConvert,
};
use reth_storage_api::{HeaderProvider, ProviderTx};
use reth_transaction_pool::{PoolTransaction, TransactionPool};

use crate::{eth::OpNodeCore, OpEthApi, OpEthApiError};

impl<N, Rpc> EthBlocks for OpEthApi<N, Rpc>
where
    Self: LoadBlock<
        Error = OpEthApiError,
        RpcConvert: RpcConvert<Error = OpEthApiError, Primitives = Self::Primitives>,
    >,
    N: OpNodeCore<Provider: ChainSpecProvider<ChainSpec: OpHardforks> + HeaderProvider>,
    Rpc: RpcConvert,
{
}

impl<N, Rpc> LoadBlock for OpEthApi<N, Rpc>
where
    Self: LoadPendingBlock<
            Pool: TransactionPool<
                Transaction: PoolTransaction<Consensus = ProviderTx<Self::Provider>>,
            >,
        > + SpawnBlocking,
    N: OpNodeCore,
    Rpc: RpcConvert,
{
}
