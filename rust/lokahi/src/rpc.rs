//! The supernode's single RPC socket, and the chain-id routing on it.
//!
//! One process hosting N chains has one address, not N of them. A caller that has been told where
//! the supernode is asks it for a chain by putting the chain's id in the path — `/<l2-chain-id>` —
//! and the supernode's own namespaces answer at `/`. That is the addressing Go op-supernode
//! already serves (`op-supernode/supernode/resources/rpc_router.go`), so a client is pointed at
//! either implementation with the same URL and no branch.
//!
//! What routing buys over N sockets is not tidiness: a chain's port is a second thing an operator
//! has to configure, publish and firewall per chain, and a caller cannot learn it from the one
//! address it was given. With routing there is one listener to configure and the chain set is
//! discoverable — `lokahi_chains` names every route.
//!
//! ## How a chain's methods get here
//!
//! Each chain's method set is built by kona, inside [`RollupNode::compose`], from the channels
//! that chain's actors read. lokahi cannot rebuild it and does not try: it hands kona a launcher
//! ([`SharedRpcServerLauncher`]) which takes the finished [`RpcModule`] and registers it as a
//! route instead of binding it to a socket. So kona's methods reach the route as kona built them,
//! including the HTTP middleware a standalone kona-node serves them behind. What a route serves
//! *beyond* kona's set — Go op-supernode's chain routes answer `superroot_atTimestamp`, because
//! each virtual op-node registers that namespace itself — is deposited per chain through
//! [`SupernodeRpc::add_chain_methods`] and merged at registration.
//!
//! [`RollupNode::compose`]: kona_node_service::RollupNode::compose

use anyhow::{Context, Result};
use async_trait::async_trait;
use hyper::body::Incoming;
use jsonrpsee::{
    RpcModule,
    core::BoxError,
    server::{
        HttpBody, HttpRequest, HttpResponse, Methods, Server, ServerHandle, StopHandle,
        middleware::http::ProxyGetRequestLayer, serve_with_graceful_shutdown, stop_channel,
    },
};
use kona_node_service::{RpcServerHandle, RpcServerLauncher, SharedRpcServerLauncher};
use std::{
    collections::HashMap,
    convert::Infallible,
    fmt,
    future::Future,
    net::SocketAddr,
    pin::Pin,
    sync::{Arc, RwLock},
    time::Duration,
};
use tokio::{net::TcpListener, sync::watch};
use tokio_util::sync::CancellationToken;
use tower::ServiceExt as _;
use tracing::{debug, info, warn};

/// How often a request for a hosted chain that has not registered its route yet re-checks.
///
/// `defaultGatePoll` in `op-supernode/supernode/resources/rpc_router.go`.
const GATE_POLL: Duration = Duration::from_millis(25);

/// How long such a request waits before it is refused.
///
/// `defaultGateTimeout` in `op-supernode/supernode/resources/rpc_router.go`.
const GATE_TIMEOUT: Duration = Duration::from_secs(60);

/// The per-request timeout kona's own per-chain server applies, kept so that a routed chain
/// answers on the same budget as a standalone one.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(2);

/// What Go's `http.NotFound` writes: an unknown chain id is not a chain this process has.
const NOT_FOUND_BODY: &str = "404 page not found";

/// What Go's router writes for a hosted chain whose RPC is not up yet.
const NOT_READY_BODY: &str = "chain RPC not ready";

/// One route's request handler: a jsonrpsee service over one chain's method set, type-erased so
/// that the route table holds the chains' handlers and the root's in one shape.
type Handler = Arc<
    dyn Fn(
            HttpRequest<Incoming>,
        ) -> Pin<Box<dyn Future<Output = Result<HttpResponse, BoxError>> + Send>>
        + Send
        + Sync,
>;

