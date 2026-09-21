//! The SDM operator opt-in as gauges, so an alert can scope itself by series value — "is SDM
//! meant to be on here" — rather than by a hardcoded list of network names. Combined with the
//! chain's Lagoon activation gauge they give the effective state without a value that could go
//! stale on a node that stops building.
//!
//! Scoped `op_sdm` rather than to a builder, so the series name does not encode which builder
//! emitted it.
//!
//! Each gauge is its own `Metrics` struct on purpose: constructing a `Metrics` struct registers
//! every field on it, so merging them would publish a value a node has no opinion on, such as a
//! boot-time value from a flag that was never configured.
//!
//! Handles are resolved on every record rather than through the generated `Default`, which caches
//! them the first time it runs and binds them to whichever recorder is global at that instant for
//! the rest of the process. Re-resolving means a record before the recorder is installed drops that
//! one sample instead of silencing the gauge permanently. Metric descriptions are still registered
//! once per process, so only the first recorder to see a gauge receives its `# HELP` text.

use reth_metrics::{
    Metrics,
    metrics::{self, Gauge},
};

const NO_LABELS: &[(&str, &str)] = &[];

/// The operator opt-in the node booted with, 0 or 1.
///
/// Written from the boot-time configuration and never by a runtime write: this is the value the
/// flag returns to on restart, so a sequencer whose runtime opt-in differs from it will silently
/// flip SDM on its next restart.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmOperatorOptInConfiguredMetrics {
    /// The SDM operator opt-in as configured at boot.
    operator_opt_in_configured: Gauge,
}

/// The operator opt-in gate, 0 or 1.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmOperatorOptInMetrics {
    /// Whether the operator has opted in to SDM post-exec production.
    operator_opt_in: Gauge,
}

/// Records the operator opt-in the node booted with.
pub(crate) fn record_operator_opt_in_configured(enabled: bool) {
    SdmOperatorOptInConfiguredMetrics::new_with_labels(NO_LABELS)
        .operator_opt_in_configured
        .set(gauge_value(enabled));
}

/// Records the operator opt-in gate.
pub(crate) fn record_operator_opt_in(enabled: bool) {
    SdmOperatorOptInMetrics::new_with_labels(NO_LABELS).operator_opt_in.set(gauge_value(enabled));
}

const fn gauge_value(flag: bool) -> f64 {
    if flag { 1.0 } else { 0.0 }
}
