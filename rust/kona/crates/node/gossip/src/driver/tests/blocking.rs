//! Tests for blocking peers, addresses and subnets, on inbound connections, open
//! connections and dials in flight.

use super::*;
use rstest::rstest;

/// A blocked peer can't connect to us: the gate refuses its connection as soon as the handshake
/// shows its peer ID, so the driver never sees it established. Once unblocked, it connects.
#[tokio::test]
async fn test_blocked_peer_cannot_connect_inbound() {
    let mut driver_1 = memory_driver().start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let (peer_1, peer_2) = (*driver_1.local_peer_id(), *driver_2.local_peer_id());
    driver_1.block_peer(peer_2);

    driver_2.dial_multiaddr(addr_1.clone()).expect("dial");
    let refusal = inbound_refusal(&mut driver_1, &mut driver_2).await;
    assert!(
        matches!(refusal, Some(DialError::PeerBlocked { peer_id }) if peer_id == peer_2),
        "{refusal:?}"
    );
    // A sanity check that neither side is left connected. We never were, and the blocked peer
    // saw the connection closed before `inbound_refusal` returned.
    assert!(!driver_1.is_connected(&peer_2));
    assert!(!driver_2.is_connected(&peer_1));

    driver_1.unblock_peer(peer_2);
    driver_2.dial_multiaddr(addr_1).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A blocked IP address or subnet can't connect to us: the gate refuses the connection before
/// the handshake, from its address alone. Once unblocked, it connects.
///
/// Over TCP on localhost, since an in-memory address has no IP to block.
#[rstest]
#[case::address(
    |driver: &mut GossipDriver<ConnectionGater>, block: bool| {
        let localhost = "127.0.0.1".parse().unwrap();
        if block { driver.block_addr(localhost) } else { driver.unblock_addr(localhost) }
    },
    "IP address is blocked: 127.0.0.1"
)]
#[case::subnet(
    |driver: &mut GossipDriver<ConnectionGater>, block: bool| {
        // The first 8 bits are the network, so this is every address from 127.0.0.0 to
        // 127.255.255.255: the whole IPv4 loopback range, which contains localhost.
        let loopback = "127.0.0.0/8".parse().unwrap();
        if block { driver.block_subnet(loopback) } else { driver.unblock_subnet(loopback) }
    },
    "IP address 127.0.0.1 is in a blocked subnet"
)]
#[tokio::test]
async fn test_blocked_address_cannot_connect_inbound_until_unblocked(
    #[case] set_blocked: fn(&mut GossipDriver<ConnectionGater>, bool),
    #[case] expected_refusal: &str,
) {
    let mut driver_1 = tcp_driver().start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    let mut driver_2 = tcp_driver().start().await.expect("listen");

    set_blocked(&mut driver_1, true);
    driver_2.dial_multiaddr(addr_1.clone()).expect("dial");
    let refusal = inbound_refusal(&mut driver_1, &mut driver_2).await;
    assert_eq!(refusal.map(|refusal| refusal.to_string()).as_deref(), Some(expected_refusal));

    set_blocked(&mut driver_1, false);
    driver_2.dial_multiaddr(addr_1).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A subnet block doesn't keep out an address outside the subnet.
#[tokio::test]
async fn test_address_outside_a_blocked_subnet_connects_inbound() {
    let mut driver_1 = tcp_driver().start().await.expect("listen");
    let mut driver_2 = tcp_driver().start().await.expect("listen");

    driver_1.block_subnet("10.0.0.0/8".parse().unwrap());
    driver_2.dial_multiaddr(driver_1.peer_addr()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// Blocking a connected peer closes its connection on both sides, and its redial is refused.
/// That holds for a protected peer too: protection only exempts a peer from the redial limit.
#[rstest]
#[case::unprotected(false)]
#[case::protected(true)]
#[tokio::test]
async fn test_blocking_a_connected_peer_closes_its_connection(#[case] protected: bool) {
    let mut driver_1 = memory_driver().start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let (peer_1, peer_2) = (*driver_1.local_peer_id(), *driver_2.local_peer_id());
    driver_2.dial_multiaddr(addr_1.clone()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
    if protected {
        driver_1.connection_gate.protect_peer(peer_2);
    }

    driver_1.block_peer(peer_2);
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        !driver_1.is_connected(&peer_2) && !driver_2.is_connected(&peer_1)
    })
    .await;

    driver_2.dial_multiaddr(addr_1).expect("dial");
    let refusal = inbound_refusal(&mut driver_1, &mut driver_2).await;
    assert!(
        matches!(refusal, Some(DialError::PeerBlocked { peer_id }) if peer_id == peer_2),
        "{refusal:?}"
    );
}

/// A dial of ours still in flight when its peer is blocked doesn't connect: blocking the peer
/// aborts it.
#[tokio::test]
async fn test_dial_in_flight_to_a_peer_blocked_meanwhile_does_not_connect() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    // Started, but not yet polled, so the block lands while the dial is in flight.
    driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    driver_1.block_peer(peer_2);

    let mut dial_error = None;
    let poll = async {
        while dial_error.is_none() {
            tokio::select! {
                Some(event) = driver_1.next() => {
                    match &event {
                        SwarmEvent::OutgoingConnectionError {
                            peer_id: Some(peer_id), error, ..
                        } if *peer_id == peer_2 => dial_error = Some(error.to_string()),
                        SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                            assert_ne!(*peer_id, peer_2, "the blocked peer connected");
                        }
                        _ => {}
                    }
                    driver_1.handle_event(event);
                }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("the dial ended");
    assert_eq!(dial_error.as_deref(), Some(libp2p::swarm::DialError::Aborted.to_string().as_str()));
    assert!(!driver_1.is_connected(&peer_2));
}

/// Two TCP drivers on localhost, connected by the first dialing the second if `outbound`, or
/// the second dialing the first if not.
async fn connected_tcp_drivers(
    outbound: bool,
) -> (GossipDriver<ConnectionGater>, GossipDriver<ConnectionGater>) {
    let mut driver_1 = tcp_driver().start().await.expect("listen");
    let mut driver_2 = tcp_driver().start().await.expect("listen");
    if outbound {
        driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    } else {
        driver_2.dial_multiaddr(driver_1.peer_addr()).expect("dial");
    }
    poll_until_connected(&mut driver_1, &mut driver_2).await;
    (driver_1, driver_2)
}

/// Blocking an IP address closes the connections to and from it, whichever side dialed.
#[rstest]
#[case::inbound(false)]
#[case::outbound(true)]
#[tokio::test]
async fn test_blocking_an_address_closes_its_connections(#[case] outbound: bool) {
    let (mut driver_1, mut driver_2) = connected_tcp_drivers(outbound).await;
    driver_1.block_addr("127.0.0.1".parse().unwrap());
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        driver_1.connections.is_empty() && driver_2.connections.is_empty()
    })
    .await;
}

/// Blocking a subnet closes the connections to and from the addresses in it, whichever side
/// dialed.
#[rstest]
#[case::inbound(false)]
#[case::outbound(true)]
#[tokio::test]
async fn test_blocking_a_subnet_closes_its_connections(#[case] outbound: bool) {
    let (mut driver_1, mut driver_2) = connected_tcp_drivers(outbound).await;
    driver_1.block_subnet("127.0.0.0/8".parse().unwrap());
    poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
        driver_1.connections.is_empty() && driver_2.connections.is_empty()
    })
    .await;
}

