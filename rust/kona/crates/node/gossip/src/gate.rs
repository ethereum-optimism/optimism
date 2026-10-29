//! Connection gating for the libp2p gossip swarm.
//!
//! Two parts decide which connections the swarm makes and keeps:
//!
//! - The gate [`Behaviour`] holds the peers, IP addresses and subnets the operator blocked. It is a
//!   [`NetworkBehaviour`], a field of the gossip [`crate::Behaviour`], and its docs say why.
//! - A [`ConnectionGate`] keeps the driver's dial bookkeeping: which peers are being dialed, how
//!   often each address was dialed, and which peers are protected. Before each dial, it checks the
//!   address against the gate [`Behaviour`].

use crate::DialError;
use ipnet::IpNet;
use libp2p::{
    Multiaddr, PeerId,
    core::{Endpoint, transport::PortUse},
    swarm::{
        ConnectionDenied, ConnectionId, FromSwarm, NetworkBehaviour, THandler, THandlerInEvent,
        THandlerOutEvent, ToSwarm, dummy,
    },
};
use std::{
    collections::HashSet,
    convert::Infallible,
    net::IpAddr,
    task::{Context, Poll},
};

/// Connection Gate
///
/// The connection gate is used to isolate and abstract the
/// logic for which peers are allowed to connect to the
/// gossip swarm.
pub trait ConnectionGate {
    /// Checks whether the gossip swarm may dial `addr`, which must end in the peer's
    /// `/p2p/<peer id>`, given the peers, addresses and subnets in `gate`. Returns the reason
    /// in a [`DialError`] if not.
    fn can_dial(&mut self, addr: &Multiaddr, gate: &Behaviour) -> Result<(), DialError>;

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

    /// Protects a peer from being disconnected.
    fn protect_peer(&mut self, peer_id: PeerId);

    /// Unprotects a peer, allowing it to be disconnected.
    fn unprotect_peer(&mut self, peer_id: PeerId);

    /// Lists all protected peers.
    fn list_protected_peers(&self) -> Vec<PeerId>;
}

/// The peers, IP addresses and subnets the operator blocked.
///
/// # Why it is a behaviour
///
/// The swarm asks its behaviours about a connection while it polls, before it reports one:
/// [`NetworkBehaviour::handle_pending_inbound_connection`] before the handshake, with only the
/// remote address, and [`NetworkBehaviour::handle_established_inbound_connection`] once the peer
/// ID is known. A behaviour that returns an error there refuses the connection. These are the
/// only points where that is possible, as op-node's `InterceptAccept` and `InterceptSecured` are
/// in go-libp2p.
///
/// The driver only sees a [`libp2p::swarm::SwarmEvent`] afterwards. By
/// [`libp2p::swarm::SwarmEvent::ConnectionEstablished`], gossipsub has queued our subscriptions
/// to the peer, identify has started, and the peer counts as connected, so the driver could only
/// close the connection again. The peer would finish the Noise handshake every time, and could
/// keep redialing.
///
/// So the gate is a field of the gossip [`crate::Behaviour`], owned by the swarm, and the driver
/// reaches it through the swarm.
#[derive(Debug, Default, Clone)]
pub struct Behaviour {
    /// The blocked peers.
    peers: HashSet<PeerId>,
    /// The blocked IP addresses.
    addrs: HashSet<IpAddr>,
    /// The blocked subnets.
    subnets: HashSet<IpNet>,
}

