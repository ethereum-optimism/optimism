use std::{net::Ipv4Addr, time::Duration};

use crate::actors::network::mocks::{TestNetwork, builder::TestNetworkBuilder};
use discv5::Enr;
use enr::NodeId;
use kona_genesis::OP_MAINNET_CHAIN_ID;
use kona_gossip::PeerAddr;
use kona_rpc::OpP2PApiServer;
use libp2p::{Multiaddr, identity::Keypair, multiaddr::Protocol};

/// Two nodes connect through discovery: the second bootstraps from the first's ENR, and its
/// network actor dials the ENRs discovery hands it, over TCP.
#[tokio::test(flavor = "multi_thread")]
async fn test_p2p_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new();
    let node_1 = builder.build(vec![]).await;
    let enr_1 = node_1.peer_enr().await?;

    let node_2 = builder.build(vec![enr_1]).await;

    node_2.check_connected_to_with_retries(&node_1).await?;

    node_1.check_connected_to_with_retries(&node_2).await?;

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

            node.check_connected_to_with_retries(other_node).await?;
        }
    }

    Ok(())
}

/// Nodes gossiping over the in-memory transport, without discovery, connect through the
/// `ConnectPeer` command that `opp2p_connectPeer` sends.
#[tokio::test(flavor = "multi_thread")]
async fn test_memory_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().in_memory();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;

    node_2.connect_to(&node_1).await?;

    node_2.check_connected_to_with_retries(&node_1).await?;
    node_1.check_connected_to_with_retries(&node_2).await?;

    assert!(node_1.has_discovery_disabled().await);
    assert!(node_2.has_discovery_disabled().await);

    Ok(())
}

/// A node dials its static peers at startup, with no `ConnectPeer` command and no discovery.
#[tokio::test(flavor = "multi_thread")]
async fn test_static_peer_connects_at_startup() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().in_memory();
    let node_1 = builder.build(vec![]).await;

    let static_peer = PeerAddr::try_from(node_1.listen_addr().await?)?;
    let mut builder = builder.with_static_peers(vec![static_peer]);
    let node_2 = builder.build(vec![]).await;

    node_2.check_connected_to_with_retries(&node_1).await?;
    node_1.check_connected_to_with_retries(&node_2).await?;

    assert!(node_1.has_discovery_disabled().await);
    assert!(node_2.has_discovery_disabled().await);

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

    node_1.check_connected_to_with_retries(&static_node).await?;
    node_1.check_connected_to_with_retries(&node_2).await?;
    node_2.check_connected_to_with_retries(&node_1).await?;

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
    assert!(!node_2.is_connected_to(&static_node).await?);
    assert_eq!(static_node.peers().await?.peers.len(), 1);

    Ok(())
}

/// Returns a UDP port that is free now.
fn free_udp_port() -> u16 {
    let socket = std::net::UdpSocket::bind((Ipv4Addr::UNSPECIFIED, 0)).expect("bind a free port");
    socket.local_addr().expect("bound socket has an address").port()
}

/// Returns the TCP port of a listen address.
fn tcp_port(addr: &Multiaddr) -> Option<u16> {
    addr.iter().find_map(|p| match p {
        Protocol::Tcp(port) => Some(port),
        _ => None,
    })
}

/// A node without discovery doesn't bind the UDP port it would discover on. That holds on a chain
/// with built-in bootnodes too, such as OP Mainnet, which a running discovery bootstraps from even
/// with no bootnodes configured.
///
/// The port is checked after an RPC call that a running discovery only answers once it has
/// started, and so bound the port.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_without_discovery_binds_no_udp_port() -> anyhow::Result<()> {
    for builder in
        [TestNetworkBuilder::new(), TestNetworkBuilder::new().with_chain_id(OP_MAINNET_CHAIN_ID)]
    {
        let udp_port = free_udp_port();
        let mut builder = builder.without_discovery().with_udp_port(udp_port);
        let node = builder.build(vec![]).await;

        node.peers().await?;
        std::net::UdpSocket::bind((Ipv4Addr::UNSPECIFIED, udp_port))?;
        assert!(node.has_discovery_disabled().await);
    }

    Ok(())
}

