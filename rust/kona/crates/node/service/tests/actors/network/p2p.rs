use std::{
    net::{IpAddr, Ipv4Addr},
    time::Duration,
};

use crate::actors::network::mocks::{
    TestNetwork,
    builder::{TestNetworkBuilder, local_node_key},
};
use discv5::Enr;
use enr::NodeId;
use kona_genesis::OP_MAINNET_CHAIN_ID;
use kona_gossip::PeerAddr;
use kona_node_service::IpVotes;
use kona_peers::BootNode;
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

/// Few enough votes for a handful of test nodes to agree on an address.
const TWO_VOTES: IpVotes = IpVotes::new(2).expect("at least 2");

/// A node listening on all interfaces advertises no IP at first, so it learns its address from
/// the peers it contacts. It finds R1 and R2 through its bootnode, and once both report the same
/// address, as its 2 IP votes ask, it advertises it. A node that joins later through the same
/// bootnode then finds it and connects to it.
///
/// The bootnode, R1 and R2 don't add it: they ignored it when their sessions with it started, as
/// it had no IP then, and discv5 doesn't tell them its record changed. The late node gets its
/// record when the node finds it through the bootnode and starts a new session with it.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_on_all_interfaces_learns_its_ip_from_votes() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));
    let bootnode = builder.build(vec![]).await;
    let bootnode_enr = bootnode.peer_enr().await?;
    let reachable_1 = builder.build(vec![bootnode_enr.clone()]).await;
    let reachable_2 = builder.build(vec![bootnode_enr.clone()]).await;
    for reachable in [&reachable_1, &reachable_2] {
        bootnode.check_in_discovery_table_with_retries(reachable.node_id().await?).await?;
    }

    let mut builder = builder
        .with_next_listen_ip(IpAddr::V4(Ipv4Addr::UNSPECIFIED))
        .with_next_ip_votes(TWO_VOTES);
    let node = builder.build(vec![bootnode_enr.clone()]).await;

    let enr = node.advertised_enr_with_retries().await?;
    assert_eq!(
        enr.ip4(),
        Some(Ipv4Addr::LOCALHOST),
        "on all interfaces the node starts without an IP, so only the votes of R1 and R2 give it one"
    );

    let late_node = builder.build(vec![bootnode_enr]).await;
    late_node.check_in_discovery_table_with_retries(node.node_id().await?).await?;
    late_node.check_connected_to_with_retries(&node).await?;

    Ok(())
}

/// A node listening on all interfaces gets no votes from its bootnodes: discv5 adds bootnodes as
/// if they had contacted the node, and only counts the votes of peers the node contacted. So with
/// two bootnodes that know no other node, the node never advertises an IP, even if 2 votes would
/// do, and the bootnodes don't keep it in their tables, however many lookups it runs.
#[tokio::test(flavor = "multi_thread")]
async fn test_node_on_all_interfaces_gets_no_votes_from_bootnodes() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));
    let bootnode_1 = builder.build(vec![]).await;
    let bootnode_2 = builder.build(vec![]).await;
    let bootnodes = vec![bootnode_1.peer_enr().await?, bootnode_2.peer_enr().await?];

    let mut builder = builder
        .with_next_listen_ip(IpAddr::V4(Ipv4Addr::UNSPECIFIED))
        .with_next_ip_votes(TWO_VOTES);
    let node = builder.build(bootnodes).await;

    // Discovery has sessions with both bootnodes, so it reached them. Gossip connecting to them
    // wouldn't show it: it dials bootnodes whether or not discovery reached them.
    node.check_discovery_peer_count_with_retries(2).await?;
    // Give it many lookups to collect votes. discv5 doesn't ping the bootnodes in that time: it
    // pings at once only peers it contacted, and the others every `ping_interval` (300s). A pong
    // from a bootnode wouldn't count anyway, for the same reason.
    tokio::time::sleep(Duration::from_secs(2)).await;

    assert_eq!(node.peer_enr().await?.ip4(), None);
    let node_id = node.node_id().await?;
    for bootnode in [&bootnode_1, &bootnode_2] {
        assert!(!node_ids(&bootnode.discovery_table().await?).contains(&node_id));
    }

    Ok(())
}

