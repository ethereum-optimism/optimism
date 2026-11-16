//! Tests for the [`GossipDriver`], one file per topic, with the helpers they share.

use super::*;
use crate::builder::tests::{memory_driver, memory_driver_builder, poll_until_connected};
use libp2p::{multiaddr::Protocol, swarm::ListenError};

mod bans;
mod blocking;
mod commands;
mod dial;
mod duplicates;
mod signer;
mod start;
mod static_peers;

/// Polls both drivers' swarms until `done` holds for them, for at most 10 seconds.
async fn poll_until(
    driver_1: &mut GossipDriver<ConnectionGater>,
    driver_2: &mut GossipDriver<ConnectionGater>,
    done: impl Fn(&GossipDriver<ConnectionGater>, &GossipDriver<ConnectionGater>) -> bool,
) {
    let poll = async {
        while !done(driver_1, driver_2) {
            tokio::select! {
                Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("condition reached");
}

/// Polls both drivers' swarms for `duration`, for a test that checks that nothing happens
/// meanwhile.
async fn poll_for(
    driver_1: &mut GossipDriver<ConnectionGater>,
    driver_2: &mut GossipDriver<ConnectionGater>,
    duration: Duration,
) {
    let poll = async {
        loop {
            tokio::select! {
                Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    };
    // Polling only ends when the time is up.
    let _ = tokio::time::timeout(duration, poll).await;
}

/// Polls one driver's swarm until `done` holds for it, for at most 10 seconds.
async fn poll_one_until(
    driver: &mut GossipDriver<ConnectionGater>,
    done: impl Fn(&GossipDriver<ConnectionGater>) -> bool,
) {
    let poll = async {
        while !done(driver) {
            if let Some(event) = driver.next().await {
                driver.handle_event(event);
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("condition reached");
}

/// Returns the number of connections the driver has open.
fn num_connections(driver: &GossipDriver<ConnectionGater>) -> u32 {
    driver.swarm.network_info().connection_counters().num_established()
}

/// Starts both drivers and has them dial each other at the same time.
async fn cross_dial(
    starter_1: GossipDriverStarter<ConnectionGater>,
    starter_2: GossipDriverStarter<ConnectionGater>,
) -> (GossipDriver<ConnectionGater>, GossipDriver<ConnectionGater>) {
    let mut driver_1 = starter_1.start().await.expect("listen");
    let mut driver_2 = starter_2.start().await.expect("listen");
    driver_1.dial_multiaddr(driver_2.peer_addr()).expect("dial");
    driver_2.dial_multiaddr(driver_1.peer_addr()).expect("dial");
    (driver_1, driver_2)
}

/// Builds a gossip driver on TCP, listening on a free port on localhost once started.
fn tcp_driver() -> GossipDriverStarter<ConnectionGater> {
    let mut driver = crate::GossipDriverBuilder::new(
        kona_genesis::RollupConfig::default(),
        alloy_primitives::Address::ZERO,
        "/ip4/127.0.0.1/tcp/0".parse().expect("valid multiaddr"),
        libp2p::identity::Keypair::generate_secp256k1(),
    )
    .build()
    .expect("build gossip driver");
    driver.config.duplicate_connection_grace = Duration::from_millis(100);
    driver
}

/// Runs `f` and returns the dial errors it recorded: for each `type` label of
/// `kona_node_dial_peer_error`, how many times it went up, in no particular order.
fn dial_errors(f: impl FnOnce()) -> Vec<(String, usize)> {
    let recorder = metrics_util::debugging::DebuggingRecorder::new();
    let snapshotter = recorder.snapshotter();
    metrics::with_local_recorder(&recorder, f);
    snapshotter
        .snapshot()
        .into_vec()
        .into_iter()
        .filter(|(key, ..)| key.key().name() == crate::Metrics::DIAL_PEER_ERROR)
        .filter_map(|(key, _, _, value)| {
            let kind = key.key().labels().find(|l| l.key() == "type")?.value().to_owned();
            let metrics_util::debugging::DebugValue::Counter(count) = value else {
                return None;
            };
            Some((kind, count as usize))
        })
        .collect()
}

/// An address no one listens on, so a dial to it fails.
fn unused_memory_addr(peer: PeerId) -> Multiaddr {
    Multiaddr::empty().with(Protocol::Memory(u64::MAX)).with(Protocol::P2p(peer))
}

/// Polls both drivers, after `dialer` dialed `listener`, until the listener has refused the
/// connection or seen it established, and the dialer's dial has ended, so that it may dial
/// again. After a refusal, that includes the dialer seeing the refused connection closed.
/// Returns the gate's refusal, or `None` if the listener accepted the connection.
async fn inbound_refusal(
    listener: &mut GossipDriver<ConnectionGater>,
    dialer: &mut GossipDriver<ConnectionGater>,
) -> Option<DialError> {
    let (listener_peer, dialer_peer) = (*listener.local_peer_id(), *dialer.local_peer_id());
    let mut outcome = None;
    // A peer refused after the handshake saw the connection established on its side, and stays
    // connected until it notices that the listener dropped it, which takes polling both.
    let pending = |outcome: &Option<Option<DialError>>, dialer: &GossipDriver<ConnectionGater>| {
        outcome.is_none() ||
            dialer.connection_gate.current_dials.contains(&listener_peer) ||
            matches!(outcome, Some(Some(_))) && dialer.is_connected(&listener_peer)
    };
    let poll = async {
        while pending(&outcome, dialer) {
            tokio::select! {
                Some(event) = listener.next() => {
                    match &event {
                        SwarmEvent::IncomingConnectionError {
                            error: ListenError::Denied { cause }, ..
                        } => {
                            let refusal = cause.downcast_ref::<DialError>();
                            outcome = Some(Some(refusal.expect("a gate refusal").clone()));
                        }
                        SwarmEvent::ConnectionEstablished { peer_id, .. }
                            if *peer_id == dialer_peer =>
                        {
                            outcome = Some(None);
                        }
                        _ => {}
                    }
                    listener.handle_event(event);
                }
                Some(event) = dialer.next() => { dialer.handle_event(event); }
            }
        }
    };
    tokio::time::timeout(Duration::from_secs(10), poll).await.expect("refused or accepted");
    outcome.flatten()
}