/// A node without discovery doesn't answer discovery. B has A as a bootnode, so B's discovery
/// keeps A's record in its table and tries to reach it, but never gets a session with it. B does
/// dial A over TCP from that record, which shows A is reachable.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_without_discovery_does_not_answer_discovery() -> anyhow::Result<()> {
    let keypair_a = Keypair::generate_secp256k1();
    let udp_port_a = free_udp_port();
    let mut builder = TestNetworkBuilder::new()
        .with_discovery_interval(Duration::from_millis(100))
        .without_discovery()
        .with_custom_keypair(keypair_a.clone())
        .with_udp_port(udp_port_a);
    let node_a = builder.build(vec![]).await;
    let tcp_port_a = tcp_port(&node_a.listen_addr().await?).expect("listens on TCP");
    let enr_a = builder.enr(&keypair_a, tcp_port_a, udp_port_a);

    let mut builder = builder.with_discovery();
    let node_b = builder.build(vec![enr_a.clone()]).await;

    node_b.check_connected_to_with_retries(&node_a).await?;
    // Give B's discovery many lookups to reach A.
    tokio::time::sleep(Duration::from_secs(2)).await;

    assert!(node_a.has_discovery_disabled().await);
    assert!(node_ids(&node_b.discovery_table().await?).contains(&enr_a.node_id()));
    assert_eq!(node_b.p2p_rpc.opp2p_peer_count().await?.connected_discovery, Some(0));

    Ok(())
}

/// A node without discovery ignores its bootnodes: D, which runs discovery, never hears from N
/// over discovery or TCP, though N has D as a bootnode.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_without_discovery_ignores_its_bootnodes() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));
    let node_d = builder.build(vec![]).await;
    let mut builder = builder.without_discovery();
    let node_n = builder.build(vec![node_d.peer_enr().await?]).await;

    // Give a discovery that runs by mistake many lookups to reach D.
    tokio::time::sleep(Duration::from_secs(2)).await;

    assert!(node_n.has_discovery_disabled().await);
    assert!(!node_ids(&node_d.discovery_table().await?).contains(&node_n.node_id().await?));
    assert!(node_d.peers().await?.peers.is_empty());

    Ok(())
}

/// A node without discovery binds no UDP socket, so it starts and connects to its static peer even
/// though the UDP port it would discover on is taken. A discovery that ran by mistake would fail
/// to start on that port and stop, failing the RPC calls once it gave up.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_without_discovery_starts_with_its_udp_port_taken() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().without_discovery();
    let static_node = builder.build(vec![]).await;
    let static_peer = PeerAddr::try_from(static_node.listen_addr().await?)?;

    let taken = std::net::UdpSocket::bind((Ipv4Addr::UNSPECIFIED, 0))?;
    let mut builder =
        builder.with_static_peers(vec![static_peer]).with_udp_port(taken.local_addr()?.port());
    let node = builder.build(vec![]).await;

    node.check_connected_to_with_retries(&static_node).await?;
    assert!(node.has_discovery_disabled().await);
    assert_eq!(node.peers().await?.peers.len(), 1);

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

