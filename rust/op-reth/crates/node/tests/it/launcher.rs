//! Launches nodes for the tests.
//!
//! [`launch`] launches op-reth's own node with the launch the `op-reth` binary compiles in
//! `reth_optimism_node_launcher`. [`setup`] launches connected nodes through reth's E2E harness
//! instead, which this crate compiles.

use reth_db::{DatabaseEnv, init_db, mdbx::DatabaseArguments};
use reth_e2e_test_utils::{
    E2ETestSetupBuilder, NodeHelperType, node::NodeTestContext, wallet::Wallet,
};
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{args::DatadirArgs, node_config::NodeConfig};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::{
    OpLaunchHooks, OpNode,
    utils::{chain_spec, optimism_payload_attributes},
};
use reth_optimism_node_launcher::{OpRethAdapter, OpRethAddOns, OpRethNode, OpRethNodeHandle};
use reth_tasks::Runtime;
use tempfile::TempDir;

/// Launches `node` with `config` on `runtime`, storing its data in `datadir`.
///
/// Like reth's `testing_node`, except that the caller owns the data directory: keep `datadir` until
/// the node is no longer used.
pub(crate) async fn launch(
    config: NodeConfig<OpChainSpec>,
    runtime: Runtime,
    datadir: &TempDir,
    node: OpNode,
) -> eyre::Result<OpRethNodeHandle> {
    let builder = builder(config, runtime, datadir)?.node(OpRethNode::from(node));
    reth_optimism_node_launcher::launch(builder).await
}

/// A node builder with `config` and a database in `datadir`.
fn builder(
    config: NodeConfig<OpChainSpec>,
    runtime: Runtime,
    datadir: &TempDir,
) -> eyre::Result<WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>> {
    let config = config.with_datadir_args(DatadirArgs {
        datadir: datadir.path().to_path_buf().into(),
        ..Default::default()
    });
    let db = init_db(config.datadir().db(), DatabaseArguments::test())?;
    Ok(NodeBuilder::new(config).with_database(db).with_launch_context(runtime))
}

/// The test context of a node [`launch`] launched.
pub(crate) type TestContext = NodeTestContext<OpRethAdapter, OpRethAddOns>;

/// Launches `num_nodes` connected nodes on [`chain_spec`] through reth's E2E harness.
pub(crate) async fn setup(num_nodes: usize) -> eyre::Result<(Vec<NodeHelperType<OpNode>>, Wallet)> {
    E2ETestSetupBuilder::new(num_nodes, chain_spec())
        .with_attributes_generator(optimism_payload_attributes)
        .build()
        .await
}

/// The launch the `op-reth` binary uses refuses state masking before it starts the node.
#[tokio::test]
async fn launch_node_rejects_state_masking() -> eyre::Result<()> {
    let datadir = TempDir::new()?;
    let mut config = NodeConfig::test().map_chain(chain_spec());
    config.engine.num_state_masking_blocks = 1;
    let builder = builder(config, Runtime::test(), &datadir)?;

    let err = reth_optimism_node_launcher::launch_node(
        builder,
        OpNode::default(),
        OpLaunchHooks::default(),
    )
    .await
    .expect_err("state masking is not supported");
    assert!(
        err.to_string().contains("--engine.num-state-masking-blocks must be 0"),
        "unexpected error: {err:#}"
    );
    Ok(())
}
