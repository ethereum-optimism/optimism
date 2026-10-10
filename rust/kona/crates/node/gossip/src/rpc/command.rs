//! Commands that mutate the gossip network.

use crate::{ConnectionGate, GossipDriver};
use ipnet::IpNet;
use libp2p::{Multiaddr, PeerId};
use std::net::IpAddr;
use tokio::sync::{mpsc, oneshot};

/// Sends commands with notifications that their changes have been published.
pub type GossipCommandSender = mpsc::Sender<(GossipCommand, oneshot::Sender<()>)>;

/// Receives commands and their completion notifications.
pub type GossipCommandReceiver = mpsc::Receiver<(GossipCommand, oneshot::Sender<()>)>;

/// A command executed by the network actor on its gossip driver.
#[derive(Debug)]
pub enum GossipCommand {
    /// Block a peer.
    BlockPeer {
        /// The peer to block.
        id: PeerId,
    },
    /// Unblock a peer.
    UnblockPeer {
        /// The peer to unblock.
        id: PeerId,
    },
    /// Block an IP address.
    BlockAddr {
        /// The IP address to block.
        address: IpAddr,
    },
    /// Unblock an IP address.
    UnblockAddr {
        /// The IP address to unblock.
        address: IpAddr,
    },
    /// Block a subnet.
    BlockSubnet {
        /// The subnet to block.
        address: IpNet,
    },
    /// Unblock a subnet.
    UnblockSubnet {
        /// The subnet to unblock.
        address: IpNet,
    },
    /// Connect to a peer.
    ConnectPeer {
        /// The address to dial.
        address: Multiaddr,
    },
    /// Disconnect a peer.
    DisconnectPeer {
        /// The peer to disconnect.
        peer_id: PeerId,
    },
    /// Protect a peer from disconnection.
    ProtectPeer {
        /// The peer to protect.
        peer_id: PeerId,
    },
    /// Unprotect a peer.
    UnprotectPeer {
        /// The peer to unprotect.
        peer_id: PeerId,
    },
}

impl GossipCommand {
    /// Executes this command on the actor's gossip driver.
    pub fn handle<G: ConnectionGate>(self, gossip: &mut GossipDriver<G>) {
        match self {
            Self::BlockPeer { id } => {
                gossip.connection_gate.block_peer(&id);
                gossip.swarm.behaviour_mut().gossipsub.blacklist_peer(&id);
            }
            Self::UnblockPeer { id } => {
                gossip.connection_gate.unblock_peer(&id);
                gossip.swarm.behaviour_mut().gossipsub.remove_blacklisted_peer(&id);
            }
            Self::BlockAddr { address } => gossip.connection_gate.block_addr(address),
            Self::UnblockAddr { address } => gossip.connection_gate.unblock_addr(address),
            Self::BlockSubnet { address } => gossip.connection_gate.block_subnet(address),
            Self::UnblockSubnet { address } => gossip.connection_gate.unblock_subnet(address),
            Self::ConnectPeer { address } => gossip.dial_multiaddr(address),
            Self::ProtectPeer { peer_id } => gossip.connection_gate.protect_peer(peer_id),
            Self::UnprotectPeer { peer_id } => gossip.connection_gate.unprotect_peer(peer_id),
            Self::DisconnectPeer { peer_id } => {
                if let Err(e) = gossip.swarm.disconnect_peer_id(peer_id) {
                    warn!(target: "p2p::rpc", "Failed to disconnect peer {}: {:?}", peer_id, e);
                } else {
                    info!(target: "p2p::rpc", "Disconnected peer {}", peer_id);
                    if let Some(start_time) = gossip.peer_connection_start.remove(&peer_id) {
                        metrics::histogram!(
                            crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
                        )
                        .record(start_time.elapsed().as_secs_f64());
                    }
                }
            }
        }
    }
}
