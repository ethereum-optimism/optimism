use std::time::Duration;

use crate::actors::network::mocks::{TestNetworkError, builder::TestNetworkBuilder};
use discv5::Enr;
use enr::NodeId;
use kona_gossip::PeerAddr;
use kona_rpc::OpP2PApiServer;

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

/// Nodes gossiping over the in-memory transport connect through the `ConnectPeer` command that
/// `opp2p_connectPeer` sends, while discovery, with no bootnodes, finds nobody.
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

/// A node dials its static peers at startup, with no `ConnectPeer` command and no discovery: the
/// in-memory nodes have no bootnodes.
#[tokio::test(flavor = "multi_thread")]
async fn test_static_peer_connects_at_startup() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().in_memory();
    let node_1 = builder.build(vec![]).await;

    let static_peer = PeerAddr::try_from(node_1.listen_addr().await?)?;
    let mut builder = builder.with_static_peers(vec![static_peer]);
    let node_2 = builder.build(vec![]).await;

    node_2.is_connected_to_with_retries(&node_1).await?;
    node_1.is_connected_to_with_retries(&node_2).await?;

    assert!(node_1.discovery_table().await?.is_empty());
    assert!(node_2.discovery_table().await?.is_empty());

    Ok(())
}

/// Returns the node IDs of the ENRs in a discovery table.
fn node_ids(table: &[Enr]) -> Vec<NodeId> {
    table.iter().map(Enr::node_id).collect()
}

/// A static peer never becomes discoverable, over real TCP and UDP: D1 and D2 find each other
/// through discovery, and S is a static peer of D1 only, with no bootnodes. However long discovery
/// runs, S stays out of every discovery table, and D2 never connects to it.
///
/// This holds because nothing feeds the peers libp2p connects to into discovery. A sequencer that
/// peers only statically stays hidden from the rest of the network.
#[tokio::test(flavor = "multi_thread")]
async fn test_static_peer_is_not_discoverable() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));

    // S runs discovery too, but with no bootnodes, so it can only be found through other nodes.
    let static_node = builder.build(vec![]).await;
    let static_peer = PeerAddr::try_from(static_node.listen_addr().await?)?;

    let mut builder = builder.with_static_peers(vec![static_peer]);
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![node_1.peer_enr().await?]).await;

    node_1.is_connected_to_with_retries(&static_node).await?;
    node_1.is_connected_to_with_retries(&node_2).await?;
    node_2.is_connected_to_with_retries(&node_1).await?;

    // Give discovery many rounds to pass S on, if it ever would.
    tokio::time::sleep(Duration::from_secs(3)).await;

    let (static_id, id_1, id_2) =
        (static_node.node_id().await?, node_1.node_id().await?, node_2.node_id().await?);

    // D1 and D2 found each other through discovery, which S takes no part in.
    let table_1 = node_ids(&node_1.discovery_table().await?);
    let table_2 = node_ids(&node_2.discovery_table().await?);
    assert!(table_1.contains(&id_2));
    assert!(table_2.contains(&id_1));
    assert!(!table_1.contains(&static_id));
    assert!(!table_2.contains(&static_id));
    assert!(static_node.discovery_table().await?.is_empty());

    // S's only connection is the one D1 dialed as its static peer.
    assert!(matches!(
        node_2.is_connected_to(&static_node).await,
        Err(TestNetworkError::PeerNotConnected)
    ));
    assert_eq!(static_node.peers().await?.peers.len(), 1);

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn rpc_commands_publish_state_before_returning() -> anyhow::Result<()> {
    let network = TestNetworkBuilder::new().build(vec![]).await;
    let rpc = &network.p2p_rpc;
    let peer = libp2p::identity::Keypair::generate_secp256k1().public().to_peer_id().to_string();
    let address = "192.0.2.1".parse()?;
    let subnet = "192.0.2.0/24".parse()?;

    tokio::time::timeout(std::time::Duration::from_secs(10), async {
        rpc.opp2p_block_peer(peer.clone()).await?;
        assert_eq!(rpc.opp2p_list_blocked_peers().await?, vec![peer.clone()]);
        rpc.opp2p_block_addr(address).await?;
        assert_eq!(rpc.opp2p_list_blocked_addrs().await?, vec![address]);
        rpc.opp2p_block_subnet(subnet).await?;
        assert_eq!(rpc.opp2p_list_blocked_subnets().await?, vec![subnet]);
        rpc.opp2p_unblock_peer(peer).await?;
        rpc.opp2p_unblock_addr(address).await?;
        rpc.opp2p_unblock_subnet(subnet).await?;
        assert!(rpc.opp2p_list_blocked_peers().await?.is_empty());
        assert!(rpc.opp2p_list_blocked_addrs().await?.is_empty());
        assert!(rpc.opp2p_list_blocked_subnets().await?.is_empty());
        Ok::<(), anyhow::Error>(())
    })
    .await??;
    Ok(())
}