impl Behaviour {
    /// Blocks a peer.
    pub fn block_peer(&mut self, peer_id: PeerId) {
        self.peers.insert(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Blocked peer");
    }

    /// Unblocks a peer.
    pub fn unblock_peer(&mut self, peer_id: &PeerId) {
        self.peers.remove(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Unblocked peer");
    }

    /// Lists the blocked peers.
    pub fn list_blocked_peers(&self) -> Vec<PeerId> {
        self.peers.iter().copied().collect()
    }

    /// Returns whether a peer is blocked.
    pub fn is_blocked_peer(&self, peer_id: &PeerId) -> bool {
        self.peers.contains(peer_id)
    }

    /// Blocks an IP address.
    pub fn block_addr(&mut self, ip: IpAddr) {
        self.addrs.insert(ip);
        debug!(target: "gossip", ?ip, "Blocked ip address");
    }

    /// Unblocks an IP address.
    pub fn unblock_addr(&mut self, ip: &IpAddr) {
        self.addrs.remove(ip);
        debug!(target: "gossip", ?ip, "Unblocked ip address");
    }

    /// Lists the blocked IP addresses.
    pub fn list_blocked_addrs(&self) -> Vec<IpAddr> {
        self.addrs.iter().copied().collect()
    }

    /// Returns whether an IP address is blocked by itself. A blocked subnet that contains it
    /// doesn't count, see [`Self::is_in_blocked_subnet`].
    pub fn is_blocked_addr(&self, ip: &IpAddr) -> bool {
        self.addrs.contains(ip)
    }

    /// Blocks a subnet.
    pub fn block_subnet(&mut self, subnet: IpNet) {
        self.subnets.insert(subnet);
        debug!(target: "gossip", ?subnet, "Blocked subnet");
    }

    /// Unblocks a subnet.
    pub fn unblock_subnet(&mut self, subnet: &IpNet) {
        self.subnets.remove(subnet);
        debug!(target: "gossip", ?subnet, "Unblocked subnet");
    }

    /// Lists the blocked subnets.
    pub fn list_blocked_subnets(&self) -> Vec<IpNet> {
        self.subnets.iter().copied().collect()
    }

    /// Returns whether an IP address lies in a blocked subnet.
    pub fn is_in_blocked_subnet(&self, ip: &IpAddr) -> bool {
        self.subnets.iter().any(|subnet| subnet.contains(ip))
    }
}

/// Allows every connection, and has no protocol or events of its own.
impl NetworkBehaviour for Behaviour {
    type ConnectionHandler = dummy::ConnectionHandler;
    type ToSwarm = Infallible;

    fn handle_established_inbound_connection(
        &mut self,
        _connection_id: ConnectionId,
        _peer: PeerId,
        _local_addr: &Multiaddr,
        _remote_addr: &Multiaddr,
    ) -> Result<THandler<Self>, ConnectionDenied> {
        Ok(dummy::ConnectionHandler)
    }

    fn handle_established_outbound_connection(
        &mut self,
        _connection_id: ConnectionId,
        _peer: PeerId,
        _addr: &Multiaddr,
        _role_override: Endpoint,
        _port_use: PortUse,
    ) -> Result<THandler<Self>, ConnectionDenied> {
        Ok(dummy::ConnectionHandler)
    }

    fn on_swarm_event(&mut self, _event: FromSwarm<'_>) {}

    fn on_connection_handler_event(
        &mut self,
        _peer_id: PeerId,
        _connection_id: ConnectionId,
        event: THandlerOutEvent<Self>,
    ) {
        match event {}
    }

    fn poll(
        &mut self,
        _cx: &mut Context<'_>,
    ) -> Poll<ToSwarm<Self::ToSwarm, THandlerInEvent<Self>>> {
        Poll::Pending
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_in_blocked_subnet_ipv4() {
        let mut gate = Behaviour::default();
        gate.block_subnet("192.168.1.0/24".parse().unwrap());
        gate.block_subnet("10.0.0.0/8".parse().unwrap());
        gate.block_subnet("172.16.0.0/16".parse().unwrap());
        let in_blocked_subnet = |ip: &str| gate.is_in_blocked_subnet(&ip.parse().unwrap());

        // IP in blocked subnet
        assert!(in_blocked_subnet("192.168.1.100"));
        assert!(in_blocked_subnet("10.0.0.5"));
        assert!(in_blocked_subnet("172.16.255.255"));

        // IP not in any blocked subnet
        assert!(!in_blocked_subnet("192.168.2.1"));
        assert!(!in_blocked_subnet("172.17.0.1"));
        assert!(!in_blocked_subnet("8.8.8.8"));
    }

    #[test]
    fn test_is_blocked() {
        let mut gate = Behaviour::default();
        let peer_id = PeerId::random();
        let ip: IpAddr = "10.0.0.1".parse().unwrap();
        assert!(!gate.is_blocked_peer(&peer_id));
        assert!(!gate.is_blocked_addr(&ip));

        gate.block_peer(peer_id);
        gate.block_addr(ip);
        assert!(gate.is_blocked_peer(&peer_id));
        assert!(gate.is_blocked_addr(&ip));

        gate.unblock_peer(&peer_id);
        gate.unblock_addr(&ip);
        assert!(!gate.is_blocked_peer(&peer_id));
        assert!(!gate.is_blocked_addr(&ip));

        // Only an address blocked by itself counts, not one in a blocked subnet.
        gate.block_subnet("10.0.0.0/8".parse().unwrap());
        assert!(!gate.is_blocked_addr(&ip));
    }
}
