//! RPC Server Actor

use crate::{NodeActor, RpcActorError};
use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle, middleware::http::ProxyGetRequestLayer},
};
use kona_rpc::RpcBuilder;
use std::time::Duration;

/// An actor that runs the JSON-RPC server for the rollup node.
///
/// The first launch happens upstream of this actor; restarts (up to
/// [`RpcBuilder::restart_count`]) are handled inside [`Self::step`].
#[derive(Debug)]
pub struct RpcActor {
    /// Config used to relaunch the server if it stops.
    config: RpcBuilder,
    /// Module set used to relaunch the server if it stops.
    modules: RpcModule<()>,
    /// The currently-running server handle. Replaced on each successful relaunch.
    handle: Option<ServerHandle>,
    /// Remaining relaunches allowed before [`Self::step`] returns
    /// [`RpcActorError::ServerStopped`].
    restarts_remaining: u32,
}

impl RpcActor {
    /// Constructs a new [`RpcActor`].
    ///
    /// `handle` is the live server returned by `launch`. The caller is responsible for the
    /// initial launch; this actor takes ownership of the running server and handles restarts.
    pub const fn new(config: RpcBuilder, modules: RpcModule<()>, handle: ServerHandle) -> Self {
        let restarts_remaining = config.restart_count();
        Self { config, modules, handle: Some(handle), restarts_remaining }
    }
}

impl Drop for RpcActor {
    fn drop(&mut self) {
        // jsonrpsee's ServerHandle is Arc<watch::Sender<()>>; dropping is enough to close the
        // watch and stop the server, but calling `stop()` explicitly is clearer about intent.
        // Errors here mean the server is already stopped.
        if let Some(handle) = self.handle.take() {
            let _ = handle.stop();
        }
    }
}

/// Launches the jsonrpsee [`Server`].
///
/// Callers invoke this once before constructing the [`RpcActor`]; the actor uses it again
/// internally to relaunch after a server stops, up to [`RpcBuilder::restart_count`] times.
///
/// ## Errors
///
/// - [`std::io::Error`] if the server fails to start.
pub(crate) async fn launch(
    config: &RpcBuilder,
    module: RpcModule<()>,
) -> Result<ServerHandle, std::io::Error> {
    let middleware = tower::ServiceBuilder::new()
        .layer(
            ProxyGetRequestLayer::new([("/healthz", "healthz")])
                .expect("Critical: Failed to build GET method proxy"),
        )
        .timeout(Duration::from_secs(2));
    let server = Server::builder().set_http_middleware(middleware).build(config.socket).await?;

    if let Ok(addr) = server.local_addr() {
        info!(target: "rpc", addr = ?addr, "RPC server bound to address");
    } else {
        error!(target: "rpc", "Failed to get local address for RPC server");
    }

    Ok(server.start(module))
}

#[async_trait]
impl NodeActor for RpcActor {
    type Error = RpcActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let handle = self.handle.clone().ok_or(RpcActorError::ServerStopped)?;
        handle.stopped().await;

        if self.restarts_remaining == 0 {
            return Err(RpcActorError::ServerStopped);
        }
        self.restarts_remaining -= 1;

        match launch(&self.config, self.modules.clone()).await {
            Ok(new_handle) => {
                self.handle = Some(new_handle);
                Ok(())
            }
            Err(err) => {
                error!(target: "rpc", ?err, "Failed to launch rpc server");
                Err(RpcActorError::ServerStopped)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use std::net::SocketAddr;

    use super::*;

    #[tokio::test]
    async fn test_launch_no_modules() {
        let launcher = RpcBuilder {
            socket: SocketAddr::from(([127, 0, 0, 1], 8080)),
            no_restart: false,
            enable_admin: false,
            admin_persistence: None,
            ws_enabled: false,
            dev_enabled: false,
        };
        let result = launch(&launcher, RpcModule::new(())).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_launch_with_modules() {
        let launcher = RpcBuilder {
            socket: SocketAddr::from(([127, 0, 0, 1], 8081)),
            no_restart: false,
            enable_admin: false,
            admin_persistence: None,
            ws_enabled: false,
            dev_enabled: false,
        };
        let mut modules = RpcModule::new(());

        modules.merge(RpcModule::new(())).expect("module merge");
        modules.merge(RpcModule::new(())).expect("module merge");
        modules.merge(RpcModule::new(())).expect("module merge");

        let result = launch(&launcher, modules).await;
        assert!(result.is_ok());
    }
}
