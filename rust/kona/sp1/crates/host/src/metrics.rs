//! Metrics utilities for the host application.

use std::{
    fmt,
    net::{IpAddr, Ipv4Addr, SocketAddr, TcpListener},
    str::FromStr,
    thread,
    time::Duration,
};

use anyhow::{Context, anyhow};
use metrics::{describe_gauge, gauge};
use metrics_exporter_prometheus::{BuildError, PrometheusBuilder};
use metrics_process::Collector;
use strum::{EnumMessage, IntoEnumIterator};

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

/// Where the Prometheus exporter listens.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub enum MetricsListen {
    /// No exporter is installed.
    #[default]
    Disabled,
    /// A kernel-assigned port on all interfaces, reported back by
    /// [`init_metrics`]. For deployments running several instances on one
    /// host, where any fixed port risks a collision.
    Ephemeral,
    /// The given port on all interfaces.
    Fixed(u16),
}

/// Configured value selecting [`MetricsListen::Ephemeral`].
const EPHEMERAL: &str = "auto";

/// Configured value selecting [`MetricsListen::Disabled`], alongside `0` and
/// the empty value.
const DISABLED: &str = "disabled";

/// Probe-then-bind attempts before an ephemeral listen gives up.
const EPHEMERAL_BIND_ATTEMPTS: usize = 10;

impl FromStr for MetricsListen {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.trim() {
            "" | "0" | DISABLED => Ok(Self::Disabled),
            EPHEMERAL => Ok(Self::Ephemeral),
            port => Ok(Self::Fixed(
                port.parse().map_err(|err| anyhow!("expected a port or `{EPHEMERAL}`: {err}"))?,
            )),
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

/// Initialize the metrics server, returning the address it listens on, or
/// `None` when metrics are disabled. Fails when the listener cannot be
/// installed (e.g. the port is already bound).
pub fn init_metrics(listen: MetricsListen) -> anyhow::Result<Option<SocketAddr>> {
    let addr = match listen {
        MetricsListen::Disabled => return Ok(None),
        MetricsListen::Fixed(port) => {
            let addr = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), port);
            install_exporter(addr).map_err(|e| anyhow!("failed to start metrics server: {e}"))?;
            addr
        }
        MetricsListen::Ephemeral => install_exporter_on_ephemeral_port()?,
    };

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

fn install_exporter(addr: SocketAddr) -> Result<(), BuildError> {
    PrometheusBuilder::new().with_http_listener(addr).install()
}

/// The exporter binds the address itself and never reports what it bound, so
/// port 0 cannot be delegated to it: a port is reserved here and handed over
/// once the reservation is closed. Another socket can take the port in that
/// window, so an occupied port is retried rather than fatal. `install` only
/// registers the global recorder once the listener is bound, leaving a failed
/// attempt with nothing to undo.
fn install_exporter_on_ephemeral_port() -> anyhow::Result<SocketAddr> {
    let mut last_err = None;
    for _ in 0..EPHEMERAL_BIND_ATTEMPTS {
        let addr = reserve_ephemeral_addr()?;
        match install_exporter(addr) {
            Ok(()) => return Ok(addr),
            Err(BuildError::FailedToCreateHTTPListener(err)) => last_err = Some(err),
            Err(err) => return Err(anyhow!("failed to start metrics server: {err}")),
        }
    }
    Err(anyhow!(
        "failed to start metrics server on an ephemeral port in {EPHEMERAL_BIND_ATTEMPTS} \
         attempts: {}",
        last_err.unwrap_or_default()
    ))
}

fn reserve_ephemeral_addr() -> anyhow::Result<SocketAddr> {
    let listener = TcpListener::bind((Ipv4Addr::UNSPECIFIED, 0))
        .context("failed to reserve an ephemeral metrics port")?;
    listener.local_addr().context("failed to read the reserved ephemeral metrics port")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_metrics_listen() {
        for (input, want) in [
            ("", MetricsListen::Disabled),
            ("0", MetricsListen::Disabled),
            (" auto ", MetricsListen::Ephemeral),
            ("9090", MetricsListen::Fixed(9090)),
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

    #[test]
    fn ephemeral_listen_reports_the_port_it_holds() {
        let addr = init_metrics(MetricsListen::Ephemeral).unwrap().expect("an exporter is running");
        assert_ne!(addr.port(), 0, "the reported port must be the resolved one");
        assert!(TcpListener::bind(addr).is_err(), "the exporter must hold the port it reported");
    }

    #[test]
    fn disabled_listen_installs_nothing() {
        assert_eq!(init_metrics(MetricsListen::Disabled).unwrap(), None);
    }

    #[test]
    fn reserves_a_bindable_ephemeral_port() {
        let addr = reserve_ephemeral_addr().unwrap();
        assert_ne!(addr.port(), 0, "the kernel must resolve port 0");
        // The reservation is released on return, so the port rebinds.
        TcpListener::bind(addr).unwrap();
    }
}
