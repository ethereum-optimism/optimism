//! Contains the RPC Configuration.

use std::net::SocketAddr;

/// The RPC configuration.
///
/// The admin API is always registered on this socket.
#[derive(Debug, Clone)]
pub struct RpcBuilder {
    /// The RPC socket address.
    pub socket: SocketAddr,
    /// Whether to serve the experimental `opstack` block-building namespace, see
    /// [`OpStackApiServer`](crate::OpStackApiServer).
    pub experimental_sequencer_api: bool,
}

impl RpcBuilder {
    /// Returns the socket address of the [`RpcBuilder`].
    pub const fn socket(&self) -> SocketAddr {
        self.socket
    }

    /// Sets the given [`SocketAddr`] on the [`RpcBuilder`].
    pub const fn set_addr(self, addr: SocketAddr) -> Self {
        Self { socket: addr, ..self }
    }
}
