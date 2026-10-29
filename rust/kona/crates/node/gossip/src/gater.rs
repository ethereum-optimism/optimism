//! An implementation of the [`ConnectionGate`] trait.

use crate::{Connectedness, ConnectionGate, DialError};
use ipnet::IpNet;
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
    /// Holds a map from peer id to connectedness for the given peer id.
    pub connectedness: HashMap<PeerId, Connectedness>,
    /// A set of protected peers that cannot be disconnected.
    ///
    /// Protecting a peer prevents the peer from any redial thresholds or peer scoring.
    pub protected_peers: HashSet<PeerId>,
    /// A set of blocked peer ids.
    pub blocked_peers: HashSet<PeerId>,
    /// A set of blocked ip addresses that cannot be dialed.
    pub blocked_addrs: HashSet<IpAddr>,
    /// A set of blocked subnets that cannot be connected to.
    pub blocked_subnets: HashSet<IpNet>,
    /// Whether addresses without an IP or DNS name may be dialed.
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
            connectedness: HashMap::new(),
            protected_peers: HashSet::new(),
            blocked_peers: HashSet::new(),
            blocked_addrs: HashSet::new(),
            blocked_subnets: HashSet::new(),
            allow_addrs_without_ip: false,
        }
    }

    /// Allows dialing addresses without an IP or DNS name, such as `/memory` ones.
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

    /// Constructs the [`IpAddr`] from the given [`Multiaddr`].
    pub fn ip_from_addr(addr: &Multiaddr) -> Option<IpAddr> {
        addr.iter().find_map(|component| match component {
            libp2p::multiaddr::Protocol::Ip4(ip) => Some(IpAddr::V4(ip)),
            libp2p::multiaddr::Protocol::Ip6(ip) => Some(IpAddr::V6(ip)),
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

    /// Checks if a given [`IpAddr`] is within any of the `blocked_subnets`.
    pub fn check_ip_in_blocked_subnets(&self, ip_addr: &IpAddr) -> bool {
        for subnet in &self.blocked_subnets {
            if subnet.contains(ip_addr) {
                return true;
            }
        }
        false
    }

    /// Returns the IPs to check for an address: the one it has, or every one its DNS name resolves
    /// to.
    ///
    /// Returns none when there is no IP to check: DNS resolution failed, or the address has no IP
    /// and a custom transport is allowed to dial it. [`GossipDriver::dial_peer`] resolves names
    /// itself before it asks the gate, and drops those that don't resolve.
    ///
    /// [`GossipDriver::dial_peer`]: crate::GossipDriver::dial_peer
    fn ips_to_check(&self, addr: &Multiaddr) -> Result<Vec<IpAddr>, DialError> {
        match Self::resolve_dns(addr, Self::system_resolve) {
            Some(Ok(ips)) => Ok(ips),
            Some(Err(error)) => {
                debug!(target: "gossip", peer=?addr, %error, "DNS resolution failed, allowing dial");
                Ok(Vec::new())
            }
            None => match Self::ip_from_addr(addr) {
                Some(ip) => Ok(vec![ip]),
                None if self.allow_addrs_without_ip => Ok(Vec::new()),
                None => {
                    warn!(target: "p2p", peer=?addr, "Failed to extract IpAddr from Multiaddr");
                    Err(DialError::InvalidIpAddress { addr: addr.clone() })
                }
            },
        }
    }

    /// Checks the IP address of `addr` against the blocked addresses and subnets.
    fn check_ip(&mut self, addr: &Multiaddr, peer_id: PeerId, ip: IpAddr) -> Result<(), DialError> {
        // If the address is blocked, do not dial.
        if self.is_blocked_addr(&ip) {
            debug!(target: "gossip", peer=?addr, "Address is blocked, not dialing");
            self.connectedness.insert(peer_id, Connectedness::CannotConnect);
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "blocked_address")
                .increment(1);
            return Err(DialError::AddressBlocked { ip });
        }

        // If address lies in any blocked subnets, do not dial.
        if self.check_ip_in_blocked_subnets(&ip) {
            debug!(target: "gossip", ip=?ip, "IP address is in a blocked subnet, not dialing");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "blocked_subnet")
                .increment(1);
            return Err(DialError::SubnetBlocked { ip });
        }

        Ok(())
    }
}

