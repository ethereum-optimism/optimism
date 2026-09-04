//! Loads and formats OP block RPC response.

use crate::{
    OpEthApi, OpEthApiError,
    eth::{OpRpcProvider, RpcNodeCore},
};
use reth_optimism_evm::ConfigurePostExecEvm;
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert,
    helpers::{EthBlocks, LoadBlock},
};

impl<N, Rpc> EthBlocks for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    N::Provider: OpRpcProvider,
    N::Evm: ConfigurePostExecEvm,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}

impl<N, Rpc> LoadBlock for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    N::Provider: OpRpcProvider,
    N::Evm: ConfigurePostExecEvm,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}
