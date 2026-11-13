//! Tests for dialing peers: DNS resolution, the gate, and when a dial ends.

use super::*;

/// `dial_peer` resolves a DNS name before the gate, so it checks and dials the IP the name
/// resolves to, rather than leaving libp2p to resolve the name again.
#[tokio::test]
async fn test_dial_peer_resolves_dns_names_before_the_gate() {
    let mut driver_2 = tcp_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();
    let port = driver_2
        .peer_addr()
        .addr()
        .iter()
        .find_map(|protocol| match protocol {
            Protocol::Tcp(port) => Some(port),
            _ => None,
        })
        .unwrap();
    let dns_addr: Multiaddr = format!("/dns4/localhost/tcp/{port}/p2p/{peer_2}").parse().unwrap();
    let ip_addr: Multiaddr = format!("/ip4/127.0.0.1/tcp/{port}/p2p/{peer_2}").parse().unwrap();
    let mut driver_1 = tcp_driver().start().await.expect("listen");

    // `localhost` resolves to a blocked IP, so nothing is dialed.
    let localhost = "127.0.0.1".parse().unwrap();
    driver_1.block_addr(localhost);
    let refused = driver_1.dial_peer(peer_2, [dns_addr.clone()]);
    assert!(matches!(refused, Err(DialError::AddressBlocked { .. })), "{refused:?}");
    assert!(!driver_1.connection_gate.current_dials.contains(&peer_2));

    // Unblocked, the resolved address is what's dialed and counted.
    driver_1.unblock_addr(localhost);
    driver_1.dial_peer(peer_2, [dns_addr]).expect("dial");
    let dialed: Vec<_> = driver_1.connection_gate.dialed_addrs.keys().cloned().collect();
    assert_eq!(dialed, vec![ip_addr]);
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// Dialing a connected peer is skipped before the gate, and recorded as an `already_connected`
/// dial error without counting as a dial.
#[tokio::test]
async fn test_dial_skips_connected_peer() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    // The one dial that connects the peers.
    driver_1.dial_multiaddr(addr_2.clone()).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    // Dial the connected peer again, as a caller that doesn't check `is_connected` would, for
    // example `opp2p_connectPeer` for a peer that is already connected. Discovery handing
    // over the peer's ENR again used to do this too, before the network actor checked.
    let redials = 3;
    let dial_errors = dial_errors(|| {
        for _ in 0..redials {
            let redial = driver_1.dial_multiaddr(addr_2.clone());
            assert!(matches!(redial, Err(DialError::AlreadyConnected { .. })), "{redial:?}");
        }
    });

    // Each redial is turned away by the connected check in `dial_peer`: one `already_connected`
    // error per redial. Without the check, the gate would let them through, and libp2p would
    // refuse to dial a connected peer, which shows up as a `connection_error` instead.
    assert_eq!(dial_errors, vec![("already_connected".to_owned(), redials)]);
    // Only the first dial counts against the address's redial limit.
    assert_eq!(driver_1.connection_gate.dialed_addrs[addr_2.addr()].num_dials, 1);
}

/// A peer dialed at several addresses connects through the one that works, in one dial that
/// counts against each address.
#[tokio::test]
async fn test_dial_peer_connects_at_any_address() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    driver_1.dial_peer(peer_2, [unused_memory_addr(peer_2), addr_2.addr().clone()]).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    assert!(driver_1.is_connected(&peer_2));
    let gate = &driver_1.connection_gate;
    assert_eq!(gate.dialed_addrs[&unused_memory_addr(peer_2)].num_dials, 1);
    assert_eq!(gate.dialed_addrs[addr_2.addr()].num_dials, 1);
}

/// Addresses the gate refuses, or that are for another peer, are left out of the dial.
#[tokio::test]
async fn test_dial_peer_leaves_out_refused_addresses() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let addr_2 = driver_2.peer_addr();
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    let blocked: Multiaddr = format!("/ip4/10.0.0.1/tcp/9222/p2p/{peer_2}").parse().unwrap();
    driver_1.block_addr("10.0.0.1".parse().unwrap());
    let other_peer = unused_memory_addr(PeerId::random());

    driver_1.dial_peer(peer_2, [blocked, other_peer, addr_2.addr().clone()]).expect("dial");

    let dialed: Vec<_> = driver_1.connection_gate.dialed_addrs.keys().cloned().collect();
    assert_eq!(dialed, vec![addr_2.addr().clone()]);
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// Nothing is dialed when the gate refuses every address.
#[tokio::test]
async fn test_dial_peer_skips_peer_with_no_allowed_address() {
    let mut driver = memory_driver().start().await.expect("listen");
    let peer = PeerId::random();
    driver.block_peer(peer);

    let refused = driver.dial_peer(peer, [unused_memory_addr(peer)]);
    assert!(matches!(refused, Err(DialError::PeerBlocked { .. })), "{refused:?}");

    assert!(!driver.connection_gate.current_dials.contains(&peer));
    assert!(driver.connection_gate.dialed_addrs.is_empty());
}

/// When every address fails, the dial ends once, so the peer can be dialed again.
#[tokio::test]
async fn test_dial_peer_failing_at_every_address_is_no_longer_in_flight() {
    let mut driver = memory_driver().start().await.expect("listen");
    let peer = PeerId::random();
    let unused_2 =
        Multiaddr::empty().with(Protocol::Memory(u64::MAX - 1)).with(Protocol::P2p(peer));

    driver.dial_peer(peer, [unused_memory_addr(peer), unused_2]).expect("dial");
    assert!(driver.connection_gate.current_dials.contains(&peer));

    poll_one_until(&mut driver, |driver| !driver.connection_gate.current_dials.contains(&peer))
        .await;
}

/// A dial that connects is no longer in flight in the gate.
#[tokio::test]
async fn test_connected_dial_is_no_longer_in_flight() {
    let mut driver_2 = memory_driver().start().await.expect("listen");
    let peer_2 = *driver_2.local_peer_id();
    let mut driver_1 = memory_driver().start().await.expect("listen");

    driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    assert!(driver_1.connection_gate.current_dials.contains(&peer_2));
    poll_until_connected(&mut driver_1, &mut driver_2).await;

    assert!(!driver_1.connection_gate.current_dials.contains(&peer_2));
}
