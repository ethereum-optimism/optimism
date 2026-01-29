//! Contains the RPC Configuration.

use std::{net::SocketAddr, path::PathBuf};

/// The RPC configuration.
#[derive(Debug, Clone)]
pub struct RpcBuilder {
    /// Prevent the rpc server from being restarted.
    pub no_restart: bool,
    /// The RPC socket address.
    pub socket: SocketAddr,
    /// Enable the admin API.
    pub enable_admin: bool,
    /// File path used to persist state changes made via the admin API so they persist across
    /// restarts.
    pub admin_persistence: Option<PathBuf>,
    /// Enable the websocket rpc server
    pub ws_enabled: bool,
    /// Enable development RPC endpoints
    pub dev_enabled: bool,
}

impl RpcBuilder {
    /// Returns whether WebSocket RPC endpoint is enabled
    pub const fn ws_enabled(&self) -> bool {
        self.ws_enabled
    }

    /// Returns whether development RPC endpoints are enabled
    pub const fn dev_enabled(&self) -> bool {
        self.dev_enabled
    }

    /// Returns the socket address of the [`RpcBuilder`].
    pub const fn socket(&self) -> SocketAddr {
        self.socket
    }

    /// Returns the number of times the RPC server will attempt to restart if it stops.
    pub const fn restart_count(&self) -> u32 {
        if self.no_restart { 0 } else { 3 }
    }

    /// Sets the given [`SocketAddr`] on the [`RpcBuilder`].
    pub fn set_addr(self, addr: SocketAddr) -> Self {
        Self { socket: addr, ..self }
    }
}
