//! An implementation of the [`ConnectionGate`] trait.

use crate::{CanonicalIp, ConnectionGate, DialError, gate};
use libp2p::{Multiaddr, PeerId, multiaddr::Protocol};
use std::{
    collections::{HashMap, HashSet},
    io,
    net::{IpAddr, ToSocketAddrs},
    time::Duration,
};
use tokio::time::Instant;

/// Dial information tracking for peer connection management.
///
/// Tracks connection attempt statistics for rate limiting and connection gating.
/// Used to prevent excessive connection attempts to the same peer within a
/// configured time window.
#[derive(Debug, Clone)]
pub(crate) struct DialInfo {
    /// Number of times the peer has been dialed during the current dial period.
    /// This number is reset once the last time the peer was dialed is longer than the dial period.
    pub(crate) num_dials: u64,
    /// The last time the peer was dialed.
    pub(crate) last_dial: Instant,
}

impl Default for DialInfo {
    fn default() -> Self {
        Self { num_dials: 0, last_dial: Instant::now() }
    }
}

/// Configuration parameters for the connection gater.
///
/// Controls rate limiting, connection management, and peer protection policies
/// to maintain network health and prevent abuse.
#[derive(Debug, Clone)]
pub struct GaterConfig {
    /// Maximum number of connection attempts per dial period for a single peer.
    ///
    /// If set to `None`, unlimited redials are allowed. When set, prevents
    /// excessive connection attempts to unresponsive or problematic peers.
    pub peer_redialing: Option<u64>,

    /// Duration of the rate limiting window for peer connections.
    ///
    /// A peer cannot be dialed more than `peer_redialing` times during this
    /// period. The period resets after this duration has elapsed since the
    /// last dial attempt. Default is 1 hour.
    pub dial_period: Duration,
}

impl Default for GaterConfig {
    fn default() -> Self {
        Self { peer_redialing: None, dial_period: Duration::from_secs(60 * 60) }
    }
}

/// Connection Gater
///
/// A connection gate that regulates peer connections for the libp2p gossip swarm.
///
/// An implementation of the [`ConnectionGate`] trait.
#[derive(Default, Debug, Clone)]
pub struct ConnectionGater {
    /// The configuration for the connection gater.
    config: GaterConfig,
    /// A set of [`PeerId`]s that are currently being dialed.
    pub current_dials: HashSet<PeerId>,
    /// The dial info for each address dialed.
    pub(crate) dialed_addrs: HashMap<Multiaddr, DialInfo>,
    /// A set of protected peers that cannot be disconnected.
    ///
    /// Protecting a peer prevents the peer from any redial thresholds or peer scoring.
    pub protected_peers: HashSet<PeerId>,
    /// Whether addresses without an IP, such as `/memory` ones, may be dialed.
    ///
    /// The checks on the peer ID still apply to them, but IP and subnet blocks cannot, so they
    /// are only allowed for a [`crate::GossipTransport::Other`] transport.
    allow_addrs_without_ip: bool,
}

impl ConnectionGater {
    /// Creates a new instance of the `ConnectionGater`.
    pub fn new(config: GaterConfig) -> Self {
        Self {
            config,
            current_dials: HashSet::new(),
            dialed_addrs: HashMap::new(),
            protected_peers: HashSet::new(),
            allow_addrs_without_ip: false,
        }
    }

    /// Allows dialing addresses without an IP, such as `/memory` ones.
    pub(crate) const fn allowing_addrs_without_ip(mut self) -> Self {
        self.allow_addrs_without_ip = true;
        self
    }

    /// Returns if the given [`Multiaddr`] has been dialed the maximum number of times.
    pub fn dial_threshold_reached(&self, addr: &Multiaddr) -> bool {
        // If the peer has not been dialed yet, the threshold is not reached.
        let Some(dialed) = self.dialed_addrs.get(addr) else {
            return false;
        };
        // Without a threshold, a peer can be redialed any number of times.
        let Some(redialing) = self.config.peer_redialing else {
            return false;
        };
        // If the threshold is set to `0`, redial indefinitely.
        if redialing == 0 {
            return false;
        }
        if dialed.num_dials >= redialing {
            return true;
        }
        false
    }

