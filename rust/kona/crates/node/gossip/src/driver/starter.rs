//! The [`GossipDriverStarter`], which starts a [`GossipDriver`].

use derive_more::Debug;
use futures::stream::StreamExt;
use libp2p::{Multiaddr, PeerId, Swarm, swarm::SwarmEvent};
use std::collections::HashMap;

use crate::{
    Behaviour, BlockHandler, ConnectionGate, GossipDriver, GossipDriverConfig, StartError,
};

/// A gossip swarm and the settings to run it with, not listening yet. [`Self::start`] consumes it
/// and returns the running [`GossipDriver`], so a driver starts only once.
///
/// [`GossipDriverBuilder::build`](crate::GossipDriverBuilder::build) makes one.
#[derive(Debug)]
pub struct GossipDriverStarter<G: ConnectionGate> {
    /// The [`Swarm`] instance.
    #[debug(skip)]
    pub(crate) swarm: Swarm<Behaviour>,
    /// The address to listen on.
    pub(crate) addr: Multiaddr,
    /// The [`BlockHandler`].
    pub(crate) handler: BlockHandler,
    /// The connection gate.
    pub(crate) connection_gate: G,
    /// The addresses of each peer to dial at startup and keep connected, without the node's own
    /// ID.
    pub(crate) static_peers: HashMap<PeerId, Vec<Multiaddr>>,
    /// How the driver behaves once it runs.
    pub(crate) config: GossipDriverConfig,
}

impl<G: ConnectionGate> GossipDriverStarter<G> {
    /// Returns the address the driver will listen on.
    pub const fn addr(&self) -> &Multiaddr {
        &self.addr
    }

    /// Returns the [`BlockHandler`] the driver will validate blocks with.
    pub const fn handler(&self) -> &BlockHandler {
        &self.handler
    }

    /// Starts the libp2p Swarm.
    ///
    /// - Tells the swarm to listen on the given [`Multiaddr`], and waits until it does.
    /// - Dials the static peers.
    ///
    /// The dials only start once the swarm listens, so a start that fails leaves nothing running.
    ///
    /// Returns the running driver. Its [`GossipDriver::peer_addr`] is where peers can dial this
    /// node: one interface and port, where the configured address may have been `0.0.0.0` or port
    /// 0.
    pub async fn start(self) -> Result<GossipDriver<G>, StartError> {
        // Unpacked without `..`, so a new field that is never used fails to compile.
        let Self { mut swarm, addr, handler, connection_gate, static_peers, config } = self;
        let listener_id = swarm.listen_on(addr.clone()).inspect_err(|err| {
            error!(target: "gossip", "Fail to listen on {addr}: {err}");
        })?;
        let mut driver = GossipDriver {
            swarm,
            addr,
            config,
            handler,
            peerstore: HashMap::new(),
            peer_connection_start: HashMap::new(),
            connection_gate,
            static_peers,
            next_static_peer_check: None,
            ping: HashMap::new(),
            pending_connects: HashMap::new(),
            pending_disconnects: HashMap::new(),
            connections: HashMap::new(),
            next_duplicate_check: None,
            listener_id,
        };
        loop {
            match driver.swarm.select_next_some().await {
                SwarmEvent::NewListenAddr { address, listener_id }
                    if listener_id == driver.listener_id =>
                {
                    info!(target: "gossip", peer_id = %driver.local_peer_id(), "Swarm now listening on: {address}");

                    driver.addr = address;
                    driver.dial_static_peers();

                    return Ok(driver);
                }
                SwarmEvent::ListenerClosed { listener_id, reason, .. }
                    if listener_id == driver.listener_id =>
                {
                    error!(target: "gossip", ?reason, "Listener on {} closed before it reported an address", driver.addr);
                    return Err(StartError::ListenerClosed(reason.err()));
                }
                // Keep the driver's state in step with the swarm. A block arriving here would be
                // dropped, but blocks need a connection, and the driver dials its static peers only
                // once it listens.
                event => {
                    driver.handle_event(event);
                }
            }
        }
    }
}
