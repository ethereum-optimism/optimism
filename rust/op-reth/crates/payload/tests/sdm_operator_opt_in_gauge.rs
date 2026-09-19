//! Exposition test for `reth_op_sdm_operator_opt_in`.
//!
//! Its own integration binary, holding exactly one test, because the metrics recorder is global
//! and the assertion below requires that nothing else in the process has reported the effective
//! gate. Any co-resident payload build would register it and break that.

use metrics_exporter_prometheus::PrometheusBuilder;
use reth_node_metrics::recorder::{
    install_prometheus_recorder, try_install_prometheus_recorder_with_builder,
};
use reth_optimism_payload_builder::config::OpBuilderConfig;

/// Every store to the operator gate reports its value, so an alert can tell a node that opted in
/// from one that silently booted with SDM off — the flag is in-memory and starts `false`.
///
/// Reporting the operator gate must not register the effective gate along with it: a node that
/// never builds has no opinion on whether production is effective, and a flat 0 there would be
/// indistinguishable from SDM being switched off on a sequencer.
#[test]
fn operator_opt_in_gauge_follows_every_write() {
    // Install before the first record: the generated `Default` latches whichever recorder is
    // global on the process's first write, permanently.
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
    assert!(
        !exposition.contains("reth_op_sdm_effective"),
        "operator gate must not register the effective gate: {exposition}"
    );
}
