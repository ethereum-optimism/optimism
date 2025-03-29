//! Additional support for pooled interop transactions.

/// Helper trait that allows attaching an interop deadline.
pub trait MaybeInteropTransaction {
    /// Attach an interop deadline
    fn set_interop_deadlone(&self, deadline: u64);

    /// Get attached deadline if any.
    fn interop_deadline(&self) -> Option<u64>;

    /// Helper that sets the interop and returns the instance again
    fn with_interop_deadline(self, interop: u64) -> Self
    where
        Self: Sized,
    {
        self.set_interop_deadlone(interop);
        self
    }
}

/// Helper to keep track of cross transaction interop validity
/// Checks if provided timestamp fits into tx validation window
#[inline]
pub fn is_valid_interop(timeout: u64, timestamp: u64) -> bool {
    timestamp < timeout
}

/// Checks if transaction needs revalidation based on offset
#[inline]
pub fn is_stale_interop(timeout: u64, timestamp: u64, offset: u64) -> bool {
    timestamp + offset > timeout
}
