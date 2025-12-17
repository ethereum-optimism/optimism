use crate::client::http::HttpClient;
use crate::{Request, Response, from_buffered_request, into_buffered_request};
use http_body_util::BodyExt as _;
use jsonrpsee::core::BoxError;
use jsonrpsee::server::HttpBody;
use std::task::{Context, Poll};
use std::{future::Future, pin::Pin};
use tower::{Layer, Service};
use tracing::info;

const ENGINE_METHOD: &str = "engine_";

/// Requests that should be forwarded to both the builder and default execution client
const FORWARD_REQUESTS: [&str; 6] = [
    "eth_sendRawTransaction",
    "eth_sendRawTransactionConditional",
    "miner_setExtra",
    "miner_setGasPrice",
    "miner_setGasLimit",
    "miner_setMaxDASize",
];

#[derive(Debug, Clone)]
pub struct ProxyLayer {
    l2_client: HttpClient,
    builder_client: HttpClient,
}

impl ProxyLayer {
    pub fn new(l2_client: HttpClient, builder_client: HttpClient) -> Self {
        ProxyLayer {
            l2_client,
            builder_client,
        }
    }
}

impl<S> Layer<S> for ProxyLayer {
    type Service = ProxyService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        ProxyService {
            inner,
            l2_client: self.l2_client.clone(),
            builder_client: self.builder_client.clone(),
        }
    }
}

#[derive(Clone)]
pub struct ProxyService<S> {
    inner: S,
    l2_client: HttpClient,
    builder_client: HttpClient,
}

// Consider using `RpcServiceT` when https://github.com/paritytech/jsonrpsee/pull/1521 is merged
impl<S> Service<Request> for ProxyService<S>
where
    S: Service<Request, Response = Response> + Send + Sync + Clone + 'static,
    S::Response: 'static,
    S::Error: Into<BoxError> + 'static,
    S::Future: Send + 'static,
{
    type Response = Response;
    type Error = BoxError;
    type Future =
        Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send + 'static>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, req: Request) -> Self::Future {
        #[derive(serde::Deserialize, Debug)]
        struct RpcRequest<'a> {
            #[serde(borrow)]
            method: &'a str,
        }

        // See https://github.com/tower-rs/tower/blob/abb375d08cf0ba34c1fe76f66f1aba3dc4341013/tower-service/src/lib.rs#L276
        // for an explanation of this pattern
        let mut service = self.clone();
        service.inner = std::mem::replace(&mut self.inner, service.inner);

        let fut = async move {
            let buffered = into_buffered_request(req).await?;
            let body_bytes = buffered.clone().collect().await?.to_bytes();

            // Deserialize the bytes to find the method
            let method = serde_json::from_slice::<RpcRequest>(&body_bytes)?
                .method
                .to_string();

            // If the request is an Engine API method, call the inner RollupBoostServer
            if method.starts_with(ENGINE_METHOD) {
                info!(target: "proxy::call", message = "proxying request to rollup-boost server", ?method);
                return service
                    .inner
                    .call(from_buffered_request(buffered))
                    .await
                    .map_err(|e| e.into());
            }

            if FORWARD_REQUESTS.contains(&method.as_str()) {
                // If the request should be forwarded, send to both the
                // default execution client and the builder
                let method_clone = method.clone();
                let buffered_clone = buffered.clone();
                let mut builder_client = service.builder_client.clone();

                // Fire and forget the builder request
                tokio::spawn(async move {
                    let _ = builder_client.forward(buffered_clone, method_clone).await;
                });
            }

            // Return the response from the L2 client
            service
                .l2_client
                .forward(buffered, method)
                .await
                .map(|res| res.map(HttpBody::new))
        };

        Box::pin(fut)
    }
}

#[cfg(test)]
mod tests {
    use crate::ClientArgs;
    use crate::payload::PayloadSource;
    use crate::probe::ProbeLayer;