/// The supernode's route table.
///
/// Seeded at construction with every chain the supernode was configured to host, each without a
/// handler. That is what lets the three cases a caller can be in be told apart: a chain this
/// process does not host is a `404`, a chain it hosts but has not composed yet is waited for, and
/// a composed chain is served. Without the seeding, a request that arrived while a chain was still
/// composing — which takes as long as reaching the L1 and an execution layer takes — would be
/// indistinguishable from a request for a chain that does not exist.
struct Routes {
    /// Route by chain id's decimal string: the path segment a caller writes.
    chains: RwLock<HashMap<String, Option<Handler>>>,
    /// Supernode-owned methods merged into a chain's route when it registers, by the same key.
    ///
    /// This is how a chain's route comes to serve more than kona's method set, the way the Go
    /// op-supernode's routes do: op-node registers a `superroot` namespace of its own, so every
    /// virtual node op-supernode serves under `/<chainID>` answers `superroot_atTimestamp`.
    /// kona builds its module set from its actors and lokahi does not reopen it; what lokahi
    /// serves per chain beyond it is deposited here and merged at registration.
    extras: RwLock<HashMap<String, Methods>>,
    /// The handler for `/`: the supernode's own namespaces.
    root: RwLock<Option<Handler>>,
    /// The stop handle every service built from this table holds.
    stop_handle: StopHandle,
    /// Held so that handle stays live for as long as the table does: dropping it stops every
    /// service built from it, which is how the socket stops answering when the supernode stops.
    _server_handle: ServerHandle,
}

impl fmt::Debug for Routes {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // The handlers are boxed closures and cannot be formatted; which chains have registered
        // one is the part worth printing.
        let chains = self.chains.read().map_or_else(
            |_| Vec::new(),
            |chains| {
                let mut rendered = chains
                    .iter()
                    .map(|(id, handler)| format!("{id}={}", handler.is_some()))
                    .collect::<Vec<_>>();
                rendered.sort();
                rendered
            },
        );
        f.debug_struct("Routes").field("chains", &chains).finish_non_exhaustive()
    }
}

/// Where a request's chain id landed.
enum Lookup {
    /// The supernode does not host this chain.
    Unknown,
    /// The supernode hosts it, but it did not register a route within [`GATE_TIMEOUT`].
    NotReady,
    /// The chain's handler.
    Ready(Handler),
}

impl Routes {
    /// Registers `methods` as chain `chain_id`'s route, replacing any handler it had.
    fn register(&self, chain_id: &str, methods: Methods) {
        let handler = handler(methods, self.stop_handle.clone());
        self.chains
            .write()
            .expect("the route table lock is never held across a panic")
            .insert(chain_id.to_owned(), Some(handler));
        info!(target: "lokahi", chain_id, "Chain RPC route registered at /{chain_id}");
    }

    /// Removes chain `chain_id`'s route.
    ///
    /// Removed rather than emptied, matching Go's `RemoveHandler`: a chain that has been torn down
    /// is not a chain this process hosts, so a request for it is answered like a request for any
    /// other chain it does not host rather than waiting a minute for a route that is not coming.
    fn remove(&self, chain_id: &str) {
        let _ = self
            .chains
            .write()
            .expect("the route table lock is never held across a panic")
            .remove(chain_id);
    }

    /// Looks `chain_id` up, waiting up to [`GATE_TIMEOUT`] for a hosted chain to register.
    async fn lookup(&self, chain_id: &str) -> Lookup {
        let deadline = tokio::time::Instant::now() + GATE_TIMEOUT;

        loop {
            // Scoped so the guard is dropped before the sleep below: a reader holding it across an
            // await would block the registration it is waiting for.
            let pending = {
                let chains =
                    self.chains.read().expect("the route table lock is never held across a panic");
                match chains.get(chain_id) {
                    None => return Lookup::Unknown,
                    Some(Some(handler)) => return Lookup::Ready(Arc::clone(handler)),
                    Some(None) => true,
                }
            };

            if pending && tokio::time::Instant::now() >= deadline {
                return Lookup::NotReady;
            }
            tokio::time::sleep(GATE_POLL).await;
        }
    }
}

/// Builds the service one method set is served by.
///
/// The server configuration and the HTTP middleware are the ones kona's own per-chain launcher
/// uses (`JsonrpseeServerLauncher`), so a routed chain behaves as a standalone one does: the same
/// request limits, the same per-request timeout, and `/healthz` answered as a `GET`. The middleware
/// sees the path with the chain segment already stripped, so a chain's health check is
/// `/<chain-id>/healthz` and everything under a chain's route reads exactly as it does on a
/// single-chain node's `/`.
fn handler(methods: Methods, stop_handle: StopHandle) -> Handler {
    let service = Server::builder()
        .set_http_middleware(
            tower::ServiceBuilder::new()
                .layer(
                    ProxyGetRequestLayer::new([("/healthz", "healthz")])
                        .expect("the healthz proxy path is a literal"),
                )
                .timeout(REQUEST_TIMEOUT),
        )
        .to_service_builder()
        .build(methods, stop_handle);

    Arc::new(move |request| {
        let service = service.clone();
        Box::pin(async move { service.oneshot(request).await })
    })
}

