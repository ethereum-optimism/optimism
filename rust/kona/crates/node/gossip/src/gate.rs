//! Connection Gate for the libp2p Gossip Swarm.

use crate::{Connectedness, DialError};
use ipnet::IpNet;
use libp2p::{Multiaddr, PeerId};
use std::net::IpAddr;

/// Connection Gate
///
/// The connection gate is used to isolate and abstract the
/// logic for which peers are allowed to connect to the
/// gossip swarm.
pub trait ConnectionGate {
    /// Checks whether the gossip swarm may dial `addr`, which must end in the peer's
    /// `/p2p/<peer id>`. Returns the reason in a [`DialError`] if not.
    fn can_dial(&mut self, addr: &Multiaddr) -> Result<(), DialError>;

    /// Returns the [`Connectedness`] for a given peer id.
    fn connectedness(&self, peer_id: &PeerId) -> Connectedness;

    /// Marks a peer as being dialed, at one or more addresses, until [`Self::remove_dial`].
    fn dialing(&mut self, peer_id: &PeerId);

    /// Marks an address as dialed.
    fn dialed(&mut self, addr: &Multiaddr);

    /// Removes a peer id from the current dials set.
    fn remove_dial(&mut self, peer: &PeerId);

    /// Checks if a peer can be removed from the gossip swarm.
    ///
    /// Since peers can be protected from disconnection, this method
    /// checks if the peer is protected or not.
    fn can_disconnect(&self, addr: &Multiaddr) -> bool;

    /// Blocks a given peer id.
    fn block_peer(&mut self, peer_id: &PeerId);

    /// Unblocks a given peer id.
    fn unblock_peer(&mut self, peer_id: &PeerId);

    /// Lists the blocked peers.
    fn list_blocked_peers(&self) -> Vec<PeerId>;

    /// Returns whether a peer is blocked.
    fn is_blocked_peer(&self, peer_id: &PeerId) -> bool;

    /// Blocks a given IP address from connecting to the gossip swarm.
    fn block_addr(&mut self, ip: IpAddr);

    /// Unblocks a given IP address, allowing it to connect to the gossip swarm.
    fn unblock_addr(&mut self, ip: IpAddr);

    /// Lists all blocked IP addresses.
    fn list_blocked_addrs(&self) -> Vec<IpAddr>;

    /// Returns whether an IP address is blocked by itself. A blocked subnet that contains it
    /// doesn't count.
    fn is_blocked_addr(&self, ip: &IpAddr) -> bool;

    /// Blocks a subnet from connecting to the gossip swarm.
    fn block_subnet(&mut self, subnet: IpNet);

    /// Unblocks a subnet, allowing it to connect to the gossip swarm.
    fn unblock_subnet(&mut self, subnet: IpNet);

    /// Lists all blocked subnets.
    fn list_blocked_subnets(&self) -> Vec<IpNet>;

    /// Protects a peer from being disconnected.
    fn protect_peer(&mut self, peer_id: PeerId);

    /// Unprotects a peer, allowing it to be disconnected.
    fn unprotect_peer(&mut self, peer_id: PeerId);

    /// Lists all protected peers.
    fn list_protected_peers(&self) -> Vec<PeerId>;
}
