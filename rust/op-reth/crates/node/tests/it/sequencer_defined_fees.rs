//! Pending RPC and txpool behavior with a sequencer-defined Lagoon base fee.

use alloy_genesis::Genesis;
use alloy_primitives::{Address, U256, address};
use alloy_rpc_types_eth::FeeHistory;
use jsonrpsee::{core::client::ClientT, rpc_params};
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_optimism_payload_builder::config::{
    BaseFeePolicy, BaseFeePolicyError, BaseFeePolicyInput,
};
use reth_rpc_eth_api::helpers::LoadPendingBlock;
use reth_tasks::Runtime;
use reth_transaction_pool::TransactionPool;
use revm::context::Block;
use serde_json::json;
use std::sync::Arc;

const SELECTED_BASE_FEE: u64 = 777;
const SUGGESTED_PRIORITY_FEE: u64 = 1_000_000;
const FUNDED: Address = address!("f39fd6e51aad88f6f4ce6ab8827279cfffb92266");

#[derive(Debug)]
struct FixedBaseFeePolicy;

impl BaseFeePolicy for FixedBaseFeePolicy {
    fn select_base_fee(&self, _input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError> {
        Ok(SELECTED_BASE_FEE)
    }
}

#[derive(Debug)]
struct SimulationBaseFeePolicy;

impl BaseFeePolicy for SimulationBaseFeePolicy {
    fn select_base_fee(&self, _input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError> {
        Ok(SELECTED_BASE_FEE)
    }

    fn quote_base_fee(&self, input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError> {
        if input.next_timestamp == 100 {
            Ok(input.next_timestamp)
        } else {
            Err(BaseFeePolicyError::msg(format!(
                "unexpected simulation timestamp {}",
                input.next_timestamp
            )))
        }
    }
}

#[tokio::test]
async fn pending_rpc_and_txpool_use_selected_base_fee() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    // Keep the latest observed fee distinct from the selected pending fee so every assertion proves
    // that the policy hook was used rather than the canonical header value.
    genesis.base_fee_per_gas = Some(1_000_000_000);
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build(),
    );

    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config = NodeConfig::test()
        .map_chain(chain_spec)
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());
    let op_node = OpNode::default().with_base_fee_policy(Arc::new(FixedBaseFeePolicy));

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(Runtime::test()).node(op_node).launch().await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    assert_eq!(node.pool.block_info().pending_basefee, SELECTED_BASE_FEE);
    assert_eq!(
        node.add_ons_handle.eth_api().pending_block_env_and_cfg()?.evm_env.block_env.basefee(),
        SELECTED_BASE_FEE,
    );

    let base_fee: Option<U256> = client.request("eth_baseFee", rpc_params![]).await?;
    assert_eq!(base_fee, Some(U256::from(SELECTED_BASE_FEE)));

    let gas_price: U256 = client.request("eth_gasPrice", rpc_params![]).await?;
    assert_eq!(gas_price, U256::from(SELECTED_BASE_FEE + SUGGESTED_PRIORITY_FEE));

    let history: FeeHistory =
        client.request("eth_feeHistory", rpc_params![1, "latest", Vec::<f64>::new()]).await?;
    assert_eq!(history.base_fee_per_gas.last(), Some(&(SELECTED_BASE_FEE as u128)));

    let filled: serde_json::Value = client
        .request(
            "eth_fillTransaction",
            rpc_params![json!({
                "from": FUNDED,
                "to": Address::with_last_byte(1),
                "gas": "0x5208"
            })],
        )
        .await?;
    assert_eq!(
        filled["tx"]["maxFeePerGas"],
        json!(format!("0x{:x}", SELECTED_BASE_FEE * 2 + SUGGESTED_PRIORITY_FEE)),
    );

    // Pending estimation must remain usable when callers price against the selected fee.
    let estimated: U256 = client
        .request(
            "eth_estimateGas",
            rpc_params![
                json!({
                    "from": FUNDED,
                    "to": Address::with_last_byte(1),
                    "maxFeePerGas": format!("0x{SELECTED_BASE_FEE:x}"),
                    "maxPriorityFeePerGas": "0x0"
                }),
                "pending"
            ],
        )
        .await?;
    assert!(estimated >= U256::from(21_000));

    Ok(())
}

#[tokio::test]
async fn simulation_quotes_effective_timestamp_only_when_needed() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build(),
    );

    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config = NodeConfig::test()
        .map_chain(chain_spec)
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());
    let op_node = OpNode::default().with_base_fee_policy(Arc::new(SimulationBaseFeePolicy));

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(Runtime::test()).node(op_node).launch().await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    // Validation-disabled simulations do not need a sequencer fee quote.
    let without_validation: Vec<serde_json::Value> = client
        .request(
            "eth_simulateV1",
            rpc_params![json!({
                "blockStateCalls": [{ "calls": [] }],
                "validation": false
            })],
        )
        .await?;
    assert_eq!(without_validation.len(), 1);

    // An explicit base fee also makes the policy irrelevant, even with validation enabled.
    let explicit_fee: Vec<serde_json::Value> = client
        .request(
            "eth_simulateV1",
            rpc_params![json!({
                "blockStateCalls": [{
                    "blockOverrides": {
                        "time": "0x3",
                        "baseFeePerGas": "0x2a"
                    },
                    "calls": []
                }],
                "validation": true
            })],
        )
        .await?;
    assert_eq!(explicit_fee[0]["baseFeePerGas"], json!("0x2a"));

    // Without an explicit fee, the quote observes the sanitized simulation timestamp.
    let quoted_fee: Vec<serde_json::Value> = client
        .request(
            "eth_simulateV1",
            rpc_params![json!({
                "blockStateCalls": [{
                    "blockOverrides": { "time": "0x64" },
                    "calls": []
                }],
                "validation": true
            })],
        )
        .await?;
    assert_eq!(quoted_fee[0]["baseFeePerGas"], json!("0x64"));

    Ok(())
}
