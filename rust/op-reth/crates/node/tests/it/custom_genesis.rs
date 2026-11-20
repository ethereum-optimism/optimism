//! Tests for custom genesis block number support.

use crate::launcher::launch;
use alloy_consensus::{BlockHeader, Sealable};
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{
    node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_provider::{HeaderProvider, StageCheckpointReader};
use reth_stages_types::StageId;
use std::sync::Arc;
use tempfile::TempDir;
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

    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );

    let wallet = Arc::new(Mutex::new(Wallet::default().with_chain_id(chain_spec.chain().into())));

    // Configure and launch the node
    let config = NodeConfig::new(chain_spec.clone());
    let datadir = TempDir::new().unwrap();
    let node_handle = launch(config, reth_tasks::Runtime::test(), &datadir, OpNode::default())
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

    // Initialize the forkchoice state with the genesis block hash so that
    // `current_forkchoice_state()` can resolve the latest header. With storage_v2,
    // the canonical chain tip must be explicitly set via a forkchoice update.
    let genesis_hash = genesis_header.unwrap().seal_slow().hash();
    node.update_forkchoice(genesis_hash, genesis_hash).await.unwrap();

    // Advance the chain with a single block.
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
        "Block number should be 1001 after advancing from genesis 1000"
    );
}
