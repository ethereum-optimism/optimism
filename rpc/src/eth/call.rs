use super::OpNodeCore;
use crate::{OpEthApi, OpEthApiError};
use alloy_rpc_types_eth::TransactionRequest;
use op_revm::OpTransaction;
use reth_evm::{execute::BlockExecutorFactory, ConfigureEvm, EvmFactory, TxEnvFor};
use reth_node_api::NodePrimitives;
use reth_rpc_eth_api::{
    helpers::{estimate::EstimateCall, Call, EthCall, LoadBlock, LoadState, SpawnBlocking},
    FromEvmError, FullEthApiTypes, RpcConvert, RpcTypes,
};
use reth_storage_api::{errors::ProviderError, ProviderHeader, ProviderTx};
use revm::context::TxEnv;

impl<N> EthCall for OpEthApi<N>
where
    Self: EstimateCall + LoadBlock + FullEthApiTypes,
    N: OpNodeCore,
{
}

impl<N> EstimateCall for OpEthApi<N>
where
    Self: Call,
    Self::Error: From<OpEthApiError>,
    N: OpNodeCore,
{
}

impl<N> Call for OpEthApi<N>
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
            NetworkTypes: RpcTypes<TransactionRequest: From<TransactionRequest>>,
            Error: FromEvmError<Self::Evm>
                       + From<<Self::RpcConvert as RpcConvert>::Error>
                       + From<ProviderError>,
        > + SpawnBlocking,
    Self::Error: From<OpEthApiError>,
    N: OpNodeCore,
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
