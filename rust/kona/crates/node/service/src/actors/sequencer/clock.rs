//! Wall-clock time for the sequencer.

use std::{fmt::Debug, time::SystemTime};

/// A source of wall-clock time.
///
/// Block timestamps are wall-clock seconds, so the sequencer schedules blocks by wall-clock time,
/// which tests can control by providing their own [`Clock`].
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
