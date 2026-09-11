//! Utilities for spinning up a prometheus metrics server.

use crate::{CliError, CliResult};
use kona_metrics::ChainLabelRecorder;
use metrics_exporter_prometheus::PrometheusBuilder;
use metrics_process::Collector;
use std::{
    net::{IpAddr, SocketAddr},
    thread::{self, sleep},
    time::Duration,
};
use tracing::info;

/// Start a Prometheus metrics server on the given port.
///
/// `chain_id` labels every metric of a single-chain process, including those emitted outside any
/// [`kona_metrics::scoped`] scope. A multi-chain process passes `None`.
pub fn init_prometheus_server(
    addr: IpAddr,
    metrics_port: u16,
    chain_id: Option<u64>,
) -> CliResult<()> {
    let prometheus_addr = SocketAddr::from((addr, metrics_port));
    let builder = PrometheusBuilder::new().with_http_listener(prometheus_addr);

    // `PrometheusBuilder::install` installs the recorder itself, leaving no seam to wrap it in.
    // This is that method's body with the wrapper added.
    let recorder = if let Ok(handle) = tokio::runtime::Handle::try_current() {
        let (recorder, exporter) = {
            let _guard = handle.enter();
            builder.build()?
        };
        handle.spawn(exporter);
        recorder
    } else {
        let runtime = tokio::runtime::Builder::new_current_thread().enable_all().build()?;
        let (recorder, exporter) = {
            let _guard = runtime.enter();
            builder.build()?
        };
        thread::Builder::new()
            .name("metrics-exporter-prometheus-http-listener".to_string())
            .spawn(move || runtime.block_on(exporter))?;
        recorder
    };

    let recorder = ChainLabelRecorder::new(recorder, chain_id.map(kona_metrics::chain_label));
    metrics::set_global_recorder(recorder)
        .map_err(|e| CliError::Metrics(format!("failed to install the metrics recorder: {e}")))?;

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
