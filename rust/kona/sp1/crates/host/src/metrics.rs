//! Metrics utilities for the host application.

use std::{
    convert::Infallible,
    fmt,
    net::{IpAddr, Ipv4Addr, SocketAddr},
    num::NonZeroU16,
    str::FromStr,
    thread,
    time::Duration,
};

use anyhow::{Context, anyhow};
use bytes::Bytes;
use http_body_util::Full;
use hyper::{
    Request, Response, StatusCode, body::Incoming, server::conn::http1, service::service_fn,
};
use hyper_util::rt::TokioIo;
use metrics::{describe_gauge, gauge};
use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle};
use metrics_process::Collector;
use strum::{EnumMessage, IntoEnumIterator};
use tokio::net::TcpListener;

/// Trait for metrics gauge that provides common functionality.
pub trait MetricsGauge: Sized + IntoEnumIterator + EnumMessage + ToString {
    /// Describe the gauge metric.
    fn describe(&self) {
        describe_gauge!(self.to_string(), self.get_message().unwrap());
    }

    /// Set the gauge value.
    fn set(&self, value: f64) {
        gauge!(self.to_string()).set(value);
    }

    /// Increment the gauge value.
    fn increment(&self, value: f64) {
        gauge!(self.to_string()).increment(value);
    }

    /// Register all gauges.
    fn register_all() {
        for metric in Self::iter() {
            metric.describe();
        }
    }

    /// Initialize all gauges to 0.0.
    fn init_all() {
        for metric in Self::iter() {
            metric.set(0.0);
        }
    }
}

/// Where the Prometheus endpoint listens.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub enum MetricsListen {
    /// No recorder is installed and no endpoint is served.
    #[default]
    Disabled,
    /// A kernel-assigned port on all interfaces, reported back by
    /// [`init_metrics`]. For deployments running several instances on one
    /// host, where any fixed port risks a collision.
    Ephemeral,
    /// The given port on all interfaces.
    Fixed(NonZeroU16),
}

/// Configured value selecting [`MetricsListen::Ephemeral`].
const EPHEMERAL: &str = "auto";

/// Configured value selecting [`MetricsListen::Disabled`], alongside any
/// numeric zero and the empty value.
const DISABLED: &str = "disabled";

/// How often the recorder is swept; `install_recorder` leaves upkeep to the
/// caller. Matches the exporter's own default.
const UPKEEP_INTERVAL: Duration = Duration::from_secs(5);

impl FromStr for MetricsListen {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.trim() {
            "" | DISABLED => Ok(Self::Disabled),
            EPHEMERAL => Ok(Self::Ephemeral),
            port => {
                let port: u16 = port
                    .parse()
                    .map_err(|err| anyhow!("expected a port or `{EPHEMERAL}`: {err}"))?;
                // Any spelling of zero disables, as it did before the mode
                // became explicit.
                Ok(NonZeroU16::new(port).map_or(Self::Disabled, Self::Fixed))
            }
        }
    }
}

impl fmt::Display for MetricsListen {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Disabled => f.write_str(DISABLED),
            Self::Ephemeral => f.write_str(EPHEMERAL),
            Self::Fixed(port) => write!(f, "{port}"),
        }
    }
}

/// Initialize the metrics endpoint, returning the address it serves on, or
/// `None` when metrics are disabled. Fails when the address cannot be bound
/// (e.g. the port is already taken).
///
/// The endpoint serves from the listener bound here rather than delegating an
/// address to the exporter, so an ephemeral port is resolved by the very bind
/// that keeps it: nothing can take the port in between.
pub async fn init_metrics(listen: MetricsListen) -> anyhow::Result<Option<SocketAddr>> {
    let port = match listen {
        MetricsListen::Disabled => return Ok(None),
        MetricsListen::Ephemeral => 0,
        MetricsListen::Fixed(port) => port.get(),
    };

    let listener = TcpListener::bind(SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), port))
        .await
        .context("failed to bind the metrics endpoint")?;
    let addr = listener.local_addr().context("failed to read the metrics endpoint address")?;

    let handle = PrometheusBuilder::new()
        .install_recorder()
        .map_err(|e| anyhow!("failed to install the metrics recorder: {e}"))?;

    let upkeep = handle.clone();
    tokio::spawn(async move {
        loop {
            tokio::time::sleep(UPKEEP_INTERVAL).await;
            upkeep.run_upkeep();
        }
    });
    tokio::spawn(serve_metrics(listener, handle));

    // Spawn a thread to collect process metrics.
    thread::spawn(move || {
        let collector = Collector::default();
        collector.describe();
        loop {
            // Periodically call `collect()` method to update information.
            collector.collect();
            thread::sleep(Duration::from_millis(750));
        }
    });
    Ok(Some(addr))
}

