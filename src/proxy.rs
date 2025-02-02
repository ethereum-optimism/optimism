use http::header::AUTHORIZATION;
use http::Uri;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::{http_helpers, BoxError};
use jsonrpsee::http_client::{HttpBody, HttpRequest, HttpResponse};
use reth_rpc_layer::{secret_to_bearer_header, JwtSecret};
use std::task::{Context, Poll};
use std::{future::Future, pin::Pin};
use tower::{Layer, Service};
use tracing::{debug, error, info};

const MULTIPLEX_METHODS: [&str; 3] = ["engine_", "eth_sendRawTransaction", "miner_"];
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
    l2_uri: Uri,
    l2_auth: JwtSecret,
    builder_uri: Uri,
    builder_auth: Option<JwtSecret>,
}

impl ProxyLayer {
    pub fn new(
        l2_uri: Uri,
        l2_auth: JwtSecret,
        builder_uri: Uri,
        builder_auth: Option<JwtSecret>,
    ) -> Self {
        ProxyLayer {
            l2_uri,
            l2_auth,
            builder_uri,
            builder_auth,
        }
    }
}

impl<S> Layer<S> for ProxyLayer {
    type Service = ProxyService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        ProxyService {
            inner,
            client: Client::builder(TokioExecutor::new()).build_http(),
            l2_uri: self.l2_uri.clone(),
            l2_auth: self.l2_auth,
            builder_uri: self.builder_uri.clone(),
            builder_auth: self.builder_auth,
        }
    }
}

#[derive(Clone)]
pub struct ProxyService<S> {
    inner: S,
    client: Client<HttpConnector, HttpBody>,
    l2_uri: Uri,
    l2_auth: JwtSecret,
    builder_uri: Uri,
    builder_auth: Option<JwtSecret>,
}

impl<S> Service<HttpRequest<HttpBody>> for ProxyService<S>
where
    S: Service<HttpRequest<HttpBody>, Response = HttpResponse> + Send + Clone + 'static,
    S::Response: 'static,
    S::Error: Into<BoxError> + 'static,
    S::Future: Send + 'static,
{
    type Response = S::Response;
    type Error = BoxError;
    type Future =
        Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send + 'static>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, req: HttpRequest<HttpBody>) -> Self::Future {
        if req.uri().path() == "/healthz" {
            return Box::pin(async { Ok(Self::Response::new(HttpBody::from("OK"))) });
        }

        let client = self.client.clone();
        let mut inner = self.inner.clone();
        let builder_uri = self.builder_uri.clone();
        let builder_auth = self.builder_auth;
        let l2_uri = self.l2_uri.clone();
        let l2_auth = self.l2_auth;

        #[derive(serde::Deserialize, Debug)]
        struct RpcRequest<'a> {
            #[serde(borrow)]
            method: &'a str,
        }

        let fut = async move {
            let (parts, body) = req.into_parts();
            let (body_bytes, _) = http_helpers::read_body(&parts.headers, body, u32::MAX).await?;

            // Deserialize the bytes to find the method
            let method = serde_json::from_slice::<RpcRequest>(&body_bytes)?
                .method
                .to_string();

            debug!(message = "received json rpc request for", ?method);

            if MULTIPLEX_METHODS.iter().any(|&m| method.starts_with(m)) {
                if FORWARD_REQUESTS.contains(&method.as_str()) {
                    let builder_client = client.clone();
                    let builder_req =
                        HttpRequest::from_parts(parts.clone(), HttpBody::from(body_bytes.clone()));
                    let builder_method = method.clone();

                    tokio::spawn(async move {
                        let _ = forward_request(
                            builder_client,
                            builder_req,
                            &builder_method,
                            builder_uri,
                            builder_auth,
                        )
                        .await;
                    });

                    let l2_req = HttpRequest::from_parts(parts, HttpBody::from(body_bytes));
                    info!(target: "proxy::call", message = "proxying request to rollup-boost server", ?method);
                    forward_request(client, l2_req, &method, l2_uri, None).await
                } else {
                    let req = HttpRequest::from_parts(parts, HttpBody::from(body_bytes));
                    info!(target: "proxy::call", message = "proxying request to rollup-boost server", ?method);
                    inner.call(req).await.map_err(|e| e.into())
                }
            } else {
                let req = HttpRequest::from_parts(parts, HttpBody::from(body_bytes));
                forward_request(client, req, &method, l2_uri, Some(l2_auth)).await
            }
        };
        Box::pin(fut)
    }
}

