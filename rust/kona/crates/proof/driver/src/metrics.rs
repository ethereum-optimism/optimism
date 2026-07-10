//! Optional, generic instrumentation for the derivation driver.
//!
//! [`Driver::advance_to_target`] runs a hot loop with two distinct, expensive phases per block:
//! payload derivation and block execution. Consumers that care about the relative cost of these
//! phases (e.g. a zkVM host tracking guest cycle counts) can supply a [`DriverMetrics`] collector
//! to observe when each phase begins and ends.
//!
//! This interface is deliberately free of any prover- or backend-specific concepts: it only names
//! the phases the driver already has. The default implementation is a no-op, so the canonical
//! derivation path pays nothing.
//!
//! [`Driver::advance_to_target`]: crate::Driver::advance_to_target

/// A phase of the per-block work performed by the derivation driver.
///
/// Phases are reported in the order they execute within a single loop iteration of
/// [`Driver::advance_to_target`]: [`DriverPhase::PayloadDerivation`] then
/// [`DriverPhase::BlockExecution`].
///
/// [`Driver::advance_to_target`]: crate::Driver::advance_to_target
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[non_exhaustive]
pub enum DriverPhase {
    /// Deriving the next block's payload attributes from the pipeline.
    PayloadDerivation,
    /// Executing the derived payload attributes to build the next block.
    BlockExecution,
}

impl DriverPhase {
    /// Returns a stable, lower-kebab-case identifier for the phase.
    ///
    /// The identifier is stable across releases so that external collectors can key persistent
    /// metrics off of it.
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::PayloadDerivation => "payload-derivation",
            Self::BlockExecution => "block-execution",
        }
    }
}

/// A collector notified when the derivation driver enters and exits each [`DriverPhase`].
///
/// Both hooks default to no-ops, so implementers only override the phases they care about, and the
/// default collector ([`NoopDriverMetrics`]) compiles away entirely.
///
/// Hooks are **not** guaranteed to be balanced. A [`DriverMetrics::phase_start`] is followed by a
/// matching [`DriverMetrics::phase_end`] only when the phase runs to completion. A phase that
/// aborts the iteration mid-flight emits a `phase_start` with no matching `phase_end`: payload
/// derivation does so when the data source is exhausted, and block execution does so when a
/// pre-Holocene block is discarded after a failed execution. Collectors that accumulate deltas
/// between start and end (e.g. the SP1 cycle tracker, whose markers sum per key) tolerate this;
/// stateful collectors that pair calls must not assume every `phase_start` is closed. Iterations
/// that reach the target return before entering any phase.
pub trait DriverMetrics {
    /// Called immediately before the driver begins `phase`.
    #[inline]
    fn phase_start(&self, phase: DriverPhase) {
        let _ = phase;
    }

    /// Called immediately after the driver finishes `phase`.
    #[inline]
    fn phase_end(&self, phase: DriverPhase) {
        let _ = phase;
    }
}

/// The default [`DriverMetrics`] collector: every hook is a no-op.
///
/// Used by [`Driver::advance_to_target`] so the canonical derivation path is not instrumented and
/// incurs no overhead.
///
/// [`Driver::advance_to_target`]: crate::Driver::advance_to_target
#[derive(Debug, Clone, Copy, Default)]
pub struct NoopDriverMetrics;

impl DriverMetrics for NoopDriverMetrics {}
