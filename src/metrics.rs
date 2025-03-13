use std::{net::SocketAddr, sync::Arc, time::Duration};

use eyre::Result;
use metrics::{counter, histogram, Counter, Histogram};
use metrics_derive::Metrics;
use metrics_exporter_prometheus::PrometheusBuilder;
use metrics_util::layers::{PrefixLayer, Stack};

use crate::{init_metrics_server, server::PayloadSource, Args};

#[derive(Metrics)]
#[metrics(scope = "rpc")]
pub struct ServerMetrics {
    // Builder proxy metrics
    #[metric(describe = "Latency for builder client forwarded rpc calls (excluding the engine api)", labels = ["method"])]
    #[allow(dead_code)]
    pub builder_forwarded_call: Histogram,

    #[metric(describe = "Number of builder client rpc responses", labels = ["http_status_code", "rpc_status_code", "method"])]
    #[allow(dead_code)]
    pub builder_rpc_response_count: Counter,

    // L2 proxy metrics
    #[metric(describe = "Latency for l2 client forwarded rpc calls (excluding the engine api)", labels = ["method"])]
    #[allow(dead_code)]
    pub l2_forwarded_call: Histogram,

    #[metric(describe = "Number of l2 client rpc responses", labels = ["http_status_code", "rpc_status_code", "method"])]
    #[allow(dead_code)]
    pub l2_rpc_response_count: Counter,
}

impl ServerMetrics {
    pub fn increment_blocks_created(&self, source: &PayloadSource) {
        counter!("rpc.blocks_created", "source" => source.to_string()).increment(1);
    }

    pub fn record_builder_forwarded_call(&self, latency: Duration, method: String) {
        histogram!("rpc.builder_forwarded_call", "method" => method).record(latency.as_secs_f64());
    }

    pub fn increment_builder_rpc_response_count(
        &self,
        http_status_code: String,
        rpc_status_code: Option<String>,
        method: String,
    ) {
        counter!("rpc.builder_response_count",
            "http_status_code" => http_status_code,
            "rpc_status_code" => rpc_status_code.unwrap_or("".to_string()),
            "method" => method,
        )
        .increment(1);
    }

    pub fn record_l2_forwarded_call(&self, latency: Duration, method: String) {
        histogram!("rpc.l2_forwarded_call", "method" => method).record(latency.as_secs_f64());
    }

    pub fn increment_l2_rpc_response_count(
        &self,
        http_status_code: String,
        rpc_status_code: Option<String>,
        method: String,
    ) {
        counter!("rpc.l2_response_count",
            "http_status_code" => http_status_code,
            "rpc_status_code" => rpc_status_code.unwrap_or("".to_string()),
            "method" => method,
        )
        .increment(1);
    }
}

pub(crate) fn init_metrics(args: &Args) -> Result<Option<Arc<ServerMetrics>>> {
    if args.metrics {
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();

        Stack::new(recorder)
            .push(PrefixLayer::new("rollup-boost"))
            .install()?;

        // Start the metrics server
        let metrics_addr = format!("{}:{}", args.metrics_host, args.metrics_port);
        let addr: SocketAddr = metrics_addr.parse()?;
        tokio::spawn(init_metrics_server(addr, handle)); // Run the metrics server in a separate task
        Ok(Some(Arc::new(ServerMetrics::default())))
    } else {
        Ok(None)
    }
}
