//! Utilities for spinning up a prometheus metrics server.

use bytes::Bytes;
use http_body_util::Full;
use hyper::{Request, Response, body::Incoming, server::conn::http1, service::service_fn};
use hyper_util::rt::TokioIo;
use metrics_exporter_prometheus::{BuildError, PrometheusBuilder, PrometheusHandle};
use metrics_process::Collector;
use std::{
    net::{IpAddr, SocketAddr, TcpListener},
    thread::{self, sleep},
    time::Duration,
};
use tokio::runtime;
use tracing::{error, info};

const PROMETHEUS_UPKEEP_INTERVAL: Duration = Duration::from_secs(5);

/// Start a Prometheus metrics server on the given port.
pub fn init_prometheus_server(addr: IpAddr, metrics_port: u16) -> Result<(), BuildError> {
    let (listener, prometheus_addr) = bind_prometheus_listener(addr, metrics_port)?;
    let handle = PrometheusBuilder::new().install_recorder()?;
    spawn_prometheus_server(listener, handle)?;

    // Initialise collector for system metrics e.g. CPU, memory, etc.
    let collector = Collector::default();
    collector.describe();

    thread::spawn(move || {
        loop {
            collector.collect();
            sleep(Duration::from_secs(60));
        }
    });

    info!(
        target: "prometheus",
        "Serving metrics at: http://{}",
        prometheus_addr
    );

    Ok(())
}

fn bind_prometheus_listener(
    addr: IpAddr,
    metrics_port: u16,
) -> Result<(TcpListener, SocketAddr), BuildError> {
    let listener = TcpListener::bind(SocketAddr::from((addr, metrics_port)))
        .map_err(|err| BuildError::FailedToCreateHTTPListener(err.to_string()))?;
    listener
        .set_nonblocking(true)
        .map_err(|err| BuildError::FailedToCreateHTTPListener(err.to_string()))?;
    let bound_addr = listener
        .local_addr()
        .map_err(|err| BuildError::FailedToCreateHTTPListener(err.to_string()))?;
    Ok((listener, bound_addr))
}

fn build_prometheus_runtime() -> Result<runtime::Runtime, BuildError> {
    runtime::Builder::new_current_thread()
        .enable_io()
        .enable_time()
        .build()
        .map_err(|err| BuildError::FailedToCreateRuntime(err.to_string()))
}

fn spawn_prometheus_server(
    listener: TcpListener,
    handle: PrometheusHandle,
) -> Result<(), BuildError> {
    if let Ok(runtime) = runtime::Handle::try_current() {
        let listener = {
            let _guard = runtime.enter();
            tokio::net::TcpListener::from_std(listener)
                .map_err(|err| BuildError::FailedToCreateHTTPListener(err.to_string()))?
        };
        runtime.spawn(run_prometheus_upkeep(handle.clone()));
        runtime.spawn(serve_prometheus(listener, handle));
    } else {
        let runtime = build_prometheus_runtime()?;
        let listener = {
            let _guard = runtime.enter();
            tokio::net::TcpListener::from_std(listener)
                .map_err(|err| BuildError::FailedToCreateHTTPListener(err.to_string()))?
        };
        thread::Builder::new()
            .name("kona-prometheus-exporter".to_string())
            .spawn(move || {
                runtime.spawn(run_prometheus_upkeep(handle.clone()));
                runtime.block_on(serve_prometheus(listener, handle));
            })
            .map_err(|err| BuildError::FailedToCreateRuntime(err.to_string()))?;
    }
    Ok(())
}

async fn run_prometheus_upkeep(handle: PrometheusHandle) {
    loop {
        tokio::time::sleep(PROMETHEUS_UPKEEP_INTERVAL).await;
        handle.run_upkeep();
    }
}

async fn serve_prometheus(listener: tokio::net::TcpListener, handle: PrometheusHandle) {
    loop {
        match listener.accept().await {
            Ok((stream, _)) => {
                let handle = handle.clone();
                tokio::spawn(async move {
                    let service = service_fn(move |request: Request<Incoming>| {
                        let handle = handle.clone();
                        async move {
                            let body = if request.uri().path() == "/health" {
                                Bytes::from_static(b"OK")
                            } else {
                                Bytes::from(
                                    tokio::task::spawn_blocking(move || handle.render())
                                        .await
                                        .expect("metrics rendering task must not panic"),
                                )
                            };
                            Ok::<_, hyper::Error>(
                                Response::builder()
                                    .header("content-type", "text/plain")
                                    .body(Full::new(body))
                                    .expect("valid metrics response"),
                            )
                        }
                    });
                    if let Err(err) =
                        http1::Builder::new().serve_connection(TokioIo::new(stream), service).await
                    {
                        error!(%err, "Error serving metrics connection");
                    }
                });
            }
            Err(err) => error!(%err, "Error accepting metrics connection"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::{IpAddr, Ipv4Addr};

    #[test]
    fn binds_ephemeral_prometheus_port() {
        let (listener, bound_addr) =
            bind_prometheus_listener(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).unwrap();

        assert_ne!(bound_addr.port(), 0);
        assert_eq!(listener.local_addr().unwrap(), bound_addr);
    }

    #[test]
    fn dedicated_prometheus_runtime_supports_timers() {
        let runtime = build_prometheus_runtime().unwrap();

        runtime.block_on(async { tokio::time::sleep(Duration::ZERO).await });
    }
}