    fn dial_period_expired(&self, addr: &Multiaddr) -> bool {
        let Some(dial_info) = self.dialed_addrs.get(addr) else {
            return false;
        };
        dial_info.last_dial.elapsed() > self.config.dial_period
    }

    /// Gets the [`PeerId`] from a given [`Multiaddr`].
    pub fn peer_id_from_addr(addr: &Multiaddr) -> Option<PeerId> {
        addr.iter().find_map(|component| match component {
            libp2p::multiaddr::Protocol::P2p(peer_id) => Some(peer_id),
            _ => None,
        })
    }

    /// Returns the IP address in the given [`Multiaddr`], in the form the gate matches.
    pub fn ip_from_addr(addr: &Multiaddr) -> Option<CanonicalIp> {
        addr.iter().find_map(|component| match component {
            libp2p::multiaddr::Protocol::Ip4(ip) => Some(IpAddr::V4(ip).into()),
            libp2p::multiaddr::Protocol::Ip6(ip) => Some(IpAddr::V6(ip).into()),
            _ => None,
        })
    }

    /// Looks up the IPs of a host name with the system resolver. This blocks until the lookup is
    /// done.
    pub fn system_resolve(host: &str) -> io::Result<Vec<IpAddr>> {
        Ok((host, 0).to_socket_addrs()?.map(|socket_addr| socket_addr.ip()).collect())
    }

    /// Resolves the host name in a `/dns`, `/dns4` or `/dns6` address with `resolve`, to the IPs of
    /// the family the protocol allows: `/dns4` only IPv4 ones, `/dns6` only IPv6 ones.
    ///
    /// Returns `None` for an address without such a name. A `/dnsaddr` name doesn't count: it
    /// stands for the TXT records under `_dnsaddr.<name>`, which hold whole multiaddrs.
    pub fn resolve_dns(
        addr: &Multiaddr,
        resolve: impl FnOnce(&str) -> io::Result<Vec<IpAddr>>,
    ) -> Option<io::Result<Vec<IpAddr>>> {
        let (host, family): (_, fn(&IpAddr) -> bool) =
            addr.iter().find_map(|component| match component {
                Protocol::Dns(host) => Some((host, (|_| true) as fn(&IpAddr) -> bool)),
                Protocol::Dns4(host) => Some((host, IpAddr::is_ipv4 as fn(&IpAddr) -> bool)),
                Protocol::Dns6(host) => Some((host, IpAddr::is_ipv6 as fn(&IpAddr) -> bool)),
                _ => None,
            })?;
        let ips = resolve(&host).map(|ips| {
            let mut seen = HashSet::new();
            ips.into_iter().filter(|ip| family(ip) && seen.insert(*ip)).collect::<Vec<_>>()
        });
        debug!(target: "gossip", %host, ?ips, "Resolved DNS name");
        Some(ips)
    }

    /// Replaces the host name in a `/dns`, `/dns4` or `/dns6` address with each IP that `resolve`
    /// finds for it, so that the gate checks the same addresses that libp2p then dials.
    ///
    /// Returns any other address as it is, and no address for a name that doesn't resolve.
    pub fn resolve_dns_addrs(
        addr: Multiaddr,
        resolve: impl FnOnce(&str) -> io::Result<Vec<IpAddr>>,
    ) -> Vec<Multiaddr> {
        let ips = match Self::resolve_dns(&addr, resolve) {
            None => return vec![addr],
            Some(Ok(ips)) => ips,
            Some(Err(error)) => {
                warn!(target: "gossip", peer=?addr, %error, "DNS resolution failed, not dialing the address");
                return Vec::new();
            }
        };
        if ips.is_empty() {
            warn!(target: "gossip", peer=?addr, "DNS name has no IP of the right family, not dialing the address");
        }
        ips.into_iter()
            .map(|ip| {
                addr.iter()
                    .map(|component| match component {
                        Protocol::Dns(_) | Protocol::Dns4(_) | Protocol::Dns6(_) => ip.into(),
                        component => component,
                    })
                    .collect()
            })
            .collect()
    }

