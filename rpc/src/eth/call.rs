use super::OpNodeCore;
use crate::{OpEthApi, OpEthApiError};
use op_revm::OpTransaction;
use reth_evm::{execute::BlockExecutorFactory, ConfigureEvm, EvmFactory, TxEnvFor};
use reth_node_api::NodePrimitives;
use reth_rpc_eth_api::{
    helpers::{estimate::EstimateCall, Call, EthCall, LoadBlock, LoadState, SpawnBlocking},
    FromEvmError, FullEthApiTypes, TransactionCompat,
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
            TransactionCompat: TransactionCompat<TxEnv = TxEnvFor<Self::Evm>>,
            Error: FromEvmError<Self::Evm>
                       + From<<Self::TransactionCompat as TransactionCompat>::Error>
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
