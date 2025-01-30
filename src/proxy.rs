use http::Uri;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::client::ClientT;
use jsonrpsee::core::traits::ToRpcParams;
use jsonrpsee::core::{http_helpers, BoxError};
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpBody, HttpClient, HttpRequest, HttpResponse};
use reth_rpc_layer::AuthClientService;
use std::sync::Arc;
use std::task::{Context, Poll};
use std::{future::Future, pin::Pin};
use tower::{Layer, Service};
use tracing::{debug, info};

const MULTIPLEX_METHODS: [&str; 3] = ["engine_", "eth_sendRawTransaction", "miner_"];

#[derive(Debug, Clone)]
pub struct ProxyLayer {
    l2_auth_client: Arc<HttpClient<AuthClientService<HttpBackend>>>,
}

impl ProxyLayer {
    pub fn new(l2_auth_client: Arc<HttpClient<AuthClientService<HttpBackend>>>) -> Self {
        ProxyLayer { l2_auth_client }
    }
}

impl<S> Layer<S> for ProxyLayer {
    type Service = ProxyService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        ProxyService {
            inner,
            l2_auth_client: self.l2_auth_client.clone(),
        }
    }
}

#[derive(Clone)]
pub struct ProxyService<S> {
    inner: S,
    l2_auth_client: Arc<HttpClient<AuthClientService<HttpBackend>>>,
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

        let l2_auth_client = self.l2_auth_client.clone();
        let mut inner = self.inner.clone();

        #[derive(serde::Deserialize, Debug)]
        struct RpcRequest<'a> {
            #[serde(borrow)]
            method: &'a str,
        }

        let fut = async move {
            let (parts, body) = req.into_parts();

            let (body, _is_single) =
                http_helpers::read_body(&parts.headers, body, u32::MAX).await?;
            // Deserialize the bytes to find the method
            let rpc_request: RpcRequest = serde_json::from_slice(&body)?;

            // Create a new body from the bytes
            let new_body = HttpBody::from(body.clone());

            // Reconstruct the request
            let mut req = HttpRequest::from_parts(parts, new_body);

            debug!(
                message = "received json rpc request for",
                method = rpc_request.method
            );
            if MULTIPLEX_METHODS
                .iter()
                .any(|&m| rpc_request.method.starts_with(m))
            {
                info!(target: "proxy::call", message = "proxying request to rollup-boost server", "method" = ?rpc_request.method);

                // let rpc server handle engine rpc requests
                let res = inner.call(req).await.map_err(|e| e.into())?;
                Ok(res)
            } else {
                info!(target: "proxy::call", message = "forwarding request to l2 client", "method" = ?rpc_request.method);

                // TODO: forward to l2_auth_client

                todo!();
            }
        };
        Box::pin(fut)
    }
}

#[cfg(test)]
mod tests {
    use std::{
        net::{IpAddr, SocketAddr},
        str::FromStr,
    };

    use http_body_util::BodyExt;
    use jsonrpsee::{
        core::{client::ClientT, ClientError},
        http_client::{HttpClient, HttpClientBuilder},
        rpc_params,
        server::{ServerBuilder, ServerHandle},
        types::{ErrorCode, ErrorObject},
        RpcModule,
    };
    use reth_rpc_layer::{AuthClientLayer, JwtSecret};

    use super::*;

    const PORT: u32 = 8552;
    const ADDR: &str = "0.0.0.0";
    const PROXY_PORT: u32 = 8553;

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
        let auth_layer = AuthClientLayer::new(jwt);
        let l2_auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .build(format!(
                "http://{}",
                SocketAddr::new(IpAddr::from_str(ADDR).unwrap(), PORT as u16)
            ))
            .expect("Could not build auth client");

        let proxy_layer = ProxyLayer::new(Arc::new(l2_auth_client));

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
}