/// The supernode's RPC: one socket, the process-wide namespaces at `/`, one route per chain.
#[derive(Debug)]
pub(crate) struct SupernodeRpc {
    /// The route table the accept loop reads.
    routes: Arc<Routes>,
    /// The address the listener actually got, which is the requested one unless port 0 was asked
    /// for.
    addr: SocketAddr,
    /// Stops the accept loop and gracefully shuts every live connection down.
    shutdown: CancellationToken,
}

impl SupernodeRpc {
    /// Binds `socket` and starts serving, with a route seeded for each of `chain_ids`.
    ///
    /// Bound before any chain is composed, so the address it logs is available to a caller that
    /// launched this process and has to wait for something. Requests that arrive in that window
    /// are refused with [`NOT_READY_BODY`] rather than with a 404, because the chain is coming.
    pub(crate) async fn bind(
        socket: SocketAddr,
        chain_ids: impl IntoIterator<Item = u64>,
    ) -> Result<Self> {
        let listener = TcpListener::bind(socket)
            .await
            .with_context(|| format!("failed to bind the supernode rpc to {socket}"))?;
        let addr =
            listener.local_addr().context("the supernode rpc server has no local address")?;

        let (stop_handle, server_handle) = stop_channel();
        let routes = Arc::new(Routes {
            chains: RwLock::new(chain_ids.into_iter().map(|id| (id.to_string(), None)).collect()),
            extras: RwLock::new(HashMap::new()),
            root: RwLock::new(None),
            stop_handle,
            _server_handle: server_handle,
        });

        let shutdown = CancellationToken::new();
        tokio::spawn(accept(listener, Arc::clone(&routes), shutdown.clone()));

        // The bound address, not the requested one: a harness that asked for port 0 learns its
        // port from this line, and this is the line an out-of-process launch waits for. The
        // wording is the one this line had when the socket served only the admin API, because its
        // job — *here is the supernode's address* — is the same, and the address is now the
        // address of everything the process serves.
        info!(target: "lokahi", %addr, "Admin RPC server bound to address");

        Ok(Self { routes, addr, shutdown })
    }

    /// The address the supernode answers on.
    pub(crate) const fn addr(&self) -> SocketAddr {
        self.addr
    }

    /// Mounts the supernode's own namespaces at `/`.
    pub(crate) fn set_root(&self, methods: impl Into<Methods>) {
        let handler = handler(methods.into(), self.routes.stop_handle.clone());
        *self.routes.root.write().expect("the root lock is never held across a panic") =
            Some(handler);
    }

    /// The launcher chain `chain_id`'s RPC module set is handed to instead of a socket.
    pub(crate) fn launcher(&self, chain_id: u64) -> SharedRpcServerLauncher {
        Arc::new(ChainLauncher { routes: Arc::clone(&self.routes), chain_id: chain_id.to_string() })
    }

    /// Adds supernode-owned methods to chain `chain_id`'s route, merged when the route registers.
    ///
    /// Must be called before the chain is composed, because kona registers the route *while*
    /// composing: the supernode deposits each chain's extra methods first and only then composes
    /// it, so a route never registers before its extras are here. The methods therefore exist
    /// before the state they answer from — the same ordering the root's query API lives with —
    /// and hold a handle that is filled once the chain exists. A method that collides with one
    /// kona already serves fails the chain's launch loudly rather than serving one of the two
    /// quietly — if kona grows one of these methods natively, the right move is to stop supplying
    /// it here, not to shadow it.
    pub(crate) fn add_chain_methods(&self, chain_id: u64, methods: impl Into<Methods>) {
        let _ = self
            .routes
            .extras
            .write()
            .expect("the extras lock is never held across a panic")
            .insert(chain_id.to_string(), methods.into());
    }
}

impl Drop for SupernodeRpc {
    fn drop(&mut self) {
        self.shutdown.cancel();
    }
}

