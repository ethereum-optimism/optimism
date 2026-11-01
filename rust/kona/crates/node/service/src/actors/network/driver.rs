use std::net::{IpAddr, SocketAddr};

use kona_disc::Discv5Driver;
use kona_gossip::{ConnectionGater, GossipDriver, PEER_SCORE_INSPECT_FREQUENCY, StartError};
use libp2p::{Multiaddr, multiaddr::Protocol};

use crate::actors::network::handler::{NetworkHandler, RunningDiscovery};

/// A network driver. This is the driver that is used to start the network.
#[derive(Debug)]
pub struct NetworkDriver {
    /// The gossip driver.
    pub gossip: GossipDriver<ConnectionGater>,
    /// The discovery driver, or `None` if discovery is disabled.
    pub discovery: Option<Discv5Driver>,
    /// Whether to update the ENR socket after the libp2p Swarm is started.
    /// This is set to true by default.
    /// This may be set to false if the node is configured to use a static advertised address (when
    /// used with a nat for example).
    pub enr_update: bool,
}

/// An error from the [`NetworkDriver`].
#[derive(Debug, thiserror::Error)]
pub enum NetworkDriverError {
    /// An error occurred starting the libp2p Swarm.
    #[error("error starting libp2p Swarm")]
    GossipStartError(#[from] StartError),
    /// An error occurred parsing the gossip listen address.
    #[error("error parsing gossip listen address: {0}")]
    InvalidGossipListenAddr(Multiaddr),
}

impl NetworkDriver {
    /// Starts the network.
    pub async fn start(mut self) -> Result<NetworkHandler, NetworkDriverError> {
        // Start the libp2p Swarm.
        let gossip_listen_addr = self.gossip.start().await?.transport_addr();

        if self.enr_update &&
            let Some(discovery) = &mut self.discovery
        {
            // Update the local ENR socket to the gossip listen address.
            // Parse the multiaddr to a socket address.
            let ip_address = gossip_listen_addr
                .iter()
                .find_map(|p| match p {
                    Protocol::Ip4(ip) => Some(IpAddr::V4(ip)),
                    Protocol::Ip6(ip) => Some(IpAddr::V6(ip)),
                    _ => None,
                })
                .ok_or_else(|| {
                    NetworkDriverError::InvalidGossipListenAddr(gossip_listen_addr.clone())
                })?;
            let port = gossip_listen_addr
                .iter()
                .find_map(|p| match p {
                    Protocol::Tcp(port) => Some(port),
                    _ => None,
                })
                .ok_or_else(|| {
                    NetworkDriverError::InvalidGossipListenAddr(gossip_listen_addr.clone())
                })?;

            discovery.disc.update_local_enr_socket(SocketAddr::new(ip_address, port), true);
        }

        // Start the discovery service.
        let discovery = self.discovery.map(|discovery| {
            let (handler, enr_receiver) = discovery.start();
            RunningDiscovery { handler, enr_receiver }
        });
        if discovery.is_none() {
            info!(target: "node::p2p", "Discovery is disabled");
        }

        // We are checking the peer scores every [`PEER_SCORE_INSPECT_FREQUENCY`] seconds.
        let peer_score_inspector = tokio::time::interval(*PEER_SCORE_INSPECT_FREQUENCY);

        Ok(NetworkHandler { gossip: self.gossip, discovery, peer_score_inspector })
    }
}
