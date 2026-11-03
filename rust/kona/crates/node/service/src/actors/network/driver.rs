use kona_disc::Discv5Driver;
use kona_gossip::{ConnectionGater, GossipDriverStarter, PEER_SCORE_INSPECT_FREQUENCY, StartError};
use libp2p::{Multiaddr, multiaddr::Protocol};

use crate::actors::network::handler::{NetworkHandler, RunningDiscovery};

/// A network driver. This is the driver that is used to start the network.
#[derive(Debug)]
pub struct NetworkDriver {
    /// The gossip driver, started with the network.
    pub gossip: GossipDriverStarter<ConnectionGater>,
    /// The discovery driver, or `None` if discovery is disabled.
    pub discovery: Option<Discv5Driver>,
    /// Whether to advertise the TCP port gossip bound in the ENR, once the libp2p Swarm is
    /// started. Only the port: the ENR's IP is the one it was built with, or none on all
    /// interfaces.
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
    /// The TCP port gossip bound could not be written into the local ENR.
    #[error("error updating the local ENR: {0}")]
    EnrUpdate(enr::Error),
}

impl NetworkDriver {
    /// Starts the network.
    pub async fn start(mut self) -> Result<NetworkHandler, NetworkDriverError> {
        // Start the libp2p Swarm.
        let gossip = self.gossip.start().await?;
        let gossip_listen_addr = gossip.peer_addr().transport_addr();

        if self.enr_update &&
            let Some(discovery) = &mut self.discovery
        {
            // Advertise the TCP port gossip bound, which may have been 0 in the config. Not its
            // IP: on all interfaces, libp2p reports one listen address per interface, and the
            // first is arbitrary. The ENR keeps the IP it was built with, if any.
            let key = gossip_listen_addr
                .iter()
                .find_map(|p| match p {
                    Protocol::Ip4(_) => Some("tcp"),
                    Protocol::Ip6(_) => Some("tcp6"),
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

            discovery.disc.enr_insert(key, &port).map_err(NetworkDriverError::EnrUpdate)?;
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

        Ok(NetworkHandler { gossip, discovery, peer_score_inspector })
    }
}