/// Accepts connections until the supernode stops.
async fn accept(listener: TcpListener, routes: Arc<Routes>, shutdown: CancellationToken) {
    loop {
        let stream = tokio::select! {
            () = shutdown.cancelled() => return,
            accepted = listener.accept() => match accepted {
                Ok((stream, _peer)) => stream,
                Err(err) => {
                    // One failed accept is not the listener being gone; the next one is retried.
                    warn!(target: "lokahi", %err, "The supernode rpc failed to accept");
                    continue;
                }
            },
        };

        let routes = Arc::clone(&routes);
        let shutdown = shutdown.clone();
        let _handle = tokio::spawn(async move {
            let service = tower::service_fn(move |request: HttpRequest<Incoming>| {
                let routes = Arc::clone(&routes);
                // Infallible rather than `BoxError`: a service whose error type is a boxed trait
                // object cannot satisfy the transport's `Into<BoxError>` bound for every lifetime,
                // and a handler failure has a better answer than a dropped connection anyway.
                async move { Ok::<_, Infallible>(route(&routes, request).await) }
            });
            if let Err(err) =
                serve_with_graceful_shutdown(stream, service, shutdown.cancelled_owned()).await
            {
                // A client that hangs up mid-request lands here, which is ordinary.
                debug!(target: "lokahi", %err, "A supernode rpc connection ended with an error");
            }
        });
    }
}

/// Routes one request by the first path segment.
///
/// Mirrors `Router.ServeHTTP` in `op-supernode/supernode/resources/rpc_router.go`: an empty first
/// segment is the root, an unknown chain id is a `404`, and a hosted chain that has not registered
/// its route yet is waited for and then refused with `503`. The path handed downstream is the
/// remainder, so the chain's own handler sees the path it would see on a single-chain node.
async fn route(routes: &Routes, request: HttpRequest<Incoming>) -> HttpResponse {
    let path = request.uri().path().to_owned();
    let (chain_id, remainder) = split_first_segment(&path);

    if chain_id.is_empty() {
        let Some(root) =
            routes.root.read().expect("the root lock is never held across a panic").clone()
        else {
            return plain_text(404, NOT_FOUND_BODY);
        };
        return call(&root, request).await;
    }

    match routes.lookup(chain_id).await {
        Lookup::Unknown => plain_text(404, NOT_FOUND_BODY),
        Lookup::NotReady => plain_text(503, NOT_READY_BODY),
        Lookup::Ready(handler) => call(&handler, with_path(request, remainder)).await,
    }
}

/// Calls one route's handler, answering `500` rather than dropping the connection if it fails.
///
/// The only failures reachable here are the HTTP middleware's: a request that outran
/// [`REQUEST_TIMEOUT`], or a body that could not be read. A standalone kona-node hands those to
/// hyper, which aborts the connection; a caller is better served by a status it can read.
async fn call(handler: &Handler, request: HttpRequest<Incoming>) -> HttpResponse {
    match handler(request).await {
        Ok(response) => response,
        Err(err) => {
            warn!(target: "lokahi", %err, "A supernode rpc handler failed");
            plain_text(500, "internal error")
        }
    }
}

/// Splits the first non-empty path segment off, returning it and the remainder starting with `/`.
///
/// `splitFirstSegment` in `op-supernode/supernode/resources/rpc_router.go`, including its handling
/// of a path with no second segment (remainder `/`) and of a leading `//` (an empty first segment,
/// which is the root).
fn split_first_segment(path: &str) -> (&str, &str) {
    let path = path.strip_prefix('/').unwrap_or(path);
    if path.is_empty() {
        return ("", "/");
    }
    path.find('/').map_or((path, "/"), |index| (&path[..index], &path[index..]))
}

/// Replaces a request's path, keeping its query string.
///
/// Go rewrites `URL.Path` before delegating for the same reason: the chain's handler is the one a
/// single-chain node serves at `/`, so it has to be handed the path with the chain segment gone —
/// `/<chain-id>/healthz` is that chain's `/healthz`.
fn with_path(mut request: HttpRequest<Incoming>, path: &str) -> HttpRequest<Incoming> {
    let mut parts = request.uri().clone().into_parts();
    let path_and_query =
        request.uri().query().map_or_else(|| path.to_owned(), |query| format!("{path}?{query}"));

    // Both fallible steps are infallible here: `path` is a suffix of a path that already parsed,
    // and the scheme and authority are carried over from the same URI so the parts stay
    // consistent. A request whose path could not be rebuilt is passed through unchanged rather
    // than refused — the chain would then see the full path, which is worse than nothing but not
    // a reason to fail the call.
    let Ok(path_and_query) = path_and_query.parse() else { return request };
    parts.path_and_query = Some(path_and_query);
    let Ok(uri) = http::Uri::from_parts(parts) else { return request };
    *request.uri_mut() = uri;
    request
}

