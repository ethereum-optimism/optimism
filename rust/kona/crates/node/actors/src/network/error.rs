//! Errors from starting the P2P stack.

use kona_disc::Discv5BuilderError;
use kona_gossip::GossipDriverBuilderError;
use libp2p::{Multiaddr, TransportError};

/// An error from [`NetworkConfig::start`](super::NetworkConfig::start).
#[derive(Debug, thiserror::Error)]
pub enum StartError {
    /// The gossip driver could not be built.
    #[error(transparent)]
    Gossip(#[from] GossipDriverBuilderError),
    /// The discovery driver could not be built.
    #[error(transparent)]
    Discovery(#[from] Discv5BuilderError),
    /// The gossip swarm could not listen on its address.
    #[error("Failed to listen for gossip: {0}")]
    Listen(#[from] TransportError<std::io::Error>),
    /// The gossip swarm listens on an address without an IP and TCP port to advertise.
    #[error("Gossip listen address has no IP and TCP port: {0}")]
    InvalidListenAddr(Multiaddr),
}
