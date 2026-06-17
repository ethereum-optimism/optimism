//! Additional support for pooled interop transactions.

use alloy_consensus::Transaction;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};

use crate::interop_filter::CROSS_L2_INBOX_ADDRESS;

/// Returns true if the transaction's access list targets `CROSS_L2_INBOX_ADDRESS`
/// with at least one storage key.
///
/// Detection is intrinsic to the transaction (its access list) rather than the pool's
/// interop-deadline marker, so it identifies interop transactions even when they never went
/// through interop validation (e.g. private or locally-submitted transactions).
pub fn is_interop_tx<T>(tx: &T) -> bool
where
    T: Transaction,
{
    tx.access_list()
        .map(|al| {
            al.iter()
                .any(|item| item.address == CROSS_L2_INBOX_ADDRESS && !item.storage_keys.is_empty())
        })
        .unwrap_or(false)
}

/// Shareable interop failsafe gate.
///
/// `false` on construction. The interop filter client writes it (the background poll task and the
/// admission fast-path), while the payload builder reads it. When enabled, no interop transaction
/// may be admitted to the pool or sealed into a block. Cloning shares the underlying flag, so a
/// single handle threaded through node setup keeps the filter (writer) and the builder (reader) in
/// sync.
#[derive(Debug, Clone, Default)]
pub struct InteropFailsafe {
    inner: Arc<AtomicBool>,
}

impl InteropFailsafe {
    /// Returns the current failsafe state.
    pub fn enabled(&self) -> bool {
        self.inner.load(Ordering::Acquire)
    }

    /// Sets the failsafe state.
    pub fn set(&self, enabled: bool) {
        self.inner.store(enabled, Ordering::Release);
    }
}

/// Helper trait that allows attaching an interop deadline.
pub trait MaybeInteropTransaction {
    /// Attach an interop deadline
    fn set_interop_deadline(&self, deadline: u64);

    /// Get attached deadline if any.
    fn interop_deadline(&self) -> Option<u64>;

    /// Helper that sets the interop and returns the instance again
    fn with_interop_deadline(self, interop: u64) -> Self
    where
        Self: Sized,
    {
        self.set_interop_deadline(interop);
        self
    }
}

/// Helper to keep track of cross transaction interop validity
/// Checks if provided timestamp fits into tx validation window
#[inline]
pub const fn is_valid_interop(timeout: u64, timestamp: u64) -> bool {
    timestamp < timeout
}

/// Checks if transaction needs revalidation based on offset
#[inline]
pub const fn is_stale_interop(timeout: u64, timestamp: u64, offset: u64) -> bool {
    timestamp + offset > timeout
}
