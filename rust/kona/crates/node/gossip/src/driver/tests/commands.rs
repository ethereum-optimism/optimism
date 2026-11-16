//! Tests for the replies to connect and disconnect commands.

use super::*;
use rstest::rstest;

/// Connecting to a peer just blocked is told it is connected while its connection closes, and
/// is refused with the block once it has closed.
#[tokio::test]
async fn test_connect_to_a_peer_just_blocked_is_refused_once_disconnected() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = addr_2.peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");
    driver_1.dial_multiaddr(addr_2.clone()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    // Not polled since the block, so the connection is still open.
    driver_1.block_peer(peer_2);
    let (reply, outcome) = oneshot::channel();
    driver_1.connect_peer(addr_2.clone(), reply);
    assert!(matches!(outcome.await, Ok(Ok(()))));

    poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;
    let (reply, outcome) = oneshot::channel();
    driver_1.connect_peer(addr_2, reply);
    let refusal = outcome.await.expect("replied at once");
    assert!(
        matches!(refusal, Err(ConnectPeerError::Refused(DialError::PeerBlocked { .. }))),
        "{refusal:?}"
    );
}

/// A reply whose caller has given up, such as an RPC call that timed out, is dropped when the
/// next reply for the same peer is added, rather than kept until the peer's dial ends.
#[tokio::test]
async fn test_abandoned_connect_replies_are_dropped() {
    let mut driver = memory_driver().start().await.expect("listen");
    let peer = PeerId::random();

    // Not polled, so the dial stays in flight, and the second call joins it.
    let (abandoned, _) = oneshot::channel();
    let addr = PeerAddr::try_from(unused_memory_addr(peer)).unwrap();
    driver.connect_peer(addr.clone(), abandoned);
    let (waiting, _waiting_rx) = oneshot::channel();
    driver.connect_peer(addr, waiting);

    assert_eq!(driver.pending_connects[&peer].len(), 1);
}

