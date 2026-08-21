//! Server launcher and handle traits used by [`crate::RpcActor`].
//!
//! These traits decouple [`crate::RpcActor`] from the concrete [`jsonrpsee::server::Server`] so
//! the actor's restart logic can be unit-tested with a controllable mock. The production
//! [`JsonrpseeServerLauncher`] implementation is a thin pass-through over jsonrpsee.

use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle, middleware::http::ProxyGetRequestLayer},
};
use kona_rpc::RpcBuilder;
use std::{fmt::Debug, sync::Arc, time::Duration};

/// A handle to a running RPC server.
///
/// The actor awaits [`Self::stopped`] to detect that the server has terminated, and calls
/// [`Self::stop`] from its `Drop` impl to request graceful termination on shutdown.
#[async_trait]
pub trait RpcServerHandle: Debug + Send + Sync + 'static {
    /// Resolves when the server has stopped.
    async fn stopped(&self);

    /// Requests that the server stop. May be a no-op if already stopped.
    fn stop(&self);
}

#[async_trait]
impl RpcServerHandle for Box<dyn RpcServerHandle> {
    async fn stopped(&self) {
        (**self).stopped().await;
    }

    fn stop(&self) {
        (**self).stop();
    }
}

/// Launches an RPC server bound to the configuration carried by the implementor.
#[async_trait]
pub trait RpcServerLauncher: Debug + Send + Sync + 'static {
    /// The handle type produced by a successful [`Self::launch`].
    type Handle: RpcServerHandle;

    /// Launches a new server instance bound to `modules`.
    async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error>;
}

/// The object-safe form of [`RpcServerLauncher`], so a host can supply its own launcher without
/// the node's actor types becoming generic over it.
///
/// Every [`RpcServerLauncher`] is one of these, and [`SharedRpcServerLauncher`] is in turn an
/// [`RpcServerLauncher`] again, so a host hands in `Arc::new(its_launcher)` and the node keeps a
/// single concrete actor type either way.
#[async_trait]
pub trait DynRpcServerLauncher: Debug + Send + Sync + 'static {
    /// Launches a new server instance bound to `modules`, behind a boxed handle.
    async fn launch_boxed(
        &self,
        modules: RpcModule<()>,
    ) -> Result<Box<dyn RpcServerHandle>, std::io::Error>;
}

#[async_trait]
impl<L> DynRpcServerLauncher for L
where
    L: RpcServerLauncher,
{
    async fn launch_boxed(
        &self,
        modules: RpcModule<()>,
    ) -> Result<Box<dyn RpcServerHandle>, std::io::Error> {
        let handle = self.launch(modules).await?;
        Ok(Box::new(handle))
    }
}

/// The launcher a composed chain's [`crate::RpcActor`] holds.
///
/// Shared rather than owned because a multi-chain host's launcher is one object that knows about
/// every chain — it routes to them — so each chain's actor holds a handle to the same one.
pub type SharedRpcServerLauncher = Arc<dyn DynRpcServerLauncher>;

#[async_trait]
impl RpcServerLauncher for SharedRpcServerLauncher {
    type Handle = Box<dyn RpcServerHandle>;

    async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
        (**self).launch_boxed(modules).await
    }
}

#[async_trait]
impl RpcServerHandle for ServerHandle {
    async fn stopped(&self) {
        self.clone().stopped().await;
    }

    fn stop(&self) {
        // jsonrpsee returns `Err` only when the server is already stopped; for the actor's
        // purposes that is indistinguishable from success.
        //
        // UFCS is required here to disambiguate: `self.stop()` and `Self::stop(self)` would both
        // resolve to this trait method (infinite recursion). We want the inherent
        // `ServerHandle::stop` from jsonrpsee, which clippy's `use_self` lint can't model.
        #[allow(clippy::use_self)]
        let _ = ServerHandle::stop(self);
    }
}

/// Production [`RpcServerLauncher`] backed by [`jsonrpsee::server::Server`].
#[derive(Debug, Clone)]
pub struct JsonrpseeServerLauncher {
    config: RpcBuilder,
}

impl JsonrpseeServerLauncher {
    /// Wraps an [`RpcBuilder`] for use as a launcher.
    pub const fn new(config: RpcBuilder) -> Self {
        Self { config }
    }
}

#[async_trait]
impl RpcServerLauncher for JsonrpseeServerLauncher {
    type Handle = ServerHandle;

    async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
        launch(&self.config, modules).await
    }
}

/// Launches the jsonrpsee [`Server`].
///
/// ## Errors
///
/// - [`std::io::Error`] if the server fails to start.
async fn launch(
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
