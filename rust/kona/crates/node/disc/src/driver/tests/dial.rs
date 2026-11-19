//! Tests for the records the driver hands over to gossip to dial, and which of them it saves.

use super::*;
use rstest::rstest;

/// The socket a session in these tests is with.
const SOCKET: SocketAddr = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 9223);

/// Gossip dials the peer a session or an unverifiable record brings, if it is for the driver's
/// chain, and nothing for other events: a discovered record isn't dialed until a session with
/// its node shows where it is. The record comes with the socket of its session, for the bootstore
/// to check against the record's UDP address.
#[rstest]
#[case::discovered(discv5::Event::Discovered, None)]
#[case::session_established(
    |enr| discv5::Event::SessionEstablished(enr, SOCKET),
    Some("session_established")
)]
#[case::unverifiable_enr(
    |enr: Enr| discv5::Event::UnverifiableEnr { node_id: enr.node_id(), enr, socket: SOCKET },
    Some("unverifiable_enr")
)]
#[case::node_inserted(
    |enr: Enr| discv5::Event::NodeInserted { node_id: enr.node_id(), replaced: None },
    None
)]
fn test_record_to_dial(
    #[case] event: impl Fn(Enr) -> discv5::Event,
    #[case] expected: Option<&'static str>,
) {
    let enr = enr_at_localhost(OP_SEPOLIA_CHAIN_ID);
    assert_eq!(
        Discv5Driver::record_to_dial(event(enr.clone()), OP_SEPOLIA_CHAIN_ID, accept_all),
        expected.map(|event| RecordToDial { event, enr, socket: SOCKET })
    );
    assert_eq!(
        Discv5Driver::record_to_dial(
            event(enr_at_localhost(OP_MAINNET_CHAIN_ID)),
            OP_SEPOLIA_CHAIN_ID,
            accept_all
        ),
        None,
        "a record for another chain is never dialed"
    );
}

/// A session with a node whose packets came from `sender`.
fn session_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
    let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
    move |enr| discv5::Event::SessionEstablished(enr, socket)
}

/// A session with a node whose record doesn't match `sender`, the address its packets came
/// from.
fn unverifiable_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
    let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
    move |enr: Enr| discv5::Event::UnverifiableEnr { node_id: enr.node_id(), enr, socket }
}

/// A session from `sender` with another node than the one whose record it presented.
fn unverifiable_with_other_node_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
    let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
    move |enr: Enr| discv5::Event::UnverifiableEnr { node_id: NodeId::random(), enr, socket }
}

/// A session's record holding another public host's address, such as a web server's, with a
/// TCP port and no UDP port, which discv5 doesn't check against the session.
fn third_party_record() -> Enr {
    enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::new(93, 184, 216, 34)).tcp4(443);
    })
}

/// A session's record holding loopback with a TCP port, and no UDP port.
fn loopback_record_without_udp() -> Enr {
    enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::LOCALHOST).tcp4(6379);
    })
}

/// A session's record whose IPv4 address has a UDP port but no TCP port, with loopback and a
/// TCP port in its IPv6 address.
fn record_with_tcp_only_in_other_family() -> Enr {
    enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::new(1, 2, 3, 4)).udp4(9223).ip6(Ipv6Addr::LOCALHOST).tcp6(6379);
    })
}

/// A record whose address matches its session's: `1.2.3.4`, with UDP and TCP ports.
fn matching_record() -> Enr {
    enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::new(1, 2, 3, 4)).udp4(9223).tcp4(9222);
    })
}