/// The IP this host would send from to another host, if it isn't loopback: usually a LAN address,
/// or a public one on a host with one on its interface. Neither can reach another node's loopback.
///
/// Connecting a UDP socket only picks the route to an address, and sends nothing, so this connects
/// to a documentation address (RFC 5737), which no host has.
fn host_ip() -> Option<IpAddr> {
    let socket = std::net::UdpSocket::bind((Ipv4Addr::UNSPECIFIED, 0)).ok()?;
    socket.connect((Ipv4Addr::new(192, 0, 2, 1), 9)).ok()?;
    let ip = socket.local_addr().ok()?.ip();
    (!ip.is_loopback() && !ip.is_unspecified()).then_some(ip)
}

/// Starts B (`bootnode`), which listens on, and advertises, the host's own IP, and R (`relayed`),
/// which listens on, and advertises, `127.0.0.1`, and joins through B. B adds R to its table, as R
/// started a session with it from the address R advertises, and hands R out to the nodes that look
/// it up.
///
/// B and R run their only lookup at startup, so R never learns of a node that joins later. Given
/// more lookups, R would find such a node through B and contact it, which works on one host. That
/// node would then keep R and dial it, as a node that reached it from the address it advertises.
///
/// Returns the builder for the node under test, which looks up every 100ms.
async fn bootnode_relaying_loopback_node(
    host_ip: IpAddr,
) -> anyhow::Result<(TestNetworkBuilder, TestNetwork, TestNetwork)> {
    let mut builder = TestNetworkBuilder::new()
        .with_discovery_interval(Duration::from_secs(3600))
        .with_next_listen_ip(host_ip);
    let bootnode = builder.build(vec![]).await;
    let relayed = builder.build(vec![bootnode.peer_enr().await?]).await;
    bootnode.check_in_discovery_table_with_retries(relayed.node_id().await?).await?;
    Ok((builder.with_discovery_interval(Duration::from_millis(100)), bootnode, relayed))
}

/// A node that allows private addresses, as a devnet node with `--p2p.discovery.private` does,
/// dials a loopback node it found through a LAN bootnode once it has a session with it: P (`node`)
/// listens on, and advertises, the host's own IP, and joins through B, which holds R on
/// loopback, see [`bootnode_relaying_loopback_node`]. P's lookups through B return R, P's discovery
/// contacts R, and gossip dials it at the address that session came from.
///
/// Skipped on a host with no address but loopback.
#[tokio::test(flavor = "multi_thread")]
async fn test_lan_node_finds_loopback_nodes_through_a_lan_bootnode() -> anyhow::Result<()> {
    let Some(host_ip) = host_ip() else {
        eprintln!("skipping, this host has no address but loopback");
        return Ok(());
    };
    let (builder, bootnode, relayed) = bootnode_relaying_loopback_node(host_ip).await?;

    let mut builder = builder.with_next_listen_ip(host_ip);
    let node = builder.build(vec![bootnode.peer_enr().await?]).await;
    node.check_in_discovery_table_with_retries(relayed.node_id().await?).await?;
    node.check_connected_to_with_retries(&relayed).await?;

    Ok(())
}

/// Stands in for `kona_disc::public_only` on one host, where no address is public: the host's own
/// IP plays the public address, and loopback the private one. A node with this filter uses no
/// loopback address in other nodes' records, as a public-only node uses no private one.
fn not_loopback(enr: &Enr) -> bool {
    let ips = [enr.ip4().map(IpAddr::V4), enr.ip6().map(IpAddr::V6)];
    ips.iter().any(Option::is_some) && ips.into_iter().flatten().all(|ip| !ip.is_loopback())
}

