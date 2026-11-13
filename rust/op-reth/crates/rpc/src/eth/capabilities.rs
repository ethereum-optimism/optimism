//! `eth_capabilities` endpoint that accounts for the Bedrock boundary.

use alloy_primitives::U64;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use reth_rpc_eth_api::helpers::{EthApiSpec, EthState};
use reth_rpc_eth_types::EthCapabilities;
use reth_rpc_server_types::ToRpcResult;

/// `eth_capabilities` endpoint.
#[rpc(server, namespace = "eth")]
pub trait OpEthCapabilitiesApi {
    /// Returns the effective routing capabilities of this node.
    #[method(name = "capabilities")]
    fn capabilities(&self) -> RpcResult<EthCapabilities>;
}

/// Serves `eth_capabilities` with no resource available below the Bedrock block.
///
/// The node never holds pre-Bedrock history: a datadir initialized from the Bedrock state dump
/// has only placeholder headers below it. Pre-Bedrock requests can at most be forwarded to
/// `--rollup.historicalrpc`, which is not this node's capability to advertise.
#[derive(Debug, Clone)]
pub struct OpEthCapabilities<EthApi> {
    eth_api: EthApi,
    bedrock_block: u64,
}

impl<EthApi> OpEthCapabilities<EthApi> {
    /// Creates a new [`OpEthCapabilities`] wrapping `eth_api`.
    pub const fn new(eth_api: EthApi, bedrock_block: u64) -> Self {
        Self { eth_api, bedrock_block }
    }
}

impl<EthApi> OpEthCapabilitiesApiServer for OpEthCapabilities<EthApi>
where
    EthApi: EthApiSpec + EthState,
{
    fn capabilities(&self) -> RpcResult<EthCapabilities> {
        let capabilities = EthApiSpec::capabilities(&self.eth_api).to_rpc_result()?;
        Ok(raise_oldest_block(capabilities, self.bedrock_block))
    }
}

/// Raises the oldest available block of every enabled resource to at least `floor`.
fn raise_oldest_block(mut capabilities: EthCapabilities, floor: u64) -> EthCapabilities {
    let floor = U64::from(floor);
    for resource in [
        &mut capabilities.state,
        &mut capabilities.tx,
        &mut capabilities.logs,
        &mut capabilities.receipts,
        &mut capabilities.blocks,
        &mut capabilities.state_proofs,
    ] {
        if let Some(oldest_block) = &mut resource.oldest_block {
            *oldest_block = (*oldest_block).max(floor);
        }
    }
    capabilities
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use reth_rpc_eth_types::{EthCapabilitiesHead, EthCapabilitiesResource};

    const BEDROCK: u64 = 105_235_063;

    #[test]
    fn raises_oldest_block_to_bedrock() {
        let capabilities = EthCapabilities {
            head: EthCapabilitiesHead { number: BEDROCK + 1_000, hash: B256::ZERO },
            state: EthCapabilitiesResource::available_from(0),
            tx: EthCapabilitiesResource::available_from(BEDROCK - 1),
            logs: EthCapabilitiesResource::window(0, 10_064),
            receipts: EthCapabilitiesResource::available_from(BEDROCK + 500),
            blocks: EthCapabilitiesResource::disabled(),
            state_proofs: EthCapabilitiesResource::window(BEDROCK, 0),
        };

        let raised = raise_oldest_block(capabilities, BEDROCK);

        assert_eq!(raised.head.number, BEDROCK + 1_000);
        assert_eq!(raised.state, EthCapabilitiesResource::available_from(BEDROCK));
        assert_eq!(raised.tx, EthCapabilitiesResource::available_from(BEDROCK));
        assert_eq!(raised.logs, EthCapabilitiesResource::window(BEDROCK, 10_064));
        assert_eq!(raised.receipts, EthCapabilitiesResource::available_from(BEDROCK + 500));
        assert_eq!(raised.blocks, EthCapabilitiesResource::disabled());
        assert_eq!(raised.state_proofs, EthCapabilitiesResource::window(BEDROCK, 0));
    }
}
