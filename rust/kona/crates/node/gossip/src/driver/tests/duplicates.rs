//! Tests for closing duplicate connections to a peer.

use super::*;

/// Returns whether the driver knows that `peer` is subscribed to some topic.
fn knows_subscriptions(driver: &GossipDriver<ConnectionGater>, peer: PeerId) -> bool {
    driver
        .swarm
        .behaviour()
        .gossipsub
        .all_peers()
        .any(|(id, topics)| *id == peer && !topics.is_empty())
}

/// Two peers that dial each other at the same time end up with one connection, the one dialed
/// by the peer with the smaller ID, and each knows the other's subscriptions.
#[tokio::test]
async fn test_crossing_dials_keep_one_connection() {
    let mut driver_1 = memory_driver();
    let mut driver_2 = memory_driver();
    driver_1.config.duplicate_connection_grace = Duration::from_millis(100);
    driver_2.config.duplicate_connection_grace = Duration::from_millis(100);

    let (mut driver_1, mut driver_2) = cross_dial(driver_1, driver_2).await;
    let (peer_1, peer_2) = (*driver_1.local_peer_id(), *driver_2.local_peer_id());
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        num_connections(driver_1) == 1 &&
            num_connections(driver_2) == 1 &&
            knows_subscriptions(driver_1, peer_2) &&
            knows_subscriptions(driver_2, peer_1)
    })
    .await;

    let preferred_dialer = peer_1.min(peer_2);
    for (driver, peer) in [(&driver_1, peer_2), (&driver_2, peer_1)] {
        let dialers: Vec<_> =
            driver.connections[&peer].values().map(|connection| connection.dialer).collect();
        assert_eq!(dialers, vec![preferred_dialer]);
    }
}

/// A duplicate connection stays open for its grace period, so that gossipsub's subscriptions,
/// which it sends only on the first connection, are not lost with it. It closes once the
/// grace is over.
#[tokio::test(start_paused = true)]
async fn test_duplicate_connection_stays_open_during_grace() {
    // Within `poll_until`'s timeout, which the paused clock also counts.
    let grace = Duration::from_secs(5);
    let mut driver_1 = memory_driver();
    let mut driver_2 = memory_driver();
    driver_1.config.duplicate_connection_grace = grace;
    driver_2.config.duplicate_connection_grace = grace;

    let start = Instant::now();
    let (mut driver_1, mut driver_2) = cross_dial(driver_1, driver_2).await;
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        num_connections(driver_1) == 2 && num_connections(driver_2) == 2
    })
    .await;

    // The grace runs from when the duplicate opened, which is after `start`.
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        num_connections(driver_1) == 1 && num_connections(driver_2) == 1
    })
    .await;
    assert!(start.elapsed() >= grace, "closed after {:?}", start.elapsed());
}

/// Peers dialing each other over TCP at the same time connect. Dialing from the listen port,
/// both dials would share one TCP connection and fail the handshake, which happens to about
/// one crossing in five, so ten crossings catch it most of the time.
#[tokio::test]
async fn test_tcp_crossing_dials_connect() {
    for _ in 0..10 {
        let (mut driver_1, mut driver_2) = cross_dial(tcp_driver(), tcp_driver()).await;
        poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
            num_connections(driver_1) == 1 && num_connections(driver_2) == 1
        })
        .await;
    }
}