/// Two memory drivers, the first connected to the second by dialing it.
async fn connected_memory_drivers()
-> (GossipDriver<ConnectionGater>, GossipDriver<ConnectionGater>, PeerAddr) {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let mut driver_1 = memory_driver().start().await.expect("listen");
    driver_1.dial_multiaddr(addr_2.clone()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
    (driver_1, driver_2, addr_2)
}

/// Connecting to a peer that is already connected succeeds at once, without a dial.
#[tokio::test]
async fn test_connect_to_a_connected_peer_replies_at_once() {
    let (mut driver_1, _driver_2, addr_2) = connected_memory_drivers().await;
    let peer_2 = addr_2.peer_id();

    let (reply, mut outcome) = oneshot::channel();
    driver_1.connect_peer(addr_2, reply);
    assert!(matches!(outcome.try_recv(), Ok(Ok(()))));
    assert!(!driver_1.pending_connects.contains_key(&peer_2));
}

/// Callers connecting to a peer that is already being dialed join that dial, and are each told
/// how it ends.
#[rstest]
#[case::connects(true)]
#[case::fails(false)]
#[tokio::test]
async fn test_connects_joining_a_dial_are_each_told_how_it_ends(#[case] reachable: bool) {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = addr_2.peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");
    let addr =
        if reachable { addr_2 } else { PeerAddr::try_from(unused_memory_addr(peer_2)).unwrap() };

    // Not polled in between, so the second call finds the first one's dial in flight.
    let (replies, outcomes): (Vec<_>, Vec<_>) = (0..2).map(|_| oneshot::channel()).unzip();
    for reply in replies {
        driver_1.connect_peer(addr.clone(), reply);
    }
    assert_eq!(driver_1.pending_connects[&peer_2].len(), 2);

    poll_until(&mut driver_1, &mut driver_2, |driver_1, _| {
        !driver_1.pending_connects.contains_key(&peer_2)
    })
    .await;
    for outcome in outcomes {
        let outcome = outcome.await.expect("replied");
        if reachable {
            assert!(outcome.is_ok(), "{outcome:?}");
        } else {
            assert!(matches!(outcome, Err(ConnectPeerError::DialFailed(_))), "{outcome:?}");
        }
    }
}

/// Our dial failing after the peer has connected anyway, as when both dial at once, tells the
/// callers waiting on it that the peer is connected.
#[tokio::test]
async fn test_failed_dial_to_a_connected_peer_is_a_success() {
    let (mut driver_1, _driver_2, addr_2) = connected_memory_drivers().await;
    let peer_2 = addr_2.peer_id();

    let (reply, outcome) = oneshot::channel();
    add_reply(&mut driver_1.pending_connects, peer_2, reply);
    driver_1.handle_event(SwarmEvent::OutgoingConnectionError {
        connection_id: ConnectionId::new_unchecked(0),
        peer_id: Some(peer_2),
        error: libp2p::swarm::DialError::Aborted,
    });
    let outcome = outcome.await.expect("replied");
    assert!(outcome.is_ok(), "{outcome:?}");
}

/// Disconnecting a peer that isn't connected succeeds at once.
#[tokio::test]
async fn test_disconnect_from_a_peer_not_connected_replies_at_once() {
    let mut driver = memory_driver().start().await.expect("listen");
    let (reply, mut outcome) = oneshot::channel();
    driver.disconnect_peer_with_reply(PeerId::random(), reply);
    assert!(outcome.try_recv().is_ok());
    assert!(driver.pending_disconnects.is_empty());
}

/// A disconnect is told only once the last of the peer's connections has closed.
#[tokio::test]
async fn test_disconnect_replies_once_the_last_connection_closes() {
    let mut driver_1 = memory_driver();
    let mut driver_2 = memory_driver();
    driver_1.config.duplicate_connection_grace = Duration::from_secs(60);
    driver_2.config.duplicate_connection_grace = Duration::from_secs(60);
    let (mut driver_1, mut driver_2) = cross_dial(driver_1, driver_2).await;
    let peer_2 = *driver_2.local_peer_id();
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        num_connections(driver_1) == 2 && num_connections(driver_2) == 2
    })
    .await;

    let (reply, mut outcome) = oneshot::channel();
    driver_1.disconnect_peer_with_reply(peer_2, reply);
    // Whether the reply was sent after each closed connection, by the connections still open.
    let mut closed = Vec::new();
    let poll = async {
        while closed.last().is_none_or(|(num_established, _)| *num_established > 0) {
            tokio::select! {
                Some(event) = driver_1.next() => {
                    let num_established = match &event {
                        SwarmEvent::ConnectionClosed { peer_id, num_established, .. }
                            if *peer_id == peer_2 => Some(*num_established),
                        _ => None,
                    };
                    driver_1.handle_event(event);
                    if let Some(num_established) = num_established {
                        closed.push((num_established, outcome.try_recv().is_ok()));
                    }
                }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("disconnected");
    assert_eq!(closed, vec![(1, false), (0, true)]);
}

/// Disconnecting a peer with the `DisconnectPeer` command records the connection's duration
/// when the connection closes, not when the command runs.
///
/// Pings record the duration so far too, so the test doesn't count samples. It checks the start
/// time instead: recording the duration at close also removes it.
#[tokio::test]
async fn test_disconnect_peer_records_connection_duration_at_close() {
    let recorder = metrics_util::debugging::DebuggingRecorder::new();
    let snapshotter = recorder.snapshotter();
    // For the whole test thread, so that it sees what polling the swarm records too.
    let _recorder = metrics::set_default_local_recorder(&recorder);
    // Each snapshot empties the histograms, so this returns the durations since the last one.
    let durations = || -> usize {
        snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(key, ..)| {
                key.key().name() == crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
            })
            .map(|(.., value)| match value {
                metrics_util::debugging::DebugValue::Histogram(values) => values.len(),
                _ => 0,
            })
            .sum()
    };

    let mut driver_2 = memory_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");
    driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
    durations();

    // The command only starts the disconnect: the duration isn't recorded yet.
    let reply = oneshot::channel().0;
    crate::GossipCommand::DisconnectPeer { peer_id: peer_2, reply }.handle(&mut driver_1);
    assert!(driver_1.peer_connection_start.contains_key(&peer_2));

    // The close records it, and removes the start time, after which pings record nothing.
    poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;
    assert!(!driver_1.peer_connection_start.contains_key(&peer_2));
    assert!(durations() >= 1);
}