async fn forward_request(
    client: Client<HttpConnector, HttpBody>,
    mut req: http::Request<HttpBody>,
    method: &str,
    uri: Uri,
    auth: Option<JwtSecret>,
) -> Result<http::Response<HttpBody>, BoxError> {
    *req.uri_mut() = uri.clone();
    if let Some(auth) = auth {
        req.headers_mut()
            .insert(AUTHORIZATION, secret_to_bearer_header(&auth));
    }

    debug!(
        target: "proxy::forward_request",
        url = ?uri,
        ?method,
        ?req,
    );

    match client.request(req).await {
        Ok(resp) => Ok(resp.map(HttpBody::new)),
        Err(e) => {
            error!(
                target: "proxy::call",
                message = "error forwarding request",
                url = ?uri,
                method = %method,
                error = %e,
            );
            Err(e.into())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use super::*;
    use alloy_primitives::{hex, B256, U64};
    use alloy_primitives::{FixedBytes, U256};
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV2, PayloadStatusEnum,
    };
    use http_body_util::BodyExt;
    use hyper::service::service_fn;
    use hyper_util::rt::TokioIo;
    use jsonrpsee::server::Server;
    use jsonrpsee::{
        core::{client::ClientT, ClientError},
        http_client::HttpClient,
        rpc_params,
        server::{ServerBuilder, ServerHandle},
        types::{ErrorCode, ErrorObject},
        RpcModule,
    };
    use reth_rpc_layer::JwtSecret;
    use serde_json::json;
    use std::thread::sleep;
    use std::time::Duration;
    use std::{
        net::{IpAddr, SocketAddr},
        str::FromStr,
        sync::{Arc, Mutex},
    };
    use tokio::net::{TcpListener, TcpSocket};
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
                format!("http://{}:{}", builder.addr.ip(), builder.addr.port()).parse::<Uri>()?,
                None,
            ));

            let temp_listener = TcpListener::bind("0.0.0.0:0").await?;
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
        requests: Arc<Mutex<Vec<serde_json::Value>>>,
        join_handle: JoinHandle<()>,
    }

    impl Drop for MockHttpServer {
        fn drop(&mut self) {
            self.join_handle.abort();
        }
    }

    impl MockHttpServer {
        async fn serve() -> eyre::Result<Self> {
            let listener = TcpListener::bind("0.0.0.0:0").await?;
            let addr = listener.local_addr()?;
            let requests = Arc::new(Mutex::new(vec![]));

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
            requests: Arc<Mutex<Vec<serde_json::Value>>>,
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

            requests.lock().unwrap().push(request_body.clone());

            let method = request_body["method"].as_str().unwrap_or_default();

            let response = match method {
                "eth_sendRawTransaction" => json!({
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
                _ => {
                    let error_response = json!({
                        "jsonrpc": "2.0",
                        "error": { "code": -32601, "message": "Method not found" },
                        "id": request_body["id"]
                    });
                    return Ok(hyper::Response::new(error_response.to_string()));
                }
            };

            return Ok(hyper::Response::new(response.to_string()));
        }
    }

    #[tokio::test]
    async fn test_proxy_service() {
        proxy_success().await;
        proxy_failure().await;
        does_not_proxy_engine_method().await;
        does_not_proxy_eth_send_raw_transaction_method().await;
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

    async fn does_not_proxy_eth_send_raw_transaction_method() {
        let response = send_request("eth_sendRawTransaction").await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "raw transaction response");
    }

    async fn health_check() {
        let proxy_server = spawn_proxy_server().await;
        // Create a new HTTP client
        let client: Client<HttpConnector, HttpBody> =
            Client::builder(TokioExecutor::new()).build_http();

        // Test the health check endpoint
        let health_check_url = format!("http://{ADDR}:{PORT}/healthz");
        let health_response = client.get(health_check_url.parse::<Uri>().unwrap()).await;
        assert!(health_response.is_ok());
        let b = health_response
            .unwrap()
            .into_body()
            .collect()
            .await
            .unwrap()
            .to_bytes();
        // Convert the collected bytes to a string
        let body_string = String::from_utf8(b.to_vec()).unwrap();
        assert_eq!(body_string, "OK");

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

        // TODO: update uri
        let proxy_layer = ProxyLayer::new(l2_auth_uri, jwt, Uri::default(), None);

        // Create a layered server
        let server = ServerBuilder::default()
            .set_http_middleware(tower::ServiceBuilder::new().layer(proxy_layer))
            .build(addr.parse::<SocketAddr>().unwrap())
            .await
            .unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("engine_method", |_, _, _| "engine response")
            .unwrap();
        module
            .register_method("eth_sendRawTransaction", |_, _, _| {
                "raw transaction response"
            })
            .unwrap();
        module
            .register_method("non_existent_method", |_, _, _| "no proxy response")
            .unwrap();

        server.start(module)
    }

    #[tokio::test]
    async fn test_set_max_da_size() -> eyre::Result<()> {
        let test_harness = TestHarness::new().await?;

        let max_tx_size = U64::MAX;
        let max_block_size = U64::MAX;

        let _response: serde_json::Value = test_harness
            .proxy_client
            .request("miner_setMaxDASize", (max_tx_size, max_block_size))
            .await?;

        let expected_method = "miner_setMaxDASize";
        let expected_tx_size = json!(max_tx_size);
        let expected_block_size = json!(max_block_size);

        // Assert the builder received the correct payload
        let builder = &test_harness.builder;
        let builder_requests = builder.requests.lock().unwrap();
        let builder_req = builder_requests.first().unwrap();
        assert_eq!(builder_requests.len(), 1);
        assert_eq!(builder_req["method"], expected_method);
        assert_eq!(builder_req["params"][0], expected_tx_size);
        assert_eq!(builder_req["params"][1], expected_block_size);

        // Assert the l2 received the correct payload
        let l2 = &test_harness.l2;
        let l2_requests = l2.requests.lock().unwrap();
        let l2_req = l2_requests.first().unwrap();
        assert_eq!(l2_requests.len(), 1);
        assert_eq!(l2_req["method"], expected_method);
        assert_eq!(l2_req["params"][0], expected_tx_size);
        assert_eq!(builder_req["params"][1], expected_block_size);

        Ok(())
    }
}
