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
    collections::{HashMap, HashSet},
    convert::Infallible,
    net::IpAddr,
    task::{Context, Poll},
};
use tokio::time::Instant;

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

/// The peers, IP addresses and subnets the operator blocked, and the peers banned for their
/// score.
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
    /// The peers banned for their score, and when each ban expires. An expired ban no longer
    /// counts, and is removed when the next ban is added.
    bans: HashMap<PeerId, Instant>,
}

impl Behaviour {
    /// Blocks a peer.
    pub fn block_peer(&mut self, peer_id: PeerId) {
        self.peers.insert(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Blocked peer");
    }

    /// Unblocks a peer, and lifts its ban if it has one, as op-node does.
    pub fn unblock_peer(&mut self, peer_id: &PeerId) {
        self.peers.remove(peer_id);
        self.bans.remove(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Unblocked peer");
    }

    /// Bans a peer for its score until `until`.
    pub fn ban_peer(&mut self, peer_id: PeerId, until: Instant) {
        let now = Instant::now();
        self.bans.retain(|_, expiry| *expiry > now);
        self.bans.insert(peer_id, until);
        debug!(target: "gossip", peer=?peer_id, ?until, "Banned peer");
    }

    /// Returns whether a peer is banned for its score, and its ban hasn't expired yet.
    pub fn is_banned_peer(&self, peer_id: &PeerId) -> bool {
        self.bans.get(peer_id).is_some_and(|expiry| *expiry > Instant::now())
    }

    /// Lists the blocked peers. Banned peers aren't listed, as in op-node.
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

    /// Returns why the gate refuses an IP address: it is blocked by itself, or lies in a blocked
    /// subnet.
    pub fn check_ip(&self, ip: IpAddr) -> Result<(), DialError> {
        if self.is_blocked_addr(&ip) {
            return Err(DialError::AddressBlocked { ip });
        }
        if self.is_in_blocked_subnet(&ip) {
            return Err(DialError::SubnetBlocked { ip });
        }
        Ok(())
    }

    /// Returns whether an IP address lies in a blocked subnet.
    pub fn is_in_blocked_subnet(&self, ip: &IpAddr) -> bool {
        self.subnets.iter().any(|subnet| subnet.contains(ip))
    }
}

/// Counts an inbound connection the gate refuses, by its reason, and wraps the refusal for the
/// swarm.
fn refuse_inbound(refusal: DialError) -> ConnectionDenied {
    metrics::counter!(crate::Metrics::INBOUND_CONNECTIONS_REFUSED, "reason" => refusal.reason())
        .increment(1);
    ConnectionDenied::new(refusal)
}

/// Refuses inbound connections from blocked or banned peers and from blocked IP addresses and
/// subnets, and our own connections to addresses blocked while we dialed them. Has no protocol or
/// events of its own.
///
/// A refusal is a [`DialError`], as a dial the gate refuses gets, in the
/// [`ListenError::Denied`](libp2p::swarm::ListenError::Denied) the swarm reports.
impl NetworkBehaviour for Behaviour {
    type ConnectionHandler = dummy::ConnectionHandler;
    type ToSwarm = Infallible;

    /// Refuses a connection from a blocked IP address or subnet before the handshake, from the
    /// remote address alone, as op-node's `InterceptAccept` does. An address without an IP, such as
    /// a `/memory` one, is allowed, as in go-libp2p's basic gater.
    fn handle_pending_inbound_connection(
        &mut self,
        _connection_id: ConnectionId,
        _local_addr: &Multiaddr,
        remote_addr: &Multiaddr,
    ) -> Result<(), ConnectionDenied> {
        let Some(ip) = crate::ConnectionGater::ip_from_addr(remote_addr) else {
            return Ok(());
        };
        self.check_ip(ip).map_err(|refusal| {
            debug!(target: "gossip", %remote_addr, %refusal, "Refusing a connection from a blocked address");
            refuse_inbound(refusal)
        })
    }

    /// Refuses a connection from a blocked peer, the first point where its peer ID is known: once
    /// the handshake is done, and before any other behaviour sees the connection.
    fn handle_established_inbound_connection(
        &mut self,
        _connection_id: ConnectionId,
        peer: PeerId,
        _local_addr: &Multiaddr,
        remote_addr: &Multiaddr,
    ) -> Result<THandler<Self>, ConnectionDenied> {
        if self.is_blocked_peer(&peer) {
            debug!(target: "gossip", %peer, %remote_addr, "Refusing a connection from a blocked peer");
            return Err(refuse_inbound(DialError::PeerBlocked { peer_id: peer }));
        }
        if self.is_banned_peer(&peer) {
            debug!(target: "gossip", %peer, %remote_addr, "Refusing a connection from a banned peer");
            return Err(refuse_inbound(DialError::PeerBanned { peer_id: peer }));
        }
        Ok(dummy::ConnectionHandler)
    }

    /// Refuses a connection we dialed to an address blocked while the dial was in flight. The
    /// gate refuses to dial a blocked address, and blocking a peer aborts our dials to it, but no
    /// call on the swarm aborts the dials to an address.
    fn handle_established_outbound_connection(
        &mut self,
        _connection_id: ConnectionId,
        _peer: PeerId,
        addr: &Multiaddr,
        _role_override: Endpoint,
        _port_use: PortUse,
    ) -> Result<THandler<Self>, ConnectionDenied> {
        if let Some(ip) = crate::ConnectionGater::ip_from_addr(addr) &&
            let Err(refusal) = self.check_ip(ip)
        {
            debug!(target: "gossip", %addr, %refusal, "Refusing our connection to a blocked address");
            return Err(ConnectionDenied::new(refusal));
        }
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
    use metrics_util::debugging::{DebugValue, DebuggingRecorder};
    use rstest::rstest;
    use std::time::Duration;

    /// Each inbound connection the gate refuses is counted under its reason.
    #[rstest]
    #[case::blocked_peer("blocked_peer", |gate: &mut Behaviour, peer: PeerId| gate.block_peer(peer))]
    #[case::banned_peer("banned_peer", |gate: &mut Behaviour, peer: PeerId| {
        gate.ban_peer(peer, Instant::now() + Duration::from_secs(60))
    })]
    #[case::blocked_address("blocked_address", |gate: &mut Behaviour, _: PeerId| gate.block_addr("10.0.0.1".parse().unwrap()))]
    #[case::blocked_subnet("blocked_subnet", |gate: &mut Behaviour, _: PeerId| gate.block_subnet("10.0.0.0/8".parse().unwrap()))]
    fn test_refused_inbound_connections_are_counted_by_reason(
        #[case] reason: &str,
        #[case] block: fn(&mut Behaviour, PeerId),
    ) {
        let peer = PeerId::random();
        let local: Multiaddr = "/ip4/127.0.0.1/tcp/9222".parse().unwrap();
        let remote: Multiaddr = "/ip4/10.0.0.1/tcp/40000".parse().unwrap();
        let mut gate = Behaviour::default();
        block(&mut gate, peer);

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            // The swarm asks before the handshake, then once the peer ID is known.
            let id = ConnectionId::new_unchecked(0);
            let refused = gate.handle_pending_inbound_connection(id, &local, &remote).is_err() ||
                gate.handle_established_inbound_connection(id, peer, &local, &remote).is_err();
            assert!(refused, "the gate let the connection in");
        });

        let counted: Vec<_> = snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(key, ..)| key.key().name() == crate::Metrics::INBOUND_CONNECTIONS_REFUSED)
            .map(|(key, _, _, value)| {
                let label =
                    key.key().labels().find(|l| l.key() == "reason").map(|l| l.value().to_owned());
                (label, value)
            })
            .collect();
        assert_eq!(counted, vec![(Some(reason.to_owned()), DebugValue::Counter(1))]);
    }

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
    fn test_is_in_blocked_subnet_ipv6() {
        let mut gate = Behaviour::default();
        // The first 32 bits are the network: the documentation prefix, 2001:db8::/32.
        gate.block_subnet("2001:db8::/32".parse().unwrap());
        let in_blocked_subnet = |ip: &str| gate.is_in_blocked_subnet(&ip.parse().unwrap());

        assert!(in_blocked_subnet("2001:db8::1"));
        assert!(in_blocked_subnet("2001:db8:ffff:ffff::1"));
        assert!(!in_blocked_subnet("2001:db9::1"));
        assert!(!in_blocked_subnet("::1"));
    }

    /// A subnet only contains addresses of its own family.
    #[test]
    fn test_blocked_subnet_does_not_cross_ip_families() {
        let mut gate = Behaviour::default();
        gate.block_subnet("10.0.0.0/8".parse().unwrap());
        gate.block_subnet("2001:db8::/32".parse().unwrap());
        let in_blocked_subnet = |ip: &str| gate.is_in_blocked_subnet(&ip.parse().unwrap());

        // Not even the IPv4-mapped IPv6 form of an address in 10.0.0.0/8. Peers don't arrive in
        // that form, though: libp2p-tcp makes its IPv6 listeners IPv6-only, so an IPv4 peer
        // connects to an IPv4 listener, as `/ip4/...`.
        assert!(!in_blocked_subnet("::ffff:10.0.0.1"));
        // The IPv4 address 32.1.13.184 has the same first 32 bits as 2001:db8::.
        assert!(!in_blocked_subnet("32.1.13.184"));
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