/// The wrong way to keep a sequencer behind a sentry: running discovery against it. Here S, the
/// sequencer, is on a private network, which loopback stands for, and B, its sentry, and E, an
/// external node, are on the public one, which the host's own IP stands for. All three use only
/// public addresses, as kona does by default, see [`not_loopback`]:
///
/// - B listens on the host's IP.
/// - E listens on the host's IP, and joins through B.
/// - S listens on loopback, and joins through B.
///
/// S and B connect over gossip, as S dials B. But B keeps S in its discovery table, as discv5
/// keeps every node it has a session with, whatever its table filter, and hands S's record, with
/// its private address, out to the nodes that look it up. E drops that record. But S learns E
/// through B, and contacts it directly. That session puts S into E's table, though E never
/// contacts S itself, and the two connect over gossip: S is no longer behind its sentry.
///
/// Skipped on a host with no address but loopback.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_running_discovery_against_its_sentry_is_exposed() -> anyhow::Result<()> {
    let Some(host_ip) = host_ip() else {
        eprintln!("skipping, this host has no address but loopback");
        return Ok(());
    };
    let mut builder = TestNetworkBuilder::new()
        .with_discovery_interval(Duration::from_millis(100))
        .with_next_listen_ip(host_ip)
        .with_next_table_filter(not_loopback);
    let sentry = builder.build(vec![]).await;
    let sentry_enr = sentry.peer_enr().await?;

    let mut builder = builder.with_next_listen_ip(host_ip).with_next_table_filter(not_loopback);
    let external = builder.build(vec![sentry_enr.clone()]).await;
    let external_id = external.node_id().await?;
    sentry.check_in_discovery_table_with_retries(external_id).await?;

    let mut builder = builder.with_next_table_filter(not_loopback);
    let sequencer = builder.build(vec![sentry_enr]).await;
    let sequencer_id = sequencer.node_id().await?;

    sequencer.check_connected_to_with_retries(&sentry).await?;
    // The sentry keeps the sequencer, and hands it out to the nodes that look it up.
    sentry.check_in_discovery_table_with_retries(sequencer_id).await?;
    // The sequencer finds the external node through its sentry, and contacts it: the external
    // node drops the sequencer's loopback record, so only the sequencer's session gets the
    // sequencer into its table.
    sequencer.check_in_discovery_table_with_retries(external_id).await?;
    external.check_in_discovery_table_with_retries(sequencer_id).await?;
    sequencer.check_connected_to_with_retries(&external).await?;

    Ok(())
}

/// The right way to keep a sequencer behind a sentry: S runs without discovery, and has B as a
/// static peer. The nodes are as in
/// [`test_sequencer_running_discovery_against_its_sentry_is_exposed`].
///
/// S and B connect over gossip, but S never gets into B's discovery table, as kona never passes
/// static peers on through discovery. So E never learns S, and the two never connect, however
/// long discovery runs.
///
/// S runs no discovery, so this holds by construction: the test documents the setup, and isn't a
/// regression test.
///
/// Skipped on a host with no address but loopback.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_with_its_sentry_as_static_peer_stays_hidden() -> anyhow::Result<()> {
    let Some(host_ip) = host_ip() else {
        eprintln!("skipping, this host has no address but loopback");
        return Ok(());
    };
    let mut builder = TestNetworkBuilder::new()
        .with_discovery_interval(Duration::from_millis(100))
        .with_next_listen_ip(host_ip)
        .with_next_table_filter(not_loopback);
    let sentry = builder.build(vec![]).await;
    let sentry_enr = sentry.peer_enr().await?;

    let mut builder = builder.with_next_listen_ip(host_ip);
    let external = builder.build(vec![sentry_enr]).await;
    sentry.check_in_discovery_table_with_retries(external.node_id().await?).await?;
    external.check_connected_to_with_retries(&sentry).await?;

    let sentry_addr = PeerAddr::try_from(sentry.listen_addr().await?)?;
    let mut builder = builder.without_discovery().with_next_static_peers(vec![sentry_addr]);
    let sequencer = builder.build(vec![]).await;
    let sequencer_id = sequencer.node_id().await?;
    sequencer.check_connected_to_with_retries(&sentry).await?;

    // Give the sentry and the external node many lookups.
    tokio::time::sleep(Duration::from_secs(2)).await;

    assert!(
        !node_ids(&sentry.discovery_table().await?).contains(&sequencer_id),
        "the sentry passes the sequencer on through discovery"
    );
    assert!(
        !node_ids(&external.discovery_table().await?).contains(&sequencer_id),
        "the external node learned the sequencer"
    );
    assert!(
        !external.is_connected_to(&sequencer).await?,
        "the external node connected to the sequencer"
    );

    Ok(())
}

