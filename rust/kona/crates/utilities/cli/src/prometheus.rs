//! Utilities for spinning up a prometheus metrics server.
//!
//! # Design
//!
//! We manually serve metrics via `hyper` rather than using the built-in HTTP listener from
//! `metrics-exporter-prometheus`. This is because `PrometheusBuilder::with_http_listener`
//! doesn't expose the actual bound address—it only accepts a [`SocketAddr`] and returns
//! `Result<(), BuildError>`. When port 0 is used (letting the OS assign an available port),
//! there's no way to discover what port was actually assigned.
//!
//! We considered pre-binding a `TcpListener` to get the port, dropping it, then passing that
//! port to the builder—but this has a race condition where another process could grab the port
//! in between.
//!
//! Instead, we use [`PrometheusBuilder::build_recorder`] to create the recorder without an HTTP
//! server, bind our own [`TcpListener`], and serve metrics manually. This approach:
//! - Eliminates the race condition (the same listener is used throughout)
//! - Allows returning the actual bound address to callers
//! - Follows the same pattern used by [reth's metrics infrastructure][reth-metrics]
//!
//! [reth-metrics]: https://github.com/paradigmxyz/reth/blob/main/crates/node/metrics/src/server.rs

use crate::PrometheusError;
use http::{Response, header::CONTENT_TYPE};
use http_body_util::Full;
use hyper::{body::Bytes, server::conn::http1, service::service_fn};
use hyper_util::rt::TokioIo;
use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle};
use metrics_process::Collector;
use std::{convert::Infallible, net::SocketAddr, sync::Arc, time::Duration};
use tokio::{net::TcpListener, sync::Semaphore};
use tracing::{error, info};

/// Start a Prometheus metrics server on the given address.
///
/// If the port is 0, the OS will assign an available port.
/// Returns the actual address the server is bound to.
///
/// This function must be called from within a tokio runtime.
pub async fn init_prometheus_server(addr: SocketAddr) -> Result<SocketAddr, PrometheusError> {
    let listener = TcpListener::bind(addr).await?;
    let actual_addr = listener.local_addr()?;

    // Build recorder without HTTP server - we serve it ourselves
    let recorder = PrometheusBuilder::new().build_recorder();
    let handle = Arc::new(recorder.handle());

    // Set as global recorder
    metrics::set_global_recorder(recorder)?;

    // Spawn task for periodic system metrics collection
    tokio::spawn(async move {
        let collector = Collector::default();
        collector.describe();
        loop {
            collector.collect();
            tokio::time::sleep(Duration::from_secs(60)).await;
        }
    });

    // Spawn task to serve metrics endpoint
    tokio::spawn(serve_metrics(listener, handle));

    info!(
        target: "prometheus",
        "Serving metrics at: http://{}",
        actual_addr
    );

    Ok(actual_addr)
}

async fn serve_metrics(listener: TcpListener, handle: Arc<PrometheusHandle>) {
    let semaphore = Arc::new(Semaphore::new(100));

    loop {
        let permit = match Arc::clone(&semaphore).acquire_owned().await {
            Ok(permit) => permit,
            Err(_) => {
                error!(target: "prometheus", "metrics connection semaphore closed");
                return;
            }
        };

        let (stream, _) = match listener.accept().await {
            Ok(conn) => conn,
            Err(e) => {
                error!(target: "prometheus", "failed to accept connection: {}", e);
                continue;
            }
        };

        let handle = Arc::clone(&handle);
        tokio::spawn(async move {
            let _permit = permit;

            let service = service_fn(move |_req| {
                let metrics = handle.render();
                async move {
                    let response = Response::builder()
                        .header(CONTENT_TYPE, "text/plain; charset=utf-8")
                        .body(Full::new(Bytes::from(metrics)))
                        .expect("building metrics response with valid inputs should not fail");
                    Ok::<_, Infallible>(response)
                }
            });

            if let Err(e) =
                http1::Builder::new().serve_connection(TokioIo::new(stream), service).await
            {
                error!(target: "prometheus", "error serving metrics: {}", e);
            }
        });
    }
}
