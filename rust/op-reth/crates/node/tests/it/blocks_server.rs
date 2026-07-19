//! Blocks server integration tests.

use alloy_genesis::Genesis;
use futures::StreamExt;
use op_alloy_rpc_types_engine::decode_block_frame;
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{
    node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_builder::{NodeBuilder, NodeConfig};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, args::RollupArgs, utils::optimism_payload_attributes};
use reth_tasks::Runtime;
use std::{net::TcpListener, sync::Arc, time::Duration};
use tokio::time::timeout;
use tokio_tungstenite::{connect_async, tungstenite::Message};

#[tokio::test]
async fn streams_block_containing_submitted_transaction() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );

    // Reserve an unused port for the blocks server. The listener is dropped before op-reth binds
    // it during launch.
    let listener = TcpListener::bind("127.0.0.1:0")?;
    let blocks_addr = listener.local_addr()?;
    drop(listener);

    let mut rollup_args = RollupArgs::default();
    rollup_args.blocks_server.enabled = true;
    rollup_args.blocks_server.addr = blocks_addr;

    let node_handle = NodeBuilder::new(NodeConfig::new(chain_spec.clone()).with_unused_ports())
        .testing_node(Runtime::test())
        .node(OpNode::new(rollup_args))
        .launch()
        .await?;
    let mut node = NodeTestContext::new(node_handle.node, optimism_payload_attributes).await?;

    let genesis_hash = node.block_hash(0);
    node.update_forkchoice(genesis_hash, genesis_hash).await?;

    // Start at the next block so the first WebSocket frame is the block built below.
    let (mut blocks, _) = connect_async(format!("ws://{blocks_addr}/blocks?start=1")).await?;

    let wallet = Wallet::default().with_chain_id(chain_spec.chain().into());
    let l1_info =
        TransactionTestContext::optimism_l1_block_info_tx(wallet.chain_id, wallet.inner.clone(), 0)
            .await;
    node.rpc.inject_tx(l1_info).await?;

    let submitted = TransactionTestContext::transfer_tx_bytes_with_nonce(
        wallet.chain_id,
        wallet.inner.clone(),
        1,
    )
    .await;
    let submitted_hash = node.rpc.inject_tx(submitted.clone()).await?;

    let payload = node.advance_block().await?;

    let message = timeout(Duration::from_secs(10), blocks.next())
        .await
        .expect("blocks server should stream the new block")
        .expect("blocks connection should remain open")
        .expect("blocks server should send a valid WebSocket message");
    let Message::Binary(frame) = message else { panic!("expected binary block frame") };
    let streamed = decode_block_frame(&frame)?;

    assert_eq!(streamed.execution_payload.block_hash(), payload.block().hash());
    assert!(
        streamed.execution_payload.transactions().contains(&submitted),
        "streamed block should contain submitted transaction {submitted_hash}"
    );

    Ok(())
}
