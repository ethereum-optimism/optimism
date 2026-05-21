use metrics::Histogram;
use metrics_derive::Metrics;

#[derive(Metrics, Clone)]
#[metrics(scope = "tdx_quote_provider")]
pub struct Metrics {
    /// Duration of attestation request
    pub attest_duration: Histogram,
}
