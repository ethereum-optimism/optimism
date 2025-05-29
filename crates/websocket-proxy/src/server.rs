use crate::auth::Authentication;
use crate::client::ClientConnection;
use crate::metrics::Metrics;
use crate::rate_limit::{RateLimit, RateLimitError};
use crate::registry::Registry;
use axum::body::Body;
use axum::extract::{ConnectInfo, Path, State, WebSocketUpgrade};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::routing::{any, get};
use axum::{Error, Router};
use http::{HeaderMap, HeaderValue};
use serde_json::json;
use std::net::{IpAddr, SocketAddr};
use std::sync::Arc;
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};

#[derive(Clone)]
struct ServerState {
    registry: Registry,
    rate_limiter: Arc<dyn RateLimit>,
    metrics: Arc<Metrics>,
    auth: Authentication,
    ip_addr_http_header: String,
}

#[derive(Clone)]
pub struct Server {
    listen_addr: SocketAddr,
    registry: Registry,
    rate_limiter: Arc<dyn RateLimit>,
    metrics: Arc<Metrics>,
    ip_addr_http_header: String,
    authentication: Option<Authentication>,
}

impl Server {
    pub fn new(
        listen_addr: SocketAddr,
        registry: Registry,
        metrics: Arc<Metrics>,
        rate_limiter: Arc<dyn RateLimit>,
        authentication: Option<Authentication>,
        ip_addr_http_header: String,
    ) -> Self {
        Self {
            listen_addr,
            registry,
            rate_limiter,
            metrics,
            authentication,
            ip_addr_http_header,
        }
    }

    pub async fn listen(&self, cancellation_token: CancellationToken) {
        let mut router: Router<ServerState> = Router::new().route("/healthz", get(healthz_handler));

        if self.authentication.is_some() {
            info!("Authentication is enabled");
            router = router.route("/ws/{api_key}", any(authenticated_websocket_handler));
        } else {
            info!("Public endpoint is enabled");
            router = router.route("/ws", any(unauthenticated_websocket_handler));
        }

        let router = router.with_state(ServerState {
            registry: self.registry.clone(),
            rate_limiter: self.rate_limiter.clone(),
            metrics: self.metrics.clone(),
            auth: self
                .authentication
                .clone()
                .unwrap_or_else(Authentication::none),
            ip_addr_http_header: self.ip_addr_http_header.clone(),
        });

        let listener = tokio::net::TcpListener::bind(self.listen_addr)
            .await
            .unwrap();

        info!(
            message = "starting server",
            address = listener.local_addr().unwrap().to_string()
        );

        axum::serve(
            listener,
            router.into_make_service_with_connect_info::<SocketAddr>(),
        )
        .with_graceful_shutdown(cancellation_token.cancelled_owned())
        .await
        .unwrap()
    }
}

async fn healthz_handler() -> impl IntoResponse {
    StatusCode::OK
}

async fn authenticated_websocket_handler(
    State(state): State<ServerState>,
    ws: WebSocketUpgrade,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    headers: HeaderMap,
    Path(api_key): Path<String>,
) -> impl IntoResponse {
    let application = state.auth.get_application_for_key(&api_key);

    match application {
        None => {
            state.metrics.unauthorized_requests.increment(1);

            Response::builder()
                .status(StatusCode::UNAUTHORIZED)
                .body(Body::from(
                    json!({"message": "Invalid API key"}).to_string(),
                ))
                .unwrap()
        }
        Some(app) => {
            state.metrics.proxy_connections_by_app(app);
            websocket_handler(state, ws, addr, headers)
        }
    }
}

async fn unauthenticated_websocket_handler(
    State(state): State<ServerState>,
    ws: WebSocketUpgrade,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    headers: HeaderMap,
) -> impl IntoResponse {
    websocket_handler(state, ws, addr, headers)
}

fn websocket_handler(
    state: ServerState,
    ws: WebSocketUpgrade,
    addr: SocketAddr,
    headers: HeaderMap,
) -> Response {
    let connect_addr = addr.ip();

    let client_addr = match headers.get(state.ip_addr_http_header) {
        None => connect_addr,
        Some(value) => extract_addr(value, connect_addr),
    };

    let ticket = match state.rate_limiter.try_acquire(client_addr) {
        Ok(ticket) => ticket,
        Err(RateLimitError::Limit { reason }) => {
            state.metrics.rate_limited_requests.increment(1);

            return Response::builder()
                .status(StatusCode::TOO_MANY_REQUESTS)
                .body(Body::from(json!({"message": reason}).to_string()))
                .unwrap();
        }
    };

    ws.on_failed_upgrade(move |e: Error| {
        info!(
            message = "failed to upgrade connection",
            error = e.to_string(),
            client = addr.to_string()
        )
    })
    .on_upgrade(async move |socket| {
        let client = ClientConnection::new(client_addr, ticket, socket);
        state.registry.subscribe(client).await;
    })
}

fn extract_addr(header: &HeaderValue, fallback: IpAddr) -> IpAddr {
    if header.is_empty() {
        return fallback;
    }

    match header.to_str() {
        Ok(header_value) => {
            let raw_value = header_value
                .split(',')
                .map(|ip| ip.trim().to_string())
                .next_back();

            if let Some(raw_value) = raw_value {
                return raw_value.parse::<IpAddr>().unwrap_or(fallback);
            }

            fallback
        }
        Err(e) => {
            warn!(
                message = "could not get header value",
                error = e.to_string()
            );
            fallback
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::Ipv4Addr;

    #[tokio::test]
    async fn test_header_addr() {
        let fb = Ipv4Addr::new(127, 0, 0, 1);

        let test = |header: &str, expected: Ipv4Addr| {
            let hv = HeaderValue::from_str(header).unwrap();
            let result = extract_addr(&hv, IpAddr::V4(fb));
            assert_eq!(result, expected);
        };

        test("129.1.1.1", Ipv4Addr::new(129, 1, 1, 1));
        test("129.1.1.1,130.1.1.1", Ipv4Addr::new(130, 1, 1, 1));
        test("129.1.1.1  ,  130.1.1.1   ", Ipv4Addr::new(130, 1, 1, 1));
        test("nonsense", fb);
        test("400.0.0.1", fb);
        test("120.0.0.1.0", fb);
    }
}
