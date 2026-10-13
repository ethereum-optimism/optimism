//! The error [`RollupNode::start`](crate::RollupNode::start) returns.

use alloy_transport::TransportError;
use std::{fmt, net::SocketAddr};

/// The transport protocol of a P2P listener.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ListenProtocol {
    /// TCP, which the gossip listener uses.
    Tcp,
    /// UDP, which the discovery listener uses.
    Udp,
}

impl fmt::Display for ListenProtocol {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::Tcp => "TCP",
            Self::Udp => "UDP",
        })
    }
}

/// An error starting or running a [`RollupNode`](crate::RollupNode).
///
/// Each variant's message describes the failure itself; the error that caused it, if any, is
/// its [`source`](std::error::Error::source).
#[derive(Debug, thiserror::Error)]
pub enum RollupNodeError {
    /// A P2P listen address could not be bound: the port is in use, the IP is not the host's, or
    /// the node lacks permission.
    #[error("failed to bind the P2P {protocol} listen address {addr}")]
    ListenAddressUnavailable {
        /// The listener's protocol.
        protocol: ListenProtocol,
        /// The address that could not be bound.
        addr: SocketAddr,
        /// The error binding the address.
        #[source]
        source: std::io::Error,
    },
    /// The engine rejected the node's JWT: the secret differs from the engine's, or the node's
    /// and the engine's clocks differ by more than the Engine API allows.
    #[error("the engine rejected the node's JWT")]
    JwtRejected,
    /// The capability exchange that checks the JWT secret failed for another reason after the
    /// retries, such as the engine being unreachable or answering with an error.
    #[error("the engine capability exchange that checks the JWT secret failed")]
    EngineCapabilityExchange(#[source] TransportError),
    /// The unsafe block signer could not be read from L1.
    #[error("failed to read the unsafe block signer from L1")]
    UnsafeBlockSigner(#[source] TransportError),
    /// A component of the node could not be started.
    #[error("{0}")]
    Startup(String),
    /// A component of the node stopped with an error.
    #[error("{0}")]
    Stopped(String),
}
