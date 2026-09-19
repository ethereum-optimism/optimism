//! SDM enablement gauges, so an alert can scope itself by series value — "is SDM on here" —
//! rather than by a hardcoded list of network names.
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
///
/// Deliberately a separate struct from [`SdmEffectiveMetrics`] even though both are
/// `op_sdm`-scoped: constructing a `Metrics` struct registers every gauge on it, so merging the two
/// would publish `effective` as a flat 0 from nodes that never build a payload and have no opinion
/// on it.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmOperatorOptInMetrics {
    /// Whether the operator has opted in to SDM post-exec production.
    operator_opt_in: Gauge,
}

/// Both SDM gates resolved together, 0 or 1.
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmEffectiveMetrics {
    /// Whether SDM post-exec production is active: the operator opt-in and the protocol gate.
    effective: Gauge,
}

/// Records the operator opt-in gate.
pub(crate) fn record_operator_opt_in(enabled: bool) {
    SdmOperatorOptInMetrics::default().operator_opt_in.set(gauge_value(enabled));
}

/// Records both gates as resolved for a block this node is producing.
pub(crate) fn record_effective(effective: bool) {
    SdmEffectiveMetrics::default().effective.set(gauge_value(effective));
}

const fn gauge_value(flag: bool) -> f64 {
    if flag { 1.0 } else { 0.0 }
}
