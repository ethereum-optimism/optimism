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
    /// Unprotect a peer. Static peers stay protected.
    UnprotectPeer {
        /// The peer to unprotect.
        peer_id: PeerId,
    },
}

impl GossipCommand {
    /// Executes this command on the actor's gossip driver.
    pub fn handle<G: ConnectionGate>(self, gossip: &mut GossipDriver<G>) {
        match self {
            Self::BlockPeer { id } => gossip.block_peer(id),
            Self::UnblockPeer { id } => gossip.unblock_peer(id),
            Self::BlockAddr { address } => gossip.block_addr(address),
            Self::UnblockAddr { address } => gossip.unblock_addr(address),
            Self::BlockSubnet { address } => gossip.block_subnet(address),
            Self::UnblockSubnet { address } => gossip.unblock_subnet(address),
            Self::ConnectPeer { address } => gossip.dial_multiaddr(address),
            Self::ProtectPeer { peer_id } => gossip.connection_gate.protect_peer(peer_id),
            Self::UnprotectPeer { peer_id } => gossip.unprotect_peer(peer_id),
            Self::DisconnectPeer { peer_id } => gossip.disconnect_peer(peer_id),
        }
    }
}