/// A plain-text response shaped like the one Go's `http.Error` writes, headers included, so a
/// client that reads either implementation's refusal reads the same thing.
fn plain_text(status: u16, message: &str) -> HttpResponse {
    http::Response::builder()
        .status(status)
        .header("content-type", "text/plain; charset=utf-8")
        .header("x-content-type-options", "nosniff")
        .body(HttpBody::from(format!("{message}\n")))
        .expect("a response built from a literal status and header")
}

/// The launcher one hosted chain's RPC module set is handed to.
#[derive(Debug)]
struct ChainLauncher {
    /// The table the chain's route is registered in.
    routes: Arc<Routes>,
    /// The chain id, as the path segment its route answers on.
    chain_id: String,
}

#[async_trait]
impl RpcServerLauncher for ChainLauncher {
    type Handle = Route;

    async fn launch(&self, mut modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
        // The supernode's own additions to this chain's route, deposited before the actors
        // started. Merged into kona's set rather than replacing anything in it; a collision is a
        // real conflict — two implementations of one method on one route — and fails the launch
        // with the method's name rather than picking one silently.
        let extra = self
            .routes
            .extras
            .read()
            .expect("the extras lock is never held across a panic")
            .get(&self.chain_id)
            .cloned();
        if let Some(extra) = extra {
            modules.merge(extra).map_err(|err| {
                std::io::Error::other(format!(
                    "chain {} route: a supernode-supplied method collides with kona's: {err}",
                    self.chain_id
                ))
            })?;
        }
        self.routes.register(&self.chain_id, modules.into());
        Ok(Route::new(Arc::clone(&self.routes), self.chain_id.clone()))
    }
}

/// A registered route, held by the chain's RPC actor for as long as the chain runs.
///
/// A route is not a socket that can be lost, so unlike a bound server this never stops on its own:
/// the actor's `stopped` resolves only once the chain is torn down and the route removed, and
/// there is correspondingly nothing to restart.
#[derive(Debug)]
struct Route {
    /// The table this route lives in.
    routes: Arc<Routes>,
    /// The chain id whose route this is.
    chain_id: String,
    /// Flipped when the route is removed. A `watch` rather than a notification so that a waiter
    /// which arrives after the removal still observes it.
    removed: watch::Sender<bool>,
    /// An idle receiver, so [`watch::Sender::send`] still succeeds when nothing is waiting.
    _keepalive: Arc<watch::Receiver<bool>>,
}

impl Route {
    fn new(routes: Arc<Routes>, chain_id: String) -> Self {
        let (removed, keepalive) = watch::channel(false);
        Self { routes, chain_id, removed, _keepalive: Arc::new(keepalive) }
    }
}

#[async_trait]
impl RpcServerHandle for Route {
    async fn stopped(&self) {
        let mut removed = self.removed.subscribe();
        if *removed.borrow() {
            return;
        }
        let _ = removed.changed().await;
    }

    fn stop(&self) {
        self.routes.remove(&self.chain_id);
        let _ = self.removed.send(true);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_first_segment_is_the_chain_id_and_the_rest_is_the_path() {
        assert_eq!(split_first_segment("/901"), ("901", "/"));
        assert_eq!(split_first_segment("/901/"), ("901", "/"));
        assert_eq!(split_first_segment("/901/healthz"), ("901", "/healthz"));
        assert_eq!(split_first_segment("/901/a/b"), ("901", "/a/b"));
    }

    #[test]
    fn a_path_with_no_chain_id_is_the_root() {
        assert_eq!(split_first_segment(""), ("", "/"));
        assert_eq!(split_first_segment("/"), ("", "/"));
        // One leading slash is stripped, as Go's `strings.TrimPrefix` strips one: `//901` has an
        // empty first segment and is therefore the root, not chain 901.
        assert_eq!(split_first_segment("//901"), ("", "/901"));
    }

    #[test]
    fn an_unknown_chain_reads_like_gos_not_found() {
        let response = plain_text(404, NOT_FOUND_BODY);
        assert_eq!(response.status(), 404);
        assert_eq!(response.headers()["content-type"], "text/plain; charset=utf-8");
        assert_eq!(response.headers()["x-content-type-options"], "nosniff");
    }
}
