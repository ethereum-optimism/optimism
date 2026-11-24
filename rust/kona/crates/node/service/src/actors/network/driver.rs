use kona_disc::{AdvertiseTcpPortError, Discv5Driver};
use kona_gossip::{ConnectionGater, GossipDriverStarter, PEER_SCORE_INSPECT_FREQUENCY, StartError};

use crate::actors::network::handler::{NetworkHandler, RunningDiscovery};

/// A network driver. This is the driver that is used to start the network.
#[derive(Debug)]
pub struct NetworkDriver {
    /// The gossip driver, started with the network.
    pub gossip: GossipDriverStarter<ConnectionGater>,
    /// The discovery driver, or `None` if discovery is disabled.
    pub discovery: Option<Discv5Driver>,
}

/// An error from the [`NetworkDriver`].
#[derive(Debug, thiserror::Error)]
pub enum NetworkDriverError {
    /// An error occurred starting the libp2p Swarm.
    #[error("error starting libp2p Swarm")]
    GossipStartError(#[from] StartError),
    /// The TCP port gossip bound could not be advertised in the ENR.
    #[error("error advertising the gossip TCP port")]
    AdvertiseTcpPort(#[from] AdvertiseTcpPortError),
}

impl NetworkDriver {
    /// Starts the network.
    pub async fn start(self) -> Result<NetworkHandler, NetworkDriverError> {
        // Start the libp2p Swarm.
        let gossip = self.gossip.start().await?;
        let gossip_listen_addr = gossip.peer_addr().transport_addr();

        // Start the discovery service, advertising the TCP port gossip bound.
        let discovery = match self.discovery {
            Some(discovery) => {
                discovery.advertise_gossip_tcp_port(&gossip_listen_addr)?;
                let (handler, enr_receiver) = discovery.start();
                Some(RunningDiscovery { handler, enr_receiver })
            }
            None => {
                info!(target: "node::p2p", "Discovery is disabled");
                None
            }
        };

        // We are checking the peer scores every [`PEER_SCORE_INSPECT_FREQUENCY`] seconds.
        let peer_score_inspector = tokio::time::interval(*PEER_SCORE_INSPECT_FREQUENCY);

        Ok(NetworkHandler { gossip, discovery, peer_score_inspector })
    }
}
