//! Server launcher and handle traits used by [`crate::RpcActor`].
//!
//! These traits decouple [`crate::RpcActor`] from the concrete [`jsonrpsee::server::Server`] so
//! the actor's restart logic can be unit-tested with a controllable mock. The production
//! [`JsonrpseeServerLauncher`] implementation is a thin pass-through over jsonrpsee.

use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerHandle,
        middleware::{
            http::ProxyGetRequestLayer,
            rpc::{Batch, Notification, Request, RpcServiceBuilder, RpcServiceT},
        },
    },
};
use kona_metrics::Label;
use kona_rpc::RpcBuilder;
use std::{future::Future, sync::Arc, time::Duration};

/// A handle to a running RPC server.
///
/// The actor awaits [`Self::stopped`] to detect that the server has terminated, and calls
/// [`Self::stop`] from its `Drop` impl to request graceful termination on shutdown.
#[async_trait]
pub trait RpcServerHandle: Send + Sync + 'static {
    /// Resolves when the server has stopped.
    async fn stopped(&self);

    /// Requests that the server stop. May be a no-op if already stopped.
    fn stop(&self);
}

/// Launches an RPC server bound to the configuration carried by the implementor.
#[async_trait]
pub trait RpcServerLauncher: Send + Sync + 'static {
    /// The handle type produced by a successful [`Self::launch`].
    type Handle: RpcServerHandle;

    /// Launches a new server instance bound to `modules`.
    async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error>;
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
    chain: Label,
}

impl JsonrpseeServerLauncher {
    /// Wraps an [`RpcBuilder`] for use as a launcher.
    ///
    /// `chain` labels the metrics the RPC handlers emit, via a [`ChainScope`] middleware.
    pub const fn new(config: RpcBuilder, chain: Label) -> Self {
        Self { config, chain }
    }
}

#[async_trait]
impl RpcServerLauncher for JsonrpseeServerLauncher {
    type Handle = ServerHandle;

    async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
        launch(&self.config, self.chain.clone(), modules).await
    }
}

/// An RPC middleware that puts a chain scope around every call jsonrpsee serves, which it
/// otherwise serves on tasks that inherit no scope.
///
/// Register with `RpcServiceBuilder::new().layer_fn(ChainScope::layer(chain))`.
///
/// Covers every `#[method]` handler. It does not cover a subscription or a blocking method, which
/// jsonrpsee runs under `tokio::spawn` and `spawn_blocking`; one that emits a metric would need
/// its own [`kona_metrics::scoped`].
#[derive(Debug, Clone)]
pub struct ChainScope<S> {
    // The returned future is bound to the request lifetime, not `&self`, so the inner service
    // must be owned rather than borrowed.
    inner: Arc<S>,
    chain: Label,
}

impl<S> ChainScope<S> {
    /// A `layer_fn` closure that wraps an inner service in this scope.
    pub fn layer(chain: Label) -> impl Fn(S) -> Self + Clone {
        move |inner| Self { inner: Arc::new(inner), chain: chain.clone() }
    }
}

impl<S> RpcServiceT for ChainScope<S>
where
    S: RpcServiceT + Send + Sync + 'static,
{
    type MethodResponse = S::MethodResponse;
    type NotificationResponse = S::NotificationResponse;
    type BatchResponse = S::BatchResponse;

    fn call<'a>(
        &self,
        request: Request<'a>,
    ) -> impl Future<Output = Self::MethodResponse> + Send + 'a {
        let inner = self.inner.clone();
        kona_metrics::scoped(self.chain.clone(), async move { inner.call(request).await })
    }

    // jsonrpsee dispatches a batch through its own `call`, not ours.
    fn batch<'a>(
        &self,
        requests: Batch<'a>,
    ) -> impl Future<Output = Self::BatchResponse> + Send + 'a {
        let inner = self.inner.clone();
        kona_metrics::scoped(self.chain.clone(), async move { inner.batch(requests).await })
    }

    fn notification<'a>(
        &self,
        notification: Notification<'a>,
    ) -> impl Future<Output = Self::NotificationResponse> + Send + 'a {
        let inner = self.inner.clone();
        kona_metrics::scoped(
            self.chain.clone(),
            async move { inner.notification(notification).await },
        )
    }
}

/// Launches the jsonrpsee [`Server`].
///
/// ## Errors
///
/// - [`std::io::Error`] if the server fails to start.
async fn launch(
    config: &RpcBuilder,
    chain: Label,
    module: RpcModule<()>,
) -> Result<ServerHandle, std::io::Error> {
    let http_middleware = tower::ServiceBuilder::new()
        .layer(
            ProxyGetRequestLayer::new([("/healthz", "healthz")])
                .expect("Critical: Failed to build GET method proxy"),
        )
        .timeout(Duration::from_secs(2));
    let rpc_middleware = RpcServiceBuilder::new().layer_fn(ChainScope::layer(chain));
    let server = Server::builder()
        .set_http_middleware(http_middleware)
        .set_rpc_middleware(rpc_middleware)
        .build(config.socket)
        .await?;

    if let Ok(addr) = server.local_addr() {
        info!(target: "rpc", addr = ?addr, "RPC server bound to address");
    } else {
        error!(target: "rpc", "Failed to get local address for RPC server");
    }

    Ok(server.start(module))
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::{JsonrpseeServerLauncher, RpcServerLauncher};
    use crate::test_metrics::chains_of;
    use jsonrpsee::{RpcModule, core::client::ClientT, http_client::HttpClientBuilder, rpc_params};
    use kona_rpc::RpcBuilder;

    const METRIC: &str = "kona_test_rpc_calls";

    const fn builder(socket: std::net::SocketAddr) -> RpcBuilder {
        RpcBuilder {
            no_restart: true,
            socket,
            enable_admin: false,
            admin_persistence: None,
            ws_enabled: false,
            dev_enabled: false,
        }
    }

    /// The RPC handlers run on jsonrpsee's own tasks, so the middleware is the only thing that
    /// can put them in a chain scope.
    #[tokio::test]
    async fn rpc_handler_metrics_carry_the_launcher_chain() {
        // Installs the recorder before anything registers.
        assert!(chains_of(METRIC).is_empty());

        let mut module = RpcModule::new(());
        module
            .register_method(METRIC, |_, _, _| {
                metrics::counter!(METRIC).increment(1);
                ""
            })
            .expect("the method registers");

        // `launch` returns only a `ServerHandle`, so probe for a free port first. The port is
        // free between the probe and the bind, so retry if something else takes it.
        let mut launched = None;
        for _ in 0..8 {
            let socket = std::net::TcpListener::bind("127.0.0.1:0")
                .expect("a port is free")
                .local_addr()
                .expect("the listener reports its address");

            let launcher =
                JsonrpseeServerLauncher::new(builder(socket), kona_metrics::chain_label(901));
            match launcher.launch(module.clone()).await {
                Ok(handle) => {
                    launched = Some((socket, handle));
                    break;
                }
                Err(e) if e.kind() != std::io::ErrorKind::AddrInUse => {
                    panic!("the server must start: {e}")
                }
                Err(_) => {}
            }
        }
        let (socket, _handle) = launched.expect("a port stayed free for one bind in eight");

        let client = HttpClientBuilder::default()
            .build(format!("http://{socket}"))
            .expect("the client builds");
        let _: String = client.request(METRIC, rpc_params![]).await.expect("the call succeeds");

        assert_eq!(
            chains_of(METRIC),
            vec![Some("901".to_string())],
            "the RPC call must be attributed"
        );
    }
}
