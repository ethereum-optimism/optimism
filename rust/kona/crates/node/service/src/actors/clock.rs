//! Wall-clock time for actors.

use std::{fmt::Debug, time::SystemTime};

/// A source of wall-clock time.
///
/// Actors read wall-clock time from a [`Clock`] rather than the system, so tests can control it.
/// For example, block timestamps are wall-clock seconds, so the sequencer schedules blocks by
/// wall-clock time.
pub trait Clock: Debug + Send + Sync {
    /// Returns the current wall-clock time.
    fn now(&self) -> SystemTime;
}

/// The system's wall clock.
#[derive(Debug, Default, Clone, Copy)]
pub struct SystemClock;

impl Clock for SystemClock {
    fn now(&self) -> SystemTime {
        SystemTime::now()
    }
}
