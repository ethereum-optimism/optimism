//! `eth_estimateGas` integration tests.
//!
//! Regression coverage for the Karst/Osaka `eth_estimateGas` bug. EIP-7825 (Osaka) caps a single
//! transaction's gas at 2^24, and the OP Stack inherits that cap from Karst onward. op-reth's
//! EVM-env builder did not populate `cfg_env.tx_gas_limit_cap`, so reth's `estimate.rs` fell back
//! to the *block* gas limit (here 30M, from `assets/genesis.json`) as the trial upper bound. The
//! Osaka EVM then rejected that trial ("intrinsic gas too high"), so any contract-call estimate
//! that omitted an explicit `gas` failed. The fix sets the cap in `alloy-op-evm`'s
//! `evm_env_for_op`; this test drives a real op-reth RPC end-to-end to prove the estimate now
//! succeeds.

use std::sync::Arc;

use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, Bytes, TxKind, U256, address};
use alloy_rpc_types_eth::{
    Block, Header, Receipt, Transaction, TransactionInput, TransactionRequest,
};
use op_alloy_consensus::OpTxEnvelope;
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::OpNode;
use reth_rpc_api::clients::EthApiClient;
use reth_tasks::Runtime;

/// EIP-7825 caps a single transaction's gas at 2^24.
const TX_GAS_LIMIT_CAP: u64 = 1 << 24;

/// A pre-funded EOA from `assets/genesis.json` (the well-known dev account #0).
const FUNDED: Address = address!("f39fd6e51aad88f6f4ce6ab8827279cfffb92266");

/// Address at which we deploy a minimal contract so the estimate is a genuine contract call
/// (`is_basic_transfer == false` in reth's `estimate.rs`), which is the path that overshoots to the
/// block gas limit. The runtime code is a single `STOP` (0x00): it has code (so the call is not a
/// basic transfer) and returns successfully for any input.
const CONTRACT: Address = Address::repeat_byte(0xc0);

/// `decimals()` selector — a typical no-argument view call, the shape that triggered the bug.
const DECIMALS_SELECTOR: &[u8] = &[0x31, 0x3c, 0xe5, 0x67];

/// Build a chain spec from the shared test genesis with Karst (and thus the Osaka EVM base) active
/// at genesis, and a minimal contract deployed at [`CONTRACT`].
fn karst_chain_spec() -> Arc<OpChainSpec> {
    let mut genesis: Genesis =
        serde_json::from_str(include_str!("../assets/genesis.json")).unwrap();
    genesis.alloc.insert(
        CONTRACT,
        GenesisAccount { code: Some(Bytes::from_static(&[0x00])), ..Default::default() },
    );
    Arc::new(OpChainSpecBuilder::optimism_sepolia().genesis(genesis).karst_activated().build())
}

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
        .map_chain(karst_chain_spec())
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    // The regression: a contract call (non-empty input -> not a basic transfer) with no `gas`
    // field.
    let contract_call = TransactionRequest {
        from: Some(FUNDED),
        to: Some(TxKind::Call(CONTRACT)),
        input: TransactionInput::new(Bytes::from_static(DECIMALS_SELECTOR)),
        ..Default::default()
    };
    let gas = estimate_gas(&client, contract_call).await?;
    assert!(gas > U256::ZERO, "contract-call estimate must succeed under Karst, got {gas}");
    // The trial upper bound is now the 2^24 cap, so the estimate cannot exceed it.
    assert!(
        gas <= U256::from(TX_GAS_LIMIT_CAP),
        "estimate {gas} must be within the EIP-7825 per-tx cap",
    );

    Ok(())
}
