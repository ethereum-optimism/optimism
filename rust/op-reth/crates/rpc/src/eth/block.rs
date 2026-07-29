//! Loads and formats OP block RPC response.

use crate::{OpEthApi, OpEthApiError, eth::RpcNodeCore};
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert,
    helpers::{EthBlocks, LoadBlock},
};

impl<N, Rpc> EthBlocks for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}

impl<N, Rpc> LoadBlock for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}
