//! Tests for custom genesis block number support.

use alloy_consensus::BlockHeader;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use reth_chainspec::EthChainSpec;
use reth_db::test_utils::create_test_rw_db_with_path;
use reth_e2e_test_utils::{
    node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_builder::{EngineNodeLauncher, Node, NodeBuilder, NodeConfig};
use reth_node_core::args::DatadirArgs;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_provider::{HeaderProvider, StageCheckpointReader, providers::BlockchainProvider};
use reth_stages_types::StageId;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Tests that an OP node can initialize with a custom genesis block number.
#[tokio::test]
async fn test_op_node_custom_genesis_number() {
    reth_tracing::init_test_tracing();

    let genesis_number = 1000;

    // Create genesis with custom block number (1000)
    let mut genesis: Genesis =
        serde_json::from_str(include_str!("../assets/genesis.json")).unwrap();
    genesis.number = Some(genesis_number);
    genesis.parent_hash = Some(B256::random());

    let chain_spec =
        Arc::new(OpChainSpecBuilder::base_mainnet().genesis(genesis).ecotone_activated().build());

    let wallet = Arc::new(Mutex::new(Wallet::default().with_chain_id(chain_spec.chain().into())));

    // Configure and launch the node
    let mut config =
        NodeConfig::new(chain_spec.clone()).with_unused_ports().with_datadir_args(DatadirArgs {
            datadir: reth_db::test_utils::tempdir_path().into(),
            ..Default::default()
        });
    config.network.discovery.discv5_port = 0;
    config.network.discovery.discv5_port_ipv6 = 0;
    let db = create_test_rw_db_with_path(
        config
            .datadir
            .datadir
            .unwrap_or_chain_default(config.chain.chain(), config.datadir.clone())
            .db(),
    );
    let tasks = reth_tasks::TaskManager::current();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
        .with_components(OpNode::default().components())
        .with_add_ons(OpNode::new(Default::default()).add_ons())
        .launch_with_fn(|builder| {
            let launcher = EngineNodeLauncher::new(
                tasks.executor(),
                builder.config.datadir(),
                Default::default(),
            );
            builder.launch_with(launcher)
        })
        .await
        .expect("Failed to launch node");

    let mut node =
        NodeTestContext::new(node_handle.node, optimism_payload_attributes).await.unwrap();

    // Verify stage checkpoints are initialized to genesis block number (1000)
    for stage in StageId::ALL {
        let checkpoint = node.inner.provider.get_stage_checkpoint(stage).unwrap();
        assert!(checkpoint.is_some(), "Stage {stage:?} checkpoint should exist");
        assert_eq!(
            checkpoint.unwrap().block_number,
            1000,
            "Stage {stage:?} checkpoint should be at genesis block 1000"
        );
    }

    // Query genesis block should succeed
    let genesis_header = node.inner.provider.header_by_number(genesis_number).unwrap();
    assert!(genesis_header.is_some(), "Genesis block at {genesis_number} should exist");

    // Query blocks before genesis should return None
    for block_num in [0, 1, genesis_number - 1] {
        let header = node.inner.provider.header_by_number(block_num).unwrap();
        assert!(header.is_none(), "Block {block_num} before genesis should not exist");
    }

    // Advance the chain with a single block
    let _ = wallet; // wallet available for future use
    let block_payloads = node
        .advance(1, |_| {
            Box::pin({
                let value = wallet.clone();
                async move {
                    let mut wallet = value.lock().await;
                    let tx_fut = TransactionTestContext::optimism_l1_block_info_tx(
                        wallet.chain_id,
                        wallet.inner.clone(),
                        wallet.inner_nonce,
                    );
                    wallet.inner_nonce += 1;

                    tx_fut.await
                }
            })
        })
        .await
        .unwrap();

    assert_eq!(block_payloads.len(), 1);
    let block = block_payloads.first().unwrap().block();

    // Verify the new block is at 1001 (genesis 1000 + 1)
    assert_eq!(
        block.number(),
        1001,
        "Block number should be 1001 after advancing from genesis 100"
    );
}
