//! Contains the RPC Configuration.

use std::{net::SocketAddr, path::PathBuf};

/// The RPC configuration.
#[derive(Debug, Clone)]
pub struct RpcBuilder {
    /// The RPC socket address.
    pub socket: SocketAddr,
    /// Enable the admin API.
    pub enable_admin: bool,
    /// File path used to persist state changes made via the admin API so they persist across
    /// node restarts.
    pub admin_persistence: Option<PathBuf>,
}

impl RpcBuilder {
    /// Returns whether the admin API namespace is enabled.
    pub const fn enable_admin(&self) -> bool {
        self.enable_admin
    }

    /// Returns the socket address of the [`RpcBuilder`].
    pub const fn socket(&self) -> SocketAddr {
        self.socket
    }

    /// Sets the given [`SocketAddr`] on the [`RpcBuilder`].
    pub fn set_addr(self, addr: SocketAddr) -> Self {
        Self { socket: addr, ..self }
    }
}
