//! `eth_estimateGas` integration tests for EIP-7825.
//!
//! During the U19 upgrade process, several subtle [configuration footguns] and
//! [upstream issues] resulted in unfortunate user-facing estimateGas bugs.
//!
//! This test ensures that EIP-7825 is handled correctly by `eth_estimateGas`.
//!
//! [configuration footguns]: https://github.com/ethereum-optimism/optimism/pull/21337
//! [upstream issues]: https://github.com/paradigmxyz/reth/pull/25612

use alloy_primitives::{Address, Bytes, TxKind, U256};
use alloy_rpc_types_eth::{
    Block, Header, Receipt, Transaction, TransactionInput, TransactionRequest,
};
use op_alloy_consensus::OpTxEnvelope;
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_rpc_api::clients::EthApiClient;
use reth_tasks::Runtime;
use revm::primitives::eip7825::TX_GAS_LIMIT_CAP;

/// Call `eth_estimateGas` over the given RPC client. Generic over the client so the (unnameable)
/// jsonrpsee `HttpClient` type does not have to be spelled out; only the request type matters for
/// this method.
async fn estimate_gas<C>(client: &C, request: TransactionRequest) -> eyre::Result<U256>
where
    C: EthApiClient<TransactionRequest, Transaction, Block, Receipt, Header, OpTxEnvelope> + Sync,
{
    Ok(EthApiClient::estimate_gas(client, request, None, None, None).await?)
}

/// `eth_estimateGas` for a contract call with no explicit `gas` must succeed on a Karst/Osaka
/// chain. Before the fix this returned `-32000: intrinsic gas too high` because the trial gas limit
/// fell back to the 30M block gas limit, which exceeds the 2^24 per-tx cap.
#[tokio::test]
async fn test_estimate_gas_under_karst() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config = NodeConfig::test()
        .map_chain(OpChainSpecBuilder::optimism_sepolia().karst_activated().build())
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    // The regression: a call with non-empty calldata (-> not a basic transfer) and no `gas` field.
    // The callee needs no code; the calldata alone is what triggers the overshoot.
    let request = TransactionRequest {
        from: Some(Address::default()),
        to: Some(TxKind::Call(Address::default())),
        input: TransactionInput::new(Bytes::from(&[0])),
        ..Default::default()
    };
    let gas = estimate_gas(&client, request).await?;
    assert!(gas > U256::ZERO, "contract-call estimate must succeed under Karst, got {gas}");
    // The trial upper bound is now the 2^24 cap, so the estimate cannot exceed it.
    assert!(
        gas <= U256::from(TX_GAS_LIMIT_CAP),
        "estimate {gas} must be within the EIP-7825 per-tx cap",
    );

    Ok(())
}
