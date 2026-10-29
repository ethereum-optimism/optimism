//! Exposition test for `reth_op_sdm_operator_opt_in`.
//!
//! Its own integration binary because the metrics recorder is global: every write of the flag in
//! the process lands on it, and the assertions below read back a single shared value.

use metrics_exporter_prometheus::PrometheusBuilder;
use reth_node_metrics::recorder::{
    install_prometheus_recorder, try_install_prometheus_recorder_with_builder,
};
use reth_optimism_payload_builder::config::OpBuilderConfig;

/// Every store to the operator gate reports its value, so an alert can tell a node that opted in
/// from one that silently booted with SDM off — the flag is in-memory and starts `false`.
#[test]
fn operator_opt_in_gauge_follows_every_write() {
    // Install before the first record so every write below lands on this recorder.
    let recorder = try_install_prometheus_recorder_with_builder(PrometheusBuilder::new())
        .unwrap_or_else(|_| install_prometheus_recorder());
    let opt_in = OpBuilderConfig::default().operator_sdm_opt_in;

    opt_in.set(true);
    let exposition = recorder.handle().render();
    assert!(
        exposition.contains("reth_op_sdm_operator_opt_in 1"),
        "opting in must report 1: {exposition}"
    );

    opt_in.set(false);
    let exposition = recorder.handle().render();
    assert!(
        exposition.contains("reth_op_sdm_operator_opt_in 0"),
        "opting back out must report 0: {exposition}"
    );
}