/// A session should confirm the address gossip dials: a node should dial, and save, a
/// session's record only if the address gossip would dial from it, its IPv4 TCP address or else
/// its IPv6 one, has the IP the session's packets came from. Then it dials that address. A
/// record naming another host is dropped, not dialed elsewhere, because the bootstore saves
/// the record: after a restart, it would be dialed at the address in it.
///
/// An unverifiable record whose IP matches, but not its UDP port, as behind a NAT that
/// rewrites ports, is dialed too, but only if it is the session's node's: one that another
/// node presented hasn't been confirmed by any session with the node it names.
#[rstest]
#[case::matching(session_from("1.2.3.4"), matching_record(), Some(9222))]
#[case::matching_ip_unverifiable(unverifiable_from("1.2.3.4"), matching_record(), Some(9222))]
#[case::matching_ip_other_node(
    unverifiable_with_other_node_from("1.2.3.4"),
    matching_record(),
    None
)]
#[case::third_party(session_from("1.2.3.4"), third_party_record(), None)]
#[case::third_party_unverifiable(unverifiable_from("1.2.3.4"), third_party_record(), None)]
#[case::loopback(session_from("1.2.3.4"), loopback_record_without_udp(), None)]
#[case::tcp_only_in_other_family(
    session_from("1.2.3.4"),
    record_with_tcp_only_in_other_family(),
    None
)]
fn test_session_record_is_dialed_only_at_the_session_address(
    #[case] event: impl Fn(Enr) -> discv5::Event,
    #[case] record: Enr,
    #[case] tcp_port: Option<u16>,
) {
    let dialed = Discv5Driver::record_to_dial(event(record), OP_SEPOLIA_CHAIN_ID, accept_all).map(
        |record| {
            let addr = kona_peers::enr_to_multiaddr(&record.enr)
                .expect("a dialed record has a TCP address");
            addr.iter().take(2).map(|p| p.acquire()).collect::<Vec<_>>()
        },
    );
    assert_eq!(
        dialed,
        tcp_port.map(|port| vec![Protocol::Ip4(Ipv4Addr::new(1, 2, 3, 4)), Protocol::Tcp(port)]),
    );
}

/// A session's record is dialed, and saved, only if the table filter accepts it, as the
/// bootstore dials what it holds without a session.
#[rstest]
#[case::public_only(crate::public_only, false)]
#[case::allowing_private(accept_all, true)]
fn test_record_to_dial_only_what_the_table_filter_accepts(
    #[case] table_filter: fn(&Enr) -> bool,
    #[case] dialed: bool,
) {
    let enr = enr_at_localhost(OP_SEPOLIA_CHAIN_ID);
    let event = session_from("127.0.0.1")(enr.clone());
    assert_eq!(
        Discv5Driver::record_to_dial(event, OP_SEPOLIA_CHAIN_ID, table_filter),
        dialed.then(|| RecordToDial { event: "session_established", enr, socket: SOCKET })
    );
}

/// A node with a public-only table filter should neither dial nor save a node at a private
/// address after a session with it, and leave the dial to that node. Refusing the dial doesn't
/// keep the two apart: the private node can dial us, and gossip accepts it. But a node that
/// dials a record it doesn't save can never redial it, and its dial can come first, so that the
/// other node holds only a connection it didn't dial, which doesn't confirm our record. Then
/// neither node redials once the connection drops. The private node saves our record, so its
/// own dial confirms it, and it can redial it.
///
/// The driver dials a record by handing it over to gossip, which dials it. There's no gossip
/// here, so the test checks the handover.
///
/// N has a public-only filter, and a bootstore with a public record gossip reaches. L has no
/// filter, and has N as its bootnode, given by an enode URL on loopback, so L's bootstrap starts
/// a session with N from loopback, which L's record holds.
#[tokio::test]
async fn test_public_only_node_leaves_the_dial_to_a_private_session() {
    let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let (mut node, dir) = public_driver_with_bootstore(std::slice::from_ref(&public));
    node.store_interval = Duration::from_millis(100);
    let path = dir.path().join("bootstore.json");
    let node_enr = node.disc.local_enr();
    let enode = loopback_enode(&node_enr);
    let (node, mut records) = node.start();
    reach_through(&node, &public);

    let mut loopback = offline_driver();
    loopback.bootnodes = BootNodes(vec![BootNode::Enode(enode)]);
    let loopback_id = loopback.disc.local_enr().node_id();
    let (loopback, _) = loopback.start();

    // L's bootstrap ends once it has N's record, which needs the session.
    wait_until_started(&loopback).await;

    // N handles the session about when L's bootstrap ends. Each store interval after that, N
    // hands over its saved public record, so after two of them, L's record would have come.
    let mut handed_over = Vec::new();
    while let Ok(record) = records.try_recv() {
        handed_over.push(record);
    }
    let two_ticks = async {
        let mut ticks = 0;
        while ticks < 2 {
            let record = records.recv().await.expect("the driver runs");
            if record == public {
                ticks += 1;
            }
            handed_over.push(record);
        }
    };
    tokio::time::timeout(Duration::from_secs(5), two_ticks)
        .await
        .expect("the driver hands its saved records over every store interval");

    assert!(
        !handed_over.iter().any(|enr| enr.node_id() == loopback_id),
        "a public-only node handed a loopback record it doesn't save to gossip to dial"
    );
    let saved = BootStore::open(path).expect("the bootstore opens");
    assert!(
        !saved.iter().any(|enr| enr.node_id() == loopback_id),
        "a public-only node saved a loopback record"
    );
}

