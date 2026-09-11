//! `debug_trace*` over blocks containing an SDM post-exec (0x7d) transaction.
//!
//! The RPC trace path replays each transaction directly through the EVM instead of going through
//! the block executor, so it only works on post-exec transactions because `OpEvm::transact_raw`
//! short-circuits them with the consensus-defined no-op result. Without that, tracing any block
//! that carries a 0x7d tx fails whole-block ([the bug]): the replayed post-exec tx env inherited
//! revm's mainnet default chain id, and no honest env for a zero-gas-limit transaction passes
//! validation either.
//!
//! [the bug]: https://github.com/ethereum-optimism/optimism-premium/issues/163

use alloy_consensus::{Block, BlockBody, Header, Sealable};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use alloy_rpc_types_eth::TransactionRequest;
use alloy_rpc_types_trace::{common::TraceResult, geth::GethDebugTracingOptions};
use op_alloy_consensus::{OpTxEnvelope, TxDeposit, build_post_exec_tx};
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_rpc_api::clients::DebugApiClient;
use reth_rpc_server_types::RpcModuleSelection;
use reth_tasks::Runtime;

/// `debug_traceBlock` over a raw block that ends in a post-exec (0x7d) transaction must return
/// one successful trace per transaction. Before the `transact_raw` short-circuit this failed for
/// the whole block with `-32000: invalid chain ID`.
#[tokio::test]
async fn test_debug_trace_block_with_post_exec_tx() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = OpChainSpecBuilder::optimism_sepolia().lagoon_activated().build();

    let exec = Runtime::test();
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config =
        NodeConfig::test().map_chain(chain_spec.clone()).with_network(network_args).with_rpc(
            RpcServerArgs::default()
                .with_unused_ports()
                .with_http()
                // `debug_*` is not part of the default HTTP module selection.
                .with_http_api(RpcModuleSelection::All),
        );

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    // `debug_traceBlock` replays a caller-supplied raw block on its parent's state, so the block
    // only has to decode and recover senders — consensus fields like the state root are never
    // validated. Replaying on genesis state keeps the fixture self-contained: no SDM production
    // machinery is needed to put a 0x7d tx in a block.
    let genesis = chain_spec.genesis_header();
    let header = Header {
        parent_hash: chain_spec.genesis_hash(),
        number: 1,
        timestamp: genesis.timestamp + 2,
        gas_limit: genesis.gas_limit,
        base_fee_per_gas: genesis.base_fee_per_gas,
        // Post-Ecotone replay runs the EIP-4788 pre-block system call, which requires a parent
        // beacon block root; the blob-gas and withdrawals fields must be present for the later
        // optionals to round-trip through RLP.
        withdrawals_root: Some(alloy_consensus::EMPTY_ROOT_HASH),
        blob_gas_used: Some(0),
        excess_blob_gas: Some(0),
        parent_beacon_block_root: Some(B256::ZERO),
        ..Default::default()
    };

    // A deposit needs no signature (senders recover from its `from` field), so the block gets a
    // normally-executing transaction without any signing machinery.
    let deposit = TxDeposit {
        source_hash: B256::with_last_byte(1),
        from: Address::with_last_byte(0xDE),
        to: TxKind::Call(Address::with_last_byte(1)),
        mint: 0,
        value: U256::ZERO,
        gas_limit: 21_000,
        is_system_transaction: false,
        input: Bytes::new(),
    };
    let transactions = vec![
        OpTxEnvelope::Deposit(deposit.seal_slow()),
        OpTxEnvelope::PostExec(build_post_exec_tx(1, 1, vec![]).seal_slow()),
    ];
    let block = Block::new(header, BlockBody { transactions, ommers: vec![], withdrawals: None });

    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block(
        &client,
        alloy_rlp::encode(&block).into(),
        Some(GethDebugTracingOptions::default()),
    )
    .await?;

    assert_eq!(traces.len(), 2, "one trace per transaction, including the post-exec tx");
    for (index, trace) in traces.iter().enumerate() {
        assert!(
            matches!(trace, TraceResult::Success { .. }),
            "transaction {index} must trace successfully, got {trace:?}",
        );
    }

    Ok(())
}
