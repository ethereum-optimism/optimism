use super::OpNodeCore;
use crate::{OpEthApi, OpEthApiError};
use op_revm::OpTransaction;
use reth_evm::{execute::BlockExecutorFactory, ConfigureEvm, EvmFactory, TxEnvFor};
use reth_node_api::NodePrimitives;
use reth_rpc_eth_api::{
    helpers::{estimate::EstimateCall, Call, EthCall, LoadBlock, LoadState, SpawnBlocking},
    FromEvmError, FullEthApiTypes, RpcConvert,
};
use reth_storage_api::{errors::ProviderError, ProviderHeader, ProviderTx};
use revm::context::TxEnv;

impl<N, Rpc> EthCall for OpEthApi<N, Rpc>
where
    Self: EstimateCall + LoadBlock + FullEthApiTypes,
    N: OpNodeCore,
    Rpc: RpcConvert,
{
}

impl<N, Rpc> EstimateCall for OpEthApi<N, Rpc>
where
    Self: Call<Error: From<OpEthApiError>>,
    N: OpNodeCore,
    Rpc: RpcConvert,
{
}

impl<N, Rpc> Call for OpEthApi<N, Rpc>
where
    Self: LoadState<
            Evm: ConfigureEvm<
                Primitives: NodePrimitives<
                    BlockHeader = ProviderHeader<Self::Provider>,
                    SignedTx = ProviderTx<Self::Provider>,
                >,
                BlockExecutorFactory: BlockExecutorFactory<
                    EvmFactory: EvmFactory<Tx = OpTransaction<TxEnv>>,
                >,
            >,
            RpcConvert: RpcConvert<TxEnv = TxEnvFor<Self::Evm>, Network = Self::NetworkTypes>,
            Error: FromEvmError<Self::Evm>
                       + From<<Self::RpcConvert as RpcConvert>::Error>
                       + From<ProviderError>,
        > + SpawnBlocking,
    Self::Error: From<OpEthApiError>,
    N: OpNodeCore,
    Rpc: RpcConvert,
{
    #[inline]
    fn call_gas_limit(&self) -> u64 {
        self.inner.eth_api.gas_cap()
    }

    #[inline]
    fn max_simulate_blocks(&self) -> u64 {
        self.inner.eth_api.max_simulate_blocks()
    }
}
