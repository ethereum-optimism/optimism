//! Example demonstrating how to access the Engine API instance during construction.
//!
//! Run with
//!
//! ```sh
//! cargo run -p example-engine-api-access
//! ```

use reth_db::test_utils::create_test_rw_db;
use reth_node_builder::{EngineApiExt, FullNodeComponents, NodeBuilder, NodeConfig};
use reth_optimism_chainspec::BASE_MAINNET;
use reth_optimism_node::{
    args::RollupArgs, node::OpEngineValidatorBuilder, OpAddOns, OpEngineApiBuilder, OpNode,
};
use tokio::sync::oneshot;

#[tokio::main]
async fn main() {
    // Op node configuration and setup
    let config = NodeConfig::new(BASE_MAINNET.clone());
    let db = create_test_rw_db();
    let args = RollupArgs::default();
    let op_node = OpNode::new(args);

    let (engine_api_tx, _engine_api_rx) = oneshot::channel();

    let engine_api =
        EngineApiExt::new(OpEngineApiBuilder::<OpEngineValidatorBuilder>::default(), move |api| {
            let _ = engine_api_tx.send(api);
        });

    let _builder = NodeBuilder::new(config)
        .with_database(db)
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(OpAddOns::default().with_engine_api(engine_api))
        .on_component_initialized(move |ctx| {
            let _provider = ctx.provider();
            Ok(())
        })
        .on_node_started(|_full_node| Ok(()))
        .on_rpc_started(|_ctx, handles| {
            let _client = handles.rpc.http_client();
            Ok(())
        })
        .check_launch();
}
