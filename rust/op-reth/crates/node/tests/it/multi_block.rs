//! Sibling blocks: two consecutive blocks sharing one timestamp, driven over the engine API.

use alloy_consensus::BlockHeader;
use alloy_genesis::Genesis;
use alloy_rpc_types_engine::ForkchoiceUpdateError;
use reth_db::test_utils::create_test_rw_db_with_path;
use reth_e2e_test_utils::node::NodeTestContext;
use reth_node_api::BeaconForkChoiceUpdateError;
use reth_node_builder::{EngineNodeLauncher, Node, NodeBuilder, NodeConfig};
use reth_node_core::args::{DatadirArgs, NetworkArgs};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_provider::providers::BlockchainProvider;

/// The payload test context starts issuing attributes one second after this, so every block the
/// test builds is past the activation.
const MULTI_BLOCK_TIME: u64 = 1710338135;

/// Builds a block, then asks for a second one carrying the same timestamp.
///
/// With multi-block configured the sibling must become canonical; without it the engine must
/// refuse the payload attributes, which is what keeps a chain that has not opted in from ever
/// producing siblings.
async fn build_sibling(multi_block: bool) -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let mut builder = OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated();
    if multi_block {
        builder = builder.multi_block_at(MULTI_BLOCK_TIME);
    }
    let chain_spec = builder.build();

    // Ephemeral ports: the test suite launches several nodes at once.
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let config = NodeConfig::test()
        .map_chain(chain_spec)
        .with_network(network_args)
        .with_datadir_args(DatadirArgs {
            datadir: reth_db::test_utils::tempdir_path().into(),
            ..Default::default()
        });
    let db = create_test_rw_db_with_path(
        config
            .datadir
            .datadir
            .unwrap_or_chain_default(config.chain.chain(), config.datadir.clone())
            .db(),
    );
    let runtime = reth_tasks::Runtime::test();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
        .with_components(OpNode::default().components())
        .with_add_ons(OpNode::default().add_ons())
        .launch_with_fn(|builder| {
            let launcher =
                EngineNodeLauncher::new(runtime.clone(), builder.config.datadir(), <_>::default());
            builder.launch_with(launcher)
        })
        .await?;

    let mut node = NodeTestContext::new(node_handle.node, optimism_payload_attributes).await?;
    let first = node.advance_block().await?;
    assert_eq!(first.block().timestamp(), MULTI_BLOCK_TIME + 1);

    // Hand the attributes generator the timestamp it just used, so the next block is a sibling.
    node.payload.timestamp -= 1;

    if !multi_block {
        let attributes = node.payload.next_attributes();
        let result = node
            .inner
            .add_ons_handle
            .beacon_engine_handle
            .fork_choice_updated(node.current_forkchoice_state()?, Some(attributes))
            .await;
        assert!(
            matches!(
                result,
                Err(BeaconForkChoiceUpdateError::ForkchoiceUpdateError(
                    ForkchoiceUpdateError::UpdatedInvalidPayloadAttributes
                ))
            ),
            "expected the sibling's attributes to be rejected, got {result:?}"
        );
        return Ok(());
    }

    let sibling = node.advance_block().await?;
    assert_eq!(sibling.block().timestamp(), first.block().timestamp());
    assert_eq!(sibling.block().number(), first.block().number() + 1);
    assert_eq!(sibling.block().parent_hash(), first.block().hash());

    // The sibling is the canonical head, not a competing branch.
    let head = node.current_forkchoice_state()?.head_block_hash;
    assert_eq!(head, sibling.block().hash());

    Ok(())
}

#[tokio::test]
async fn sibling_block_is_canonicalised_after_multi_block_activation() -> eyre::Result<()> {
    build_sibling(true).await
}

#[tokio::test]
async fn sibling_block_is_rejected_without_multi_block() -> eyre::Result<()> {
    build_sibling(false).await
}