    use super::*;
    use alloy_primitives::{B256, Bytes, U64, U128, hex};
    use alloy_rpc_types_engine::JwtSecret;
    use alloy_rpc_types_eth::erc4337::TransactionConditional;
    use http::{StatusCode, Uri};
    use http_body_util::{BodyExt, Full};
    use hyper::service::service_fn;
    use hyper_util::client::legacy::Client;
    use hyper_util::client::legacy::connect::HttpConnector;
    use hyper_util::rt::{TokioExecutor, TokioIo};
    use jsonrpsee::server::Server;
    use jsonrpsee::types::{ErrorCode, ErrorObject};
    use jsonrpsee::{
        RpcModule,
        core::{ClientError, client::ClientT},
        http_client::HttpClient,
        rpc_params,
        server::{ServerBuilder, ServerHandle},
    };
    use serde_json::json;
    use std::{net::SocketAddr, sync::Arc};
    use tokio::net::TcpListener;
    use tokio::task::JoinHandle;

    // A JSON-RPC error is retriable if error.code ∉ (-32700, -32600]
    fn is_retriable_code(code: i32) -> bool {
        code < -32700 || code > -32600
    }

    struct TestHarness {
        builder: MockHttpServer,
        l2: MockHttpServer,
        server_handle: ServerHandle,
        proxy_client: HttpClient,
    }

    impl Drop for TestHarness {
        fn drop(&mut self) {
            self.server_handle.stop().unwrap();
        }
    }

    impl TestHarness {
        async fn new() -> eyre::Result<Self> {
            let builder = MockHttpServer::serve().await?;
            let l2 = MockHttpServer::serve().await?;
            let middleware = tower::ServiceBuilder::new().layer(ProxyLayer::new(
                ClientArgs {
                    url: format!("http://{}:{}", l2.addr.ip(), l2.addr.port()).parse::<Uri>()?,
                    jwt_token: Some(JwtSecret::random()),
                    jwt_path: None,
                    timeout: 1,
                }
                .new_http_client(PayloadSource::L2)
                .unwrap(),
                ClientArgs {
                    url: format!("http://{}:{}", builder.addr.ip(), builder.addr.port())
                        .parse::<Uri>()?,
                    jwt_token: Some(JwtSecret::random()),
                    jwt_path: None,
                    timeout: 1,
                }
                .new_http_client(PayloadSource::Builder)?,
            ));

            let temp_listener = TcpListener::bind("127.0.0.1:0").await?;
            let server_addr = temp_listener.local_addr()?;
            drop(temp_listener);

            let server = Server::builder()
                .set_http_middleware(middleware.clone())
                .build(server_addr)
                .await?;

            let server_addr = server.local_addr()?;
            let proxy_client: HttpClient = HttpClient::builder().build(format!(
                "http://{}:{}",
                server_addr.ip(),
                server_addr.port()
            ))?;

            let server_handle = server.start(RpcModule::new(()));

            Ok(Self {
                builder,
                l2,
                server_handle,
                proxy_client,
            })
        }
    }

    struct MockHttpServer {
        addr: SocketAddr,
        requests: Arc<tokio::sync::Mutex<Vec<serde_json::Value>>>,
        join_handle: JoinHandle<()>,
        shutdown_tx: Option<tokio::sync::oneshot::Sender<()>>,
        connections: Arc<tokio::sync::Mutex<Vec<JoinHandle<()>>>>,
    }

    impl Drop for MockHttpServer {
        fn drop(&mut self) {
            // Send shutdown signal if available
            if let Some(tx) = self.shutdown_tx.take() {
                let _ = tx.send(());
            }
            // Abort active connections to simulate a crash closing open sockets
            if let Ok(mut conns) = self.connections.try_lock() {
                for handle in conns.drain(..) {
                    handle.abort();
                }
            }
            self.join_handle.abort();
        }
    }

    impl MockHttpServer {
        async fn serve() -> eyre::Result<Self> {
            let listener = TcpListener::bind("127.0.0.1:0").await?;
            let addr = listener.local_addr()?;
            Self::serve_with_listener(listener, addr).await
        }

        async fn serve_on_addr(addr: SocketAddr) -> eyre::Result<Self> {
            let listener = TcpListener::bind(addr).await?;
            let actual_addr = listener.local_addr()?;
            Self::serve_with_listener(listener, actual_addr).await
        }

