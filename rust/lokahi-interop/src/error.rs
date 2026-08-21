//! Error type shared by the interop stores.

/// Errors returned by the interop stores.
///
/// The first six variants mirror the sentinel errors the Go verifier matches on, so a port of
/// its control flow keeps the same branches.
#[derive(Debug, thiserror::Error)]
pub enum StoreError {
    /// The requested record is ahead of what the store holds. Transient: it may arrive.
    #[error("record is in the future")]
    Future,
    /// The requested record predates the store's history. Permanent for this store.
    #[error("record was skipped")]
    Skipped,
    /// The store holds different data than the caller asserted for that position.
    #[error("conflicting record: {0}")]
    Conflict(&'static str),
    /// A write arrived out of order with respect to the store's append cursor.
    #[error("out of order write: {0}")]
    OutOfOrder(&'static str),
    /// A stored record did not decode: the store is damaged, not merely incomplete.
    #[error("corrupt record: {0}")]
    DataCorruption(&'static str),
    /// No record exists at the requested key.
    #[error("not found")]
    NotFound,
    /// Verified timestamps must be committed one per second with no gaps.
    #[error("non-sequential commit: expected {expected}, got {actual}")]
    NonSequential {
        /// The timestamp the store expected next.
        expected: u64,
        /// The timestamp the caller offered.
        actual: u64,
    },
    /// A timestamp was re-committed with a different result than the one already stored.
    #[error("timestamp {0} already committed with a different result")]
    AlreadyCommitted(u64),
    /// The store has been closed.
    #[error("store closed")]
    Closed,
    /// An error originating from the underlying key/value backend.
    #[cfg(feature = "rocksdb")]
    #[error(transparent)]
    Backend(#[from] rocksdb::Error),
}
