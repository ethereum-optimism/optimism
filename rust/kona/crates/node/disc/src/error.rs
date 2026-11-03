//! Error types of the discovery service.

use libp2p::Multiaddr;
use thiserror::Error;

/// An error that can occur when building the discovery service.
#[derive(Debug, Error)]
pub enum Discv5BuilderError {
    /// Could not create the discovery service.
    #[error("could not create discovery service: {0}")]
    Discv5CreationFailed(String),
    /// Failed to build the ENR.
    #[error("failed to build ENR")]
    EnrBuildFailed(#[source] enr::Error),
    /// An advertised UDP port was set, but discovery listens on port 0, so it binds a port that
    /// cannot be known in advance.
    #[error("an advertised discovery UDP port needs a fixed listen port, not 0")]
    AdvertisedUdpPortWithoutFixedListenPort,
    /// Failed to bind a free UDP port for discovery to listen on.
    #[error("failed to bind the discovery UDP socket")]
    BindFailed(#[source] std::io::Error),
    /// There is no UDP port to advertise, because a dual-stack listen config has a port of 0,
    /// which is not bound in advance.
    #[error("no discovery UDP port to advertise")]
    UdpPortZero,
}

/// An error that can occur when advertising the gossip TCP port in the ENR.
#[derive(Debug, Error)]
pub enum AdvertiseTcpPortError {
    /// The ENR leaves its TCP port to gossip, but the gossip address has no TCP port.
    #[error("no TCP port to advertise in the gossip address {0}")]
    NoGossipTcpPort(Multiaddr),
    /// Failed to write the gossip TCP port into the ENR.
    #[error("failed to update the ENR")]
    EnrUpdateFailed(#[source] enr::Error),
}

/// Why an address another node relayed is not worth dialing. See
/// [`AddrScope::check_relayed_addr`].
///
/// [`AddrScope::check_relayed_addr`]: crate::AddrScope::check_relayed_addr
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
pub enum RelayAddrError {
    /// The address is `0.0.0.0` or `::`.
    #[error("unspecified address")]
    Unspecified,
    /// The address is a multicast, broadcast, documentation or other special-purpose address.
    #[error("special-purpose address")]
    Special,
    /// The address is a loopback address, from a sender that is not on loopback.
    #[error("loopback address from a non-loopback sender")]
    Loopback,
    /// The address is a LAN address, from a sender that is on neither loopback nor a LAN.
    #[error("LAN address from a public sender")]
    Lan,
}