        async fn serve_with_listener(
            listener: TcpListener,
            addr: SocketAddr,
        ) -> eyre::Result<Self> {
            let requests = Arc::new(tokio::sync::Mutex::new(vec![]));
            let (shutdown_tx, mut shutdown_rx) = tokio::sync::oneshot::channel();
            let connections: Arc<tokio::sync::Mutex<Vec<JoinHandle<()>>>> =
                Arc::new(tokio::sync::Mutex::new(Vec::new()));

            let requests_clone = requests.clone();
            let connections_clone = connections.clone();
            let handle = tokio::spawn(async move {
                loop {
                    tokio::select! {
                        _ = &mut shutdown_rx => {
                            // Shutdown signal received
                            break;
                        }
                        result = listener.accept() => {
                            match result {
                                Ok((stream, _)) => {
                                    let io = TokioIo::new(stream);
                                    let requests = requests_clone.clone();

                                    let conn_task = tokio::spawn(async move {
                                        if let Err(err) = hyper::server::conn::http1::Builder::new()
                                            .serve_connection(
                                                io,
                                                service_fn(move |req| {
                                                    Self::handle_request(req, requests.clone())
                                                }),
                                            )
                                            .await
                                        {
                                            eprintln!("Error serving connection: {}", err);
                                        }
                                    });
                                    // Track the connection task so we can abort on crash
                                    connections_clone.lock().await.push(conn_task);
                                }
                                Err(e) => eprintln!("Error accepting connection: {}", e),
                            }
                        }
                    }
                }
            });

            Ok(Self {
                addr,
                requests,
                join_handle: handle,
                shutdown_tx: Some(shutdown_tx),
                connections,
            })
        }

        async fn handle_request(
            req: hyper::Request<hyper::body::Incoming>,
            requests: Arc<tokio::sync::Mutex<Vec<serde_json::Value>>>,
        ) -> Result<hyper::Response<String>, hyper::Error> {
            let body_bytes = match req.into_body().collect().await {
                Ok(buf) => buf.to_bytes(),
                Err(_) => {
                    let error_response = json!({
                        "jsonrpc": "2.0",
                        "error": { "code": -32700, "message": "Failed to read request body" },
                        "id": null
                    });
                    return Ok(hyper::Response::new(error_response.to_string()));
                }
            };

            let request_body: serde_json::Value = match serde_json::from_slice(&body_bytes) {
                Ok(json) => json,
                Err(_) => {
                    let error_response = json!({
                        "jsonrpc": "2.0",
                        "error": { "code": -32700, "message": "Invalid JSON format" },
                        "id": null
                    });
                    return Ok(hyper::Response::new(error_response.to_string()));
                }
            };

            // spawn and await so that the requests will eventually be processed
            // even after the request is cancelled
            let request_body_clone = request_body.clone();
            tokio::spawn(async move {
                requests.lock().await.push(request_body_clone);
            })
            .await
            .unwrap();

            let method = request_body["method"].as_str().unwrap_or_default();

            let response = match method {
                "eth_sendRawTransaction" | "eth_sendRawTransactionConditional" => json!({
                    "jsonrpc": "2.0",
                    "result": format!("{}", B256::from([1; 32])),
                    "id": request_body["id"]
                }),
                "miner_setMaxDASize" | "miner_setGasLimit" | "miner_setGasPrice"
                | "miner_setExtra" => {
                    json!({
                        "jsonrpc": "2.0",
                        "result": true,
                        "id": request_body["id"]
                    })
                }
                "mock_forwardedMethod" => {
                    json!({
                        "jsonrpc": "2.0",
                        "result": "forwarded response",
                        "id": request_body["id"]
                    })
                }
                _ => {
                    let error_response = json!({
                        "jsonrpc": "2.0",
                        "error": { "code": -32601, "message": "Method not found" },
                        "id": request_body["id"]
                    });
                    return Ok(hyper::Response::new(error_response.to_string()));
                }
            };

            Ok(hyper::Response::new(response.to_string()))
        }
    }

    #[cfg(test)]
    #[ctor::ctor]
    fn crypto_ring_init() {
        rustls::crypto::ring::default_provider()
            .install_default()
            .unwrap();
    }

    #[tokio::test]
    async fn test_proxy_service() {
        proxy_success().await;
        proxy_failure().await;
        does_not_proxy_engine_method().await;
        health_check().await;
    }

