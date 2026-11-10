//! The state masking guard on the launch path.

use reth_db::test_utils::create_test_rw_db_with_path;
use reth_node_api::TreeConfig;
use reth_node_builder::{EngineNodeLauncher, Node, NodeBuilder};
use reth_node_core::args::DatadirArgs;
use reth_optimism_chainspec::OP_SEPOLIA;
use reth_optimism_node::{OpNode, state_masking::STATE_MASKING_UNSUPPORTED, utils::node_config};
use reth_provider::providers::BlockchainProvider;
use reth_tasks::Runtime;

/// A launcher handed a masked tree config never brings the node up, whatever the node config
/// it was built from says.
#[tokio::test]
async fn launch_rejects_masked_tree_config() {
    reth_tracing::init_test_tracing();

    let mut config = node_config(OP_SEPOLIA.clone()).with_datadir_args(DatadirArgs {
        datadir: reth_db::test_utils::tempdir_path().into(),
        ..Default::default()
    });
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let db = create_test_rw_db_with_path(
        config
            .datadir
            .datadir
            .unwrap_or_chain_default(config.chain.chain(), config.datadir.clone())
            .db(),
    );
    let op_node = OpNode::default();
    let runtime = Runtime::test();
    let masked = TreeConfig::default().with_num_state_masking_blocks(30);

    let launched = NodeBuilder::new(config)
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch_with_fn(|builder| {
            let launcher =
                EngineNodeLauncher::new(runtime.clone(), builder.config.datadir(), masked);
            builder.launch_with(launcher)
        })
        .await;
    let Err(err) = launched else { panic!("a masked tree config must not launch") };

    assert!(
        err.chain().any(|cause| cause.to_string().contains(STATE_MASKING_UNSUPPORTED)),
        "{err:#}"
    );
}
