//! Metrics utilities for the host application.

use std::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    thread,
    time::Duration,
};

use metrics::{describe_gauge, gauge};
use metrics_exporter_prometheus::PrometheusBuilder;
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

/// Initialize the metrics server on the given port. Fails when the listener
/// cannot be installed (e.g. the port is already bound).
pub fn init_metrics(port: &u16) -> anyhow::Result<()> {
    let builder = PrometheusBuilder::new().with_http_listener(SocketAddr::new(
        IpAddr::V4(Ipv4Addr::new(0, 0, 0, 0)),
        port.to_owned(),
    ));

    builder.install().map_err(|e| anyhow::anyhow!("failed to start metrics server: {e}"))?;

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
    Ok(())
}
