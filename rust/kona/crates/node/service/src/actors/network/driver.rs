use std::net::{IpAddr, SocketAddr};

use alloy_primitives::Address;
use discv5::multiaddr::Protocol;
use futures::future::OptionFuture;
use kona_disc::Discv5Driver;
use kona_gossip::{ConnectionGater, GossipDriver, PEER_SCORE_INSPECT_FREQUENCY};
use kona_sources::{BlockSigner, BlockSignerStartError};
use libp2p::{Multiaddr, TransportError};
use tokio::sync::watch;

use crate::actors::network::handler::NetworkHandler;

/// A network driver. This is the driver that is used to start the network.
#[derive(Debug)]
pub struct NetworkDriver {
    /// The gossip driver.
    pub gossip: GossipDriver<ConnectionGater>,
    /// The discovery driver.
    pub discovery: Discv5Driver,
    /// Whether to update the ENR socket after the libp2p Swarm is started.
    /// This is set to true by default.
    /// This may be set to false if the node is configured to use a static advertised address (when
    /// used with a nat for example).
    pub enr_update: bool,
    /// The unsafe block signer sender.
    pub unsafe_block_signer_sender: watch::Sender<Address>,
    /// A block signer. This is optional and should be set if the node is configured to sign blocks
    pub signer: Option<BlockSigner>,
}

/// An error from the [`NetworkDriver`].
#[derive(Debug, thiserror::Error)]
pub enum NetworkDriverError {
    /// An error occurred starting the libp2p Swarm.
    #[error("error starting libp2p Swarm")]
    GossipStartError(#[from] TransportError<std::io::Error>),
    /// An error occurred starting the block signer client.
    #[error("error starting block signer client: {0}")]
    BlockSignerStartError(#[from] BlockSignerStartError),
    /// An error occurred parsing the gossip listen address.
    #[error("error parsing gossip listen address: {0}")]
    InvalidGossipListenAddr(Multiaddr),
}

impl NetworkDriver {
    /// Starts the network.
    pub async fn start(mut self) -> Result<NetworkHandler, NetworkDriverError> {
        // Start the libp2p Swarm
        let gossip_listen_addr = self.gossip.start().await?;

        if self.enr_update {
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

            self.discovery.disc.update_local_enr_socket(SocketAddr::new(ip_address, port), true);
        }

        // Start the discovery service.
        let (handler, enr_receiver) = self.discovery.start();

        // We are checking the peer scores every [`PEER_SCORE_INSPECT_FREQUENCY`] seconds.
        let peer_score_inspector = tokio::time::interval(*PEER_SCORE_INSPECT_FREQUENCY);

        // Start the block signer if it is configured.
        let signer =
            OptionFuture::from(self.signer.map(async |s| s.start().await)).await.transpose()?;

        Ok(NetworkHandler {
            gossip: self.gossip,
            discovery: handler,
            enr_receiver,
            unsafe_block_signer_sender: self.unsafe_block_signer_sender,
            peer_score_inspector,
            signer,
        })
    }
}
