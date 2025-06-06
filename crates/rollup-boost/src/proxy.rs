use crate::client::http::HttpClient;
use crate::payload::PayloadSource;
use crate::{Request, Response, from_buffered_request, into_buffered_request};
use alloy_rpc_types_engine::JwtSecret;
use http::Uri;
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
    l2_auth_rpc: Uri,
    l2_auth_secret: JwtSecret,
    l2_timeout: u64,
    builder_auth_rpc: Uri,
    builder_auth_secret: JwtSecret,
    builder_timeout: u64,
}

impl ProxyLayer {
    pub fn new(
        l2_auth_rpc: Uri,
        l2_auth_secret: JwtSecret,
        l2_timeout: u64,
        builder_auth_rpc: Uri,
        builder_auth_secret: JwtSecret,
        builder_timeout: u64,
    ) -> Self {
        ProxyLayer {
            l2_auth_rpc,
            l2_auth_secret,
            l2_timeout,
            builder_auth_rpc,
            builder_auth_secret,
            builder_timeout,
        }
    }
}

impl<S> Layer<S> for ProxyLayer {
    type Service = ProxyService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        let l2_client = HttpClient::new(
            self.l2_auth_rpc.clone(),
            self.l2_auth_secret,
            PayloadSource::L2,
            self.l2_timeout,
        );

        let builder_client = HttpClient::new(
            self.builder_auth_rpc.clone(),
            self.builder_auth_secret,
            PayloadSource::Builder,
            self.builder_timeout,
        );

        ProxyService {
            inner,
            l2_client,
            builder_client,
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
    use crate::probe::ProbeLayer;

    use super::*;
    use alloy_primitives::{B256, Bytes, U64, U128, hex};
    use alloy_rpc_types_engine::JwtSecret;
    use alloy_rpc_types_eth::erc4337::TransactionConditional;
    use http::StatusCode;
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
    use std::{
        net::{IpAddr, SocketAddr},
        str::FromStr,
        sync::Arc,
    };
    use tokio::net::TcpListener;
    use tokio::task::JoinHandle;

    const PORT: u32 = 8552;
    const ADDR: &str = "127.0.0.1";
    const PROXY_PORT: u32 = 8553;

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
                format!("http://{}:{}", l2.addr.ip(), l2.addr.port()).parse::<Uri>()?,
                JwtSecret::random(),
                1,
                format!("http://{}:{}", builder.addr.ip(), builder.addr.port()).parse::<Uri>()?,
                JwtSecret::random(),
                1,
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
    }

    impl Drop for MockHttpServer {
        fn drop(&mut self) {
            self.join_handle.abort();
        }
    }

    impl MockHttpServer {
        async fn serve() -> eyre::Result<Self> {
            let listener = TcpListener::bind("127.0.0.1:0").await?;
            let addr = listener.local_addr()?;
            let requests = Arc::new(tokio::sync::Mutex::new(vec![]));

            let requests_clone = requests.clone();
            let handle = tokio::spawn(async move {
                loop {
                    match listener.accept().await {
                        Ok((stream, _)) => {
                            let io = TokioIo::new(stream);
                            let requests = requests_clone.clone();

                            tokio::spawn(async move {
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
                        }
                        Err(e) => eprintln!("Error accepting connection: {}", e),
                    }
                }
            });

            Ok(Self {
                addr,
                requests,
                join_handle: handle,
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
        let proxy_server = spawn_proxy_server().await;
        // Create a new HTTP client
        let client: Client<HttpConnector, Full<Bytes>> =
            Client::builder(TokioExecutor::new()).build_http();

        // Test the health check endpoint
        let health_check_url = format!("http://{ADDR}:{PORT}/healthz");
        let health_response = client.get(health_check_url.parse::<Uri>().unwrap()).await;
        assert!(health_response.is_ok());
        let status = health_response.unwrap().status();
        assert_eq!(status, StatusCode::OK);

        proxy_server.stop().unwrap();
        proxy_server.stopped().await;
    }

    async fn send_request(method: &str) -> Result<String, ClientError> {
        let server = spawn_server().await;
        let proxy_server = spawn_proxy_server().await;
        let proxy_client = HttpClient::builder()
            .build(format!("http://{ADDR}:{PORT}"))
            .unwrap();

        let response = proxy_client
            .request::<String, _>(method, rpc_params![])
            .await;

        server.stop().unwrap();
        server.stopped().await;
        proxy_server.stop().unwrap();
        proxy_server.stopped().await;

        response
    }

    async fn spawn_server() -> ServerHandle {
        let server = ServerBuilder::default()
            .build(
                format!("{ADDR}:{PROXY_PORT}")
                    .parse::<SocketAddr>()
                    .unwrap(),
            )
            .await
            .unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("greet_melkor", |_, _, _| "You are the dark lord")
            .unwrap();

        server.start(module)
    }

    /// Spawn a new RPC server with a proxy layer.
    async fn spawn_proxy_server() -> ServerHandle {
        let addr = format!("{ADDR}:{PORT}");

        let jwt = JwtSecret::random();
        let l2_auth_uri = format!(
            "http://{}",
            SocketAddr::new(IpAddr::from_str(ADDR).unwrap(), PROXY_PORT as u16)
        )
        .parse::<Uri>()
        .unwrap();

        let (probe_layer, _) = ProbeLayer::new();
        let proxy_layer = ProxyLayer::new(l2_auth_uri.clone(), jwt, 1, l2_auth_uri, jwt, 1);

        // Create a layered server
        let server = ServerBuilder::default()
            .set_http_middleware(
                tower::ServiceBuilder::new()
                    .layer(probe_layer)
                    .layer(proxy_layer),
            )
            .build(addr.parse::<SocketAddr>().unwrap())
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

        server.start(module)
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
}
