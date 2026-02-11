//! RPC integration tests.

use reth_network::types::NatResolver;
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::BASE_MAINNET;
use reth_optimism_node::OpNode;
use reth_rpc_api::servers::AdminApiServer;
use reth_tasks::Runtime;

// <https://github.com/paradigmxyz/reth/issues/19765>
#[tokio::test]
async fn test_admin_external_ip() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();

    let external_ip = "10.64.128.71".parse().unwrap();
    // Node setup
    let mut network_args = NetworkArgs::default()
        .with_unused_ports()
        .with_nat_resolver(NatResolver::ExternalIp(external_ip));
    network_args.discovery.discv5_port = 0;
    network_args.discovery.discv5_port_ipv6 = 0;
    let node_config = NodeConfig::test()
        .map_chain(BASE_MAINNET.clone())
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;

    let api = node.add_ons_handle.admin_api();

    let info = api.node_info().await.unwrap();

    assert_eq!(info.ip, external_ip);

    Ok(())
}
