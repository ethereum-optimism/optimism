//! Tests for static peers: dialing them, redialing them, and how blocks, bans and
//! protection apply to them.

use super::*;
use crate::builder::tests::memory_driver_builder_at;
use rstest::rstest;

/// Starting a driver dials its static peers, at all of their addresses.
#[tokio::test]
async fn test_start_dials_static_peers() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();

    let static_peers =
        vec![PeerAddr::try_from(unused_memory_addr(peer_2)).unwrap(), driver_2.peer_addr()];
    let mut driver_1 = memory_driver_builder()
        .with_static_peers(static_peers)
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");

    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// How often static peers are dialed again in tests.
const REDIAL_INTERVAL: Duration = Duration::from_millis(100);

/// A random identity and memory address for a peer, with the static peer address they make,
/// chosen up front so the peer can be listed as a static peer before it starts, or restart at
/// the same address.
fn random_identity() -> (libp2p_identity::Keypair, Multiaddr, PeerAddr) {
    let keypair = libp2p_identity::Keypair::generate_secp256k1();
    let addr = Multiaddr::empty().with(Protocol::Memory(rand::random::<u64>()));
    let static_peer = PeerAddr::new(addr.clone(), keypair.public().to_peer_id()).unwrap();
    (keypair, addr, static_peer)
}

/// A driver whose only static peer is `static_peer`, dialed again every [`REDIAL_INTERVAL`].
fn driver_with_static_peer(static_peer: PeerAddr) -> GossipDriverStarter<ConnectionGater> {
    memory_driver_builder()
        .with_static_peers(vec![static_peer])
        .with_static_peer_redial_interval(REDIAL_INTERVAL)
        .build()
        .expect("build")
}

/// A static peer that starts after the node is dialed on the next check, and once it is
/// connected, the checks leave it alone.
#[tokio::test]
async fn test_static_peer_started_later_is_dialed() {
    let (keypair_2, addr_2, static_peer) = random_identity();
    let peer_2 = static_peer.peer_id();
    let mut driver_1 = driver_with_static_peer(static_peer.clone()).start().await.expect("listen");

    // The dial at startup fails, as nothing listens at the address yet.
    assert!(driver_1.connection_gate.current_dials.contains(&peer_2));
    poll_one_until(&mut driver_1, |driver_1| {
        !driver_1.connection_gate.current_dials.contains(&peer_2)
    })
    .await;
    assert!(!driver_1.is_connected(&peer_2));

    // Once the peer starts, a later check dials it again.
    let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2)
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    // Skipped before `dial_peer`, which would count it as an `already_connected` error.
    let num_dials = driver_1.connection_gate.dialed_addrs[static_peer.addr()].num_dials;
    assert!(dial_errors(|| driver_1.dial_static_peers()).is_empty());
    assert_eq!(driver_1.connection_gate.dialed_addrs[static_peer.addr()].num_dials, num_dials);
}

/// A static peer that restarts is dialed again soon after its connection closes, well before
/// the next regular check, which is a minute away.
#[tokio::test]
async fn test_static_peer_reconnects_after_restart() {
    let (keypair_2, addr_2, static_peer) = random_identity();
    let peer_2 = static_peer.peer_id();
    let mut driver_2 = memory_driver_builder_at(keypair_2.clone(), addr_2.clone())
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");
    // A short reconnect delay, but the default minute between regular checks: only the
    // reconnect after the connection closes can bring the peer back within the test.
    let mut driver_1 = memory_driver_builder()
        .with_static_peers(vec![static_peer])
        .with_static_peer_reconnect_delay(REDIAL_INTERVAL)
        .build()
        .unwrap()
        .start()
        .await
        .expect("listen");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    // Stop the peer, and wait until the node notices.
    drop(driver_2);
    poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;

    // Start it again, with the same identity and address.
    let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2)
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen on the same address");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A blocked static peer is skipped without a dial, but checks continue, so it is dialed again
/// once unblocked.
#[tokio::test]
async fn test_blocked_static_peer_is_skipped() {
    let (_, _, static_peer) = random_identity();
    let peer = static_peer.peer_id();
    // Starting dials the static peer, where nothing listens.
    let mut driver = driver_with_static_peer(static_peer).start().await.expect("listen");
    driver.block_peer(peer);
    poll_one_until(&mut driver, |driver| !driver.connection_gate.current_dials.contains(&peer))
        .await;
    let dialed = driver.connection_gate.dialed_addrs.len();

    // Skipped before `dial_peer`, which would count it as a `blocked_peer` error.
    assert!(dial_errors(|| driver.dial_static_peers()).is_empty());

    assert!(!driver.connection_gate.current_dials.contains(&peer));
    assert_eq!(driver.connection_gate.dialed_addrs.len(), dialed);
    assert!(driver.next_static_peer_check.is_some());

    // Once unblocked, the next check dials it.
    driver.unblock_peer(peer);
    driver.dial_static_peers();
    assert!(driver.connection_gate.current_dials.contains(&peer));
}