    /// Returns why the gate refuses to dial `peer_id` at `addr`, see
    /// [`ConnectionGate::can_dial_addr`].
    fn check_addr(
        &self,
        peer_id: &PeerId,
        addr: &Multiaddr,
        gate: &gate::Behaviour,
    ) -> Result<(), DialError> {
        // Protected peers aren't held to the redial limit.
        if !self.protected_peers.contains(peer_id) &&
            self.dial_threshold_reached(addr) &&
            !self.dial_period_expired(addr)
        {
            return Err(DialError::ThresholdReached { addr: addr.clone() });
        }
        match Self::ip_from_addr(addr) {
            Some(ip) => gate.check_ip(ip),
            // IP and subnet blocks can't apply to it.
            None if self.allow_addrs_without_ip => Ok(()),
            None => {
                warn!(target: "p2p", peer=?addr, "Failed to extract IpAddr from Multiaddr");
                Err(DialError::InvalidIpAddress { addr: addr.clone() })
            }
        }
    }
}

impl ConnectionGate for ConnectionGater {
    fn can_dial_peer(&self, peer_id: &PeerId, gate: &gate::Behaviour) -> Result<(), DialError> {
        // A block first, as the more useful answer for a blocked peer still being dialed.
        let refusal = if let Err(refusal) = gate.check_peer(peer_id) {
            refusal
        } else if self.current_dials.contains(peer_id) {
            DialError::AlreadyDialing { peer_id: *peer_id }
        } else {
            return Ok(());
        };
        debug!(target: "gossip", %peer_id, %refusal, "Not dialing peer");
        metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => refusal.reason()).increment(1);
        Err(refusal)
    }

    fn can_dial_addr(
        &self,
        peer_id: &PeerId,
        addr: &Multiaddr,
        gate: &gate::Behaviour,
    ) -> Result<(), DialError> {
        self.check_addr(peer_id, addr, gate).inspect_err(|refusal| {
            debug!(target: "gossip", peer=?addr, %refusal, "Not dialing address");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => refusal.reason())
                .increment(1);
        })
    }

    fn list_protected_peers(&self) -> Vec<PeerId> {
        self.protected_peers.iter().copied().collect()
    }

    fn dialing(&mut self, peer_id: &PeerId) {
        self.current_dials.insert(*peer_id);
    }

    fn dialed(&mut self, addr: &Multiaddr) {
        let dial_info = self
            .dialed_addrs
            .entry(addr.clone())
            .or_insert_with(|| DialInfo { num_dials: 0, last_dial: Instant::now() });

        // If the last dial was longer than the dial period, reset the number of dials.
        if dial_info.last_dial.elapsed() > self.config.dial_period {
            dial_info.num_dials = 0;
        }

        dial_info.num_dials += 1;
        dial_info.last_dial = Instant::now();
        trace!(target: "gossip", peer=?addr, "Dialed peer, current count: {}", dial_info.num_dials);
    }

    fn remove_dial(&mut self, peer_id: &PeerId) {
        self.current_dials.remove(peer_id);
    }

    fn can_disconnect(&self, addr: &Multiaddr) -> bool {
        let Some(peer_id) = Self::peer_id_from_addr(addr) else {
            warn!(target: "p2p", peer=?addr, "Failed to extract PeerId from Multiaddr when checking disconnect");
            // If we cannot extract the PeerId, disconnection is allowed.
            return true;
        };
        // If the peer is protected, do not disconnect.
        if !self.protected_peers.contains(&peer_id) {
            return true;
        }
        // Peer is protected, cannot disconnect.
        false
    }

    fn protect_peer(&mut self, peer_id: PeerId) {
        self.protected_peers.insert(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Protected peer");
    }

    fn unprotect_peer(&mut self, peer_id: PeerId) {
        self.protected_peers.remove(&peer_id);
        debug!(target: "gossip", peer=?peer_id, "Unprotected peer");
    }
}

