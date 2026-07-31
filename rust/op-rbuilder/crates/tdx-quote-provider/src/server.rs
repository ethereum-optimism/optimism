use std::{net::SocketAddr, sync::Arc, time::Instant};

use axum::{
    Router,
    body::Body,
    extract::{Path, State},
    http::{Response, StatusCode},
    response::IntoResponse,
    routing::get,
};
use serde_json::json;
use tokio::{net::TcpListener, signal};
use tracing::info;

use crate::{
    metrics::Metrics,
    provider::{AttestationConfig, AttestationProvider, get_attestation_provider},
};

/// Server configuration
#[derive(Debug, Clone)]
pub struct ServerConfig {
    /// Address to listen on
    pub listen_addr: SocketAddr,
    /// Whether to use mock attestation
    pub use_mock: bool,
    /// Path to the mock attestation file
    pub mock_attestation_path: String,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            listen_addr: "127.0.0.1:8181".parse().unwrap(),
            use_mock: false,
            mock_attestation_path: "".to_string(),
        }
    }
}

#[derive(Clone)]
struct ServerState {
    quote_provider: Arc<dyn AttestationProvider + Send + Sync>,
    metrics: Metrics,
}

#[derive(Clone)]
pub struct Server {
    /// Quote provider
    quote_provider: Arc<dyn AttestationProvider + Send + Sync>,
    /// Metrics for the server
    metrics: Metrics,
    /// Server configuration
    config: ServerConfig,
}

impl Server {
    pub fn new(config: ServerConfig) -> Self {
        let attestation_config = AttestationConfig {
            mock: config.use_mock,
            mock_attestation_path: config.mock_attestation_path.clone(),
        };
        let quote_provider = get_attestation_provider(attestation_config);
        Self {
            quote_provider,
            metrics: Metrics::default(),
            config,
        }
    }

    pub async fn listen(self) -> eyre::Result<()> {
        let router = Router::new()
            .route("/healthz", get(healthz_handler))
            .route("/attest/{appdata}", get(attest_handler))
            .with_state(ServerState {
                quote_provider: self.quote_provider,
                metrics: self.metrics,
            });

        let listener = TcpListener::bind(self.config.listen_addr).await?;
        info!(
            message = "starting server",
            address = listener.local_addr()?.to_string()
        );

        axum::serve(
            listener,
            router.into_make_service_with_connect_info::<SocketAddr>(),
        )
        .with_graceful_shutdown(shutdown_signal())
        .await
        .map_err(|e| eyre::eyre!("failed to start server: {}", e))
    }
}

async fn healthz_handler() -> impl IntoResponse {
    StatusCode::OK
}

async fn attest_handler(
    Path(appdata): Path<String>,
    State(server): State<ServerState>,
) -> impl IntoResponse {
    let start = Instant::now();

    // Decode hex
    let appdata = match hex::decode(appdata) {
        Ok(data) => data,
        Err(e) => {
            info!(target: "tdx_quote_provider", error = %e, "Invalid hex in report data for attestation");
            return Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .body(Body::from(
                    json!({"message": "Invalid hex in report data for attestation"}).to_string(),
                ))
                .unwrap()
                .into_response();
        }
    };

    // Convert to report data
    let report_data: [u8; 64] = match appdata.try_into() {
        Ok(data) => data,
        Err(e) => {
            info!(target: "tdx_quote_provider", error = ?e, "Invalid report data length for attestation");
            return Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .body(Body::from(
                    json!({"message": "Invalid report data length for attestation"}).to_string(),
                ))
                .unwrap()
                .into_response();
        }
    };

    // Get attestation
    match server.quote_provider.get_attestation(report_data) {
        Ok(attestation) => {
            let duration = start.elapsed();
            server
                .metrics
                .attest_duration
                .record(duration.as_secs_f64());

            Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "application/octet-stream")
                .body(Body::from(attestation))
                .unwrap()
                .into_response()
        }
        Err(e) => {
            tracing::error!(target: "tdx_quote_provider", error = %e, "Failed to get TDX attestation");
            Response::builder()
                .status(StatusCode::INTERNAL_SERVER_ERROR)
                .body(Body::from(
                    json!({"message": "Failed to get TDX attestation"}).to_string(),
                ))
                .unwrap()
                .into_response()
        }
    }
}

async fn shutdown_signal() {
    let mut sigterm = signal::unix::signal(signal::unix::SignalKind::terminate()).unwrap();
    let mut sigint = signal::unix::signal(signal::unix::SignalKind::interrupt()).unwrap();

    tokio::select! {
        _ = signal::ctrl_c() => {
            info!("Received Ctrl+C, shutting down gracefully");
        }
        _ = sigterm.recv() => {
            info!("Received SIGTERM, shutting down gracefully");
        }
        _ = sigint.recv() => {
            info!("Received SIGINT, shutting down gracefully");
        }
    }
}
