//! The SDM operator opt-in as a gauge, so an alert can scope itself by series value — "is SDM
//! meant to be on here" — rather than by a hardcoded list of network names. Combined with the
//! chain's Lagoon activation gauge it gives the effective state without a value that could go
//! stale on a node that stops building.
//!
//! Scoped `op_sdm` rather than to a builder, so the series name does not encode which builder
//! emitted it.
//!
//! The generated `Default` caches its handles the first time one of these is recorded, binding
//! them to whichever recorder is global at that instant for the rest of the process. A record
//! before the recorder is installed therefore silences the gauge permanently rather than losing
//! one sample.

use reth_metrics::{
    Metrics,
    metrics::{self, Gauge},
};

/// The operator opt-in gate, 0 or 1.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmOperatorOptInMetrics {
    /// Whether the operator has opted in to SDM post-exec production.
    operator_opt_in: Gauge,
}

/// Records the operator opt-in gate.
pub(crate) fn record_operator_opt_in(enabled: bool) {
    SdmOperatorOptInMetrics::default().operator_opt_in.set(gauge_value(enabled));
}

const fn gauge_value(flag: bool) -> f64 {
    if flag { 1.0 } else { 0.0 }
}
