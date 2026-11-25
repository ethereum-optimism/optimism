//! SP1 cycle-tracker instrumentation for the derivation driver.

use kona_driver::{DriverMetrics, DriverPhase};

/// A [`DriverMetrics`] collector that emits SP1 `cycle-tracker` markers around each derivation
/// phase.
///
/// In the zkVM, the SP1 executor sums the cycles between a `cycle-tracker-report-start: <key>` and
/// its matching `cycle-tracker-report-end: <key>` under `<key>`. The keys emitted here
/// ([`DriverPhase::as_str`]) are the same ones the host reads back into its execution stats
/// (`Derivation Cycles` / `Block Execution Cycles`), so the per-phase cost breakdown is preserved
/// after deduplicating onto `kona_driver`'s shared derivation loop.
///
/// Outside the zkVM (native execution, e.g. the shared range core in tests) the markers are elided,
/// so the collector is an inert no-op.
#[derive(Debug, Clone, Copy, Default)]
pub struct CycleTrackerDriverMetrics;

impl DriverMetrics for CycleTrackerDriverMetrics {
    #[inline]
    fn phase_start(&self, phase: DriverPhase) {
        #[cfg(target_os = "zkvm")]
        println!("cycle-tracker-report-start: {}", phase.as_str());
        #[cfg(not(target_os = "zkvm"))]
        let _ = phase;
    }

    #[inline]
    fn phase_end(&self, phase: DriverPhase) {
        #[cfg(target_os = "zkvm")]
        println!("cycle-tracker-report-end: {}", phase.as_str());
        #[cfg(not(target_os = "zkvm"))]
        let _ = phase;
    }
}