/// The driver should take discv5's event stream before it bootstraps, which can take seconds
/// when bootnodes don't answer: discv5 drops the events it has no stream for, so a node that
/// starts a session with us meanwhile would never be dialed.
///
/// N's only bootnode is a socket that never answers, so N's bootstrap waits for its request to
/// time out, 3s here. P has N as its bootnode, and contacts N in its first lookup, during N's
/// bootstrap.
#[tokio::test]
async fn test_driver_reports_sessions_started_during_bootstrap() {
    let silent = std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("a free UDP port");
    let silent_peer_id = kona_peers::enr_to_multiaddr(&enr_at_localhost(OP_SEPOLIA_CHAIN_ID))
        .and_then(|addr| addr.iter().last().map(|protocol| protocol.acquire()))
        .expect("the record has a peer ID");
    let silent_enode = Multiaddr::from(Ipv4Addr::LOCALHOST)
        .with(Protocol::Udp(silent.local_addr().expect("the socket is bound").port()))
        .with(silent_peer_id);

    let mut node = loopback_driver(
        loopback_config().request_timeout(Duration::from_secs(3)).request_retries(0).build(),
    );
    node.bootnodes = BootNodes(vec![BootNode::Enode(silent_enode)]);
    let node_enr = node.disc.local_enr();
    let (_node, mut records) = node.start();

    let mut peer = offline_driver();
    peer.bootnodes = BootNodes(vec![BootNode::Enr(node_enr)]);
    peer.interval = Duration::from_millis(100);
    let peer_id = peer.disc.local_enr().node_id();
    let (_peer, _) = peer.start();

    let handed_over = async {
        loop {
            if records.recv().await.expect("the driver runs").node_id() == peer_id {
                return;
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), handed_over)
        .await
        .expect("the node didn't hand over a peer that started a session during its bootstrap");
}

/// discv5 reports every record in a NODES answer as `Discovered`, up to 16 an answer, on an
/// event channel of 100, 30 with the reports off, that drops the events that don't fit. The
/// driver doesn't dial those records, so it should turn the reports off: otherwise they can
/// push out the `SessionEstablished` events that bring the peers it dials.
///
/// N looks up O through B, whose table holds O. A lookup asks each node for the nodes at its
/// distance to the target, so B's answer holds O only for a target O shares a bucket with:
/// O's own ID is one.
#[tokio::test]
async fn test_driver_turns_off_discovered_reports() {
    let other = offline_driver().init().await.expect("the other node starts");
    let other_id = other.disc.local_enr().node_id();
    let bootnode = offline_driver().init().await.expect("the bootnode starts");
    bootnode.disc.add_enr(other.disc.local_enr()).expect("the bootnode takes the record");
    let node = offline_driver().init().await.expect("the node starts");
    let mut events = node.disc.event_stream().await.expect("the node has an event stream");
    node.disc.add_enr(bootnode.disc.local_enr()).expect("the node takes the bootnode");

    let found = node.disc.find_node(other_id).await.expect("the lookup runs");
    assert!(
        found.iter().any(|enr| enr.node_id() == other_id),
        "the lookup didn't find the bootnode's other node"
    );

    // discv5 reports a record before the lookup that found it ends.
    while let Ok(event) = events.try_recv() {
        assert!(
            !matches!(event, discv5::Event::Discovered(_)),
            "discv5 reported a discovered record: {event:?}"
        );
    }
}
