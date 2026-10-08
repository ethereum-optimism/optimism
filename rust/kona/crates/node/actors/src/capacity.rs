//! Mailbox capacity.

use thiserror::Error;
use tokio::sync::Semaphore;

/// A value in `1..=tokio::sync::Semaphore::MAX_PERMITS`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Capacity(usize);

impl Capacity {
    /// Returns the value.
    pub const fn get(self) -> usize {
        self.0
    }
}

impl TryFrom<usize> for Capacity {
    type Error = InvalidCapacity;

    fn try_from(value: usize) -> Result<Self, Self::Error> {
        if (1..=Semaphore::MAX_PERMITS).contains(&value) {
            Ok(Self(value))
        } else {
            Err(InvalidCapacity(value))
        }
    }
}

/// An integer outside the supported range.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("mailbox capacity {0} must be between 1 and {max}", max = Semaphore::MAX_PERMITS)]
pub struct InvalidCapacity(usize);
