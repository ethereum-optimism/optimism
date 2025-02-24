use std::time::Duration;

use metrics::{counter, histogram, Counter, Histogram};
use metrics_derive::Metrics;

use crate::server::PayloadSource;

#[derive(Metrics)]
#[metrics(scope = "rpc")]
pub struct ServerMetrics {
    #[metric(describe = "Total latency for server `engine_newPayloadV3` call")]
    pub new_payload_v3_total: Histogram,

    #[metric(describe = "Total latency for server `engine_getPayloadV3` call")]
    pub get_payload_v3_total: Histogram,

    #[metric(describe = "Total latency for server `engine_forkChoiceUpdatedV3` call")]
    pub fork_choice_updated_v3_total: Histogram,

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

#[derive(Metrics)]
#[metrics(scope = "rpc")]
pub struct ClientMetrics {
    #[metric(describe = "Latency for client `engine_newPayloadV3` call")]
    #[allow(dead_code)]
    pub new_payload_v3: Histogram,

    #[metric(describe = "Number of client `engine_newPayloadV3` responses", labels = ["code"])]
    #[allow(dead_code)]
    pub new_payload_v3_response_count: Counter,

    #[metric(describe = "Latency for client `engine_getPayloadV3` call")]
    #[allow(dead_code)]
    pub get_payload_v3: Histogram,

    #[metric(describe = "Number of client `engine_getPayloadV3` responses", labels = ["code"])]
    #[allow(dead_code)]
    pub get_payload_v3_response_count: Counter,

    #[metric(describe = "Latency for client `engine_forkChoiceUpdatedV3` call")]
    #[allow(dead_code)]
    pub fork_choice_updated_v3: Histogram,

    #[metric(describe = "Number of client `engine_forkChoiceUpdatedV3` responses", labels = ["code"])]
    #[allow(dead_code)]
    pub fork_choice_updated_v3_response_count: Counter,
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

impl ClientMetrics {
    pub fn new(source: &PayloadSource) -> Self {
        Self {
            new_payload_v3: histogram!("rpc.new_payload_v3", "target" => source.to_string()),
            get_payload_v3: histogram!("rpc.get_payload_v3", "target" => source.to_string()),
            fork_choice_updated_v3: histogram!("rpc.fork_choice_updated_v3", "target" => source.to_string()),
            new_payload_v3_response_count: counter!("rpc.new_payload_v3_response_count"),
            get_payload_v3_response_count: counter!("rpc.get_payload_v3_response_count"),
            fork_choice_updated_v3_response_count: counter!(
                "rpc.fork_choice_updated_v3_response_count"
            ),
        }
    }

    pub fn record_new_payload_v3(&self, latency: Duration, code: String) {
        self.new_payload_v3.record(latency.as_secs_f64());
        counter!("rpc.new_payload_v3_response_count", "code" => code).increment(1);
    }

    pub fn record_get_payload_v3(&self, latency: Duration, code: String) {
        self.get_payload_v3.record(latency.as_secs_f64());
        counter!("rpc.get_payload_v3_response_count", "code" => code).increment(1);
    }

    pub fn record_fork_choice_updated_v3(&self, latency: Duration, code: String) {
        self.fork_choice_updated_v3.record(latency.as_secs_f64());
        counter!("rpc.fork_choice_updated_v3_response_count", "code" => code).increment(1);
    }
}