/// Blocking a connected static peer disconnects it, and it isn't dialed again, after the
/// reconnect delay or at a regular check, while its own dials are refused. Once unblocked, it
/// is dialed again at the next check.
#[tokio::test(start_paused = true)]
async fn test_blocked_static_peer_stays_disconnected_until_unblocked() {
    let (keypair_2, addr_2, static_peer) = random_identity();
    let peer_2 = static_peer.peer_id();
    let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2)
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");
    let mut driver_1 = driver_with_static_peer(static_peer).start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    let peer_1 = addr_1.peer_id();
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    driver_1.block_peer(peer_2);
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        !driver_1.is_connected(&peer_2) && !driver_2.is_connected(&peer_1)
    })
    .await;
    driver_2.dial_multiaddr(addr_1).expect("dial");
    let refusal = inbound_refusal(&mut driver_1, &mut driver_2).await;
    assert!(matches!(refusal, Some(DialError::PeerBlocked { .. })), "{refusal:?}");

    // Past the reconnect delay, and many regular checks.
    let wait = STATIC_PEER_RECONNECT_DELAY + 10 * REDIAL_INTERVAL;
    poll_for(&mut driver_1, &mut driver_2, wait).await;
    assert!(!driver_1.is_connected(&peer_2));
    assert!(!driver_1.connection_gate.current_dials.contains(&peer_2));

    driver_1.unblock_peer(peer_2);
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A block covers a static peer's configured IP address, but can't tell from a DNS name.
#[tokio::test]
async fn test_static_peer_addrs_covered_by_configured_ips() {
    let static_peer =
        |addr: &str| -> PeerAddr { format!("{addr}/p2p/{}", PeerId::random()).parse().unwrap() };
    let in_subnet = static_peer("/ip4/10.0.0.1/tcp/9222");
    let outside = static_peer("/ip4/192.168.1.5/tcp/9222");
    // Resolved from the hosts file, with no network.
    let by_name = static_peer("/dns4/localhost/tcp/9222");
    let driver = memory_driver_builder()
        .with_static_peers(vec![in_subnet.clone(), outside.clone(), by_name])
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");

    assert_eq!(
        driver.static_peer_addrs_in("10.0.0.0/8".parse().unwrap()),
        vec![(in_subnet.peer_id(), in_subnet.addr().clone())]
    );
    assert_eq!(
        driver.static_peer_addrs_at("192.168.1.5".parse().unwrap()),
        vec![(outside.peer_id(), outside.addr().clone())]
    );
}

/// A block covers a static peer's address exactly when the gate refuses it, including when
/// one of them is an IPv4-mapped IPv6 address. As in op-node, an IPv6 subnet such as `::/0`
/// doesn't cover an IPv4 address, even one written as a mapped address.
#[rstest]
#[case::mapped_addr_blocks_ipv4_peer("/ip4/10.0.0.1/tcp/9222", "::ffff:10.0.0.1", true)]
#[case::ipv4_addr_blocks_mapped_peer("/ip6/::ffff:10.0.0.1/tcp/9222", "10.0.0.1", true)]
#[case::other_addr("/ip4/10.0.0.1/tcp/9222", "10.0.0.2", false)]
#[tokio::test]
async fn test_blocked_addr_covers_the_static_peers_the_gate_refuses(
    #[case] static_addr: &str,
    #[case] blocked: IpAddr,
    #[case] covered: bool,
) {
    let static_peer: PeerAddr = format!("{static_addr}/p2p/{}", PeerId::random()).parse().unwrap();
    let mut driver = driver_with_static_peer(static_peer.clone()).start().await.expect("listen");
    driver.block_addr(blocked);

    assert_eq!(!driver.static_peer_addrs_at(blocked.into()).is_empty(), covered);
    assert_eq!(gate_refuses(&driver, &static_peer), covered);
}

