//! Tests for the peers a driver reports it reached, over the connections it dialed.

use super::*;
use kona_genesis::RollupConfig;

/// A driver whose swarm uses TCP on loopback, listening on a free port once started. The memory
/// transport has no TCP address for [`GossipDriver::reached_peer`] to report.
fn tcp_driver() -> GossipDriverStarter<ConnectionGater> {
    let rollup_config = RollupConfig { block_time: 2, ..Default::default() };
    let addr = Multiaddr::empty()
        .with(Protocol::Ip4(std::net::Ipv4Addr::LOCALHOST))
        .with(Protocol::Tcp(0));
    GossipDriverBuilder::new(rollup_config, Address::ZERO, addr, Keypair::generate_secp256k1())
        .build()
        .expect("build gossip driver")
}

/// What an event showed a driver reached.
#[derive(std::fmt::Debug, PartialEq, Eq)]
enum Reached {
    /// A connection it dialed was established.
    Connection(PeerId, SocketAddr),
    /// A ping succeeded on a connection it dialed.
    Ping(PeerId, SocketAddr),
}

/// Returns what `event` shows `driver` reached, if anything.
fn reached(driver: &GossipDriver<ConnectionGater>, event: &SwarmEvent<Event>) -> Option<Reached> {
    let (peer_id, addr) = driver.reached_peer(event)?;
    Some(match event {
        SwarmEvent::Behaviour(Event::Ping(_)) => Reached::Ping(peer_id, addr),
        _ => Reached::Connection(peer_id, addr),
    })
}

/// Only a connection a node dialed shows that it can reach the peer at the peer's address: the
/// connection being established, and each successful ping on it. The node that accepted the
/// connection reports nothing, though its pings on it succeed too. A dialer's address can even be
/// its listen address, as op-node dials from its listen port, which only shows that the dialer
/// can reach the node it dialed.
///
/// Both nodes ping as soon as the connection is up, so once each has had a ping, both have had the
/// chance to report one.
#[tokio::test]
async fn test_only_the_dialer_reports_the_peer_reached() {
    let mut dialer = tcp_driver().start().await.expect("listen");
    let mut listener = tcp_driver().start().await.expect("listen");
    let (dialer_id, listener_id) = (*dialer.local_peer_id(), *listener.local_peer_id());
    let listener_addr = tcp_socket(listener.addr()).expect("the listener is on TCP");
    dialer.dial_multiaddr(listener.peer_addr()).expect("dial");

    let mut dialer_reached = Vec::new();
    let mut listener_reached = Vec::new();
    let poll = async {
        while !dialer.ping.contains_key(&listener_id) || !listener.ping.contains_key(&dialer_id) {
            tokio::select! {
                Some(event) = dialer.next() => {
                    dialer_reached.extend(reached(&dialer, &event));
                    dialer.handle_event(event);
                }
                Some(event) = listener.next() => {
                    listener_reached.extend(reached(&listener, &event));
                    listener.handle_event(event);
                }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("both nodes ping");

    assert_eq!(
        dialer_reached,
        [
            Reached::Connection(listener_id, listener_addr),
            Reached::Ping(listener_id, listener_addr)
        ]
    );
    assert_eq!(
        listener_reached,
        [],
        "the node that accepted the connection reported reaching the dialer"
    );
}
