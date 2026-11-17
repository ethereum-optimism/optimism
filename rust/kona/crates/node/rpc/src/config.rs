//! Contains the RPC Configuration.

use std::net::SocketAddr;

/// The RPC configuration.
///
/// The admin API is always registered on this socket.
#[derive(Debug, Clone)]
pub struct RpcBuilder {
    /// The RPC socket address.
    pub socket: SocketAddr,
}

impl RpcBuilder {
    /// Returns the socket address of the [`RpcBuilder`].
    pub const fn socket(&self) -> SocketAddr {
        self.socket
    }

    /// Sets the given [`SocketAddr`] on the [`RpcBuilder`].
    pub const fn set_addr(self, addr: SocketAddr) -> Self {
        Self { socket: addr }
    }
}