#[test]
fn test_dial_threshold() {
    use crate::{ConnectionGate, DialError};
    use std::str::FromStr;

    let addr = Multiaddr::from_str(
        "/ip4/127.0.0.1/tcp/8080/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();
    let peer_id = ConnectionGater::peer_id_from_addr(&addr).unwrap();
    let gate = gate::Behaviour::default();
    let dial = |gater: &mut ConnectionGater| {
        let result = gater
            .can_dial_peer(&peer_id, &gate)
            .and_then(|()| gater.can_dial_addr(&peer_id, &addr, &gate));
        if result.is_ok() {
            gater.dialing(&peer_id);
            gater.dialed(&addr);
            // As when the dial fails.
            gater.remove_dial(&peer_id);
        }
        result
    };

    // Without a threshold, a peer can be redialed any number of times.
    let mut gater = ConnectionGater::new(GaterConfig::default());
    for _ in 0..10 {
        assert!(dial(&mut gater).is_ok());
    }

    // With a threshold, dials are refused once it is reached.
    let mut gater =
        ConnectionGater::new(GaterConfig { peer_redialing: Some(2), ..GaterConfig::default() });
    assert!(dial(&mut gater).is_ok());
    assert!(dial(&mut gater).is_ok());
    assert!(matches!(dial(&mut gater), Err(DialError::ThresholdReached { .. })));
}

/// A peer being dialed isn't dialed again, and a block is the answer for one that is also blocked.
#[test]
fn test_can_dial_peer() {
    use crate::{ConnectionGate, DialError};

    let mut gater = ConnectionGater::new(GaterConfig::default());
    let mut gate = gate::Behaviour::default();
    let peer_id = PeerId::random();
    assert!(gater.can_dial_peer(&peer_id, &gate).is_ok());

    gater.dialing(&peer_id);
    let refusal = gater.can_dial_peer(&peer_id, &gate);
    assert!(matches!(refusal, Err(DialError::AlreadyDialing { .. })), "{refusal:?}");

    gate.block_peer(peer_id);
    let refusal = gater.can_dial_peer(&peer_id, &gate);
    assert!(matches!(refusal, Err(DialError::PeerBlocked { .. })), "{refusal:?}");
}

#[test]
fn test_memory_multiaddr_can_dial() {
    use crate::{ConnectionGate, DialError};
    use std::str::FromStr;

    let addr = Multiaddr::from_str(
        "/memory/1234/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();
    let peer_id = ConnectionGater::peer_id_from_addr(&addr).unwrap();

    // By default, an address without an IP is rejected.
    let gater = ConnectionGater::new(GaterConfig::default());
    let mut gate = gate::Behaviour::default();
    let refusal = gater.can_dial_addr(&peer_id, &addr, &gate);
    assert!(matches!(refusal, Err(DialError::InvalidIpAddress { .. })), "{refusal:?}");

    // For a custom transport, there is no IP to check, so even a block on every IPv4 address
    // does not apply.
    let gater = ConnectionGater::new(GaterConfig::default()).allowing_addrs_without_ip();
    gate.block_subnet("0.0.0.0/0".parse().unwrap());
    assert!(gater.can_dial_addr(&peer_id, &addr, &gate).is_ok());
}

/// A DNS name the driver hasn't resolved counts as no IP, so it isn't dialed unchecked.
#[test]
fn test_unresolved_dns_name_counts_as_no_ip() {
    use crate::{ConnectionGate, DialError};

    let gater = ConnectionGater::new(GaterConfig::default());
    let gate = gate::Behaviour::default();
    let peer_id = PeerId::random();
    let addr: Multiaddr = format!("/dns4/example.com/tcp/9003/p2p/{peer_id}").parse().unwrap();

    let refusal = gater.can_dial_addr(&peer_id, &addr, &gate);
    assert!(matches!(refusal, Err(DialError::InvalidIpAddress { .. })), "{refusal:?}");
}

#[test]
fn test_dns_multiaddr_detection() {
    let peer = "12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp";
    // Resolves nothing, so the test needs no network: only whether the address has a name counts.
    let no_ips = |_: &str| Ok(Vec::new());
    let has_dns_name =
        |addr: String| ConnectionGater::resolve_dns(&addr.parse().unwrap(), no_ips).is_some();

    assert!(has_dns_name(format!("/dns4/example.com/tcp/9003/p2p/{peer}")));
    assert!(has_dns_name(format!("/dns6/example.com/tcp/9003/p2p/{peer}")));
    assert!(has_dns_name(format!("/dns/example.com/tcp/9003/p2p/{peer}")));
    // A `/dnsaddr` name stands for TXT records, not for IPs.
    assert!(!has_dns_name(format!("/dnsaddr/example.com/p2p/{peer}")));
    assert!(!has_dns_name(format!("/ip4/127.0.0.1/tcp/9003/p2p/{peer}")));
    assert!(!has_dns_name(format!("/ip6/::1/tcp/9003/p2p/{peer}")));
}

/// A resolver that finds two IPv4 addresses, an IPv6 one, and the first IPv4 one again.
#[cfg(test)]
fn two_ipv4_and_one_ipv6(_: &str) -> std::io::Result<Vec<IpAddr>> {
    Ok(["10.0.0.1", "10.0.0.2", "::1", "10.0.0.1"].map(|ip| ip.parse().unwrap()).to_vec())
}

#[test]
fn test_resolve_dns_keeps_the_family_and_drops_repeats() {
    let ips = |addr: &str| {
        ConnectionGater::resolve_dns(&addr.parse().unwrap(), two_ipv4_and_one_ipv6)
            .unwrap()
            .unwrap()
            .iter()
            .map(ToString::to_string)
            .collect::<Vec<_>>()
    };
    assert_eq!(ips("/dns4/node.example.com/tcp/9222"), ["10.0.0.1", "10.0.0.2"]);
    assert_eq!(ips("/dns6/node.example.com/tcp/9222"), ["::1"]);
    assert_eq!(ips("/dns/node.example.com/tcp/9222"), ["10.0.0.1", "10.0.0.2", "::1"]);
}

#[test]
fn test_resolve_dns_addrs() {
    let peer = "12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp";
    let resolve = |addr: String, resolver: fn(&str) -> std::io::Result<Vec<IpAddr>>| {
        ConnectionGater::resolve_dns_addrs(addr.parse().unwrap(), resolver)
            .iter()
            .map(ToString::to_string)
            .collect::<Vec<_>>()
    };

    // One address per IP of the right family, the rest of the address kept.
    assert_eq!(
        resolve(format!("/dns4/node.example.com/tcp/9222/p2p/{peer}"), two_ipv4_and_one_ipv6),
        [
            format!("/ip4/10.0.0.1/tcp/9222/p2p/{peer}"),
            format!("/ip4/10.0.0.2/tcp/9222/p2p/{peer}")
        ]
    );
    // A name that doesn't resolve is dropped, rather than dialed unchecked.
    let fails = |_: &str| Err(std::io::Error::other("no such host"));
    assert!(resolve(format!("/dns4/node.example.com/tcp/9222/p2p/{peer}"), fails).is_empty());
    // Addresses without a DNS name stay as they are.
    for addr in
        [format!("/ip4/10.0.0.3/tcp/9222/p2p/{peer}"), format!("/dnsaddr/example.com/p2p/{peer}")]
    {
        assert_eq!(resolve(addr.clone(), two_ipv4_and_one_ipv6), [addr]);
    }
}

/// Of the addresses a name resolves to, the gate refuses only the blocked one.
#[test]
fn test_gate_checks_every_resolved_ip() {
    use crate::ConnectionGate;

    let gater = ConnectionGater::new(GaterConfig::default());
    let mut gate = gate::Behaviour::default();
    gate.block_addr("10.0.0.1".parse().unwrap());
    let addr =
        "/dns4/node.example.com/tcp/9222/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp";
    let peer_id = ConnectionGater::peer_id_from_addr(&addr.parse().unwrap()).unwrap();

    let allowed: Vec<_> =
        ConnectionGater::resolve_dns_addrs(addr.parse().unwrap(), two_ipv4_and_one_ipv6)
            .into_iter()
            .filter(|addr| gater.can_dial_addr(&peer_id, addr, &gate).is_ok())
            .map(|addr| addr.to_string())
            .collect();

    assert_eq!(
        allowed,
        ["/ip4/10.0.0.2/tcp/9222/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp"]
    );
}