    async fn proxy_success() {
        let response = send_request("greet_melkor").await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "You are the dark lord");
    }

    async fn proxy_failure() {
        let response = send_request("non_existent_method").await;
        assert!(response.is_err());
        let expected_error = ErrorObject::from(ErrorCode::MethodNotFound).into_owned();
        assert!(matches!(
            response.unwrap_err(),
            ClientError::Call(e) if e == expected_error
        ));
    }

    async fn does_not_proxy_engine_method() {
        let response = send_request("engine_method").await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "engine response");
    }

    async fn health_check() {
        // Spawn a backend for the proxy to point to, and a proxy with dynamic port
        let (backend_server, backend_addr) = spawn_server().await;
        let (proxy_server, proxy_addr) = spawn_proxy_server_with_l2(backend_addr).await;
        // Create a new HTTP client
        let client: Client<HttpConnector, Full<Bytes>> =
            Client::builder(TokioExecutor::new()).build_http();

        // Test the health check endpoint
        let health_check_url = format!("http://{proxy_addr}/healthz");
        let health_response = client.get(health_check_url.parse::<Uri>().unwrap()).await;
        assert!(health_response.is_ok());
        let status = health_response.unwrap().status();
        assert_eq!(status, StatusCode::OK);

        proxy_server.stop().unwrap();
        proxy_server.stopped().await;
        backend_server.stop().unwrap();
        backend_server.stopped().await;
    }

    async fn send_request(method: &str) -> Result<String, ClientError> {
        let (backend_server, backend_addr) = spawn_server().await;
        let (proxy_server, proxy_addr) = spawn_proxy_server_with_l2(backend_addr).await;
        let proxy_client = HttpClient::builder()
            .build(format!("http://{}", proxy_addr))
            .unwrap();

        let response = proxy_client
            .request::<String, _>(method, rpc_params![])
            .await;

        backend_server.stop().unwrap();
        backend_server.stopped().await;
        proxy_server.stop().unwrap();
        proxy_server.stopped().await;

        response
    }

    async fn spawn_server() -> (ServerHandle, SocketAddr) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        drop(listener);
        let server = ServerBuilder::default().build(addr).await.unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("greet_melkor", |_, _, _| "You are the dark lord")
            .unwrap();

        (server.start(module), addr)
    }

    /// Spawn a new RPC server with a proxy layer pointing to a provided L2 address.
    async fn spawn_proxy_server_with_l2(l2_addr: SocketAddr) -> (ServerHandle, SocketAddr) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let proxy_addr = listener.local_addr().unwrap();
        drop(listener);

        let jwt = JwtSecret::random();
        let l2_auth_uri = format!("http://{}", l2_addr).parse::<Uri>().unwrap();

        let (probe_layer, _) = ProbeLayer::new();
        let proxy_layer = ProxyLayer::new(
            ClientArgs {
                url: l2_auth_uri.clone(),
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 1,
            }
            .new_http_client(PayloadSource::L2)
            .unwrap(),
            ClientArgs {
                url: l2_auth_uri.clone(),
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 1,
            }
            .new_http_client(PayloadSource::Builder)
            .unwrap(),
        );

        // Create a layered server
        let server = ServerBuilder::default()
            .set_http_middleware(
                tower::ServiceBuilder::new()
                    .layer(probe_layer)
                    .layer(proxy_layer),
            )
            .build(proxy_addr)
            .await
            .unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("engine_method", |_, _, _| "engine response")
            .unwrap();
        module
            .register_method(
                "eth_sendRawTransaction",
                |_, _, _| "raw transaction response",
            )
            .unwrap();
        module
            .register_method("non_existent_method", |_, _, _| "no proxy response")
            .unwrap();

        (server.start(module), proxy_addr)
    }

    #[tokio::test]
    async fn test_forward_set_max_da_size() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let test_harness = TestHarness::new().await?;

        let max_tx_size = U64::MAX;
        let max_block_size = U64::MAX;

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>("miner_setMaxDASize", (max_tx_size, max_block_size))
            .await?;

        tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;

        let expected_method = "miner_setMaxDASize";
        let expected_tx_size = json!(max_tx_size);
        let expected_block_size = json!(max_block_size);

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_tx_size);
        assert_eq!(builder_req["params"][1], expected_block_size);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_tx_size);
        assert_eq!(builder_req["params"][1], expected_block_size);

        Ok(())
    }

    #[tokio::test]
    async fn test_forward_eth_send_raw_transaction() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let test_harness = TestHarness::new().await?;

        let expected_tx: Bytes = hex!("1234").into();
        let expected_method = "eth_sendRawTransaction";

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(expected_method, (expected_tx.clone(),))
            .await?;

        let expected_tx = json!(expected_tx);

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_tx);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_tx);

        Ok(())
    }

    #[tokio::test]
    async fn test_forward_eth_send_raw_transaction_conditional() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let test_harness = TestHarness::new().await?;

        let expected_tx: Bytes = hex!("1234").into();
        let expected_method = "eth_sendRawTransactionConditional";
        let transact_conditionals = TransactionConditional::default();
        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(
                expected_method,
                (expected_tx.clone(), transact_conditionals.clone()),
            )
            .await?;

        let expected_tx = json!(expected_tx);
        let expected_conditionals = json!(transact_conditionals);
        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_tx);
        assert_eq!(builder_req["params"][1], expected_conditionals);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_tx);
        assert_eq!(l2_req["params"][1], expected_conditionals);

        Ok(())
    }

    #[tokio::test]
    async fn test_forward_miner_set_extra() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let test_harness = TestHarness::new().await?;

        let extra = Bytes::default();
        let expected_method = "miner_setExtra";

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(expected_method, (extra.clone(),))
            .await?;

        let expected_extra = json!(extra);

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_extra);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_extra);

        Ok(())
    }

    #[tokio::test]
    async fn test_forward_miner_set_gas_price() -> eyre::Result<()> {
        let test_harness = TestHarness::new().await?;

        let gas_price = U128::ZERO;
        let expected_method = "miner_setGasPrice";

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(expected_method, (gas_price,))
            .await?;

        let expected_price = json!(gas_price);
        tokio::time::sleep(tokio::time::Duration::from_secs(3)).await;

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_price);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_price);

        Ok(())
    }

    #[tokio::test]
    async fn test_forward_miner_set_gas_limit() -> eyre::Result<()> {
        let test_harness = TestHarness::new().await?;

        let gas_limit = U128::ZERO;
        let expected_method = "miner_setGasLimit";

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(expected_method, (gas_limit,))
            .await?;

        let expected_price = json!(gas_limit);

        tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_price);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_price);

        Ok(())
    }

    #[tokio::test]
    async fn test_direct_forward_mock_request() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let test_harness = TestHarness::new().await?;

        let mock_data = U128::ZERO;
        let expected_method = "mock_forwardedMethod";

        test_harness
            .proxy_client
            .request::<serde_json::Value, _>(expected_method, (mock_data,))
            .await?;

        let expected_price = json!(mock_data);

        // Assert the builder has not received the payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().await;
        assert_eq!(builder_requests.len(), 0);

        // Assert the l2 auth received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().await;
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_price);

        Ok(())
    }

    #[tokio::test]
    async fn test_l2_server_recovery() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;

        // Step 1: Reserve a port for L2 by binding and then releasing it
        let temp_listener = TcpListener::bind("127.0.0.1:0").await?;
        let l2_addr = temp_listener.local_addr()?;
        drop(temp_listener);

        // Wait for port to be fully released
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

        // Step 2: Create builder and proxy WITHOUT an L2 server running yet
        let builder = MockHttpServer::serve().await?;
        let builder_addr = builder.addr;
        let jwt = JwtSecret::random();

        // Create proxy layer with L2 client pointing to a non-existent server
        // Use a short timeout to fail quickly
        let proxy_layer = ProxyLayer::new(
            ClientArgs {
                url: format!("http://{}:{}", l2_addr.ip(), l2_addr.port()).parse::<Uri>()?,
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 200, // Short timeout for faster failure
            }
            .new_http_client(PayloadSource::L2)?,
            ClientArgs {
                url: format!("http://{}:{}", builder_addr.ip(), builder_addr.port())
                    .parse::<Uri>()?,
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 200,
            }
            .new_http_client(PayloadSource::Builder)?,
        );

        // Start proxy server
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let proxy_addr = listener.local_addr()?;
        drop(listener);

        let server = Server::builder()
            .set_http_middleware(tower::ServiceBuilder::new().layer(proxy_layer))
            .build(proxy_addr)
            .await?;

        let proxy_addr = server.local_addr()?;
        let proxy_client: HttpClient = HttpClient::builder().build(format!(
            "http://{}:{}",
            proxy_addr.ip(),
            proxy_addr.port()
        ))?;

        let server_handle = server.start(RpcModule::new(()));

        // Step 3: Request should fail (connection refused) because L2 server doesn't exist
        let mock_data = U128::from(42);
        let result = proxy_client
            .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
            .await;
        assert!(
            result.is_err(),
            "Request should fail when L2 server is not running"
        );
        println!("Request failed as expected (no server): {:?}", result);

        // Step 4: Start the L2 server
        let l2 = MockHttpServer::serve_on_addr(l2_addr).await?;
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

        // Step 5: Request should now succeed (demonstrating auto-recovery)
        let result = proxy_client
            .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
            .await;
        assert!(
            result.is_ok(),
            "Request should succeed after L2 server starts (auto-recovery): {:?}",
            result
        );
        println!("Request succeeded after server started: {:?}", result);

        // Step 6: Verify multiple subsequent requests work consistently
        for i in 0..3 {
            let result = proxy_client
                .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
                .await;
            assert!(
                result.is_ok(),
                "Request {} should continue to succeed: {:?}",
                i,
                result
            );
        }

        // Verify the server received requests
        {
            let l2_requests = l2.requests.lock().await;
            assert!(
                l2_requests.len() >= 1,
                "L2 server should have received requests"
            );
            assert_eq!(l2_requests[0]["method"], "mock_forwardedMethod");
        }

        // Cleanup
        server_handle.stop()?;
        drop(builder);
        drop(l2);

        Ok(())
    }

    #[tokio::test]
    async fn test_success_then_failure_then_success() -> eyre::Result<()> {
        // Dynamically bind L2 and Proxy servers
        let l2 = MockHttpServer::serve().await?;
        let l2_addr = l2.addr;

        // Build proxy with short timeouts pointing to current L2
        let jwt = JwtSecret::random();
        let proxy_layer = ProxyLayer::new(
            ClientArgs {
                url: format!("http://{}:{}", l2_addr.ip(), l2_addr.port()).parse::<Uri>()?,
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 200,
            }
            .new_http_client(PayloadSource::L2)?,
            ClientArgs {
                url: format!("http://{}:{}", l2_addr.ip(), l2_addr.port()).parse::<Uri>()?,
                jwt_token: Some(jwt),
                jwt_path: None,
                timeout: 200,
            }
            .new_http_client(PayloadSource::Builder)?,
        );

        // Start proxy on dynamic port
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let proxy_addr = listener.local_addr()?;
        drop(listener);

        let server = Server::builder()
            .set_http_middleware(tower::ServiceBuilder::new().layer(proxy_layer))
            .build(proxy_addr)
            .await?;
        let proxy_addr = server.local_addr()?;
        let proxy_client: HttpClient = HttpClient::builder().build(format!(
            "http://{}:{}",
            proxy_addr.ip(),
            proxy_addr.port()
        ))?;
        let server_handle = server.start(RpcModule::new(()));

        let mock_data = U128::from(7);

        // 1) Initial success
        let res = proxy_client
            .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
            .await;
        assert!(res.is_ok(), "initial request should succeed: {:?}", res);

        // 2) Stop L2 -> subsequent failure
        drop(l2);
        tokio::time::sleep(tokio::time::Duration::from_millis(200)).await;
        // Expect a JSON-RPC error object with code -32000 (retriable)
        let res = proxy_client
            .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
            .await;
        match res {
            Ok(v) => unreachable!("expected error when L2 down, got: {v:?}"),
            Err(ClientError::Call(e)) => {
                let code = e.code();
                assert!(
                    is_retriable_code(code),
                    "expected retriable code (not parse/invalid), got {}",
                    code
                );
            }
            Err(_other) => {
                // Transport or other non-Call errors are considered retriable
                assert!(
                    matches!(_other, ClientError::Transport(_)),
                    "expected transport error"
                );
            }
        }

        // 3) Restart L2 -> subsequent success
        let l2_restarted = MockHttpServer::serve_on_addr(l2_addr).await?;
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
        let res = proxy_client
            .request::<serde_json::Value, _>("mock_forwardedMethod", (mock_data,))
            .await;
        assert!(
            res.is_ok(),
            "request should succeed after L2 restart: {:?}",
            res
        );

        // Cleanup
        server_handle.stop()?;
        drop(l2_restarted);

        Ok(())
    }
}