/// Serves the rendered registry on `/metrics` until the process ends.
async fn serve_metrics(listener: TcpListener, handle: PrometheusHandle) {
    loop {
        let stream = match listener.accept().await {
            Ok((stream, _)) => stream,
            Err(err) => {
                // Back off rather than spin: an exhausted descriptor table
                // fails every accept until something else frees one.
                tracing::warn!(%err, "metrics endpoint could not accept a connection");
                tokio::time::sleep(Duration::from_millis(100)).await;
                continue;
            }
        };

        let handle = handle.clone();
        tokio::spawn(async move {
            let service = service_fn(move |req: Request<Incoming>| {
                let response = if req.uri().path() == "/metrics" {
                    Response::builder()
                        .header("content-type", "text/plain")
                        .body(Full::new(Bytes::from(handle.render())))
                } else {
                    Response::builder().status(StatusCode::NOT_FOUND).body(Full::new(Bytes::new()))
                };
                async move {
                    Ok::<_, Infallible>(
                        response.expect("a response with a valid status and header builds"),
                    )
                }
            });

            if let Err(err) =
                http1::Builder::new().serve_connection(TokioIo::new(stream), service).await
            {
                tracing::debug!(%err, "metrics connection ended");
            }
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixed(port: u16) -> MetricsListen {
        MetricsListen::Fixed(NonZeroU16::new(port).unwrap())
    }

    #[test]
    fn parses_metrics_listen() {
        for (input, want) in [
            ("", MetricsListen::Disabled),
            ("0", MetricsListen::Disabled),
            // Every spelling of zero disables, not just the canonical one.
            ("00", MetricsListen::Disabled),
            (" auto ", MetricsListen::Ephemeral),
            ("9090", fixed(9090)),
        ] {
            assert_eq!(input.parse::<MetricsListen>().unwrap(), want, "parsing {input:?}");
            // A rendered mode parses back to itself, so a resolved config can
            // be passed on to a child process.
            assert_eq!(want.to_string().parse::<MetricsListen>().unwrap(), want);
        }
    }

    #[test]
    fn rejects_non_port_metrics_listen() {
        assert!("ephemeral".parse::<MetricsListen>().is_err());
        assert!("70000".parse::<MetricsListen>().is_err());
    }

    #[tokio::test]
    async fn ephemeral_listen_serves_on_the_port_it_reports() {
        let addr =
            init_metrics(MetricsListen::Ephemeral).await.unwrap().expect("an endpoint is serving");
        assert_ne!(addr.port(), 0, "the reported port must be the resolved one");
        gauge!("kona_sp1_metrics_self_test").set(1.0);

        let url = format!("http://127.0.0.1:{}/metrics", addr.port());
        let body =
            reqwest::get(&url).await.unwrap().error_for_status().unwrap().text().await.unwrap();
        assert!(body.contains("kona_sp1_metrics_self_test 1"), "scraped:\n{body}");

        let missing = reqwest::get(format!("http://127.0.0.1:{}/nope", addr.port())).await.unwrap();
        assert_eq!(missing.status(), 404);
    }

    #[tokio::test]
    async fn disabled_listen_installs_nothing() {
        assert_eq!(init_metrics(MetricsListen::Disabled).await.unwrap(), None);
    }
}