/// Nodes of a chain find each other through a bootnode of no chain, as OP Mainnet's built-in enode
/// bootnodes are: their records have no `opstack` key. Such a bootnode isn't a peer of any chain,
/// so it isn't saved or dialed, but it stays in the nodes' discovery tables as an entry point for
/// lookups, which find the nodes of every chain.
///
/// B is a bare discv5 node, given to N1 and N2 by enode URL.
#[tokio::test(flavor = "multi_thread")]
async fn test_nodes_find_each_other_through_a_bootnode_of_no_chain() -> anyhow::Result<()> {
    let (bootnode, enode) = start_bootnode_of_no_chain().await?;
    let bootnode_id = bootnode.local_enr().node_id();

    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));
    let node_1 = builder.build_with_bootnodes(vec![enode.clone()]).await;
    let node_2 = builder.build_with_bootnodes(vec![enode]).await;

    node_1.check_connected_to_with_retries(&node_2).await?;
    node_1.check_in_discovery_table_with_retries(bootnode_id).await?;
    node_2.check_in_discovery_table_with_retries(bootnode_id).await?;

    Ok(())
}

/// Nodes of one chain find each other through a kona node of another chain as their bootnode: its
/// discovery table holds the nodes of every chain, and its gossip dials only those of its own. The
/// nodes don't dial it either, as it isn't a peer of their chain.
///
/// B is on one chain. N1 and N2 are on another, and have B's ENR as their bootnode.
#[tokio::test(flavor = "multi_thread")]
async fn test_nodes_find_each_other_through_a_bootnode_of_another_chain() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().with_discovery_interval(Duration::from_millis(100));
    let bootnode = builder.build(vec![]).await;
    let bootnode_enr = bootnode.peer_enr().await?;

    let mut builder = builder.with_chain_id(rand::random());
    let node_1 = builder.build(vec![bootnode_enr.clone()]).await;
    let node_2 = builder.build(vec![bootnode_enr]).await;

    node_1.check_connected_to_with_retries(&node_2).await?;
    for node in [&node_1, &node_2] {
        assert!(
            !node.is_connected_to(&bootnode).await?,
            "a node dialed a bootnode of another chain"
        );
    }

    Ok(())
}

/// Starts a bootnode of no chain, as OP Mainnet's built-in enode bootnodes are: a bare discv5 node
/// on loopback, whose record has no `opstack` key and no TCP port. Returns it, to keep it running,
/// with its enode URL.
async fn start_bootnode_of_no_chain() -> anyhow::Result<(discv5::Discv5, BootNode)> {
    let keypair = Keypair::generate_secp256k1();
    let key = enr::CombinedKey::Secp256k1(local_node_key(&keypair));
    let port = free_udp_port();
    let enr = Enr::builder().ip4(Ipv4Addr::LOCALHOST).udp4(port).build(&key)?;
    let config = discv5::ConfigBuilder::new(discv5::ListenConfig::from_ip(
        IpAddr::V4(Ipv4Addr::LOCALHOST),
        port,
    ))
    .build();
    let mut bootnode = discv5::Discv5::new(enr, key, config).map_err(anyhow::Error::msg)?;
    bootnode.start().await.map_err(|e| anyhow::anyhow!("{e:?}"))?;
    let enode = Multiaddr::from(Ipv4Addr::LOCALHOST)
        .with(Protocol::Udp(port))
        .with(Protocol::P2p(keypair.public().to_peer_id()));
    Ok((bootnode, BootNode::Enode(enode)))
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
    let mut builder = builder.with_next_static_peers(vec![static_peer]);
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

    let mut builder = builder.with_next_static_peers(vec![static_peer]);
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
    // Bind where the test nodes bind, not on all interfaces: whether a socket on one address
    // blocks the same port on another differs between platforms, so only the same address tells
    // what the node will find.
    let socket = std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("bind a free port");
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
        let mut builder = builder.without_discovery().with_next_udp_port(udp_port);
        let node = builder.build(vec![]).await;

        node.peers().await?;
        std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, udp_port))?;
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
        .with_next_keypair(keypair_a.clone())
        .with_next_udp_port(udp_port_a);
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

    let taken = std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, 0))?;
    let mut builder = builder
        .with_next_static_peers(vec![static_peer])
        .with_next_udp_port(taken.local_addr()?.port());
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

/// What the operator blocks in [`test_connect_peer_reports_the_gates_refusal`].
#[derive(Debug, Clone, Copy)]
enum Blocked {
    Peer,
    Address,
    Subnet,
}