/// Blocking a subnet the peer's address isn't in leaves its connection open.
#[rstest]
#[case::inbound(false)]
#[case::outbound(true)]
#[tokio::test]
async fn test_blocking_another_subnet_leaves_connections_open(#[case] outbound: bool) {
    let (mut driver_1, mut driver_2) = connected_tcp_drivers(outbound).await;
    let (peer_1, peer_2) = (*driver_1.local_peer_id(), *driver_2.local_peer_id());
    driver_1.block_subnet("10.0.0.0/8".parse().unwrap());
    // `block_subnet` starts closing the connections it covers before it returns, and the tests
    // above see a loopback connection close within milliseconds. So if none has closed after
    // 200ms, none was closed. On a paused clock, the 200ms could pass before TCP delivers the
    // close, so this test runs on the real clock.
    poll_for(&mut driver_1, &mut driver_2, Duration::from_millis(200)).await;
    assert!(driver_1.is_connected(&peer_2) && driver_2.is_connected(&peer_1));
}

/// A dial of ours still in flight when the address it goes to is blocked doesn't connect: the
/// gate refuses the connection once it is established, before any other behaviour sees it.
#[tokio::test]
async fn test_dial_in_flight_to_an_address_blocked_meanwhile_does_not_connect() {
    let mut driver_2 = tcp_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = addr_2.peer_id();
    let mut driver_1 = tcp_driver().start().await.expect("listen");

    // Started, but not yet polled, so the block lands while the dial is in flight.
    driver_1.dial_multiaddr(addr_2).expect("dial");
    driver_1.block_addr("127.0.0.1".parse().unwrap());

    let mut refusal = None;
    let poll = async {
        while refusal.is_none() {
            tokio::select! {
                Some(event) = driver_1.next() => {
                    match &event {
                        SwarmEvent::OutgoingConnectionError { peer_id: Some(peer_id), error, .. }
                            if *peer_id == peer_2 =>
                        {
                            let libp2p::swarm::DialError::Denied { cause } = error else {
                                panic!("not refused by the gate: {error}");
                            };
                            refusal = cause.downcast_ref::<DialError>().cloned();
                        }
                        SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                            assert_ne!(*peer_id, peer_2, "the blocked address connected");
                        }
                        _ => {}
                    }
                    driver_1.handle_event(event);
                }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("the dial ended");
    assert!(matches!(refusal, Some(DialError::AddressBlocked { .. })), "{refusal:?}");
}

/// A connect whose dial is in flight when the address is blocked fails with the block, as one
/// made after the block does, rather than with the swarm's message.
#[tokio::test]
async fn test_connect_to_an_address_blocked_during_the_dial_reports_the_block() {
    let mut driver_2 = tcp_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = addr_2.peer_id();
    let mut driver_1 = tcp_driver().start().await.expect("listen");

    // Not polled in between, so the block lands while the dial is in flight.
    let (reply, outcome) = oneshot::channel();
    driver_1.connect_peer(addr_2, reply);
    driver_1.block_addr("127.0.0.1".parse().unwrap());

    poll_until(&mut driver_1, &mut driver_2, |driver_1, _| {
        !driver_1.pending_connects.contains_key(&peer_2)
    })
    .await;
    let outcome = outcome.await.expect("replied");
    assert!(
        matches!(outcome, Err(ConnectPeerError::Refused(DialError::AddressBlocked { .. }))),
        "{outcome:?}"
    );
}

/// An in-memory connection has no IP, so even a block on every IPv4 address doesn't keep it
/// out, as in go-libp2p's basic gater.
#[tokio::test]
async fn test_connection_without_an_ip_is_not_blocked_by_a_subnet() {
    let mut driver_1 = memory_driver().start().await.expect("listen");
    let mut driver_2 = memory_driver().start().await.expect("listen");
    driver_1.block_subnet("0.0.0.0/0".parse().unwrap());

    driver_2.dial_multiaddr(driver_1.peer_addr()).expect("dial");
    assert_eq!(inbound_refusal(&mut driver_1, &mut driver_2).await.map(|e| e.to_string()), None);
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}
