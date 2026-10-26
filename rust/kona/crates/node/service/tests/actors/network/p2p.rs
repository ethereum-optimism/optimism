use crate::actors::network::mocks::builder::TestNetworkBuilder;

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

#[tokio::test(flavor = "multi_thread")]
async fn test_large_network_conn() -> anyhow::Result<()> {
    const NODES: usize = 10;

    let mut builder = TestNetworkBuilder::new();

    let (mut nodes, mut bootnodes) = (vec![], vec![]);

    for _ in 0..NODES {
        let node = builder.build(bootnodes.clone()).await;
        let enr = node.peer_enr().await?;
        nodes.push(node);
        bootnodes.push(enr);
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
