//! Wall-clock time for actors.

use std::{fmt::Debug, time::SystemTime};

/// A source of wall-clock time.
///
/// Actors that need wall-clock time read it from a [`Clock`] rather than the system, so tests can
/// control it. For example, block timestamps are wall-clock seconds, so the sequencer schedules
/// blocks by wall-clock time.
///
/// Implementations used in production must follow the system's wall clock, including its
/// adjustments by NTP, because block timestamps are compared against it.
pub trait Clock: Debug + Send + Sync {
    /// Returns the current wall-clock time.
    fn now(&self) -> SystemTime;
}

/// The system's wall clock.
#[derive(Debug, Default, Clone, Copy)]
pub struct SystemClock;

impl Clock for SystemClock {
    // The system clock is what this implementation provides.
    #[allow(clippy::disallowed_methods)]
    fn now(&self) -> SystemTime {
        SystemTime::now()
    }
}
