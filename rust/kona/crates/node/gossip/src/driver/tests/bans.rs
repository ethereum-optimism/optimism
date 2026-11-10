//! Tests for banning peers, and for lifting bans.

use super::*;
use rstest::rstest;

/// How long the drivers in the ban tests ban a peer for.
const BAN_DURATION: Duration = Duration::from_secs(60);

/// A memory driver that bans peers for [`BAN_DURATION`].
fn banning_driver() -> GossipDriverStarter<ConnectionGater> {
    let monitoring = PeerMonitoring { ban_threshold: -100.0, ban_duration: BAN_DURATION };
    memory_driver_builder().with_peer_monitoring(Some(monitoring)).build().expect("build")
}

/// A peer banned for its score can't connect, whichever side dials, until the ban expires.
#[rstest]
#[case::we_dial(true)]
#[case::it_dials(false)]
#[tokio::test(start_paused = true)]
async fn test_banned_peer_is_refused_until_the_ban_expires(#[case] we_dial: bool) {
    let mut driver_1 = banning_driver().start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let (peer_1, peer_2) = (addr_1.peer_id(), addr_2.peer_id());
    driver_2.dial_multiaddr(addr_1.clone()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    driver_1.ban_peer(peer_2);
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        !driver_1.is_connected(&peer_2) && !driver_2.is_connected(&peer_1)
    })
    .await;
    if we_dial {
        let refused = driver_1.dial_multiaddr(addr_2.clone());
        assert!(matches!(refused, Err(DialError::PeerBanned { .. })), "{refused:?}");
    } else {
        driver_2.dial_multiaddr(addr_1.clone()).expect("dial");
        let refusal = inbound_refusal(&mut driver_1, &mut driver_2).await;
        assert!(matches!(refusal, Some(DialError::PeerBanned { .. })), "{refusal:?}");
    }

    tokio::time::advance(BAN_DURATION).await;
    if we_dial {
        driver_1.dial_multiaddr(addr_2).expect("dial");
    } else {
        driver_2.dial_multiaddr(addr_1).expect("dial");
    }
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A ban duration too long to add to the clock bans the peer for as long as the clock goes,
/// rather than panicking when the first peer is banned.
#[tokio::test]
async fn test_ban_duration_beyond_the_clock_does_not_panic() {
    let monitoring = PeerMonitoring { ban_threshold: -100.0, ban_duration: Duration::MAX };
    let starter =
        memory_driver_builder().with_peer_monitoring(Some(monitoring)).build().expect("build");
    let mut driver = starter.start().await.expect("listen");
    let peer = PeerId::random();

    driver.ban_peer(peer);
    assert!(driver.swarm.behaviour().gate.is_banned_peer(&peer));
}

/// Unblocking a peer lifts its ban too, and a ban isn't listed as a block, as in op-node.
#[tokio::test(start_paused = true)]
async fn test_unblocking_a_peer_lifts_its_ban() {
    let mut driver = banning_driver().start().await.expect("listen");
    let peer = PeerId::random();

    driver.ban_peer(peer);
    assert!(driver.list_blocked_peers().is_empty());
    let refused = driver.dial_multiaddr(unused_memory_addr(peer));
    assert!(matches!(refused, Err(DialError::PeerBanned { .. })), "{refused:?}");

    driver.unblock_peer(peer);
    driver.dial_multiaddr(unused_memory_addr(peer)).expect("dial");
}

/// Banning a peer disconnects it, and the connection is cleaned up once it closes. Its identify
/// info stays, as for any known peer that isn't connected.
// TODO(#23800): prune the peerstore once a peer has been disconnected for a while.
#[tokio::test]
async fn test_ban_peer() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    // Wait for identify, so there is identify info to keep.
    poll_until(&mut driver_1, &mut driver_2, |driver_1, _| {
        driver_1.peerstore.contains_key(&peer_2)
    })
    .await;

    driver_1.ban_peer(peer_2);

    poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;
    assert!(!driver_1.peer_connection_start.contains_key(&peer_2));
    assert!(driver_1.peerstore.contains_key(&peer_2));
}
