//! Bounds the witness collected for one proof request.

use std::{
    collections::HashSet,
    sync::atomic::{AtomicBool, Ordering},
};

use kona_preimage::PreimageKey;
use parking_lot::Mutex;
use tokio::sync::Notify;

/// Archived-size allowance per preimage-map entry beyond its value bytes
/// (32-byte key, relative pointer and length, table slot, alignment).
pub const PREIMAGE_ENTRY_OVERHEAD_BYTES: u64 = 64;

/// Witness bytes of one blob: blob, KZG commitment and KZG proof.
pub const BLOB_WITNESS_BYTES: u64 = 131_072 + 48 + 48;

#[derive(Debug, Default)]
struct SizeState {
    seen: HashSet<PreimageKey>,
    bytes: u64,
}

/// Counts distinct witness bytes during collection and trips once over `limit`.
#[derive(Debug)]
pub struct WitnessSizeLimit {
    limit: u64,
    state: Mutex<SizeState>,
    exceeded: AtomicBool,
    notify: Notify,
}

impl WitnessSizeLimit {
    /// Creates a limit of `limit` witness bytes.
    pub fn new(limit: u64) -> Self {
        Self {
            limit,
            state: Mutex::default(),
            exceeded: AtomicBool::new(false),
            notify: Notify::new(),
        }
    }

    /// Records a preimage. Returns `false` once the limit has been exceeded (now or earlier).
    pub fn record(&self, key: PreimageKey, len: usize) -> bool {
        if self.is_exceeded() {
            return false;
        }
        let bytes = {
            let mut state = self.state.lock();
            if state.seen.insert(key) {
                state.bytes += len as u64 + PREIMAGE_ENTRY_OVERHEAD_BYTES;
            }
            state.bytes
        };
        if bytes <= self.limit {
            return true;
        }
        if !self.exceeded.swap(true, Ordering::AcqRel) {
            // `notify_one` stores a permit, so a waiter that registers later still wakes.
            self.notify.notify_one();
        }
        false
    }

    /// Distinct preimage bytes recorded so far, including per-entry overhead.
    pub fn bytes(&self) -> u64 {
        self.state.lock().bytes
    }

    /// The configured limit in bytes.
    pub const fn limit(&self) -> u64 {
        self.limit
    }

    /// Returns true once the limit has been exceeded.
    pub fn is_exceeded(&self) -> bool {
        self.exceeded.load(Ordering::Acquire)
    }

    /// Resolves once the limit has been exceeded. Supports a single waiter.
    pub async fn exceeded(&self) {
        if self.is_exceeded() {
            return;
        }
        self.notify.notified().await;
    }
}

#[cfg(test)]
mod tests {
    use std::{sync::Arc, time::Duration};

    use super::*;

    #[tokio::test]
    async fn counts_distinct_preimages_and_trips_once() {
        let limit = Arc::new(WitnessSizeLimit::new(200));
        let a = PreimageKey::new_local(1);
        let b = PreimageKey::new_local(2);
        // Registered before the trip, as the select in `collect_witness` is.
        let waiter = tokio::spawn({
            let limit = Arc::clone(&limit);
            async move { limit.exceeded().await }
        });
        tokio::task::yield_now().await;
        assert!(!waiter.is_finished());

        assert!(limit.record(a, 10));
        assert!(limit.record(a, 10));
        assert_eq!(limit.bytes(), 74);
        assert!(!limit.is_exceeded());

        assert!(!limit.record(b, 100));
        assert_eq!(limit.bytes(), 238);
        assert!(limit.is_exceeded());

        assert!(!limit.record(PreimageKey::new_local(3), 1));
        assert_eq!(limit.bytes(), 238);

        tokio::time::timeout(Duration::from_secs(1), waiter)
            .await
            .expect("the trip wakes a waiter registered before it")
            .unwrap();
        tokio::time::timeout(Duration::from_secs(1), limit.exceeded())
            .await
            .expect("a waiter after the trip returns immediately");
    }
}
