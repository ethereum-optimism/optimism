//! Minimal supervisor RPC server implementation

#[cfg(feature = "server")]
use alloy_rpc_types_engine::JwtSecret;
#[cfg(feature = "server")]
use jsonrpsee::server::ServerHandle;
#[cfg(feature = "server")]
use kona_interop::{ControlEvent, ManagedEvent};
#[cfg(feature = "server")]
use std::net::SocketAddr;
#[cfg(feature = "server")]
use tokio::sync::broadcast;

/// Minimal supervisor RPC server
#[cfg(feature = "server")]
#[derive(Debug)]
pub struct SupervisorRpcServer {
    /// A channel to receive [`ManagedEvent`] from the node.
    #[allow(dead_code)]
    managed_events: broadcast::Receiver<ManagedEvent>,
    /// A channel to send [`ControlEvent`].
    #[allow(dead_code)]
    control_events: broadcast::Sender<ControlEvent>,
    /// A JWT token for authentication.
    #[allow(dead_code)]
    jwt_token: JwtSecret,
    /// The socket address for the RPC server.
    socket: SocketAddr,
}

#[cfg(feature = "server")]
impl SupervisorRpcServer {
    /// Creates a new instance of the `SupervisorRpcServer`.
    pub const fn new(
        managed_events: broadcast::Receiver<ManagedEvent>,
        control_events: broadcast::Sender<ControlEvent>,
        jwt_token: JwtSecret,
        socket: SocketAddr,
    ) -> Self {
        Self { managed_events, control_events, jwt_token, socket }
    }

    /// Returns the socket address for the RPC server.
    pub const fn socket(&self) -> SocketAddr {
        self.socket
    }

    /// Launches the RPC server with the given socket address.
    pub async fn launch(self) -> std::io::Result<ServerHandle> {
        let server = jsonrpsee::server::ServerBuilder::default().build(self.socket).await?;
        // For now, start without any RPC methods - this is a minimal implementation
        let module = jsonrpsee::RpcModule::new(());
        Ok(server.start(module))
    }
}