/// `opp2p_connectPeer` fails at once, with the gate's reason, when the operator has blocked the
/// peer, its IP address or its subnet, instead of waiting for a connection that never comes. Once
/// unblocked, the peer connects.
///
/// Over TCP, because the gate only applies IP and subnet blocks to addresses with an IP, which an
/// in-memory one doesn't have.
#[rstest::rstest]
#[case::peer(Blocked::Peer)]
#[case::address(Blocked::Address)]
#[case::subnet(Blocked::Subnet)]
#[tokio::test(flavor = "multi_thread")]
async fn test_connect_peer_reports_the_gates_refusal(
    #[case] blocked: Blocked,
) -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().without_discovery();
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;
    let peer_1 = node_1.peer_id().await?;
    let addr_1 = loopback_addr(&node_1).await?;
    let ip = "127.0.0.1".parse()?;
    let subnet = "127.0.0.0/8".parse()?;
    let rpc = &node_2.p2p_rpc;

    let expected = match blocked {
        Blocked::Peer => {
            rpc.opp2p_block_peer(peer_1.clone()).await?;
            format!("Peer is blocked: {peer_1}")
        }
        Blocked::Address => {
            rpc.opp2p_block_addr(ip).await?;
            "IP address is blocked: 127.0.0.1".to_owned()
        }
        Blocked::Subnet => {
            rpc.opp2p_block_subnet(subnet).await?;
            "IP address 127.0.0.1 is in a blocked subnet".to_owned()
        }
    };
    // Well under the 30 s that opp2p_connectPeer waits for a connection once it has dialed.
    let err = tokio::time::timeout(Duration::from_secs(2), rpc.opp2p_connect_peer(addr_1.clone()))
        .await
        .expect("refused at once")
        .expect_err("refused");
    assert_eq!(err.code(), -32000);
    assert_eq!(err.message(), expected);

    match blocked {
        Blocked::Peer => rpc.opp2p_unblock_peer(peer_1).await?,
        Blocked::Address => rpc.opp2p_unblock_addr(ip).await?,
        Blocked::Subnet => rpc.opp2p_unblock_subnet(subnet).await?,
    }
    tokio::time::timeout(Duration::from_secs(10), rpc.opp2p_connect_peer(addr_1)).await??;
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

/// A new connection shows in the RPC soon after it opens, even if the RPC accepts gossip state
/// of any age: the network actor publishes new state when a connection opens, without waiting
/// for a reader to find its state too old.
#[tokio::test(flavor = "multi_thread")]
async fn test_new_connection_is_published_without_a_stale_read() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().in_memory().with_snapshot_max_age(Duration::MAX);
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;

    // The command publishes state from before the connection opens.
    node_2.connect_to(&node_1).await?;

    node_2.check_connected_to_with_retries(&node_1).await?;
    node_1.check_connected_to_with_retries(&node_2).await?;
    Ok(())
}

/// Right after `opp2p_connectPeer` returns, `opp2p_peers` lists the new peer, even though the RPC
/// accepts gossip state up to a second old, as it does in the node. The op-devstack reads the
/// peers once, right after connecting.
#[tokio::test(flavor = "multi_thread")]
async fn test_peers_show_a_connection_once_connect_peer_returns() -> anyhow::Result<()> {
    let mut builder =
        TestNetworkBuilder::new().without_discovery().with_snapshot_max_age(Duration::from_secs(1));
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;

    node_2.p2p_rpc.opp2p_connect_peer(loopback_addr(&node_1).await?).await?;
    assert!(node_2.is_connected_to(&node_1).await?, "connected, but not listed yet");
    Ok(())
}

/// Right after `opp2p_disconnectPeer` returns, `opp2p_peers` no longer lists the peer, even though
/// the RPC accepts gossip state up to a second old, as it does in the node.
#[tokio::test(flavor = "multi_thread")]
async fn test_peers_drop_a_connection_once_disconnect_peer_returns() -> anyhow::Result<()> {
    let mut builder =
        TestNetworkBuilder::new().without_discovery().with_snapshot_max_age(Duration::from_secs(1));
    let node_1 = builder.build(vec![]).await;
    let node_2 = builder.build(vec![]).await;
    node_2.p2p_rpc.opp2p_connect_peer(loopback_addr(&node_1).await?).await?;
    node_2.check_connected_to_with_retries(&node_1).await?;

    node_2.p2p_rpc.opp2p_disconnect_peer(node_1.peer_id().await?).await?;
    assert!(!node_2.is_connected_to(&node_1).await?, "disconnected, but still listed");
    Ok(())
}