/// `opp2p_connectPeer` fails at once, with the gate's reason, when the operator has blocked the
/// peer, its IP address or its subnet, instead of waiting for a connection that never comes. Once
/// unblocked, the peer connects.
///
/// Over TCP, because the gate only applies IP and subnet blocks to addresses with an IP, which an
/// in-memory one doesn't have.
#[tokio::test(flavor = "multi_thread")]
async fn test_connect_peer_reports_the_gates_refusal() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().without_discovery();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;
    let peer_1 = node_1.peer_id().await?;
    let port = tcp_port(&node_1.listen_addr().await?).expect("a TCP listen address");
    let addr_1 = format!("/ip4/127.0.0.1/tcp/{port}/p2p/{peer_1}");
    let ip = "127.0.0.1".parse()?;
    let subnet = "127.0.0.0/8".parse()?;
    let rpc = &node_2.p2p_rpc;

    // Well under the 10 s that opp2p_connectPeer waits for a connection once it has dialed.
    let refusal = async || {
        let connect = rpc.opp2p_connect_peer(addr_1.clone());
        let err = tokio::time::timeout(Duration::from_secs(2), connect)
            .await
            .expect("refused at once")
            .expect_err("refused");
        assert_eq!(err.code(), -32000);
        err.message().to_owned()
    };

    rpc.opp2p_block_peer(peer_1.clone()).await?;
    assert_eq!(refusal().await, format!("Peer is blocked: {peer_1}"));
    rpc.opp2p_unblock_peer(peer_1.clone()).await?;

    rpc.opp2p_block_addr(ip).await?;
    assert_eq!(refusal().await, "IP address is blocked: 127.0.0.1");
    rpc.opp2p_unblock_addr(ip).await?;

    rpc.opp2p_block_subnet(subnet).await?;
    assert_eq!(refusal().await, "IP address 127.0.0.1 is in a blocked subnet");
    rpc.opp2p_unblock_subnet(subnet).await?;

    tokio::time::timeout(Duration::from_secs(10), rpc.opp2p_connect_peer(addr_1.clone())).await??;
    node_1.check_connected_to_with_retries(&node_2).await?;
    Ok(())
}

/// Returns the address of `node`'s gossip listener on the loopback interface, ending in its
/// peer ID, for `opp2p_connectPeer`.
async fn loopback_addr(node: &TestNetwork) -> anyhow::Result<String> {
    let peer = node.peer_id().await?;
    let port = tcp_port(&node.listen_addr().await?).expect("a TCP listen address");
    Ok(format!("/ip4/127.0.0.1/tcp/{port}/p2p/{peer}"))
}

/// `opp2p_connectPeer` returns once the peers are connected.
///
/// The nodes connect over TCP, as real nodes do, though any transport would show it: the
/// command only starts the dial, and the connection is established later, in the background.
/// The RPC has to wait for that.
#[tokio::test(flavor = "multi_thread")]
async fn test_connect_peer_waits_for_the_connection() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().without_discovery();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;

    let connect = node_2.p2p_rpc.opp2p_connect_peer(loopback_addr(&node_1).await?);
    tokio::time::timeout(Duration::from_secs(10), connect).await??;
    node_2.check_connected_to(&node_1).await?;
    Ok(())
}

/// `opp2p_disconnectPeer` returns once the peers are disconnected.
///
/// Over TCP, as in [`test_connect_peer_waits_for_the_connection`], though any transport would
/// show it: the command only starts closing the connections, and they close later, in the
/// background. The RPC has to wait for that.
#[tokio::test(flavor = "multi_thread")]
async fn test_disconnect_peer_waits_for_the_disconnection() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().without_discovery();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;
    node_2.connect_to(&node_1).await?;
    node_2.check_connected_to_with_retries(&node_1).await?;

    let disconnect = node_2.p2p_rpc.opp2p_disconnect_peer(node_1.peer_id().await?);
    tokio::time::timeout(Duration::from_secs(10), disconnect).await??;
    assert!(!node_2.is_connected_to(&node_1).await?, "still connected");
    Ok(())
}

/// `opp2p_connectPeer` to an address where nothing listens fails as soon as the dial does, with
/// the dial's error, as op-node's does.
///
/// Over TCP, so that the dial reaches a real port and the OS refuses it.
#[tokio::test(flavor = "multi_thread")]
async fn test_connect_peer_reports_a_failed_dial() -> anyhow::Result<()> {
    let node = TestNetworkBuilder::new().without_discovery().build(vec![]).await;
    let port = std::net::TcpListener::bind((Ipv4Addr::LOCALHOST, 0))?.local_addr()?.port();
    let peer = Keypair::generate_secp256k1().public().to_peer_id();

    let connect = node.p2p_rpc.opp2p_connect_peer(format!("/ip4/127.0.0.1/tcp/{port}/p2p/{peer}"));
    let err = tokio::time::timeout(Duration::from_secs(5), connect)
        .await
        .expect("fails as soon as the dial does")
        .expect_err("nothing listens");
    assert_eq!(err.code(), -32000, "{err:?}");
    assert!(err.message().contains("Connection refused"), "{err:?}");
    Ok(())
}