/// See [`test_blocked_addr_covers_the_static_peers_the_gate_refuses`].
#[rstest]
#[case::mapped_subnet_blocks_ipv4_peer("/ip4/10.0.0.1/tcp/9222", "::ffff:10.0.0.0/104", true)]
#[case::ipv4_subnet_blocks_mapped_peer("/ip6/::ffff:10.0.0.1/tcp/9222", "10.0.0.0/8", true)]
#[case::all_ipv6_leaves_ipv4_peer("/ip4/10.0.0.1/tcp/9222", "::/0", false)]
#[case::all_ipv6_leaves_mapped_peer("/ip6/::ffff:10.0.0.1/tcp/9222", "::/0", false)]
#[tokio::test]
async fn test_blocked_subnet_covers_the_static_peers_the_gate_refuses(
    #[case] static_addr: &str,
    #[case] blocked: IpNet,
    #[case] covered: bool,
) {
    let static_peer: PeerAddr = format!("{static_addr}/p2p/{}", PeerId::random()).parse().unwrap();
    let mut driver = driver_with_static_peer(static_peer.clone()).start().await.expect("listen");
    driver.block_subnet(blocked);

    assert_eq!(!driver.static_peer_addrs_in(blocked.into()).is_empty(), covered);
    assert_eq!(gate_refuses(&driver, &static_peer), covered);
}

/// Returns whether the driver's gate refuses `peer`'s address.
fn gate_refuses(driver: &GossipDriver<ConnectionGater>, peer: &PeerAddr) -> bool {
    let ip = ConnectionGater::ip_from_addr(peer.addr()).expect("an IP address");
    driver.swarm.behaviour().gate.check_ip(ip).is_err()
}

/// A static peer configured with a DNS name is covered at the address of its open connection.
#[tokio::test]
async fn test_static_peer_addrs_covered_by_open_connections() {
    let mut driver_2 = tcp_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let port = addr_2.transport_addr().iter().find_map(|protocol| match protocol {
        Protocol::Tcp(port) => Some(port),
        _ => None,
    });
    let by_name: PeerAddr =
        format!("/dns4/localhost/tcp/{}/p2p/{}", port.unwrap(), addr_2.peer_id()).parse().unwrap();
    let mut driver_1 = crate::GossipDriverBuilder::new(
        kona_genesis::RollupConfig::default(),
        alloy_primitives::Address::ZERO,
        "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
        libp2p::identity::Keypair::generate_secp256k1(),
    )
    .with_static_peers(vec![by_name])
    .build()
    .expect("build")
    .start()
    .await
    .expect("listen");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    let covered = driver_1.static_peer_addrs_at("127.0.0.1".parse().unwrap());
    assert_eq!(covered.len(), 1, "{covered:?}");
    assert_eq!(covered[0].0, addr_2.peer_id());
}

/// Peers scoring below the ban threshold are banned, but a static peer never is.
#[tokio::test]
async fn test_peers_to_ban_skips_static_peers() {
    let (keypair_2, addr_2, static_peer) = random_identity();
    let static_peer_id = static_peer.peer_id();
    let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2)
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");
    let mut driver_3 = memory_driver().start().await.expect("listen");
    let peer_3 = *driver_3.local_peer_id();

    // Light scoring weighs the application score, which the test can set.
    let mut driver_1 = memory_driver_builder()
        .with_peer_scoring(kona_peers::PeerScoreLevel::Light)
        .with_peer_monitoring(Some(kona_peers::PeerMonitoring {
            ban_threshold: -100.0,
            ban_duration: Duration::from_secs(60),
        }))
        .with_static_peers(vec![static_peer])
        .build()
        .expect("build")
        .start()
        .await
        .expect("listen");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
    driver_1.dial_multiaddr(driver_3.peer_addr()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_3).await;
    assert!(driver_1.peers_to_ban().is_empty());

    let gossipsub = &mut driver_1.swarm.behaviour_mut().gossipsub;
    for peer_id in [static_peer_id, peer_3] {
        assert!(gossipsub.set_application_score(&peer_id, -1000.0), "{peer_id} is scored");
    }

    assert_eq!(driver_1.peers_to_ban(), vec![peer_3]);
}

/// The `UnprotectPeer` command leaves a static peer protected, but unprotects others.
#[tokio::test]
async fn test_unprotect_peer_keeps_static_peers_protected() {
    use crate::ConnectionGate;
    use std::collections::HashSet;

    let (_, _, static_peer) = random_identity();
    let static_peer_id = static_peer.peer_id();
    let mut driver = driver_with_static_peer(static_peer).start().await.expect("listen");
    let other_peer = PeerId::random();
    crate::GossipCommand::ProtectPeer { peer_id: other_peer }.handle(&mut driver);
    // Both are protected to begin with, so unprotecting has something to remove.
    let protected: HashSet<_> = driver.connection_gate.list_protected_peers().into_iter().collect();
    assert_eq!(protected, HashSet::from([static_peer_id, other_peer]));

    for peer_id in [static_peer_id, other_peer] {
        crate::GossipCommand::UnprotectPeer { peer_id }.handle(&mut driver);
    }

    assert_eq!(driver.connection_gate.list_protected_peers(), vec![static_peer_id]);
}
