use std::time::Duration;

use crate::actors::network::mocks::builder::TestNetworkBuilder;

/// Two nodes connect through discovery: the second bootstraps from the first's ENR, and its
/// network actor dials the ENRs discovery hands it, over TCP.
#[tokio::test(flavor = "multi_thread")]
async fn test_p2p_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new();
    let node_1 = builder.build(vec![]).await;
    let enr_1 = node_1.peer_enr().await?;

    let node_2 = builder.build(vec![enr_1]).await;

    node_2.is_connected_to_with_retries(&node_1).await?;

    node_1.is_connected_to_with_retries(&node_2).await?;

    Ok(())
}

/// Nodes that all bootstrap from the first one end up connected to each other through discovery.
///
/// A node dials its bootnodes directly, so only the connections to the first node come from
/// bootstrapping. The others need each node to discover the rest through the first node's
/// discovery table. That gives a full mesh because a node dials every peer discovery finds, with
/// no lower or upper limit on its connections.
#[tokio::test(flavor = "multi_thread")]
async fn test_large_network_conn() -> anyhow::Result<()> {
    const NODES: usize = 10;

    // Nodes built after an earlier node's lookup only find each other at a later lookup.
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));

    let bootnode = builder.build(vec![]).await;
    let bootnode_enr = bootnode.peer_enr().await?;

    let mut nodes = vec![bootnode];
    for _ in 1..NODES {
        nodes.push(builder.build(vec![bootnode_enr.clone()]).await);
    }

    for node in &nodes {
        for other_node in &nodes {
            if node.peer_id().await? == other_node.peer_id().await? {
                continue;
            }

            node.is_connected_to_with_retries(other_node).await?;
        }
    }

    Ok(())
}

/// Nodes gossiping over the in-memory transport connect through `opp2p_connectPeer`, while
/// discovery, with no bootnodes, finds nobody.
#[tokio::test(flavor = "multi_thread")]
async fn test_memory_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().in_memory();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;

    node_2.connect_to(&node_1).await?;

    node_2.is_connected_to_with_retries(&node_1).await?;
    node_1.is_connected_to_with_retries(&node_2).await?;

    assert!(node_1.discovery_table().await?.is_empty());
    assert!(node_2.discovery_table().await?.is_empty());

    Ok(())
}