impl ConnectionGate for ConnectionGater {
    fn can_dial(&mut self, addr: &Multiaddr) -> Result<(), DialError> {
        // Get the peer id from the given multiaddr.
        let peer_id = Self::peer_id_from_addr(addr).ok_or_else(|| {
            warn!(target: "p2p", peer=?addr, "Failed to extract PeerId from Multiaddr");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_multiaddr")
                .increment(1);
            DialError::InvalidMultiaddr { addr: addr.clone() }
        })?;

        // Cannot dial a peer that is already being dialed.
        if self.current_dials.contains(&peer_id) {
            debug!(target: "gossip", peer=?addr, "Already dialing peer, not dialing");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "already_dialing")
                .increment(1);
            return Err(DialError::AlreadyDialing { peer_id });
        }

        // If the peer is protected, do not apply thresholds.
        let protected = self.protected_peers.contains(&peer_id);

        // If the peer is not protected, its dial threshold is reached and dial period is not
        // expired, do not dial.
        if !protected && self.dial_threshold_reached(addr) && !self.dial_period_expired(addr) {
            debug!(target: "gossip", peer=?addr, "Dial threshold reached, not dialing");
            self.connectedness.insert(peer_id, Connectedness::CannotConnect);
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "threshold_reached")
                .increment(1);
            return Err(DialError::ThresholdReached { addr: addr.clone() });
        }

        // If the peer is blocked, do not dial.
        if self.is_blocked_peer(&peer_id) {
            debug!(target: "gossip", peer=?addr, "Peer is blocked, not dialing");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "blocked_peer").increment(1);
            return Err(DialError::PeerBlocked { peer_id });
        }

        // Checks on the IP address only apply to addresses that have one.
        for ip in self.ips_to_check(addr)? {
            self.check_ip(addr, peer_id, ip)?;
        }

        Ok(())
    }

    fn connectedness(&self, peer_id: &PeerId) -> Connectedness {
        self.connectedness.get(peer_id).copied().unwrap_or(Connectedness::NotConnected)
    }

    fn list_protected_peers(&self) -> Vec<PeerId> {
        self.protected_peers.iter().copied().collect()
    }

    fn dialing(&mut self, peer_id: &PeerId) {
        self.current_dials.insert(*peer_id);
        self.connectedness.insert(*peer_id, Connectedness::Connected);
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

    fn block_peer(&mut self, peer_id: &PeerId) {
        self.blocked_peers.insert(*peer_id);
        debug!(target: "gossip", peer=?peer_id, "Blocked peer");
        self.connectedness.insert(*peer_id, Connectedness::CannotConnect);
    }

    fn unblock_peer(&mut self, peer_id: &PeerId) {
        self.blocked_peers.remove(peer_id);
        debug!(target: "gossip", peer=?peer_id, "Unblocked peer");
        self.connectedness.insert(*peer_id, Connectedness::NotConnected);
    }

    fn list_blocked_peers(&self) -> Vec<PeerId> {
        self.blocked_peers.iter().copied().collect()
    }

    fn is_blocked_peer(&self, peer_id: &PeerId) -> bool {
        self.blocked_peers.contains(peer_id)
    }

    fn block_addr(&mut self, ip: IpAddr) {
        self.blocked_addrs.insert(ip);
        debug!(target: "gossip", ?ip, "Blocked ip address");
    }

    fn unblock_addr(&mut self, ip: IpAddr) {
        self.blocked_addrs.remove(&ip);
        debug!(target: "gossip", ?ip, "Unblocked ip address");
    }

    fn list_blocked_addrs(&self) -> Vec<IpAddr> {
        self.blocked_addrs.iter().copied().collect()
    }

    fn is_blocked_addr(&self, ip: &IpAddr) -> bool {
        self.blocked_addrs.contains(ip)
    }

    fn block_subnet(&mut self, subnet: IpNet) {
        self.blocked_subnets.insert(subnet);
        debug!(target: "gossip", ?subnet, "Blocked subnet");
    }

    fn unblock_subnet(&mut self, subnet: IpNet) {
        self.blocked_subnets.remove(&subnet);
        debug!(target: "gossip", ?subnet, "Unblocked subnet");
    }

    fn list_blocked_subnets(&self) -> Vec<IpNet> {
        self.blocked_subnets.iter().copied().collect()
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
fn test_check_ip_in_blocked_subnets_ipv4() {
    use std::str::FromStr;

    let mut gater = ConnectionGater::new(GaterConfig {
        peer_redialing: None,
        dial_period: Duration::from_secs(60 * 60),
    });
    gater.blocked_subnets.insert("192.168.1.0/24".parse::<IpNet>().unwrap());
    gater.blocked_subnets.insert("10.0.0.0/8".parse::<IpNet>().unwrap());
    gater.blocked_subnets.insert("172.16.0.0/16".parse::<IpNet>().unwrap());

    // IP in blocked subnet
    assert!(gater.check_ip_in_blocked_subnets(&IpAddr::from_str("192.168.1.100").unwrap()));
    assert!(gater.check_ip_in_blocked_subnets(&IpAddr::from_str("10.0.0.5").unwrap()));
    assert!(gater.check_ip_in_blocked_subnets(&IpAddr::from_str("172.16.255.255").unwrap()));

    // IP not in any blocked subnet
    assert!(!gater.check_ip_in_blocked_subnets(&IpAddr::from_str("192.168.2.1").unwrap()));
    assert!(!gater.check_ip_in_blocked_subnets(&IpAddr::from_str("172.17.0.1").unwrap()));
    assert!(!gater.check_ip_in_blocked_subnets(&IpAddr::from_str("8.8.8.8").unwrap()));
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
    let dial = |gater: &mut ConnectionGater| {
        let result = gater.can_dial(&addr);
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

#[test]
fn test_dial_error_handling() {
    use crate::{ConnectionGate, DialError};
    use std::str::FromStr;

    let mut gater = ConnectionGater::new(GaterConfig::default());

    // Test invalid multiaddr (missing peer ID)
    let invalid_addr = Multiaddr::from_str("/ip4/127.0.0.1/tcp/8080").unwrap();
    let result = gater.can_dial(&invalid_addr);
    assert!(matches!(result, Err(DialError::InvalidMultiaddr { .. })));

    // Test with valid address
    let valid_addr = Multiaddr::from_str(
        "/ip4/127.0.0.1/tcp/8080/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();

    // First dial should succeed
    assert!(gater.can_dial(&valid_addr).is_ok());

    // Mark as dialing
    gater.dialing(&ConnectionGater::peer_id_from_addr(&valid_addr).unwrap());

    // Second dial should fail with AlreadyDialing
    let result = gater.can_dial(&valid_addr);
    assert!(matches!(result, Err(DialError::AlreadyDialing { .. })));
}

#[test]
fn test_memory_multiaddr_can_dial() {
    use crate::{ConnectionGate, DialError};
    use std::str::FromStr;

    let addr = Multiaddr::from_str(
        "/memory/1234/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();

    // By default, an address without an IP or DNS name is rejected.
    let mut gater = ConnectionGater::new(GaterConfig::default());
    assert!(matches!(gater.can_dial(&addr), Err(DialError::InvalidIpAddress { .. })));

    // For a custom transport, there is no IP to check, so even a block on every IPv4 address
    // does not apply.
    let mut gater = ConnectionGater::new(GaterConfig::default()).allowing_addrs_without_ip();
    gater.block_subnet("0.0.0.0/0".parse::<IpNet>().unwrap());
    assert!(gater.can_dial(&addr).is_ok());

    // The checks on the peer ID still apply.
    let peer_id = ConnectionGater::peer_id_from_addr(&addr).unwrap();
    gater.block_peer(&peer_id);
    assert!(matches!(gater.can_dial(&addr), Err(DialError::PeerBlocked { .. })));
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

    let mut gater = ConnectionGater::new(GaterConfig::default());
    gater.block_addr("10.0.0.1".parse().unwrap());
    let addr =
        "/dns4/node.example.com/tcp/9222/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp";

    let allowed: Vec<_> =
        ConnectionGater::resolve_dns_addrs(addr.parse().unwrap(), two_ipv4_and_one_ipv6)
            .into_iter()
            .filter(|addr| gater.can_dial(addr).is_ok())
            .map(|addr| addr.to_string())
            .collect();

    assert_eq!(
        allowed,
        ["/ip4/10.0.0.2/tcp/9222/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp"]
    );
}

#[test]
fn test_dns_multiaddr_can_dial() {
    use crate::ConnectionGate;
    use std::str::FromStr;

    let mut gater = ConnectionGater::new(GaterConfig::default());

    // DNS4 multiaddr should be allowed to dial (IP checks skipped)
    let dns4_addr = Multiaddr::from_str(
        "/dns4/example.com/tcp/9003/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();
    assert!(gater.can_dial(&dns4_addr).is_ok());

    // Real-world DNS multiaddr format (like the one from the issue)
    let real_world_dns = Multiaddr::from_str(
        "/dns4/alfonso-0-opn-reth-a-rpc-1-p2p.primary.infra.dev.oplabs.cloud/tcp/9003/p2p/16Uiu2HAmUSo81N6iNQNKZCiqDAg5Mcmh9gwvPgKmKj1HH6qCR4Kq",
    )
    .unwrap();
    assert!(gater.can_dial(&real_world_dns).is_ok());

    // DNS multiaddr with blocked peer should still be blocked
    let peer_id = ConnectionGater::peer_id_from_addr(&dns4_addr).unwrap();
    gater.block_peer(&peer_id);
    assert!(gater.can_dial(&dns4_addr).is_err());
}

#[test]
fn test_dns_multiaddr_blocked_by_resolved_ip() {
    use crate::{ConnectionGate, DialError};
    use std::{net::IpAddr, str::FromStr};

    let mut gater = ConnectionGater::new(GaterConfig::default());

    // localhost resolves to 127.0.0.1
    let dns_localhost = Multiaddr::from_str(
        "/dns4/localhost/tcp/9003/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();

    // Should succeed before blocking
    assert!(gater.can_dial(&dns_localhost).is_ok());

    // Block 127.0.0.1
    gater.block_addr(IpAddr::from_str("127.0.0.1").unwrap());

    // Should now fail because localhost resolves to blocked IP
    let result = gater.can_dial(&dns_localhost);
    assert!(matches!(result, Err(DialError::AddressBlocked { .. })));
}

#[test]
fn test_dns_multiaddr_blocked_by_subnet() {
    use crate::{ConnectionGate, DialError};
    use std::str::FromStr;

    let mut gater = ConnectionGater::new(GaterConfig::default());

    // localhost resolves to 127.0.0.1
    let dns_localhost = Multiaddr::from_str(
        "/dns4/localhost/tcp/9003/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
    )
    .unwrap();

    // Should succeed before blocking
    assert!(gater.can_dial(&dns_localhost).is_ok());

    // Block the 127.0.0.0/8 subnet
    gater.block_subnet("127.0.0.0/8".parse().unwrap());

    // Should now fail because localhost resolves to IP in blocked subnet
    let result = gater.can_dial(&dns_localhost);
    assert!(matches!(result, Err(DialError::SubnetBlocked { .. })));
}

#[test]
fn test_is_blocked() {
    let mut gater = ConnectionGater::new(GaterConfig::default());
    let peer_id = PeerId::random();
    let ip: IpAddr = "10.0.0.1".parse().unwrap();
    assert!(!gater.is_blocked_peer(&peer_id));
    assert!(!gater.is_blocked_addr(&ip));

    gater.block_peer(&peer_id);
    gater.block_addr(ip);
    assert!(gater.is_blocked_peer(&peer_id));
    assert!(gater.is_blocked_addr(&ip));

    gater.unblock_peer(&peer_id);
    gater.unblock_addr(ip);
    assert!(!gater.is_blocked_peer(&peer_id));
    assert!(!gater.is_blocked_addr(&ip));

    // Only an address blocked by itself counts, not one in a blocked subnet.
    gater.block_subnet("10.0.0.0/8".parse().unwrap());
    assert!(!gater.is_blocked_addr(&ip));
}
