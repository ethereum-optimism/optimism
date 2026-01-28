//! Loads and formats OP block RPC response.

use crate::{eth::RpcNodeCore, OpEthApi, OpEthApiError};
use reth_rpc_eth_api::{
    helpers::{EthBlocks, LoadBlock},
    FromEvmError, RpcConvert,
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
